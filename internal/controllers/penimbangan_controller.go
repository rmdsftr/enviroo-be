package controllers

import (
	"net/http"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type PenimbanganController struct {
	Svc services.PenimbanganService
}

func NewPenimbanganController(svc services.PenimbanganService) *PenimbanganController {
	return &PenimbanganController{Svc: svc}
}

func (pc *PenimbanganController) CheckJadwalHariIni(c *gin.Context) {
	sesi, err := pc.Svc.ListSesiHariIni(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data sesi penimbangan hari ini",
		"data":    sesi,
	})
}

func (pc *PenimbanganController) CheckJadwalActive(c *gin.Context) {
	result, err := pc.Svc.CekSesiAktif(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, result)
}

func (pc *PenimbanganController) UpdatePenimbangan(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req struct {
		StatusPenimbangan models.StatusPenimbangan `json:"status_penimbangan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Status penimbangan harus disertakan (aktif/selesai)"})
		return
	}

	penimbangan, err := pc.Svc.UbahStatusSesi(services.UbahStatusSesiInput{
		PenimbanganID: c.Param("penimbangan_id"),
		StatusTujuan:  req.StatusPenimbangan,
		ActorUserID:   claims.UserID,
		ActorBankID:   claims.BankID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Status sesi penimbangan berhasil diperbarui",
		"data":    penimbangan,
	})
}

func (pc *PenimbanganController) BatalkanPenimbangan(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req struct {
		JadwalID         string `json:"jadwal_id"`
		TanggalSesi      string `json:"tanggal_sesi"`
		AlasanPembatalan string `json:"alasan_pembatalan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alasan_pembatalan wajib diisi"})
		return
	}

	result, err := pc.Svc.BatalkanSesi(services.BatalkanSesiInput{
		PenimbanganID:    c.Query("penimbangan_id"),
		JadwalID:         req.JadwalID,
		TanggalSesi:      req.TanggalSesi,
		AlasanPembatalan: req.AlasanPembatalan,
		ActorUserID:      claims.UserID,
		ActorBankID:      claims.BankID,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	status := http.StatusOK
	if result.Dibuat {
		status = http.StatusCreated
	}
	c.JSON(status, gin.H{
		"message": "Sesi penimbangan berhasil dibatalkan",
		"data":    result.Penimbangan,
	})
}

func (pc *PenimbanganController) GetPenimbangan(c *gin.Context) {
	riwayat, err := pc.Svc.RiwayatPenimbangan(c.Param("bank_id"), c.Query("start_date"), c.Query("end_date"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data penimbangan",
		"data":    riwayat,
	})
}

func (pc *PenimbanganController) ListSetoranPenimbangan(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	detail, err := pc.Svc.DetailSesi(c.Param("penimbangan_id"), services.AksesSesiInput{
		ActorBankID: claims.BankID,
		ActorRole:   claims.Role,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil data setoran",
		"data":    detail,
	})
}
