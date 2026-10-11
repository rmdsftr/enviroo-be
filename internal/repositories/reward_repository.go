package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
)

type HistoryNilaiRewardRow struct {
	HistoryRewardID string    `gorm:"column:history_reward_id" json:"history_reward_id"`
	NilaiRewardID   string    `gorm:"column:nilai_reward_id"   json:"nilai_reward_id"`
	OldPersen       *float64  `gorm:"column:old_persen"        json:"old_persen"`
	NewPersen       *float64  `gorm:"column:new_persen"        json:"new_persen"`
	ChangedBy       string    `gorm:"column:changed_by"        json:"changed_by"`
	ChangedAt       time.Time `gorm:"column:changed_at"        json:"changed_at"`
}

type PemakaianReward struct {
	Label  string
	Jumlah int64
}

var tabelPemakaiReward = []struct {
	Tabel string
	Label string
}{
	{"nilai_reward_bank", "konfigurasi nilai reward bank"},
	{"katalog_sampah", "katalog sampah"},
	{"saldo_rekening", "saldo rekening nasabah"},
	{"penjualan", "transaksi penjualan"},
	{"penarikan", "transaksi penarikan"},
	{"penerima_bagi_hasil", "penerima bagi hasil"},
}

type RewardRepo struct {
	db *gorm.DB
}

func NewRewardRepo(db *gorm.DB) *RewardRepo {
	return &RewardRepo{db: db}
}

func (r *RewardRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *RewardRepo) ListRewards() ([]models.Reward, error) {
	var rewards []models.Reward
	err := r.db.Order("created_at DESC").Find(&rewards).Error
	return rewards, err
}

func (r *RewardRepo) FindReward(tx *gorm.DB, rewardID int) (*models.Reward, error) {
	var reward models.Reward
	err := r.conn(tx).Where("reward_id = ?", rewardID).First(&reward).Error
	return &reward, err
}

func (r *RewardRepo) FindRewardByJenis(nama models.RewardEnum, satuan models.SatuanRewardEnum, exceptID int) (*models.Reward, error) {
	q := r.db.Where("nama_reward = ? AND satuan = ?", nama, satuan)
	if exceptID > 0 {
		q = q.Where("reward_id <> ?", exceptID)
	}

	var reward models.Reward
	err := q.First(&reward).Error
	return &reward, err
}

func (r *RewardRepo) CreateReward(tx *gorm.DB, reward *models.Reward) error {
	return r.conn(tx).Create(reward).Error
}

func (r *RewardRepo) SaveReward(tx *gorm.DB, reward *models.Reward) error {
	return r.conn(tx).Save(reward).Error
}

func (r *RewardRepo) DeleteReward(tx *gorm.DB, reward *models.Reward) error {
	return r.conn(tx).Delete(reward).Error
}

func (r *RewardRepo) HitungPemakaianReward(rewardID int) ([]PemakaianReward, error) {
	var hasil []PemakaianReward
	for _, t := range tabelPemakaiReward {
		var count int64
		if err := r.db.Table(t.Tabel).Where("reward_id = ?", rewardID).Count(&count).Error; err != nil {
			return nil, err
		}
		if count > 0 {
			hasil = append(hasil, PemakaianReward{Label: t.Label, Jumlah: count})
		}
	}
	return hasil, nil
}

func (r *RewardRepo) FindBank(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.db.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *RewardRepo) ListNilaiRewardByBank(bankID string) ([]models.NilaiRewardBank, error) {
	var rows []models.NilaiRewardBank
	err := r.db.Preload("Reward").Where("bank_id = ?", bankID).Find(&rows).Error
	return rows, err
}

func (r *RewardRepo) ListNilaiRewardByBankReward(bankID string, rewardID int) ([]models.NilaiRewardBank, error) {
	var rows []models.NilaiRewardBank
	err := r.db.Where("bank_id = ? AND reward_id = ?", bankID, rewardID).Find(&rows).Error
	return rows, err
}

func (r *RewardRepo) FindNilaiReward(tx *gorm.DB, bankID string, rewardID int, level models.LevelUser) (*models.NilaiRewardBank, error) {
	var row models.NilaiRewardBank
	err := r.conn(tx).Preload("Reward").
		Where("bank_id = ? AND reward_id = ? AND level_user = ?", bankID, rewardID, level).
		First(&row).Error
	return &row, err
}

func (r *RewardRepo) FindNilaiRewardByID(nilaiRewardID string) (*models.NilaiRewardBank, error) {
	var row models.NilaiRewardBank
	err := r.db.Preload("Reward").Where("nilai_reward_id = ?", nilaiRewardID).First(&row).Error
	return &row, err
}

func (r *RewardRepo) CreateNilaiReward(tx *gorm.DB, row *models.NilaiRewardBank) error {
	return r.conn(tx).Create(row).Error
}

func (r *RewardRepo) SaveNilaiReward(tx *gorm.DB, row *models.NilaiRewardBank) error {
	return r.conn(tx).Save(row).Error
}

func (r *RewardRepo) NonaktifkanNilaiRewardPasangan(tx *gorm.DB, bankID string, rewardID int, adminID string) (int64, error) {
	res := r.conn(tx).Model(&models.NilaiRewardBank{}).
		Where("bank_id = ? AND reward_id = ? AND is_active = ?", bankID, rewardID, true).
		Updates(map[string]any{
			"is_active":  false,
			"updated_by": adminID,
			"updated_at": time.Now(),
		})
	return res.RowsAffected, res.Error
}

func nilaiRewardAktifSah(db *gorm.DB, bankID string) *gorm.DB {
	return db.Model(&models.NilaiRewardBank{}).
		Joins("JOIN reward r ON r.reward_id = nilai_reward_bank.reward_id").
		Where("nilai_reward_bank.bank_id = ? AND nilai_reward_bank.is_active = ?", bankID, true).
		Where("((r.nama_reward = ? AND r.satuan = ?) OR (r.nama_reward = ? AND r.satuan = ?))",
			models.RewardEnumUang, models.SatuanRewardEnumRp,
			models.RewardEnumBarang, models.SatuanRewardEnumPoin)
}

func (r *RewardRepo) ListHistoryByNilaiReward(nilaiRewardIDs []string) ([]HistoryNilaiRewardRow, error) {
	if len(nilaiRewardIDs) == 0 {
		return []HistoryNilaiRewardRow{}, nil
	}

	rows := []HistoryNilaiRewardRow{}
	err := r.db.Table("history_nilai_reward hnr").
		Select(`hnr.history_reward_id, hnr.nilai_reward_id,
			hnr.old_persen, hnr.new_persen,
			COALESCE(u.nama, hnr.changed_by) AS changed_by,
			hnr.changed_at`).
		Joins("LEFT JOIN admin a ON a.admin_id = hnr.changed_by").
		Joins("LEFT JOIN users u ON u.user_id = a.user_id").
		Where("hnr.nilai_reward_id IN ?", nilaiRewardIDs).
		Order("hnr.changed_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *RewardRepo) CreateHistory(tx *gorm.DB, history *models.HistoryNilaiReward) error {
	return r.conn(tx).Create(history).Error
}

func (r *RewardRepo) FindAdminIDByUserBank(userID, bankID string) (string, error) {
	var adminID string
	err := r.db.Model(&models.Admin{}).
		Where("user_id = ? AND bank_id = ? AND status_admin = ?", userID, bankID, models.Aktif).
		Limit(1).
		Pluck("admin_id", &adminID).Error
	return adminID, err
}
