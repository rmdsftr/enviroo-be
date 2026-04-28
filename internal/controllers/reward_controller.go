package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RewardController struct {
	db        *gorm.DB
	cfStorage *storage.CloudflareStorage
	mailer    *utils.Mailer
}

func NewRewardController(db *gorm.DB, cfStorage *storage.CloudflareStorage, mailer *utils.Mailer) *RewardController {
	return &RewardController{
		db:        db,
		cfStorage: cfStorage,
		mailer:    mailer,
	}
}

// ─── AddReward ──────────────────────────────────────────────────────────────
func (rc *RewardController) AddReward(c *gin.Context) {
	var req struct {
		NamaReward string  `json:"nama_reward" binding:"required"`
		Satuan     string  `json:"satuan" binding:"required"`
		Deskripsi  *string `json:"deskripsi"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	newReward := models.Reward{
		NamaReward: req.NamaReward,
		Satuan:     req.Satuan,
		Deskripsi:  req.Deskripsi,
	}

	if err := rc.db.Create(&newReward).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan data reward: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Reward berhasil ditambahkan",
		"data":    newReward,
	})
}

// ─── GetRewards ─────────────────────────────────────────────────────────────
func (rc *RewardController) GetRewards(c *gin.Context) {
	var rewards []models.Reward

	if err := rc.db.Order("created_at DESC").Find(&rewards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data reward: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data reward berhasil diambil",
		"data":    rewards,
	})
}

// ─── UpdateReward ───────────────────────────────────────────────────────────
func (rc *RewardController) UpdateReward(c *gin.Context) {
	rewardID := c.Param("reward_id")

	var reward models.Reward
	if err := rc.db.Where("reward_id = ?", rewardID).First(&reward).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data reward tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencari data reward: " + err.Error()})
		}
		return
	}

	var req struct {
		NamaReward string  `json:"nama_reward" binding:"required"`
		Satuan     string  `json:"satuan" binding:"required"`
		Deskripsi  *string `json:"deskripsi"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reward.NamaReward = req.NamaReward
	reward.Satuan = req.Satuan
	reward.Deskripsi = req.Deskripsi

	if err := rc.db.Save(&reward).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate data reward: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data reward berhasil diupdate",
		"data":    reward,
	})
}

// ─── DeleteReward ───────────────────────────────────────────────────────────
func (rc *RewardController) DeleteReward(c *gin.Context) {
	rewardID := c.Param("reward_id")

	var reward models.Reward
	if err := rc.db.Where("reward_id = ?", rewardID).First(&reward).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Data reward tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencari data reward: " + err.Error()})
		}
		return
	}

	if err := rc.db.Delete(&reward).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus data reward: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data reward berhasil dihapus",
	})
}

// ─── GetNilaiRewardBank ──────────────────────────────────────────────────────
func (rc *RewardController) GetNilaiRewardBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := rc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	targetBankID := bankID
	if bank.JenisBank == models.BSU {
		if bank.ParentBankID == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "BSU tidak memiliki parent bank"})
			return
		}
		targetBankID = *bank.ParentBankID
	}

	var nilaiRewards []models.NilaiRewardBank
	if err := rc.db.Preload("Reward").Where("bank_id = ?", targetBankID).Find(&nilaiRewards).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data nilai reward: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data nilai reward berhasil diambil",
		"data":    nilaiRewards,
	})
}

// ─── AddNewNilaiRewardBank ───────────────────────────────────────────────────
func (rc *RewardController) AddNewNilaiRewardBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := rc.db.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusForbidden, gin.H{"error": "BSU tidak diizinkan mengelola nilai reward, nilai mengikuti BSI"})
		return
	}

	var req struct {
		RewardID      int     `json:"reward_id" binding:"required"`
		NilaiPoin     float64 `json:"nilai_poin" binding:"required"`
		NilaiKonversi float64 `json:"nilai_konversi" binding:"required"`
		CreatedBy     string  `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek apakah mapping sudah ada
	var existing models.NilaiRewardBank
	if err := rc.db.Where("bank_id = ? AND reward_id = ?", bankID, req.RewardID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Nilai reward untuk bank ini dan tipe reward tersebut sudah ada"})
		return
	}

	nilaiRewardID := utils.GenerateID("NRB")

	newNilaiReward := models.NilaiRewardBank{
		NilaiRewardID: nilaiRewardID,
		BankID:        bankID,
		RewardID:      req.RewardID,
		NilaiPoin:     req.NilaiPoin,
		NilaiKonversi: req.NilaiKonversi,
	}

	if req.CreatedBy != "" {
		newNilaiReward.CreatedBy = &req.CreatedBy
	}

	if err := rc.db.Create(&newNilaiReward).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan nilai reward: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Nilai reward berhasil ditambahkan",
		"data":    newNilaiReward,
	})
}

// ─── UpdateNilaiRewardBank ───────────────────────────────────────────────────
func (rc *RewardController) UpdateNilaiRewardBank(c *gin.Context) {
	nilaiRewardID := c.Param("nilai_reward_id")

	var req struct {
		NilaiPoin     float64 `json:"nilai_poin" binding:"required"`
		NilaiKonversi float64 `json:"nilai_konversi" binding:"required"`
		UpdatedBy     string  `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := rc.db.Begin()

	var nilaiReward models.NilaiRewardBank
	if err := tx.Where("nilai_reward_id = ?", nilaiRewardID).First(&nilaiReward).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Data nilai reward tidak ditemukan"})
		return
	}

	oldPoin := nilaiReward.NilaiPoin
	oldKonversi := nilaiReward.NilaiKonversi

	nilaiReward.NilaiPoin = req.NilaiPoin
	nilaiReward.NilaiKonversi = req.NilaiKonversi
	if req.UpdatedBy != "" {
		nilaiReward.UpdatedBy = &req.UpdatedBy
	}

	if err := tx.Save(&nilaiReward).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate nilai reward: " + err.Error()})
		return
	}

	// Buat history
	history := models.HistoryNilaiReward{
		NilaiRewardID:    nilaiRewardID,
		OldNilaiPoin:     &oldPoin,
		NewNilaiPoin:     &req.NilaiPoin,
		OldNilaiKonversi: &oldKonversi,
		NewNilaiKonversi: &req.NilaiKonversi,
	}
	if req.UpdatedBy != "" {
		history.ChangedBy = &req.UpdatedBy
	}

	if err := tx.Create(&history).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat history nilai reward: " + err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Nilai reward berhasil diupdate",
		"data":    nilaiReward,
	})
}

// ─── GetHistoryNilaiRewardBank ───────────────────────────────────────────────
func (rc *RewardController) GetHistoryNilaiRewardBank(c *gin.Context) {
	nilaiRewardID := c.Param("nilai_reward_id")

	var history []models.HistoryNilaiReward
	if err := rc.db.Where("nilai_reward_id = ?", nilaiRewardID).Order("changed_at DESC").Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil history nilai reward: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History nilai reward berhasil diambil",
		"data":    history,
	})
}

// ─── DeleteNilaiRewardBank ───────────────────────────────────────────────────
func (rc *RewardController) DeleteNilaiRewardBank(c *gin.Context) {
	nilaiRewardID := c.Param("nilai_reward_id")

	tx := rc.db.Begin()

	var nilaiReward models.NilaiRewardBank
	if err := tx.Where("nilai_reward_id = ?", nilaiRewardID).First(&nilaiReward).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Data nilai reward tidak ditemukan"})
		return
	}

	// Hapus history-nya terlebih dahulu karena foreign key constraint
	if err := tx.Where("nilai_reward_id = ?", nilaiRewardID).Delete(&models.HistoryNilaiReward{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus history nilai reward: " + err.Error()})
		return
	}

	// Hapus nilai reward
	if err := tx.Delete(&nilaiReward).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus nilai reward: " + err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Data nilai reward berhasil dihapus",
	})
}