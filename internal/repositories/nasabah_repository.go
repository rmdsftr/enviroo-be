package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
)

type ProfilNasabahRow struct {
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

	SaldoRupiah float64 `json:"saldo_rupiah" gorm:"column:saldo_rupiah"`
	SaldoPoin   float64 `json:"saldo_poin" gorm:"column:saldo_poin"`
}

type DetailNasabahRow struct {
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

type NasabahRepo struct {
	db *gorm.DB
}

func NewNasabahRepo(db *gorm.DB) *NasabahRepo {
	return &NasabahRepo{db: db}
}

func (r *NasabahRepo) CountUser(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *NasabahRepo) CountUserByEmail(email string) (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count, err
}

func (r *NasabahRepo) FindUser(userID string) (*models.User, error) {
	var u models.User
	err := r.db.Where("user_id = ?", userID).First(&u).Error
	return &u, err
}

func (r *NasabahRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *NasabahRepo) CountNasabahAktif(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Nasabah{}).
		Where("user_id = ? AND status_nasabah = ?", userID, models.Aktif).
		Count(&count).Error
	return count, err
}

func (r *NasabahRepo) CountNasabahPendingDiBankLain(userID, bankID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Nasabah{}).
		Where("user_id = ? AND status_nasabah = ? AND bank_id != ?", userID, models.Pending, bankID).
		Count(&count).Error
	return count, err
}

func (r *NasabahRepo) FindNasabahByUserAndBank(userID, bankID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.db.Where("user_id = ? AND bank_id = ?", userID, bankID).First(&n).Error
	return &n, err
}

func (r *NasabahRepo) CreateUser(tx *gorm.DB, user *models.User) error {
	return tx.Create(user).Error
}

func (r *NasabahRepo) CreateNasabah(tx *gorm.DB, nasabah *models.Nasabah) error {
	return tx.Create(nasabah).Error
}

func (r *NasabahRepo) FindRewardByNama(tx *gorm.DB, nama models.RewardEnum) (*models.Reward, error) {
	var reward models.Reward
	err := tx.Where("nama_reward = ?", nama).First(&reward).Error
	return &reward, err
}

func (r *NasabahRepo) CreateSaldoRekening(tx *gorm.DB, saldo *models.SaldoRekening) error {
	return tx.Create(saldo).Error
}

func (r *NasabahRepo) CreateRiwayatSaldo(tx *gorm.DB, riwayat *models.RiwayatArusSaldo) error {
	return tx.Create(riwayat).Error
}

func (r *NasabahRepo) DeleteAktivasiNasabahBelumDipakai(tx *gorm.DB, userID string) error {
	return tx.Where("user_id = ? AND as_role = ? AND tujuan = ? AND is_used = ?",
		userID, models.RoleUserNasabah, models.TujuanAktivasi, false).
		Delete(&models.AktivasiAkun{}).Error
}

func (r *NasabahRepo) FindNasabah(nasabahID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.db.Where("nasabah_id = ?", nasabahID).First(&n).Error
	return &n, err
}

var tabelTransaksiNasabah = []string{
	"setoran_nasabah",
	"penarikan",
	"penerima_bagi_hasil",
	"tabungan_sampah",
}

func (r *NasabahRepo) CountTransaksiNasabah(nasabahID string) (int64, error) {
	for _, tabel := range tabelTransaksiNasabah {
		var count int64
		if err := r.db.Table(tabel).Where("nasabah_id = ?", nasabahID).Count(&count).Error; err != nil {
			return 0, err
		}
		if count > 0 {
			return count, nil
		}
	}
	return 0, nil
}

func (r *NasabahRepo) PluckRekeningIDs(tx *gorm.DB, nasabahID string) ([]string, error) {
	var rekeningIDs []string
	err := tx.Model(&models.SaldoRekening{}).
		Where("nasabah_id = ?", nasabahID).
		Pluck("rekening_id", &rekeningIDs).Error
	return rekeningIDs, err
}

func (r *NasabahRepo) DeleteRiwayatByRekening(tx *gorm.DB, rekeningIDs []string) error {
	if len(rekeningIDs) == 0 {
		return nil
	}
	return tx.Where("rekening_id IN ?", rekeningIDs).Delete(&models.RiwayatArusSaldo{}).Error
}

func (r *NasabahRepo) DeleteSaldoRekening(tx *gorm.DB, nasabahID string) error {
	return tx.Where("nasabah_id = ?", nasabahID).Delete(&models.SaldoRekening{}).Error
}

func (r *NasabahRepo) DeleteNasabah(tx *gorm.DB, nasabah *models.Nasabah) error {
	return tx.Delete(nasabah).Error
}

func (r *NasabahRepo) GetProfilNasabah(nasabahID string) (*ProfilNasabahRow, error) {
	var result ProfilNasabahRow
	err := r.db.Model(&models.Nasabah{}).
		Select(`
			nasabah.nasabah_id, nasabah.bank_id, nasabah.user_id, nasabah.joined_at, nasabah.status_nasabah, nasabah.nomor_rekening,
			users.nama, users.email, users.no_whatsapp, users.photo_url, users.created_at, users.updated_at,
			CASE WHEN b.jenis_bank = 'bsu' THEN parent.bank_id ELSE b.bank_id END as bsi_id,
			CASE WHEN b.jenis_bank = 'bsu' THEN parent.nama_bank ELSE b.nama_bank END as nama_bsi,
			CASE WHEN b.jenis_bank = 'bsu' THEN b.bank_id ELSE NULL END as bsu_id,
			CASE WHEN b.jenis_bank = 'bsu' THEN b.nama_bank ELSE NULL END as nama_bsu,
			COALESCE((SELECT SUM(sr.nominal_saldo) FROM saldo_rekening sr
				WHERE sr.nasabah_id = nasabah.nasabah_id AND sr.satuan_nominal_saldo = 'Rp'), 0) as saldo_rupiah,
			COALESCE((SELECT SUM(sr.nominal_saldo) FROM saldo_rekening sr
				WHERE sr.nasabah_id = nasabah.nasabah_id AND sr.satuan_nominal_saldo = 'poin'), 0) as saldo_poin
		`).
		Joins("JOIN users ON nasabah.user_id = users.user_id").
		Joins("LEFT JOIN bank_sampah b ON nasabah.bank_id = b.bank_id").
		Joins("LEFT JOIN bank_sampah parent ON b.parent_bank_id = parent.bank_id").
		Where("nasabah.nasabah_id = ?", nasabahID).
		First(&result).Error
	return &result, err
}

func (r *NasabahRepo) GetDetailNasabah(nasabahID string) (*DetailNasabahRow, error) {
	var result DetailNasabahRow
	err := r.db.Model(&models.Nasabah{}).
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
	return &result, err
}
