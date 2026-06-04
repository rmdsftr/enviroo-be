package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type LokasiController struct {
	DB *gorm.DB
}

func NewLokasiController(db *gorm.DB) *LokasiController {
	return &LokasiController{DB: db}
}

func (lc *LokasiController) GetLokasiBankSampah(c *gin.Context) {
	type LokasiBankSampahResponse struct {
		BankID    string  `json:"bank_id" gorm:"column:bank_id"`
		NamaBank  string  `json:"nama_bank" gorm:"column:nama_bank"`
		JenisBank string  `json:"jenis_bank" gorm:"column:jenis_bank"`
		Latitude  float64 `json:"latitude" gorm:"column:latitude"`
		Longitude float64 `json:"longitude" gorm:"column:longitude"`
	}

	var results []LokasiBankSampahResponse

	query := lc.DB.Model(&models.BankSampah{}).
		Select("bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.jenis_bank, bank_sampah.latitude, bank_sampah.longitude")

	if err := query.Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank sampah locations: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah locations fetched successfully",
		"data":    results,
	})
}

// GET /lokasi/statistik-kecamatan
//
// Query params (optional):
//
//	sort=asc  → urutkan dari kecamatan dengan bank paling sedikit
//	sort=desc → urutkan dari kecamatan dengan bank paling banyak (default)
func (lc *LokasiController) StatistikBankSampahPerKecamatan(c *gin.Context) {
	sort := c.DefaultQuery("sort", "desc")
	if sort != "asc" && sort != "desc" {
		sort = "desc"
	}

	type BankItem struct {
		BankID   string `json:"bank_id"`
		NamaBank string `json:"nama_bank"`
	}

	type KecamatanStatistik struct {
		IDKecamatan int        `json:"id_kecamatan"`
		Kecamatan   string     `json:"kecamatan"`
		JumlahBank  int        `json:"jumlah_bank"`
		Banks       []BankItem `json:"banks"`
	}

	// Ambil semua kecamatan beserta jumlah bank, diurutkan sesuai sort
	type kecamatanCount struct {
		IDKecamatan int    `gorm:"column:id_kecamatan"`
		Kecamatan   string `gorm:"column:kecamatan"`
		JumlahBank  int    `gorm:"column:jumlah_bank"`
	}

	var kecamatanList []kecamatanCount
	if err := lc.DB.Table("kecamatan").
		Select("kecamatan.id_kecamatan, kecamatan.kecamatan, COUNT(bank_sampah.bank_id) as jumlah_bank").
		Joins("LEFT JOIN bank_sampah ON bank_sampah.id_kecamatan = kecamatan.id_kecamatan").
		Group("kecamatan.id_kecamatan, kecamatan.kecamatan").
		Order("jumlah_bank " + sort).
		Find(&kecamatanList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil statistik kecamatan: " + err.Error()})
		return
	}

	// Ambil semua bank sekaligus, lalu map ke kecamatan masing-masing
	type bankRow struct {
		IDKecamatan int    `gorm:"column:id_kecamatan"`
		BankID      string `gorm:"column:bank_id"`
		NamaBank    string `gorm:"column:nama_bank"`
	}

	var bankRows []bankRow
	if err := lc.DB.Table("bank_sampah").
		Select("id_kecamatan, bank_id, nama_bank").
		Where("id_kecamatan IS NOT NULL").
		Find(&bankRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bank: " + err.Error()})
		return
	}

	bankMap := make(map[int][]BankItem)
	for _, b := range bankRows {
		bankMap[b.IDKecamatan] = append(bankMap[b.IDKecamatan], BankItem{
			BankID:   b.BankID,
			NamaBank: b.NamaBank,
		})
	}

	results := make([]KecamatanStatistik, 0, len(kecamatanList))
	for _, k := range kecamatanList {
		banks := bankMap[k.IDKecamatan]
		if banks == nil {
			banks = []BankItem{}
		}
		results = append(results, KecamatanStatistik{
			IDKecamatan: k.IDKecamatan,
			Kecamatan:   k.Kecamatan,
			JumlahBank:  k.JumlahBank,
			Banks:       banks,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Statistik persebaran bank sampah per kecamatan berhasil diambil",
		"data":    results,
	})
}

// ─── KECAMATAN ────────────────────────────────────────────────────────────────

func (lc *LokasiController) GetAllKecamatan(c *gin.Context) {
	var results []models.Kecamatan
	if err := lc.DB.Order("id_kecamatan asc").Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kecamatan: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Data kecamatan berhasil diambil",
		"data":    results,
	})
}

func (lc *LokasiController) GetKecamatanByID(c *gin.Context) {
	id := c.Param("id")
	var result models.Kecamatan
	if err := lc.DB.Where("id_kecamatan = ?", id).First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kecamatan tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kecamatan: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Data kecamatan berhasil diambil",
		"data":    result,
	})
}

func (lc *LokasiController) CreateKecamatan(c *gin.Context) {
	type CreateKecamatanRequest struct {
		Kecamatan string `json:"kecamatan" binding:"required"`
	}

	var req CreateKecamatanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var dupCount int64
	lc.DB.Model(&models.Kecamatan{}).
		Where("LOWER(REGEXP_REPLACE(kecamatan, '[^a-zA-Z0-9]', '', 'g')) = ?", utils.NormalizeString(req.Kecamatan)).
		Count(&dupCount)
	if dupCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kecamatan dengan nama tersebut sudah ada"})
		return
	}

	newKecamatan := models.Kecamatan{
		Kecamatan: req.Kecamatan,
	}
	if err := lc.DB.Create(&newKecamatan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambah kecamatan: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "Kecamatan berhasil ditambahkan",
		"data":    newKecamatan,
	})
}

func (lc *LokasiController) UpdateKecamatan(c *gin.Context) {
	id := c.Param("id")

	type UpdateKecamatanRequest struct {
		Kecamatan string `json:"kecamatan" binding:"required"`
	}

	var req UpdateKecamatanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var kecamatan models.Kecamatan
	if err := lc.DB.Where("id_kecamatan = ?", id).First(&kecamatan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kecamatan tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kecamatan: " + err.Error()})
		}
		return
	}

	var dupCount int64
	lc.DB.Model(&models.Kecamatan{}).
		Where("LOWER(REGEXP_REPLACE(kecamatan, '[^a-zA-Z0-9]', '', 'g')) = ? AND id_kecamatan != ?", utils.NormalizeString(req.Kecamatan), id).
		Count(&dupCount)
	if dupCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kecamatan dengan nama tersebut sudah ada"})
		return
	}

	kecamatan.Kecamatan = req.Kecamatan
	if err := lc.DB.Save(&kecamatan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate kecamatan: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Kecamatan berhasil diupdate",
		"data":    kecamatan,
	})
}

func (lc *LokasiController) DeleteKecamatan(c *gin.Context) {
	id := c.Param("id")

	var kecamatan models.Kecamatan
	if err := lc.DB.Where("id_kecamatan = ?", id).First(&kecamatan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kecamatan tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kecamatan: " + err.Error()})
		}
		return
	}

	if err := lc.DB.Delete(&kecamatan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kecamatan: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Kecamatan berhasil dihapus"})
}

// ─── KELURAHAN ────────────────────────────────────────────────────────────────

func (lc *LokasiController) GetAllKelurahan(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "25"))
	if err != nil || limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var total int64
	if err := lc.DB.Model(&models.Kelurahan{}).Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data kelurahan: " + err.Error()})
		return
	}

	var results []models.Kelurahan
	if err := lc.DB.Preload("Kecamatan").Order("id_kelurahan asc").Offset(offset).Limit(limit).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kelurahan: " + err.Error()})
		return
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	c.JSON(http.StatusOK, gin.H{
		"message": "Data kelurahan berhasil diambil",
		"data":    results,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total_items": total,
			"total_pages": totalPages,
		},
	})
}

func (lc *LokasiController) GetKelurahanByID(c *gin.Context) {
	id := c.Param("id")
	var result models.Kelurahan
	if err := lc.DB.Preload("Kecamatan").Where("id_kelurahan = ?", id).First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kelurahan tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kelurahan: " + err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Data kelurahan berhasil diambil",
		"data":    result,
	})
}

// GetKelurahanByKecamatan mengembalikan semua kelurahan dalam satu kecamatan
func (lc *LokasiController) GetKelurahanByKecamatan(c *gin.Context) {
	kecamatanID := c.Param("id")
	var results []models.Kelurahan
	if err := lc.DB.Preload("Kecamatan").
		Where("id_kecamatan = ?", kecamatanID).
		Order("kelurahan asc").
		Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kelurahan: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Data kelurahan berhasil diambil",
		"data":    results,
	})
}

func (lc *LokasiController) CreateKelurahan(c *gin.Context) {
	type CreateKelurahanRequest struct {
		IDKecamatan int    `json:"id_kecamatan" binding:"required"`
		Kelurahan   string `json:"kelurahan" binding:"required"`
	}

	var req CreateKelurahanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validasi kecamatan exists
	var kecamatan models.Kecamatan
	if err := lc.DB.Where("id_kecamatan = ?", req.IDKecamatan).First(&kecamatan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Kecamatan tidak ditemukan"})
		return
	}

	var dupCount int64
	lc.DB.Model(&models.Kelurahan{}).
		Where("id_kecamatan = ? AND LOWER(REGEXP_REPLACE(kelurahan, '[^a-zA-Z0-9]', '', 'g')) = ?", req.IDKecamatan, utils.NormalizeString(req.Kelurahan)).
		Count(&dupCount)
	if dupCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kelurahan dengan nama tersebut sudah ada di kecamatan ini"})
		return
	}

	newKelurahan := models.Kelurahan{
		IDKecamatan: req.IDKecamatan,
		Kelurahan:   req.Kelurahan,
	}
	if err := lc.DB.Create(&newKelurahan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambah kelurahan: " + err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"message": "Kelurahan berhasil ditambahkan",
		"data":    newKelurahan,
	})
}

func (lc *LokasiController) UpdateKelurahan(c *gin.Context) {
	id := c.Param("id")

	type UpdateKelurahanRequest struct {
		IDKecamatan int    `json:"id_kecamatan"`
		Kelurahan   string `json:"kelurahan" binding:"required"`
	}

	var req UpdateKelurahanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var kelurahan models.Kelurahan
	if err := lc.DB.Where("id_kelurahan = ?", id).First(&kelurahan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kelurahan tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kelurahan: " + err.Error()})
		}
		return
	}

	kelurahan.Kelurahan = req.Kelurahan
	if req.IDKecamatan != 0 {
		var kecamatan models.Kecamatan
		if err := lc.DB.Where("id_kecamatan = ?", req.IDKecamatan).First(&kecamatan).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kecamatan tidak ditemukan"})
			return
		}
		kelurahan.IDKecamatan = req.IDKecamatan
	}

	targetKecamatan := kelurahan.IDKecamatan
	var dupCount int64
	lc.DB.Model(&models.Kelurahan{}).
		Where("id_kecamatan = ? AND LOWER(REGEXP_REPLACE(kelurahan, '[^a-zA-Z0-9]', '', 'g')) = ? AND id_kelurahan != ?", targetKecamatan, utils.NormalizeString(req.Kelurahan), id).
		Count(&dupCount)
	if dupCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Kelurahan dengan nama tersebut sudah ada di kecamatan ini"})
		return
	}

	if err := lc.DB.Save(&kelurahan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate kelurahan: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Kelurahan berhasil diupdate",
		"data":    kelurahan,
	})
}

func (lc *LokasiController) DeleteKelurahan(c *gin.Context) {
	id := c.Param("id")

	var kelurahan models.Kelurahan
	if err := lc.DB.Where("id_kelurahan = ?", id).First(&kelurahan).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Kelurahan tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data kelurahan: " + err.Error()})
		}
		return
	}

	if err := lc.DB.Delete(&kelurahan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus kelurahan: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Kelurahan berhasil dihapus"})
}
