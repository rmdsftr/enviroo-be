package controllers

import (
	"encoding/json"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PenjualanController struct {
	Svc       *services.PenjualanService
	CFStorage *storage.CloudflareStorage
}

func NewPenjualanController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *PenjualanController {
	repo := repositories.NewPenjualanRepo(db)
	svc := services.NewPenjualanService(db, repo)
	return &PenjualanController{Svc: svc, CFStorage: cfStorage}
}

// POST /penjualan/preview/:bank_id
func (pc *PenjualanController) PreviewPenjualanEksternal(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca form data"})
		return
	}

	rewardID, items, err := parseFormPenjualan(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, svcErr := pc.Svc.PreviewPenjualan(c.Param("bank_id"), rewardID, items)
	if svcErr != nil {
		svcErr2(c, svcErr)
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /penjualan/add-eksternal/:bank_id/:admin_id
func (pc *PenjualanController) AddNewPenjualanEksternal(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca form data"})
		return
	}

	rewardID, items, err := parseFormPenjualan(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	identitas := c.PostForm("identitas_pembeli")
	if identitas == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Identitas pembeli wajib diisi"})
		return
	}

	fileHeader, err := c.FormFile("bukti_foto")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bukti foto wajib diupload"})
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca file foto"})
		return
	}
	defer file.Close()

	fotoURL, err := pc.CFStorage.UploadFile(file, fileHeader, "penjualan_eksternal")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload bukti foto: " + err.Error()})
		return
	}

	result, svcErr := pc.Svc.SubmitPenjualan(
		c.Param("bank_id"), c.Param("admin_id"),
		rewardID, identitas, fotoURL, items,
	)
	if svcErr != nil {
		svcErr2(c, svcErr)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message":           "Penjualan eksternal berhasil dicatat",
		"penjualan_id":      result.PenjualanID,
		"total_penjualan":   result.TotalPenjualan,
		"satuan":            result.Satuan,
		"status_bagi_hasil": result.StatusBagiHasil,
	})
}

// GET /penjualan/riwayat-eksternal/:bank_id
func (pc *PenjualanController) GetRiwayatPenjualanEksternal(c *gin.Context) {
	riwayat, err := pc.Svc.GetRiwayat(c.Param("bank_id"), c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		svcErr2(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Riwayat penjualan berhasil diambil", "data": riwayat})
}

// GET /penjualan/detail-eksternal/:penjualan_id
func (pc *PenjualanController) DetailPenjualanEksternal(c *gin.Context) {
	detail, err := pc.Svc.GetDetail(c.Param("penjualan_id"))
	if err != nil {
		svcErr2(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Detail penjualan berhasil diambil", "data": detail})
}

// GET /penjualan/mitra/:bank_id
func (pc *PenjualanController) GetListMitraPenjualan(c *gin.Context) {
	mitra, err := pc.Svc.GetMitra(c.Param("bank_id"))
	if err != nil {
		svcErr2(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "List mitra berhasil diambil", "data": mitra})
}

// ── helpers ─────────────────────────────────────────────────────────────────

// parseFormPenjualan extracts reward_id and items_sampah from a multipart form.
func parseFormPenjualan(c *gin.Context) (rewardID int, items []services.ItemSampahDijual, err error) {
	rewardIDStr := c.PostForm("reward_id")
	if _, scanErr := fmt.Sscan(rewardIDStr, &rewardID); scanErr != nil || rewardID <= 0 {
		return 0, nil, fmt.Errorf("reward_id tidak valid")
	}

	itemsStr := c.PostForm("items_sampah")
	if jsonErr := json.Unmarshal([]byte(itemsStr), &items); jsonErr != nil || len(items) == 0 {
		return 0, nil, fmt.Errorf("format items_sampah tidak valid atau kosong")
	}
	for _, item := range items {
		if item.HargaJual <= 0 {
			return 0, nil, fmt.Errorf("harga_jual untuk setiap item harus lebih dari 0")
		}
	}
	return rewardID, items, nil
}

// svcErr2 writes a ServiceError (or generic 500) to the response.
// Named svcErr2 to avoid collision with svcErr in distribusi_sisa_controller.go
// within the same package.
func svcErr2(c *gin.Context, err error) {
	if se, ok := err.(*services.ServiceError); ok {
		resp := gin.H{"error": se.Message}
		for k, v := range se.Data {
			resp[k] = v
		}
		c.JSON(se.Code, resp)
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
