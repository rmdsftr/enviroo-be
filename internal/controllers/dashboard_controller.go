package controllers

import (
	"net/http"
	"strconv"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/services"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
)

type DashboardController struct {
	Svc services.DashboardService
}

func NewDashboardController(svc services.DashboardService) *DashboardController {
	return &DashboardController{Svc: svc}
}

func aksesDashboard(c *gin.Context) (services.AksesDashboardInput, bool) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return services.AksesDashboardInput{}, false
	}
	return services.AksesDashboardInput{
		ActorUserID: claims.UserID,
		ActorBankID: claims.BankID,
		ActorRole:   claims.Role,
	}, true
}

func filterMutasi(c *gin.Context) services.FilterMutasiInput {
	rewardID, _ := strconv.Atoi(c.Query("reward_id"))
	return services.FilterMutasiInput{
		RewardID:  rewardID,
		StartDate: c.Query("start_date"),
		EndDate:   c.Query("end_date"),
	}
}

func (dc *DashboardController) GetDashboardPetugas(c *gin.Context) {
	hasil, err := dc.Svc.DashboardPetugas(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": hasil})
}

func (dc *DashboardController) GetSaldoBank(c *gin.Context) {
	hasil, err := dc.Svc.SaldoBank(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data saldo bank berhasil diambil",
		"data":    hasil,
	})
}

func (dc *DashboardController) GetSaldoNasabah(c *gin.Context) {
	hasil, err := dc.Svc.SaldoNasabah(c.Param("nasabah_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data saldo nasabah berhasil diambil",
		"data":    hasil,
	})
}

func (dc *DashboardController) MutasiSaldoBank(c *gin.Context) {
	hasil, err := dc.Svc.MutasiBank(c.Param("bank_id"), filterMutasi(c))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data mutasi saldo bank berhasil diambil",
		"data":    hasil,
	})
}

func (dc *DashboardController) MutasiSaldoNasabah(c *gin.Context) {
	hasil, err := dc.Svc.MutasiNasabah(c.Param("nasabah_id"), filterMutasi(c))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data mutasi saldo nasabah berhasil diambil",
		"data":    hasil,
	})
}

func (dc *DashboardController) CatatManualMutasiBank(c *gin.Context) {
	akses, ok := aksesDashboard(c)
	if !ok {
		return
	}

	var body struct {
		TipeMutasi  string  `json:"tipe_mutasi" binding:"required,oneof=debit kredit"`
		JenisReward string  `json:"jenis_reward" binding:"required,oneof=uang poin"`
		Nominal     float64 `json:"nominal" binding:"required,gt=0"`
		Keterangan  string  `json:"keterangan" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Request tidak valid", "error": err.Error()})
		return
	}

	err := dc.Svc.CatatManualMutasiBank(c.Param("bank_id"), services.CatatMutasiInput{
		TipeMutasi:  body.TipeMutasi,
		JenisReward: body.JenisReward,
		Nominal:     body.Nominal,
		Keterangan:  body.Keterangan,
	}, akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mutasi berhasil dicatat"})
}

func (dc *DashboardController) TotalSaldoAllNasabah(c *gin.Context) {
	hasil, err := dc.Svc.TotalSaldoSemuaNasabah(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Total saldo semua nasabah berhasil diambil",
		"data":    hasil,
	})
}

func (dc *DashboardController) ListSaldoAllNasabah(c *gin.Context) {
	items, meta, err := dc.Svc.ListSaldoSemuaNasabah(
		c.Param("bank_id"),
		utils.ParsePage(c.DefaultQuery("page", "1")),
	)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil list saldo nasabah",
		"data":    items,
		"meta":    meta,
	})
}
