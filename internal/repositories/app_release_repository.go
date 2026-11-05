package repositories

import (
	"enviroo-be/internal/models"

	"gorm.io/gorm"
)

type AppReleaseRepo struct {
	db *gorm.DB
}

func NewAppReleaseRepo(db *gorm.DB) *AppReleaseRepo {
	return &AppReleaseRepo{db: db}
}

func (r *AppReleaseRepo) conn(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return r.db
}

func (r *AppReleaseRepo) FindLatest() (*models.AppRelease, error) {
	var rilis models.AppRelease
	err := r.db.Order("version_code DESC").First(&rilis).Error
	return &rilis, err
}

func (r *AppReleaseRepo) FindByID(releaseID int) (*models.AppRelease, error) {
	var rilis models.AppRelease
	err := r.db.Where("release_id = ?", releaseID).First(&rilis).Error
	return &rilis, err
}

func (r *AppReleaseRepo) MaxVersionCode() (int, error) {
	var maks int
	err := r.db.Model(&models.AppRelease{}).
		Select("COALESCE(MAX(version_code), 0)").
		Scan(&maks).Error
	return maks, err
}

// AdaWajibDiAntara: apakah ada rilis wajib dengan current < version_code <= latest.
func (r *AppReleaseRepo) AdaWajibDiAntara(current, latest int) (bool, error) {
	var jumlah int64
	err := r.db.Model(&models.AppRelease{}).
		Where("version_code > ? AND version_code <= ? AND is_mandatory = ?", current, latest, true).
		Count(&jumlah).Error
	return jumlah > 0, err
}

func (r *AppReleaseRepo) List() ([]models.AppRelease, error) {
	var rows []models.AppRelease
	err := r.db.Order("version_code DESC").Find(&rows).Error
	return rows, err
}

func (r *AppReleaseRepo) Create(tx *gorm.DB, rilis *models.AppRelease) error {
	return r.conn(tx).Create(rilis).Error
}

func (r *AppReleaseRepo) Save(rilis *models.AppRelease) error {
	return r.db.Save(rilis).Error
}

func (r *AppReleaseRepo) Delete(rilis *models.AppRelease) error {
	return r.db.Delete(rilis).Error
}
