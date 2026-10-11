package repositories

import (
	"enviroo-be/internal/models"

	"gorm.io/gorm"
)

type MitraRepo struct {
	DB *gorm.DB
}

func NewMitraRepo(db *gorm.DB) *MitraRepo {
	return &MitraRepo{DB: db}
}

func (r *MitraRepo) GetBankByID(bankID string) (*models.BankSampah, error) {
	var bank models.BankSampah
	err := r.DB.Where("bank_id = ?", bankID).First(&bank).Error
	return &bank, err
}

func (r *MitraRepo) ListByBank(bankID string) ([]models.MitraPengepul, error) {
	var mitra []models.MitraPengepul
	err := r.DB.Where("bank_id = ?", bankID).Order("nama_mitra ASC").Find(&mitra).Error
	return mitra, err
}

func (r *MitraRepo) GetByID(db *gorm.DB, bankID, mitraID string) (*models.MitraPengepul, error) {
	var mitra models.MitraPengepul
	err := db.Where("mitra_id = ? AND bank_id = ?", mitraID, bankID).First(&mitra).Error
	return &mitra, err
}

func (r *MitraRepo) FindByNormalizedName(db *gorm.DB, bankID, normalizedName string) (*models.MitraPengepul, error) {
	var mitra models.MitraPengepul
	err := db.Where(
		"bank_id = ? AND lower(regexp_replace(nama_mitra, '[^a-zA-Z0-9]', '', 'g')) = ?",
		bankID, normalizedName,
	).First(&mitra).Error
	return &mitra, err
}

func (r *MitraRepo) Create(db *gorm.DB, m *models.MitraPengepul) error {
	return db.Create(m).Error
}

func (r *MitraRepo) Update(db *gorm.DB, m *models.MitraPengepul) error {
	return db.Save(m).Error
}

func (r *MitraRepo) IsUsedInPenjualan(db *gorm.DB, mitraID string) (bool, error) {
	var count int64
	err := db.Model(&models.Penjualan{}).Where("mitra_id = ?", mitraID).Count(&count).Error
	return count > 0, err
}

func (r *MitraRepo) Delete(db *gorm.DB, m *models.MitraPengepul) error {
	return db.Delete(m).Error
}
