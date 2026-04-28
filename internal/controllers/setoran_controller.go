package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SetoranController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
}

func NewSetoranController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *SetoranController {
	return &SetoranController{
		DB:        db,
		CFStorage: cfStorage,
	}
}

// VerifikasiSetoranNasabah memvalidasi apakah penimbangan, admin, dan nasabah
// memenuhi syarat untuk melakukan setoran.
// GET /setoran/verifikasi/:penimbangan_id/:nasabah_id/:admin_id
func (sc *SetoranController) VerifikasiSetoranNasabah(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")
	nasabahID := c.Param("nasabah_id")
	adminID := c.Param("admin_id")

	// 1. Cek penimbangan aktif
	var penimbangan models.Penimbangan
	if err := sc.DB.Where("penimbangan_id = ? AND status_penimbangan = ?", penimbanganID, models.StatusAktif).First(&penimbangan).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "unverified",
			"message": "Sesi penimbangan tidak aktif atau tidak ditemukan",
		})
		return
	}

	// 2. Cek nasabah aktif
	var nasabah models.Nasabah
	if err := sc.DB.Where("nasabah_id = ? AND status_nasabah = ?", nasabahID, models.StatusAktif).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "unverified",
			"message": "Nasabah tidak aktif atau tidak ditemukan",
		})
		return
	}

	// 3. Cek admin aktif
	var admin models.Admin
	if err := sc.DB.Where("admin_id = ? AND status_admin = ?", adminID, models.StatusAktif).First(&admin).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status":  "unverified",
			"message": "Admin tidak aktif atau tidak ditemukan",
		})
		return
	}

	// Semua valid
	c.JSON(http.StatusOK, gin.H{
		"status":  "verified",
		"message": "Semua pihak terverifikasi, setoran dapat dilanjutkan",
	})
}

// InputSetoranNasabah mencatat transaksi setoran nasabah beserta detail item sampahnya.
// POST /setoran/input/:penimbangan_id/:nasabah_id/:admin_id
func (sc *SetoranController) InputSetoranNasabah(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")
	nasabahID := c.Param("nasabah_id")
	adminID := c.Param("admin_id")

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

	type ItemRequest struct {
		SampahID  string  `json:"sampah_id" binding:"required"`
		Qty       float64 `json:"qty" binding:"required"`
		NilaiPoin float64 `json:"nilai_poin" binding:"required"`
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

		// Upload ke Cloudflare
		url, err := sc.CFStorage.UploadFile(file, fileHeader, "bukti_setoran")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengunggah foto bukti: " + err.Error()})
			return
		}
		buktiURL = url
	}

	// ── Generate setoran_id ───────────────────────────────────────────────────
	now := time.Now()
	setoranID := utils.GenerateID("STR")
	transaksiID := utils.GenerateID("TRX")

	// ── Hitung total ──────────────────────────────────────────────────────────
	totalItem := len(items)
	totalPoin := float64(0)
	for _, item := range items {
		totalPoin += item.Qty * item.NilaiPoin
	}

	// ── Transaksi DB ──────────────────────────────────────────────────────────
	err := sc.DB.Transaction(func(tx *gorm.DB) error {

		// 0. Cek penimbangan untuk dapatkan BankID
		var penimbangan models.Penimbangan
		if err := tx.Where("penimbangan_id = ?", penimbanganID).First(&penimbangan).Error; err != nil {
			return fmt.Errorf("sesi penimbangan tidak ditemukan: %w", err)
		}

		// 1. Insert setoran_nasabah (header)
		setoran := models.SetoranNasabah{
			SetoranID:      setoranID,
			AdminID:        adminID,
			NasabahID:      nasabahID,
			PenimbanganID:  penimbanganID,
			TotalItem:      totalItem,
			TotalPoin:      totalPoin,
			StatusSetoran:  models.StatusBerhasil,
			BuktiViaManual: buktiURL,
		}
		if err := tx.Create(&setoran).Error; err != nil {
			return fmt.Errorf("gagal menyimpan setoran: %w", err)
		}

		// 2. Insert detail_setoran_nasabah (per item)
		for _, item := range items {
			detail := models.DetailSetoranNasabah{
				SetoranID:    setoranID,
				SampahID:     item.SampahID,
				Qty:          item.Qty,
				NilaiPoin:    item.NilaiPoin,
				SubtotalPoin: item.Qty * item.NilaiPoin,
			}
			if err := tx.Create(&detail).Error; err != nil {
				return fmt.Errorf("gagal menyimpan detail setoran (sampah_id=%s): %w", item.SampahID, err)
			}

			// Update stok di tabel stok_sampah (per bank)
			var stok models.StokSampah
			res := tx.Where("bank_id = ? AND sampah_id = ?", penimbangan.BankID, item.SampahID).First(&stok)

			switch res.Error {
			case gorm.ErrRecordNotFound:
				// Jika record stok belum ada untuk bank ini, buat baru
				newStok := models.StokSampah{
					BankID:   *penimbangan.BankID,
					SampahID: item.SampahID,
					Stok:     item.Qty,
				}
				if err := tx.Create(&newStok).Error; err != nil {
					return fmt.Errorf("gagal membuat record stok awal: %w", err)
				}
			case nil:
				// Jika sudah ada, tambahkan stoknya
				if err := tx.Model(&stok).Update("stok", gorm.Expr("stok + ?", item.Qty)).Error; err != nil {
					return fmt.Errorf("gagal memperbarui stok sampah: %w", err)
				}
			default:
				return res.Error
			}
		}

		// 3. Ambil saldo nasabah saat ini (untuk SaldoID + nilai sebelum)
		var saldo models.SaldoNasabah
		if err := tx.Where("nasabah_id = ?", nasabahID).First(&saldo).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				// Buat record saldo baru jika belum ada
				saldo = models.SaldoNasabah{
					SaldoID:       utils.GenerateID("SLD"),
					NasabahID:     nasabahID,
					TotalPoin:     0,
					LastUpdatedBy: adminID,
					LastUpdatedAt: now,
				}
				if createErr := tx.Create(&saldo).Error; createErr != nil {
					return fmt.Errorf("gagal membuat saldo nasabah baru: %w", createErr)
				}
			} else {
				return fmt.Errorf("gagal mengambil saldo nasabah: %w", err)
			}
		}
		saldoBefore := saldo.TotalPoin
		saldoAfter := saldoBefore + totalPoin

		// 4. Update saldo_nasabah — tambahkan total poin
		if err := tx.Model(&saldo).Updates(map[string]interface{}{
			"total_poin":      saldoAfter,
			"last_updated_by": adminID,
			"last_updated_at": now,
		}).Error; err != nil {
			return fmt.Errorf("gagal memperbarui saldo nasabah: %w", err)
		}

		// 5. Insert transaksi_saldo_nasabah sebagai log perubahan saldo
		newTransaksi := models.TransaksiSaldoNasabah{
			TransaksiID:    transaksiID,
			SaldoID:        saldo.SaldoID,
			JenisTransaksi: models.Setoran,
			Jumlah:         totalPoin,
			SaldoSebelum:   saldoBefore,
			SaldoSesudah:   saldoAfter,
			CreatedAt:      now,
			CreatedBy:      adminID,
		}
		if err := tx.Create(&newTransaksi).Error; err != nil {
			return fmt.Errorf("gagal menyimpan transaksi saldo: %w", err)
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Setoran berhasil dicatat",
		"setoran_id": setoranID,
		"total_item": totalItem,
		"total_poin": totalPoin,
	})
}

func (sc *SetoranController) DetailSetoranNasabah(c *gin.Context) {
	setoranID := c.Param("setoran_id")

	type itemSetoran struct {
		NamaSampah   string  `json:"nama_sampah" gorm:"column:nama_sampah"`
		Qty          float64 `json:"qty" gorm:"column:qty"`
		NilaiPoin    float64 `json:"nilai_poin" gorm:"column:nilai_poin"`
		SubtotalPoin float64 `json:"subtotal_poin" gorm:"column:subtotal_poin"`
	}

	type headerStrukSetoran struct {
		SetoranID          string               `json:"setoran_id" gorm:"column:setoran_id"`
		NamaPetugas        string               `json:"nama_petugas" gorm:"column:nama_petugas"`
		NamaNasabah        string               `json:"nama_nasabah" gorm:"column:nama_nasabah"`
		TransaksiTimestamp time.Time            `json:"transaksi_timestamp" gorm:"column:transaksi_timestamp"`
		TotalItem          int                  `json:"total_item" gorm:"column:total_item"`
		TotalPoin          float64              `json:"total_poin" gorm:"column:total_poin"`
		StatusSetoran      models.StatusSetoran `json:"status_setoran" gorm:"column:status_setoran"`
	}

	var header headerStrukSetoran
	if err := sc.DB.Table("setoran_nasabah").
		Select("setoran_nasabah.setoran_id, u_petugas.nama as nama_petugas, u_nasabah.nama as nama_nasabah, setoran_nasabah.created_at as transaksi_timestamp, setoran_nasabah.total_item, setoran_nasabah.total_poin, setoran_nasabah.status_setoran").
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
		Select("katalog_sampah.nama_sampah, detail_setoran_nasabah.qty, detail_setoran_nasabah.nilai_poin, detail_setoran_nasabah.subtotal_poin").
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

func (sc *SetoranController) ListRiwayatSetoranNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	// Struct untuk ringkasan riwayat
	type RiwayatSummary struct {
		SetoranID          string               `json:"setoran_id"`
		NamaPetugas        string               `json:"nama_petugas"`
		TransaksiTimestamp time.Time            `json:"transaksi_timestamp"`
		TotalItem          int                  `json:"total_item"`
		TotalPoin          float64              `json:"total_poin"`
		StatusSetoran      models.StatusSetoran `json:"status_setoran"`
	}

	var history []RiwayatSummary // Menggunakan slice agar bisa menampung banyak data

	// Query mengambil daftar riwayat setoran milik nasabah tertentu
	if err := sc.DB.Table("setoran_nasabah").
		Select("setoran_nasabah.setoran_id, u_petugas.nama as nama_petugas, setoran_nasabah.created_at as transaksi_timestamp, setoran_nasabah.total_item, setoran_nasabah.total_poin, setoran_nasabah.status_setoran").
		Joins("LEFT JOIN admin ON admin.admin_id = setoran_nasabah.admin_id").
		Joins("LEFT JOIN users u_petugas ON u_petugas.user_id = admin.user_id").
		Where("setoran_nasabah.nasabah_id = ?", nasabahID).
		Order("setoran_nasabah.created_at DESC"). // Urutkan dari yang terbaru
		Find(&history).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat setoran"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil riwayat setoran nasabah",
		"data":    history,
	})
}

