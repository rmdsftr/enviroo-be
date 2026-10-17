package services_test

import (
	"errors"
	"net/http"
	"testing"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
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

func rwdKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func rewardInputUang() services.RewardInput {
	return services.RewardInput{NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnumRp, Deskripsi: ptrStr("Saldo rupiah")}
}

func rewardInputBarang() services.RewardInput {
	return services.RewardInput{NamaReward: models.RewardEnumBarang, Satuan: models.SatuanRewardEnumPoin, Deskripsi: ptrStr("Tukar barang")}
}

func persenNasabah(persen float64) services.NilaiRewardInput {
	return services.NilaiRewardInput{
		RewardID:   1,
		Persentase: []services.PersentaseInput{{LevelUser: models.LevelNasabah, PersenBagiHasil: persen}},
	}
}

func bankBSI() *models.BankSampah {
	return &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI, IsActive: true}
}

func bankBSUAnak(parentID *string) *models.BankSampah {
	return &models.BankSampah{BankID: "BSU-001", JenisBank: models.BSU, ParentBankID: parentID, IsActive: true}
}

// ===== Master reward =====

func TestListReward_RepoError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("ListRewards").Return(nil, errors.New("db error"))

	hasil, err := svc.ListReward()

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestListReward_Sukses(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("ListRewards").Return([]models.Reward{
		{RewardID: 1, NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnumRp, Deskripsi: ptrStr("Saldo rupiah")},
		{RewardID: 2, NamaReward: models.RewardEnumBarang, Satuan: models.SatuanRewardEnumPoin},
	}, nil)

	hasil, err := svc.ListReward()

	assert.NoError(t, err)
	assert.Equal(t, []services.RewardRow{
		{RewardID: 1, NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnumRp, Deskripsi: ptrStr("Saldo rupiah")},
		{RewardID: 2, NamaReward: models.RewardEnumBarang, Satuan: models.SatuanRewardEnumPoin},
	}, hasil)
}

func TestTambahReward_JenisTidakValid(t *testing.T) {
	cases := map[string]services.RewardInput{
		"nama tidak dikenal":   {NamaReward: models.RewardEnum("Emas"), Satuan: models.SatuanRewardEnumRp},
		"satuan tidak dikenal": {NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnum("gram")},
	}
	for nama, in := range cases {
		t.Run(nama, func(t *testing.T) {
			_, repo, svc := newRewardSvc(t)

			hasil, err := svc.TambahReward(in)

			assert.Nil(t, hasil)
			assert.Equal(t, http.StatusBadRequest, rwdKode(t, err))
			repo.AssertNotCalled(t, "FindRewardByJenis", mock.Anything, mock.Anything, mock.Anything)
		})
	}
}

func TestTambahReward_SudahAda(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp, 0).Return(rewardUang(), nil)

	hasil, err := svc.TambahReward(rewardInputUang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, rwdKode(t, err))
	repo.AssertNotCalled(t, "CreateReward", mock.Anything, mock.Anything)
}

func TestTambahReward_CekJenisError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp, 0).Return(nil, errors.New("db error"))

	hasil, err := svc.TambahReward(rewardInputUang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestTambahReward_CreateError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp, 0).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateReward", mock.Anything, mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.TambahReward(rewardInputUang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestTambahReward_Sukses(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp, 0).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateReward", mock.Anything, mock.MatchedBy(func(r *models.Reward) bool {
		return r.NamaReward == models.RewardEnumUang && r.Satuan == models.SatuanRewardEnumRp &&
			r.Deskripsi != nil && *r.Deskripsi == "Saldo rupiah"
	})).Return(nil).Once()

	hasil, err := svc.TambahReward(rewardInputUang())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.RewardEnumUang, hasil.NamaReward)
		assert.Equal(t, models.SatuanRewardEnumRp, hasil.Satuan)
		assert.Equal(t, "Saldo rupiah", *hasil.Deskripsi)
	}
	repo.AssertExpectations(t)
}

func TestUbahReward_IDTidakValid(t *testing.T) {
	for _, param := range []string{"abc", "0"} {
		t.Run(param, func(t *testing.T) {
			_, repo, svc := newRewardSvc(t)

			hasil, err := svc.UbahReward(param, rewardInputUang())

			assert.Nil(t, hasil)
			assert.Equal(t, http.StatusBadRequest, rwdKode(t, err))
			repo.AssertNotCalled(t, "FindReward", mock.Anything, mock.Anything)
		})
	}
}

func TestUbahReward_TidakDitemukan(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 9).Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.UbahReward("9", rewardInputUang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, rwdKode(t, err))
}

func TestUbahReward_FindError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(nil, errors.New("db error"))

	hasil, err := svc.UbahReward("1", rewardInputUang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestUbahReward_JenisTidakValid(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)

	hasil, err := svc.UbahReward("1", services.RewardInput{NamaReward: models.RewardEnumUang, Satuan: "gram"})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, rwdKode(t, err))
	repo.AssertNotCalled(t, "SaveReward", mock.Anything, mock.Anything)
}

func TestUbahReward_HanyaDeskripsi(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("SaveReward", mock.Anything, mock.MatchedBy(func(r *models.Reward) bool {
		return r.Deskripsi != nil && *r.Deskripsi == "Saldo rupiah"
	})).Return(nil).Once()

	hasil, err := svc.UbahReward("1", rewardInputUang())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Saldo rupiah", *hasil.Deskripsi)
	}
	repo.AssertNotCalled(t, "HitungPemakaianReward", mock.Anything)
	repo.AssertNotCalled(t, "FindRewardByJenis", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestUbahReward_JenisBerubahSudahDipakai(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("HitungPemakaianReward", 1).Return([]repositories.PemakaianReward{
		{Label: "katalog sampah", Jumlah: 2},
		{Label: "penjualan", Jumlah: 1},
	}, nil)

	hasil, err := svc.UbahReward("1", rewardInputBarang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, rwdKode(t, err))
	assert.Contains(t, err.Error(), "2 katalog sampah, 1 penjualan")
	repo.AssertNotCalled(t, "SaveReward", mock.Anything, mock.Anything)
}

func TestUbahReward_JenisBerubahDuplikat(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("HitungPemakaianReward", 1).Return(nil, nil)
	repo.On("FindRewardByJenis", models.RewardEnumBarang, models.SatuanRewardEnumPoin, 1).
		Return(&models.Reward{RewardID: 2, NamaReward: models.RewardEnumBarang, Satuan: models.SatuanRewardEnumPoin}, nil)

	hasil, err := svc.UbahReward("1", rewardInputBarang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, rwdKode(t, err))
	repo.AssertNotCalled(t, "SaveReward", mock.Anything, mock.Anything)
}

func TestUbahReward_HitungPemakaianError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("HitungPemakaianReward", 1).Return(nil, errors.New("db error"))

	hasil, err := svc.UbahReward("1", rewardInputBarang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestUbahReward_JenisBerubahSukses(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("HitungPemakaianReward", 1).Return(nil, nil)
	repo.On("FindRewardByJenis", models.RewardEnumBarang, models.SatuanRewardEnumPoin, 1).Return(nil, gorm.ErrRecordNotFound)
	repo.On("SaveReward", mock.Anything, mock.MatchedBy(func(r *models.Reward) bool {
		return r.RewardID == 1 && r.NamaReward == models.RewardEnumBarang && r.Satuan == models.SatuanRewardEnumPoin
	})).Return(nil).Once()

	hasil, err := svc.UbahReward("1", rewardInputBarang())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, 1, hasil.RewardID)
		assert.Equal(t, models.RewardEnumBarang, hasil.NamaReward)
		assert.Equal(t, models.SatuanRewardEnumPoin, hasil.Satuan)
		assert.Equal(t, "Tukar barang", *hasil.Deskripsi)
	}
	repo.AssertExpectations(t)
}

func TestUbahReward_SaveError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("SaveReward", mock.Anything, mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.UbahReward("1", rewardInputUang())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestHapusReward_IDTidakValid(t *testing.T) {
	_, repo, svc := newRewardSvc(t)

	err := svc.HapusReward("-1")

	assert.Equal(t, http.StatusBadRequest, rwdKode(t, err))
	repo.AssertNotCalled(t, "FindReward", mock.Anything, mock.Anything)
}

func TestHapusReward_TidakDitemukan(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 9).Return(nil, gorm.ErrRecordNotFound)

	err := svc.HapusReward("9")

	assert.Equal(t, http.StatusNotFound, rwdKode(t, err))
}

func TestHapusReward_FindError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(nil, errors.New("db error"))

	err := svc.HapusReward("1")

	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestHapusReward_MasihDipakai(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("HitungPemakaianReward", 1).Return([]repositories.PemakaianReward{
		{Label: "konfigurasi nilai reward bank", Jumlah: 3},
	}, nil)

	err := svc.HapusReward("1")

	assert.Equal(t, http.StatusConflict, rwdKode(t, err))
	assert.Contains(t, err.Error(), "3 konfigurasi nilai reward bank")
	repo.AssertNotCalled(t, "DeleteReward", mock.Anything, mock.Anything)
}

func TestHapusReward_HitungPemakaianError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("HitungPemakaianReward", 1).Return(nil, errors.New("db error"))

	err := svc.HapusReward("1")

	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	repo.AssertNotCalled(t, "DeleteReward", mock.Anything, mock.Anything)
}

func TestHapusReward_DeleteError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("HitungPemakaianReward", 1).Return(nil, nil)
	repo.On("DeleteReward", mock.Anything, mock.Anything).Return(errors.New("db error"))

	err := svc.HapusReward("1")

	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestHapusReward_Sukses(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	reward := rewardUang()
	repo.On("FindReward", mock.Anything, 1).Return(reward, nil)
	repo.On("HitungPemakaianReward", 1).Return(nil, nil)
	repo.On("DeleteReward", mock.Anything, reward).Return(nil).Once()

	assert.NoError(t, svc.HapusReward("1"))
	repo.AssertExpectations(t)
}

// ===== Nilai reward bank =====

func TestListNilaiRewardBank_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSI-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.ListNilaiRewardBank("BSI-XXX")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, rwdKode(t, err))
}

func TestListNilaiRewardBank_BSUPakaiNilaiBSI(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSU-001").Return(bankBSUAnak(ptrStr("BSI-001")), nil)
	repo.On("ListNilaiRewardByBank", "BSI-001").Return([]models.NilaiRewardBank{
		*nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah),
	}, nil).Once()

	hasil, err := svc.ListNilaiRewardBank("BSU-001")

	assert.NoError(t, err)
	if assert.Len(t, hasil, 1) {
		assert.Equal(t, "BSI-001", hasil[0].BankID, "BSU memakai nilai reward milik BSI induknya")
	}
	repo.AssertExpectations(t)
}

func TestListNilaiRewardBank_BSUTanpaParent(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSU-001").Return(bankBSUAnak(nil), nil)

	hasil, err := svc.ListNilaiRewardBank("BSU-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, rwdKode(t, err))
	repo.AssertNotCalled(t, "ListNilaiRewardByBank", mock.Anything)
}

func TestListNilaiRewardBank_RepoError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSI-001").Return(bankBSI(), nil)
	repo.On("ListNilaiRewardByBank", "BSI-001").Return(nil, errors.New("db error"))

	hasil, err := svc.ListNilaiRewardBank("BSI-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestListNilaiRewardBank_Sukses(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	denganReward := *nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah)
	denganReward.Reward = *rewardUang()
	denganReward.CreatedBy = ptrStr("ADM-001")
	tanpaReward := *nilaiRewardAktif("1BSI-0011", "BSI-001", 1, models.LevelEksternal)
	tanpaReward.PersenBagiHasil = 100
	repo.On("FindBank", "BSI-001").Return(bankBSI(), nil)
	repo.On("ListNilaiRewardByBank", "BSI-001").Return([]models.NilaiRewardBank{denganReward, tanpaReward}, nil)

	hasil, err := svc.ListNilaiRewardBank("BSI-001")

	assert.NoError(t, err)
	if assert.Len(t, hasil, 2) {
		assert.Equal(t, "2BSI-0011", hasil[0].NilaiRewardID)
		assert.Equal(t, models.LevelNasabah, hasil[0].LevelUser)
		assert.Equal(t, 30.0, hasil[0].PersenBagiHasil)
		assert.True(t, hasil[0].IsActive)
		assert.Equal(t, "ADM-001", *hasil[0].CreatedBy)
		if assert.NotNil(t, hasil[0].Reward) {
			assert.Equal(t, models.RewardEnumUang, hasil[0].Reward.NamaReward)
		}
		assert.Equal(t, 100.0, hasil[1].PersenBagiHasil)
		assert.Nil(t, hasil[1].Reward, "relasi reward kosong tidak ikut dikirim")
	}
}

func TestDetailNilaiReward_TidakDitemukan(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0019").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.DetailNilaiReward("2BSI-0019", aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, rwdKode(t, err))
}

func TestDetailNilaiReward_FindError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nil, errors.New("db error"))

	hasil, err := svc.DetailNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestDetailNilaiReward_BankLainDitolak(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("FindBank", "BSI-002").Return(&models.BankSampah{BankID: "BSI-002", JenisBank: models.BSI}, nil)

	hasil, err := svc.DetailNilaiReward("2BSI-0011", aksesAdminBank("BSI-002"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, rwdKode(t, err))
	repo.AssertNotCalled(t, "ListNilaiRewardByBankReward", mock.Anything, mock.Anything)
}

func TestDetailNilaiReward_BSUAnakBolehLihat(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("FindBank", "BSU-001").Return(bankBSUAnak(ptrStr("BSI-001")), nil)
	repo.On("ListNilaiRewardByBankReward", "BSI-001", 1).Return([]models.NilaiRewardBank{}, nil)
	repo.On("ListHistoryByNilaiReward", []string{}).Return([]repositories.HistoryNilaiRewardRow{}, nil)

	hasil, err := svc.DetailNilaiReward("2BSI-0011", aksesAdminBank("BSU-001"))

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestDetailNilaiReward_ListLevelError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("ListNilaiRewardByBankReward", "BSI-001", 1).Return(nil, errors.New("db error"))

	hasil, err := svc.DetailNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	repo.AssertNotCalled(t, "ListHistoryByNilaiReward", mock.Anything)
}

func TestDetailNilaiReward_HistoryError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("ListNilaiRewardByBankReward", "BSI-001", 1).Return([]models.NilaiRewardBank{}, nil)
	repo.On("ListHistoryByNilaiReward", mock.Anything).Return(nil, errors.New("db error"))

	hasil, err := svc.DetailNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestDetailNilaiReward_Sukses(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	awal := nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah)
	awal.Reward = *rewardUang()
	eksternal := *nilaiRewardAktif("1BSI-0011", "BSI-001", 1, models.LevelEksternal)
	eksternal.PersenBagiHasil = 100
	history := []repositories.HistoryNilaiRewardRow{
		{HistoryRewardID: "2BSI-00110001", NilaiRewardID: "2BSI-0011", OldPersen: pnrFloat(25), NewPersen: pnrFloat(30), ChangedBy: "ADM-001"},
	}
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(awal, nil)
	repo.On("ListNilaiRewardByBankReward", "BSI-001", 1).Return([]models.NilaiRewardBank{eksternal, *awal}, nil)
	repo.On("ListHistoryByNilaiReward", []string{"1BSI-0011", "2BSI-0011"}).Return(history, nil).Once()

	hasil, err := svc.DetailNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSI-001", hasil.BankID)
		assert.Equal(t, 1, hasil.RewardID)
		assert.Equal(t, models.RewardEnumUang, hasil.NamaReward)
		assert.Equal(t, models.SatuanRewardEnumRp, hasil.Satuan)
		assert.Equal(t, []services.PersentaseLevelRow{
			{LevelUser: models.LevelEksternal, PersenBagiHasil: 100},
			{LevelUser: models.LevelNasabah, PersenBagiHasil: 30},
		}, hasil.Persentase)
		assert.Equal(t, history, hasil.History)
	}
	repo.AssertNotCalled(t, "FindBank", mock.Anything)
	repo.AssertExpectations(t)
}

func TestHistoryNilaiReward_BSUAnakDitolak(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)

	hasil, err := svc.HistoryNilaiReward("2BSI-0011", aksesAdminBank("BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, rwdKode(t, err))
	repo.AssertNotCalled(t, "FindBank", mock.Anything)
	repo.AssertNotCalled(t, "ListHistoryByNilaiReward", mock.Anything)
}

func TestHistoryNilaiReward_RepoError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("ListHistoryByNilaiReward", []string{"2BSI-0011"}).Return(nil, errors.New("db error"))

	hasil, err := svc.HistoryNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestHistoryNilaiReward_Sukses(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	history := []repositories.HistoryNilaiRewardRow{
		{HistoryRewardID: "2BSI-00110001", NilaiRewardID: "2BSI-0011", OldPersen: pnrFloat(25), NewPersen: pnrFloat(30), ChangedBy: "ADM-001"},
	}
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("ListHistoryByNilaiReward", []string{"2BSI-0011"}).Return(history, nil).Once()

	hasil, err := svc.HistoryNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	assert.Equal(t, history, hasil)
	repo.AssertExpectations(t)
}

func TestHapusNilaiReward_AdminTidakAda(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("", nil)

	err := svc.HapusNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.Equal(t, http.StatusForbidden, rwdKode(t, err))
	repo.AssertNotCalled(t, "NonaktifkanNilaiRewardPasangan", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestHapusNilaiReward_NonaktifkanError(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	repo.On("FindNilaiRewardByID", "2BSI-0011").Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("NonaktifkanNilaiRewardPasangan", mock.Anything, "BSI-001", 1, "ADM-001").Return(int64(0), errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.HapusNilaiReward("2BSI-0011", aksesAdminBank("BSI-001"))

	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahNilaiReward_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSI-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.TambahNilaiReward("BSI-XXX", persenNasabah(40), aksesAdminBank("BSI-XXX"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, rwdKode(t, err))
}

func TestTambahNilaiReward_BSUDitolak(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSU-001").Return(bankBSUAnak(ptrStr("BSI-001")), nil)

	hasil, err := svc.TambahNilaiReward("BSU-001", persenNasabah(40), aksesAdminBank("BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, rwdKode(t, err))
	repo.AssertNotCalled(t, "FindReward", mock.Anything, mock.Anything)
}

func TestTambahNilaiReward_PersentaseTidakValid(t *testing.T) {
	cases := map[string][]services.PersentaseInput{
		"level tidak dikenal": {{LevelUser: models.LevelUser("vip"), PersenBagiHasil: 40}},
		"nasabah dua kali":    {{LevelUser: models.LevelNasabah, PersenBagiHasil: 40}, {LevelUser: models.LevelNasabah, PersenBagiHasil: 50}},
		"persen nol":          {{LevelUser: models.LevelNasabah, PersenBagiHasil: 0}},
		"persen lebih 100":    {{LevelUser: models.LevelNasabah, PersenBagiHasil: 101}},
		"hanya eksternal":     {{LevelUser: models.LevelEksternal, PersenBagiHasil: 100}},
	}
	for nama, persentase := range cases {
		t.Run(nama, func(t *testing.T) {
			_, repo, svc := newRewardSvc(t)
			repo.On("FindBank", "BSI-001").Return(bankBSI(), nil)

			hasil, err := svc.TambahNilaiReward("BSI-001",
				services.NilaiRewardInput{RewardID: 1, Persentase: persentase}, aksesAdminBank("BSI-001"))

			assert.Nil(t, hasil)
			assert.Equal(t, http.StatusBadRequest, rwdKode(t, err))
			repo.AssertNotCalled(t, "FindReward", mock.Anything, mock.Anything)
		})
	}
}

func TestTambahNilaiReward_RewardTidakDitemukan(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSI-001").Return(bankBSI(), nil)
	repo.On("FindReward", mock.Anything, 1).Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(40), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, rwdKode(t, err))
}

func TestTambahNilaiReward_FindRewardError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSI-001").Return(bankBSI(), nil)
	repo.On("FindReward", mock.Anything, 1).Return(nil, errors.New("db error"))

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(40), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

func TestTambahNilaiReward_AdminError(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindBank", "BSI-001").Return(bankBSI(), nil)
	repo.On("FindReward", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("", errors.New("db error"))

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(40), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
}

// Bank BSI-001, reward 1, admin ADM-001 sudah lolos validasi; FindNilaiReward diatur per test.
func siapTambahNilaiReward(repo *mocks.MockRewardRepo, reward *models.Reward) {
	repo.On("FindBank", "BSI-001").Return(bankBSI(), nil)
	repo.On("FindReward", mock.Anything, reward.RewardID).Return(reward, nil)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
}

func expectNilaiRewardDibuat(repo *mocks.MockRewardRepo, id string, level models.LevelUser, persen float64) {
	repo.On("CreateNilaiReward", mock.Anything, mock.MatchedBy(func(r *models.NilaiRewardBank) bool {
		return r.NilaiRewardID == id && r.BankID == "BSI-001" && r.LevelUser == level &&
			r.PersenBagiHasil == persen && r.CreatedBy != nil && *r.CreatedBy == "ADM-001"
	})).Return(nil).Once()
}

func TestTambahNilaiReward_BaruSemua(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	siapTambahNilaiReward(repo, rewardUang())
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).Return(nil, gorm.ErrRecordNotFound)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelEksternal).Return(nil, gorm.ErrRecordNotFound)
	expectNilaiRewardDibuat(repo, "2BSI-0011", models.LevelNasabah, 40)
	expectNilaiRewardDibuat(repo, "1BSI-0011", models.LevelEksternal, 100)
	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(40), aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	if assert.Len(t, hasil, 2) {
		assert.Equal(t, "2BSI-0011", hasil[0].NilaiRewardID)
		assert.Equal(t, 40.0, hasil[0].PersenBagiHasil)
		assert.Equal(t, "1BSI-0011", hasil[1].NilaiRewardID)
		assert.Equal(t, 100.0, hasil[1].PersenBagiHasil, "eksternal selalu 100%")
		for _, h := range hasil {
			if assert.NotNil(t, h.Reward) {
				assert.Equal(t, 1, h.Reward.RewardID)
			}
		}
	}
	repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahNilaiReward_EksternalSudahAktif(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	siapTambahNilaiReward(repo, rewardUang())
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).Return(nil, gorm.ErrRecordNotFound)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelEksternal).
		Return(nilaiRewardAktif("1BSI-0011", "BSI-001", 1, models.LevelEksternal), nil)
	expectNilaiRewardDibuat(repo, "2BSI-0011", models.LevelNasabah, 40)
	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(40), aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	if assert.Len(t, hasil, 1, "eksternal yang masih aktif tidak ikut dibuat ulang") {
		assert.Equal(t, "2BSI-0011", hasil[0].NilaiRewardID)
	}
	repo.AssertNotCalled(t, "SaveNilaiReward", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestTambahNilaiReward_AktifkanKembaliPersenSama(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	nonaktif := nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah)
	nonaktif.IsActive = false
	siapTambahNilaiReward(repo, rewardUang())
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).Return(nonaktif, nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelEksternal).
		Return(nilaiRewardAktif("1BSI-0011", "BSI-001", 1, models.LevelEksternal), nil)
	repo.On("SaveNilaiReward", mock.Anything, nonaktif).Return(nil).Once()
	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(30), aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	assert.Len(t, hasil, 1)
	assert.True(t, nonaktif.IsActive)
	assert.Equal(t, "ADM-001", *nonaktif.UpdatedBy)
	repo.AssertNotCalled(t, "CreateHistory", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahNilaiReward_CekNilaiError(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	siapTambahNilaiReward(repo, rewardUang())
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).Return(nil, errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(40), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahNilaiReward_CekEksternalError(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	siapTambahNilaiReward(repo, rewardUang())
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).Return(nil, gorm.ErrRecordNotFound)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelEksternal).Return(nil, errors.New("db error"))
	expectNilaiRewardDibuat(repo, "2BSI-0011", models.LevelNasabah, 40)
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(40), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	assert.Contains(t, err.Error(), "eksternal")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahNilaiReward_CreateError(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	siapTambahNilaiReward(repo, rewardUang())
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateNilaiReward", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	hasil, err := svc.TambahNilaiReward("BSI-001", persenNasabah(40), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahNilaiReward_GenerateIDGagal(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	siapTambahNilaiReward(repo, &models.Reward{RewardID: 12, NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnumRp})
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 12, models.LevelNasabah).Return(nil, gorm.ErrRecordNotFound)
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	in := persenNasabah(40)
	in.RewardID = 12
	hasil, err := svc.TambahNilaiReward("BSI-001", in, aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	assert.Contains(t, err.Error(), "Gagal generate nilai reward ID")
	repo.AssertNotCalled(t, "CreateNilaiReward", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUbahNilaiReward_IDTidakValid(t *testing.T) {
	_, repo, svc := newRewardSvc(t)

	hasil, err := svc.UbahNilaiReward("BSI-001", "x", persenNasabah(45), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, rwdKode(t, err))
	repo.AssertNotCalled(t, "FindAdminIDByUserBank", mock.Anything, mock.Anything)
}

func TestUbahNilaiReward_PersentaseTidakValid(t *testing.T) {
	_, repo, svc := newRewardSvc(t)

	hasil, err := svc.UbahNilaiReward("BSI-001", "1", services.NilaiRewardInput{}, aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, rwdKode(t, err))
	repo.AssertNotCalled(t, "FindAdminIDByUserBank", mock.Anything, mock.Anything)
}

func TestUbahNilaiReward_AdminTidakAda(t *testing.T) {
	_, repo, svc := newRewardSvc(t)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("", nil)

	hasil, err := svc.UbahNilaiReward("BSI-001", "1", persenNasabah(45), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, rwdKode(t, err))
	repo.AssertNotCalled(t, "FindNilaiReward", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestUbahNilaiReward_TidakDitemukan(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).Return(nil, gorm.ErrRecordNotFound)
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	hasil, err := svc.UbahNilaiReward("BSI-001", "1", persenNasabah(45), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, rwdKode(t, err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUbahNilaiReward_FindError(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).Return(nil, errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	hasil, err := svc.UbahNilaiReward("BSI-001", "1", persenNasabah(45), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUbahNilaiReward_PersenSama(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).
		Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	hasil, err := svc.UbahNilaiReward("BSI-001", "1", persenNasabah(30), aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	if assert.Len(t, hasil, 1) {
		assert.Equal(t, 30.0, hasil[0].PersenBagiHasil)
	}
	repo.AssertNotCalled(t, "SaveNilaiReward", mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "CreateHistory", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUbahNilaiReward_PersenBerubah(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).
		Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("SaveNilaiReward", mock.Anything, mock.MatchedBy(func(r *models.NilaiRewardBank) bool {
		return r.PersenBagiHasil == 45 && r.UpdatedBy != nil && *r.UpdatedBy == "ADM-001"
	})).Return(nil).Once()
	repo.On("CreateHistory", mock.Anything, mock.MatchedBy(func(h *models.HistoryNilaiReward) bool {
		return h.HistoryRewardID != "" && h.NilaiRewardID == "2BSI-0011" &&
			*h.OldPersen == 30 && *h.NewPersen == 45 && *h.ChangedBy == "ADM-001"
	})).Return(nil).Once()
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"history_reward_id"}))
	dbMock.ExpectCommit()

	hasil, err := svc.UbahNilaiReward("BSI-001", "1", persenNasabah(45), aksesAdminBank("BSI-001"))

	assert.NoError(t, err)
	if assert.Len(t, hasil, 1) {
		assert.Equal(t, 45.0, hasil[0].PersenBagiHasil)
	}
	repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUbahNilaiReward_SaveError(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).
		Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("SaveNilaiReward", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	hasil, err := svc.UbahNilaiReward("BSI-001", "1", persenNasabah(45), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	repo.AssertNotCalled(t, "CreateHistory", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUbahNilaiReward_CreateHistoryError(t *testing.T) {
	dbMock, repo, svc := newRewardSvc(t)
	repo.On("FindAdminIDByUserBank", "USR-ADM-001", "BSI-001").Return("ADM-001", nil)
	repo.On("FindNilaiReward", mock.Anything, "BSI-001", 1, models.LevelNasabah).
		Return(nilaiRewardAktif("2BSI-0011", "BSI-001", 1, models.LevelNasabah), nil)
	repo.On("SaveNilaiReward", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateHistory", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"history_reward_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.UbahNilaiReward("BSI-001", "1", persenNasabah(45), aksesAdminBank("BSI-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, rwdKode(t, err))
	assert.Contains(t, err.Error(), "history")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}
