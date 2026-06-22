package controllers

import (
	"fmt"
	"net/http"
	"time"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"

	"github.com/gin-gonic/gin"
)

type PenarikanController struct {
	Svc services.PenarikanService
	CF  *storage.CloudflareStorage
}

func NewPenarikanController(svc services.PenarikanService, cf *storage.CloudflareStorage) *PenarikanController {
	return &PenarikanController{Svc: svc, CF: cf}
}

// POST /penarikan/preview/:nasabah_id
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

// POST /penarikan/ajukan/:nasabah_id
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

// POST /penarikan/konfirmasi/:penarikan_id
func (pc *PenarikanController) KonfirmasiPenarikanNasabah(c *gin.Context) {
	penarikanID := c.Param("penarikan_id")

	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak valid"})
		return
	}

	var req struct {
		BuktiFoto string `json:"bukti_foto" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fotoURL, err := pc.uploadFoto(req.BuktiFoto, penarikanID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah bukti foto: " + err.Error()})
		return
	}

	if err := pc.Svc.Konfirmasi(penarikanID, claims.UserID, fotoURL); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Penarikan berhasil dikonfirmasi"})
}

// PATCH /penarikan/batal/:penarikan_id
func (pc *PenarikanController) BatalPenarikan(c *gin.Context) {
	penarikanID := c.Param("penarikan_id")

	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesi tidak ditemukan"})
		return
	}

	if err := pc.Svc.Batal(penarikanID, claims.UserID); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Pengajuan penarikan berhasil dibatalkan dan saldo telah dikembalikan"})
}

// GET /penarikan/list/:nasabah_id
func (pc *PenarikanController) ListPenarikanNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	f := bindPenarikanFilter(c)
	result, err := pc.Svc.GetList(nasabahID, f)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, result)
}

// GET /penarikan/list-bank/:bank_id
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

// GET /penarikan/detail/:penarikan_id
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

// ── Helpers ────────────────────────────────────────────────────────────────

func (pc *PenarikanController) uploadFoto(base64Foto, penarikanID string) (string, error) {
	if base64Foto == "" {
		return "", nil
	}
	// Jika pendek → sudah berupa URL, pakai langsung
	if len(base64Foto) <= 100 {
		return base64Foto, nil
	}
	path := fmt.Sprintf("penarikan/%s_%d.jpg", penarikanID, time.Now().Unix())
	return pc.CF.UploadBase64(base64Foto, path)
}

func bindPenarikanFilter(c *gin.Context) repositories.PenarikanFilter {
	var f repositories.PenarikanFilter
	_ = c.ShouldBindQuery(&f)
	return f
}

// handleServiceError memetakan ServiceError ke HTTP response.
// Mengembalikan error non-nil jika sudah menulis response (agar caller bisa return).
func handleServiceError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	svcErr, ok := err.(*services.ServiceError)
	if ok {
		if svcErr.Data != nil {
			c.JSON(svcErr.Code, gin.H{"error": svcErr.Message, "data": svcErr.Data})
		} else {
			c.JSON(svcErr.Code, gin.H{"error": svcErr.Message})
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
	return err
}

// rolePenarikan dipertahankan agar kompatibel dengan kode yang sudah ada
func rolePenarikan(role models.RoleAdmin) string {
	if role == models.RoleNasabah {
		return "nasabah"
	}
	return "admin"
}
