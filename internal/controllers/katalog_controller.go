package controllers

import (
	"net/http"

	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/utils"

	"github.com/gin-gonic/gin"
)

type KatalogController struct {
	Svc       services.KatalogService
	CFStorage *storage.CloudflareStorage
}

func NewKatalogController(svc services.KatalogService, cfStorage *storage.CloudflareStorage) *KatalogController {
	return &KatalogController{Svc: svc, CFStorage: cfStorage}
}

func aksesKatalog(c *gin.Context) (services.AksesKatalogInput, bool) {
	claims, ok := middleware.GetClaims(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tidak terautentikasi"})
		return services.AksesKatalogInput{}, false
	}
	return services.AksesKatalogInput{ActorBankID: claims.BankID, ActorRole: claims.Role}, true
}

func (kc *KatalogController) uploadFotoKatalog(c *gin.Context) (url string, ada bool, ok bool) {
	fileHeader, err := c.FormFile("foto")
	if err != nil {
		return "", false, true
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
		return "", false, false
	}
	defer file.Close()

	fotoURL, err := kc.CFStorage.UploadFile(file, fileHeader, "katalog_sampah")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto: " + err.Error()})
		return "", false, false
	}
	return fotoURL, true, true
}

func (kc *KatalogController) GetMasterSampah(c *gin.Context) {
	results, err := kc.Svc.ListMasterSampah(c.Query("q"))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": results})
}

func (kc *KatalogController) AddKatalog(c *gin.Context) {
	var req struct {
		NamaSampah      string            `form:"nama_sampah" binding:"required"`
		SarokID         int               `form:"sarok_id"`
		Satuan          models.SatuanEnum `form:"satuan" binding:"required"`
		KategoriID      int               `form:"kategori_id" binding:"required"`
		RewardID        int               `form:"reward_id" binding:"required"`
		SyaratPemilahan string            `form:"syarat_pemilahan"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fotoURL, _, ok := kc.uploadFotoKatalog(c)
	if !ok {
		return
	}

	result, err := kc.Svc.TambahKatalog(services.TambahKatalogInput{
		BankID:          c.Param("bank_id"),
		NamaSampah:      req.NamaSampah,
		SarokID:         req.SarokID,
		Satuan:          req.Satuan,
		KategoriID:      req.KategoriID,
		RewardID:        req.RewardID,
		SyaratPemilahan: req.SyaratPemilahan,
		FotoURL:         fotoURL,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Katalog berhasil ditambahkan",
		"data":    result,
	})
}

func (kc *KatalogController) EditKatalog(c *gin.Context) {
	akses, ok := aksesKatalog(c)
	if !ok {
		return
	}

	var req struct {
		SyaratPemilahan string `form:"syarat_pemilahan" binding:"required"`
		KategoriID      int    `form:"kategori_id"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	fotoURL, adaFoto, ok := kc.uploadFotoKatalog(c)
	if !ok {
		return
	}

	katalog, err := kc.Svc.EditKatalog(c.Param("sampah_id"), services.EditKatalogInput{
		SyaratPemilahan: req.SyaratPemilahan,
		KategoriID:      req.KategoriID,
		FotoURL:         fotoURL,
		AdaFileFoto:     adaFoto,
		Akses:           akses,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Katalog sampah berhasil diupdate",
		"data":    katalog,
	})
}

func (kc *KatalogController) GetKatalogSampahBank(c *gin.Context) {
	items, meta, err := kc.Svc.ListKatalogBank(c.Param("bank_id"), utils.ParsePage(c.Query("page")))
	if err := handleServiceError(c, err); err != nil {
		return
	}

	resp := gin.H{
		"message": "Katalog berhasil diambil",
		"data":    items,
	}
	if meta != nil {
		resp["pagination"] = meta
	}
	c.JSON(http.StatusOK, resp)
}

func (kc *KatalogController) GetDetailSampah(c *gin.Context) {
	akses, ok := aksesKatalog(c)
	if !ok {
		return
	}

	detail, err := kc.Svc.DetailKatalog(c.Param("sampah_id"), akses)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail sampah berhasil diambil",
		"data":    detail,
	})
}

func (kc *KatalogController) DeleteKatalogSampah(c *gin.Context) {
	akses, ok := aksesKatalog(c)
	if !ok {
		return
	}

	err := kc.Svc.HapusKatalog(services.HapusKatalogInput{
		SampahID: c.Param("sampah_id"),
		Akses:    akses,
	})
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Katalog sampah berhasil dihapus"})
}

func (kc *KatalogController) AddNewKategori(c *gin.Context) {
	var req struct {
		Kategori string `form:"kategori" binding:"required"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kategori, err := kc.Svc.TambahKategori(req.Kategori)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Kategori berhasil ditambahkan",
		"data":    kategori,
	})
}

func (kc *KatalogController) GetKategori(c *gin.Context) {
	kategori, err := kc.Svc.ListKategori()
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kategori berhasil diambil",
		"data":    kategori,
	})
}

func (kc *KatalogController) UpdateKategori(c *gin.Context) {
	var req struct {
		Kategori string `json:"kategori" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	kategori, err := kc.Svc.UbahKategori(c.Param("kategori_id"), req.Kategori)
	if err := handleServiceError(c, err); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Kategori berhasil diupdate",
		"data":    kategori,
	})
}

func (kc *KatalogController) DeleteKategori(c *gin.Context) {
	if err := handleServiceError(c, kc.Svc.HapusKategori(c.Param("kategori_id"))); err != nil {
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Kategori berhasil dihapus"})
}
