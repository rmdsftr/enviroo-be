package controllers

import (
	"context"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/utils"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type BagiHasilController struct {
	DB       *gorm.DB
	NotifSvc services.NotifikasiService
}

func NewBagiHasilController(db *gorm.DB, notifSvc services.NotifikasiService) *BagiHasilController {
	return &BagiHasilController{DB: db, NotifSvc: notifSvc}
}

// ─── PreviewHitungBagiHasil ───────────────────────────────────────────────────
// GET /bagi-hasil/preview/:bank_id/:penjualan_id
func (bhc *BagiHasilController) PreviewHitungBagiHasil(c *gin.Context) {
	penjualanID := c.Param("penjualan_id")
	bankID := c.Param("bank_id")

	var penjualan models.Penjualan
	if err := bhc.DB.Preload("Reward").
		Where("penjualan_id = ? AND bank_id = ?", penjualanID, bankID).
		First(&penjualan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Penjualan tidak ditemukan"})
		return
	}
	if penjualan.StatusBagiHasil == models.BagiHasilBerhasil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bagi hasil untuk penjualan ini sudah dilakukan"})
		return
	}

	var bank models.BankSampah
	if err := bhc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusForbidden, gin.H{"error": "BSU tidak dapat melakukan bagi hasil"})
		return
	}

	var detailPenjualans []models.DetailPenjualan
	if err := bhc.DB.Where("penjualan_id = ?", penjualanID).Find(&detailPenjualans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail penjualan"})
		return
	}

	totalDibagikanKeNasabah := make(map[string]float64)
	nasabahQtyPerTabungan := make(map[string]map[string]float64) // nasabahID -> tabunganID -> qty

	bankIDsNasabah := []string{bankID}
	if bank.JenisBank == models.BSI {
		var daftarBSU []models.BankSampah
		if err := bhc.DB.Where("parent_bank_id = ? AND jenis_bank = ?", bankID, models.BSU).
			Find(&daftarBSU).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar BSU"})
			return
		}
		for _, bsu := range daftarBSU {
			bankIDsNasabah = append(bankIDsNasabah, bsu.BankID)
		}
	}

	for _, detail := range detailPenjualans {
		if detail.Qty <= 0 || detail.HargaNasabahSnapshot <= 0 {
			continue
		}
		var tabunganNasabah []models.TabunganSampah
		if err := bhc.DB.Where("entitas = ? AND sampah_id = ? AND sisa_qty > 0 AND bank_id IN ?",
			models.EntitasNasabah, detail.SampahID, bankIDsNasabah).
			Order("created_at ASC").Find(&tabunganNasabah).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil tabungan nasabah"})
			return
		}
		sisaQty := detail.Qty
		for _, tabungan := range tabunganNasabah {
			if sisaQty <= 0 {
				break
			}
			nasabahID := *tabungan.NasabahID
			qtyDiambil := math.Min(tabungan.SisaQty, sisaQty)
			totalDibagikanKeNasabah[nasabahID] += qtyDiambil * detail.HargaNasabahSnapshot
			if nasabahQtyPerTabungan[nasabahID] == nil {
				nasabahQtyPerTabungan[nasabahID] = make(map[string]float64)
			}
			nasabahQtyPerTabungan[nasabahID][tabungan.TabunganID] += qtyDiambil
			sisaQty -= qtyDiambil
		}
		if sisaQty > 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "Stok tabungan nasabah tidak mencukupi", "sampah_id": detail.SampahID, "kurang_qty": sisaQty,
			})
			return
		}
	}

	grossBank := penjualan.TotalPenjualan
	var totalDistribusiNasabah float64
	for _, v := range totalDibagikanKeNasabah {
		totalDistribusiNasabah += v
	}
	sisaBagiHasil := grossBank - totalDistribusiNasabah

	type NasabahInfo struct {
		NamaNasabah string
		NamaBank    string
		BankID      string
	}
	nasabahInfoCache := make(map[string]NasabahInfo)

	if len(totalDibagikanKeNasabah) > 0 {
		nasabahIDs := make([]string, 0, len(totalDibagikanKeNasabah))
		for id := range totalDibagikanKeNasabah {
			nasabahIDs = append(nasabahIDs, id)
		}
		type nasabahRow struct {
			NasabahID   string
			NamaNasabah string
			BankID      string
			NamaBank    string
		}
		var rows []nasabahRow
		bhc.DB.Table("nasabah").
			Select("nasabah.nasabah_id, users.nama AS nama_nasabah, nasabah.bank_id, bank_sampah.nama_bank").
			Joins("JOIN users ON users.user_id = nasabah.user_id").
			Joins("JOIN bank_sampah ON bank_sampah.bank_id = nasabah.bank_id").
			Where("nasabah.nasabah_id IN ?", nasabahIDs).
			Scan(&rows)
		for _, r := range rows {
			nasabahInfoCache[r.NasabahID] = NasabahInfo{
				NamaNasabah: r.NamaNasabah,
				NamaBank:    r.NamaBank,
				BankID:      r.BankID,
			}
		}
	}

	type NasabahPenerimaPreview struct {
		NasabahID     string  `json:"nasabah_id"`
		NamaNasabah   string  `json:"nama_nasabah"`
		TotalDiterima float64 `json:"total_diterima"`
	}
	type BankPenerimaPreview struct {
		BankID          string                   `json:"bank_id"`
		NamaBank        string                   `json:"nama_bank"`
		NasabahPenerima []NasabahPenerimaPreview `json:"nasabah_penerima"`
	}

	// Kelompokkan nasabah berdasarkan bank_id-nya
	bankMap := make(map[string]*BankPenerimaPreview)
	bankOrder := []string{} // untuk menjaga urutan bank konsisten
	for nasabahID, total := range totalDibagikanKeNasabah {
		info := nasabahInfoCache[nasabahID]
		if _, ada := bankMap[info.BankID]; !ada {
			bankMap[info.BankID] = &BankPenerimaPreview{
				BankID:          info.BankID,
				NamaBank:        info.NamaBank,
				NasabahPenerima: []NasabahPenerimaPreview{},
			}
			bankOrder = append(bankOrder, info.BankID)
		}
		bankMap[info.BankID].NasabahPenerima = append(bankMap[info.BankID].NasabahPenerima, NasabahPenerimaPreview{
			NasabahID:     nasabahID,
			NamaNasabah:   info.NamaNasabah,
			TotalDiterima: total,
		})
	}

	var penerima []BankPenerimaPreview
	for _, bankID := range bankOrder {
		penerima = append(penerima, *bankMap[bankID])
	}

	c.JSON(http.StatusOK, gin.H{
		"penjualan_id": penjualanID,
		"reward":       penjualan.Reward.NamaReward,
		"summary": gin.H{
			"gross_bank":               grossBank,
			"total_distribusi_nasabah": totalDistribusiNasabah,
			"sisa_bagi_hasil":          sisaBagiHasil,
		},
		"penerima": penerima,
	})
}

// ─── SubmitBagiHasil ──────────────────────────────────────────────────────────
// POST /bagi-hasil/submit/:bank_id/:penjualan_id
func (bhc *BagiHasilController) SubmitBagiHasil(c *gin.Context) {
	penjualanID := c.Param("penjualan_id")
	bankID := c.Param("bank_id")

	var req struct {
		AdminID string `json:"admin_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ── 1. Validasi penjualan ─────────────────────────────────────────────────
	var penjualan models.Penjualan
	if err := bhc.DB.Preload("Reward").
		Where("penjualan_id = ? AND bank_id = ?", penjualanID, bankID).
		First(&penjualan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Penjualan tidak ditemukan"})
		return
	}
	if penjualan.StatusBagiHasil == models.BagiHasilBerhasil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bagi hasil untuk penjualan ini sudah dilakukan"})
		return
	}

	// ── 2. Validasi bank ──────────────────────────────────────────────────────
	var bank models.BankSampah
	if err := bhc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusForbidden, gin.H{"error": "BSU tidak dapat melakukan bagi hasil"})
		return
	}

	// ── 3. Ambil detail penjualan ─────────────────────────────────────────────
	var detailPenjualans []models.DetailPenjualan
	if err := bhc.DB.Where("penjualan_id = ?", penjualanID).Find(&detailPenjualans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail penjualan"})
		return
	}

	// ── Persiapan data notifikasi ────────────────────────────────────────────
	type bsuNotifData struct {
		penerimaID string
		bsuID      string
		nettBSU    float64
		satuan     models.SatuanRewardEnum
	}
	type nasabahNotifData struct {
		penerimaID string
		nasabahID  string
		total      float64
		satuan     models.SatuanRewardEnum
	}

	var bsuNotifs []bsuNotifData
	var nasabahNotifs []nasabahNotifData

	// ── 4. Mulai transaksi ────────────────────────────────────────────────────
	tx := bhc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}
	now := time.Now()

	type itemAccum struct {
		sampahID string
		qty      float64
		harga    float64
		subtotal float64
	}

	nasabahItems := make(map[string]map[string]*itemAccum)
	nasabahTabungan := make(map[string]map[string]float64) // nasabahID -> tabunganID -> qtyDipakai
	nasabahTotals := make(map[string]float64)

	bankIDsNasabah := []string{bankID}
	if bank.JenisBank == models.BSI {
		var daftarBSU []models.BankSampah
		if err := tx.Where("parent_bank_id = ? AND jenis_bank = ?", bankID, models.BSU).
			Find(&daftarBSU).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar BSU"})
			return
		}
		for _, bsu := range daftarBSU {
			bankIDsNasabah = append(bankIDsNasabah, bsu.BankID)
		}
	}

	// ── 5. Akumulasi tabungan nasabah + update sisa_qty ───────────────────────
	for _, detail := range detailPenjualans {
		if detail.Qty <= 0 {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "qty tidak valid", "sampah_id": detail.SampahID})
			return
		}

		// 5a. Bagi hasil ke nasabah
		if detail.HargaNasabahSnapshot > 0 {
			var tabunganNasabah []models.TabunganSampah
			if err := tx.Where("entitas = ? AND sampah_id = ? AND sisa_qty > 0 AND bank_id IN ?",
				models.EntitasNasabah, detail.SampahID, bankIDsNasabah).
				Order("created_at ASC").Find(&tabunganNasabah).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil tabungan nasabah"})
				return
			}
			sisaQty := detail.Qty
			for i := range tabunganNasabah {
				if sisaQty <= 0 {
					break
				}
				tabungan := &tabunganNasabah[i]
				nasabahID := *tabungan.NasabahID
				qtyDiambil := math.Min(tabungan.SisaQty, sisaQty)
				nilaiDapat := qtyDiambil * detail.HargaNasabahSnapshot

				if nasabahItems[nasabahID] == nil {
					nasabahItems[nasabahID] = make(map[string]*itemAccum)
				}
				if nasabahItems[nasabahID][detail.SampahID] == nil {
					nasabahItems[nasabahID][detail.SampahID] = &itemAccum{sampahID: detail.SampahID, harga: detail.HargaNasabahSnapshot}
				}
				nasabahItems[nasabahID][detail.SampahID].qty += qtyDiambil
				nasabahItems[nasabahID][detail.SampahID].subtotal += nilaiDapat
				if nasabahTabungan[nasabahID] == nil {
					nasabahTabungan[nasabahID] = make(map[string]float64)
				}
				nasabahTabungan[nasabahID][tabungan.TabunganID] += qtyDiambil
				nasabahTotals[nasabahID] += nilaiDapat
				sisaQty -= qtyDiambil

				tabungan.SisaQty -= qtyDiambil
				if err := tx.Save(tabungan).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update tabungan nasabah"})
					return
				}
			}
			if sisaQty > 0 {
				tx.Rollback()
				c.JSON(http.StatusUnprocessableEntity, gin.H{
					"error": "Stok tabungan nasabah tidak mencukupi", "sampah_id": detail.SampahID, "kurang_qty": sisaQty,
				})
				return
			}
		}

		// 5b. Update schema harga nasabah
		if detail.HargaNasabahSnapshot > 0 {
			var schemaNasabah models.SchemaHargaSampah
			if err := tx.Where("sampah_id = ? AND level_user = ?", detail.SampahID, models.LevelNasabah).
				First(&schemaNasabah).Error; err != nil {
				if err := tx.Create(&models.SchemaHargaSampah{
					SampahID: detail.SampahID, LevelUser: models.LevelNasabah,
					Harga: detail.HargaNasabahSnapshot, SatuanReward: penjualan.SatuanReward,
				}).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat schema harga nasabah"})
					return
				}
			} else if schemaNasabah.Harga != detail.HargaNasabahSnapshot {
				if err := tx.Create(&models.KatalogSampahHistory{
					SchemaID: int(schemaNasabah.SchemaID), HargaLama: schemaNasabah.Harga,
					HargaBaru: detail.HargaNasabahSnapshot, ChangedBy: req.AdminID,
				}).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mencatat history harga nasabah"})
					return
				}
				schemaNasabah.Harga = detail.HargaNasabahSnapshot
				if err := tx.Save(&schemaNasabah).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update schema harga nasabah"})
					return
				}
			}
		}
	}

	// ── 6. Hitung summary totals ──────────────────────────────────────────────
	grossBank := penjualan.TotalPenjualan
	var totalDistribusiNasabah float64
	for _, v := range nasabahTotals {
		totalDistribusiNasabah += v
	}
	sisaBagiHasil := grossBank - totalDistribusiNasabah

	// ── 8. Buat satu record BagiHasil (bank-level summary) ───────────────────
	bhID := utils.GenerateID("BGH")
	bh := models.BagiHasil{
		BagiHasilID:            bhID,
		BankID:                 &bankID,
		PenjualanID:            &penjualanID,
		GrossBank:              grossBank,
		TotalDistribusiNasabah: totalDistribusiNasabah,
		SisaBagiHasil:          sisaBagiHasil,
		SatuanBagiHasil:        penjualan.SatuanReward,
		CreatedAt:              now,
		CreatedBy:              &req.AdminID,
	}
	if err := tx.Create(&bh).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan bagi hasil"})
		return
	}

	// ── 9. PenerimaBagiHasil + detail + pencairan per nasabah ───────────────
	for nasabahID, sampahMap := range nasabahItems {
		nasabahIDCopy := nasabahID
		total := nasabahTotals[nasabahID]

		penerimaID := utils.GenerateID("PBH")

		nasabahNotifs = append(nasabahNotifs, nasabahNotifData{
			penerimaID: penerimaID,
			nasabahID:  nasabahIDCopy,
			total:      total,
			satuan:     penjualan.SatuanReward,
		})

		pbh := models.PenerimaBagiHasil{
			PenerimaID:     penerimaID,
			BagiHasilID:    &bhID,
			RewardID:       &penjualan.RewardID,
			NasabahID:      &nasabahIDCopy,
			TotalItem:      len(sampahMap),
			TotalDiterima:  total,
			SatuanDiterima: penjualan.SatuanReward,
			CreatedAt:      now,
			CreatedBy:      &req.AdminID,
		}
		if err := tx.Create(&pbh).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan penerima bagi hasil nasabah"})
			return
		}
		for _, item := range sampahMap {
			if err := tx.Create(&models.DetailBagiHasil{
				PenerimaID: penerimaID, SampahID: item.sampahID,
				Qty: item.qty, HargaItem: item.harga, SubtotalHarga: item.subtotal,
			}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan detail bagi hasil nasabah"})
				return
			}
		}
		for tabunganID, qtyDipakai := range nasabahTabungan[nasabahID] {
			if err := tx.Create(&models.PencairanTabungan{
				PenerimaID: penerimaID,
				TabunganID: tabunganID,
				QtyDipakai: qtyDipakai,
			}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan pencairan tabungan nasabah"})
				return
			}
		}

		if err := updateSaldoRekening(tx, nil, &nasabahIDCopy, penjualan.RewardID, penjualan.Reward,
			total, models.EntitasNasabah, req.AdminID, now); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update saldo nasabah: " + err.Error()})
			return
		}
	}

	// ── 10. Update saldo bank ──────────────────────────────────────────────────
	// BSM (semua reward) dan BSI Sembako: sisa langsung masuk ke saldo rekening bank.
	// BSI Uang: sisa ditahan di sisa_bagi_hasil, baru dibagi ke BSU saat SubmitDistribusiSisa.
	bankGetsSisa := bank.JenisBank == models.BSM ||
		(bank.JenisBank == models.BSI && penjualan.Reward.NamaReward == models.RewardEnumSembako)
	if bankGetsSisa && sisaBagiHasil > 0 {
		if err := updateSaldoRekening(tx, &bankID, nil, penjualan.RewardID, penjualan.Reward,
			sisaBagiHasil, models.EntitasBankSampah, req.AdminID, now); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update saldo bank: " + err.Error()})
			return
		}
	}

	// ── 12. Update status penjualan ───────────────────────────────────────────
	if err := tx.Model(&penjualan).Update("status_bagi_hasil", models.BagiHasilBerhasil).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update status penjualan"})
		return
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal commit transaksi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":                  "Bagi hasil berhasil dilakukan",
		"bagi_hasil_id":            bhID,
		"gross_bank":               grossBank,
		"total_distribusi_nasabah": totalDistribusiNasabah,
		"sisa_bagi_hasil":          sisaBagiHasil,
	})

	// ── Kirim notifikasi bagi hasil (fire-and-forget) ────────────────────────────
	// Snapshot variabel yang dibutuhkan goroutine
	snapshotBankID := bankID
	snapshotNotifSvc := bhc.NotifSvc
	snapshotDB := bhc.DB

	go func() {
		// Ambil nama BSI (bank yang melakukan bagi hasil)
		var bankBSI models.BankSampah
		if err := snapshotDB.Where("bank_id = ?", snapshotBankID).First(&bankBSI).Error; err != nil {
			return
		}

		type adminUser struct {
			UserID   string
			FCMToken string
		}

		// Notif ke semua admin/petugas tiap BSU penerima
		for _, nd := range bsuNotifs {
			var bankBSU models.BankSampah
			if err := snapshotDB.Where("bank_id = ?", nd.bsuID).First(&bankBSU).Error; err != nil {
				continue
			}

			var admins []adminUser
			if err := snapshotDB.Table("admin").
				Select("users.user_id, users.fcm_token").
				Joins("JOIN users ON users.user_id = admin.user_id").
				Where("admin.bank_id = ? AND admin.status_admin = ?", nd.bsuID, models.Aktif).
				Scan(&admins).Error; err != nil {
				continue
			}

			pesanNilai := formatBagiHasilNilai(nd.nettBSU, nd.satuan)
			for _, au := range admins {
				if err := snapshotNotifSvc.NotifBagiHasilDiterima(
					context.Background(),
					au.UserID,
					au.FCMToken,
					pesanNilai,
					bankBSU.NamaBank,
					bankBSI.NamaBank,
					nd.penerimaID, // Gunakan penerimaID sebagai refID
				); err != nil {
					fmt.Printf("[Notif] Gagal kirim bagi hasil BSU ke user %s: %v\n", au.UserID, err)
				}
			}
		}

		// Notif ke tiap nasabah penerima
		for _, nd := range nasabahNotifs {
			var nasabah models.Nasabah
			if err := snapshotDB.Preload("User").Where("nasabah_id = ?", nd.nasabahID).First(&nasabah).Error; err != nil {
				continue
			}

			pesanNilai := formatBagiHasilNilai(nd.total, nd.satuan)
			if err := snapshotNotifSvc.NotifBagiHasilNasabahDiterima(
				context.Background(),
				nasabah.User.UserID,
				nasabah.User.FCMToken,
				pesanNilai,
				nd.penerimaID, // Gunakan penerimaID sebagai refID
			); err != nil {
				fmt.Printf("[Notif] Gagal kirim bagi hasil nasabah ke user %s: %v\n", nasabah.User.UserID, err)
			}
		}
	}()
}

// ─── GetDetailBagiHasil ───────────────────────────────────────────────────────
// GET /bagi-hasil/detail/:penjualan_id
func (bhc *BagiHasilController) GetDetailBagiHasil(c *gin.Context) {
	penjualanID := c.Param("penjualan_id")

	var bagiHasil models.BagiHasil
	if err := bhc.DB.Preload("Penjualan.Reward").Preload("Bank").
		Where("penjualan_id = ?", penjualanID).
		First(&bagiHasil).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data bagi hasil tidak ditemukan"})
		return
	}

	// Ambil semua penerima sekaligus
	var semuaPenerima []models.PenerimaBagiHasil
	if err := bhc.DB.Preload("Nasabah.User").Preload("Nasabah.Bank").
		Where("bagi_hasil_id = ?", bagiHasil.BagiHasilID).
		Find(&semuaPenerima).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penerima"})
		return
	}

	// ── Shared type & helper ──────────────────────────────────────────────────
	type NasabahPenerima struct {
		PenerimaID     string                  `json:"penerima_id"`
		NasabahID      string                  `json:"nasabah_id"`
		NamaNasabah    string                  `json:"nama_nasabah"`
		TotalDiterima  float64                 `json:"total_diterima"`
		SatuanDiterima models.SatuanRewardEnum `json:"satuan_diterima"`
	}

	toNasabahPenerima := func(p models.PenerimaBagiHasil) NasabahPenerima {
		nasabahID, nama := "", ""
		if p.Nasabah != nil {
			nasabahID = p.Nasabah.NasabahID
			nama = p.Nasabah.User.Nama
		}
		return NasabahPenerima{
			PenerimaID:     p.PenerimaID,
			NasabahID:      nasabahID,
			NamaNasabah:    nama,
			TotalDiterima:  p.TotalDiterima,
			SatuanDiterima: p.SatuanDiterima,
		}
	}

	// ── Cari daftar BSU di bawah bank distributor ─────────────────────────────
	bsuIDsBawahBank := make(map[string]bool)
	if bagiHasil.BankID != nil {
		var daftarBSU []models.BankSampah
		if err := bhc.DB.
			Where("parent_bank_id = ? AND jenis_bank = ?", *bagiHasil.BankID, models.BSU).
			Find(&daftarBSU).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar BSU"})
			return
		}
		for _, bsu := range daftarBSU {
			bsuIDsBawahBank[bsu.BankID] = true
		}
	}


	nasabahPerBSU := make(map[string][]NasabahPenerima)          // bankID → nasabah BSU
	var nasabahLangsung []NasabahPenerima                         // nasabah di bank distributor langsung

	for _, p := range semuaPenerima {
		if p.Nasabah != nil {
			if bsuIDsBawahBank[p.Nasabah.BankID] {
				nasabahPerBSU[p.Nasabah.BankID] = append(nasabahPerBSU[p.Nasabah.BankID], toNasabahPenerima(p))
			} else {
				nasabahLangsung = append(nasabahLangsung, toNasabahPenerima(p))
			}
		}
	}
	if nasabahLangsung == nil {
		nasabahLangsung = []NasabahPenerima{}
	}

	// Kelompokkan nasabah berdasarkan bank sampah mereka
	type BankPenerima struct {
		BankID          string            `json:"bank_id"`
		NamaBank        string            `json:"nama_bank"`
		NasabahPenerima []NasabahPenerima `json:"nasabah_penerima"`
	}

	bankMap := make(map[string]*BankPenerima)
	var bankOrder []string

	for _, p := range semuaPenerima {
		if p.Nasabah == nil {
			continue
		}
		bankID := p.Nasabah.BankID
		if !bsuIDsBawahBank[bankID] {
			continue
		}
		namaBank := p.Nasabah.Bank.NamaBank
		if namaBank == "" {
			namaBank = "Bank Sampah"
		}
		if _, ada := bankMap[bankID]; !ada {
			bankMap[bankID] = &BankPenerima{
				BankID:          bankID,
				NamaBank:        namaBank,
				NasabahPenerima: []NasabahPenerima{},
			}
			bankOrder = append(bankOrder, bankID)
		}
		bankMap[bankID].NasabahPenerima = append(bankMap[bankID].NasabahPenerima, toNasabahPenerima(p))
	}

	var penerima []BankPenerima
	for _, bID := range bankOrder {
		penerima = append(penerima, *bankMap[bID])
	}
	if penerima == nil {
		penerima = []BankPenerima{}
	}

	// ── Ambil nama petugas yang melakukan bagi hasil ──────────────────────────
	var namaPetugas string
	if bagiHasil.CreatedBy != nil {
		bhc.DB.Table("admin").
			Select("users.nama").
			Joins("JOIN users ON users.user_id = admin.user_id").
			Where("admin.admin_id = ?", *bagiHasil.CreatedBy).
			Scan(&namaPetugas)
	}

	// ── Ambil distribusi_id jika sisa sudah pernah didistribusikan ────────────
	var distribusiID *string
	var distribusiSisa models.DistribusiSisa
	if err := bhc.DB.Where("bagi_hasil_id = ?", bagiHasil.BagiHasilID).First(&distribusiSisa).Error; err == nil {
		distribusiID = &distribusiSisa.DistribusiID
	}

	response := gin.H{
		"bagi_hasil_id":            bagiHasil.BagiHasilID,
		"tanggal":                  bagiHasil.CreatedAt,
		"penjualan_id":             penjualanID,
		"nama_petugas":             namaPetugas,
		"reward":                   bagiHasil.Penjualan.Reward.NamaReward,
		"gross_bank":               bagiHasil.GrossBank,
		"total_distribusi_nasabah": bagiHasil.TotalDistribusiNasabah,
		"sisa_bagi_hasil":          bagiHasil.SisaBagiHasil,
		"satuan":                   bagiHasil.SatuanBagiHasil,
		"distribusi_id":            distribusiID,
		"nasabah_langsung":         nasabahLangsung,
	}

	if bagiHasil.Bank.JenisBank != models.BSM {
		response["penerima"] = penerima
	}

	c.JSON(http.StatusOK, response)
}

// ─── Helper: updateSaldoRekening ─────────────────────────────────────────────
func updateSaldoRekening(
	tx *gorm.DB,
	bankID *string,
	nasabahID *string,
	rewardID int,
	reward models.Reward,
	nilai float64,
	entitas models.EntitasEnum,
	adminID string,
	now time.Time,
) error {
	var saldo models.SaldoRekening

	query := tx.Where("reward_id = ? AND entitas = ?", rewardID, entitas)
	if entitas == models.EntitasBankSampah {
		query = query.Where("bank_id = ?", *bankID)
	} else {
		query = query.Where("nasabah_id = ?", *nasabahID)
	}

	if err := query.First(&saldo).Error; err != nil {
		if entitas == models.EntitasNasabah {
			return fmt.Errorf("rekening nasabah %s untuk reward_id %d tidak ditemukan", *nasabahID, rewardID)
		}
		return fmt.Errorf("rekening bank %s untuk reward_id %d tidak ditemukan", *bankID, rewardID)
	}

	nominalSebelum := saldo.NominalSaldo
	saldo.NominalSaldo += nilai
	saldo.LastUpdatedAt = now
	saldo.LastUpdatedBy = &adminID
	if err := tx.Save(&saldo).Error; err != nil {
		return err
	}

	return tx.Create(&models.RiwayatArusSaldo{
		RiwayatSaldoID: utils.GenerateID("RS"),
		RekeningID:     &saldo.RekeningID,
		NominalSebelum: nominalSebelum,
		NominalSesudah: saldo.NominalSaldo,
		CreatedAt:      now,
		CreatedBy:      &adminID,
	}).Error
}

// ─── GetListBagiHasilPerNasabah ───────────────────────────────────────────────
// GET /bagi-hasil/list-bh-nasabah/:nasabah_id
func (bhc *BagiHasilController) GetListBagiHasilPerNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	// Validasi nasabah ada
	var nasabah models.Nasabah
	if err := bhc.DB.Preload("User").
		Where("nasabah_id = ?", nasabahID).
		First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		return
	}

	// Ambil list PenerimaBagiHasil milik nasabah ini
	var penerimas []models.PenerimaBagiHasil
	if err := bhc.DB.
		Preload("BagiHasil.Penjualan.Reward").
		Where("nasabah_id = ?", nasabahID).
		Order("created_at DESC").
		Find(&penerimas).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bagi hasil"})
		return
	}

	type ListItemResp struct {
		PenerimaID     string                  `json:"penerima_id"`
		BagiHasilID    string                  `json:"bagi_hasil_id"`
		Reward         string                  `json:"reward"`
		Tanggal        string                  `json:"tanggal"`
		TotalDiterima  float64                 `json:"total_diterima"`
		SatuanDiterima models.SatuanRewardEnum `json:"satuan_diterima"`
	}

	var list []ListItemResp
	for _, p := range penerimas {
		if p.BagiHasil == nil || p.BagiHasil.Penjualan == nil {
			continue
		}
		list = append(list, ListItemResp{
			PenerimaID:     p.PenerimaID,
			BagiHasilID:    *p.BagiHasilID,
			Reward:         string(p.BagiHasil.Penjualan.Reward.NamaReward),
			Tanggal:        p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			TotalDiterima:  p.TotalDiterima,
			SatuanDiterima: p.SatuanDiterima,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"nasabah_id":         nasabah.NasabahID,
		"nama_nasabah":       nasabah.User.Nama,
		"riwayat_bagi_hasil": list,
	})
}

// ─── GetDetailBagiHasilNasabah ────────────────────────────────────────────────
// GET /bagi-hasil/detail-bh-nasabah/:penerima_id
func (bhc *BagiHasilController) GetDetailBagiHasilNasabah(c *gin.Context) {
	penerimaID := c.Param("penerima_id")

	// Ambil PenerimaBagiHasil beserta relasinya
	var penerima models.PenerimaBagiHasil
	if err := bhc.DB.
		Preload("BagiHasil.Penjualan.Reward").
		Preload("Nasabah.User").
		Where("penerima_id = ? AND nasabah_id IS NOT NULL", penerimaID).
		First(&penerima).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data bagi hasil nasabah tidak ditemukan"})
		return
	}

	if penerima.BagiHasil == nil || penerima.BagiHasil.Penjualan == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data bagi hasil tidak lengkap"})
		return
	}

	// Ambil detail item sampah
	var details []models.DetailBagiHasil
	if err := bhc.DB.Preload("KatalogSampah").
		Where("penerima_id = ?", penerimaID).
		Find(&details).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail bagi hasil"})
		return
	}

	type DetailItemResp struct {
		SampahID      string  `json:"sampah_id"`
		NamaSampah    string  `json:"nama_sampah"`
		Qty           float64 `json:"qty"`
		HargaItem     float64 `json:"harga_item"`
		SubtotalHarga float64 `json:"subtotal_harga"`
	}

	var detailItems []DetailItemResp
	for _, d := range details {
		namaSampah := ""
		if d.KatalogSampah != nil {
			namaSampah = d.KatalogSampah.NamaSampah
		}
		detailItems = append(detailItems, DetailItemResp{
			SampahID:      d.SampahID,
			NamaSampah:    namaSampah,
			Qty:           d.Qty,
			HargaItem:     d.HargaItem,
			SubtotalHarga: d.SubtotalHarga,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"penerima_id":     penerima.PenerimaID,
		"bagi_hasil_id":   *penerima.BagiHasilID,
		"nasabah_id":      penerima.Nasabah.NasabahID,
		"nama_nasabah":    penerima.Nasabah.User.Nama,
		"reward":          string(penerima.BagiHasil.Penjualan.Reward.NamaReward),
		"tanggal":         penerima.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"total_diterima":  penerima.TotalDiterima,
		"satuan_diterima": penerima.SatuanDiterima,
		"detail_item":     detailItems,
	})
}

// ─── GetListBagiHasilPerBsu ───────────────────────────────────────────────────
// GET /bagi-hasil/list-bh-bsu/:bsu_id
// Endpoint ini sekarang mengarahkan ke distribusi_sisa.
// Bagi hasil BSU tidak lagi disimpan di penerima_bagi_hasil.
func (bhc *BagiHasilController) GetListBagiHasilPerBsu(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{
		"error": "Endpoint ini sudah tidak digunakan. Gunakan GET /distribusi-sisa untuk melihat bagi hasil BSU.",
	})
}

// ─── GetDetailBagiHasilBSU ────────────────────────────────────────────────────
// GET /bagi-hasil/detail-bh-bsu/:penerima_id
// Endpoint ini sekarang mengarahkan ke distribusi_sisa.
func (bhc *BagiHasilController) GetDetailBagiHasilBSU(c *gin.Context) {
	c.JSON(http.StatusGone, gin.H{
		"error": "Endpoint ini sudah tidak digunakan. Gunakan GET /distribusi-sisa/detail/:distribusi_id untuk melihat detail bagi hasil BSU.",
	})
}

func (bhc *BagiHasilController) GetListBagiHasilBankPusat(c *gin.Context) {
	bankID := c.Param("bank_id")

	// Validasi bank ada dan bukan BSU
	var bank models.BankSampah
	if err := bhc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank == models.BSU {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank bukan bertipe Bank Pusat"})
		return
	}

	var filter struct {
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
	}
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format query param tidak valid"})
		return
	}

	const dateLayout = "2006-01-02"

	var startTime, endTime time.Time
	if filter.StartDate != "" {
		t, err := time.ParseInLocation(dateLayout, filter.StartDate, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format start_date tidak valid, gunakan YYYY-MM-DD"})
			return
		}
		startTime = t
	}
	if filter.EndDate != "" {
		t, err := time.ParseInLocation(dateLayout, filter.EndDate, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format end_date tidak valid, gunakan YYYY-MM-DD"})
			return
		}
		// End of day agar data di tanggal EndDate ikut tercakup
		endTime = t.Add(24*time.Hour - time.Second)
	}
	if !startTime.IsZero() && !endTime.IsZero() && startTime.After(endTime) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_date tidak boleh lebih besar dari end_date"})
		return
	}

	query := bhc.DB.
		Preload("Penjualan.Reward").
		Where("bank_id = ?", bankID)

	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	var bagiHasils []models.BagiHasil
	if err := query.Order("created_at DESC").Find(&bagiHasils).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bagi hasil"})
		return
	}

	type Response struct {
		BagiHasilID      string `json:"bagi_hasil_id"`
		RewardID         int    `json:"reward_id"`
		NamaReward       string `json:"nama_reward"`
		TanggalBagiHasil string `json:"tanggal_bagi_hasil"`
	}

	var list []Response
	for _, bh := range bagiHasils {
		if bh.Penjualan == nil {
			continue
		}
		list = append(list, Response{
			BagiHasilID:      bh.BagiHasilID,
			RewardID:         bh.Penjualan.RewardID,
			NamaReward:       string(bh.Penjualan.Reward.NamaReward),
			TanggalBagiHasil: bh.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"bank_id":            bank.BankID,
		"nama_bank":          bank.NamaBank,
		"jenis_bank":         bank.JenisBank,
		"riwayat_bagi_hasil": list,
	})
}

func (bhc *BagiHasilController) GetDetailBagiHasilBankPusat(c *gin.Context) {
	bagiHasilID := c.Param("bagi_hasil_id")

	// Ambil BagiHasil beserta relasi Penjualan & Reward
	var bagiHasil models.BagiHasil
	if err := bhc.DB.
		Preload("Penjualan.Reward").
		Preload("Bank").
		Where("bagi_hasil_id = ?", bagiHasilID).
		First(&bagiHasil).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data bagi hasil tidak ditemukan"})
		return
	}

	if bagiHasil.Bank == nil || bagiHasil.Penjualan == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Data bagi hasil tidak lengkap"})
		return
	}

	if bagiHasil.Bank.JenisBank == models.BSU {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank bukan bertipe Bank Pusat"})
		return
	}

	// Shared response types
	type NasabahPenerima struct {
		PenerimaID     string                  `json:"penerima_id"`
		NasabahID      string                  `json:"nasabah_id"`
		NamaNasabah    string                  `json:"nama_nasabah"`
		TotalDiterima  float64                 `json:"total_diterima"`
		SatuanDiterima models.SatuanRewardEnum `json:"satuan_diterima"`
	}

	// Helper: konversi PenerimaBagiHasil nasabah → NasabahPenerima
	toNasabahPenerima := func(p models.PenerimaBagiHasil) NasabahPenerima {
		nama := ""
		nasabahID := ""
		if p.Nasabah != nil {
			nasabahID = p.Nasabah.NasabahID
			nama = p.Nasabah.User.Nama
		}
		return NasabahPenerima{
			PenerimaID:     p.PenerimaID,
			NasabahID:      nasabahID,
			NamaNasabah:    nama,
			TotalDiterima:  p.TotalDiterima,
			SatuanDiterima: p.SatuanDiterima,
		}
	}

	penjualan := bagiHasil.Penjualan
	bank := bagiHasil.Bank

	// Ambil distribusi_id jika sisa sudah pernah didistribusikan
	var distribusiID *string
	var distribusiSisa models.DistribusiSisa
	if err := bhc.DB.Where("bagi_hasil_id = ?", bagiHasilID).First(&distribusiSisa).Error; err == nil {
		distribusiID = &distribusiSisa.DistribusiID
	}

	// ── BSM ───────────────────────────────────────────────────────────────────
	if bank.JenisBank == models.BSM {
		// Ambil semua penerima nasabah dari bagi hasil ini
		var penerimaNasabah []models.PenerimaBagiHasil
		if err := bhc.DB.
			Preload("Nasabah.User").Preload("Nasabah.Bank").
			Where("bagi_hasil_id = ? AND nasabah_id IS NOT NULL", bagiHasilID).
			Find(&penerimaNasabah).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data nasabah penerima"})
			return
		}

		var nasabahList []NasabahPenerima
		for _, p := range penerimaNasabah {
			nasabahList = append(nasabahList, toNasabahPenerima(p))
		}

		// Kelompokkan nasabah berdasarkan bank sampah mereka
		type BankPenerima struct {
			BankID          string            `json:"bank_id"`
			NamaBank        string            `json:"nama_bank"`
			NasabahPenerima []NasabahPenerima `json:"nasabah_penerima"`
		}

		bankMap := make(map[string]*BankPenerima)
		var bankOrder []string

		for _, p := range penerimaNasabah {
			if p.Nasabah != nil {
				bankID := p.Nasabah.BankID
				namaBank := p.Nasabah.Bank.NamaBank
				if namaBank == "" {
					namaBank = "Bank Sampah"
				}
				if _, ada := bankMap[bankID]; !ada {
					bankMap[bankID] = &BankPenerima{
						BankID:          bankID,
						NamaBank:        namaBank,
						NasabahPenerima: []NasabahPenerima{},
					}
					bankOrder = append(bankOrder, bankID)
				}
				bankMap[bankID].NasabahPenerima = append(bankMap[bankID].NasabahPenerima, toNasabahPenerima(p))
			}
		}

		var penerima []BankPenerima
		for _, bID := range bankOrder {
			penerima = append(penerima, *bankMap[bID])
		}
		if penerima == nil {
			penerima = []BankPenerima{}
		}

		c.JSON(http.StatusOK, gin.H{
			"bagi_hasil_id":            bagiHasil.BagiHasilID,
			"penjualan_id":             *bagiHasil.PenjualanID,
			"reward_id":                penjualan.RewardID,
			"nama_reward":              string(penjualan.Reward.NamaReward),
			"tanggal":                  bagiHasil.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"gross_bsm":                bagiHasil.GrossBank,
			"total_distribusi_nasabah": bagiHasil.TotalDistribusiNasabah,
			"sisa_bagi_hasil":          bagiHasil.SisaBagiHasil,
			"distribusi_id":            distribusiID,
			"penerima":                 penerima,
			"nasabah_bsm":              nasabahList,
		})
		return
	}

	// ── BSI ───────────────────────────────────────────────────────────────────

	// Ambil semua penerima (BSU + nasabah) sekaligus
	var semuaPenerima []models.PenerimaBagiHasil
	if err := bhc.DB.
		Preload("Nasabah.User").
		Preload("Nasabah.Bank").
		Where("bagi_hasil_id = ?", bagiHasilID).
		Find(&semuaPenerima).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penerima"})
		return
	}

	// Kumpulkan semua BSU di bawah BSI, kelompokkan nasabah per BSU
	bsuIDsBawahBSI := make(map[string]bool)
	var daftarBSU []models.BankSampah
	if err := bhc.DB.
		Where("parent_bank_id = ? AND jenis_bank = ?", *bagiHasil.BankID, models.BSU).
		Find(&daftarBSU).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar BSU"})
		return
	}
	for _, bsu := range daftarBSU {
		bsuIDsBawahBSI[bsu.BankID] = true
	}

	var nasabahLangsung []models.PenerimaBagiHasil // nasabah langsung di BSI

	for _, p := range semuaPenerima {
		// Nasabah langsung di BSI: bank_id nasabah bukan salah satu BSU di bawah BSI
		if p.Nasabah != nil && !bsuIDsBawahBSI[p.Nasabah.BankID] {
			nasabahLangsung = append(nasabahLangsung, p)
		}
	}

	// Kelompokkan nasabah penerima berdasarkan bank_id nasabahnya
	nasabahPerBSU := make(map[string][]NasabahPenerima) // bsuID → []NasabahPenerima
	for _, p := range semuaPenerima {
		if p.Nasabah == nil {
			continue
		}
		bankNasabahID := p.Nasabah.BankID
		if bsuIDsBawahBSI[bankNasabahID] {
			nasabahPerBSU[bankNasabahID] = append(nasabahPerBSU[bankNasabahID], toNasabahPenerima(p))
		}
	}

	var nasabahBSIList []NasabahPenerima
	for _, p := range nasabahLangsung {
		nasabahBSIList = append(nasabahBSIList, toNasabahPenerima(p))
	}

	// Kelompokkan nasabah berdasarkan bank sampah mereka
	type BankPenerima struct {
		BankID          string            `json:"bank_id"`
		NamaBank        string            `json:"nama_bank"`
		NasabahPenerima []NasabahPenerima `json:"nasabah_penerima"`
	}

	bankMap := make(map[string]*BankPenerima)
	var bankOrder []string

	for _, p := range semuaPenerima {
		if p.Nasabah == nil {
			continue
		}
		bankID := p.Nasabah.BankID
		if !bsuIDsBawahBSI[bankID] {
			continue
		}
		namaBank := p.Nasabah.Bank.NamaBank
		if namaBank == "" {
			namaBank = "Bank Sampah"
		}
		if _, ada := bankMap[bankID]; !ada {
			bankMap[bankID] = &BankPenerima{
				BankID:          bankID,
				NamaBank:        namaBank,
				NasabahPenerima: []NasabahPenerima{},
			}
			bankOrder = append(bankOrder, bankID)
		}
		bankMap[bankID].NasabahPenerima = append(bankMap[bankID].NasabahPenerima, toNasabahPenerima(p))
	}

	var penerima []BankPenerima
	for _, bID := range bankOrder {
		penerima = append(penerima, *bankMap[bID])
	}
	if penerima == nil {
		penerima = []BankPenerima{}
	}

	c.JSON(http.StatusOK, gin.H{
		"bagi_hasil_id":            bagiHasil.BagiHasilID,
		"penjualan_id":             *bagiHasil.PenjualanID,
		"reward_id":                penjualan.RewardID,
		"nama_reward":              string(penjualan.Reward.NamaReward),
		"tanggal":                  bagiHasil.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"gross_bsi":                bagiHasil.GrossBank,
		"total_distribusi_nasabah": bagiHasil.TotalDistribusiNasabah,
		"sisa_bagi_hasil":          bagiHasil.SisaBagiHasil,
		"distribusi_id":            distribusiID,
		"penerima":                 penerima,
		"nasabah_bsi":              nasabahBSIList,
	})
}

// formatBagiHasilNilai memformat nilai bagi hasil sesuai satuannya.
// - Uang  : Rp1.500.000
// - Emas  : Rp1.500.000 (bank tidak punya saldo emas, rupiah saja)
// - Sembako: 1500 poin
func formatBagiHasilNilai(nilai float64, satuan models.SatuanRewardEnum) string {
	switch satuan {
	case models.SatuanRewardEnumPoin:
		return fmt.Sprintf("%.0f poin", nilai)
	default:
		// Rp di depan angka (untuk satuan Rp, gram/emas → tetap tampilkan sebagai Rp)
		return fmt.Sprintf("Rp%s", formatAngka(int64(nilai)))
	}
}

func formatAngka(n int64) string {
	s := fmt.Sprintf("%d", n)
	ln := len(s)
	if ln <= 3 {
		return s
	}
	result := ""
	for i, c := range s {
		if i > 0 && (ln-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return result
}
