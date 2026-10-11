package controllers

import (
	"encoding/json"
	"enviroo-be/internal/adapters/storage"
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"fmt"
	"net/http"
	"time"


	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type KontenController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
}

func NewKontenController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *KontenController {
	return &KontenController{
		DB:        db,
		CFStorage: cfStorage,
	}
}

type BodyBlock struct {
	Type     string `json:"type"`
	Content  string `json:"content,omitempty"`
	MediaURL string `json:"media_url,omitempty"`
	Index    int    `json:"index,omitempty"`
}

func (kc *KontenController) AddNewKonten(c *gin.Context) {
	adminID := c.Param("admin_id")

	var admin models.Admin
	if err := kc.DB.Where("admin_id = ?", adminID).First(&admin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data admin: " + err.Error()})
		}
		return
	}

	judul := c.PostForm("judul")
	if judul == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Judul tidak boleh kosong"})
		return
	}
	deskripsi := c.PostForm("deskripsi")

	bodyJSONStr := c.PostForm("body_json")
	if bodyJSONStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "body_json tidak boleh kosong"})
		return
	}

	var blocks []BodyBlock
	if err := json.Unmarshal([]byte(bodyJSONStr), &blocks); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format body_json tidak valid: " + err.Error()})
		return
	}

	isPublished := c.PostForm("is_published") == "true"

	var thumbnailURL string
	thumbHeader, err := c.FormFile("thumbnail")
	if err == nil {
		thumbFile, err := thumbHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca thumbnail"})
			return
		}
		defer thumbFile.Close()

		thumbnailURL, err = kc.CFStorage.UploadFile(thumbFile, thumbHeader, "konten_thumbnail")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload thumbnail: " + err.Error()})
			return
		}
	}

	for i, block := range blocks {
		if block.Type != "image" {
			continue
		}

		fieldName := fmt.Sprintf("image_%d", block.Index)
		imgHeader, err := c.FormFile(fieldName)
		if err != nil {
			continue
		}

		imgFile, err := imgHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gagal membaca gambar blok %d", block.Index)})
			return
		}
		defer imgFile.Close()

		imgURL, err := kc.CFStorage.UploadFile(imgFile, imgHeader, "konten_media")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Gagal upload gambar blok %d: %s", block.Index, err.Error())})
			return
		}

		blocks[i].MediaURL = imgURL
	}

	finalBodyBytes, err := json.Marshal(blocks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyusun body_json akhir"})
		return
	}

	tx := kc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}

	kontenID, err := idgen.GenerateKontenID(tx, adminID)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal generate konten ID: " + err.Error()})
		return
	}

	newKonten := models.Konten{
		KontenID:   kontenID,
		Judul:      judul,
		Deskripsi:  deskripsi,
		Body:       string(finalBodyBytes),
		Thumbnail:  thumbnailURL,
		IsUploaded: isPublished,
		BankID:     admin.BankID,
		AdminID:    adminID,
	}

	if err := tx.Create(&newKonten).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan konten: " + err.Error()})
		return
	}

	for _, block := range blocks {
		if block.Type != "image" || block.MediaURL == "" {
			continue
		}
		media := models.Media{
			MediaID:  uuid.New().String(),
			MediaURL: block.MediaURL,
			FileName: "",
			FileType: "image",
			FileSize: 0,
			AdminID:  adminID,
			KontenID: kontenID,
		}
		if err := tx.Create(&media).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan media: " + err.Error()})
			return
		}
	}

	tx.Commit()

	status := "draft"
	if isPublished {
		status = "published"
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": fmt.Sprintf("Konten berhasil disimpan sebagai %s", status),
		"data":    newKonten,
	})
}

func (kc *KontenController) GetAllKonten(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bank: " + err.Error()})
		}
		return
	}

	publishedParam := c.Query("published")

	var query *gorm.DB
	if bank.JenisBank == models.BSU {
		parentID := *bank.ParentBankID
		switch publishedParam {
		case "true":
			query = kc.DB.Where(
				"((konten.bank_id = ? OR konten.bank_id = ?) AND konten.is_uploaded = ?) OR (konten.bank_id IS NULL AND konten.is_uploaded = ?)",
				parentID, bankID, true, true,
			)
		case "false":
			query = kc.DB.Where("konten.bank_id = ? AND konten.is_uploaded = ?", bankID, false)
		default:
			query = kc.DB.Where(
				"(konten.bank_id = ? AND konten.is_uploaded = ?) OR konten.bank_id = ? OR (konten.bank_id IS NULL AND konten.is_uploaded = ?)",
				parentID, true, bankID, true,
			)
		}
	} else {
		switch publishedParam {
		case "true":
			query = kc.DB.Where(
				"(konten.bank_id = ? AND konten.is_uploaded = ?) OR (konten.bank_id IS NULL AND konten.is_uploaded = ?)",
				bankID, true, true,
			)
		case "false":
			query = kc.DB.Where("konten.bank_id = ? AND konten.is_uploaded = ?", bankID, false)
		default:
			query = kc.DB.Where(
				"konten.bank_id = ? OR (konten.bank_id IS NULL AND konten.is_uploaded = ?)",
				bankID, true,
			)
		}
	}

	type KontenListItem struct {
		KontenID     string `json:"konten_id" gorm:"column:konten_id"`
		Judul        string `json:"judul" gorm:"column:judul"`
		Deskripsi    string `json:"deskripsi" gorm:"column:deskripsi"`
		Thumbnail    string `json:"thumbnail" gorm:"column:thumbnail"`
		NamaInstansi string `json:"nama_instansi" gorm:"column:nama_instansi"`
	}

	baseQuery := query.Table("konten").
		Select("konten.konten_id, konten.judul, konten.deskripsi, konten.thumbnail, " +
			"COALESCE(bank_sampah.nama_bank, 'Dinas Lingkungan Hidup Kota Padang') as nama_instansi").
		Joins("left join bank_sampah on bank_sampah.bank_id = konten.bank_id").
		Order("konten.created_at DESC")

	const perPage = 8
	page := 1
	if p := c.Query("page"); p != "" {
		if _, err := fmt.Sscanf(p, "%d", &page); err != nil || page < 1 {
			page = 1
		}
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data konten: " + err.Error()})
		return
	}

	var kontenList []KontenListItem
	if err := baseQuery.Offset((page - 1) * perPage).Limit(perPage).Find(&kontenList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data konten: " + err.Error()})
		return
	}

	totalPages := int((total + perPage - 1) / perPage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Konten berhasil diambil",
		"data":    kontenList,
		"pagination": gin.H{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (kc *KontenController) GetKontenByID(c *gin.Context) {
	kontenID := c.Param("konten_id")

	type KontenResponse struct {
		KontenID     string    `json:"KontenID" gorm:"column:konten_id"`
		Judul        string    `json:"Judul" gorm:"column:judul"`
		Deskripsi    string    `json:"Deskripsi" gorm:"column:deskripsi"`
		Body         string    `json:"Body" gorm:"column:body"`
		Thumbnail    string    `json:"Thumbnail" gorm:"column:thumbnail"`
		IsUploaded   bool      `json:"IsUploaded" gorm:"column:is_uploaded"`
		BankID       *string   `json:"BankID" gorm:"column:bank_id"`
		AdminID      string    `json:"AdminID" gorm:"column:admin_id"`
		CreatedAt    time.Time `json:"CreatedAt" gorm:"column:created_at"`
		UpdatedAt    time.Time `json:"UpdatedAt" gorm:"column:updated_at"`
		NamaAdmin    string    `json:"nama_admin" gorm:"column:nama_admin"`
		NamaInstansi string    `json:"nama_instansi"`
	}

	var konten KontenResponse
	if err := kc.DB.Table("konten").
		Select("konten.*, users.nama as nama_admin, bank_sampah.nama_bank as nama_instansi").
		Joins("left join admin on konten.admin_id = admin.admin_id").
		Joins("left join users on admin.user_id = users.user_id").
		Joins("left join bank_sampah on bank_sampah.bank_id = konten.bank_id").
		Where("konten.konten_id = ?", kontenID).
		First(&konten).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Konten tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data konten: " + err.Error()})
		}
		return
	}

	if konten.BankID == nil {
		konten.NamaInstansi = "Dinas Lingkungan Hidup Kota Padang"
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Konten berhasil diambil",
		"data":    konten,
	})
}

func (kc *KontenController) DeleteKonten(c *gin.Context) {
	kontenID := c.Param("konten_id")

	var konten models.Konten
	if err := kc.DB.Where("konten_id = ?", kontenID).First(&konten).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Konten tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data konten: " + err.Error()})
		}
		return
	}

	if err := kc.DB.Where("konten_id = ?", kontenID).Delete(&models.Media{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus media: " + err.Error()})
		return
	}

	if err := kc.DB.Delete(&konten).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus konten: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Konten berhasil dihapus",
	})
}

func (kc *KontenController) EditKonten(c *gin.Context) {
	kontenID := c.Param("konten_id")

	var oldKonten models.Konten
	if err := kc.DB.Where("konten_id = ?", kontenID).First(&oldKonten).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Konten tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data konten: " + err.Error()})
		}
		return
	}

	judul := c.PostForm("judul")
	if judul == "" {
		judul = oldKonten.Judul
	}
	deskripsi := c.PostForm("deskripsi")
	if deskripsi == "" {
		deskripsi = oldKonten.Deskripsi
	}

	isPublishedStr := c.PostForm("is_published")
	isPublished := oldKonten.IsUploaded
	if isPublishedStr != "" {
		isPublished = isPublishedStr == "true"
	}

	bodyJSONStr := c.PostForm("body_json")
	var blocks []BodyBlock
	if bodyJSONStr != "" {
		if err := json.Unmarshal([]byte(bodyJSONStr), &blocks); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format body_json tidak valid: " + err.Error()})
			return
		}
	} else {
		if err := json.Unmarshal([]byte(oldKonten.Body), &blocks); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses body_json lama"})
			return
		}
	}

	thumbnailURL := oldKonten.Thumbnail
	thumbHeader, err := c.FormFile("thumbnail")
	if err == nil {
		thumbFile, err := thumbHeader.Open()
		if err == nil {
			defer thumbFile.Close()
			newThumbURL, err := kc.CFStorage.UploadFile(thumbFile, thumbHeader, "konten_thumbnail")
			if err == nil {
				thumbnailURL = newThumbURL
			}
		}
	}

	now := time.Now()
	for i, block := range blocks {
		if block.Type != "image" {
			continue
		}

		fieldName := fmt.Sprintf("image_%d", block.Index)
		imgHeader, err := c.FormFile(fieldName)
		if err == nil {
			imgFile, err := imgHeader.Open()
			if err == nil {
				defer imgFile.Close()
				imgURL, err := kc.CFStorage.UploadFile(imgFile, imgHeader, "konten_media")
				if err == nil {
					blocks[i].MediaURL = imgURL
				}
			}
		}
	}

	finalBodyBytes, err := json.Marshal(blocks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyusun body_json akhir"})
		return
	}

	tx := kc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}

	updateData := map[string]interface{}{
		"judul":       judul,
		"deskripsi":   deskripsi,
		"body":        string(finalBodyBytes),
		"thumbnail":   thumbnailURL,
		"is_uploaded": isPublished,
		"updated_at":  now,
	}

	if err := tx.Model(&oldKonten).Updates(updateData).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate konten: " + err.Error()})
		return
	}

	currentMediaURLs := make(map[string]bool)
	for _, b := range blocks {
		if b.Type == "image" && b.MediaURL != "" {
			currentMediaURLs[b.MediaURL] = true
		}
	}

	if len(currentMediaURLs) == 0 {
		if err := tx.Where("konten_id = ?", kontenID).Delete(&models.Media{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus semua media: " + err.Error()})
			return
		}
	} else {
		if err := tx.Where("konten_id = ? AND media_url NOT IN ?", kontenID, getKeys(currentMediaURLs)).Delete(&models.Media{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal sinkronisasi media (delete): " + err.Error()})
			return
		}
	}

	var existingMedia []models.Media
	tx.Where("konten_id = ?", kontenID).Find(&existingMedia)
	existingURLs := make(map[string]bool)
	for _, m := range existingMedia {
		existingURLs[m.MediaURL] = true
	}

	for url := range currentMediaURLs {
		if !existingURLs[url] {
			media := models.Media{
				MediaID:  uuid.New().String(),
				MediaURL: url,
				FileType: "image",
				AdminID:  oldKonten.AdminID,
				KontenID: kontenID,
			}
			if err := tx.Create(&media).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan media baru: " + err.Error()})
				return
			}
		}
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Konten berhasil diperbarui",
		"data":    oldKonten,
	})
}

func (kc *KontenController) GetAllKontenSuperadmin(c *gin.Context) {
	publishedParam := c.Query("published")

	type KontenListItem struct {
		KontenID     string `json:"konten_id" gorm:"column:konten_id"`
		Judul        string `json:"judul" gorm:"column:judul"`
		Deskripsi    string `json:"deskripsi" gorm:"column:deskripsi"`
		Thumbnail    string `json:"thumbnail" gorm:"column:thumbnail"`
		NamaInstansi string `json:"nama_instansi" gorm:"column:nama_instansi"`
	}

	baseQuery := kc.DB.Table("konten").
		Select("konten.konten_id, konten.judul, konten.deskripsi, konten.thumbnail, " +
			"COALESCE(bank_sampah.nama_bank, 'Dinas Lingkungan Hidup Kota Padang') as nama_instansi").
		Joins("left join bank_sampah on bank_sampah.bank_id = konten.bank_id").
		Order("konten.created_at DESC")

	switch publishedParam {
	case "true":
		baseQuery = baseQuery.Where("konten.is_uploaded = ?", true)
	case "false":
		baseQuery = baseQuery.Where("konten.is_uploaded = ?", false)
	}

	const perPage = 8
	page := 1
	if p := c.Query("page"); p != "" {
		if _, err := fmt.Sscanf(p, "%d", &page); err != nil || page < 1 {
			page = 1
		}
	}

	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghitung data konten: " + err.Error()})
		return
	}

	var kontenList []KontenListItem
	if err := baseQuery.Offset((page - 1) * perPage).Limit(perPage).Find(&kontenList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data konten: " + err.Error()})
		return
	}

	totalPages := int((total + perPage - 1) / perPage)

	c.JSON(http.StatusOK, gin.H{
		"message": "Konten berhasil diambil",
		"data":    kontenList,
		"pagination": gin.H{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
