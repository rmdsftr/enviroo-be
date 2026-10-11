package repositories

import (
	"time"

	"enviroo-be/internal/models"

	"gorm.io/gorm"
)

type PenjualanRepo struct {
	DB *gorm.DB
}

func NewPenjualanRepo(db *gorm.DB) *PenjualanRepo {
	return &PenjualanRepo{DB: db}
}

func (r *PenjualanRepo) GetBankByID(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.DB.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *PenjualanRepo) GetRewardByID(db *gorm.DB, rewardID int) (*models.Reward, error) {
	var reward models.Reward
	err := db.Where("reward_id = ?", rewardID).First(&reward).Error
	return &reward, err
}

func (r *PenjualanRepo) GetPersenNasabah(db *gorm.DB, bankID string, rewardID int) (float64, error) {
	var nilai models.NilaiRewardBank
	err := db.Where("bank_id = ? AND reward_id = ? AND level_user = ?",
		bankID, rewardID, models.LevelNasabah).First(&nilai).Error
	if err != nil {
		return 0, err
	}
	return nilai.PersenBagiHasil, nil
}

func (r *PenjualanRepo) GetKatalogSampah(db *gorm.DB, sampahID string) (*models.KatalogSampah, error) {
	var sampah models.KatalogSampah
	err := db.Preload("Sarok").Where("sampah_id = ?", sampahID).First(&sampah).Error
	return &sampah, err
}

func (r *PenjualanRepo) GetStokSampah(db *gorm.DB, bankID, sampahID string) (*models.StokSampah, error) {
	var stok models.StokSampah
	err := db.Where("bank_id = ? AND sampah_id = ?", bankID, sampahID).First(&stok).Error
	return &stok, err
}

func (r *PenjualanRepo) GetSchemaHarga(db *gorm.DB, sampahID string, level models.LevelUser) (*models.SchemaHargaSampah, error) {
	var schema models.SchemaHargaSampah
	err := db.Where("sampah_id = ? AND level_user = ?", sampahID, level).First(&schema).Error
	return &schema, err
}

func (r *PenjualanRepo) CreateSchemaHarga(db *gorm.DB, s *models.SchemaHargaSampah) error {
	return db.Create(s).Error
}

func (r *PenjualanRepo) UpdateSchemaHarga(db *gorm.DB, s *models.SchemaHargaSampah) error {
	return db.Save(s).Error
}

func (r *PenjualanRepo) CreateHistoryHarga(db *gorm.DB, h *models.KatalogSampahHistory) error {
	return db.Create(h).Error
}

func (r *PenjualanRepo) DecrStokSampah(db *gorm.DB, bankID, sampahID string, newStok float64) error {
	return db.Model(&models.StokSampah{}).
		Where("bank_id = ? AND sampah_id = ?", bankID, sampahID).
		Update("stok", newStok).Error
}

func (r *PenjualanRepo) CreatePenjualan(db *gorm.DB, p *models.Penjualan) error {
	return db.Create(p).Error
}

func (r *PenjualanRepo) CreateDetailPenjualan(db *gorm.DB, details []models.DetailPenjualan) error {
	return db.Create(&details).Error
}

func (r *PenjualanRepo) GetPenjualan(penjualanID string) (*models.Penjualan, error) {
	var p models.Penjualan
	err := r.DB.Preload("BankSampah").Preload("Reward").Preload("MitraPengepul").
		Where("penjualan_id = ?", penjualanID).First(&p).Error
	return &p, err
}

func (r *PenjualanRepo) GetDetailItems(penjualanID string) ([]models.DetailPenjualan, error) {
	var details []models.DetailPenjualan
	err := r.DB.Preload("Sampah.Sarok").Where("penjualan_id = ?", penjualanID).Find(&details).Error
	return details, err
}

func (r *PenjualanRepo) GetAdminNama(adminID string) string {
	var nama string
	r.DB.Table("users").Select("users.nama").
		Joins("JOIN admin ON admin.user_id = users.user_id").
		Where("admin.admin_id = ?", adminID).
		Scan(&nama)
	return nama
}

type RiwayatPenjualanRow struct {
	PenjualanID     string                     `gorm:"column:penjualan_id"`
	MitraID         string                     `gorm:"column:mitra_id"`
	NamaMitra       string                     `gorm:"column:nama_mitra"`
	TotalItem       int                        `gorm:"column:total_item"`
	TotalPenjualan  float64                    `gorm:"column:total_penjualan"`
	NamaReward      string                     `gorm:"column:nama_reward"`
	SatuanReward    models.SatuanRewardEnum    `gorm:"column:satuan_reward"`
	BuktiFoto       string                     `gorm:"column:bukti_foto"`
	CreatedAt       time.Time                  `gorm:"column:created_at"`
	AdminName       string                     `gorm:"column:admin_name"`
	StatusBagiHasil models.StatusBagiHasilEnum `gorm:"column:status_bagi_hasil"`
}

func (r *PenjualanRepo) GetRiwayatPenjualan(bankID string, start, end *time.Time) ([]RiwayatPenjualanRow, error) {
	q := r.DB.Table("penjualan").
		Select(`penjualan.penjualan_id, penjualan.mitra_id, mitra_pengepul.nama_mitra,
		        penjualan.total_item, penjualan.total_penjualan, penjualan.satuan_reward,
		        penjualan.bukti_foto, penjualan.created_at, penjualan.status_bagi_hasil,
		        users.nama as admin_name, reward.nama_reward as nama_reward`).
		Joins("LEFT JOIN admin ON penjualan.sold_by = admin.admin_id").
		Joins("LEFT JOIN users ON admin.user_id = users.user_id").
		Joins("LEFT JOIN reward ON penjualan.reward_id = reward.reward_id").
		Joins("LEFT JOIN mitra_pengepul ON penjualan.mitra_id = mitra_pengepul.mitra_id").
		Where("penjualan.bank_id = ?", bankID)

	if start != nil {
		q = q.Where("penjualan.created_at >= ?", *start)
	}
	if end != nil {
		q = q.Where("penjualan.created_at <= ?", *end)
	}

	rows := []RiwayatPenjualanRow{}
	err := q.Order("penjualan.created_at DESC").Find(&rows).Error
	return rows, err
}
