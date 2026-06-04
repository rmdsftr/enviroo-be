package controllers

import (
	"context"
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"
	"fmt"
	"net/http"

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

// ─── AddNewSembako ───────────────────────────────────────────────────────────
// POST /sembako/add-sembako/:bank_id
func (sc *SembakoController) AddNewSembako(c *gin.Context) {
	bankID := c.Param("bank_id")

	var req struct {
		NamaSembako string  `form:"nama_sembako" binding:"required"`
		NilaiPoin   float64 `form:"nilai_poin" binding:"required"`
		Stok        float64 `form:"stok"`
		CreatedBy   string  `form:"created_by"`
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

	// FIX #3: Cek duplikasi nama secara case-insensitive
	var count int64
	sc.DB.Model(&models.KatalogSembako{}).
		Where("bank_id = ? AND LOWER(nama_sembako) = LOWER(?)", bankID, req.NamaSembako).
		Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Nama sembako sudah ada di bank ini"})
		return
	}

	newSembako := models.KatalogSembako{
		SembakoID:   utils.GenerateBankRelatedID(bankID),
		BankID:      &bankID,
		NamaSembako: req.NamaSembako,
		PhotoURL:    fotoURL,
		NilaiPoin:   req.NilaiPoin,
		Stok:        req.Stok,
		CreatedBy:   &req.CreatedBy,
		UpdatedBy:   &req.CreatedBy,
	}

	if err := sc.DB.Create(&newSembako).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan sembako: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Sembako berhasil ditambahkan",
		"data":    newSembako,
	})
}

// ─── GetSembakoBank ──────────────────────────────────────────────────────────
// GET /sembako/get-sembako/:bank_id
//
// BSI/BSM → ambil katalog_sembako milik bank itu sendiri.
// BSU     → ambil katalog_sembako milik BSU itu sendiri
//           (record BSU dibuat otomatis saat pertama kali distribusi dari BSI).
func (sc *SembakoController) GetSembakoBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank tidak ditemukan"})
		return
	}

	// Khusus BSU: pastikan punya induk BSI
	if bank.JenisBank == models.BSU && bank.ParentBankID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU ini tidak memiliki BSI induk"})
		return
	}

	// BSI, BSM, dan BSU semuanya ambil dari katalog milik bank itu sendiri.
	// created_by dan updated_by di-resolve menjadi nama user via join admin → users.
	var sembakoRows []models.KatalogSembako
	if err := sc.DB.Raw(`
		SELECT
			ks.sembako_id,
			ks.bank_id,
			ks.nama_sembako,
			ks.photo_url,
			ks.nilai_poin,
			ks.stok,
			ks.created_at,
			u_created.nama AS created_by,
			ks.updated_at,
			u_updated.nama AS updated_by
		FROM katalog_sembako ks
		LEFT JOIN admin adm_created ON adm_created.admin_id = ks.created_by
		LEFT JOIN users u_created   ON u_created.user_id    = adm_created.user_id
		LEFT JOIN admin adm_updated ON adm_updated.admin_id = ks.updated_by
		LEFT JOIN users u_updated   ON u_updated.user_id    = adm_updated.user_id
		WHERE ks.bank_id = ?
		ORDER BY ks.nama_sembako ASC
	`, bankID).Scan(&sembakoRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil katalog sembako: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Katalog sembako berhasil diambil",
		"data":    sembakoRows,
	})
}

// ─── GetDetailSembakoBSU ─────────────────────────────────────────────────────
// GET /sembako/detail-sembako-bsu/:sembako_id
//
// Mengembalikan info sembako BSU (dari katalog_sembako) beserta riwayat
// distribusi yang masuk ke BSU tersebut untuk item ini.
// :sembako_id adalah SembakoID dari katalog_sembako milik BSU (bukan BSI).
func (sc *SembakoController) GetDetailSembakoBSU(c *gin.Context) {
	sembakoID := c.Param("sembako_id")

	// 1. Ambil data sembako BSU
	var sembako models.KatalogSembako
	if err := sc.DB.Where("sembako_id = ?", sembakoID).First(&sembako).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sembako tidak ditemukan"})
		return
	}

	if sembako.BankID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Sembako tidak memiliki bank terkait"})
		return
	}

	var bank models.BankSampah
	if err := sc.DB.Where("bank_id = ?", *sembako.BankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank terkait tidak ditemukan"})
		return
	}
	if bank.JenisBank != models.BSU {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Endpoint ini hanya untuk sembako milik BSU"})
		return
	}

	// 2. Riwayat distribusi untuk item ini.
	// distribusi_sembako_bsu.sembako_id merujuk ke KatalogSembako BSI,
	// sinkronkan via nama_sembako + bsu_id agar tidak salah ambil item lain.
	type riwayatItem struct {
		DistribusiID      string  `json:"distribusi_id"`
		TanggalKirim      string  `json:"tanggal_kirim"`
		StokTerdistribusi float64 `json:"stok_terdistribusi"`
		NamaAdminBSI      string  `json:"nama_admin_bsi"`
		NamaAdminBSU      string  `json:"nama_admin_bsu"`
	}

	var riwayat []riwayatItem
	sc.DB.Raw(`
		SELECT
			d.distribusi_id,
			TO_CHAR(d.created_at, 'YYYY-MM-DD HH24:MI:SS') AS tanggal_kirim,
			d.stok_terdistribusi,
			u_bsi.nama                                      AS nama_admin_bsi,
			u_bsu.nama                                      AS nama_admin_bsu
		FROM distribusi_sembako_bsu d
		JOIN katalog_sembako ks
			ON ks.sembako_id = d.sembako_id
			AND ks.nama_sembako = ?
		-- Join untuk Admin BSI
		LEFT JOIN admin adm_bsi ON adm_bsi.admin_id = d.admin_bsi_id
		LEFT JOIN users u_bsi   ON u_bsi.user_id = adm_bsi.user_id
		-- Join untuk Admin BSU
		LEFT JOIN admin adm_bsu ON adm_bsu.admin_id = d.admin_bsu_id
		LEFT JOIN users u_bsu   ON u_bsu.user_id = adm_bsu.user_id
		WHERE d.bsu_id = ?
		ORDER BY d.created_at DESC
	`, sembako.NamaSembako, *sembako.BankID).Scan(&riwayat)

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail sembako BSU berhasil diambil",
		"data": gin.H{
			"sembako":            sembako,
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
		NamaSembako string  `form:"nama_sembako"`
		NilaiPoin   float64 `form:"nilai_poin"`
		Stok        float64 `form:"stok"`
		TambahStok  float64 `form:"tambah_stok"`
		UpdatedBy   string  `form:"updated_by"`
	}
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.NamaSembako != "" {
		sembako.NamaSembako = req.NamaSembako
	}

	// FIX #2: Gunakan PostForm agar nilai_poin bisa di-set ke 0
	if c.PostForm("nilai_poin") != "" {
		sembako.NilaiPoin = req.NilaiPoin
	}

	// Logic Stok:
	// 1. Jika ada 'tambah_stok', maka akumulatif (tambah ke yang sudah ada)
	if req.TambahStok != 0 {
		sembako.Stok += req.TambahStok
	}
	// 2. Jika ada 'stok' (absolut), maka timpa nilai yang ada (untuk koreksi)
	// Cek via PostForm untuk membedakan antara angka 0 beneran vs tidak dikirim
	if c.PostForm("stok") != "" {
		sembako.Stok = req.Stok
	}

	if req.UpdatedBy != "" {
		sembako.UpdatedBy = &req.UpdatedBy
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

	if err := sc.DB.Save(&sembako).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengupdate sembako: " + err.Error()})
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

	// FIX #5: Cek RowsAffected agar 404 kalau ID tidak ditemukan
	result := sc.DB.Where("sembako_id = ?", sembakoID).Delete(&models.KatalogSembako{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menghapus sembako: " + result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Sembako tidak ditemukan"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sembako berhasil dihapus"})
}

// POST /sembako/add-distribusi-bsu/:bsi_id/:bsu_id
func (sc *SembakoController) AddNewDistribusiSembakoBSU(c *gin.Context) {
	bsiID := c.Param("bsi_id")
	bsuID := c.Param("bsu_id")

	type ItemReq struct {
		SembakoIDBSI string  `json:"sembako_id" binding:"required"`
		StokKirim    float64 `json:"stok" binding:"required,gt=0"`
	}

	var req struct {
		AdminBSIID string    `json:"admin_bsi_id" binding:"required"`
		AdminBSUID string    `json:"admin_bsu_id" binding:"required"`
		Items      []ItemReq `json:"items" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// FIX #4: Validasi eksplisit bahwa items tidak kosong
	if len(req.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Items tidak boleh kosong"})
		return
	}

	// 1. Validasi BSU & Parent
	var bsu models.BankSampah
	if err := sc.DB.Where("bank_id = ? AND jenis_bank = ?", bsuID, models.BSU).First(&bsu).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "BSU tidak ditemukan"})
		return
	}

	if bsu.ParentBankID == nil || *bsu.ParentBankID != bsiID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "BSU ini bukan cabang dari BSI tersebut"})
		return
	}

	// 2. Jalankan Transaksi
	err := sc.DB.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			// A. Lock Stok BSI (Source)
			var sembakoBSI models.KatalogSembako
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("sembako_id = ? AND bank_id = ?", item.SembakoIDBSI, bsiID).
				First(&sembakoBSI).Error; err != nil {
				return fmt.Errorf("sembako BSI tidak ditemukan: %s", item.SembakoIDBSI)
			}

			if sembakoBSI.Stok < item.StokKirim {
				return fmt.Errorf("stok %s di BSI tidak mencukupi (sisa: %.2f)", sembakoBSI.NamaSembako, sembakoBSI.Stok)
			}

			stokBSI_sebelum := sembakoBSI.Stok
			stokBSI_sesudah := stokBSI_sebelum - item.StokKirim

			// B. Cari/Buat Sembako di BSU (Destination)
			// Cari berdasarkan Nama agar sinkron antara BSI & BSU
			var sembakoBSU models.KatalogSembako
			err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("bank_id = ? AND nama_sembako = ?", bsuID, sembakoBSI.NamaSembako).
				First(&sembakoBSU).Error

			// stokBSU_sebelum: 0 jika record baru, atau nilai dari DB jika sudah ada
			stokBSU_sebelum := 0.0

			if err == gorm.ErrRecordNotFound {
				// Belum ada, buat record baru di katalog BSU
				sembakoBSU = models.KatalogSembako{
					SembakoID:   utils.GenerateBankRelatedID(bsuID),
					BankID:      &bsuID,
					NamaSembako: sembakoBSI.NamaSembako,
					PhotoURL:    sembakoBSI.PhotoURL,
					NilaiPoin:   sembakoBSI.NilaiPoin,
					Stok:        0, // Start from 0, akan diupdate di step C
					CreatedBy:   &req.AdminBSIID,
				}
				if err := tx.Create(&sembakoBSU).Error; err != nil {
					return err
				}
				// stokBSU_sebelum tetap 0
			} else if err != nil {
				return err
			} else {
				// Record sudah ada: ambil stok saat ini sebagai stok sebelum
				stokBSU_sebelum = sembakoBSU.Stok
			}

			stokBSU_sesudah := stokBSU_sebelum + item.StokKirim

			// C. Update Stok di Kedua Sisi
			// FIX #1: Gunakan Where eksplisit agar GORM bisa track record dengan benar,
			// terutama untuk sembakoBSU yang baru saja di-Create dalam transaksi ini.
			if err := tx.Model(&sembakoBSI).
				Where("sembako_id = ?", sembakoBSI.SembakoID).
				Update("stok", stokBSI_sesudah).Error; err != nil {
				return err
			}
			// FIX #6: Sync nilai_poin BSU dengan BSI saat distribusi,
			// agar perubahan harga di BSI selalu tercermin ke BSU.
			if err := tx.Model(&sembakoBSU).
				Where("sembako_id = ?", sembakoBSU.SembakoID).
				Updates(map[string]interface{}{
					"stok":      stokBSU_sesudah,
					"nilai_poin": sembakoBSI.NilaiPoin,
				}).Error; err != nil {
				return err
			}

			// D. Simpan Log Distribusi
			logDistribusi := models.DistribusiSembakoBSU{
				DistribusiID:      utils.GenerateID("DST"),
				BSUID:             &bsuID,
				BSIID:             &bsiID,
				SembakoID:         &sembakoBSI.SembakoID, // Merujuk ke Sembako BSI
				AdminBSIID:        &req.AdminBSIID,
				AdminBSUID:        &req.AdminBSUID,
				StokTerdistribusi: item.StokKirim,
				StokBSUSebelum:    stokBSU_sebelum,
				StokBSUSesudah:    stokBSU_sesudah,
				StokBSISebelum:    stokBSI_sebelum,
				StokBSISesudah:    stokBSI_sesudah,
			}
			if err := tx.Create(&logDistribusi).Error; err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// ── Kirim notifikasi ke semua admin/petugas BSU (fire-and-forget) ─────────
	totalJenisSembako := len(req.Items)
	go func() {
		// Ambil nama BSI dan BSU
		var bankBSI, bankBSU models.BankSampah
		if err := sc.DB.Where("bank_id = ?", bsiID).First(&bankBSI).Error; err != nil {
			return
		}
		if err := sc.DB.Where("bank_id = ?", bsuID).First(&bankBSU).Error; err != nil {
			return
		}

		// Buat ref ID unik untuk grup notifikasi ini
		refID := fmt.Sprintf("%s-to-%s", bsiID, bsuID)

		// Ambil semua admin/petugas BSU yang aktif beserta fcm_token
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
		AdminBSIID string    `json:"admin_bsi_id" binding:"required"`
		Items      []ItemReq `json:"items" binding:"required"`
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
		SembakoID       string  `json:"sembako_id"`
		NamaSembako     string  `json:"nama_sembako"`
		PhotoURL        *string `json:"photo_url"`
		NilaiPoin       float64 `json:"nilai_poin"`
		StokKirim       float64 `json:"stok_kirim"`
		StokBSISebelum  float64 `json:"stok_bsi_sebelum"`
		StokBSISesudah  float64 `json:"stok_bsi_sesudah"`
		StokBSUSebelum  float64 `json:"stok_bsu_sebelum"`
		StokBSUSesudah  float64 `json:"stok_bsu_sesudah"`
		BSUItemBaru     bool    `json:"bsu_item_baru"` // true jika item belum ada di katalog BSU
	}
 
	var errors []string
	var previews []previewItem
 
	for _, item := range req.Items {
		// Ambil stok BSI
		var sembakoBSI models.KatalogSembako
		if err := sc.DB.Where("sembako_id = ? AND bank_id = ?", item.SembakoIDBSI, bsiID).
			First(&sembakoBSI).Error; err != nil {
			errors = append(errors, fmt.Sprintf("sembako_id '%s' tidak ditemukan di BSI", item.SembakoIDBSI))
			continue
		}
 
		if sembakoBSI.Stok < item.StokKirim {
			errors = append(errors, fmt.Sprintf(
				"%s: stok BSI tidak mencukupi (tersedia: %.2f, diminta: %.2f)",
				sembakoBSI.NamaSembako, sembakoBSI.Stok, item.StokKirim,
			))
			continue
		}
 
		// Cek apakah item sudah ada di katalog BSU
		var sembakoBSU models.KatalogSembako
		err := sc.DB.Where("bank_id = ? AND nama_sembako = ?", bsuID, sembakoBSI.NamaSembako).
			First(&sembakoBSU).Error
 
		stokBSUSebelum := 0.0
		bsuItemBaru := false
		if err == gorm.ErrRecordNotFound {
			bsuItemBaru = true
		} else if err != nil {
			errors = append(errors, fmt.Sprintf("gagal mengecek katalog BSU untuk %s: %s", sembakoBSI.NamaSembako, err.Error()))
			continue
		} else {
			stokBSUSebelum = sembakoBSU.Stok
		}
 
		previews = append(previews, previewItem{
			SembakoID:      sembakoBSI.SembakoID,
			NamaSembako:    sembakoBSI.NamaSembako,
			PhotoURL:       sembakoBSI.PhotoURL,
			NilaiPoin:      sembakoBSI.NilaiPoin,
			StokKirim:      item.StokKirim,
			StokBSISebelum: sembakoBSI.Stok,
			StokBSISesudah: sembakoBSI.Stok - item.StokKirim,
			StokBSUSebelum: stokBSUSebelum,
			StokBSUSesudah: stokBSUSebelum + item.StokKirim,
			BSUItemBaru:    bsuItemBaru,
		})
	}
 
	// Jika ada item yang gagal validasi, tolak semua — konsisten dengan perilaku transaksi
	if len(errors) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "Preview gagal: ada item yang tidak valid",
			"errors":  errors,
		})
		return
	}
 
	c.JSON(http.StatusOK, gin.H{
		"message": "Preview distribusi berhasil",
		"data":    previews,
	})
}