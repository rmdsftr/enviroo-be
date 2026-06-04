package controllers

import (
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProfilController struct {
	DB        *gorm.DB
	CFStorage *storage.CloudflareStorage
}

func NewProfilController(db *gorm.DB, cfStorage *storage.CloudflareStorage) *ProfilController {
	return &ProfilController{
		DB:        db,
		CFStorage: cfStorage,
	}
}

func (pc *ProfilController) GetProfilBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	type ProfilBankSampahResponse struct {
		BankID        string  `json:"bank_id" gorm:"column:bank_id"`
		NamaBank      string  `json:"nama_bank" gorm:"column:nama_bank"`
		JenisBank     string  `json:"jenis_bank" gorm:"column:jenis_bank"`
		Foto          string  `json:"foto" gorm:"column:photo_url"`
		Deskripsi     string  `json:"deskripsi" gorm:"column:deskripsi"`
		IsActive      bool    `json:"is_active" gorm:"column:is_active"`
		Provinsi      string  `json:"provinsi" gorm:"column:provinsi"`
		KabupatenKota string  `json:"kabupaten_kota" gorm:"column:kabupaten_kota"`
		Kecamatan     string  `json:"kecamatan" gorm:"column:kecamatan"`
		Kelurahan     string  `json:"kelurahan" gorm:"column:kelurahan"`
		Alamat        string  `json:"alamat_lengkap" gorm:"column:alamat"`
		ParentID      *string `json:"parent_id" gorm:"column:parent_bank_id"`
		BankInduk     *string `json:"bank_induk" gorm:"column:bank_induk"`
	}

	var result ProfilBankSampahResponse

	query := pc.DB.Model(&models.BankSampah{}).
		Select("bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.jenis_bank, bank_sampah.photo_url, bank_sampah.deskripsi, bank_sampah.is_active, bank_sampah.provinsi, bank_sampah.kabupaten_kota, COALESCE(kec.kecamatan, '') AS kecamatan, COALESCE(kel.kelurahan, '') AS kelurahan, bank_sampah.alamat, bank_sampah.parent_bank_id, parent.nama_bank as bank_induk").
		Joins("LEFT JOIN bank_sampah as parent ON bank_sampah.parent_bank_id = parent.bank_id").
		Joins("LEFT JOIN kecamatan kec ON bank_sampah.id_kecamatan = kec.id_kecamatan").
		Joins("LEFT JOIN kelurahan kel ON bank_sampah.id_kelurahan = kel.id_kelurahan").
		Where("bank_sampah.bank_id = ?", bankID)

	if err := query.First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank sampah profile: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah profile fetched successfully",
		"data":    result,
	})
}

func (pc *ProfilController) GetProfilNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	type ProfilNasabahResponse struct {
		NasabahID     string    `json:"nasabah_id" gorm:"column:nasabah_id"`
		BankID        string    `json:"bank_id" gorm:"column:bank_id"`
		UserID        string    `json:"user_id" gorm:"column:user_id"`
		JoinedAt      time.Time `json:"joined_at" gorm:"column:joined_at"`
		StatusNasabah string    `json:"status_nasabah" gorm:"column:status_nasabah"`
		NomorRekening string    `json:"nomor_rekening" gorm:"column:nomor_rekening"`

		Nama       string    `json:"nama" gorm:"column:nama"`
		Email      string    `json:"email" gorm:"column:email"`
		NoWhatsapp string    `json:"no_whatsapp" gorm:"column:no_whatsapp"`
		PhotoURL   string    `json:"foto" gorm:"column:photo_url"`
		CreatedAt  time.Time `json:"created_at" gorm:"column:created_at"`
		UpdatedAt  time.Time `json:"updated_at" gorm:"column:updated_at"`

		BsiID   *string `json:"bsi_id" gorm:"column:bsi_id"`
		NamaBsi *string `json:"nama_bsi" gorm:"column:nama_bsi"`
		BsuID   *string `json:"bsu_id" gorm:"column:bsu_id"`
		NamaBsu *string `json:"nama_bsu" gorm:"column:nama_bsu"`

		SaldoPoin float64 `json:"saldo_poin" gorm:"column:saldo_poin"`
	}

	var result ProfilNasabahResponse

	query := pc.DB.Model(&models.Nasabah{}).
		Select(`
			nasabah.nasabah_id, nasabah.bank_id, nasabah.user_id, nasabah.joined_at, nasabah.status_nasabah, nasabah.nomor_rekening,
			users.nama, users.email, users.no_whatsapp, users.photo_url, users.created_at, users.updated_at,
			CASE WHEN b.jenis_bank = 'bsu' THEN parent.bank_id ELSE b.bank_id END as bsi_id,
			CASE WHEN b.jenis_bank = 'bsu' THEN parent.nama_bank ELSE b.nama_bank END as nama_bsi,
			CASE WHEN b.jenis_bank = 'bsu' THEN b.bank_id ELSE NULL END as bsu_id,
			CASE WHEN b.jenis_bank = 'bsu' THEN b.nama_bank ELSE NULL END as nama_bsu,
			COALESCE((SELECT SUM(sr.nominal_saldo) FROM saldo_rekening sr WHERE sr.nasabah_id = nasabah.nasabah_id), 0) as saldo_poin
		`).
		Joins("JOIN users ON nasabah.user_id = users.user_id").
		Joins("LEFT JOIN bank_sampah b ON nasabah.bank_id = b.bank_id").
		Joins("LEFT JOIN bank_sampah parent ON b.parent_bank_id = parent.bank_id").
		Where("nasabah.nasabah_id = ?", nasabahID)

	if err := query.First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah profile: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah profile fetched successfully",
		"data":    result,
	})
}


func (pc *ProfilController) AktivasiNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var nasabah models.Nasabah
	if err := pc.DB.Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah: " + err.Error()})
		}
		return
	}

	// Toggle nilai status berdasarkan status saat ini
	currentStatus := nasabah.StatusNasabah
	var targetStatus models.StatusAkun

	// Jika statusnya nonaktif atau pending, maka diaktifkan. Selain itu (jika aktif), maka dinonaktifkan.
	if currentStatus == models.Nonaktif || currentStatus == models.Pending {
		targetStatus = models.Aktif
	} else {
		targetStatus = models.Nonaktif
	}

	// Sebelum mengaktifkan, pastikan user tidak punya akun nasabah aktif/pending di bank sampah lain
	if targetStatus == models.Aktif {
		var aktifOrPendingElsewhere int64
		if err := pc.DB.Model(&models.Nasabah{}).
			Where("user_id = ? AND status_nasabah IN ? AND nasabah_id != ?",
				nasabah.UserID,
				[]models.StatusAkun{models.Aktif, models.Pending},
				nasabahID).
			Count(&aktifOrPendingElsewhere).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memvalidasi data nasabah"})
			return
		}
		if aktifOrPendingElsewhere > 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "User sudah memiliki akun nasabah aktif atau sedang pending di bank sampah lain. Selesaikan atau batalkan proses tersebut terlebih dahulu."})
			return
		}
	}

	if err := pc.DB.Model(&nasabah).Update("status_nasabah", targetStatus).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update nasabah status: " + err.Error()})
		return
	}

	// Update secara lokal instance nasabah
	nasabah.StatusNasabah = targetStatus

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah status updated successfully",
		"data":    nasabah,
	})
}

func (pc *ProfilController) DeleteBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	var bank models.BankSampah
	if err := pc.DB.Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get bank sampah: " + err.Error()})
		}
		return
	}

	if err := pc.DB.Delete(&bank).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			c.JSON(http.StatusConflict, gin.H{"error": "Bank sampah tidak dapat dihapus karena masih memiliki data terkait"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete bank sampah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Bank sampah deleted successfully",
	})
}

func (pc *ProfilController) DeleteNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	var nasabah models.Nasabah
	if err := pc.DB.Where("nasabah_id = ?", nasabahID).First(&nasabah).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get nasabah: " + err.Error()})
		}
		return
	}

	if err := pc.DB.Delete(&nasabah).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete nasabah: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Nasabah deleted successfully",
	})
}

func (pc *ProfilController) GetProfilUser(c *gin.Context) {
	userID := c.Param("user_id")

	type ProfilUserResponse struct {
		models.User
		models.Admin
	}

	var result ProfilUserResponse
	// Gunakan Model(&models.User{}) agar GORM tahu tabel utamanya adalah 'users',
	// lalu gunakan LEFT JOIN agar user tetap ketemu meskipun dia bukan admin.
	if err := pc.DB.Model(&models.User{}).
		Select("users.*, admin.*").
		Joins("LEFT JOIN admin ON admin.user_id = users.user_id").
		Where("users.user_id = ?", userID).
		First(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "User tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User profile fetched successfully",
		"data":    result,
	})
}

func (pc *ProfilController) GetHistoryAkunBank(c *gin.Context) {
	bankID := c.Param("bank_id")

	type HistoryResponse struct {
		HistoryBankID uuid.UUID      `json:"history_bank_id" gorm:"column:history_bank_id"`
		BankID        string         `json:"bank_id" gorm:"column:bank_id"`
		Action        string         `json:"action" gorm:"column:action"`
		OldValue      datatypes.JSON `json:"old_value" gorm:"column:old_value"`
		NewValue      datatypes.JSON `json:"new_value" gorm:"column:new_value"`
		Informasi     string         `json:"informasi" gorm:"column:informasi"`
		Keterangan    string         `json:"keterangan" gorm:"column:keterangan"`
		CreatedAt     time.Time      `json:"created_at" gorm:"column:created_at"`
		CreatedByName string         `json:"created_by_name" gorm:"column:created_by_name"`
	}

	var result []HistoryResponse

	query := pc.DB.Model(&models.HistoryAkunBank{}).
		Select(`
			history_akun_bank.history_bank_id, 
			history_akun_bank.bank_id, 
			history_akun_bank.action, 
			history_akun_bank.old_value, 
			history_akun_bank.new_value, 
			history_akun_bank.informasi, 
			history_akun_bank.keterangan, 
			history_akun_bank.created_at, 
			users.nama as created_by_name
		`).
		Joins("LEFT JOIN admin ON history_akun_bank.created_by = admin.admin_id").
		Joins("LEFT JOIN users ON admin.user_id = users.user_id").
		Where("history_akun_bank.bank_id = ?", bankID).
		Order("history_akun_bank.created_at desc")

	if err := query.Find(&result).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get history: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "History bank fetched successfully",
		"data":    result,
	})
}

func (pc *ProfilController) GetDetailNasabah(c *gin.Context) {
	nasabahID := c.Param("nasabah_id")

	type DetailNasabahResponse struct {
		NasabahID     string    `json:"nasabah_id"     gorm:"column:nasabah_id"`
		UserID        string    `json:"user_id"        gorm:"column:user_id"`
		Nama          string    `json:"nama"           gorm:"column:nama"`
		Email         string    `json:"email"          gorm:"column:email"`
		NoWhatsapp    string    `json:"no_whatsapp"    gorm:"column:no_whatsapp"`
		PhotoURL      string    `json:"photo_url"      gorm:"column:photo_url"`
		StatusNasabah string    `json:"status_nasabah" gorm:"column:status_nasabah"`
		NomorRekening string    `json:"nomor_rekening" gorm:"column:nomor_rekening"`
		JoinedAt      time.Time `json:"joined_at"      gorm:"column:joined_at"`
		BankID        string    `json:"bank_id"        gorm:"column:bank_id"`
		NamaBank      string    `json:"nama_bank"      gorm:"column:nama_bank"`
		IsAdmin       bool      `json:"is_admin"       gorm:"column:is_admin"`
		AdminID       string    `json:"admin_id"       gorm:"column:admin_id"`
		RoleAdmin     string    `json:"role_admin"     gorm:"column:role_admin"`
		NamaBankAdmin string    `json:"nama_bank_admin" gorm:"column:nama_bank_admin"`
	}

	var result DetailNasabahResponse
	err := pc.DB.Model(&models.Nasabah{}).
		Select(`
			nasabah.nasabah_id, nasabah.user_id, nasabah.joined_at, nasabah.status_nasabah, nasabah.nomor_rekening, nasabah.bank_id,
			users.nama, users.email, users.no_whatsapp, users.photo_url,
			b.nama_bank as nama_bank,
			CASE WHEN a.admin_id IS NOT NULL THEN true ELSE false END as is_admin,
			COALESCE(a.admin_id, '') as admin_id,
			COALESCE(a.role::text, '') as role_admin,
			COALESCE(ba.nama_bank, '') as nama_bank_admin
		`).
		Joins("JOIN users ON nasabah.user_id = users.user_id").
		Joins("LEFT JOIN bank_sampah b ON nasabah.bank_id = b.bank_id").
		Joins("LEFT JOIN admin a ON nasabah.user_id = a.user_id AND a.status_admin = 'aktif'").
		Joins("LEFT JOIN bank_sampah ba ON a.bank_id = ba.bank_id").
		Where("nasabah.nasabah_id = ?", nasabahID).
		First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nasabah tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail nasabah: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail nasabah berhasil diambil",
		"data":    result,
	})
}

func (pc *ProfilController) DetailBankSampah(c *gin.Context) {
	bankID := c.Param("bank_id")
	if bankID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Bank ID wajib diisi"})
		return
	}

	type ListAdmin struct {
		AdminID   string `json:"admin_id" gorm:"column:admin_id"`
		NamaAdmin string `json:"nama_admin" gorm:"column:nama_admin"`
		PhotoURL  string `json:"photo_url" gorm:"column:photo_url"`
		RoleAdmin string `json:"role_admin" gorm:"column:role_admin"`
	}

	type DetailBankSampahResponse struct {
		BankID        string      `json:"bank_id" gorm:"column:bank_id"`
		NamaBank      string      `json:"nama_bank" gorm:"column:nama_bank"`
		PhotoURL      string      `json:"photo_url" gorm:"column:photo_url"`
		IsBSU         bool        `json:"is_bsu" gorm:"column:is_bsu"`
		BankIndukNama string      `json:"bank_induk_nama" gorm:"column:bank_induk_nama"`
		JenisBank     string      `json:"jenis_bank" gorm:"column:jenis_bank"`
		Alamat        string      `json:"alamat" gorm:"column:alamat"`
		Provinsi      string      `json:"provinsi" gorm:"column:provinsi"`
		KabupatenKota string      `json:"kabupaten_kota" gorm:"column:kabupaten_kota"`
		Kecamatan     string      `json:"kecamatan" gorm:"column:kecamatan"`
		Kelurahan     string      `json:"kelurahan" gorm:"column:kelurahan"`
		Deskripsi     string      `json:"deskripsi" gorm:"column:deskripsi"`
		IsActive      bool        `json:"is_active" gorm:"column:is_active"`
		JoinedAt      time.Time   `json:"joined_at" gorm:"column:joined_at"`
		Latitude		float64		`json:"latitude" gorm:"latitude"`
		Longitude		float64		`json:"longitude" gorm:"longitude"`
		Admins        []ListAdmin `json:"admins" gorm:"-"`
	}

	// Ambil data Bank Sampah
	var bank models.BankSampah
	if err := pc.DB.Preload("ParentBank").Preload("Kecamatan").Preload("Kelurahan").Where("bank_id = ?", bankID).First(&bank).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank Sampah tidak ditemukan"})
		return
	}

	// Ambil data Admin di Bank Sampah
	var admins []models.Admin
	if err := pc.DB.Preload("User").Where("bank_id = ?", bankID).Find(&admins).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data staff"})
		return
	}

	// Mapping Admin Role
	var listAdmins []ListAdmin
	for _, a := range admins {
		roleStr := string(a.Role)
		if a.Role == models.AdminBSI || a.Role == models.AdminBSU || a.Role == models.AdminBSM {
			roleStr = "Admin"
		} else if a.Role == models.PetugasBSI || a.Role == models.PetugasBSM || a.Role == models.PetugasBSU {
			roleStr = "Petugas"
		} else if a.Role == models.SuperAdmin {
			roleStr = "Superadmin"
		}

		listAdmins = append(listAdmins, ListAdmin{
			AdminID:   a.AdminID,
			NamaAdmin: a.User.Nama,
			PhotoURL:  a.User.PhotoURL,
			RoleAdmin: roleStr,
		})
	}

	// Mapping Jenis Bank
	jenisBankStr := string(bank.JenisBank)
	isBSU := false
	if bank.JenisBank == models.BSU {
		jenisBankStr = "Bank Sampah Unit (BSU)"
		isBSU = true
	} else if bank.JenisBank == models.BSM {
		jenisBankStr = "Bank Sampah Mandiri (BSM)"
	} else if bank.JenisBank == models.BSI {
		jenisBankStr = "Bank Sampah Induk (BSI)"
	}

	// Mapping Bank Induk
	bankIndukNama := ""
	if bank.ParentBank != nil {
		bankIndukNama = bank.ParentBank.NamaBank
	}

	kecamatanStr := ""
	if bank.Kecamatan != nil {
		kecamatanStr = bank.Kecamatan.Kecamatan
	}
	kelurahanStr := ""
	if bank.Kelurahan != nil {
		kelurahanStr = bank.Kelurahan.Kelurahan
	}

	response := DetailBankSampahResponse{
		BankID:        bank.BankID,
		NamaBank:      bank.NamaBank,
		PhotoURL:      bank.PhotoURL,
		IsBSU:         isBSU,
		BankIndukNama: bankIndukNama,
		JenisBank:     jenisBankStr,
		Alamat:        bank.Alamat,
		Provinsi:      bank.Provinsi,
		KabupatenKota: bank.KabupatenKota,
		Kecamatan:     kecamatanStr,
		Kelurahan:     kelurahanStr,
		Deskripsi:     bank.Deskripsi,
		IsActive:      bank.IsActive,
		JoinedAt:      bank.CreatedAt,
		Latitude : bank.Latitude,
		Longitude : bank.Longitude,
		Admins:        listAdmins,
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Berhasil mengambil detail bank sampah",
		"data":    response,
	})
}

func (pc *ProfilController) GetDetailPetugas(c *gin.Context) {
	petugasID := c.Param("petugas_id")
	if petugasID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Petugas ID wajib diisi"})
		return
	}

	type DetailPetugasResponse struct {
		PetugasID       string    `json:"petugas_id"         gorm:"column:petugas_id"`
		UserID          string    `json:"user_id"            gorm:"column:user_id"`
		Nama            string    `json:"nama"               gorm:"column:nama"`
		Email           string    `json:"email"              gorm:"column:email"`
		NoWhatsapp      string    `json:"no_whatsapp"        gorm:"column:no_whatsapp"`
		PhotoURL        string    `json:"photo_url"          gorm:"column:photo_url"`
		StatusPetugas   string    `json:"status_petugas"     gorm:"column:status_petugas"`
		RolePetugas     string    `json:"role_petugas"       gorm:"column:role_petugas"`
		JoinedAt        time.Time `json:"joined_at"          gorm:"column:joined_at"`
		BankID          string    `json:"bank_id"            gorm:"column:bank_id"`
		NamaBank        string    `json:"nama_bank"          gorm:"column:nama_bank"`
		IsNasabah       bool      `json:"is_nasabah"         gorm:"column:is_nasabah"`
		NasabahID       string    `json:"nasabah_id"         gorm:"column:nasabah_id"`
		NamaBankNasabah string    `json:"nama_bank_nasabah"  gorm:"column:nama_bank_nasabah"`
	}

	var result DetailPetugasResponse
	err := pc.DB.Model(&models.Admin{}).
		Select(`
			admin.admin_id as petugas_id, admin.user_id, admin.joined_at, admin.status_admin as status_petugas, admin.role as role_petugas, admin.bank_id,
			users.nama, users.email, users.no_whatsapp, users.photo_url,
			b.nama_bank as nama_bank,
			CASE WHEN n.nasabah_id IS NOT NULL THEN true ELSE false END as is_nasabah,
			COALESCE(n.nasabah_id, '') as nasabah_id,
			COALESCE(bn.nama_bank, '') as nama_bank_nasabah
		`).
		Joins("JOIN users ON admin.user_id = users.user_id").
		Joins("LEFT JOIN bank_sampah b ON admin.bank_id = b.bank_id").
		Joins("LEFT JOIN nasabah n ON admin.user_id = n.user_id AND n.status_nasabah = 'aktif'").
		Joins("LEFT JOIN bank_sampah bn ON n.bank_id = bn.bank_id").
		Where("admin.admin_id = ?", petugasID).
		First(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Petugas tidak ditemukan"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail petugas: " + err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Detail petugas berhasil diambil",
		"data":    result,
	})
}
