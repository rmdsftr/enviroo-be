package controllers

import (
	"net/http"

	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"

	"github.com/gin-gonic/gin"
)

type PenarikanController struct {
	Svc services.PenarikanService
	CF  *storage.CloudflareStorage
}

func NewPenarikanController(svc services.PenarikanService, cf *storage.CloudflareStorage) *PenarikanController {
	return &PenarikanController{Svc: svc, CF: cf}
}

func (pc *PenarikanController) PreviewAjukanPenarikan(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var req services.AjukanPenarikanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid: " + err.Error()})
		return
	}

	result, err := pc.Svc.Preview(nasabahID, req)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": result})
}

func (pc *PenarikanController) AjukanPenarikan(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var req services.AjukanPenarikanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid: " + err.Error()})
		return
	}

	result, err := pc.Svc.Ajukan(nasabahID, req)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Pengajuan penarikan berhasil disimpan",
		"data":    result,
	})
}

func (pc *PenarikanController) KonfirmasiPenarikanNasabah(c *gin.Context) {
	penarikanID := c.Param("penarikan_id")

	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req services.KonfirmasiPenarikanReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := pc.Svc.Konfirmasi(penarikanID, claims.UserID, req); err != nil {
		handleServiceError(c, err)
		return
	}

	pesan := "Penarikan berhasil ditolak"
	if req.Status == string(models.StatusPenarikanApproved) {
		pesan = "Penarikan berhasil disetujui"
	}
	c.JSON(http.StatusOK, gin.H{"message": pesan})
}

func (pc *PenarikanController) BatalPenarikan(c *gin.Context) {
	penarikanID := c.Param("penarikan_id")

	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak ditemukan"})
		return
	}

	var req struct {
		Catatan string `json:"catatan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := pc.Svc.Batal(penarikanID, claims.UserID, req.Catatan); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pengajuan penarikan berhasil dibatalkan dan saldo telah dikembalikan"})
}

func (pc *PenarikanController) SelesaiPenarikan(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	req := services.SelesaiPenarikanReq{
		QRData:      c.PostForm("qr_data"),
		NasabahID:   c.PostForm("nasabah_id"),
		PenarikanID: c.PostForm("penarikan_id"),
	}
	if catatan, ada := c.GetPostForm("catatan"); ada {
		req.Catatan = &catatan
	}

	if req.QRData == "" {
		if file, header, err := c.Request.FormFile("bukti_foto"); err == nil {
			defer file.Close()

			fotoURL, errUpload := pc.CF.UploadFile(file, header, "penarikan")
			if errUpload != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah bukti foto: " + errUpload.Error()})
				return
			}
			req.BuktiFoto = fotoURL
		}
	}

	if err := pc.Svc.Selesai(claims.UserID, req); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Penarikan berhasil diselesaikan"})
}

func (pc *PenarikanController) ListPenarikanNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	f := bindPenarikanFilter(c)
	result, err := pc.Svc.GetList(nasabahID, f)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, result)
}

func (pc *PenarikanController) ListPenarikanNasabahByBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	f := bindPenarikanFilter(c)
	summaries, err := pc.Svc.GetListByBank(bankID, f)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data penarikan berhasil diambil",
		"data":    summaries,
	})
}

func (pc *PenarikanController) DetailPenarikanNasabah(c *gin.Context) {
	penarikanID := c.Param("penarikan_id")

	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	detail, err := pc.Svc.GetDetail(penarikanID, claims.UserID, claims.Role)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, detail)
}

func bindPenarikanFilter(c *gin.Context) repositories.PenarikanFilter {
	var f repositories.PenarikanFilter
	_ = c.ShouldBindQuery(&f)
	return f
}

func (pc *PenarikanController) GetSettingPenarikan(c *gin.Context) {
	setting, err := pc.Svc.GetSettingPenarikan(c.Param("bank_id"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Ketentuan penarikan berhasil diambil",
		"data":    setting,
	})
}

func (pc *PenarikanController) SimpanSettingWaktuPenarikan(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req services.SimpanSettingWaktuReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid: " + err.Error()})
		return
	}

	setting, err := pc.Svc.SimpanSettingWaktuPenarikan(c.Param("bank_id"), claims.UserID, req)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Syarat waktu penarikan berhasil disimpan",
		"data":    setting,
	})
}

func (pc *PenarikanController) SimpanSettingSaldoPenarikan(c *gin.Context) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req services.SimpanSettingSaldoReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format request tidak valid: " + err.Error()})
		return
	}

	setting, err := pc.Svc.SimpanSettingSaldoPenarikan(c.Param("bank_id"), claims.UserID, req)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Syarat saldo penarikan berhasil disimpan",
		"data":    setting,
	})
}

func handleServiceError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	svcErr, ok := err.(*services.ServiceError)
	if ok {
		resp := gin.H{"error": svcErr.Message}
		if svcErr.Data != nil {
			resp["data"] = svcErr.Data
		}
		if svcErr.ErrCode != "" {
			resp["code"] = svcErr.ErrCode
		}
		c.JSON(svcErr.Code, resp)
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	return err
}
