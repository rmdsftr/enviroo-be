package repositories

import (
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

// GetPersenNasabah returns the bagi-hasil percentage for the bank+reward combination.
// Returns 0 if no active nilai_reward_bank record is found (graceful default).
func (r *PenjualanRepo) GetPersenNasabah(db *gorm.DB, bankID string, rewardID int) float64 {
	var nilai models.NilaiRewardBank
	err := db.Where("bank_id = ? AND reward_id = ? AND level_user = ? AND is_active = true",
		bankID, rewardID, models.LevelNasabah).First(&nilai).Error
	if err != nil {
		return 0
	}
	return nilai.PersenBagiHasil
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
	err := r.DB.Preload("BankSampah").Preload("Reward").
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

func (r *PenjualanRepo) GetDistinctMitra(bankID string) ([]string, error) {
	var names []string
	err := r.DB.Table("penjualan").
		Select("DISTINCT identitas_pembeli").
		Where("bank_id = ? AND identitas_pembeli IS NOT NULL AND identitas_pembeli != ''", bankID).
		Pluck("identitas_pembeli", &names).Error
	return names, err
}
