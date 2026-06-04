package controllers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var errNasabahBankMismatch = errors.New("nasabah tidak terdaftar di bank sampah ini")

type SetoranController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
	NotifSvc  services.NotifikasiService
}

func NewSetoranController(db *gorm.DB, cfStorage *storage.CloudflareStorage, notifSvc services.NotifikasiService) *SetoranController {
	return &SetoranController{
		DB:        db,
		CFStorage: cfStorage,
		NotifSvc:  notifSvc,
	}
}

// VerifikasiSetoranNasabah memvalidasi apakah penimbangan, admin, dan nasabah
// memenuhi syarat untuk melakukan setoran.
// GET /setoran/verifikasi/:penimbangan_id/:nasabah_id/:admin_id
func (sc *SetoranController) VerifikasiSetoranNasabah(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")
	nasabahID := c.Param("nasabah_id")
	adminID := c.Param("admin_id")

	// 1. Cek nasabah aktif
	var nasabah models.Nasabah
	if err := sc.DB.Preload("User").Where("nasabah_id = ? AND status_nasabah = ?", nasabahID, models.Aktif).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "unverified",
			"message": "Nasabah tidak aktif atau tidak ditemukan",
		})
		return
	}

	// 2. Cek admin aktif
	var admin models.Admin
	if err := sc.DB.Where("admin_id = ? AND status_admin = ?", adminID, models.Aktif).First(&admin).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "unverified",
			"message": "Admin tidak aktif atau tidak ditemukan",
		})
		return
	}

	// 3. Cek admin tidak boleh verifikasi setoran dirinya sendiri sebagai nasabah
	if nasabah.UserID == admin.UserID {
		c.JSON(http.StatusOK, gin.H{
			"status":  "unverified",
			"message": "Petugas tidak boleh mengisi setoran sampah sebagai nasabah sendiri",
		})
		return
	}

	// 4. Cek penimbangan aktif
	var penimbangan models.Penimbangan
	if err := sc.DB.Where("penimbangan_id = ? AND status_penimbangan = ?", penimbanganID, models.StatusAktif).First(&penimbangan).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "unverified",
			"message": "Sesi penimbangan tidak aktif atau tidak ditemukan",
		})
		return
	}

	// 5. Cek nasabah terdaftar di bank yang sama dengan penimbangan
	if penimbangan.BankID == nil || nasabah.BankID != *penimbangan.BankID {
		c.JSON(http.StatusOK, gin.H{
			"status":  "unverified",
			"message": "Nasabah tidak terdaftar di bank sampah ini",
		})
		return
	}

	// Semua valid
	c.JSON(http.StatusOK, gin.H{
		"status":  "verified",
		"message": "Semua pihak terverifikasi, setoran dapat dilanjutkan",
		"data": gin.H{
			"nasabah_id":   nasabah.NasabahID,
			"nama_nasabah": nasabah.User.Nama,
			"photo_url" : nasabah.User.PhotoURL,
		},
	})
}

func (sc *SetoranController) PreviewSetoranNasabah(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")
	nasabahID := c.Param("nasabah_id")

	// ── 1. Parse form ─────────────────────────────────────────────────────────
	itemsStr := c.PostForm("items")
	if itemsStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Data items tidak boleh kosong"})
		return
	}

	type ItemRequest struct {
		SampahID string  `json:"sampah_id"`
		Qty      float64 `json:"qty"`
	}

	var items []ItemRequest
	if err := json.Unmarshal([]byte(itemsStr), &items); err != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid"})
		return
	}

	// ── 2. Validasi nasabah ───────────────────────────────────────────────────
	var nasabah models.Nasabah
	if err := sc.DB.Preload("User").Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		return
	}

	// ── 4. Preview per item ───────────────────────────────────────────────────
	type ItemPreview struct {
		SampahID    string  `json:"sampah_id"`
		NamaSampah  string  `json:"nama_sampah"`
		JenisReward string  `json:"jenis_reward"`
		Qty         float64 `json:"qty"`
		Satuan      string  `json:"satuan"`
	}

	var itemPreviews []ItemPreview

	for _, item := range items {
		if item.Qty <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Qty harus lebih dari 0"})
			return
		}

		// Ambil data sampah + reward
		var sampah models.KatalogSampah
		if err := sc.DB.Preload("Reward").Where("sampah_id = ?", item.SampahID).First(&sampah).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Sampah tidak ditemukan: " + item.SampahID})
			return
		}

		itemPreviews = append(itemPreviews, ItemPreview{
			SampahID:    item.SampahID,
			NamaSampah:  sampah.NamaSampah,
			JenisReward: string(sampah.Reward.NamaReward),
			Qty:         item.Qty,
			Satuan:      string(sampah.Satuan),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preview setoran berhasil dihitung",
		"data": gin.H{
			"penimbangan_id": penimbanganID,
			"nasabah_id":     nasabahID,
			"nama_nasabah":   nasabah.User.Nama,
			"total_item":     len(items),
			"items":          itemPreviews,
		},
	})
}

// InputSetoranNasabah mencatat transaksi setoran nasabah beserta detail item sampahnya.
// Setiap item yang disetor akan menghasilkan satu baris di tabungan_sampah (untuk FIFO bagi hasil).
// POST /setoran/input/:penimbangan_id/:nasabah_id/:admin_id
func (sc *SetoranController) InputSetoranNasabah(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")
	nasabahID := c.Param("nasabah_id")
	adminID := c.Param("admin_id")

	var nasabah models.Nasabah
	if err := sc.DB.Where("nasabah_id = ? AND status_nasabah = ?", nasabahID, models.Aktif).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Nasabah tidak aktif atau tidak ditemukan"})
		return
	}

	var admin models.Admin
	if err := sc.DB.Where("admin_id = ? AND status_admin = ?", adminID, models.Aktif).First(&admin).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Admin tidak aktif atau tidak ditemukan"})
		return
	}

	if nasabah.UserID == admin.UserID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Petugas tidak boleh mengisi setoran sampah sebagai nasabah sendiri"})
		return
	}

	// ── Parsing Multipart Form ────────────────────────────────────────────────
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

	// Sekarang ItemRequest hanya perlu sampah_id dan qty.
	// Harga/poin tidak diinput di sini — akan dihitung saat penjualan (bagi hasil).
	type ItemRequest struct {
		SampahID string  `json:"sampah_id"`
		Qty      float64 `json:"qty"`
	}

	var items []ItemRequest
	if err := json.Unmarshal([]byte(itemsStr), &items); err != nil || len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format data items tidak valid"})
		return
	}

	// ── Handle Upload Foto Bukti ──────────────────────────────────────────────
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

	// ── Generate ID ───────────────────────────────────────────────────────────
	setoranID := utils.GenerateID("STR")
	totalItem := len(items)

	// ── Transaksi DB ──────────────────────────────────────────────────────────
	var bankID string
	err := sc.DB.Transaction(func(tx *gorm.DB) error {

		// 0. Cek penimbangan untuk dapatkan BankID
		var penimbangan models.Penimbangan
		if err := tx.Where("penimbangan_id = ? AND status_penimbangan = ?", penimbanganID, models.StatusAktif).First(&penimbangan).Error; err != nil {
			return err
		}
		if penimbangan.BankID == nil {
			return gorm.ErrRecordNotFound
		}
		bankID = *penimbangan.BankID

		if nasabah.BankID != bankID {
			return errNasabahBankMismatch
		}

		// 1. Insert setoran_nasabah (header)
		setoran := models.SetoranNasabah{
			SetoranID:      setoranID,
			AdminID:        adminID,
			NasabahID:      nasabahID,
			PenimbanganID:  penimbanganID,
			TotalItem:      totalItem,
			StatusSetoran:  models.StatusBerhasil,
			BuktiViaManual: buktiURL,
		}
		if err := tx.Create(&setoran).Error; err != nil {
			return err
		}

		// 2. Insert detail_setoran_nasabah + update stok + insert tabungan_sampah (per item)
		for _, item := range items {
			// 2a. Detail setoran (catatan fisik timbangan)
			detail := models.DetailSetoranNasabah{
				SetoranID: setoranID,
				SampahID:  item.SampahID,
				Qty:       item.Qty,
			}
			if err := tx.Create(&detail).Error; err != nil {
				return err
			}

			// 2b. Update stok_sampah di bank
			var stok models.StokSampah
			res := tx.Where("bank_id = ? AND sampah_id = ?", bankID, item.SampahID).First(&stok)
			switch res.Error {
			case gorm.ErrRecordNotFound:
				newStok := models.StokSampah{BankID: bankID, SampahID: item.SampahID, Stok: item.Qty}
				if err := tx.Create(&newStok).Error; err != nil {
					return err
				}
			case nil:
				if err := tx.Model(&stok).Update("stok", gorm.Expr("stok + ?", item.Qty)).Error; err != nil {
					return err
				}
			default:
				return res.Error
			}

			// 2c. Insert tabungan_sampah (FIFO inventory untuk bagi hasil)
			tabungan := models.TabunganSampah{
				TabunganID: utils.GenerateID("TBG"),
				NasabahID:  &nasabahID,
				BankID:     &bankID,
				SampahID:   item.SampahID,
				Entitas:    models.EntitasNasabah,
				Qty:        item.Qty,
				SisaQty:    item.Qty,  // Sisa penuh karena belum ada bagi hasil
				CreatedAt:  time.Now(),
				SourceID:   &setoranID,
			}
			if err := tx.Create(&tabungan).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if errors.Is(err, errNasabahBankMismatch) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ── Kirim notifikasi ke nasabah (fire-and-forget) ─────────────────────────
	go func() {
		var nasabah models.Nasabah
		if err := sc.DB.Preload("User").Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
			log.Printf("[Notif] Gagal ambil nasabah %s: %v", nasabahID, err)
			return
		}
		var bank models.BankSampah
		if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
			log.Printf("[Notif] Gagal ambil bank %s: %v", bankID, err)
			return
		}
		if err := sc.NotifSvc.NotifSetoranBerhasil(
			context.Background(),
			nasabah.User.UserID,
			nasabah.User.FCMToken,
			totalItem,
			bank.NamaBank,
			setoranID,
		); err != nil {
			log.Printf("[Notif] Gagal kirim notif setoran %s: %v", setoranID, err)
		}
	}()

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Setoran berhasil dicatat",
		"setoran_id": setoranID,
		"total_item": totalItem,
	})
}

// DetailSetoranNasabah mengambil detail satu transaksi setoran.
// GET /setoran/detail/:setoran_id
func (sc *SetoranController) DetailSetoranNasabah(c *gin.Context) {
	setoranID := c.Param("setoran_id")

	type itemSetoran struct {
		NamaSampah string  `json:"nama_sampah" gorm:"column:nama_sampah"`
		Satuan     string  `json:"satuan"      gorm:"column:satuan"`
		Qty        float64 `json:"qty"         gorm:"column:qty"`
	}

	type headerStrukSetoran struct {
		SetoranID          string               `json:"setoran_id"          gorm:"column:setoran_id"`
		NamaPetugas        string               `json:"nama_petugas"        gorm:"column:nama_petugas"`
		NamaNasabah        string               `json:"nama_nasabah"        gorm:"column:nama_nasabah"`
		TransaksiTimestamp time.Time            `json:"transaksi_timestamp" gorm:"column:transaksi_timestamp"`
		TotalItem          int                  `json:"total_item"          gorm:"column:total_item"`
		StatusSetoran      models.StatusSetoran `json:"status_setoran"      gorm:"column:status_setoran"`
		BuktiViaManual     string               `json:"bukti_via_manual"    gorm:"column:bukti_via_manual"`
	}

	var header headerStrukSetoran
	if err := sc.DB.Table("setoran_nasabah").
		Select(`setoran_nasabah.setoran_id,
			u_petugas.nama as nama_petugas,
			u_nasabah.nama as nama_nasabah,
			setoran_nasabah.created_at as transaksi_timestamp,
			setoran_nasabah.total_item,
			setoran_nasabah.status_setoran,
			setoran_nasabah.bukti_via_manual`).
		Joins("LEFT JOIN admin ON admin.admin_id = setoran_nasabah.admin_id").
		Joins("LEFT JOIN users u_petugas ON u_petugas.user_id = admin.user_id").
		Joins("LEFT JOIN nasabah ON nasabah.nasabah_id = setoran_nasabah.nasabah_id").
		Joins("LEFT JOIN users u_nasabah ON u_nasabah.user_id = nasabah.user_id").
		Where("setoran_nasabah.setoran_id = ?", setoranID).
		First(&header).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data setoran tidak ditemukan"})
		return
	}

	var listItems []itemSetoran
	if err := sc.DB.Table("detail_setoran_nasabah").
		Select("katalog_sampah.nama_sampah, katalog_sampah.satuan, detail_setoran_nasabah.qty").
		Joins("LEFT JOIN katalog_sampah ON katalog_sampah.sampah_id = detail_setoran_nasabah.sampah_id").
		Where("detail_setoran_nasabah.setoran_id = ?", setoranID).
		Find(&listItems).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail item setoran"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil detail setoran",
		"data": gin.H{
			"header": header,
			"items":  listItems,
		},
	})
}

// ListRiwayatSetoranNasabah mengambil daftar semua setoran milik nasabah.
// GET /setoran/riwayat/:nasabah_id
func (sc *SetoranController) ListRiwayatSetoranNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	type RiwayatSummary struct {
		SetoranID          string               `json:"setoran_id"          gorm:"column:setoran_id"`
		NamaPetugas        string               `json:"nama_petugas"        gorm:"column:nama_petugas"`
		TransaksiTimestamp time.Time            `json:"transaksi_timestamp" gorm:"column:transaksi_timestamp"`
		TotalItem          int                  `json:"total_item"          gorm:"column:total_item"`
		StatusSetoran      models.StatusSetoran `json:"status_setoran"      gorm:"column:status_setoran"`
	}

	var history []RiwayatSummary

	if err := sc.DB.Table("setoran_nasabah").
		Select(`setoran_nasabah.setoran_id,
			u_petugas.nama as nama_petugas,
			setoran_nasabah.created_at as transaksi_timestamp,
			setoran_nasabah.total_item,
			setoran_nasabah.status_setoran`).
		Joins("LEFT JOIN admin ON admin.admin_id = setoran_nasabah.admin_id").
		Joins("LEFT JOIN users u_petugas ON u_petugas.user_id = admin.user_id").
		Where("setoran_nasabah.nasabah_id = ?", nasabahID).
		Order("setoran_nasabah.created_at DESC").
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat setoran"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil riwayat setoran nasabah",
		"data":    history,
	})
}
