package services_test

import (
	"testing"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func newRewardSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockRewardRepo, services.RewardService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockRewardRepo)
	return dbMock, repo, services.NewRewardService(gdb, repo)
}

func nilaiRewardAktif(nilaiRewardID, bankID string, rewardID int, level models.LevelUser) *models.NilaiRewardBank {
	return &models.NilaiRewardBank{
		NilaiRewardID:   nilaiRewardID,
		BankID:          bankID,
		RewardID:        rewardID,
		LevelUser:       level,
		PersenBagiHasil: 30,
		IsActive:        true,
	}
}

func aksesAdminBank(bankID string) services.AksesRewardInput {
	return services.AksesRewardInput{ActorUserID: "USR-ADM-001", ActorBankID: bankID}
}

func TestHapusNilaiReward_NonaktifkanSeluruhLevelPasangan(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)

	row := nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(row, nil)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("NonaktifkanNilaiRewardPasangan", mock.Anything, "BSI-001", 1, "ADM-001").
		Return(int64(2), nil)

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	err := svc.HapusNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestHapusNilaiReward_TidakMenghapusHistory(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)

	row := nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(row, nil)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("NonaktifkanNilaiRewardPasangan", mock.Anything, "BSI-001", 1, "ADM-001").
		Return(int64(2), nil)

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	assert.NoError(t, svc.HapusNilaiReward("2BSI-0011", aksesAdminBank("BSI-001")))

	repo.AssertNotCalled(t, "CreateHistory", mock.Anything, mock.Anything)
	assert.NotContains(t, methodNames(repo), "DeleteNilaiRewardCascade")
}

func TestHapusNilaiReward_TidakDitemukan(t *testing.T) {
	_, repo, svc := newRewardSvc(t)

	repo.On("FindNilaiRewardByID", "2BSI-0019").Return(nil, gorm.ErrRecordNotFound)

	err := svc.HapusNilaiReward("2BSI-0019", aksesAdminBank("BSI-001"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak ditemukan")
}

func TestHapusNilaiReward_TolakBankLain(t *testing.T) {
	_, repo, svc := newRewardSvc(t)

	row := nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(row, nil)

	err := svc.HapusNilaiReward("2BSI-0011", aksesAdminBank("BSI-002"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Akses ditolak")
	repo.AssertNotCalled(t, "HitungPemakaianNilaiReward", mock.Anything, mock.Anything)
}

func TestTambahNilaiReward_AktifkanKembaliYangPernahDinonaktifkan(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)

	reward := &models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnumRp}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI, IsActive: true}

	nonaktifNasabah := nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah)
	nonaktifNasabah.IsActive = false
	nonaktifEksternal := nilaiRewardAktif("1BSI-0011", "BSI-001", 1, models.LevelEksternal)
	nonaktifEksternal.IsActive = false
	nonaktifEksternal.PersenBagiHasil = 100

	repo.On("FindBank", "BSI-001").Return(bank, nil)
	repo.On("FindReward", mock.Anything, 1).Return(reward, nil)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).
		Return(nonaktifNasabah, nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelEksternal).
		Return(nonaktifEksternal, nil)
	repo.On("SaveNilaiReward", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateHistory", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"history_reward_id"}))
	dbMock.ExpectCommit()

	hasil, err := svc.TambahNilaiReward("BSI-001", services.NilaiRewardInput{
		RewardID:   1,
		Persentase: []services.PersentaseInput{{LevelUser: models.LevelNasabah, PersenBagiHasil: 40}},
	}, aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	assert.Len(t, hasil, 2, "level nasabah dan eksternal harus sama-sama hidup lagi")
	repo.AssertNotCalled(t, "CreateNilaiReward", mock.Anything, mock.Anything)
	assert.True(t, nonaktifNasabah.IsActive)
	assert.Equal(t, float64(40), nonaktifNasabah.PersenBagiHasil)
	assert.True(t, nonaktifEksternal.IsActive)
	repo.AssertNumberOfCalls(t, "CreateHistory", 1)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahNilaiReward_TolakKalauMasihAktif(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)

	reward := &models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnumRp}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI, IsActive: true}

	repo.On("FindBank", "BSI-001").Return(bank, nil)
	repo.On("FindReward", mock.Anything, 1).Return(reward, nil)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).
		Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	_, err := svc.TambahNilaiReward("BSI-001", services.NilaiRewardInput{
		RewardID:   1,
		Persentase: []services.PersentaseInput{{LevelUser: models.LevelNasabah, PersenBagiHasil: 40}},
	}, aksesAdminBank("BSI-001"))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sudah ada")
}

func methodNames(repo *mocks.MockRewardRepo) []string {
	nama := make([]string, 0, len(repo.ExpectedCalls))
	for _, call := range repo.ExpectedCalls {
		nama = append(nama, call.Method)
	}
	return nama
}
