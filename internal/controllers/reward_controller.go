package controllers

import (
	"net/http"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type RewardController struct {
	Svc services.RewardService
}

func NewRewardController(svc services.RewardService) *RewardController {
	return &RewardController{Svc: svc}
}

func aksesReward(c *gin.Context) (services.AksesRewardInput, bool) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return services.AksesRewardInput{}, false
	}
	return services.AksesRewardInput{
		ActorUserID: claims.UserID,
		ActorBankID: claims.BankID,
		ActorRole:   claims.Role,
	}, true
}

func (rc *RewardController) AddReward(c *gin.Context) {
	var req struct {
		NamaReward models.RewardEnum       `json:"nama_reward" binding:"required"`
		Satuan     models.SatuanRewardEnum `json:"satuan" binding:"required"`
		Deskripsi  *string                 `json:"deskripsi"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reward, err := rc.Svc.TambahReward(services.RewardInput{
		NamaReward: req.NamaReward,
		Satuan:     req.Satuan,
		Deskripsi:  req.Deskripsi,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Reward berhasil ditambahkan",
		"data":    reward,
	})
}

func (rc *RewardController) GetRewards(c *gin.Context) {
	rewards, err := rc.Svc.ListReward()
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data reward berhasil diambil",
		"data":    rewards,
	})
}

func (rc *RewardController) UpdateReward(c *gin.Context) {
	var req struct {
		NamaReward models.RewardEnum       `json:"nama_reward" binding:"required"`
		Satuan     models.SatuanRewardEnum `json:"satuan" binding:"required"`
		Deskripsi  *string                 `json:"deskripsi"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reward, err := rc.Svc.UbahReward(c.Param("reward_id"), services.RewardInput{
		NamaReward: req.NamaReward,
		Satuan:     req.Satuan,
		Deskripsi:  req.Deskripsi,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data reward berhasil diupdate",
		"data":    reward,
	})
}

func (rc *RewardController) DeleteReward(c *gin.Context) {
	if err := handleServiceError(c, rc.Svc.HapusReward(c.Param("reward_id"))); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data reward berhasil dihapus"})
}

func (rc *RewardController) GetNilaiRewardBank(c *gin.Context) {
	rows, err := rc.Svc.ListNilaiRewardBank(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data nilai reward berhasil diambil",
		"data":    rows,
	})
}

func (rc *RewardController) AddNewNilaiRewardBank(c *gin.Context) {
	akses, ok := aksesReward(c)
	if !ok {
		return
	}

	var req struct {
		RewardID   int                        `json:"reward_id" binding:"required"`
		Persentase []services.PersentaseInput `json:"persentase" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rows, err := rc.Svc.TambahNilaiReward(c.Param("bank_id"), services.NilaiRewardInput{
		RewardID:   req.RewardID,
		Persentase: req.Persentase,
	}, akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Nilai reward berhasil ditambahkan",
		"data":    rows,
	})
}

func (rc *RewardController) UpdateNilaiRewardBank(c *gin.Context) {
	akses, ok := aksesReward(c)
	if !ok {
		return
	}

	var req struct {
		Persentase []services.PersentaseInput `json:"persentase" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rows, err := rc.Svc.UbahNilaiReward(
		c.Param("bank_id"),
		c.Param("reward_id"),
		services.NilaiRewardInput{Persentase: req.Persentase},
		akses,
	)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nilai reward berhasil diupdate",
		"data":    rows,
	})
}

func (rc *RewardController) GetHistoryNilaiRewardBank(c *gin.Context) {
	akses, ok := aksesReward(c)
	if !ok {
		return
	}

	history, err := rc.Svc.HistoryNilaiReward(c.Param("nilai_reward_id"), akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History nilai reward berhasil diambil",
		"data":    history,
	})
}

func (rc *RewardController) DeleteNilaiRewardBank(c *gin.Context) {
	akses, ok := aksesReward(c)
	if !ok {
		return
	}

	if err := handleServiceError(c, rc.Svc.HapusNilaiReward(c.Param("nilai_reward_id"), akses)); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Data nilai reward berhasil dihapus"})
}

func (rc *RewardController) GetDetailNilaiRewardBank(c *gin.Context) {
	akses, ok := aksesReward(c)
	if !ok {
		return
	}

	detail, err := rc.Svc.DetailNilaiReward(c.Param("nilai_reward_id"), akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail nilai reward berhasil diambil",
		"data":    detail,
	})
}
