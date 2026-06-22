package controllers

import (
	"encoding/json"
	"net/http"

	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"

	"github.com/gin-gonic/gin"
)

type SetoranController struct {
	Svc      services.SetoranService
	CFStorage *storage.CloudflareStorage
}

func NewSetoranController(svc services.SetoranService, cfStorage *storage.CloudflareStorage) *SetoranController {
	return &SetoranController{Svc: svc, CFStorage: cfStorage}
}

// POST /setoran/verifikasi
func (sc *SetoranController) VerifikasiSetoranNasabah(c *gin.Context) {
	var body struct {
		QRData  string `json:"qr_data"  binding:"required"`
		AdminID string `json:"admin_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result := sc.Svc.Verifikasi(body.QRData, body.AdminID)
	c.JSON(http.StatusOK, result)
}

// POST /setoran/preview/:penimbangan_id/:nasabah_id
func (sc *SetoranController) PreviewSetoranNasabah(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")
	nasabahID := c.Param("nasabah_id")

	itemsStr := c.PostForm("items")
	if itemsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data items tidak boleh kosong"})
		return
	}

	var items []services.ItemSetoranReq
	if err := json.Unmarshal([]byte(itemsStr), &items); err != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid"})
		return
	}

	result, err := sc.Svc.Preview(penimbanganID, nasabahID, items)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Preview setoran berhasil dihitung", "data": result})
}

// POST /setoran/input/:penimbangan_id/:nasabah_id/:admin_id
func (sc *SetoranController) InputSetoranNasabah(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")
	nasabahID := c.Param("nasabah_id")
	adminID := c.Param("admin_id")

	via := c.PostForm("via")
	if via == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Metode setoran (via) wajib diisi (qr/manual)"})
		return
	}

	itemsStr := c.PostForm("items")
	if itemsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data items tidak boleh kosong"})
		return
	}

	var items []services.ItemSetoranReq
	if err := json.Unmarshal([]byte(itemsStr), &items); err != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid"})
		return
	}

	var buktiURL string
	if via == "manual" {
		file, fileHeader, err := c.Request.FormFile("foto")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Foto bukti setoran wajib diunggah untuk metode manual"})
			return
		}
		defer file.Close()

		url, err := sc.CFStorage.UploadFile(file, fileHeader, "bukti_setoran")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah foto bukti: " + err.Error()})
			return
		}
		buktiURL = url
	}

	result, err := sc.Svc.Input(penimbanganID, nasabahID, adminID, via, buktiURL, items)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Setoran berhasil dicatat",
		"setoran_id": result.SetoranID,
		"total_item": result.TotalItem,
	})
}

// GET /setoran/detail/:setoran_id
func (sc *SetoranController) DetailSetoranNasabah(c *gin.Context) {
	setoranID := c.Param("setoran_id")

	detail, err := sc.Svc.GetDetail(setoranID)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil detail setoran",
		"data":    gin.H{"header": detail.Header, "items": detail.Items},
	})
}

// GET /setoran/riwayat/:nasabah_id
func (sc *SetoranController) ListRiwayatSetoranNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	rows, err := sc.Svc.GetRiwayat(nasabahID, startDate, endDate)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil riwayat setoran nasabah",
		"data":    rows,
	})
}
