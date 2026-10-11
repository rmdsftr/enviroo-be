package services

import (
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/utils"
	"time"

	"gorm.io/gorm"
)

func createAktivasiToken(tx *gorm.DB, userID string, asRole models.RoleUserEnum, nasabahID, adminID *string, generatedBy string) (otp string, expiredAt time.Time, err error) {
	otp, err = utils.GenerateOTP()
	if err != nil {
		return "", time.Time{}, err
	}

	hashedOTP, err := utils.HashOTP(otp)
	if err != nil {
		return "", time.Time{}, err
	}

	expiredAt = time.Now().Add(24 * time.Hour)
	aktivasiID, err := idgen.GenerateAktivasiID(tx, userID, asRole, models.TujuanAktivasi)
	if err != nil {
		return "", time.Time{}, err
	}
	aktivasi := models.AktivasiAkun{
		AktivasiID:  aktivasiID,
		UserID:      userID,
		AsRole:      asRole,
		NasabahID:   nasabahID,
		AdminID:     adminID,
		Token:       hashedOTP,
		ExpiredAt:   expiredAt,
		GeneratedBy: generatedBy,
		Tujuan:      models.TujuanAktivasi,
	}
	if err := tx.Create(&aktivasi).Error; err != nil {
		return "", time.Time{}, err
	}

	return otp, expiredAt, nil
}
