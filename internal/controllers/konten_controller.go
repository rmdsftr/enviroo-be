package controllers

import (
	"encoding/json"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"fmt"
	"net/http"
	"time"

	"enviroo-be/pkg/utils"

	"github.com/gin-gonic/gin"
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

// BodyBlock merepresentasikan satu elemen konten (teks atau gambar).
// "index" dipakai untuk mencocokkan form field gambar, bukan disimpan ke DB.
type BodyBlock struct {
	Type     string `json:"type"`                // "text" | "image"
	Content  string `json:"content,omitempty"`   // Isi teks (jika type=text)
	MediaURL string `json:"media_url,omitempty"` // URL gambar (diisi controller saat upload)
	Index    int    `json:"index,omitempty"`     // Indeks blok gambar dari frontend
}

// POST /konten/add-konten/:bank_id/:admin_id
//
// Form fields:
//
//	judul         string (required)
//	deskripsi     string (optional)
//	body_json     string (JSON array of BodyBlock, required)
//	is_published  string "true"|"false" (optional, default: false → simpan sebagai draft)
//	thumbnail     file   (optional)
//	image_<idx>   file   (optional, satu per blok gambar; idx sesuai BodyBlock.Index)
func (kc *KontenController) AddNewKonten(c *gin.Context) {
	bankID := c.Param("bank_id")
	adminID := c.Param("admin_id")

	// ── 1. Validasi bank sampah ──────────────────────────────────────────────────
	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bank: " + err.Error()})
		}
		return
	}

	// ── 2. Validasi admin ────────────────────────────────────────────────────────
	var admin models.Admin
	if err := kc.DB.Where("admin_id = ? AND bank_id = ?", adminID, bankID).First(&admin).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Admin tidak ditemukan atau bukan milik bank ini"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data admin: " + err.Error()})
		}
		return
	}

	// ── 3. Bind form fields ──────────────────────────────────────────────────────
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

	// is_published: "true" → publish sekarang, lainnya → draft
	isPublished := c.PostForm("is_published") == "true"

	// ── 4. Upload thumbnail (opsional) ───────────────────────────────────────────
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

	// ── 5. Upload tiap blok gambar & ganti MediaURL di blok ─────────────────────
	for i, block := range blocks {
		if block.Type != "image" {
			continue
		}

		// field name: image_<index>
		fieldName := fmt.Sprintf("image_%d", block.Index)
		imgHeader, err := c.FormFile(fieldName)
		if err != nil {
			// Blok gambar tanpa file → biarkan MediaURL kosong
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

	// ── 6. Rebuild body JSON dengan URL gambar yang sudah diisi ──────────────────
	finalBodyBytes, err := json.Marshal(blocks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyusun body_json akhir"})
		return
	}

	// ── 7. Generate ID konten ────────────────────────────────────────────────────
	kontenID := utils.GenerateID("KTN")

	// ── 8. Simpan ke DB dalam satu transaksi ─────────────────────────────────────
	tx := kc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}

	newKonten := models.Konten{
		KontenID:   kontenID,
		Judul:      judul,
		Deskripsi:  deskripsi,
		Body:       string(finalBodyBytes),
		Thumbnail:  thumbnailURL,
		IsUploaded: isPublished,
		BankID:     bankID,
		AdminID:    adminID,
	}

	if err := tx.Create(&newKonten).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan konten: " + err.Error()})
		return
	}

	// Simpan tiap blok gambar sebagai record di tabel media
	for _, block := range blocks {
		if block.Type != "image" || block.MediaURL == "" {
			continue
		}
		mediaID := utils.GenerateID("MED")
		media := models.Media{
			MediaID:  mediaID,
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

// GET /konten/all-konten/:bank_id
//
// Query params (optional):
//
//	published=true  → hanya yang published
//	published=false → hanya draft
//	(kosong)        → semua konten
func (kc *KontenController) GetAllKonten(c *gin.Context) {
	bankID := c.Param("bank_id")

	// Cek bank ada
	var bank models.BankSampah
	if err := kc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bank: " + err.Error()})
		}
		return
	}

	if bank.JenisBank == models.BSU {
		bankID = *bank.ParentBankID
	}

	query := kc.DB.Where("konten.bank_id = ?", bankID)

	// Filter by published status jika query param dikirim
	if publishedParam := c.Query("published"); publishedParam != "" {
		switch publishedParam {
		case "true":
			query = query.Where("konten.is_uploaded = ?", true)
		case "false":
			query = query.Where("konten.is_uploaded = ?", false)
		}
	}

	type KontenResponse struct {
		KontenID   string    `json:"KontenID" gorm:"column:konten_id"`
		Judul      string    `json:"Judul" gorm:"column:judul"`
		Deskripsi  string    `json:"Deskripsi" gorm:"column:deskripsi"`
		Body       string    `json:"Body" gorm:"column:body"`
		Thumbnail  string    `json:"Thumbnail" gorm:"column:thumbnail"`
		IsUploaded bool      `json:"IsUploaded" gorm:"column:is_uploaded"`
		BankID     string    `json:"BankID" gorm:"column:bank_id"`
		AdminID    string    `json:"AdminID" gorm:"column:admin_id"`
		CreatedAt  time.Time `json:"CreatedAt" gorm:"column:created_at"`
		UpdatedAt  time.Time `json:"UpdatedAt" gorm:"column:updated_at"`
		NamaAdmin  string    `json:"nama_admin" gorm:"column:nama_admin"`
	}

	var kontenList []KontenResponse
	if err := query.Table("konten").
		Select("konten.*, users.nama as nama_admin").
		Joins("left join admin on konten.admin_id = admin.admin_id").
		Joins("left join users on admin.user_id = users.user_id").
		Order("konten.created_at DESC").
		Find(&kontenList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data konten: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Konten berhasil diambil",
		"data":    kontenList,
	})
}

// GET /konten/get-konten/:konten_id
func (kc *KontenController) GetKontenByID(c *gin.Context) {
	kontenID := c.Param("konten_id")

	type KontenResponse struct {
		KontenID   string    `json:"KontenID" gorm:"column:konten_id"`
		Judul      string    `json:"Judul" gorm:"column:judul"`
		Deskripsi  string    `json:"Deskripsi" gorm:"column:deskripsi"`
		Body       string    `json:"Body" gorm:"column:body"`
		Thumbnail  string    `json:"Thumbnail" gorm:"column:thumbnail"`
		IsUploaded bool      `json:"IsUploaded" gorm:"column:is_uploaded"`
		BankID     string    `json:"BankID" gorm:"column:bank_id"`
		AdminID    string    `json:"AdminID" gorm:"column:admin_id"`
		CreatedAt  time.Time `json:"CreatedAt" gorm:"column:created_at"`
		UpdatedAt  time.Time `json:"UpdatedAt" gorm:"column:updated_at"`
		NamaAdmin  string    `json:"nama_admin" gorm:"column:nama_admin"`
	}

	var konten KontenResponse
	if err := kc.DB.Table("konten").
		Select("konten.*, users.nama as nama_admin").
		Joins("left join admin on konten.admin_id = admin.admin_id").
		Joins("left join users on admin.user_id = users.user_id").
		Where("konten.konten_id = ?", kontenID).
		First(&konten).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Konten tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data konten: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Konten berhasil diambil",
		"data":    konten,
	})
}

// DELETE /konten/delete-konten/:konten_id
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

	// Hapus media terkait terlebih dahulu
	if err := kc.DB.Where("konten_id = ?", kontenID).Delete(&models.Media{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus media: " + err.Error()})
		return
	}

	// Hapus konten
	if err := kc.DB.Delete(&konten).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus konten: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Konten berhasil dihapus",
	})
}

// PATCH /konten/edit-konten/:konten_id
//
// Form fields:
//  judul         string (optional)
//  deskripsi     string (optional)
//  body_json     string (JSON array of BodyBlock, required for body update)
//  is_published  string "true"|"false" (optional)
//  thumbnail     file   (optional)
//  image_<idx>   file   (optional, for new image blocks)
func (kc *KontenController) EditKonten(c *gin.Context) {
	kontenID := c.Param("konten_id")

	// ── 1. Cari konten lama ──────────────────────────────────────────────────────
	var oldKonten models.Konten
	if err := kc.DB.Where("konten_id = ?", kontenID).First(&oldKonten).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Konten tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data konten: " + err.Error()})
		}
		return
	}

	// ── 2. Bind form fields ──────────────────────────────────────────────────────
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
		// Jika body_json tidak dikirim, pakai yang lama
		if err := json.Unmarshal([]byte(oldKonten.Body), &blocks); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses body_json lama"})
			return
		}
	}

	// ── 3. Upload thumbnail baru (jika ada) ──────────────────────────────────────
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

	// ── 4. Upload gambar blok baru ───────────────────────────────────────────────
	now := time.Now()
	for i, block := range blocks {
		if block.Type != "image" {
			continue
		}

		// Jika ada media_url asal (dari frontend), berarti gambar lama tetap dipakai.
		// Jika ada index tanpa media_url, atau ada file image_<index>, kita upload baru.
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

	// ── 5. Rebuild body JSON ─────────────────────────────────────────────────────
	finalBodyBytes, err := json.Marshal(blocks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyusun body_json akhir"})
		return
	}

	// ── 6. Update DB dalam transaksi ─────────────────────────────────────────────
	tx := kc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}

	updateData := map[string]interface{}{
		"judul":        judul,
		"deskripsi":    deskripsi,
		"body":         string(finalBodyBytes),
		"thumbnail":    thumbnailURL,
		"is_uploaded":  isPublished,
		"updated_at":   now,
	}

	if err := tx.Model(&oldKonten).Updates(updateData).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate konten: " + err.Error()})
		return
	}

	// Sinkronisasi tabel media
	// 1. Ambil semua media_url yang ada di blocks baru
	currentMediaURLs := make(map[string]bool)
	for _, b := range blocks {
		if b.Type == "image" && b.MediaURL != "" {
			currentMediaURLs[b.MediaURL] = true
		}
	}

	// 2. Hapus record media yang sudah tidak ada di blocks
	if len(currentMediaURLs) == 0 {
		// Jika tidak ada gambar sama sekali di body baru, hapus semua media terkait konten ini
		if err := tx.Where("konten_id = ?", kontenID).Delete(&models.Media{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus semua media: " + err.Error()})
			return
		}
	} else {
		// Hapus media yang media_url nya tidak ada dalam daftar saat ini
		if err := tx.Where("konten_id = ? AND media_url NOT IN ?", kontenID, getKeys(currentMediaURLs)).Delete(&models.Media{}).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal sinkronisasi media (delete): " + err.Error()})
			return
		}
	}

	// 3. Tambah record media baru yang belum ada di tabel media untuk konten ini
	var existingMedia []models.Media
	tx.Where("konten_id = ?", kontenID).Find(&existingMedia)
	existingURLs := make(map[string]bool)
	for _, m := range existingMedia {
		existingURLs[m.MediaURL] = true
	}

	for url := range currentMediaURLs {
		if !existingURLs[url] {
			mediaID := utils.GenerateID("MED")
			media := models.Media{
				MediaID:  mediaID,
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

// helper
func getKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
