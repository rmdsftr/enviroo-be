package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SaldoPerSatuan struct {
	Uang float64
	Poin float64
}

type NasabahSaldoRow struct {
	NasabahID     string            `gorm:"column:nasabah_id"`
	NamaNasabah   string            `gorm:"column:nama_nasabah"`
	StatusNasabah models.StatusAkun `gorm:"column:status_nasabah"`
	SaldoUang     float64           `gorm:"column:saldo_uang"`
	SaldoPoin     float64           `gorm:"column:saldo_poin"`
}

type DashboardRepo struct {
	db *gorm.DB
}

func NewDashboardRepo(db *gorm.DB) *DashboardRepo {
	return &DashboardRepo{db: db}
}

func (r *DashboardRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *DashboardRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *DashboardRepo) FindNasabah(nasabahID string) (*models.Nasabah, error) {
	var nasabah models.Nasabah
	err := r.db.Where("nasabah_id = ?", nasabahID).First(&nasabah).Error
	return &nasabah, err
}

func (r *DashboardRepo) CountNasabahAktif(bankID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Nasabah{}).
		Where("bank_id = ? AND status_nasabah = ?", bankID, models.Aktif).
		Count(&count).Error
	return count, err
}

func (r *DashboardRepo) CountAdminAktif(bankID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Admin{}).
		Where("bank_id = ? AND status_admin = ?", bankID, models.Aktif).
		Count(&count).Error
	return count, err
}

func (r *DashboardRepo) CountBSU(bsiID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.BankSampah{}).
		Where("parent_bank_id = ? AND jenis_bank = ?", bsiID, models.BSU).
		Count(&count).Error
	return count, err
}

func (r *DashboardRepo) CountNasabahBank(bankID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Nasabah{}).Where("bank_id = ?", bankID).Count(&count).Error
	return count, err
}

func (r *DashboardRepo) sumSaldo(query *gorm.DB) (SaldoPerSatuan, error) {
	type barisSaldo struct {
		Satuan models.SatuanRewardEnum `gorm:"column:satuan_nominal_saldo"`
		Total  float64                 `gorm:"column:total"`
	}

	var rows []barisSaldo
	if err := query.Model(&models.SaldoRekening{}).
		Select("satuan_nominal_saldo, COALESCE(SUM(nominal_saldo), 0) AS total").
		Group("satuan_nominal_saldo").
		Scan(&rows).Error; err != nil {
		return SaldoPerSatuan{}, err
	}

	var hasil SaldoPerSatuan
	for _, row := range rows {
		switch row.Satuan {
		case models.SatuanRewardEnumRp:
			hasil.Uang += row.Total
		case models.SatuanRewardEnumPoin:
			hasil.Poin += row.Total
		}
	}
	return hasil, nil
}

func (r *DashboardRepo) SumSaldoBank(bankID string) (SaldoPerSatuan, error) {
	return r.sumSaldo(r.db.Where("bank_id = ? AND entitas = ?", bankID, models.EntitasBankSampah))
}

func (r *DashboardRepo) SumSaldoNasabah(nasabahID string) (SaldoPerSatuan, error) {
	return r.sumSaldo(r.db.Where("nasabah_id = ? AND entitas = ?", nasabahID, models.EntitasNasabah))
}

func (r *DashboardRepo) SumSaldoSemuaNasabah(bankID string) (SaldoPerSatuan, error) {
	subNasabah := r.db.Model(&models.Nasabah{}).Select("nasabah_id").Where("bank_id = ?", bankID)
	return r.sumSaldo(r.db.Where("entitas = ? AND nasabah_id IN (?)", models.EntitasNasabah, subNasabah))
}

func (r *DashboardRepo) ListNasabahDenganSaldo(bankID string, limit, offset int) ([]NasabahSaldoRow, error) {
	rows := []NasabahSaldoRow{}
	err := r.db.Model(&models.Nasabah{}).
		Select(`
			nasabah.nasabah_id, nasabah.status_nasabah, users.nama AS nama_nasabah,
			COALESCE((SELECT SUM(sr.nominal_saldo) FROM saldo_rekening sr
				WHERE sr.nasabah_id = nasabah.nasabah_id AND sr.satuan_nominal_saldo = 'Rp'), 0) AS saldo_uang,
			COALESCE((SELECT SUM(sr.nominal_saldo) FROM saldo_rekening sr
				WHERE sr.nasabah_id = nasabah.nasabah_id AND sr.satuan_nominal_saldo = 'poin'), 0) AS saldo_poin
		`).
		Joins("JOIN users ON users.user_id = nasabah.user_id").
		Where("nasabah.bank_id = ?", bankID).
		Order("nasabah.nasabah_id ASC").
		Limit(limit).
		Offset(offset).
		Scan(&rows).Error
	return rows, err
}

func (r *DashboardRepo) FindReward(rewardID int) (*models.Reward, error) {
	var reward models.Reward
	err := r.db.Where("reward_id = ?", rewardID).First(&reward).Error
	return &reward, err
}

func (r *DashboardRepo) FindRewardByNama(nama models.RewardEnum) (*models.Reward, error) {
	var reward models.Reward
	err := r.db.Where("nama_reward = ?", nama).First(&reward).Error
	return &reward, err
}

func (r *DashboardRepo) FindRewardByJenis(nama models.RewardEnum, satuan models.SatuanRewardEnum) (*models.Reward, error) {
	var reward models.Reward
	err := r.db.Where("nama_reward = ? AND satuan = ?", nama, satuan).First(&reward).Error
	return &reward, err
}

func (r *DashboardRepo) AdaNilaiRewardAktif(bankID string, rewardID int) (bool, error) {
	var n int64
	err := nilaiRewardAktifSah(r.db, bankID).
		Where("nilai_reward_bank.reward_id = ?", rewardID).
		Count(&n).Error
	return n > 0, err
}

func (r *DashboardRepo) FindRekeningBank(tx *gorm.DB, bankID string, rewardID int) (*models.SaldoRekening, error) {
	var rekening models.SaldoRekening
	err := r.conn(tx).
		Where("bank_id = ? AND reward_id = ? AND entitas = ?", bankID, rewardID, models.EntitasBankSampah).
		First(&rekening).Error
	return &rekening, err
}

func (r *DashboardRepo) LockRekeningBank(tx *gorm.DB, bankID string, rewardID int) (*models.SaldoRekening, error) {
	var rekening models.SaldoRekening
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("bank_id = ? AND reward_id = ? AND entitas = ?", bankID, rewardID, models.EntitasBankSampah).
		First(&rekening).Error
	return &rekening, err
}

func (r *DashboardRepo) FindRekeningNasabah(nasabahID string, rewardID int) (*models.SaldoRekening, error) {
	var rekening models.SaldoRekening
	err := r.db.
		Where("nasabah_id = ? AND reward_id = ? AND entitas = ?", nasabahID, rewardID, models.EntitasNasabah).
		First(&rekening).Error
	return &rekening, err
}

func (r *DashboardRepo) CreateRekening(tx *gorm.DB, rekening *models.SaldoRekening) error {
	return r.conn(tx).Create(rekening).Error
}

func (r *DashboardRepo) ListArusSaldo(rekeningID string, start, end *time.Time) ([]models.RiwayatArusSaldo, error) {
	q := r.db.Where("rekening_id = ?", rekeningID)
	if start != nil {
		q = q.Where("created_at >= ?", *start)
	}
	if end != nil {
		q = q.Where("created_at <= ?", *end)
	}

	rows := []models.RiwayatArusSaldo{}
	err := q.Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *DashboardRepo) FindAdminIDByUserBank(userID, bankID string) (string, error) {
	var adminID string
	err := r.db.Model(&models.Admin{}).
		Where("user_id = ? AND bank_id = ? AND status_admin = ?", userID, bankID, models.Aktif).
		Limit(1).
		Pluck("admin_id", &adminID).Error
	return adminID, err
}
