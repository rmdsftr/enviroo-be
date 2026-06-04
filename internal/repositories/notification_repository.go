package repositories

import (
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"enviroo-be/internal/models"
)

type NotifikasiRepository interface {
	Create(notif *models.Notifikasi) error
	FindByID(notifikasiID string) (*models.Notifikasi, error)
	FindByUserID(userID string, limit, offset int) ([]models.Notifikasi, int64, error)
	MarkAsRead(notifikasiID, userID string) error
	MarkAllAsRead(userID string) error
	CountUnread(userID string) (int64, error)
	Delete(notifikasiID, userID string) error
}

type notifikasiRepository struct {
	db *gorm.DB
}

func NewNotifikasiRepository(db *gorm.DB) NotifikasiRepository {
	return &notifikasiRepository{db: db}
}

func (r *notifikasiRepository) Create(notif *models.Notifikasi) error {
	if notif.NotifikasiID == "" {
		notif.NotifikasiID = uuid.New().String()
	}
	return r.db.Create(notif).Error
}

func (r *notifikasiRepository) FindByID(notifikasiID string) (*models.Notifikasi, error) {
	var notif models.Notifikasi
	err := r.db.Where("notifikasi_id = ?", notifikasiID).First(&notif).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &notif, nil
}

// FindByUserID returns paginated notifications for a user, newest first.
func (r *notifikasiRepository) FindByUserID(userID string, limit, offset int) ([]models.Notifikasi, int64, error) {
	var notifs []models.Notifikasi
	var total int64

	base := r.db.Model(&models.Notifikasi{}).Where("user_id = ?", userID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifs).Error; err != nil {
		return nil, 0, err
	}

	return notifs, total, nil
}

func (r *notifikasiRepository) MarkAsRead(notifikasiID, userID string) error {
	result := r.db.Model(&models.Notifikasi{}).
		Where("notifikasi_id = ? AND user_id = ?", notifikasiID, userID).
		Update("is_read", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("notifikasi tidak ditemukan")
	}
	return nil
}

func (r *notifikasiRepository) MarkAllAsRead(userID string) error {
	return r.db.Model(&models.Notifikasi{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}

func (r *notifikasiRepository) CountUnread(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Notifikasi{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count).Error
	return count, err
}

func (r *notifikasiRepository) Delete(notifikasiID, userID string) error {
	result := r.db.
		Where("notifikasi_id = ? AND user_id = ?", notifikasiID, userID).
		Delete(&models.Notifikasi{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("notifikasi tidak ditemukan")
	}
	return nil
}