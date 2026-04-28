package controllers

import (
	"encoding/json"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PenjualanController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
}

func NewPenjualanController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *PenjualanController {
	return &PenjualanController{
		DB:        db,
		CFStorage: cfStorage,
	}
}

type ItemSampahDijual struct {
	SampahID string  `json:"sampah_id" binding:"required"`
	Qty      float64 `json:"qty" binding:"required"`
}

type ItemSembakoDidapat struct {
	SembakoID string  `json:"sembako_id" binding:"required"`
	Qty       float64 `json:"qty" binding:"required"`
}

type InputPenjualan struct {
	RewardID         int                  `json:"reward_id" binding:"required"`
	IdentitasPembeli string               `json:"identitas_pembeli" binding:"required"`
	ItemsSampah      []ItemSampahDijual   `json:"items_sampah" binding:"required"`
	ItemsSembako     []ItemSembakoDidapat `json:"items_sembako"` // Opsional
}

func (pc *PenjualanController) AddNewPenjualanEksternal(c *gin.Context) {
	bankID := c.Param("bank_id")
	adminID := c.Param("admin_id")

	// Parse multipart form
	if err := c.Request.ParseMultipartForm(10 << 20); err != nil { // 10 MB limit
		c.JSON(http.StatusBadRequest, gin.H{"error": "Gagal membaca form data"})
		return
	}

	rewardIDStr := c.PostForm("reward_id")
	rewardID, err := strconv.Atoi(rewardIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format reward_id tidak valid"})
		return
	}

	identitasPembeli := c.PostForm("identitas_pembeli")
	if identitasPembeli == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Identitas pembeli wajib diisi"})
		return
	}

	// Parse items_sampah (JSON string)
	itemsSampahStr := c.PostForm("items_sampah")
	var itemsSampah []ItemSampahDijual
	if err := json.Unmarshal([]byte(itemsSampahStr), &itemsSampah); err != nil || len(itemsSampah) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format items_sampah tidak valid atau kosong"})
		return
	}

	// Parse items_sembako (JSON string, optional)
	itemsSembakoStr := c.PostForm("items_sembako")
	var itemsSembako []ItemSembakoDidapat
	if itemsSembakoStr != "" {
		if err := json.Unmarshal([]byte(itemsSembakoStr), &itemsSembako); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Format items_sembako tidak valid"})
			return
		}
	}

	// Handle Foto Upload
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

	buktiFotoURL, err := pc.CFStorage.UploadFile(file, fileHeader, "penjualan_eksternal")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal upload bukti foto: " + err.Error()})
		return
	}

	var input InputPenjualan
	input.RewardID = rewardID
	input.IdentitasPembeli = identitasPembeli
	input.ItemsSampah = itemsSampah
	input.ItemsSembako = itemsSembako

	admin := adminID

	// Get Reward
	var reward models.Reward
	if err := pc.DB.Where("reward_id = ?", input.RewardID).First(&reward).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reward tidak ditemukan"})
		return
	}

	isBarterSembako := strings.ToLower(reward.NamaReward) == "sembako"

	if isBarterSembako && len(input.ItemsSembako) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "items_sembako wajib diisi jika jenis reward adalah sembako"})
		return
	}

	// Get NilaiRewardBank (Hanya jika BUKAN Sembako)
	var nilaiRewardBank models.NilaiRewardBank
	if !isBarterSembako {
		if err := pc.DB.Where("bank_id = ? AND reward_id = ?", bankID, input.RewardID).First(&nilaiRewardBank).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Nilai konversi reward belum diatur untuk bank ini"})
			return
		}
	} else {
		// Mock nilai konversi 1:1 untuk sembako karena sembako tidak pakai rasio nilai_reward_bank
		nilaiRewardBank.NilaiPoin = 1
		nilaiRewardBank.NilaiKonversi = 1
	}

	tx := pc.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
		return
	}

	// Get SaldoBank (Initialize if not exists)
	var saldoBank models.SaldoBank
	if err := tx.Where("bank_id = ?", bankID).First(&saldoBank).Error; err != nil {
		saldoBank = models.SaldoBank{
			SaldoBankID:   utils.GenerateID("SLB"),
			BankID:        bankID,
			TotalPoin:     0,
			LastUpdatedAt: time.Now(),
			LastUpdatedBy: admin,
		}
		if err := tx.Create(&saldoBank).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal inisialisasi saldo bank: " + err.Error()})
			return
		}
	}

	// Hitung Total Poin dari Sampah
	var totalPoin float64 = 0
	var detailPenjualans []models.DetailPenjualan

	penjualanID := utils.GenerateBankRelatedID(bankID)

	for _, item := range input.ItemsSampah {
		// Get Sampah
		var sampah models.KatalogSampah
		if err := tx.Where("sampah_id = ?", item.SampahID).First(&sampah).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Sampah tidak ditemukan: " + item.SampahID})
			return
		}

		// Get Harga Eksternal
		var schemaHarga models.SchemaHargaSampah
		if err := tx.Where("sampah_id = ? AND level_user = ?", item.SampahID, models.LevelEksternal).First(&schemaHarga).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Harga eksternal untuk sampah " + sampah.NamaSampah + " belum diatur"})
			return
		}

		// Get Stok
		var stok models.StokSampah
		if err := tx.Where("bank_id = ? AND sampah_id = ?", bankID, item.SampahID).First(&stok).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok sampah tidak ada di bank: " + item.SampahID})
			return
		}

		if stok.Stok < item.Qty {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Stok sampah " + sampah.NamaSampah + " tidak mencukupi"})
			return
		}

		poinJual := schemaHarga.PoinHarga
		subtotalPoin := item.Qty * poinJual
		totalPoin += subtotalPoin

		subtotalKonversi := (subtotalPoin / nilaiRewardBank.NilaiPoin) * nilaiRewardBank.NilaiKonversi

		detailPenjualans = append(detailPenjualans, models.DetailPenjualan{
			PenjualanID:      penjualanID,
			SampahID:         item.SampahID,
			Qty:              item.Qty,
			PoinJual:         poinJual,
			NilaiPoin:        nilaiRewardBank.NilaiPoin,
			NilaiKonversi:    nilaiRewardBank.NilaiKonversi,
			SubtotalPoin:     subtotalPoin,
			SubtotalKonversi: subtotalKonversi,
		})

		// Deduct Stok Sampah
		if err := tx.Model(&models.StokSampah{}).Where("bank_id = ? AND sampah_id = ?", bankID, item.SampahID).Update("stok", stok.Stok-item.Qty).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengurangi stok sampah"})
			return
		}
	}

	// Validasi dan Hitung Sembako (jika barter)
	var poinDitambahkan float64 = totalPoin
	if isBarterSembako {
		var totalPoinSembako float64 = 0
		for _, item := range input.ItemsSembako {
			var schemaSembako models.SchemaHargaSembako
			if err := tx.Where("sembako_id = ? AND level_user = ?", item.SembakoID, models.LevelEksternal).First(&schemaSembako).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Harga eksternal untuk sembako " + item.SembakoID + " belum diatur"})
				return
			}
			totalPoinSembako += item.Qty * schemaSembako.PoinHarga
		}

		if totalPoinSembako > totalPoin {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": "Total nilai sembako melebihi nilai sampah yang dijual"})
			return
		}
		poinDitambahkan = totalPoinSembako
	}

	// Save Penjualan
	penjualan := models.Penjualan{
		PenjualanID:      penjualanID,
		BankID:           bankID,
		RewardID:         input.RewardID,
		IdentitasPembeli: input.IdentitasPembeli,
		TotalItem:        len(input.ItemsSampah),
		TotalPoin:        totalPoin,
		SoldBy:           admin,
		BuktiFoto:        buktiFotoURL,
		CreatedAt:        time.Now(),
	}

	if err := tx.Create(&penjualan).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan data penjualan: " + err.Error()})
		return
	}

	// Save Detail Penjualan
	if err := tx.Create(&detailPenjualans).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan detail penjualan: " + err.Error()})
		return
	}

	// Update Saldo Bank (Kapasitas Bertambah sesuai aset yang diterima)
	saldoSebelum := saldoBank.TotalPoin
	saldoSesudah := saldoBank.TotalPoin + poinDitambahkan

	saldoBank.TotalPoin = saldoSesudah
	saldoBank.LastUpdatedAt = time.Now()
	saldoBank.LastUpdatedBy = admin

	if err := tx.Save(&saldoBank).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate saldo bank: " + err.Error()})
		return
	}

	// Simpan Transaksi Saldo Bank
	transaksiSaldo := models.TransaksiSaldoBank{
		TransaksiBankID: utils.GenerateID("TXB"),
		SaldoBankID:     saldoBank.SaldoBankID,
		JenisTransaksi:  models.TransaksiPenjualan,
		Jumlah:          poinDitambahkan,
		SaldoSebelum:    saldoSebelum,
		SaldoSesudah:    saldoSesudah,
		UpdatedAt:       time.Now(),
	}
	if err := tx.Create(&transaksiSaldo).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan log transaksi saldo: " + err.Error()})
		return
	}

	// Proses Reward (Kas vs Sembako)
	if isBarterSembako {
		var detailBarters []models.DetailBarter
		for _, item := range input.ItemsSembako {
			
			// Ambil harga poin sembako dari schema (Level Eksternal)
			var schemaSembako models.SchemaHargaSembako
			if err := tx.Where("sembako_id = ? AND level_user = ?", item.SembakoID, models.LevelEksternal).First(&schemaSembako).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{"error": "Harga eksternal untuk sembako " + item.SembakoID + " belum diatur"})
				return
			}
			
			hargaPoin := schemaSembako.PoinHarga
			subtotalPoinSembako := item.Qty * hargaPoin

			detailBarters = append(detailBarters, models.DetailBarter{
				PenjualanID:  penjualanID,
				SembakoID:    item.SembakoID,
				Qty:          item.Qty,
				HargaPoin:    hargaPoin,
				SubtotalPoin: subtotalPoinSembako,
			})

			// Add Stok Sembako
			var stokSembako models.StokSembakoBank
			errStok := tx.Where("bank_id = ? AND sembako_id = ?", bankID, item.SembakoID).First(&stokSembako).Error
			if errStok == nil {
				if err := tx.Model(&models.StokSembakoBank{}).Where("bank_id = ? AND sembako_id = ?", bankID, item.SembakoID).Update("stok", stokSembako.Stok+item.Qty).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambah stok sembako: " + err.Error()})
					return
				}
			} else {
				// Create initial stock if not exists
				newStok := models.StokSembakoBank{
					BankID:    bankID,
					SembakoID: item.SembakoID,
					Stok:      item.Qty,
				}
				if err := tx.Create(&newStok).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat stok awal sembako: " + err.Error()})
					return
				}
			}
		}

		if err := tx.Create(&detailBarters).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan detail barter: " + err.Error()})
			return
		}
	} else {
		// Kas Bank (Uang/Emas)
		totalAsetDidapat := (totalPoin / nilaiRewardBank.NilaiPoin) * nilaiRewardBank.NilaiKonversi

		var kas models.KasBank
		errKas := tx.Where("bank_id = ? AND reward_id = ?", bankID, input.RewardID).First(&kas).Error

		var currentKasID uuid.UUID
		var nominalSebelum float64
		var nominalSesudah float64

		if errKas == nil {
			currentKasID = kas.KasID
			nominalSebelum = kas.Nominal
			nominalSesudah = kas.Nominal + totalAsetDidapat

			kas.Nominal = nominalSesudah
			kas.LastUpdatedAt = time.Now()
			kas.LastUpdatedBy = admin

			if err := tx.Save(&kas).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate kas bank: " + err.Error()})
				return
			}
		} else {
			currentKasID = uuid.New()
			nominalSebelum = 0
			nominalSesudah = totalAsetDidapat

			newKas := models.KasBank{
				KasID:         currentKasID,
				BankID:        bankID,
				RewardID:      input.RewardID,
				Nominal:       nominalSesudah,
				LastUpdatedAt: time.Now(),
				LastUpdatedBy: admin,
			}

			if err := tx.Create(&newKas).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat kas bank: " + err.Error()})
				return
			}
		}

		// Riwayat Arus Kas
		riwayatKas := models.RiwayatArusKas{
			ArusKasID:      utils.GenerateID("AK"),
			KasID:          currentKasID.String(),
			Jumlah:         totalAsetDidapat,
			NominalSebelum: nominalSebelum,
			NominalSesudah: nominalSesudah,
			CreatedAt:      time.Now(),
		}

		if err := tx.Create(&riwayatKas).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat riwayat arus kas: " + err.Error()})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal commit transaksi: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Penjualan eksternal berhasil dicatat",
	})
}

func (pc *PenjualanController) GetRiwayatPenjualanEksternal(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := pc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	if bank.JenisBank != models.BSI && bank.JenisBank != models.BSM {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur ini hanya untuk BSI dan BSM"})
		return
	}

	type RiwayatResponse struct {
		PenjualanID      string    `json:"penjualan_id"`
		IdentitasPembeli string    `json:"identitas_pembeli"`
		RewardName       string    `json:"reward_name"`
		TotalItem        int       `json:"total_item"`
		TotalPoin        float64   `json:"total_poin"`
		TotalKonversi    float64   `json:"total_konversi"`
		Satuan           string    `json:"satuan"`
		BuktiFoto        string    `json:"bukti_foto"`
		CreatedAt        time.Time `json:"created_at"`
		AdminName        string    `json:"admin_name"`
	}

	var riwayats []RiwayatResponse
	if err := pc.DB.Table("penjualan").
		Select("penjualan.penjualan_id, penjualan.identitas_pembeli, reward.nama_reward as reward_name, reward.satuan, penjualan.total_item, penjualan.total_poin, COALESCE(SUM(detail_penjualan.subtotal_konversi), 0) as total_konversi, penjualan.bukti_foto, penjualan.created_at, users.nama as admin_name").
		Joins("left join reward on penjualan.reward_id = reward.reward_id").
		Joins("left join detail_penjualan on penjualan.penjualan_id = detail_penjualan.penjualan_id").
		Joins("left join admin on penjualan.sold_by = admin.admin_id").
		Joins("left join users on admin.user_id = users.user_id").
		Where("penjualan.bank_id = ?", bankID).
		Group("penjualan.penjualan_id, reward.nama_reward, reward.satuan, users.nama").
		Order("penjualan.created_at desc").
		Find(&riwayats).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat penjualan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Riwayat penjualan berhasil diambil",
		"data":    riwayats,
	})
}

func (pc *PenjualanController) DetailPenjualanEksternal(c *gin.Context) {
	penjualanID := c.Param("penjualan_id")

	var penjualan models.Penjualan
	if err := pc.DB.Preload("Reward").Preload("BankSampah").Where("penjualan_id = ?", penjualanID).First(&penjualan).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data penjualan tidak ditemukan"})
		return
	}

	if penjualan.BankSampah.JenisBank != models.BSI && penjualan.BankSampah.JenisBank != models.BSM {
		c.JSON(http.StatusForbidden, gin.H{"error": "Fitur ini hanya untuk BSI dan BSM"})
		return
	}

	type SampahDetail struct {
		SampahID         string  `json:"sampah_id"`
		NamaSampah       string  `json:"nama_sampah"`
		Qty              float64 `json:"qty"`
		PoinJual         float64 `json:"poin_jual"`
		NilaiKonversi    float64 `json:"nilai_konversi"`
		SubtotalPoin     float64 `json:"subtotal_poin"`
		SubtotalKonversi float64 `json:"subtotal_konversi"`
	}

	type SembakoDetail struct {
		SembakoID    string  `json:"sembako_id"`
		NamaSembako  string  `json:"nama_sembako"`
		Qty          float64 `json:"qty"`
		HargaPoin    float64 `json:"harga_poin"`
		SubtotalPoin float64 `json:"subtotal_poin"`
	}

	var rawDetailSampah []models.DetailPenjualan
	pc.DB.Preload("Sampah").Where("penjualan_id = ?", penjualanID).Find(&rawDetailSampah)

	var itemsSampah []SampahDetail
	for _, ds := range rawDetailSampah {
		itemsSampah = append(itemsSampah, SampahDetail{
			SampahID:         ds.SampahID,
			NamaSampah:       ds.Sampah.NamaSampah,
			Qty:              ds.Qty,
			PoinJual:         ds.PoinJual,
			NilaiKonversi:    ds.NilaiKonversi,
			SubtotalPoin:     ds.SubtotalPoin,
			SubtotalKonversi: ds.SubtotalKonversi,
		})
	}

	var itemsSembako []SembakoDetail
	isBarterSembako := strings.ToLower(penjualan.Reward.NamaReward) == "sembako"
	if isBarterSembako {
		var rawDetailSembako []models.DetailBarter
		pc.DB.Preload("Sembako").Where("penjualan_id = ?", penjualanID).Find(&rawDetailSembako)
		for _, db := range rawDetailSembako {
			itemsSembako = append(itemsSembako, SembakoDetail{
				SembakoID:    db.SembakoID,
				NamaSembako:  db.Sembako.NamaSembako,
				Qty:          db.Qty,
				HargaPoin:    db.HargaPoin,
				SubtotalPoin: db.SubtotalPoin,
			})
		}
	}

	// Fetch admin name for display
	var adminName string
	pc.DB.Table("users").Select("users.nama").
		Joins("join admin on admin.user_id = users.user_id").
		Where("admin.admin_id = ?", penjualan.SoldBy).
		Scan(&adminName)

	// Calculate total konversi for detail
	var totalKonversi float64 = 0
	for _, item := range itemsSampah {
		totalKonversi += item.SubtotalKonversi
	}

	response := gin.H{
		"penjualan_id":      penjualan.PenjualanID,
		"bank_id":           penjualan.BankID,
		"identitas_pembeli": penjualan.IdentitasPembeli,
		"reward_name":       penjualan.Reward.NamaReward,
		"satuan":            penjualan.Reward.Satuan,
		"total_item":        penjualan.TotalItem,
		"total_poin":        penjualan.TotalPoin,
		"total_konversi":    totalKonversi,
		"bukti_foto":        penjualan.BuktiFoto,
		"created_at":        penjualan.CreatedAt,
		"admin_name":        adminName,
		"items_sampah":      itemsSampah,
	}

	if isBarterSembako {
		response["items_sembako"] = itemsSembako
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail penjualan berhasil diambil",
		"data":    response,
	})
}
