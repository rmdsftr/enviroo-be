package repositories

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"enviroo-be/internal/models"
)

type NotifikasiRepository interface {
	Create(ctx context.Context, notif *models.Notifikasi) error
	FindByID(ctx context.Context, notifikasiID string) (*models.Notifikasi, error)
	FindByUserID(ctx context.Context, userID, roleTarget string, limit, offset int) ([]models.Notifikasi, int64, error)
	MarkAsRead(ctx context.Context, notifikasiID, userID string) error
	MarkAllAsRead(ctx context.Context, userID, roleTarget string) error
	CountUnread(ctx context.Context, userID, roleTarget string) (int64, error)
	Delete(ctx context.Context, notifikasiID, userID string) error
}

type notifikasiRepository struct {
	db *gorm.DB
}

func NewNotifikasiRepository(db *gorm.DB) NotifikasiRepository {
	return &notifikasiRepository{db: db}
}

func (r *notifikasiRepository) Create(ctx context.Context, notif *models.Notifikasi) error {
	if notif.NotifikasiID == "" {
		notif.NotifikasiID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(notif).Error
}

func (r *notifikasiRepository) FindByID(ctx context.Context, notifikasiID string) (*models.Notifikasi, error) {
	var notif models.Notifikasi
	err := r.db.WithContext(ctx).Where("notifikasi_id = ?", notifikasiID).First(&notif).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &notif, nil
}

func (r *notifikasiRepository) FindByUserID(ctx context.Context, userID, roleTarget string, limit, offset int) ([]models.Notifikasi, int64, error) {
	var notifs []models.Notifikasi
	var total int64

	base := r.db.WithContext(ctx).Model(&models.Notifikasi{}).Where("user_id = ? AND role_target = ?", userID, roleTarget)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := base.Order("created_at DESC").Limit(limit).Offset(offset).Find(&notifs).Error; err != nil {
		return nil, 0, err
	}

	return notifs, total, nil
}

func (r *notifikasiRepository) MarkAsRead(ctx context.Context, notifikasiID, userID string) error {
	result := r.db.WithContext(ctx).Model(&models.Notifikasi{}).
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

func (r *notifikasiRepository) MarkAllAsRead(ctx context.Context, userID, roleTarget string) error {
	return r.db.WithContext(ctx).Model(&models.Notifikasi{}).
		Where("user_id = ? AND role_target = ? AND is_read = false", userID, roleTarget).
		Update("is_read", true).Error
}

func (r *notifikasiRepository) CountUnread(ctx context.Context, userID, roleTarget string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Notifikasi{}).
		Where("user_id = ? AND role_target = ? AND is_read = false", userID, roleTarget).
		Count(&count).Error
	return count, err
}

func (r *notifikasiRepository) Delete(ctx context.Context, notifikasiID, userID string) error {
	result := r.db.WithContext(ctx).
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
