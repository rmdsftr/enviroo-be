package controllers

import (
	"log"
	"math"
	"net/http"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DistribusiSisaController struct {
	DB       *gorm.DB
	NotifSvc services.NotifikasiService
}

func NewDistribusiSisa(db *gorm.DB, notifSvc services.NotifikasiService) *DistribusiSisaController {
	return &DistribusiSisaController{DB: db, NotifSvc: notifSvc}
}

// ─── GetKonfigurasi ───────────────────────────────────────────────────────────
// GET /distribusi-sisa/konfigurasi/:bank_id
func (dsc *DistribusiSisaController) GetKonfigurasi(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := dsc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank != models.BSI {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya bank sampah bertipe BSI yang memiliki konfigurasi ini"})
		return
	}

	var setting models.SettingSisaBagiHasil
	if err := dsc.DB.Where("bank_id = ?", bankID).First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Konfigurasi belum diatur untuk bank ini"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil konfigurasi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"setting_id":      setting.SettingID,
		"bank_id":         setting.BankID,
		"nama_bank":       bank.NamaBank,
		"porsi_bsu":       setting.PorsiBSU,
		"porsi_bsi":       setting.PorsiBSI,
		"porsi_transport": setting.PorsiTransport,
		"total_porsi":     setting.PorsiBSU + setting.PorsiBSI + setting.PorsiTransport,
	})
}

// ─── AddKonfigurasi ───────────────────────────────────────────────────────────
// POST /distribusi-sisa/konfigurasi/:bank_id
func (dsc *DistribusiSisaController) AddKonfigurasi(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := dsc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank != models.BSI {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya bank sampah bertipe BSI yang dapat mengatur konfigurasi ini"})
		return
	}

	var req struct {
		PorsiBSU       float64 `json:"porsi_bsu" binding:"required,gt=0"`
		PorsiBSI       float64 `json:"porsi_bsi" binding:"required,gt=0"`
		PorsiTransport float64 `json:"porsi_transport" binding:"required,gt=0"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	total := req.PorsiBSU + req.PorsiBSI + req.PorsiTransport
	if total < 99.99 || total > 100.01 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Total porsi harus berjumlah 100%",
			"total_porsi": total,
		})
		return
	}

	var existing models.SettingSisaBagiHasil
	if err := dsc.DB.Where("bank_id = ?", bankID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":      "Konfigurasi untuk bank ini sudah ada, gunakan PATCH untuk mengubahnya",
			"setting_id": existing.SettingID,
		})
		return
	}

	setting := models.SettingSisaBagiHasil{
		BankID:         bankID,
		PorsiBSU:       req.PorsiBSU,
		PorsiBSI:       req.PorsiBSI,
		PorsiTransport: req.PorsiTransport,
	}
	if err := dsc.DB.Create(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan konfigurasi"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":         "Konfigurasi berhasil dibuat",
		"setting_id":      setting.SettingID,
		"bank_id":         setting.BankID,
		"nama_bank":       bank.NamaBank,
		"porsi_bsu":       setting.PorsiBSU,
		"porsi_bsi":       setting.PorsiBSI,
		"porsi_transport": setting.PorsiTransport,
		"total_porsi":     total,
	})
}

// ─── UpdateKonfigurasi ────────────────────────────────────────────────────────
// PATCH /distribusi-sisa/konfigurasi/:bank_id
func (dsc *DistribusiSisaController) UpdateKonfigurasi(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := dsc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}
	if bank.JenisBank != models.BSI {
		c.JSON(http.StatusForbidden, gin.H{"error": "Hanya bank sampah bertipe BSI yang dapat mengatur konfigurasi ini"})
		return
	}

	var req struct {
		PorsiBSU       *float64 `json:"porsi_bsu"`
		PorsiBSI       *float64 `json:"porsi_bsi"`
		PorsiTransport *float64 `json:"porsi_transport"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.PorsiBSU == nil && req.PorsiBSI == nil && req.PorsiTransport == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Minimal satu field porsi harus diisi"})
		return
	}

	var setting models.SettingSisaBagiHasil
	if err := dsc.DB.Where("bank_id = ?", bankID).First(&setting).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Konfigurasi belum diatur, gunakan POST untuk membuat konfigurasi baru"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil konfigurasi"})
		return
	}

	if req.PorsiBSU != nil {
		setting.PorsiBSU = *req.PorsiBSU
	}
	if req.PorsiBSI != nil {
		setting.PorsiBSI = *req.PorsiBSI
	}
	if req.PorsiTransport != nil {
		setting.PorsiTransport = *req.PorsiTransport
	}

	total := setting.PorsiBSU + setting.PorsiBSI + setting.PorsiTransport
	if total < 99.99 || total > 100.01 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":       "Total porsi harus berjumlah 100%",
			"total_porsi": total,
		})
		return
	}

	if err := dsc.DB.Save(&setting).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan konfigurasi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Konfigurasi berhasil diperbarui",
		"setting_id":      setting.SettingID,
		"bank_id":         setting.BankID,
		"nama_bank":       bank.NamaBank,
		"porsi_bsu":       setting.PorsiBSU,
		"porsi_bsi":       setting.PorsiBSI,
		"porsi_transport": setting.PorsiTransport,
		"total_porsi":     total,
	})
}

// ─── PreviewDistribusiSisa ───────────────────────────────────────────────────
// GET /distribusi-sisa/preview/:bagi_hasil_id
func (dsc *DistribusiSisaController) PreviewDistribusiSisa(c *gin.Context) {
	bagiHasilID := c.Param("bagi_hasil_id")

	// ── 1. Ambil BagiHasil ────────────────────────────────────────────────────
	var bagiHasil models.BagiHasil
	if err := dsc.DB.
		Preload("Bank").
		Where("bagi_hasil_id = ?", bagiHasilID).
		First(&bagiHasil).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data bagi hasil tidak ditemukan"})
		return
	}

	if bagiHasil.Bank == nil || bagiHasil.Bank.JenisBank != models.BSI {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Distribusi sisa hanya berlaku untuk bagi hasil BSI"})
		return
	}
	bsiID := *bagiHasil.BankID

	// ── 2. Cek double distribution ────────────────────────────────────────────
	var existingDistribusi models.DistribusiSisa
	if err := dsc.DB.Where("bagi_hasil_id = ?", bagiHasilID).First(&existingDistribusi).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Sisa bagi hasil ini sudah pernah didistribusikan",
			"distribusi_id": existingDistribusi.DistribusiID,
		})
		return
	}

	totalSisa := bagiHasil.SisaBagiHasil
	if totalSisa <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada sisa bagi hasil yang perlu didistribusikan"})
		return
	}

	// ── 3. Ambil konfigurasi porsi milik BSI ini ──────────────────────────────
	var setting models.SettingSisaBagiHasil
	if err := dsc.DB.Where("bank_id = ?", bsiID).First(&setting).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Konfigurasi porsi distribusi sisa belum diatur untuk bank ini"})
		return
	}

	// ── 4. Hitung kontribusi proporsional tiap BSU ────────────────────────────
	var penerimaNasabah []models.PenerimaBagiHasil
	if err := dsc.DB.
		Preload("Nasabah").
		Where("bagi_hasil_id = ? AND nasabah_id IS NOT NULL", bagiHasilID).
		Find(&penerimaNasabah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penerima nasabah"})
		return
	}

	// Ambil daftar BSU di bawah BSI ini
	var daftarBSU []models.BankSampah
	if err := dsc.DB.
		Where("parent_bank_id = ? AND jenis_bank = ?", bsiID, models.BSU).
		Find(&daftarBSU).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar BSU"})
		return
	}
	bsuMap := make(map[string]models.BankSampah)
	for _, bsu := range daftarBSU {
		bsuMap[bsu.BankID] = bsu
	}

	kontribusiPerBSU := make(map[string]float64) // bankID → total diterima nasabah BSU tersebut
	var totalNilaiNasabahBSU float64
	for _, p := range penerimaNasabah {
		if p.Nasabah != nil {
			if _, ok := bsuMap[p.Nasabah.BankID]; ok {
				kontribusiPerBSU[p.Nasabah.BankID] += p.TotalDiterima
				totalNilaiNasabahBSU += p.TotalDiterima
			}
		}
	}

	type BSUPreview struct {
		BankID           string  `json:"bank_id"`
		NamaBank         string  `json:"nama_bank"`
		TotalKontribusi  float64 `json:"total_kontribusi_nasabah"`
		PersenKontribusi float64 `json:"kontribusi_persen"`
	}

	var bsuTerlibat []BSUPreview
	for bsuID, kontribusi := range kontribusiPerBSU {
		if kontribusi > 0 && totalNilaiNasabahBSU > 0 {
			persen := (kontribusi / totalNilaiNasabahBSU) * 100
			bsuTerlibat = append(bsuTerlibat, BSUPreview{
				BankID:           bsuID,
				NamaBank:         bsuMap[bsuID].NamaBank,
				TotalKontribusi:  kontribusi,
				PersenKontribusi: persen,
			})
		}
	}

	if bsuTerlibat == nil {
		bsuTerlibat = []BSUPreview{}
	}

	c.JSON(http.StatusOK, gin.H{
		"bagi_hasil_id":   bagiHasilID,
		"total_sisa":      totalSisa,
		"satuan":          bagiHasil.SatuanBagiHasil,
		"porsi_bsi":       setting.PorsiBSI,
		"porsi_bsu":       setting.PorsiBSU,
		"porsi_transport": setting.PorsiTransport,
		"bsu_terlibat":    bsuTerlibat,
	})
}

// ─── SubmitDistribusiSisa ─────────────────────────────────────────────────────
// POST /distribusi-sisa/submit/:bagi_hasil_id
func (dsc *DistribusiSisaController) SubmitDistribusiSisa(c *gin.Context) {
	bagiHasilID := c.Param("bagi_hasil_id")

	type PengirimanBSU struct {
		BankID      string           `json:"bank_id" binding:"required"`
		DiantarOleh models.AntarEnum `json:"diantar_oleh" binding:"required"`
	}

	var req struct {
		AdminID       string          `json:"admin_id" binding:"required"`
		PengirimanBSU []PengirimanBSU `json:"pengiriman_bsu" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Map untuk lookup cara pengiriman per BSU
	pengirimanMap := make(map[string]models.AntarEnum)
	for _, p := range req.PengirimanBSU {
		if p.DiantarOleh != models.AntarBSI && p.DiantarOleh != models.AntarBSU {
			c.JSON(http.StatusBadRequest, gin.H{"error": "diantar_oleh harus 'bsi' atau 'bsu'"})
			return
		}
		pengirimanMap[p.BankID] = p.DiantarOleh
	}

	// ── 1. Ambil BagiHasil ────────────────────────────────────────────────────
	var bagiHasil models.BagiHasil
	if err := dsc.DB.
		Preload("Bank").
		Where("bagi_hasil_id = ?", bagiHasilID).
		First(&bagiHasil).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data bagi hasil tidak ditemukan"})
		return
	}

	if bagiHasil.Bank == nil || bagiHasil.Bank.JenisBank != models.BSI {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Distribusi sisa hanya berlaku untuk bagi hasil BSI"})
		return
	}
	bsiID := *bagiHasil.BankID

	// ── 2. Cek double distribution ────────────────────────────────────────────
	var existingDistribusi models.DistribusiSisa
	if err := dsc.DB.Where("bagi_hasil_id = ?", bagiHasilID).First(&existingDistribusi).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":         "Sisa bagi hasil ini sudah pernah didistribusikan",
			"distribusi_id": existingDistribusi.DistribusiID,
		})
		return
	}

	totalSisa := bagiHasil.SisaBagiHasil
	if totalSisa <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada sisa bagi hasil yang perlu didistribusikan"})
		return
	}

	// ── 3. Ambil konfigurasi porsi milik BSI ini ──────────────────────────────
	var setting models.SettingSisaBagiHasil
	if err := dsc.DB.Where("bank_id = ?", bsiID).First(&setting).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Konfigurasi porsi distribusi sisa belum diatur untuk bank ini"})
		return
	}

	// Porsi tersimpan dalam persen (60 → 60%), konversi ke desimal
	porsiBSU := setting.PorsiBSU / 100
	porsiBSI := setting.PorsiBSI / 100
	porsiTransport := setting.PorsiTransport / 100

	// ── 3b. Resolve reward berdasarkan satuan distribusi ─────────────────────
	// Bank sampah hanya punya saldo Uang (Rp) dan Sembako (poin)
	var namaRewardTarget models.RewardEnum
	if bagiHasil.SatuanBagiHasil == models.SatuanRewardEnumRp {
		namaRewardTarget = models.RewardEnumUang
	} else {
		namaRewardTarget = models.RewardEnumSembako
	}
	var rewardDistribusi models.Reward
	if err := dsc.DB.Where("nama_reward = ?", namaRewardTarget).First(&rewardDistribusi).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menentukan jenis reward untuk distribusi"})
		return
	}

	// ── 4. Hitung kontribusi proporsional tiap BSU ────────────────────────────
	var penerimaNasabah []models.PenerimaBagiHasil
	if err := dsc.DB.
		Preload("Nasabah").
		Where("bagi_hasil_id = ? AND nasabah_id IS NOT NULL", bagiHasilID).
		Find(&penerimaNasabah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penerima nasabah"})
		return
	}

	// Ambil daftar BSU di bawah BSI ini
	var daftarBSU []models.BankSampah
	if err := dsc.DB.
		Where("parent_bank_id = ? AND jenis_bank = ?", bsiID, models.BSU).
		Find(&daftarBSU).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil daftar BSU"})
		return
	}
	bsuIDSet := make(map[string]bool)
	for _, bsu := range daftarBSU {
		bsuIDSet[bsu.BankID] = true
	}

	kontribusiPerBSU := make(map[string]float64) // bankID → total diterima nasabah BSU tersebut
	var totalNilaiNasabahBSU float64
	for _, p := range penerimaNasabah {
		if p.Nasabah != nil && bsuIDSet[p.Nasabah.BankID] {
			kontribusiPerBSU[p.Nasabah.BankID] += p.TotalDiterima
			totalNilaiNasabahBSU += p.TotalDiterima
		}
	}

	// Validasi: pastikan semua BSU yang berkontribusi ada di request pengiriman_bsu
	for bsuID, kontribusi := range kontribusiPerBSU {
		if kontribusi > 0 {
			if _, ada := pengirimanMap[bsuID]; !ada {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Status pengiriman untuk BSU " + bsuID + " belum ditentukan dalam request"})
				return
			}
		}
	}

	// ── 5. Mulai transaksi DB ─────────────────────────────────────────────────
	tx := dsc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memulai transaksi"})
		return
	}
	now := time.Now()

	// ── 6. Buat record distribusi_sisa (induk) ────────────────────────────────
	distribusiID := utils.GenerateID("DS")
	distribusi := models.DistribusiSisa{
		DistribusiID: distribusiID,
		BagiHasilID:  bagiHasilID,
		TotalSisa:    totalSisa,
		SatuanTotal:  string(bagiHasil.SatuanBagiHasil),
		CreatedAt:    now,
		CreatedBy:    req.AdminID,
	}
	if err := tx.Create(&distribusi).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data distribusi"})
		return
	}

	// ── 7. Distribusi dan perhitungan ke tiap BSU ─────────────────────────────

	// Ambil sampah_id yang terlibat di penjualan ini agar perhitungan_sisa hanya
	// menyentuh tabungan yang relevan (tidak ikut menghabiskan tabungan dari bagi hasil lain)
	var detailPenjualan []models.DetailPenjualan
	if bagiHasil.PenjualanID == nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Bagi hasil tidak memiliki penjualan terkait"})
		return
	}
	if err := tx.Where("penjualan_id = ?", *bagiHasil.PenjualanID).Find(&detailPenjualan).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail penjualan"})
		return
	}
	sampahIDsPenjualan := make([]string, 0, len(detailPenjualan))
	for _, d := range detailPenjualan {
		sampahIDsPenjualan = append(sampahIDsPenjualan, d.SampahID)
	}

	type ringkasanBSU struct {
		BankID         string           `json:"bank_id"`
		NamaBank       string           `json:"nama_bank"`
		DiantarOleh    models.AntarEnum `json:"diantar_oleh"`
		Nominal        float64          `json:"nominal"`
		PenerimaSisaID string           `json:"-"`
	}
	var ringkasanBSUList []ringkasanBSU
	var totalPorsiBSI float64
	var totalTransportBSI float64

	for _, bsu := range daftarBSU {
		kontribusi, ada := kontribusiPerBSU[bsu.BankID]
		if !ada || kontribusi == 0 || totalNilaiNasabahBSU == 0 {
			continue // Skip BSU yang tidak berkontribusi
		}

		persentaseKontribusi := kontribusi / totalNilaiNasabahBSU
		diantarOleh := pengirimanMap[bsu.BankID]

		// Hitung porsi
		bagianPokokBSU := totalSisa * porsiBSU * persentaseKontribusi
		bagianPokokBSI := totalSisa * porsiBSI * persentaseKontribusi
		bagianTransport := totalSisa * porsiTransport * persentaseKontribusi

		totalPorsiBSI += bagianPokokBSI

		var nominalBSU, transportasiBSU float64
		if diantarOleh == models.AntarBSU {
			// BSU antar sendiri → BSU dapat transport
			transportasiBSU = bagianTransport
			nominalBSU = bagianPokokBSU + bagianTransport
		} else {
			// BSI jemput → BSI dapat transport
			totalTransportBSI += bagianTransport
			nominalBSU = bagianPokokBSU
		}

		// Buat penerima_distribusi_sisa untuk BSU
		penerimaSisaID := utils.GenerateID("PDS")
		penerimaSisa := models.PenerimaDistribusiSisa{
			PenerimaSisaID:  penerimaSisaID,
			DistribusiID:    distribusiID,
			BankID:          bsu.BankID,
			JenisPenerimaan: models.JenisTerimaBagiHasilBSU,
			NominalDiterima: nominalBSU,
			Porsi:           bagianPokokBSU,
			Transportasi:    transportasiBSU,
			SatuanNominal:   string(bagiHasil.SatuanBagiHasil),
			DiantarOleh:     &diantarOleh,
		}
		if err := tx.Create(&penerimaSisa).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan penerima BSU: " + bsu.BankID})
			return
		}

		// Update saldo rekening BSU
		bsuIDCopy := bsu.BankID
		if err := updateSaldoRekening(tx, &bsuIDCopy, nil,
			rewardDistribusi.RewardID, rewardDistribusi,
			nominalBSU, models.EntitasBankSampah,
			req.AdminID, now); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update saldo BSU " + bsu.BankID + ": " + err.Error()})
			return
		}

		// ── 7a. Perhitungan sisa: audit lineage tabungan_sampah BSU ──────────
		// Filter ke sampah yang terlibat di penjualan ini saja agar distribusi
		// berbeda reward tidak saling menghabiskan tabungan satu sama lain
		var tabunganBSU []models.TabunganSampah
		if err := tx.
			Where("bank_id = ? AND entitas = ? AND sisa_qty > 0 AND sampah_id IN ?",
				bsu.BankID, models.EntitasBankSampah, sampahIDsPenjualan).
			Find(&tabunganBSU).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil tabungan BSU " + bsu.BankID})
			return
		}

		var totalSisaQtyBSU float64
		for _, t := range tabunganBSU {
			totalSisaQtyBSU += t.SisaQty
		}

		if totalSisaQtyBSU > 0 {
			sisaNominal := nominalBSU
			for i, t := range tabunganBSU {
				if sisaNominal <= 0 {
					break
				}
				var qtyDipakai float64
				if i == len(tabunganBSU)-1 {
					qtyDipakai = math.Min(sisaNominal, t.SisaQty)
				} else {
					qtyDipakai = (t.SisaQty / totalSisaQtyBSU) * nominalBSU
					if qtyDipakai > t.SisaQty {
						qtyDipakai = t.SisaQty
					}
				}
				sisaNominal -= qtyDipakai

				perhitungan := models.PerhitunganSisa{
					PenerimaSisaID: penerimaSisaID,
					TabunganID:     t.TabunganID,
					QtyDipakai:     qtyDipakai,
				}
				if err := tx.Create(&perhitungan).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan perhitungan sisa"})
					return
				}
				if err := tx.Model(&t).Update("sisa_qty", t.SisaQty-qtyDipakai).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update sisa_qty tabungan"})
					return
				}
			}
		}

		ringkasanBSUList = append(ringkasanBSUList, ringkasanBSU{
			BankID:         bsu.BankID,
			NamaBank:       bsu.NamaBank,
			DiantarOleh:    diantarOleh,
			Nominal:        nominalBSU,
			PenerimaSisaID: penerimaSisaID,
		})
	}

	// ── 8. Distribusi ke BSI ──────────────────────────────────────────────────
	totalNominalBSI := totalPorsiBSI + totalTransportBSI
	penerimaSisaBSIID := utils.GenerateID("PDS")
	penerimaSisaBSI := models.PenerimaDistribusiSisa{
		PenerimaSisaID:  penerimaSisaBSIID,
		DistribusiID:    distribusiID,
		BankID:          bsiID,
		JenisPenerimaan: models.JenisTerimaBagiHasilBSI,
		NominalDiterima: totalNominalBSI,
		Porsi:           totalPorsiBSI,
		Transportasi:    totalTransportBSI,
		SatuanNominal:   string(bagiHasil.SatuanBagiHasil),
	}
	if err := tx.Create(&penerimaSisaBSI).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan penerima BSI"})
		return
	}

	if err := updateSaldoRekening(tx, &bsiID, nil,
		rewardDistribusi.RewardID, rewardDistribusi,
		totalNominalBSI, models.EntitasBankSampah,
		req.AdminID, now); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal update saldo BSI: " + err.Error()})
		return
	}

	// ── 9. Commit ────────────────────────────────────────────────────────────
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal commit transaksi"})
		return
	}

	if ringkasanBSUList == nil {
		ringkasanBSUList = []ringkasanBSU{}
	}

	// ── 10. Kirim notifikasi ke admin tiap BSU penerima ───────────────────────
	namaBSI := bagiHasil.Bank.NamaBank
	pesanNilai := formatBagiHasilNilai(0, bagiHasil.SatuanBagiHasil) // format helper
	ctx := c.Request.Context()
	for _, item := range ringkasanBSUList {
		pesanNilai = formatBagiHasilNilai(item.Nominal, bagiHasil.SatuanBagiHasil)

		var adminsBSU []models.Admin
		if err := dsc.DB.Preload("User").Where("bank_id = ?", item.BankID).Find(&adminsBSU).Error; err != nil {
			log.Printf("[DistribusiSisa] gagal ambil admin BSU %s: %v", item.BankID, err)
			continue
		}
		for _, adm := range adminsBSU {
			if err := dsc.NotifSvc.NotifBagiHasilDiterima(ctx,
				adm.UserID, adm.User.FCMToken,
				pesanNilai, item.NamaBank, namaBSI,
				item.PenerimaSisaID,
			); err != nil {
				log.Printf("[DistribusiSisa] notif gagal untuk admin %s: %v", adm.AdminID, err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Distribusi sisa bagi hasil berhasil",
		"distribusi_id": distribusiID,
		"bagi_hasil_id": bagiHasilID,
		"total_sisa":    totalSisa,
		"satuan":        bagiHasil.SatuanBagiHasil,
		"nominal_bsi":   totalNominalBSI,
		"penerima_bsu":  ringkasanBSUList,
	})
}

// ─── GetDetailDistribusiSisa ──────────────────────────────────────────────────
// GET /distribusi-sisa/detail/:distribusi_id
func (dsc *DistribusiSisaController) GetDetailDistribusiSisa(c *gin.Context) {
	distribusiID := c.Param("distribusi_id")

	// ── 1. Ambil header distribusi ────────────────────────────────────────────
	var distribusi models.DistribusiSisa
	if err := dsc.DB.
		Preload("BagiHasil.Bank").
		Where("distribusi_id = ?", distribusiID).
		First(&distribusi).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data distribusi tidak ditemukan"})
		return
	}

	// ── 2. Ambil nama admin pembuat ──────────────────────────────────────────
	var namaAdmin string
	var admin models.Admin
	if err := dsc.DB.
		Preload("User").
		Where("admin_id = ?", distribusi.CreatedBy).
		First(&admin).Error; err == nil {
		namaAdmin = admin.User.Nama
	}

	// ── 3. Ambil penerima BSI ─────────────────────────────────────────────────
	var penerimaBSI models.PenerimaDistribusiSisa
	if err := dsc.DB.
		Preload("BankSampah").
		Where("distribusi_id = ? AND jenis_penerimaan = ?", distribusiID, models.JenisTerimaBagiHasilBSI).
		First(&penerimaBSI).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penerima BSI"})
		return
	}

	// ── 4. Ambil semua BSU penerima ───────────────────────────────────────────
	var penerimaBSU []models.PenerimaDistribusiSisa
	if err := dsc.DB.
		Preload("BankSampah").
		Where("distribusi_id = ? AND jenis_penerimaan = ?", distribusiID, models.JenisTerimaBagiHasilBSU).
		Find(&penerimaBSU).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penerima BSU"})
		return
	}

	// ── 5. Bangun response ────────────────────────────────────────────────────
	type PenerimaBSIResp struct {
		PenerimaSisaID  string  `json:"penerima_sisa_id"`
		BankID          string  `json:"bank_id"`
		NamaBank        string  `json:"nama_bank"`
		NominalDiterima float64 `json:"nominal_diterima"`
		Porsi           float64 `json:"porsi"`
		Transportasi    float64 `json:"transportasi"`
		SatuanNominal   string  `json:"satuan_nominal"`
	}
	type PenerimaBSUResp struct {
		PenerimaSisaID  string            `json:"penerima_sisa_id"`
		BankID          string            `json:"bank_id"`
		NamaBank        string            `json:"nama_bank"`
		NominalDiterima float64           `json:"nominal_diterima"`
		Porsi           float64           `json:"porsi"`
		Transportasi    float64           `json:"transportasi"`
		SatuanNominal   string            `json:"satuan_nominal"`
		DiantarOleh     *models.AntarEnum `json:"diantar_oleh"`
	}

	bsuList := make([]PenerimaBSUResp, 0, len(penerimaBSU))
	for _, p := range penerimaBSU {
		bsuList = append(bsuList, PenerimaBSUResp{
			PenerimaSisaID:  p.PenerimaSisaID,
			BankID:          p.BankID,
			NamaBank:        p.BankSampah.NamaBank,
			NominalDiterima: p.NominalDiterima,
			Porsi:           p.Porsi,
			Transportasi:    p.Transportasi,
			SatuanNominal:   p.SatuanNominal,
			DiantarOleh:     p.DiantarOleh,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"distribusi_id": distribusi.DistribusiID,
		"bagi_hasil_id": distribusi.BagiHasilID,
		"total_sisa":    distribusi.TotalSisa,
		"satuan":        distribusi.SatuanTotal,
		"created_at":    distribusi.CreatedAt,
		"created_by":    namaAdmin,
		"penerima_bsi": PenerimaBSIResp{
			PenerimaSisaID:  penerimaBSI.PenerimaSisaID,
			BankID:          penerimaBSI.BankID,
			NamaBank:        penerimaBSI.BankSampah.NamaBank,
			NominalDiterima: penerimaBSI.NominalDiterima,
			Porsi:           penerimaBSI.Porsi,
			Transportasi:    penerimaBSI.Transportasi,
			SatuanNominal:   penerimaBSI.SatuanNominal,
		},
		"penerima_bsu": bsuList,
	})
}

// ─── ListBagiHasilBank ────────────────────────────────────────────────────────
// GET /distribusi-sisa/list-bh-bank/:bank_id
// Bisa digunakan oleh BSU maupun BSI
func (dsc *DistribusiSisaController) ListBagiHasilBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := dsc.DB.
		Where("bank_id = ? AND jenis_bank IN ?", bankID, []models.JenisBank{models.BSU, models.BSI}).
		First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	var jenisPenerimaan models.JenisTerimaEnum
	if bank.JenisBank == models.BSI {
		jenisPenerimaan = models.JenisTerimaBagiHasilBSI
	} else {
		jenisPenerimaan = models.JenisTerimaBagiHasilBSU
	}

	var penerima []models.PenerimaDistribusiSisa
	if err := dsc.DB.
		Preload("DistribusiSisa").
		Where("bank_id = ? AND jenis_penerimaan = ?", bankID, jenisPenerimaan).
		Find(&penerima).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat distribusi"})
		return
	}

	type ItemResp struct {
		PenerimaSisaID    string            `json:"penerima_sisa_id"`
		DistribusiID      string            `json:"distribusi_id"`
		BagiHasilID       string            `json:"bagi_hasil_id"`
		NominalDiterima   float64           `json:"nominal_diterima"`
		Porsi             float64           `json:"porsi"`
		Transportasi      float64           `json:"transportasi"`
		SatuanNominal     string            `json:"satuan_nominal"`
		DiantarOleh       *models.AntarEnum `json:"diantar_oleh,omitempty"`
		TanggalDistribusi time.Time         `json:"tanggal_distribusi"`
	}

	list := make([]ItemResp, 0, len(penerima))
	for _, p := range penerima {
		list = append(list, ItemResp{
			PenerimaSisaID:    p.PenerimaSisaID,
			DistribusiID:      p.DistribusiID,
			BagiHasilID:       p.DistribusiSisa.BagiHasilID,
			NominalDiterima:   p.NominalDiterima,
			Porsi:             p.Porsi,
			Transportasi:      p.Transportasi,
			SatuanNominal:     p.SatuanNominal,
			DiantarOleh:       p.DiantarOleh,
			TanggalDistribusi: p.DistribusiSisa.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"bank_id":    bankID,
		"nama_bank":  bank.NamaBank,
		"jenis_bank": bank.JenisBank,
		"riwayat":    list,
	})
}

// ─── DetailBagiHasilBank ──────────────────────────────────────────────────────
// GET /distribusi-sisa/detail-bh-bank/:penerima_sisa_id
// Bisa digunakan oleh BSU maupun BSI
func (dsc *DistribusiSisaController) DetailBagiHasilBank(c *gin.Context) {
	penerimaSisaID := c.Param("penerima_sisa_id")

	var penerima models.PenerimaDistribusiSisa
	if err := dsc.DB.
		Preload("BankSampah").
		Preload("DistribusiSisa.BagiHasil").
		Where("penerima_sisa_id = ?", penerimaSisaID).
		First(&penerima).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data penerimaan tidak ditemukan"})
		return
	}

	var perhitungan []models.PerhitunganSisa
	if err := dsc.DB.
		Preload("TabunganSampah.KatalogSampah").
		Where("penerima_sisa_id = ?", penerimaSisaID).
		Find(&perhitungan).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail perhitungan"})
		return
	}

	type PerhitunganResp struct {
		TabunganID string  `json:"tabungan_id"`
		NamaSampah string  `json:"nama_sampah"`
		QtyDipakai float64 `json:"qty_dipakai"`
		Satuan     string  `json:"satuan"`
	}

	perhitunganList := make([]PerhitunganResp, 0, len(perhitungan))
	for _, ph := range perhitungan {
		namaSampah := ""
		satuan := ""
		if ph.TabunganSampah.KatalogSampah != nil {
			namaSampah = ph.TabunganSampah.KatalogSampah.NamaSampah
			satuan = string(ph.TabunganSampah.KatalogSampah.Satuan)
		}
		perhitunganList = append(perhitunganList, PerhitunganResp{
			TabunganID: ph.TabunganID,
			NamaSampah: namaSampah,
			QtyDipakai: ph.QtyDipakai,
			Satuan:     satuan,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"penerima_sisa_id":   penerima.PenerimaSisaID,
		"distribusi_id":      penerima.DistribusiID,
		"bagi_hasil_id":      penerima.DistribusiSisa.BagiHasilID,
		"bank_id":            penerima.BankID,
		"nama_bank":          penerima.BankSampah.NamaBank,
		"nominal_diterima":   penerima.NominalDiterima,
		"porsi":              penerima.Porsi,
		"transportasi":       penerima.Transportasi,
		"satuan_nominal":     penerima.SatuanNominal,
		"diantar_oleh":       penerima.DiantarOleh,
		"tanggal_distribusi": penerima.DistribusiSisa.CreatedAt,
		"perhitungan_sisa":   perhitunganList,
	})
}