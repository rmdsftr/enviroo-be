package repositories

import (
	"errors"
	"fmt"
	"time"

	"enviroo-be/internal/models"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrBankMasihDipakai = errors.New("bank sampah masih memiliki data terkait")

type NasabahBankRow struct {
	NasabahID     string            `gorm:"column:nasabah_id" json:"nasabah_id"`
	Nama          string            `gorm:"column:nama" json:"nama"`
	Email         string            `gorm:"column:email" json:"email"`
	NIK           string            `gorm:"column:nik" json:"nik"`
	Foto          string            `gorm:"column:foto" json:"foto"`
	Status        models.StatusAkun `gorm:"column:status" json:"status"`
	NomorRekening string            `gorm:"column:nomor_rekening" json:"nomor_rekening"`
	TotalCount    int               `gorm:"column:total_count" json:"-"`
}

type BSIRow struct {
	models.BankSampah
	JumlahBSU     int64 `json:"jumlah_bsu" gorm:"column:jumlah_bsu"`
	JumlahNasabah int64 `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
	TotalCount    int   `json:"-" gorm:"column:total_count"`
}

type BSURow struct {
	BankID        string `json:"bank_id" gorm:"column:bank_id"`
	NamaBSU       string `json:"nama_bsu" gorm:"column:nama_bank"`
	PhotoURL      string `json:"photo_url" gorm:"column:photo_url"`
	IsActive      bool   `json:"is_active" gorm:"column:is_active"`
	NamaBankInduk string `json:"nama_bank_induk" gorm:"column:nama_bank_induk"`
	JumlahNasabah int64  `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
	JumlahStaff   int64  `json:"jumlah_staff" gorm:"column:jumlah_staff"`
	TotalCount    int    `json:"-" gorm:"column:total_count"`
}

type BSMRow struct {
	BankID        string `json:"bank_id" gorm:"column:bank_id"`
	NamaBSM       string `json:"nama_bsm" gorm:"column:nama_bank"`
	PhotoURL      string `json:"photo_url" gorm:"column:photo_url"`
	IsActive      bool   `json:"is_active" gorm:"column:is_active"`
	JumlahNasabah int64  `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
	JumlahStaff   int64  `json:"jumlah_staff" gorm:"column:jumlah_staff"`
	TotalCount    int    `json:"-" gorm:"column:total_count"`
}

type UnitBSURow struct {
	BankID        string `gorm:"column:bank_id" json:"BankID"`
	NamaBank      string `gorm:"column:nama_bank" json:"NamaBank"`
	PhotoURL      string `gorm:"column:photo_url" json:"PhotoURL"`
	JumlahNasabah int64  `gorm:"column:jumlah_nasabah" json:"jumlah_nasabah"`
	JumlahStaff   int64  `gorm:"column:jumlah_staff" json:"jumlah_staff"`
	IsActive      bool   `gorm:"column:is_active" json:"IsActive"`
	TotalCount    int    `gorm:"column:total_count" json:"-"`
}

type BSUSimpleRow struct {
	BankID        string `json:"bank_id" gorm:"column:bank_id"`
	NamaBSU       string `json:"nama_bsu" gorm:"column:nama_bank"`
	JumlahNasabah int64  `json:"jumlah_nasabah" gorm:"column:jumlah_nasabah"`
	JumlahStaff   int64  `json:"jumlah_staff" gorm:"column:jumlah_staff"`
	IsActive      bool   `json:"is_active" gorm:"column:is_active"`
}

type BankRepo struct {
	db *gorm.DB
}

func NewBankRepo(db *gorm.DB) *BankRepo {
	return &BankRepo{db: db}
}

func (r *BankRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var b models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&b).Error
	return &b, err
}

func (r *BankRepo) ListBankAktifWithWilayah() ([]models.BankSampah, error) {
	var banks []models.BankSampah
	err := r.db.
		Preload("Kecamatan").
		Preload("Kelurahan").
		Where("is_active = ?", true).
		Find(&banks).Error
	return banks, err
}

func (r *BankRepo) ListSemuaBank() ([]models.BankSampah, error) {
	var banks []models.BankSampah
	err := r.db.Find(&banks).Error
	return banks, err
}

func (r *BankRepo) ListNasabahByBank(bankID string, limit, offset int) ([]NasabahBankRow, error) {
	const baseSelect = `
		n.nasabah_id,
		u.nama,
		u.email,
		u.user_id AS nik,
		u.photo_url AS foto,
		n.status_nasabah AS status,
		n.nomor_rekening
		%s`

	const baseFrom = `
		FROM nasabah n
		JOIN users u ON u.user_id = n.user_id
		WHERE n.bank_id = ?
		ORDER BY u.nama ASC
		%s`

	var items []NasabahBankRow
	var err error

	if limit <= 0 {
		q := fmt.Sprintf("SELECT "+baseSelect+baseFrom, "", "")
		err = r.db.Raw(q, bankID).Scan(&items).Error
	} else {
		q := fmt.Sprintf("SELECT "+baseSelect+baseFrom, ", COUNT(*) OVER() AS total_count", "LIMIT ? OFFSET ?")
		err = r.db.Raw(q, bankID, limit, offset).Scan(&items).Error
	}

	return items, err
}

func (r *BankRepo) ListBSI(limit, offset int) ([]BSIRow, error) {
	const selectCols = "bank_sampah.*, " +
		"(SELECT COUNT(bank_id) FROM bank_sampah bsu WHERE bsu.parent_bank_id = bank_sampah.bank_id AND bsu.jenis_bank = 'bsu') AS jumlah_bsu, " +
		"(SELECT COUNT(nasabah_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah"

	var results []BSIRow
	query := r.db.Model(&models.BankSampah{}).Where("bank_sampah.jenis_bank = ?", models.BSI)

	if limit <= 0 {
		query = query.Select(selectCols)
	} else {
		query = query.Select(selectCols + ", COUNT(*) OVER() AS total_count").Limit(limit).Offset(offset)
	}

	err := query.Find(&results).Error
	return results, err
}

func (r *BankRepo) ListBSU(limit, offset int) ([]BSURow, error) {
	const selectCols = "bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.photo_url, bank_sampah.is_active, " +
		"(SELECT nama_bank FROM bank_sampah bsi WHERE bsi.bank_id = bank_sampah.parent_bank_id) AS nama_bank_induk, " +
		"(SELECT COUNT(user_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah, " +
		"(SELECT COUNT(user_id) FROM admin WHERE admin.bank_id = bank_sampah.bank_id) AS jumlah_staff"

	var results []BSURow
	query := r.db.Model(&models.BankSampah{}).Where("bank_sampah.jenis_bank = ?", models.BSU)

	if limit <= 0 {
		query = query.Select(selectCols)
	} else {
		query = query.Select(selectCols + ", COUNT(*) OVER() AS total_count").Limit(limit).Offset(offset)
	}

	err := query.Find(&results).Error
	return results, err
}

func (r *BankRepo) ListBSM(limit, offset int) ([]BSMRow, error) {
	const selectCols = "bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.photo_url, bank_sampah.is_active, " +
		"(SELECT COUNT(user_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah, " +
		"(SELECT COUNT(user_id) FROM admin WHERE admin.bank_id = bank_sampah.bank_id) AS jumlah_staff"

	var results []BSMRow
	query := r.db.Model(&models.BankSampah{}).Where("bank_sampah.jenis_bank = ?", models.BSM)

	if limit <= 0 {
		query = query.Select(selectCols)
	} else {
		query = query.Select(selectCols + ", COUNT(*) OVER() AS total_count").Limit(limit).Offset(offset)
	}

	err := query.Find(&results).Error
	return results, err
}

func (r *BankRepo) ListUnitByParent(parentBankID string, limit, offset int) ([]UnitBSURow, error) {
	const baseQuery = `
		SELECT
			b.bank_id,
			b.nama_bank,
			b.photo_url,
			b.is_active,
			(SELECT COUNT(*) FROM nasabah n WHERE n.bank_id = b.bank_id) AS jumlah_nasabah,
			(SELECT COUNT(*) FROM admin a WHERE a.bank_id = b.bank_id AND a.role IN ('admin_bsu', 'petugas_bsu')) AS jumlah_staff
			%s
		FROM bank_sampah b
		WHERE b.parent_bank_id = ? AND b.jenis_bank = 'bsu'
		ORDER BY b.nama_bank ASC
		%s`

	var results []UnitBSURow
	var err error

	if limit <= 0 {
		q := fmt.Sprintf(baseQuery, "", "")
		err = r.db.Raw(q, parentBankID).Scan(&results).Error
	} else {
		q := fmt.Sprintf(baseQuery, ", COUNT(*) OVER() AS total_count", "LIMIT ? OFFSET ?")
		err = r.db.Raw(q, parentBankID, limit, offset).Scan(&results).Error
	}

	return results, err
}

func (r *BankRepo) ListBSUByParent(parentBankID string) ([]BSUSimpleRow, error) {
	var results []BSUSimpleRow
	err := r.db.Model(&models.BankSampah{}).
		Select("bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.is_active, "+
			"(SELECT COUNT(user_id) FROM nasabah WHERE nasabah.bank_id = bank_sampah.bank_id) AS jumlah_nasabah, "+
			"(SELECT COUNT(user_id) FROM admin WHERE admin.bank_id = bank_sampah.bank_id) AS jumlah_staff").
		Where("bank_sampah.jenis_bank = ? AND bank_sampah.parent_bank_id = ?", models.BSU, parentBankID).
		Find(&results).Error
	return results, err
}

func (r *BankRepo) FindRewardByNama(tx *gorm.DB, nama models.RewardEnum) (*models.Reward, error) {
	var reward models.Reward
	err := tx.Where("nama_reward = ?", nama).First(&reward).Error
	return &reward, err
}

func (r *BankRepo) FindUserTx(tx *gorm.DB, userID string) (*models.User, error) {
	var u models.User
	err := tx.Where("user_id = ?", userID).First(&u).Error
	return &u, err
}

func (r *BankRepo) CountAdminAktifPending(tx *gorm.DB, userID string) (int64, error) {
	var count int64
	err := tx.Model(&models.Admin{}).
		Where("user_id = ? AND status_admin IN ?", userID, []models.StatusAkun{models.Aktif, models.Pending}).
		Count(&count).Error
	return count, err
}

func (r *BankRepo) CreateBank(tx *gorm.DB, b *models.BankSampah) error {
	return tx.Create(b).Error
}

func (r *BankRepo) SaveBank(tx *gorm.DB, b *models.BankSampah) error {
	return tx.Save(b).Error
}

func (r *BankRepo) CreateSaldoRekening(tx *gorm.DB, s *models.SaldoRekening) error {
	return tx.Create(s).Error
}

func (r *BankRepo) CreateAdmin(tx *gorm.DB, a *models.Admin) error {
	return tx.Create(a).Error
}

func (r *BankRepo) CreateHistoryAkunBank(tx *gorm.DB, h *models.HistoryAkunBank) error {
	return tx.Create(h).Error
}

type BankProfilRow struct {
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

type HistoryBankRow struct {
	HistoryBankID string         `json:"history_bank_id" gorm:"column:history_bank_id"`
	BankID        string         `json:"bank_id" gorm:"column:bank_id"`
	Action        string         `json:"action" gorm:"column:action"`
	OldValue      datatypes.JSON `json:"old_value" gorm:"column:old_value"`
	NewValue      datatypes.JSON `json:"new_value" gorm:"column:new_value"`
	Informasi     string         `json:"informasi" gorm:"column:informasi"`
	Keterangan    string         `json:"keterangan" gorm:"column:keterangan"`
	CreatedAt     time.Time      `json:"created_at" gorm:"column:created_at"`
	CreatedByName string         `json:"created_by_name" gorm:"column:created_by_name"`
}

type AdminBankRow struct {
	AdminID     string `json:"admin_id" gorm:"column:admin_id"`
	UserID      string `json:"user_id" gorm:"column:user_id"`
	Nama        string `json:"nama" gorm:"column:nama"`
	Foto        string `json:"foto" gorm:"column:photo_url"`
	Email       string `json:"email" gorm:"column:email"`
	Role        string `json:"role" gorm:"column:role"`
	StatusAdmin string `json:"status_admin" gorm:"column:status_admin"`
}

func (r *BankRepo) GetProfilBank(bankID string) (*BankProfilRow, error) {
	var result BankProfilRow
	err := r.db.Model(&models.BankSampah{}).
		Select("bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.jenis_bank, bank_sampah.photo_url, bank_sampah.deskripsi, bank_sampah.is_active, bank_sampah.provinsi, bank_sampah.kabupaten_kota, COALESCE(kec.kecamatan, '') AS kecamatan, COALESCE(kel.kelurahan, '') AS kelurahan, bank_sampah.alamat, bank_sampah.parent_bank_id, parent.nama_bank as bank_induk").
		Joins("LEFT JOIN bank_sampah as parent ON bank_sampah.parent_bank_id = parent.bank_id").
		Joins("LEFT JOIN kecamatan kec ON bank_sampah.id_kecamatan = kec.id_kecamatan").
		Joins("LEFT JOIN kelurahan kel ON bank_sampah.id_kelurahan = kel.id_kelurahan").
		Where("bank_sampah.bank_id = ?", bankID).
		First(&result).Error
	return &result, err
}

func (r *BankRepo) GetHistoryAkunBank(bankID string) ([]HistoryBankRow, error) {
	var result []HistoryBankRow
	err := r.db.Model(&models.HistoryAkunBank{}).
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
		Order("history_akun_bank.created_at desc").
		Find(&result).Error
	return result, err
}

func (r *BankRepo) FindBankWithRelasi(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.db.Preload("ParentBank").Preload("Kecamatan").Preload("Kelurahan").
		Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *BankRepo) ListAdminWithUserByBank(bankID string) ([]models.Admin, error) {
	var admins []models.Admin
	err := r.db.Preload("User").Where("bank_id = ?", bankID).Find(&admins).Error
	return admins, err
}

func (r *BankRepo) ListAdminBank(bankID string) ([]AdminBankRow, error) {
	var results []AdminBankRow
	err := r.db.Table("admin").
		Select("admin.admin_id, admin.user_id, u.nama, u.photo_url, u.email, admin.role, admin.status_admin").
		Joins("LEFT JOIN users u ON admin.user_id = u.user_id").
		Where("admin.bank_id = ?", bankID).
		Find(&results).Error
	return results, err
}

func (r *BankRepo) DeleteBank(bank *models.BankSampah) error {
	if err := r.db.Delete(bank).Error; err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return ErrBankMasihDipakai
		}
		return err
	}
	return nil
}

func (r *BankRepo) FindUser(userID string) (*models.User, error) {
	var u models.User
	err := r.db.Where("user_id = ?", userID).First(&u).Error
	return &u, err
}

func (r *BankRepo) FindAdminWithUserBank(adminID string) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.Preload("User").Preload("Bank").Where("admin_id = ?", adminID).First(&admin).Error
	return &admin, err
}

func (r *BankRepo) FindNamaAdmin(adminID string) string {
	var nama string
	r.db.Table("admin").
		Joins("JOIN users ON admin.user_id = users.user_id").
		Where("admin.admin_id = ?", adminID).
		Select("users.nama").
		Scan(&nama)
	return nama
}

func (r *BankRepo) FindAdminByUserAndBankTx(tx *gorm.DB, userID, bankID string) (*models.Admin, error) {
	var admin models.Admin
	err := tx.Where("user_id = ? AND bank_id = ?", userID, bankID).First(&admin).Error
	return &admin, err
}

func (r *BankRepo) FindAdminAktifByUser(userID string) (*models.Admin, error) {
	var admin models.Admin
	err := r.db.Where("user_id = ? AND status_admin = ?", userID, models.Aktif).First(&admin).Error
	return &admin, err
}

func (r *BankRepo) AdminPernahAktif(adminID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.AktivasiAkun{}).
		Where("admin_id = ? AND is_used = ?", adminID, true).
		Count(&count).Error
	return count > 0, err
}

func (r *BankRepo) DeleteAktivasiAdminBelumDipakai(tx *gorm.DB, userID string, tujuan *models.TujuanEnum) error {
	q := tx.Where("user_id = ? AND as_role = ? AND is_used = ?", userID, models.RoleUserAdmin, false)
	if tujuan != nil {
		q = q.Where("tujuan = ?", *tujuan)
	}
	return q.Delete(&models.AktivasiAkun{}).Error
}

func (r *BankRepo) DeleteAdmin(tx *gorm.DB, adminID string) error {
	return tx.Where("admin_id = ?", adminID).Delete(&models.Admin{}).Error
}

func (r *BankRepo) UpdateStatusAdmin(tx *gorm.DB, adminID string, status models.StatusAkun) error {
	return tx.Model(&models.Admin{}).Where("admin_id = ?", adminID).Update("status_admin", status).Error
}
