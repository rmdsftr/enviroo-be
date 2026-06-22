package controllers

import (
	"context"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"enviroo-be/pkg/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SembakoController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
	NotifSvc  services.NotifikasiService
}

func NewSembakoController(db *gorm.DB, cfStorage *storage.CloudflareStorage, notifSvc services.NotifikasiService) *SembakoController {
	return &SembakoController{
		DB:        db,
		CFStorage: cfStorage,
		NotifSvc:  notifSvc,
	}
}

func (sc *SembakoController) GetMasterSembako(c *gin.Context) {
	query := c.Query("q")

	var results []models.Sembako

	db := sc.DB.Model(&models.Sembako{})

	if query != "" {
		db = db.Where("nama_barang ~* ?", query)
	}

	if err := db.Limit(10).Find(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sembako"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": results,
	})
}

// ─── AddNewSembako ───────────────────────────────────────────────────────────
// POST /sembako/add-sembako/:bank_id
// ─── AddNewSembako ───────────────────────────────────────────────────────────
// POST /sembako/add-sembako/:bank_id
func (sc *SembakoController) AddNewSembako(c *gin.Context) {
	bankID := c.Param("bank_id")

	var req struct {
		NamaBarang string  `form:"nama_barang" binding:"required"`
		BarangID   int     `form:"barang_id"`
		NilaiPoin  float64 `form:"nilai_poin" binding:"required"`
		StokAwal   float64 `form:"stok_awal"`
		CreatedBy  string  `form:"created_by"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Upload foto (opsional)
	var fotoURL *string
	if fileHeader, err := c.FormFile("foto"); err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
			return
		}
		defer file.Close()

		url, err := sc.CFStorage.UploadFile(file, fileHeader, "katalog_sembako")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto: " + err.Error()})
			return
		}
		fotoURL = &url
	}

	// Resolve BarangID: kalau di-supply langsung, validasi dulu.
	// Kalau tidak, FindOrCreate dari nama.
	var barangID int
	if req.BarangID > 0 {
		var masterSembako models.Sembako
		if err := sc.DB.First(&masterSembako, "barang_id = ?", req.BarangID).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "barang_id tidak valid"})
			return
		}
		barangID = masterSembako.BarangID
	} else {
		masterSembako, err := utils.FindOrCreateMasterSembako(sc.DB, req.NamaBarang)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses master sembako: " + err.Error()})
			return
		}
		barangID = masterSembako.BarangID
	}

	sembakoID := utils.GenerateBankRelatedID(bankID)

	tx := sc.DB.Begin()

	newKatalog := models.KatalogSembako{
		SembakoID: sembakoID,
		BankID:    &bankID,
		BarangID:  barangID,
		PhotoURL:  fotoURL,
		NilaiPoin: req.NilaiPoin,
		CreatedBy: &req.CreatedBy,
		UpdatedBy: &req.CreatedBy,
	}

	if err := tx.Create(&newKatalog).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat katalog: " + err.Error()})
		return
	}

	stokAwal := models.StokSembako{
		SembakoID: sembakoID,
		BankID:    bankID,
		Stok:      req.StokAwal,
	}

	if err := tx.Create(&stokAwal).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat stok awal: " + err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sembako berhasil ditambahkan",
		"data": gin.H{
			"katalog": newKatalog,
			"stok":    stokAwal,
		},
	})
}

// ─── GetSembakoBank ──────────────────────────────────────────────────────────
// GET /sembako/get-sembako/:bank_id
//
// BSI/BSM → ambil katalog_sembako milik bank itu sendiri.
// BSU     → ambil katalog_sembako milik BSU itu sendiri
//
//	(record BSU dibuat otomatis saat pertama kali distribusi dari BSI).
func (sc *SembakoController) GetSembakoBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	// BSU ambil katalog dari BSI induknya, stok tetap milik sendiri
	targetBankID := bankID
	if bank.JenisBank == models.BSU && bank.ParentBankID != nil {
		targetBankID = *bank.ParentBankID
	}

	type SembakoResponse struct {
		SembakoID  string  `json:"sembako_id" gorm:"column:sembako_id"`
		NamaBarang string  `json:"nama_barang" gorm:"column:nama_barang"`
		BarangID   int     `json:"barang_id" gorm:"column:barang_id"`
		PhotoURL   *string `json:"photo_url" gorm:"column:photo_url"`
		NilaiPoin  float64 `json:"nilai_poin" gorm:"column:nilai_poin"`
		Stok       float64 `json:"stok" gorm:"column:stok"`
		TotalCount int     `json:"-" gorm:"column:total_count"`
	}

	pageStr := c.Query("page")
	usePagination := pageStr != ""

	const limit = 12
	baseSQL := `
		SELECT
			ks.sembako_id,
			s.nama_barang,
			s.barang_id,
			ks.photo_url,
			ks.nilai_poin,
			COALESCE(st.stok, 0) AS stok
			%s
		FROM katalog_sembako ks
		JOIN sembako s ON s.barang_id = ks.barang_id
		LEFT JOIN stok_sembako st
			ON st.sembako_id = ks.sembako_id
			AND st.bank_id = ?
		WHERE ks.bank_id = ?
		ORDER BY s.nama_barang ASC
		%s`

	var rows []SembakoResponse
	var err error

	if !usePagination {
		q := fmt.Sprintf(baseSQL, "", "")
		err = sc.DB.Raw(q, bankID, targetBankID).Scan(&rows).Error
	} else {
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		offset := (page - 1) * limit
		q := fmt.Sprintf(baseSQL, ", COUNT(*) OVER() AS total_count", "LIMIT ? OFFSET ?")
		err = sc.DB.Raw(q, bankID, targetBankID, limit, offset).Scan(&rows).Error
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data sembako"})
		return
	}

	if rows == nil {
		rows = []SembakoResponse{}
	}

	if !usePagination {
		c.JSON(http.StatusOK, gin.H{
			"message": "Katalog sembako berhasil diambil",
			"data":    rows,
		})
		return
	}

	page, _ := strconv.Atoi(pageStr)
	if page < 1 {
		page = 1
	}
	totalCount := 0
	if len(rows) > 0 {
		totalCount = rows[0].TotalCount
	}
	totalPages := (totalCount + limit - 1) / limit

	c.JSON(http.StatusOK, gin.H{
		"message": "Katalog sembako berhasil diambil",
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       totalCount,
			"total_pages": totalPages,
		},
		"data": rows,
	})
}

// ─── GetDetailSembakoBSU ─────────────────────────────────────────────────────
// GET /sembako/detail-sembako-bsu/:sembako_id
//
// Mengembalikan info sembako BSU (dari katalog_sembako) beserta riwayat
// distribusi yang masuk ke BSU tersebut untuk item ini.
// :sembako_id adalah SembakoID dari katalog_sembako milik BSU (bukan BSI).
func (sc *SembakoController) GetDetailSembako(c *gin.Context) {
	sembakoID := c.Param("sembako_id")
	bankID := c.Query("bank_id")

	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bank_id wajib diisi"})
		return
	}

	// Validasi bank dan tentukan perspektif stok
	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	type riwayatItem struct {
		DisbakoID    string  `json:"disbako_id"`
		TanggalKirim string  `json:"tanggal_kirim"`
		Item         float64 `json:"item"`
		StokSebelum  float64 `json:"stok_sebelum"`
		StokSesudah  float64 `json:"stok_sesudah"`
	}

	var stokSebelumCol, stokSesudahCol, filterCol string
	if bank.JenisBank == models.BSU {
		stokSebelumCol = "d.stok_bsu_sebelum"
		stokSesudahCol = "d.stok_bsu_sesudah"
		filterCol = "dsb.bsu_id"
	} else {
		stokSebelumCol = "d.stok_bsi_sebelum"
		stokSesudahCol = "d.stok_bsi_sesudah"
		filterCol = "dsb.bsi_id"
	}

	query := fmt.Sprintf(`
		SELECT
			d.disbako_id,
			TO_CHAR(dsb.created_at, 'YYYY-MM-DD HH24:MI:SS') AS tanggal_kirim,
			d.item,
			%s AS stok_sebelum,
			%s AS stok_sesudah
		FROM detail_distribusi_sembako d
		JOIN distribusi_sembako_bsu dsb
			ON dsb.disbako_id = d.disbako_id
			AND %s = ?
		WHERE d.sembako_id = ?
		ORDER BY dsb.created_at DESC
	`, stokSebelumCol, stokSesudahCol, filterCol)

	var riwayat []riwayatItem
	if err := sc.DB.Raw(query, bankID, sembakoID).Scan(&riwayat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat distribusi"})
		return
	}

	if riwayat == nil {
		riwayat = []riwayatItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail sembako berhasil diambil",
		"data": gin.H{
			"riwayat_distribusi": riwayat,
		},
	})
}

// ─── EditSembako ─────────────────────────────────────────────────────────────
// PATCH /sembako/edit-sembako/:sembako_id
func (sc *SembakoController) EditSembako(c *gin.Context) {
	sembakoID := c.Param("sembako_id")

	var sembako models.KatalogSembako
	if err := sc.DB.Where("sembako_id = ?", sembakoID).First(&sembako).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sembako tidak ditemukan"})
		return
	}

	var req struct {
		NilaiPoin  float64 `form:"nilai_poin"`
		Stok       float64 `form:"stok"`
		TambahStok float64 `form:"tambah_stok"`
		UpdatedBy  string  `form:"updated_by"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Upload foto baru (opsional)
	if fileHeader, err := c.FormFile("foto"); err == nil {
		file, err := fileHeader.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membaca foto"})
			return
		}
		defer file.Close()

		fotoURL, err := sc.CFStorage.UploadFile(file, fileHeader, "katalog_sembako")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupload foto: " + err.Error()})
			return
		}
		sembako.PhotoURL = &fotoURL
	}

	tx := sc.DB.Begin()

	// Update kolom katalog yang berubah saja
	katalogUpdates := map[string]any{
		"updated_by": req.UpdatedBy,
		"photo_url":  sembako.PhotoURL,
	}
	if c.PostForm("nilai_poin") != "" {
		katalogUpdates["nilai_poin"] = req.NilaiPoin
	}

	if err := tx.Model(&sembako).Updates(katalogUpdates).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate katalog: " + err.Error()})
		return
	}

	// Update stok jika ada perubahan
	if req.TambahStok != 0 {
		if err := tx.Model(&models.StokSembako{}).
			Where("sembako_id = ?", sembakoID).
			UpdateColumn("stok", gorm.Expr("stok + ?", req.TambahStok)).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menambah stok: " + err.Error()})
			return
		}
	} else if c.PostForm("stok") != "" {
		if err := tx.Model(&models.StokSembako{}).
			Where("sembako_id = ?", sembakoID).
			UpdateColumn("stok", req.Stok).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate stok: " + err.Error()})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan transaksi"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Sembako berhasil diupdate",
		"data":    sembako,
	})
}

// ─── DeleteSembako ───────────────────────────────────────────────────────────
// DELETE /sembako/delete-sembako/:sembako_id
func (sc *SembakoController) DeleteSembako(c *gin.Context) {
	sembakoID := c.Param("sembako_id")

	var count int64
	if err := sc.DB.Raw(`
		SELECT COUNT(*) FROM (
			SELECT sembako_id FROM detail_distribusi_sembako WHERE sembako_id = ?
			UNION ALL
			SELECT sembako_id FROM detail_penarikan_sembako WHERE sembako_id = ?
		) t`, sembakoID, sembakoID).Scan(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi data sembako"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Sembako tidak dapat dihapus karena sudah digunakan dalam transaksi."})
		return
	}

	if err := sc.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("sembako_id = ?", sembakoID).Delete(&models.StokSembako{}).Error; err != nil {
			return err
		}
		result := tx.Where("sembako_id = ?", sembakoID).Delete(&models.KatalogSembako{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	}); err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Sembako tidak ditemukan"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus sembako"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sembako berhasil dihapus"})
}

// POST /sembako/add-distribusi-bsu
func (sc *SembakoController) AddNewDistribusiSembakoBSU(c *gin.Context) {
	var req struct {
		DisbakoID  string `json:"disbako_id" binding:"required"`
		BSUID      string `json:"bsu_id" binding:"required"`
		AdminBSUID string `json:"admin_bsu_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Ambil dan validasi sesi distribusi yang dibuat oleh QRDistribusiSembako
	var disbako models.DistribusiSembakoBsu
	if err := sc.DB.Where("disbako_id = ?", req.DisbakoID).First(&disbako).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sesi distribusi tidak ditemukan"})
		return
	}
	if disbako.StatusDistribusi != models.DisbakoPending {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sesi distribusi sudah diproses atau tidak valid"})
		return
	}
	if disbako.BsuID != req.BSUID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU tidak sesuai dengan sesi distribusi"})
		return
	}

	bsiID := disbako.BsiID
	bsuID := req.BSUID
	disbakoID := req.DisbakoID

	// Load detail items yang sudah diinsert saat QR dibuat
	var details []models.DetailDistribusiSembako
	if err := sc.DB.Where("disbako_id = ?", disbakoID).Find(&details).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memuat detail distribusi"})
		return
	}
	if len(details) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tidak ada item pada sesi distribusi ini"})
		return
	}

	totalJenisSembako := len(details)

	err := sc.DB.Transaction(func(tx *gorm.DB) error {
		for _, detail := range details {
			// A. Lock & validasi stok BSI
			var stokBSI models.StokSembako
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("sembako_id = ? AND bank_id = ?", detail.SembakoID, bsiID).
				First(&stokBSI).Error; err != nil {
				return fmt.Errorf("stok BSI tidak ditemukan untuk sembako: %s", detail.SembakoID)
			}

			if stokBSI.Stok < detail.Item {
				return fmt.Errorf("stok sembako %s di BSI tidak mencukupi (sisa: %.2f)", detail.SembakoID, stokBSI.Stok)
			}

			stokBSISebelum := stokBSI.Stok
			stokBSISesudah := stokBSISebelum - detail.Item

			// B. Lock stok BSU, buat jika belum ada
			var stokBSU models.StokSembako
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("sembako_id = ? AND bank_id = ?", detail.SembakoID, bsuID).
				First(&stokBSU).Error

			var stokBSUSebelum float64

			if err == gorm.ErrRecordNotFound {
				stokBSU = models.StokSembako{
					SembakoID: detail.SembakoID,
					BankID:    bsuID,
					Stok:      0,
				}
				if err := tx.Create(&stokBSU).Error; err != nil {
					return fmt.Errorf("gagal membuat stok BSU: %v", err)
				}
			} else if err != nil {
				return err
			} else {
				stokBSUSebelum = stokBSU.Stok
			}

			stokBSUSesudah := stokBSUSebelum + detail.Item

			// C. Update stok kedua sisi
			if err := tx.Model(&models.StokSembako{}).
				Where("sembako_id = ? AND bank_id = ?", detail.SembakoID, bsiID).
				UpdateColumn("stok", stokBSISesudah).Error; err != nil {
				return fmt.Errorf("gagal update stok BSI: %v", err)
			}
			if err := tx.Model(&models.StokSembako{}).
				Where("sembako_id = ? AND bank_id = ?", detail.SembakoID, bsuID).
				UpdateColumn("stok", stokBSUSesudah).Error; err != nil {
				return fmt.Errorf("gagal update stok BSU: %v", err)
			}

			// D. Update stok fields di detail record
			if err := tx.Model(&models.DetailDistribusiSembako{}).
				Where("disbako_id = ? AND sembako_id = ?", disbakoID, detail.SembakoID).
				Updates(map[string]any{
					"stok_bsi_sebelum": stokBSISebelum,
					"stok_bsi_sesudah": stokBSISesudah,
					"stok_bsu_sebelum": stokBSUSebelum,
					"stok_bsu_sesudah": stokBSUSesudah,
				}).Error; err != nil {
				return fmt.Errorf("gagal update stok detail distribusi: %v", err)
			}

		}

		// E. Update header distribusi
		if err := tx.Model(&models.DistribusiSembakoBsu{}).
			Where("disbako_id = ?", disbakoID).
			Updates(map[string]any{
				"admin_bsu_id":      req.AdminBSUID,
				"status_distribusi": models.DisbakoSelesai,
			}).Error; err != nil {
			return fmt.Errorf("gagal mengupdate header distribusi: %v", err)
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Notifikasi fire-and-forget
	go func() {
		var bankBSI, bankBSU models.BankSampah
		if err := sc.DB.Where("bank_id = ?", bsiID).First(&bankBSI).Error; err != nil {
			return
		}
		if err := sc.DB.Where("bank_id = ?", bsuID).First(&bankBSU).Error; err != nil {
			return
		}

		refID := fmt.Sprintf("%s-to-%s", bsiID, bsuID)

		type AdminUser struct {
			UserID   string
			FCMToken string
		}
		var adminUsers []AdminUser
		if err := sc.DB.Table("admin").
			Select("users.user_id, users.fcm_token").
			Joins("JOIN users ON users.user_id = admin.user_id").
			Where("admin.bank_id = ? AND admin.status_admin = ?", bsuID, models.Aktif).
			Scan(&adminUsers).Error; err != nil {
			return
		}

		for _, au := range adminUsers {
			if err := sc.NotifSvc.NotifDistribusiSembakoBerhasil(
				context.Background(),
				au.UserID,
				au.FCMToken,
				bankBSU.NamaBank,
				totalJenisSembako,
				bankBSI.NamaBank,
				refID,
			); err != nil {
				fmt.Printf("[Notif] Gagal kirim notif distribusi sembako ke user %s: %v\n", au.UserID, err)
			}
		}
	}()

	c.JSON(http.StatusCreated, gin.H{"message": "Distribusi sembako ke BSU berhasil dilakukan"})
}

// POST /sembako/preview-distribusi-bsu/:bsi_id/:bsu_id
//
// Dry-run dari AddNewDistribusiSembakoBSU: validasi stok dan kembalikan
// simulasi hasil akhir tanpa menyimpan apapun ke database.
// Request body identik dengan AddNewDistribusiSembakoBSU.
func (sc *SembakoController) PreviewDistribusiSembakoBSU(c *gin.Context) {
	bsiID := c.Param("bsi_id")
	bsuID := c.Param("bsu_id")

	type ItemReq struct {
		SembakoIDBSI string  `json:"sembako_id" binding:"required"`
		StokKirim    float64 `json:"stok" binding:"required,gt=0"`
	}

	var req struct {
		Items []ItemReq `json:"items" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Items tidak boleh kosong"})
		return
	}

	// Validasi BSU & Parent
	var bsu models.BankSampah
	if err := sc.DB.Where("bank_id = ? AND jenis_bank = ?", bsuID, models.BSU).First(&bsu).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSU tidak ditemukan"})
		return
	}
	if bsu.ParentBankID == nil || *bsu.ParentBankID != bsiID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU ini bukan cabang dari BSI tersebut"})
		return
	}

	type previewItem struct {
		SembakoID      string  `json:"sembako_id"`
		NamaBarang     string  `json:"nama_barang"`
		PhotoURL       *string `json:"photo_url"`
		NilaiPoin      float64 `json:"nilai_poin"`
		StokKirim      float64 `json:"stok_kirim"`
		StokBSISebelum float64 `json:"stok_bsi_sebelum"`
		StokBSISesudah float64 `json:"stok_bsi_sesudah"`
		StokBSUSebelum float64 `json:"stok_bsu_sebelum"`
		StokBSUSesudah float64 `json:"stok_bsu_sesudah"`
		BSUItemBaru    bool    `json:"bsu_item_baru"`
	}

	var validationErrors []string
	var previews []previewItem

	for _, item := range req.Items {
		// A. Ambil katalog BSI beserta master sembako
		var katalogBSI models.KatalogSembako
		if err := sc.DB.Preload("MasterSembako").
			Where("sembako_id = ? AND bank_id = ?", item.SembakoIDBSI, bsiID).
			First(&katalogBSI).Error; err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("sembako_id '%s' tidak ditemukan di BSI", item.SembakoIDBSI))
			continue
		}

		// B. Ambil stok BSI
		var stokBSI models.StokSembako
		if err := sc.DB.Where("sembako_id = ? AND bank_id = ?", item.SembakoIDBSI, bsiID).
			First(&stokBSI).Error; err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf(
				"%s: stok BSI tidak ditemukan", katalogBSI.MasterSembako.NamaBarang,
			))
			continue
		}

		if stokBSI.Stok < item.StokKirim {
			validationErrors = append(validationErrors, fmt.Sprintf(
				"%s: stok BSI tidak mencukupi (tersedia: %.2f, diminta: %.2f)",
				katalogBSI.MasterSembako.NamaBarang, stokBSI.Stok, item.StokKirim,
			))
			continue
		}

		// C. Cek stok BSU — BSU tidak punya katalog, cukup cek StokSembako
		var stokBSU models.StokSembako
		err := sc.DB.Where("sembako_id = ? AND bank_id = ?", item.SembakoIDBSI, bsuID).
			First(&stokBSU).Error

		stokBSUSebelum := 0.0
		bsuItemBaru := false

		if err == gorm.ErrRecordNotFound {
			bsuItemBaru = true
		} else if err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf(
				"gagal mengecek stok BSU untuk %s: %s",
				katalogBSI.MasterSembako.NamaBarang, err.Error(),
			))
			continue
		} else {
			stokBSUSebelum = stokBSU.Stok
		}

		previews = append(previews, previewItem{
			SembakoID:      katalogBSI.SembakoID,
			NamaBarang:     katalogBSI.MasterSembako.NamaBarang,
			PhotoURL:       katalogBSI.PhotoURL,
			NilaiPoin:      katalogBSI.NilaiPoin,
			StokKirim:      item.StokKirim,
			StokBSISebelum: stokBSI.Stok,
			StokBSISesudah: stokBSI.Stok - item.StokKirim,
			StokBSUSebelum: stokBSUSebelum,
			StokBSUSesudah: stokBSUSebelum + item.StokKirim,
			BSUItemBaru:    bsuItemBaru,
		})
	}

	if len(validationErrors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Preview gagal: ada item yang tidak valid",
			"errors":  validationErrors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Preview distribusi berhasil",
		"data":    previews,
	})
}

// GET /sembako/list-distribusi/:bank_id
func (sc *SembakoController) ListDistribusiSembako(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	var filterCol string
	switch bank.JenisBank {
	case models.BSI:
		filterCol = "d.bsi_id"
	case models.BSU:
		filterCol = "d.bsu_id"
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": "Akses ditolak: jenis bank tidak diizinkan"})
		return
	}

	type distribusiRow struct {
		DisbakoID        string    `gorm:"column:disbako_id"        json:"disbako_id"`
		BsiID            string    `gorm:"column:bsi_id"            json:"bsi_id"`
		NamaBSI          string    `gorm:"column:nama_bsi"          json:"nama_bsi"`
		BsuID            string    `gorm:"column:bsu_id"            json:"bsu_id"`
		NamaBSU          string    `gorm:"column:nama_bsu"          json:"nama_bsu"`
		NamaAdminBSI     string    `gorm:"column:nama_admin_bsi"    json:"nama_admin_bsi"`
		NamaAdminBSU     string    `gorm:"column:nama_admin_bsu"    json:"nama_admin_bsu"`
		CreatedAt        time.Time `gorm:"column:created_at"        json:"created_at"`
		TotalItem        float64   `gorm:"column:total_item"        json:"total_item"`
		TotalPoin        float64   `gorm:"column:total_poin"        json:"total_poin"`
		StatusDistribusi string    `gorm:"column:status_distribusi" json:"status_distribusi"`
	}

	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	where := fmt.Sprintf("WHERE %s = ?", filterCol)
	args := []interface{}{bankID}

	if startDateStr != "" {
		if t, err := time.Parse("2006-01-02", startDateStr); err == nil {
			where += " AND d.created_at >= ?"
			args = append(args, t)
		}
	}
	if endDateStr != "" {
		if t, err := time.Parse("2006-01-02", endDateStr); err == nil {
			where += " AND d.created_at <= ?"
			args = append(args, t.Add(24*time.Hour-time.Second))
		}
	}

	query := fmt.Sprintf(`
		SELECT
			d.disbako_id,
			d.bsi_id,
			bsi.nama_bank                 AS nama_bsi,
			d.bsu_id,
			bsu.nama_bank                 AS nama_bsu,
			COALESCE(u_bsi.nama, '-')     AS nama_admin_bsi,
			COALESCE(u_bsu.nama, '-')     AS nama_admin_bsu,
			d.created_at,
			d.total_item,
			d.total_poin,
			d.status_distribusi
		FROM distribusi_sembako_bsu d
		LEFT JOIN bank_sampah bsi  ON bsi.bank_id  = d.bsi_id
		LEFT JOIN bank_sampah bsu  ON bsu.bank_id  = d.bsu_id
		LEFT JOIN admin a_bsi      ON a_bsi.admin_id = d.admin_bsi_id
		LEFT JOIN users u_bsi      ON u_bsi.user_id  = a_bsi.user_id
		LEFT JOIN admin a_bsu      ON a_bsu.admin_id = d.admin_bsu_id
		LEFT JOIN users u_bsu      ON u_bsu.user_id  = a_bsu.user_id
		%s
		ORDER BY d.created_at DESC
	`, where)

	var results []distribusiRow
	if err := sc.DB.Raw(query, args...).Scan(&results).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data distribusi"})
		return
	}

	if results == nil {
		results = []distribusiRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Data distribusi sembako berhasil diambil",
		"data":    results,
	})
}

// GET /sembako/detail-distribusi/:distribusi_id
func (sc *SembakoController) GetDetailDistribusiSembako(c *gin.Context) {
	distribusiID := c.Param("distribusi_id")

	// ── 1. Header distribusi ──────────────────────────────────────────────────
	type headerRow struct {
		DisbakoID        string    `gorm:"column:disbako_id"        json:"disbako_id"`
		BsiID            string    `gorm:"column:bsi_id"            json:"bsi_id"`
		NamaBSI          string    `gorm:"column:nama_bsi"          json:"nama_bsi"`
		BsuID            string    `gorm:"column:bsu_id"            json:"bsu_id"`
		NamaBSU          string    `gorm:"column:nama_bsu"          json:"nama_bsu"`
		NamaAdminBSI     string    `gorm:"column:nama_admin_bsi"    json:"nama_admin_bsi"`
		NamaAdminBSU     string    `gorm:"column:nama_admin_bsu"    json:"nama_admin_bsu"`
		CreatedAt        time.Time `gorm:"column:created_at"        json:"created_at"`
		TotalItem        float64   `gorm:"column:total_item"        json:"total_item"`
		TotalPoin        float64   `gorm:"column:total_poin"        json:"total_poin"`
		StatusDistribusi string    `gorm:"column:status_distribusi" json:"status_distribusi"`
	}

	var hdr headerRow
	if err := sc.DB.Raw(`
		SELECT
			d.disbako_id,
			d.bsi_id,
			bsi.nama_bank                 AS nama_bsi,
			d.bsu_id,
			bsu.nama_bank                 AS nama_bsu,
			COALESCE(u_bsi.nama, '-')     AS nama_admin_bsi,
			COALESCE(u_bsu.nama, '-')     AS nama_admin_bsu,
			d.created_at,
			d.total_item,
			d.total_poin,
			d.status_distribusi
		FROM distribusi_sembako_bsu d
		LEFT JOIN bank_sampah bsi  ON bsi.bank_id    = d.bsi_id
		LEFT JOIN bank_sampah bsu  ON bsu.bank_id    = d.bsu_id
		LEFT JOIN admin a_bsi      ON a_bsi.admin_id = d.admin_bsi_id
		LEFT JOIN users u_bsi      ON u_bsi.user_id  = a_bsi.user_id
		LEFT JOIN admin a_bsu      ON a_bsu.admin_id = d.admin_bsu_id
		LEFT JOIN users u_bsu      ON u_bsu.user_id  = a_bsu.user_id
		WHERE d.disbako_id = ?
	`, distribusiID).Scan(&hdr).Error; err != nil || hdr.DisbakoID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Distribusi tidak ditemukan"})
		return
	}

	// ── 2. Detail item ────────────────────────────────────────────────────────
	type itemRow struct {
		SembakoID      string  `gorm:"column:sembako_id"       json:"sembako_id"`
		NamaBarang     string  `gorm:"column:nama_barang"      json:"nama_barang"`
		PhotoURL       *string `gorm:"column:photo_url"        json:"photo_url"`
		NilaiPoin      float64 `gorm:"column:nilai_poin"       json:"nilai_poin"`
		Item           float64 `gorm:"column:item"             json:"item"`
		SubtotalPoin   float64 `gorm:"column:subtotal_poin"    json:"subtotal_poin"`
		StokBsiSebelum float64 `gorm:"column:stok_bsi_sebelum" json:"stok_bsi_sebelum"`
		StokBsiSesudah float64 `gorm:"column:stok_bsi_sesudah" json:"stok_bsi_sesudah"`
		StokBsuSebelum float64 `gorm:"column:stok_bsu_sebelum" json:"stok_bsu_sebelum"`
		StokBsuSesudah float64 `gorm:"column:stok_bsu_sesudah" json:"stok_bsu_sesudah"`
	}

	var items []itemRow
	if err := sc.DB.Raw(`
		SELECT
			dd.sembako_id,
			sm.nama_barang,
			ks.photo_url,
			ks.nilai_poin,
			dd.item,
			dd.subtotal_poin,
			dd.stok_bsi_sebelum,
			dd.stok_bsi_sesudah,
			dd.stok_bsu_sebelum,
			dd.stok_bsu_sesudah
		FROM detail_distribusi_sembako dd
		JOIN katalog_sembako ks ON ks.sembako_id = dd.sembako_id
		JOIN sembako sm         ON sm.barang_id  = ks.barang_id
		WHERE dd.disbako_id = ?
		ORDER BY sm.nama_barang ASC
	`, distribusiID).Scan(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail distribusi: " + err.Error()})
		return
	}

	if items == nil {
		items = []itemRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail distribusi sembako berhasil diambil",
		"data": gin.H{
			"header": hdr,
			"items":  items,
		},
	})
}

// POST /sembako/qr-distribusi
func (sc *SembakoController) QRDistribusiSembako(c *gin.Context) {
	type ItemReq struct {
		SembakoIDBSI string  `json:"sembako_id" binding:"required"`
		StokKirim    float64 `json:"stok" binding:"required,gt=0"`
	}

	var req struct {
		BSIID      string    `json:"bsi_id" binding:"required"`
		BSUID      string    `json:"bsu_id" binding:"required"`
		AdminBSIID string    `json:"admin_bsi_id" binding:"required"`
		Items      []ItemReq `json:"items" binding:"required,min=1"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var bsu models.BankSampah
	if err := sc.DB.Where("bank_id = ? AND jenis_bank = ?", req.BSUID, models.BSU).First(&bsu).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSU tidak ditemukan"})
		return
	}
	if bsu.ParentBankID == nil || *bsu.ParentBankID != req.BSIID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU ini bukan cabang dari BSI tersebut"})
		return
	}

	disbakoID := utils.GenerateBankRelatedID(req.BSIID)

	err := sc.DB.Transaction(func(tx *gorm.DB) error {
		var totalItem, totalPoin float64

		newDisbako := models.DistribusiSembakoBsu{
			DisbakoID:        disbakoID,
			BsiID:            req.BSIID,
			BsuID:            req.BSUID,
			AdminBsiID:       req.AdminBSIID,
			StatusDistribusi: models.DisbakoPending,
		}
		if err := tx.Create(&newDisbako).Error; err != nil {
			return fmt.Errorf("gagal membuat sesi distribusi: %v", err)
		}

		for _, item := range req.Items {
			var katalogBSI models.KatalogSembako
			if err := tx.Where("sembako_id = ? AND bank_id = ?", item.SembakoIDBSI, req.BSIID).
				First(&katalogBSI).Error; err != nil {
				return fmt.Errorf("sembako BSI tidak ditemukan: %s", item.SembakoIDBSI)
			}

			subtotalPoin := katalogBSI.NilaiPoin * item.StokKirim
			totalItem += item.StokKirim
			totalPoin += subtotalPoin

			detail := models.DetailDistribusiSembako{
				SembakoID:    item.SembakoIDBSI,
				DisbakoID:    disbakoID,
				Item:         item.StokKirim,
				Poin:         katalogBSI.NilaiPoin,
				SubtotalPoin: subtotalPoin,
			}
			if err := tx.Create(&detail).Error; err != nil {
				return fmt.Errorf("gagal menyimpan detail distribusi: %v", err)
			}
		}

		if err := tx.Model(&models.DistribusiSembakoBsu{}).
			Where("disbako_id = ?", disbakoID).
			Updates(map[string]any{
				"total_item": totalItem,
				"total_poin": totalPoin,
			}).Error; err != nil {
			return fmt.Errorf("gagal mengupdate total distribusi: %v", err)
		}

		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"disbako_id": disbakoID})
}
