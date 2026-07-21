package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/utils"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DashboardController struct {
	DB *gorm.DB
}

func NewDashboardController(db *gorm.DB) *DashboardController {
	return &DashboardController{
		DB: db,
	}
}

func (dc *DashboardController) GetDashboardPetugas(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID is required"})
		return
	}

	var bank models.BankSampah
	if err := dc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank not found"})
		return
	}

	var jumlahNasabah int64
	dc.DB.Model(&models.Nasabah{}).Where("bank_id = ?", bankID).Count(&jumlahNasabah)

	var jumlahStaff int64
	dc.DB.Model(&models.Admin{}).Where("bank_id = ?", bankID).Count(&jumlahStaff)

	// Ambil kas bank dari saldo_rekening untuk entitas bank
	var kasList []models.SaldoRekening
	dc.DB.Preload("Reward").Where("bank_id = ? AND entitas = ?", bankID, models.EntitasBankSampah).Find(&kasList)

	var totalUang, totalPoin float64
	for _, k := range kasList {
		if k.Reward == nil {
			continue
		}
		if k.Reward.NamaReward == models.RewardEnumUang {
			totalUang += k.NominalSaldo
		} else if k.Reward.NamaReward == models.RewardEnumSembako {
			totalPoin += k.NominalSaldo
		}
	}

	kekayaanBank := gin.H{
		"total_uang": totalUang,
	}
	if bank.JenisBank != models.BSU {
		kekayaanBank["total_poin"] = totalPoin
	}

	switch bank.JenisBank {
	case models.BSU:
		var namaBankPusat string
		if bank.ParentBankID != nil {
			var parentBank models.BankSampah
			if err := dc.DB.Where("bank_id = ?", *bank.ParentBankID).First(&parentBank).Error; err == nil {
				namaBankPusat = parentBank.NamaBank
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"nama_bank":       bank.NamaBank,
				"photo_bank":      bank.PhotoURL,
				"alamat_bank":     bank.Alamat,
				"nama_bank_pusat": namaBankPusat,
				"jumlah_nasabah":  jumlahNasabah,
				"jumlah_staff":    jumlahStaff,
				"kas":             kekayaanBank,
			},
		})

	case models.BSI:
		var jumlahBSU int64
		dc.DB.Model(&models.BankSampah{}).Where("parent_bank_id = ? AND jenis_bank = ?", bankID, models.BSU).Count(&jumlahBSU)
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"nama_bank":      bank.NamaBank,
				"photo_bank":     bank.PhotoURL,
				"alamat_bank":    bank.Alamat,
				"jumlah_nasabah": jumlahNasabah,
				"jumlah_bsu":     jumlahBSU,
				"jumlah_staff":   jumlahStaff,
				"kas":            kekayaanBank,
			},
		})

	case models.BSM:
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"nama_bank":      bank.NamaBank,
				"photo_bank":     bank.PhotoURL,
				"alamat_bank":    bank.Alamat,
				"jumlah_nasabah": jumlahNasabah,
				"jumlah_staff":   jumlahStaff,
				"kas":            kekayaanBank,
			},
		})

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bank type"})
	}
}

func (dc *DashboardController) GetSaldoBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	// Validasi bank
	var bank models.BankSampah
	if err := dc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Bank tidak ditemukan"})
		return
	}

	type SaldoUang struct {
		TotalUang  float64                 `json:"total_uang"`
		SatuanUang models.SatuanRewardEnum `json:"satuan_uang"`
	}

	type SaldoPoin struct {
		TotalPoin  float64                 `json:"total_poin"`
		SatuanPoin models.SatuanRewardEnum `json:"satuan_poin"`
	}

	type ResponseSaldo struct {
		Uang SaldoUang  `json:"uang"`
		Poin *SaldoPoin `json:"poin,omitempty"`
	}

	res := ResponseSaldo{
		Uang: SaldoUang{TotalUang: 0, SatuanUang: models.SatuanRewardEnumRp},
	}

	if bank.JenisBank != models.BSU {
		res.Poin = &SaldoPoin{TotalPoin: 0, SatuanPoin: models.SatuanRewardEnumPoin}
	}

	var saldoRekening []models.SaldoRekening
	if err := dc.DB.Preload("Reward").Where("bank_id = ? AND entitas = ?", bankID, models.EntitasBankSampah).Find(&saldoRekening).Error; err == nil {
		for _, s := range saldoRekening {
			if s.Reward == nil {
				continue
			}

			switch s.Reward.NamaReward {
			case models.RewardEnumUang:
				res.Uang.TotalUang += s.NominalSaldo
			case models.RewardEnumSembako:
				if res.Poin != nil {
					res.Poin.TotalPoin += s.NominalSaldo
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data saldo bank berhasil diambil",
		"data":    res,
	})
}

func (dc *DashboardController) GetSaldoNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var nasabah models.Nasabah
	if err := dc.DB.Preload("User").Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Nasabah tidak ditemukan"})
		return
	}

	type SaldoUang struct {
		TotalUang  float64                 `json:"total_uang"`
		SatuanUang models.SatuanRewardEnum `json:"satuan_uang"`
	}

	type SaldoPoin struct {
		TotalPoin  float64                 `json:"total_poin"`
		SatuanPoin models.SatuanRewardEnum `json:"satuan_poin"`
	}

	type ResponseSaldo struct {
		Uang SaldoUang `json:"uang"`
		Poin SaldoPoin `json:"poin"`
	}

	res := ResponseSaldo{
		Uang: SaldoUang{TotalUang: 0, SatuanUang: models.SatuanRewardEnumRp},
		Poin: SaldoPoin{TotalPoin: 0, SatuanPoin: models.SatuanRewardEnumPoin},
	}

	var saldoRekening []models.SaldoRekening
	if err := dc.DB.Where("nasabah_id = ? AND entitas = ?", nasabahID, models.EntitasNasabah).Find(&saldoRekening).Error; err == nil {
		for _, s := range saldoRekening {
			switch s.SatuanNominalSaldo {
			case models.SatuanRewardEnumRp:
				res.Uang.TotalUang += s.NominalSaldo
			case models.SatuanRewardEnumPoin:
				res.Poin.TotalPoin += s.NominalSaldo
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data saldo nasabah berhasil diambil",
		"data":    res,
	})
}

func (dc *DashboardController) MutasiSaldoBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := dc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Bank tidak ditemukan"})
		return
	}

	var filter struct {
		RewardID  int    `form:"reward_id"`
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
	}
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Filter tidak valid"})
		return
	}

	rewardID := filter.RewardID
	if rewardID == 0 {
		rewardID = 1
	}

	var rekeningBank models.SaldoRekening
	if err := dc.DB.Preload("Reward").
		Where("bank_id = ? AND reward_id = ? AND entitas = ?", bankID, rewardID, models.EntitasBankSampah).
		First(&rekeningBank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Rekening untuk jenis reward ini tidak ditemukan"})
		return
	}

	query := dc.DB.Where("rekening_id = ?", rekeningBank.RekeningID)

	if filter.StartDate != "" {
		if start, err := time.Parse("2006-01-02", filter.StartDate); err == nil {
			query = query.Where("created_at >= ?", start)
		}
	}
	if filter.EndDate != "" {
		if end, err := time.Parse("2006-01-02", filter.EndDate); err == nil {
			end = end.Add(24*time.Hour - time.Second)
			query = query.Where("created_at <= ?", end)
		}
	}

	var arusSaldo []models.RiwayatArusSaldo
	if err := query.Order("created_at DESC").Find(&arusSaldo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil riwayat arus saldo"})
		return
	}

	type DebitKredit struct {
		IsPositive       bool      `json:"is_positive"`
		Nominal          float64   `json:"nominal"`
		Keterangan       string    `json:"keterangan"`
		TanggalTransaksi time.Time `json:"tanggal_transaksi"`
	}

	var mutasiItems []DebitKredit
	var totalDebit float64 = 0
	var totalKredit float64 = 0

	for _, s := range arusSaldo {
		nominal := 0.0
		isPositive := false

		if s.NominalSesudah > s.NominalSebelum {
			nominal = s.NominalSesudah - s.NominalSebelum
			isPositive = true
			totalKredit += nominal
		} else if s.NominalSebelum > s.NominalSesudah {
			nominal = s.NominalSebelum - s.NominalSesudah
			isPositive = false
			totalDebit += nominal
		} else {
			continue
		}

		mutasiItems = append(mutasiItems, DebitKredit{
			IsPositive:       isPositive,
			Nominal:          nominal,
			Keterangan:       s.Keterangan,
			TanggalTransaksi: s.CreatedAt,
		})
	}

	resp := gin.H{
		"nama_reward":   rekeningBank.Reward.NamaReward,
		"satuan_reward": rekeningBank.Reward.Satuan,
		"total_debit":   totalDebit,
		"total_kredit":  totalKredit,
		"mutasi_items":  mutasiItems,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data mutasi saldo bank berhasil diambil",
		"data":    resp,
	})
}

func (dc *DashboardController) MutasiSaldoNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	// 1. Validasi Nasabah
	var nasabah models.Nasabah
	if err := dc.DB.Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Nasabah tidak ditemukan"})
		return
	}

	// 2. Filter dari Query Params
	var filter struct {
		RewardID  int    `form:"reward_id"`
		StartDate string `form:"start_date"`
		EndDate   string `form:"end_date"`
	}
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Filter tidak valid"})
		return
	}

	// 3. Cari Rekening Nasabah (Default ke reward_id 1 jika tidak diisi)
	rewardID := filter.RewardID
	if rewardID == 0 {
		rewardID = 1 // Default: Uang
	}

	var rekeningNasabah models.SaldoRekening
	if err := dc.DB.Preload("Reward").
		Where("nasabah_id = ? AND reward_id = ? AND entitas = ?", nasabahID, rewardID, models.EntitasNasabah).
		First(&rekeningNasabah).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Rekening untuk jenis reward ini tidak ditemukan"})
		return
	}

	// 4. Query Arus Saldo dengan Filter Tanggal
	query := dc.DB.Where("rekening_id = ?", rekeningNasabah.RekeningID)

	if filter.StartDate != "" {
		if start, err := time.Parse("2006-01-02", filter.StartDate); err == nil {
			query = query.Where("created_at >= ?", start)
		}
	}
	if filter.EndDate != "" {
		if end, err := time.Parse("2006-01-02", filter.EndDate); err == nil {
			// Tambahkan 23:59:59 untuk mencakup seluruh hari terakhir
			end = end.Add(24*time.Hour - time.Second)
			query = query.Where("created_at <= ?", end)
		}
	}

	var arusSaldo []models.RiwayatArusSaldo
	if err := query.Order("created_at DESC").Find(&arusSaldo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mengambil riwayat arus saldo"})
		return
	}

	// 5. Kalkulasi Mutasi
	type DebitKredit struct {
		IsPositive       bool      `json:"is_positive"`
		Nominal          float64   `json:"nominal"`
		TanggalTransaksi time.Time `json:"tanggal_transaksi"`
	}

	var mutasiItems []DebitKredit
	var totalDebit float64 = 0
	var totalKredit float64 = 0

	for _, s := range arusSaldo {
		nominal := 0.0
		isPositive := false

		if s.NominalSesudah > s.NominalSebelum {
			nominal = s.NominalSesudah - s.NominalSebelum
			isPositive = true
			totalKredit += nominal
		} else if s.NominalSebelum > s.NominalSesudah {
			nominal = s.NominalSebelum - s.NominalSesudah
			isPositive = false
			totalDebit += nominal
		} else {
			// Jika nominal tetap, mungkin sisa yang berubah (e.g. blokir saldo)
			continue
		}

		mutasiItems = append(mutasiItems, DebitKredit{
			IsPositive:       isPositive,
			Nominal:          nominal,
			TanggalTransaksi: s.CreatedAt,
		})
	}

	// 6. Response
	resp := gin.H{
		"nama_reward":   rekeningNasabah.Reward.NamaReward,
		"satuan_reward": rekeningNasabah.Reward.Satuan,
		"total_debit":   totalDebit,
		"total_kredit":  totalKredit,
		"mutasi_items":  mutasiItems,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data mutasi saldo nasabah berhasil diambil",
		"data":    resp,
	})
}

func (dc *DashboardController) CatatManualMutasiBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := dc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Bank tidak ditemukan"})
		return
	}

	var body struct {
		TipeMutasi  string  `json:"tipe_mutasi" binding:"required,oneof=debit kredit"`
		JenisReward string  `json:"jenis_reward" binding:"required,oneof=uang poin"`
		Nominal     float64 `json:"nominal" binding:"required,gt=0"`
		Keterangan  string  `json:"keterangan" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Request tidak valid", "error": err.Error()})
		return
	}

	rewardName := models.RewardEnumUang
	if body.JenisReward == "poin" {
		rewardName = models.RewardEnumSembako
	}

	var reward models.Reward
	if err := dc.DB.Where("nama_reward = ?", rewardName).First(&reward).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Jenis reward tidak ditemukan"})
		return
	}

	rewardID := reward.RewardID
	satuan := models.SatuanRewardEnumRp
	if body.JenisReward == "poin" {
		satuan = models.SatuanRewardEnumPoin
	}

	err := dc.DB.Transaction(func(tx *gorm.DB) error {
		var rekeningBank models.SaldoRekening
		findErr := tx.Where("bank_id = ? AND reward_id = ? AND entitas = ?", bankID, rewardID, models.EntitasBankSampah).
			First(&rekeningBank).Error

		if findErr != nil {
			if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return findErr
			}
			rekeningBank = models.SaldoRekening{
				RekeningID:         utils.GenerateRekeningID(rewardID, bankID),
				BankID:             &bankID,
				RewardID:           &rewardID,
				Entitas:            models.EntitasBankSampah,
				NominalSaldo:       0,
				SatuanNominalSaldo: satuan,
			}
			if err := tx.Create(&rekeningBank).Error; err != nil {
				return err
			}
		}

		saldoSebelum := rekeningBank.NominalSaldo
		var saldoSesudah float64

		if body.TipeMutasi == "kredit" {
			saldoSesudah = saldoSebelum + body.Nominal
		} else {
			if saldoSebelum < body.Nominal {
				return fmt.Errorf("saldo tidak mencukupi")
			}
			saldoSesudah = saldoSebelum - body.Nominal
		}

		if err := tx.Model(&rekeningBank).Update("nominal_saldo", saldoSesudah).Error; err != nil {
			return err
		}

		riwayat := models.RiwayatArusSaldo{
			RiwayatSaldoID: utils.GenerateID("RS"),
			RekeningID:     &rekeningBank.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			Keterangan:     body.Keterangan,
		}
		return tx.Create(&riwayat).Error
	})

	if err != nil {
		if err.Error() == "saldo tidak mencukupi" {
			c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Gagal mencatat mutasi", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Mutasi berhasil dicatat"})
}

func (dc *DashboardController) TotalSaldoAllNasabah(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := dc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Bank tidak ditemukan"})
		return
	}

	type response struct {
		TotalUang float64 `json:"total_uang"`
		TotalPoin float64 `json:"total_poin"`
	}

	var nasabahIDs []string
	dc.DB.Model(&models.Nasabah{}).Where("bank_id = ?", bankID).Pluck("nasabah_id", &nasabahIDs)

	res := response{}

	if len(nasabahIDs) > 0 {
		dc.DB.Model(&models.SaldoRekening{}).
			Where("nasabah_id IN ? AND entitas = ? AND satuan_nominal_saldo = ?", nasabahIDs, models.EntitasNasabah, models.SatuanRewardEnumRp).
			Select("COALESCE(SUM(nominal_saldo), 0)").
			Scan(&res.TotalUang)

		dc.DB.Model(&models.SaldoRekening{}).
			Where("nasabah_id IN ? AND entitas = ? AND satuan_nominal_saldo = ?", nasabahIDs, models.EntitasNasabah, models.SatuanRewardEnumPoin).
			Select("COALESCE(SUM(nominal_saldo), 0)").
			Scan(&res.TotalPoin)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Total saldo semua nasabah berhasil diambil",
		"data":    res,
	})
}

func (dc *DashboardController) ListSaldoAllNasabah(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := dc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": "Bank tidak ditemukan"})
		return
	}

	type nasabahRow struct {
		NasabahID     string            `json:"nasabah_id"`
		NamaNasabah   string            `json:"nama_nasabah"`
		StatusNasabah models.StatusAkun `json:"status_nasabah"`
		SaldoUang     float64           `json:"saldo_uang"`
		SaldoPoin     float64           `json:"saldo_poin"`
	}

	const limit = 20
	page := 1
	if p, err := fmt.Sscanf(c.DefaultQuery("page", "1"), "%d", &page); p == 0 || err != nil || page < 1 {
		page = 1
	}
	offset := (page - 1) * limit

	var total int64
	dc.DB.Model(&models.Nasabah{}).Where("bank_id = ?", bankID).Count(&total)

	var nasabahList []models.Nasabah
	dc.DB.Preload("User").Where("bank_id = ?", bankID).Limit(limit).Offset(offset).Find(&nasabahList)

	if len(nasabahList) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "Berhasil mengambil list saldo nasabah",
			"data":    []nasabahRow{},
			"meta": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": (total + int64(limit) - 1) / int64(limit),
			},
		})
		return
	}

	nasabahIDs := make([]string, len(nasabahList))
	for i, n := range nasabahList {
		nasabahIDs[i] = n.NasabahID
	}

	var saldoList []models.SaldoRekening
	dc.DB.Where("nasabah_id IN ? AND entitas = ?", nasabahIDs, models.EntitasNasabah).Find(&saldoList)

	type saldoPair struct{ uang, poin float64 }
	saldoMap := make(map[string]saldoPair)
	for _, s := range saldoList {
		if s.NasabahID == nil {
			continue
		}
		pair := saldoMap[*s.NasabahID]
		switch s.SatuanNominalSaldo {
		case models.SatuanRewardEnumRp:
			pair.uang += s.NominalSaldo
		case models.SatuanRewardEnumPoin:
			pair.poin += s.NominalSaldo
		}
		saldoMap[*s.NasabahID] = pair
	}

	response := make([]nasabahRow, 0, len(nasabahList))
	for _, n := range nasabahList {
		pair := saldoMap[n.NasabahID]
		response = append(response, nasabahRow{
			NasabahID:     n.NasabahID,
			NamaNasabah:   n.User.Nama,
			StatusNasabah: n.StatusNasabah,
			SaldoUang:     pair.uang,
			SaldoPoin:     pair.poin,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil list saldo nasabah",
		"data":    response,
		"meta": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}
