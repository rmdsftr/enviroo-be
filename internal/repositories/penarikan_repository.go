package repositories

import (
	"enviroo-be/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdminInfo menyimpan user_id dan fcm_token admin aktif (untuk notifikasi)
type AdminInfo struct {
	UserID   string
	FCMToken string
}

// PenarikanFilter adalah parameter filter untuk query daftar penarikan
type PenarikanFilter struct {
	Status    string `form:"status"`
	RewardID  *int   `form:"reward_id"`
	StartDate string `form:"start_date"`
	EndDate   string `form:"end_date"`
}

type PenarikanRepo struct {
	db *gorm.DB
}

func NewPenarikanRepo(db *gorm.DB) *PenarikanRepo {
	return &PenarikanRepo{db: db}
}

// ── Non-transactional reads ────────────────────────────────────────────────

func (r *PenarikanRepo) FindNasabahAktif(nasabahID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.db.Preload("Bank").
		Where("nasabah_id = ? AND status_nasabah = ?", nasabahID, models.Aktif).First(&n).Error
	return &n, err
}

func (r *PenarikanRepo) FindNasabahByUserID(userID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.db.Where("user_id = ? AND status_nasabah = ?", userID, models.Aktif).First(&n).Error
	return &n, err
}

func (r *PenarikanRepo) FindReward(rewardID int) (*models.Reward, error) {
	var rw models.Reward
	err := r.db.Where("reward_id = ?", rewardID).First(&rw).Error
	return &rw, err
}

func (r *PenarikanRepo) FindSaldoRekening(nasabahID string, rewardID int) (*models.SaldoRekening, error) {
	var s models.SaldoRekening
	err := r.db.Where("nasabah_id = ? AND reward_id = ?", nasabahID, rewardID).First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) FindSembako(sembakoID, bankID string) (*models.KatalogSembako, error) {
	var s models.KatalogSembako
	err := r.db.Preload("MasterSembako").
		Where("sembako_id = ? AND bank_id = ?", sembakoID, bankID).
		First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) FindStokSembako(sembakoID, bankID string) (*models.StokSembako, error) {
	var s models.StokSembako
	err := r.db.Where("sembako_id = ? AND bank_id = ?", sembakoID, bankID).First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) FindPenarikanByID(penarikanID string) (*models.Penarikan, error) {
	var p models.Penarikan
	err := r.db.
		Preload("Nasabah.User").
		Preload("Bank").
		Preload("Reward").
		Preload("DetailPenarikanSembako.KatalogSembako.MasterSembako").
		Where("penarikan_id = ?", penarikanID).
		First(&p).Error
	return &p, err
}

func (r *PenarikanRepo) FindByNasabahID(nasabahID string, f PenarikanFilter) ([]models.Penarikan, error) {
	q := r.penarikanBaseQuery().Where("penarikan.nasabah_id = ?", nasabahID)
	q = applyPenarikanFilter(q, f)

	var list []models.Penarikan
	err := q.Order("penarikan.created_at DESC").Find(&list).Error
	return list, err
}

func (r *PenarikanRepo) FindByBankID(bankID string, f PenarikanFilter) ([]models.Penarikan, error) {
	q := r.penarikanBaseQuery().Where("penarikan.bank_id = ?", bankID)
	q = applyPenarikanFilter(q, f)

	var list []models.Penarikan
	err := q.Order("penarikan.created_at DESC").Find(&list).Error
	return list, err
}

func (r *PenarikanRepo) FindAdminsByBankID(bankID string) ([]AdminInfo, error) {
	var admins []AdminInfo
	err := r.db.Table("admin").
		Select("users.user_id, users.fcm_token").
		Joins("JOIN users ON users.user_id = admin.user_id").
		Where("admin.bank_id = ? AND admin.status_admin = ?", bankID, models.Aktif).
		Scan(&admins).Error
	return admins, err
}

// ── Transactional (caller menyediakan tx dari db.Transaction) ─────────────

func (r *PenarikanRepo) LockSaldo(tx *gorm.DB, nasabahID string, rewardID int) (*models.SaldoRekening, error) {
	var s models.SaldoRekening
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("nasabah_id = ? AND reward_id = ?", nasabahID, rewardID).
		First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) LockSembako(tx *gorm.DB, sembakoID, bankID string) (*models.KatalogSembako, error) {
	var s models.KatalogSembako
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("MasterSembako").
		Where("sembako_id = ? AND bank_id = ?", sembakoID, bankID).
		First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) LockStok(tx *gorm.DB, sembakoID, bankID string) (*models.StokSembako, error) {
	var s models.StokSembako
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("sembako_id = ? AND bank_id = ?", sembakoID, bankID).
		First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) LockPenarikan(tx *gorm.DB, penarikanID string) (*models.Penarikan, error) {
	var p models.Penarikan
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Reward").
		Where("penarikan_id = ?", penarikanID).
		First(&p).Error
	return &p, err
}

func (r *PenarikanRepo) FindAdminTx(tx *gorm.DB, userID string) (*models.Admin, error) {
	var a models.Admin
	err := tx.Where("user_id = ?", userID).First(&a).Error
	return &a, err
}

func (r *PenarikanRepo) FindDetailSembako(tx *gorm.DB, penarikanID string) ([]models.DetailPenarikanSembako, error) {
	var details []models.DetailPenarikanSembako
	err := tx.Where("penarikan_id = ?", penarikanID).Find(&details).Error
	return details, err
}

func (r *PenarikanRepo) UpdateSaldo(tx *gorm.DB, saldo *models.SaldoRekening, nominal float64) error {
	return tx.Model(saldo).Update("nominal_saldo", nominal).Error
}

func (r *PenarikanRepo) CreateRiwayat(tx *gorm.DB, rw *models.RiwayatArusSaldo) error {
	return tx.Create(rw).Error
}

func (r *PenarikanRepo) SavePenarikan(tx *gorm.DB, p *models.Penarikan) error {
	return tx.Create(p).Error
}

func (r *PenarikanRepo) SaveDetailSembako(tx *gorm.DB, d *models.DetailPenarikanSembako) error {
	return tx.Create(d).Error
}

func (r *PenarikanRepo) UpdateStok(tx *gorm.DB, sembakoID, bankID string, delta float64) error {
	return tx.Model(&models.StokSembako{}).
		Where("sembako_id = ? AND bank_id = ?", sembakoID, bankID).
		Update("stok", gorm.Expr("stok + ?", delta)).Error
}

func (r *PenarikanRepo) UpdateStatus(tx *gorm.DB, p *models.Penarikan, updates map[string]interface{}) error {
	return tx.Model(p).Updates(updates).Error
}

// ── Helpers ────────────────────────────────────────────────────────────────

func (r *PenarikanRepo) penarikanBaseQuery() *gorm.DB {
	return r.db.Model(&models.Penarikan{}).
		Preload("Nasabah.User").
		Preload("Bank").
		Preload("Reward")
}

func applyPenarikanFilter(q *gorm.DB, f PenarikanFilter) *gorm.DB {
	if f.Status != "" {
		q = q.Where("penarikan.status_penarikan = ?", f.Status)
	}
	if f.RewardID != nil {
		q = q.Where("penarikan.reward_id = ?", *f.RewardID)
	}
	if f.StartDate != "" {
		if t, err := time.Parse("2006-01-02", f.StartDate); err == nil {
			q = q.Where("penarikan.created_at >= ?", t)
		}
	}
	if f.EndDate != "" {
		if t, err := time.Parse("2006-01-02", f.EndDate); err == nil {
			q = q.Where("penarikan.created_at <= ?", t.Add(24*time.Hour-time.Second))
		}
	}
	return q
}

