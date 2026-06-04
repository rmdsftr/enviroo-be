package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"net/http"
	"time"

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
// Menambahkan jenis reward baru. NamaReward harus salah satu dari: Uang, Emas, Sembako.
// POST /reward/add
func (rc *RewardController) AddReward(c *gin.Context) {
	var req struct {
		NamaReward models.RewardEnum        `json:"nama_reward" binding:"required"`
		Satuan     models.SatuanRewardEnum  `json:"satuan" binding:"required"`
		Deskripsi  *string                  `json:"deskripsi"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validasi enum RewardEnum
	switch req.NamaReward {
	case models.RewardEnumUang, models.RewardEnumSembako:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama_reward harus salah satu dari: Uang, Sembako"})
		return
	}

	switch req.Satuan {
	case models.SatuanRewardEnumRp, models.SatuanRewardEnumPoin:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "satuan harus salah satu dari: Rp, poin"})
		return
	}

	newReward := models.Reward{
		NamaReward: req.NamaReward,
		Satuan:     string(req.Satuan),
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
// GET /reward/list
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
// PATCH /reward/update/:reward_id
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
		NamaReward models.RewardEnum        `json:"nama_reward" binding:"required"`
		Satuan     models.SatuanRewardEnum  `json:"satuan" binding:"required"`
		Deskripsi  *string                  `json:"deskripsi"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch req.NamaReward {
	case models.RewardEnumUang, models.RewardEnumSembako:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "nama_reward harus salah satu dari: Uang, Sembako"})
		return
	}

	switch req.Satuan {
	case models.SatuanRewardEnumRp, models.SatuanRewardEnumPoin:
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "satuan harus salah satu dari: Rp, poin"})
		return
	}

	reward.NamaReward = req.NamaReward
	reward.Satuan = string(req.Satuan)
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
// DELETE /reward/delete/:reward_id
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

	c.JSON(http.StatusOK, gin.H{"message": "Data reward berhasil dihapus"})
}

// ─── GetNilaiRewardBank ──────────────────────────────────────────────────────
// GET /reward/nilai-reward/:bank_id
// Mengambil konfigurasi bagi hasil per reward per level_user untuk satu bank.
// BSU akan mengambil dari BSI induknya.
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
// POST /reward/nilai-reward/:bank_id
// Menambahkan konfigurasi bagi hasil untuk satu reward di satu bank.
// Field RasioHarga, RasioKonversi, SatuanHarga, SatuanKonversi WAJIB diisi jika RewardEnum = Emas.
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

	type Persentase struct{
		LevelUser       models.LevelUser        `json:"level_user" binding:"required"`
		PersenBagiHasil float64                 `json:"persen_bagi_hasil" binding:"required"`
	}

	var req struct {
		RewardID        int                     `json:"reward_id" binding:"required"`
		Persentase      []Persentase            `json:"persentase" binding:"required"`
		CreatedBy       string                  `json:"created_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Cek jenis reward
	var reward models.Reward
	if err := rc.db.Where("reward_id = ?", req.RewardID).First(&reward).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Reward tidak ditemukan"})
		return
	}

	tx := rc.db.Begin()
	var newNilaiRewards []models.NilaiRewardBank

	for _, p := range req.Persentase {
		// Abaikan jika frontend mengirimkan eksternal secara eksplisit
		if p.LevelUser == models.LevelEksternal {
			continue
		}

		// Cek duplikasi (bank + reward + level_user harus unik)
		var existing models.NilaiRewardBank
		if err := tx.Where("bank_id = ? AND reward_id = ? AND level_user = ?", bankID, req.RewardID, p.LevelUser).First(&existing).Error; err == nil {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "Konfigurasi reward untuk bank, tipe reward, dan level user " + string(p.LevelUser) + " sudah ada"})
			return
		}

		newNilaiReward := models.NilaiRewardBank{
			NilaiRewardID:   utils.GenerateID("NRB"),
			BankID:          bankID,
			RewardID:        req.RewardID,
			LevelUser:       p.LevelUser,
			PersenBagiHasil: p.PersenBagiHasil,
		}
		if req.CreatedBy != "" {
			newNilaiReward.CreatedBy = &req.CreatedBy
		}

		if err := tx.Create(&newNilaiReward).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan nilai reward: " + err.Error()})
			return
		}
		newNilaiRewards = append(newNilaiRewards, newNilaiReward)
	}

	// ── Auto-create / Auto-update Level Eksternal ─────────────────────────────
	var eksternalReward models.NilaiRewardBank
	errEks := tx.Where("bank_id = ? AND reward_id = ? AND level_user = ?", bankID, req.RewardID, models.LevelEksternal).First(&eksternalReward).Error

	if errEks != nil {
		// Belum ada, buat baru dengan persentase = 100
		newEksternal := models.NilaiRewardBank{
			NilaiRewardID:   utils.GenerateID("NRB"),
			BankID:          bankID,
			RewardID:        req.RewardID,
			LevelUser:       models.LevelEksternal,
			PersenBagiHasil: 100, // 100% untuk eksternal
		}
		if req.CreatedBy != "" {
			newEksternal.CreatedBy = &req.CreatedBy
		}

		if err := tx.Create(&newEksternal).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambahkan nilai reward eksternal: " + err.Error()})
			return
		}
		newNilaiRewards = append(newNilaiRewards, newEksternal)
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal commit data: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Nilai reward berhasil ditambahkan",
		"data":    newNilaiRewards,
	})
}

// ─── UpdateNilaiRewardBank ───────────────────────────────────────────────────
// PATCH /nilai-reward/edit/:bank_id/:reward_id
// Mengupdate konfigurasi bagi hasil. Setiap perubahan akan dicatat di history.
func (rc *RewardController) UpdateNilaiRewardBank(c *gin.Context) {
	bankID := c.Param("bank_id")
	rewardIDStr := c.Param("reward_id")

	type Persentase struct {
		LevelUser       models.LevelUser `json:"level_user" binding:"required"`
		PersenBagiHasil float64          `json:"persen_bagi_hasil" binding:"required"`
	}

	var req struct {
		Persentase     []Persentase            `json:"persentase" binding:"required"`
		UpdatedBy      string                  `json:"updated_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := rc.db.Begin()

	var updatedRewards []models.NilaiRewardBank

	for _, p := range req.Persentase {
		// Abaikan jika frontend mengirimkan eksternal secara eksplisit
		if p.LevelUser == models.LevelEksternal {
			continue
		}

		var nilaiReward models.NilaiRewardBank
		if err := tx.Preload("Reward").Where("bank_id = ? AND reward_id = ? AND level_user = ?", bankID, rewardIDStr, p.LevelUser).First(&nilaiReward).Error; err != nil {
			// Jika belum ada, anggap error karena ini endpoint update
			tx.Rollback()
			c.JSON(http.StatusNotFound, gin.H{"error": "Data nilai reward untuk level user " + string(p.LevelUser) + " tidak ditemukan"})
			return
		}

		// Simpan nilai lama untuk history
		oldPersen := nilaiReward.PersenBagiHasil

		nilaiReward.PersenBagiHasil = p.PersenBagiHasil
		if req.UpdatedBy != "" {
			nilaiReward.UpdatedBy = &req.UpdatedBy
		}

		if err := tx.Save(&nilaiReward).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate nilai reward: " + err.Error()})
			return
		}

		history := models.HistoryNilaiReward{
			NilaiRewardID:    nilaiReward.NilaiRewardID,
			OldPersen:        &oldPersen,
			NewPersen:        &p.PersenBagiHasil,
		}
		if req.UpdatedBy != "" {
			history.ChangedBy = &req.UpdatedBy
		}

		if err := tx.Create(&history).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat history nilai reward: " + err.Error()})
			return
		}

		updatedRewards = append(updatedRewards, nilaiReward)
	}

	// ── Auto-update Level Eksternal sudah dihapus ───────────

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Nilai reward berhasil diupdate",
		"data":    updatedRewards,
	})
}

// ─── GetHistoryNilaiRewardBank ───────────────────────────────────────────────
// GET /reward/history-nilai-reward/:nilai_reward_id
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
// DELETE /reward/delete-nilai-reward/:nilai_reward_id
func (rc *RewardController) DeleteNilaiRewardBank(c *gin.Context) {
	nilaiRewardID := c.Param("nilai_reward_id")

	tx := rc.db.Begin()

	var nilaiReward models.NilaiRewardBank
	if err := tx.Where("nilai_reward_id = ?", nilaiRewardID).First(&nilaiReward).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Data nilai reward tidak ditemukan"})
		return
	}

	// Hapus history terlebih dahulu (FK constraint)
	if err := tx.Where("nilai_reward_id = ?", nilaiRewardID).Delete(&models.HistoryNilaiReward{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus history nilai reward: " + err.Error()})
		return
	}

	if err := tx.Delete(&nilaiReward).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus nilai reward: " + err.Error()})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{"message": "Data nilai reward berhasil dihapus"})
}

// ─── GetDetailNilaiRewardBank ────────────────────────────────────────────────
// GET /nilai-reward/detail/:nilai_reward_id
func (rc *RewardController) GetDetailNilaiRewardBank(c *gin.Context) {
	nilaiRewardID := c.Param("nilai_reward_id")

	// 1. Ambil data NilaiRewardBank awal untuk dapet bank_id & reward_id
	var initialNilai models.NilaiRewardBank
	if err := rc.db.Preload("Bank").Preload("Reward").Where("nilai_reward_id = ?", nilaiRewardID).First(&initialNilai).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data nilai reward tidak ditemukan"})
		return
	}

	// 2. Ambil semua level yang terkonfigurasi untuk reward ini di bank ini
	var allLevels []models.NilaiRewardBank
	if err := rc.db.Where("bank_id = ? AND reward_id = ?", initialNilai.BankID, initialNilai.RewardID).Find(&allLevels).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data level: " + err.Error()})
		return
	}

	// 3. Kumpulkan NilaiRewardID untuk tarik history gabungan
	var ids []string
	type PersentaseItem struct {
		LevelUser       models.LevelUser `json:"level_user"`
		PersenBagiHasil float64          `json:"persen_bagi_hasil"`
	}
	var persentaseList []PersentaseItem
	for _, l := range allLevels {
		ids = append(ids, l.NilaiRewardID)
		persentaseList = append(persentaseList, PersentaseItem{
			LevelUser:       l.LevelUser,
			PersenBagiHasil: l.PersenBagiHasil,
		})
	}

	// 4. Ambil history gabungan dengan info level user-nya
	type HistoryResponse struct {
		HistoryRewardID  int       `json:"history_reward_id"`
		NilaiRewardID    string    `json:"nilai_reward_id"`
		LevelUser        string    `json:"level_user"`
		OldPersen        *float64  `json:"old_persen"`
		NewPersen        *float64  `json:"new_persen"`
		ChangedBy        string    `json:"changed_by"`
		ChangedAt        time.Time `json:"changed_at"`
	}

	var history []HistoryResponse
	if err := rc.db.Table("history_nilai_reward hnr").
		Select(`hnr.history_reward_id, hnr.nilai_reward_id, nrb.level_user,
			hnr.old_persen, hnr.new_persen, 
			COALESCE(u.nama, hnr.changed_by) AS changed_by,
			hnr.changed_at`).
		Joins("INNER JOIN nilai_reward_bank nrb ON nrb.nilai_reward_id = hnr.nilai_reward_id").
		Joins("LEFT JOIN admin a ON a.admin_id = hnr.changed_by").
		Joins("LEFT JOIN users u ON u.user_id = a.user_id").
		Where("hnr.nilai_reward_id IN ?", ids).
		Order("hnr.changed_at DESC").
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil history: " + err.Error()})
		return
	}

	// 5. Response aggregate
	c.JSON(http.StatusOK, gin.H{
		"message": "Detail nilai reward berhasil diambil",
		"data": gin.H{
			"bank_id":           initialNilai.BankID,
			"reward_id":         initialNilai.RewardID,
			"nama_reward":       initialNilai.Reward.NamaReward,
			"satuan":            initialNilai.Reward.Satuan,
			"persentase":        persentaseList,
			"history":           history,
		},
	})
}