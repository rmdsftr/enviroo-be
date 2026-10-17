package repositories

import (
	"database/sql"
	"enviroo-be/internal/models"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AdminInfo struct {
	UserID   string
	FCMToken string
}

type RiwayatPenarikanRow struct {
	RiwayatPenarikanID string
	StatusPenarikan    models.StatusPenarikanEnum
	Catatan            *string
	CreatedByRaw       string
	CreatedByNama      *string
	CreatedAt          time.Time
}

type PenarikanListRow struct {
	PenarikanID      string                     `gorm:"column:penarikan_id"`
	CreatedAt        *time.Time                 `gorm:"column:created_at"`
	StatusPenarikan  models.StatusPenarikanEnum `gorm:"column:status_penarikan"`
	NominalPenarikan float64                    `gorm:"column:nominal_penarikan"`
	SatuanPenarikan  models.SatuanRewardEnum    `gorm:"column:satuan_penarikan"`
	RewardID         *int                       `gorm:"column:reward_id"`
	NamaReward       string                     `gorm:"column:nama_reward"`
}

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

func (r *PenarikanRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *PenarikanRepo) FindNasabah(tx *gorm.DB, nasabahID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.conn(tx).Where("nasabah_id = ?", nasabahID).First(&n).Error
	return &n, err
}

func (r *PenarikanRepo) FindNasabahWithUser(nasabahID string) (*models.Nasabah, error) {
	var n models.Nasabah
	err := r.db.Preload("User").Where("nasabah_id = ?", nasabahID).First(&n).Error
	return &n, err
}

func (r *PenarikanRepo) CountNasabahMilikUser(userID, nasabahID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Nasabah{}).
		Where("user_id = ? AND nasabah_id = ?", userID, nasabahID).
		Count(&count).Error
	return count, err
}

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

func (r *PenarikanRepo) FindProduk(produkID, bankID string) (*models.KatalogBarang, error) {
	var s models.KatalogBarang
	err := r.db.Preload("MasterBarang").
		Where("produk_id = ? AND bank_id = ?", produkID, bankID).
		First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) FindStokBarang(produkID, bankID string) (*models.StokBarang, error) {
	var s models.StokBarang
	err := r.db.Where("produk_id = ? AND bank_id = ?", produkID, bankID).First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) FindPenarikanByID(penarikanID string) (*models.Penarikan, error) {
	var p models.Penarikan
	err := r.db.
		Preload("Nasabah.User").
		Preload("Bank").
		Preload("Reward").
		Preload("DetailPenarikanBarang.KatalogBarang.MasterBarang").
		Where("penarikan_id = ?", penarikanID).
		First(&p).Error
	return &p, err
}

func (r *PenarikanRepo) FindRiwayatPenarikan(penarikanID string) ([]RiwayatPenarikanRow, error) {
	var rows []RiwayatPenarikanRow
	err := r.db.Table("riwayat_penarikan rp").
		Select("rp.riwayat_penarikan_id, rp.status_penarikan, rp.catatan, rp.created_by AS created_by_raw, u.nama AS created_by_nama, rp.created_at").
		Joins("LEFT JOIN users u ON u.user_id = rp.created_by").
		Where("rp.penarikan_id = ?", penarikanID).
		Order("rp.created_at ASC").
		Scan(&rows).Error
	return rows, err
}

func (r *PenarikanRepo) FindByNasabahID(nasabahID string, f PenarikanFilter) ([]PenarikanListRow, error) {
	q := r.db.Table("penarikan").
		Select(`penarikan.penarikan_id, rp_created.created_at, penarikan.status_penarikan,
			penarikan.nominal_penarikan, penarikan.satuan_penarikan, penarikan.reward_id,
			reward.nama_reward`).
		Joins(`LEFT JOIN riwayat_penarikan rp_created ON rp_created.penarikan_id = penarikan.penarikan_id AND rp_created.status_penarikan = 'pending'`).
		Joins(`LEFT JOIN reward ON reward.reward_id = penarikan.reward_id`).
		Where("penarikan.nasabah_id = ?", nasabahID)
	q = applyPenarikanFilter(q, f)

	var rows []PenarikanListRow
	err := q.Order("rp_created.created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *PenarikanRepo) FindByBankID(bankID string, f PenarikanFilter) ([]models.Penarikan, error) {
	q := r.penarikanBaseQuery().Where("penarikan.bank_id = ?", bankID)
	q = applyPenarikanFilter(q, f)

	var list []models.Penarikan
	err := q.Order("rp_created.created_at DESC").Find(&list).Error
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

func (r *PenarikanRepo) LockSaldo(tx *gorm.DB, nasabahID string, rewardID int) (*models.SaldoRekening, error) {
	var s models.SaldoRekening
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("nasabah_id = ? AND reward_id = ?", nasabahID, rewardID).
		First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) LockProduk(tx *gorm.DB, produkID, bankID string) (*models.KatalogBarang, error) {
	var s models.KatalogBarang
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("MasterBarang").
		Where("produk_id = ? AND bank_id = ?", produkID, bankID).
		First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) LockStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error) {
	var s models.StokBarang
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("produk_id = ? AND bank_id = ?", produkID, bankID).
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

func (r *PenarikanRepo) FindAdminAktif(tx *gorm.DB, userID string) (*models.Admin, error) {
	var a models.Admin
	err := r.conn(tx).Where("user_id = ? AND status_admin = ?", userID, models.Aktif).First(&a).Error
	return &a, err
}

func (r *PenarikanRepo) FindDetailBarang(tx *gorm.DB, penarikanID string) ([]models.DetailPenarikanBarang, error) {
	var details []models.DetailPenarikanBarang
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

func (r *PenarikanRepo) SaveRiwayatPenarikan(tx *gorm.DB, rw *models.RiwayatPenarikan) error {
	return tx.Create(rw).Error
}

func (r *PenarikanRepo) SaveDetailBarang(tx *gorm.DB, d *models.DetailPenarikanBarang) error {
	return tx.Create(d).Error
}

func (r *PenarikanRepo) UpdateStok(tx *gorm.DB, produkID, bankID string, delta float64) error {
	return tx.Model(&models.StokBarang{}).
		Where("produk_id = ? AND bank_id = ?", produkID, bankID).
		Update("stok", gorm.Expr("stok + ?", delta)).Error
}

func (r *PenarikanRepo) UpdateStatus(tx *gorm.DB, p *models.Penarikan, updates map[string]interface{}) error {
	return tx.Model(p).Updates(updates).Error
}

func (r *PenarikanRepo) FindSettingPenarikan(bankID string) (*models.SettingPenarikan, error) {
	var s models.SettingPenarikan
	err := r.db.Where("bank_id = ?", bankID).First(&s).Error
	return &s, err
}

func (r *PenarikanRepo) CreateSettingPenarikan(s *models.SettingPenarikan) error {
	return r.db.Create(s).Error
}

func (r *PenarikanRepo) SaveSettingPenarikan(s *models.SettingPenarikan) error {
	return r.db.Save(s).Error
}

func (r *PenarikanRepo) FindTanggalSetoranPertama(nasabahID string) (*time.Time, error) {
	var pertama sql.NullTime
	err := r.db.Raw(`SELECT MIN(created_at) FROM setoran_nasabah WHERE nasabah_id = ?`, nasabahID).
		Row().Scan(&pertama)
	if err != nil || !pertama.Valid {
		return nil, err
	}
	return &pertama.Time, nil
}

func (r *PenarikanRepo) PunyaSaldoAwal(nasabahID string) (bool, error) {
	var ada bool
	err := r.db.Raw(`
		SELECT EXISTS (
			SELECT 1
			FROM riwayat_arus_saldo ras
			JOIN saldo_rekening sr ON sr.rekening_id = ras.rekening_id
			WHERE sr.nasabah_id = ? AND ras.keterangan = ? AND ras.nominal_sesudah > 0
		)`, nasabahID, models.KeteranganSaldoAwal).
		Row().Scan(&ada)
	return ada, err
}

func (r *PenarikanRepo) penarikanBaseQuery() *gorm.DB {
	return r.db.Model(&models.Penarikan{}).
		Joins(`LEFT JOIN riwayat_penarikan rp_created ON rp_created.penarikan_id = penarikan.penarikan_id AND rp_created.status_penarikan = 'pending'`).
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
			q = q.Where("rp_created.created_at >= ?", t)
		}
	}
	if f.EndDate != "" {
		if t, err := time.Parse("2006-01-02", f.EndDate); err == nil {
			q = q.Where("rp_created.created_at <= ?", t.Add(24*time.Hour-time.Second))
		}
	}
	return q
}
