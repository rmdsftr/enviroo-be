package controllers

import (
	"net/http"
	"strconv"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type JadwalController struct {
	Svc services.JadwalService
}

func NewJadwalController(svc services.JadwalService) *JadwalController {
	return &JadwalController{Svc: svc}
}

func aksesJadwal(c *gin.Context) (services.AksesJadwalInput, bool) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return services.AksesJadwalInput{}, false
	}
	return services.AksesJadwalInput{
		ActorUserID: claims.UserID,
		ActorBankID: claims.BankID,
		ActorRole:   claims.Role,
	}, true
}

func periodeQuery(c *gin.Context) (month, year int) {
	month, _ = strconv.Atoi(c.Query("month"))
	year, _ = strconv.Atoi(c.Query("year"))
	return month, year
}

func (jc *JadwalController) AddNewJadwal(c *gin.Context) {
	akses, ok := aksesJadwal(c)
	if !ok {
		return
	}

	var req struct {
		Hari              models.HariEnum   `json:"hari"`
		MingguKe          int               `json:"minggu_ke"`
		JamMulai          string            `json:"jam_mulai"`
		JamSelesai        string            `json:"jam_selesai"`
		JenisJadwal       models.JadwalEnum `json:"jenis_jadwal"`
		TargetBankID      string            `json:"target_bank_id"`
		IsActive          *bool             `json:"is_active"`
		IsRutin           bool              `json:"is_rutin"`
		Tanggal           string            `json:"tanggal"`
		NamaJadwalSpesial string            `json:"nama_jadwal_spesial"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jadwal, err := jc.Svc.TambahJadwal(c.Param("bank_id"), services.TambahJadwalInput{
		Hari:              req.Hari,
		MingguKe:          req.MingguKe,
		JamMulai:          req.JamMulai,
		JamSelesai:        req.JamSelesai,
		JenisJadwal:       req.JenisJadwal,
		TargetBankID:      req.TargetBankID,
		IsActive:          req.IsActive,
		IsRutin:           req.IsRutin,
		Tanggal:           req.Tanggal,
		NamaJadwalSpesial: req.NamaJadwalSpesial,
	}, akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil ditambahkan", "data": jadwal})
}

func (jc *JadwalController) AddJadwalBatch(c *gin.Context) {
	akses, ok := aksesJadwal(c)
	if !ok {
		return
	}

	var req struct {
		Items []services.BatchJadwalItem `json:"items"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid"})
		return
	}

	jadwals, err := jc.Svc.TambahJadwalBatch(c.Param("bank_id"), req.Items, akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil ditambahkan", "data": jadwals})
}

func (jc *JadwalController) GetJadwalBank(c *gin.Context) {
	month, year := periodeQuery(c)

	hasil, err := jc.Svc.JadwalBank(c.Param("bank_id"), month, year)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": hasil})
}

func (jc *JadwalController) GetJadwalPenimbanganBank(c *gin.Context) {
	month, year := periodeQuery(c)

	items, err := jc.Svc.JadwalPenimbanganBank(c.Param("bank_id"), month, year)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": items})
}

func (jc *JadwalController) GetMasterJadwalPenimbanganBank(c *gin.Context) {
	month, year := periodeQuery(c)

	hasil, err := jc.Svc.MasterJadwalPenimbanganBank(c.Param("bank_id"), month, year)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": hasil})
}

func (jc *JadwalController) GetAllJadwal(c *gin.Context) {
	hasil, err := jc.Svc.SemuaJadwal(c.Query("type"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": hasil})
}

func (jc *JadwalController) UpdateJadwal(c *gin.Context) {
	akses, ok := aksesJadwal(c)
	if !ok {
		return
	}

	var req struct {
		Hari              models.HariEnum `json:"hari"`
		MingguKe          int             `json:"minggu_ke"`
		JamMulai          string          `json:"jam_mulai"`
		JamSelesai        string          `json:"jam_selesai"`
		IsActive          *bool           `json:"is_active"`
		Tanggal           string          `json:"tanggal"`
		NamaJadwalSpesial string          `json:"nama_jadwal_spesial"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	jadwal, err := jc.Svc.UbahJadwal(c.Param("jadwal_id"), services.UbahJadwalInput{
		Hari:              req.Hari,
		MingguKe:          req.MingguKe,
		JamMulai:          req.JamMulai,
		JamSelesai:        req.JamSelesai,
		IsActive:          req.IsActive,
		Tanggal:           req.Tanggal,
		NamaJadwalSpesial: req.NamaJadwalSpesial,
	}, akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil diupdate", "data": jadwal})
}

func (jc *JadwalController) DeleteJadwal(c *gin.Context) {
	akses, ok := aksesJadwal(c)
	if !ok {
		return
	}

	if err := handleServiceError(c, jc.Svc.HapusJadwal(c.Param("jadwal_id"), akses)); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jadwal berhasil dihapus"})
}
