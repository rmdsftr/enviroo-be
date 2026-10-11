package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ProfilUserRow struct {
	UserID     string    `json:"user_id"     gorm:"column:user_id"`
	Nama       string    `json:"nama"        gorm:"column:nama"`
	Email      string    `json:"email"       gorm:"column:email"`
	NoWhatsapp string    `json:"no_whatsapp" gorm:"column:no_whatsapp"`
	PhotoURL   string    `json:"photo_url"   gorm:"column:photo_url"`
	CreatedAt  time.Time `json:"created_at"  gorm:"column:created_at"`
	UpdatedAt  time.Time `json:"updated_at"  gorm:"column:updated_at"`

	IsAdmin     bool       `json:"is_admin"     gorm:"column:is_admin"`
	AdminID     string     `json:"admin_id"     gorm:"column:admin_id"`
	BankID      string     `json:"bank_id"      gorm:"column:bank_id"`
	NamaBank    string     `json:"nama_bank"    gorm:"column:nama_bank"`
	RoleAdmin   string     `json:"role_admin"   gorm:"column:role_admin"`
	StatusAdmin string     `json:"status_admin" gorm:"column:status_admin"`
	JoinedAt    *time.Time `json:"joined_at"    gorm:"column:joined_at"`
}

type DetailPetugasRow struct {
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

type UserRepo struct {
	db *gorm.DB
}

func NewUserRepo(db *gorm.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetProfilUser(userID string) (*ProfilUserRow, error) {
	var result ProfilUserRow
	err := r.db.Model(&models.User{}).
		Select(`
			users.user_id, users.nama, users.email, users.no_whatsapp, users.photo_url,
			users.created_at, users.updated_at,
			CASE WHEN a.admin_id IS NOT NULL THEN true ELSE false END as is_admin,
			COALESCE(a.admin_id, '') as admin_id,
			COALESCE(a.bank_id, '') as bank_id,
			COALESCE(b.nama_bank, '') as nama_bank,
			COALESCE(a.role::text, '') as role_admin,
			COALESCE(a.status_admin::text, '') as status_admin,
			a.joined_at
		`).
		Joins("LEFT JOIN admin a ON a.user_id = users.user_id").
		Joins("LEFT JOIN bank_sampah b ON a.bank_id = b.bank_id").
		Where("users.user_id = ?", userID).
		Order("CASE WHEN a.status_admin = 'aktif' THEN 0 ELSE 1 END").
		First(&result).Error
	return &result, err
}

func (r *UserRepo) GetDetailPetugas(petugasID string) (*DetailPetugasRow, error) {
	var result DetailPetugasRow
	err := r.db.Model(&models.Admin{}).
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
	return &result, err
}

type UserRow struct {
	UserID     string    `json:"user_id"     gorm:"column:user_id"`
	Nama       string    `json:"nama"        gorm:"column:nama"`
	Email      string    `json:"email"       gorm:"column:email"`
	NoWhatsapp string    `json:"no_whatsapp" gorm:"column:no_whatsapp"`
	PhotoURL   string    `json:"photo_url"   gorm:"column:photo_url"`
	CreatedAt  time.Time `json:"created_at"  gorm:"column:created_at"`
	UpdatedAt  time.Time `json:"updated_at"  gorm:"column:updated_at"`
}

type UserRingkasRow struct {
	UserID   string `json:"user_id"   gorm:"column:user_id"`
	Nama     string `json:"nama"      gorm:"column:nama"`
	PhotoURL string `json:"photo_url" gorm:"column:photo_url"`
}

type ActiveAdminRow struct {
	BankID    string           `json:"bank_id"    gorm:"column:bank_id"`
	NamaBank  string           `json:"nama_bank"  gorm:"column:nama_bank"`
	JenisBank models.JenisBank `json:"jenis_bank" gorm:"column:jenis_bank"`
	PhotoURL  string           `json:"photo_url"  gorm:"column:photo_url"`
}

type ActivePetugasRow struct {
	AdminID     string `json:"admin_id"     gorm:"column:admin_id"`
	Role        string `json:"role"         gorm:"column:role"`
	StatusAdmin string `json:"status_admin" gorm:"column:status_admin"`
	UserID      string `json:"user_id"      gorm:"column:user_id"`
	Nama        string `json:"nama"         gorm:"column:nama"`
	Email       string `json:"email"        gorm:"column:email"`
	PhotoURL    string `json:"photo_url"    gorm:"column:photo_url"`
	BankID      string `json:"bank_id"      gorm:"column:bank_id"`
	NamaBank    string `json:"nama_bank"    gorm:"column:nama_bank"`
	JenisBank   string `json:"jenis_bank"   gorm:"column:jenis_bank"`
}

type SuperadminRow struct {
	AdminID     string    `json:"admin_id"     gorm:"column:admin_id"`
	UserID      string    `json:"user_id"      gorm:"column:user_id"`
	Nama        string    `json:"nama"         gorm:"column:nama"`
	Email       string    `json:"email"        gorm:"column:email"`
	NoWhatsapp  string    `json:"no_whatsapp"  gorm:"column:no_whatsapp"`
	PhotoURL    string    `json:"photo_url"    gorm:"column:photo_url"`
	StatusAdmin string    `json:"status_admin" gorm:"column:status_admin"`
	JoinedAt    time.Time `json:"joined_at"    gorm:"column:joined_at"`
}

func (r *UserRepo) FindUser(userID string) (*models.User, error) {
	var u models.User
	err := r.db.Where("user_id = ?", userID).First(&u).Error
	return &u, err
}

func (r *UserRepo) FindUserRow(userID string) (*UserRow, error) {
	var row UserRow
	err := r.db.Model(&models.User{}).
		Select("user_id, nama, email, no_whatsapp, photo_url, created_at, updated_at").
		Where("user_id = ?", userID).
		First(&row).Error
	return &row, err
}

func (r *UserRepo) FindUserRingkas(userID string) (*UserRingkasRow, error) {
	var row UserRingkasRow
	err := r.db.Table("users").
		Select("user_id, nama, photo_url").
		Where("user_id = ?", userID).
		First(&row).Error
	return &row, err
}

func (r *UserRepo) CountUserByID(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *UserRepo) CountUserByEmail(email string) (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error
	return count, err
}

func (r *UserRepo) CountUser() (int64, error) {
	var count int64
	err := r.db.Model(&models.User{}).Count(&count).Error
	return count, err
}

func (r *UserRepo) ListUserBukanAdmin() ([]UserRow, error) {
	sub := r.db.Model(&models.Admin{}).Select("user_id").
		Where("status_admin IN ?", []models.StatusAkun{models.Aktif, models.Pending})

	var rows []UserRow
	err := r.db.Model(&models.User{}).
		Select("user_id, nama, email, no_whatsapp, photo_url, created_at, updated_at").
		Where("user_id NOT IN (?)", sub).
		Find(&rows).Error
	return rows, err
}

func (r *UserRepo) ListUserBukanNasabah() ([]UserRow, error) {
	sub := r.db.Model(&models.Nasabah{}).Select("user_id").
		Where("status_nasabah IN ?", []models.StatusAkun{models.Aktif, models.Pending})

	var rows []UserRow
	err := r.db.Model(&models.User{}).
		Select("user_id, nama, email, no_whatsapp, photo_url, created_at, updated_at").
		Where("user_id NOT IN (?)", sub).
		Find(&rows).Error
	return rows, err
}

func (r *UserRepo) ListUserWithAkun(limit, offset int) ([]models.User, error) {
	var users []models.User
	err := r.db.Preload("Nasabahs").Preload("Admins").
		Offset(offset).Limit(limit).
		Find(&users).Error
	return users, err
}

func (r *UserRepo) FindUserWithAkun(userID string) (*models.User, error) {
	var user models.User
	err := r.db.
		Preload("Nasabahs.Bank").
		Preload("Admins.Bank").
		Where("user_id = ?", userID).
		First(&user).Error
	return &user, err
}

func (r *UserRepo) ListNasabahWithBank(userID string) ([]models.Nasabah, error) {
	var list []models.Nasabah
	err := r.db.Preload("Bank").Where("user_id = ?", userID).
		Order("joined_at DESC").Find(&list).Error
	return list, err
}

func (r *UserRepo) ListAdminWithBank(userID string) ([]models.Admin, error) {
	var list []models.Admin
	err := r.db.Preload("Bank").Where("user_id = ?", userID).
		Order("joined_at DESC").Find(&list).Error
	return list, err
}

func (r *UserRepo) CountNasabahByUser(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Nasabah{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *UserRepo) CountAdminByUser(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Admin{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *UserRepo) CreateUser(tx *gorm.DB, user *models.User) error {
	if tx == nil {
		return r.db.Create(user).Error
	}
	return tx.Omit(clause.Associations).Create(user).Error
}

func (r *UserRepo) UpdateUser(userID string, fields map[string]interface{}) error {
	return r.db.Model(&models.User{}).Where("user_id = ?", userID).Updates(fields).Error
}

func (r *UserRepo) DeleteUser(user *models.User) error {
	return r.db.Delete(user).Error
}

func (r *UserRepo) UpdateFCMToken(userID, token string) error {
	return r.db.Model(&models.User{}).Where("user_id = ?", userID).
		Update("fcm_token", token).Error
}

func (r *UserRepo) CreateAdmin(tx *gorm.DB, admin *models.Admin) error {
	return tx.Omit(clause.Associations).Create(admin).Error
}

func (r *UserRepo) FindActiveAdmin(adminID string) (*ActiveAdminRow, error) {
	var row ActiveAdminRow
	err := r.db.Model(&models.Admin{}).
		Select("admin.bank_id", "bank_sampah.nama_bank", "bank_sampah.jenis_bank", "bank_sampah.photo_url").
		Joins("JOIN bank_sampah ON admin.bank_id = bank_sampah.bank_id").
		Where("admin.admin_id = ?", adminID).
		Scan(&row).Error
	return &row, err
}

func (r *UserRepo) FindActivePetugas(adminID string) (*ActivePetugasRow, error) {
	var row ActivePetugasRow
	err := r.db.Table("admin").
		Select("admin.admin_id, admin.role, admin.status_admin, users.user_id, users.nama, users.email, users.photo_url, bank_sampah.bank_id, bank_sampah.nama_bank, bank_sampah.jenis_bank").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Joins("LEFT JOIN bank_sampah ON bank_sampah.bank_id = admin.bank_id").
		Where("admin.admin_id = ?", adminID).
		Scan(&row).Error
	return &row, err
}

func (r *UserRepo) ListSuperadmin() ([]SuperadminRow, error) {
	var list []SuperadminRow
	err := r.db.Table("admin").
		Select("admin.admin_id, admin.status_admin, admin.joined_at, users.user_id, users.nama, users.email, users.no_whatsapp, users.photo_url").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.role = ?", models.SuperAdmin).
		Order("admin.joined_at ASC").
		Scan(&list).Error
	return list, err
}
