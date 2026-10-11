package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
)

type AuthRepo struct {
	db *gorm.DB
}

func NewAuthRepo(db *gorm.DB) *AuthRepo {
	return &AuthRepo{db: db}
}

func (r *AuthRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *AuthRepo) FindUserByEmail(email string) (*models.User, error) {
	var u models.User
	err := r.db.Where("email = ?", email).First(&u).Error
	return &u, err
}

func (r *AuthRepo) FindUserByID(tx *gorm.DB, userID string) (*models.User, error) {
	var u models.User
	err := r.conn(tx).Where("user_id = ?", userID).First(&u).Error
	return &u, err
}

func (r *AuthRepo) FindSessionUser(userID string) (*models.User, error) {
	var u models.User
	err := r.db.Select("device_session_id", "web_session_id").
		Where("user_id = ?", userID).
		First(&u).Error
	return &u, err
}

func (r *AuthRepo) UpdateFCMToken(userID, token string) error {
	return r.db.Model(&models.User{}).Where("user_id = ?", userID).
		Update("fcm_token", token).Error
}

func (r *AuthRepo) UpdatePassword(tx *gorm.DB, userID, hashedPassword string) error {
	return r.conn(tx).Model(&models.User{}).Where("user_id = ?", userID).
		Update("password", hashedPassword).Error
}

func (r *AuthRepo) SetSession(userID, platform, sessionID string) error {
	return r.db.Model(&models.User{}).
		Where("user_id = ?", userID).
		Update(models.SessionColumn(platform), sessionID).Error
}

func (r *AuthRepo) ClearSessionMatching(userID, platform, sessionID string) error {
	col := models.SessionColumn(platform)
	return r.db.Model(&models.User{}).
		Where("user_id = ? AND "+col+" = ?", userID, sessionID).
		Updates(map[string]interface{}{col: nil}).Error
}

func (r *AuthRepo) ClearAllSessions(tx *gorm.DB, userID string) error {
	return r.conn(tx).Model(&models.User{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"device_session_id": nil,
			"web_session_id":    nil,
		}).Error
}

func (r *AuthRepo) FindNasabahAktif(tx *gorm.DB, userID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.conn(tx).Where("user_id = ? AND status_nasabah = ?", userID, models.Aktif).First(&n).Error
	return &n, err
}

func (r *AuthRepo) FindAdminAktif(tx *gorm.DB, userID string) (*models.Admin, error) {
	var a models.Admin
	err := r.conn(tx).Where("user_id = ? AND status_admin = ?", userID, models.Aktif).First(&a).Error
	return &a, err
}

func (r *AuthRepo) FindNasabahPending(tx *gorm.DB, userID, nasabahID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.conn(tx).
		Where("user_id = ? AND nasabah_id = ? AND status_nasabah = ?", userID, nasabahID, models.Pending).
		First(&n).Error
	return &n, err
}

func (r *AuthRepo) FindAdminPending(tx *gorm.DB, userID, adminID string) (*models.Admin, error) {
	var a models.Admin
	err := r.conn(tx).
		Where("user_id = ? AND admin_id = ? AND status_admin = ?", userID, adminID, models.Pending).
		First(&a).Error
	return &a, err
}

func (r *AuthRepo) FindNasabahByID(tx *gorm.DB, nasabahID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.conn(tx).Where("nasabah_id = ?", nasabahID).First(&n).Error
	return &n, err
}

func (r *AuthRepo) FindNasabahByStatus(tx *gorm.DB, userID, nasabahID string, statuses []models.StatusAkun) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.conn(tx).
		Where("user_id = ? AND nasabah_id = ? AND status_nasabah IN ?", userID, nasabahID, statuses).
		First(&n).Error
	return &n, err
}

func (r *AuthRepo) FindAdminByIDRole(tx *gorm.DB, adminID string, role models.RoleAdmin) (*models.Admin, error) {
	var a models.Admin
	err := r.conn(tx).Where("admin_id = ? AND role = ?", adminID, role).First(&a).Error
	return &a, err
}

func (r *AuthRepo) FindAdminByStatus(tx *gorm.DB, userID, adminID string, statuses []models.StatusAkun) (*models.Admin, error) {
	var a models.Admin
	err := r.conn(tx).
		Where("user_id = ? AND admin_id = ? AND status_admin IN ?", userID, adminID, statuses).
		First(&a).Error
	return &a, err
}

func (r *AuthRepo) HasNasabahPending(userID string) bool {
	var n models.Nasabah
	return r.db.Where("user_id = ? AND status_nasabah = ?", userID, models.Pending).First(&n).Error == nil
}

func (r *AuthRepo) HasAdminPending(userID string) bool {
	var a models.Admin
	return r.db.Where("user_id = ? AND status_admin = ?", userID, models.Pending).First(&a).Error == nil
}

func (r *AuthRepo) CountNasabahAktif(tx *gorm.DB, userID string) (int64, error) {
	var count int64
	err := r.conn(tx).Model(&models.Nasabah{}).
		Where("user_id = ? AND status_nasabah = ?", userID, models.Aktif).
		Count(&count).Error
	return count, err
}

func (r *AuthRepo) CountAdminAktif(tx *gorm.DB, userID string) (int64, error) {
	var count int64
	err := r.conn(tx).Model(&models.Admin{}).
		Where("user_id = ? AND status_admin = ?", userID, models.Aktif).
		Count(&count).Error
	return count, err
}

func (r *AuthRepo) CountNasabahAktifSelain(tx *gorm.DB, userID, exceptNasabahID string) (int64, error) {
	var count int64
	err := r.conn(tx).Model(&models.Nasabah{}).
		Where("user_id = ? AND status_nasabah = ? AND nasabah_id != ?", userID, models.Aktif, exceptNasabahID).
		Count(&count).Error
	return count, err
}

func (r *AuthRepo) CountNasabahAktifAtauPendingSelain(tx *gorm.DB, userID, exceptNasabahID string) (int64, error) {
	var count int64
	err := r.conn(tx).Model(&models.Nasabah{}).
		Where("user_id = ? AND status_nasabah IN ? AND nasabah_id != ?",
			userID, []models.StatusAkun{models.Aktif, models.Pending}, exceptNasabahID).
		Count(&count).Error
	return count, err
}

func (r *AuthRepo) CountAdminAktifSelain(tx *gorm.DB, userID, exceptAdminID string) (int64, error) {
	var count int64
	err := r.conn(tx).Model(&models.Admin{}).
		Where("user_id = ? AND status_admin = ? AND admin_id != ?", userID, models.Aktif, exceptAdminID).
		Count(&count).Error
	return count, err
}

func (r *AuthRepo) UpdateStatusNasabah(tx *gorm.DB, nasabahID string, status models.StatusAkun) (int64, error) {
	res := r.conn(tx).Model(&models.Nasabah{}).
		Where("nasabah_id = ?", nasabahID).
		Update("status_nasabah", status)
	return res.RowsAffected, res.Error
}

func (r *AuthRepo) UpdateStatusAdmin(tx *gorm.DB, adminID string, status models.StatusAkun, refreshJoinedAt bool) (int64, error) {
	fields := map[string]interface{}{"status_admin": status}
	if refreshJoinedAt {
		fields["joined_at"] = time.Now()
	}
	res := r.conn(tx).Model(&models.Admin{}).
		Where("admin_id = ?", adminID).
		Updates(fields)
	return res.RowsAffected, res.Error
}

func (r *AuthRepo) IsBankAktif(bankID string) (bool, error) {
	var bank models.BankSampah
	err := r.db.Select("is_active").Where("bank_id = ?", bankID).First(&bank).Error
	return bank.IsActive, err
}

func (r *AuthRepo) FindBank(tx *gorm.DB, bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.conn(tx).Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *AuthRepo) FindAktivasiValidTerbaru(tx *gorm.DB, userID string, tujuan models.TujuanEnum) (*models.AktivasiAkun, error) {
	var aktivasi models.AktivasiAkun
	err := r.conn(tx).
		Where("user_id = ? AND is_used = ? AND expired_at > ? AND tujuan = ?", userID, false, time.Now(), tujuan).
		Order("created_at desc").
		First(&aktivasi).Error
	return &aktivasi, err
}

func (r *AuthRepo) ListAktivasiValid(tx *gorm.DB, userID string, tujuan models.TujuanEnum) ([]models.AktivasiAkun, error) {
	var aktivasi []models.AktivasiAkun
	err := r.conn(tx).
		Where("user_id = ? AND is_used = ? AND expired_at > ? AND tujuan = ?", userID, false, time.Now(), tujuan).
		Order("created_at desc").
		Find(&aktivasi).Error
	return aktivasi, err
}

func (r *AuthRepo) FindAktivasiValidByTarget(tx *gorm.DB, userID string, tujuan models.TujuanEnum, nasabahID, adminID *string) (*models.AktivasiAkun, error) {
	var aktivasi models.AktivasiAkun
	q := r.conn(tx).
		Where("user_id = ? AND is_used = ? AND expired_at > ? AND tujuan = ?", userID, false, time.Now(), tujuan)
	if nasabahID != nil {
		q = q.Where("nasabah_id = ?", *nasabahID)
	} else if adminID != nil {
		q = q.Where("admin_id = ?", *adminID)
	}
	err := q.Order("created_at desc").Limit(1).Find(&aktivasi).Error
	return &aktivasi, err
}

func (r *AuthRepo) CreateAktivasi(tx *gorm.DB, aktivasi *models.AktivasiAkun) error {
	return r.conn(tx).Create(aktivasi).Error
}

func (r *AuthRepo) UpdateAktivasi(tx *gorm.DB, aktivasiID string, fields map[string]interface{}) error {
	return r.conn(tx).Model(&models.AktivasiAkun{}).
		Where("aktivasi_id = ?", aktivasiID).
		Updates(fields).Error
}

func (r *AuthRepo) MarkAktivasiUsed(tx *gorm.DB, aktivasiID string) error {
	return r.conn(tx).Model(&models.AktivasiAkun{}).
		Where("aktivasi_id = ?", aktivasiID).
		Updates(map[string]interface{}{
			"is_used": true,
			"used_at": time.Now(),
		}).Error
}
