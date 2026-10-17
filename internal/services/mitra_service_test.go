package services_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

const (
	mtrBankBSI = "101002001"
	mtrBankBSM = "201002001"
	mtrBankBSU = "301002001"
	mtrID      = "MP10100200101"
)

func newMtrSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockMitraRepo, *services.MitraService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockMitraRepo)
	return dbMock, repo, services.NewMitraService(gdb, repo)
}

func mtrBank(bankID string, jenis models.JenisBank) *models.BankSampah {
	return &models.BankSampah{BankID: bankID, NamaBank: "Bank " + bankID, JenisBank: jenis, IsActive: true}
}

func mtrMitra(mitraID, nama string, aktif bool) *models.MitraPengepul {
	return &models.MitraPengepul{
		MitraID: mitraID, NamaMitra: nama, BankID: mtrBankBSI, IsActive: aktif,
		CreatedAt: time.Date(2026, time.September, 1, 8, 0, 0, 0, time.Local),
	}
}

func expectTxMitraGagal(m sqlmock.Sqlmock) {
	m.ExpectBegin()
	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"mitra_id"}))
	m.ExpectRollback()
}

func mtrKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func TestMitra_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(nil, errors.New("not found"))

	hasil, err := svc.ListMitra(mtrBankBSI)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, mtrKode(t, err))
}

func TestMitra_BSUDitolakDiSemuaEndpoint(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		_, repo, svc := newMtrSvc(t)
		repo.On("GetBankByID", mtrBankBSU).Return(mtrBank(mtrBankBSU, models.BSU), nil)

		hasil, err := svc.ListMitra(mtrBankBSU)

		assert.Nil(t, hasil)
		assert.Equal(t, http.StatusForbidden, mtrKode(t, err))
		assert.Contains(t, err.Error(), "BSI dan BSM")
	})

	t.Run("create", func(t *testing.T) {
		_, repo, svc := newMtrSvc(t)
		repo.On("GetBankByID", mtrBankBSU).Return(mtrBank(mtrBankBSU, models.BSU), nil)

		hasil, err := svc.CreateMitra(mtrBankBSU, "PT Sumber Rejeki")

		assert.Nil(t, hasil)
		assert.Equal(t, http.StatusForbidden, mtrKode(t, err))
	})

	t.Run("update", func(t *testing.T) {
		_, repo, svc := newMtrSvc(t)
		repo.On("GetBankByID", mtrBankBSU).Return(mtrBank(mtrBankBSU, models.BSU), nil)

		hasil, err := svc.UpdateMitra(mtrBankBSU, mtrID, nil, nil)

		assert.Nil(t, hasil)
		assert.Equal(t, http.StatusForbidden, mtrKode(t, err))
	})

	t.Run("delete", func(t *testing.T) {
		_, repo, svc := newMtrSvc(t)
		repo.On("GetBankByID", mtrBankBSU).Return(mtrBank(mtrBankBSU, models.BSU), nil)

		err := svc.DeleteMitra(mtrBankBSU, mtrID)

		assert.Equal(t, http.StatusForbidden, mtrKode(t, err))
	})
}

func TestListMitra_RepoError(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("ListByBank", mtrBankBSI).Return(nil, errors.New("db error"))

	hasil, err := svc.ListMitra(mtrBankBSI)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
}

func TestListMitra_KosongNonNil(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("ListByBank", mtrBankBSI).Return([]models.MitraPengepul{}, nil)

	hasil, err := svc.ListMitra(mtrBankBSI)

	assert.NoError(t, err)
	assert.NotNil(t, hasil, "slice kosong harus non-nil supaya JSON-nya [] bukan null")
	assert.Len(t, hasil, 0)
}

func TestListMitra_Mapping(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	list := []models.MitraPengepul{
		*mtrMitra(mtrID, "PT Sumber Rejeki", true),
		*mtrMitra("MP10100200102", "CV Berkah", false),
	}
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("ListByBank", mtrBankBSI).Return(list, nil)

	hasil, err := svc.ListMitra(mtrBankBSI)

	assert.NoError(t, err)
	if assert.Len(t, hasil, 2) {
		assert.Equal(t, mtrID, hasil[0].MitraID)
		assert.Equal(t, "PT Sumber Rejeki", hasil[0].NamaMitra)
		assert.Equal(t, mtrBankBSI, hasil[0].BankID)
		assert.True(t, hasil[0].IsActive)
		assert.False(t, hasil[1].IsActive, "mitra nonaktif tetap ikut di daftar")
	}
}

func TestListMitra_BSMBoleh(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSM).Return(mtrBank(mtrBankBSM, models.BSM), nil)
	repo.On("ListByBank", mtrBankBSM).Return([]models.MitraPengepul{}, nil)

	_, err := svc.ListMitra(mtrBankBSM)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCreateMitra_NamaKosong(t *testing.T) {
	for _, nama := range []string{"", "   ", "\t\n"} {
		_, repo, svc := newMtrSvc(t)
		repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)

		hasil, err := svc.CreateMitra(mtrBankBSI, nama)

		assert.Nil(t, hasil)
		assert.Equal(t, http.StatusBadRequest, mtrKode(t, err))
		assert.Contains(t, err.Error(), "wajib diisi")
		repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	}
}

func TestCreateMitra_NamaSerupaDitolak(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, "ptabc").
		Return(mtrMitra(mtrID, "pt abc", true), nil)

	hasil, err := svc.CreateMitra(mtrBankBSI, "PT. ABC")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, mtrKode(t, err))
	assert.Contains(t, err.Error(), "pt abc", "pesan menyebut nama mitra yang sudah ada")
	repo.AssertExpectations(t)
}

func TestCreateMitra_NormalisasiNama(t *testing.T) {
	kasus := []struct {
		masukan, normal string
	}{
		{"PT Sumber Rejeki", "ptsumberrejeki"},
		{"CV. Berkah-Jaya", "cvberkahjaya"},
		{"  UD   Maju  ", "udmaju"},
		{"Toko 99 Plastik", "toko99plastik"},
	}
	for _, k := range kasus {
		t.Run(k.masukan, func(t *testing.T) {
			dbMock, repo, svc := newMtrSvc(t)
			repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
			repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, k.normal).
				Return(nil, gorm.ErrRecordNotFound)
			repo.On("Create", mock.Anything, mock.Anything).Return(nil)
			expectTx(dbMock, 1)

			_, err := svc.CreateMitra(mtrBankBSI, k.masukan)

			assert.NoError(t, err)
			repo.AssertExpectations(t)
		})
	}
}

func TestCreateMitra_GenerateIDGagal(t *testing.T) {
	dbMock, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, mock.Anything).
		Return(nil, gorm.ErrRecordNotFound)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("lock timeout"))
	dbMock.ExpectRollback()

	hasil, err := svc.CreateMitra(mtrBankBSI, "PT Sumber Rejeki")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
	assert.Contains(t, err.Error(), "generate mitra ID")
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestCreateMitra_DuplicateKeyJadiBadRequest(t *testing.T) {
	dbMock, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, mock.Anything).
		Return(nil, gorm.ErrRecordNotFound)
	repo.On("Create", mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey)
	expectTxMitraGagal(dbMock)

	hasil, err := svc.CreateMitra(mtrBankBSI, "PT Sumber Rejeki")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, mtrKode(t, err))
	assert.Contains(t, err.Error(), "sudah terdaftar")
}

func TestCreateMitra_CreateError(t *testing.T) {
	dbMock, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, mock.Anything).
		Return(nil, gorm.ErrRecordNotFound)
	repo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxMitraGagal(dbMock)

	hasil, err := svc.CreateMitra(mtrBankBSI, "PT Sumber Rejeki")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
	assert.Contains(t, err.Error(), "membuat data mitra")
}

func TestCreateMitra_Sukses(t *testing.T) {
	dbMock, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, "ptsumberrejeki").
		Return(nil, gorm.ErrRecordNotFound)

	var dibuat *models.MitraPengepul
	repo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { dibuat = args.Get(1).(*models.MitraPengepul) }).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.CreateMitra(mtrBankBSI, "  PT Sumber Rejeki  ")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, mtrID, hasil.MitraID, "mitra_id = MP + bank_id + urutan")
		assert.Equal(t, "PT Sumber Rejeki", hasil.NamaMitra, "nama disimpan hasil trim, bukan hasil normalisasi")
		assert.Equal(t, mtrBankBSI, hasil.BankID)
		assert.True(t, hasil.IsActive, "mitra baru langsung aktif")
	}
	assert.NotNil(t, dibuat)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUpdateMitra_TidakDitemukan(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(nil, errors.New("not found"))

	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, nil, nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, mtrKode(t, err))
}

func TestUpdateMitra_NamaKosong(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mtrMitra(mtrID, "PT Lama", true), nil)

	kosong := "   "
	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, &kosong, nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, mtrKode(t, err))
	assert.Contains(t, err.Error(), "tidak boleh kosong")
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdateMitra_NamaBentrokMitraLain(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mtrMitra(mtrID, "PT Lama", true), nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, "ptabc").
		Return(mtrMitra("MP10100200102", "PT ABC", true), nil)

	nama := "PT ABC"
	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, &nama, nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, mtrKode(t, err))
	assert.Contains(t, err.Error(), "PT ABC")
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestUpdateMitra_NamaSamaDenganDirinyaSendiriBoleh(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "pt abc", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, "ptabc").Return(mitra, nil)
	repo.On("Update", mock.Anything, mitra).Return(nil)

	nama := "PT. ABC"
	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, &nama, nil)

	assert.NoError(t, err)
	assert.Equal(t, "PT. ABC", hasil.NamaMitra)
}

func TestUpdateMitra_HanyaNama(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, "ptbaru").
		Return(nil, gorm.ErrRecordNotFound)
	repo.On("Update", mock.Anything, mitra).Return(nil)

	nama := "  PT Baru  "
	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, &nama, nil)

	assert.NoError(t, err)
	assert.Equal(t, "PT Baru", hasil.NamaMitra, "nama disimpan hasil trim")
	assert.True(t, hasil.IsActive, "is_active tidak dikirim, nilainya dipertahankan")
}

func TestUpdateMitra_HanyaIsActive(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("Update", mock.Anything, mitra).Return(nil)

	nonaktif := false
	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, nil, &nonaktif)

	assert.NoError(t, err)
	assert.False(t, hasil.IsActive)
	assert.Equal(t, "PT Lama", hasil.NamaMitra, "nama tidak dikirim, nilainya dipertahankan")
	repo.AssertNotCalled(t, "FindByNormalizedName", mock.Anything, mock.Anything, mock.Anything)
}

func TestUpdateMitra_KeduanyaSekaligus(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, "ptbaru").
		Return(nil, gorm.ErrRecordNotFound)
	repo.On("Update", mock.Anything, mitra).Return(nil)

	nama, nonaktif := "PT Baru", false
	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, &nama, &nonaktif)

	assert.NoError(t, err)
	assert.Equal(t, "PT Baru", hasil.NamaMitra)
	assert.False(t, hasil.IsActive)
}

func TestUpdateMitra_DuplicateKeyJadiBadRequest(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("FindByNormalizedName", mock.Anything, mtrBankBSI, mock.Anything).
		Return(nil, gorm.ErrRecordNotFound)
	repo.On("Update", mock.Anything, mitra).Return(gorm.ErrDuplicatedKey)

	nama := "PT Baru"
	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, &nama, nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, mtrKode(t, err))
	assert.Contains(t, err.Error(), "sudah terdaftar")
}

func TestUpdateMitra_UpdateError(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("Update", mock.Anything, mitra).Return(errors.New("db error"))

	aktif := true
	hasil, err := svc.UpdateMitra(mtrBankBSI, mtrID, nil, &aktif)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
}

func TestDeleteMitra_TidakDitemukan(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(nil, errors.New("not found"))

	err := svc.DeleteMitra(mtrBankBSI, mtrID)

	assert.Equal(t, http.StatusNotFound, mtrKode(t, err))
}

func TestDeleteMitra_CekHistoriError(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mtrMitra(mtrID, "PT Lama", true), nil)
	repo.On("IsUsedInPenjualan", mock.Anything, mtrID).Return(false, errors.New("db error"))

	err := svc.DeleteMitra(mtrBankBSI, mtrID)

	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
	assert.Contains(t, err.Error(), "histori penjualan")
}

func TestDeleteMitra_SudahDipakaiJadiSoftDelete(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("IsUsedInPenjualan", mock.Anything, mtrID).Return(true, nil)
	repo.On("Update", mock.Anything, mitra).Return(nil)

	err := svc.DeleteMitra(mtrBankBSI, mtrID)

	assert.NoError(t, err)
	assert.False(t, mitra.IsActive, "soft delete = is_active jadi false")
	repo.AssertNotCalled(t, "Delete", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestDeleteMitra_BelumDipakaiJadiHardDelete(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("IsUsedInPenjualan", mock.Anything, mtrID).Return(false, nil)
	repo.On("Delete", mock.Anything, mitra).Return(nil)

	err := svc.DeleteMitra(mtrBankBSI, mtrID)

	assert.NoError(t, err)
	assert.True(t, mitra.IsActive, "hard delete tidak menyentuh is_active")
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestDeleteMitra_SoftDeleteError(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("IsUsedInPenjualan", mock.Anything, mtrID).Return(true, nil)
	repo.On("Update", mock.Anything, mitra).Return(errors.New("db error"))

	err := svc.DeleteMitra(mtrBankBSI, mtrID)

	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
	assert.Contains(t, err.Error(), "menghapus data mitra")
}

func TestDeleteMitra_HardDeleteError(t *testing.T) {
	_, repo, svc := newMtrSvc(t)
	mitra := mtrMitra(mtrID, "PT Lama", true)
	repo.On("GetBankByID", mtrBankBSI).Return(mtrBank(mtrBankBSI, models.BSI), nil)
	repo.On("GetByID", mock.Anything, mtrBankBSI, mtrID).Return(mitra, nil)
	repo.On("IsUsedInPenjualan", mock.Anything, mtrID).Return(false, nil)
	repo.On("Delete", mock.Anything, mitra).Return(errors.New("db error"))

	err := svc.DeleteMitra(mtrBankBSI, mtrID)

	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
}

// resolveOrCreateMitra hanya dipanggil dari SubmitPenjualan, jadi diuji lewat situ dengan daftar item kosong
// supaya alurnya langsung sampai ke penentuan mitra.
func siapSubmitMitra(dbMock sqlmock.Sqlmock, repo *mocks.MockPenjualanRepo) {
	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"penjualan_id"}))
}

func expectPenjualanDenganMitra(repo *mocks.MockPenjualanRepo, mitraID string) {
	repo.On("CreatePenjualan", mock.Anything, mock.MatchedBy(func(p *models.Penjualan) bool {
		return p.MitraID == mitraID
	})).Return(nil).Once()
	repo.On("CreateDetailPenjualan", mock.Anything, mock.Anything).Return(nil).Once()
}

func TestResolveMitraPenjualan_PakaiMitraID(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	mitraRepo.On("GetByID", mock.Anything, "BSM-001", mtrID).Return(mtrMitra(mtrID, "PT Sumber Rejeki", true), nil)
	expectPenjualanDenganMitra(repo, mtrID)
	dbMock.ExpectCommit()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, mtrID, "", "", nil)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	mitraRepo.AssertNotCalled(t, "FindByNormalizedName", mock.Anything, mock.Anything, mock.Anything)
	mitraRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestResolveMitraPenjualan_MitraIDTidakDitemukan(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	mitraRepo.On("GetByID", mock.Anything, "BSM-001", mtrID).Return(nil, errors.New("not found"))
	dbMock.ExpectRollback()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, mtrID, "", "", nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, mtrKode(t, err))
	assert.Contains(t, err.Error(), "Mitra tidak ditemukan")
	repo.AssertNotCalled(t, "CreatePenjualan", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestResolveMitraPenjualan_TanpaMitraIDDanNama(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	dbMock.ExpectRollback()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "   ", "", nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, mtrKode(t, err))
	assert.Contains(t, err.Error(), "mitra_id atau nama_mitra wajib diisi")
	mitraRepo.AssertNotCalled(t, "FindByNormalizedName", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "CreatePenjualan", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestResolveMitraPenjualan_NamaSerupaPakaiYangAda(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", "ptsumberrejeki").
		Return(mtrMitra(mtrID, "PT Sumber Rejeki", true), nil).Once()
	expectPenjualanDenganMitra(repo, mtrID)
	dbMock.ExpectCommit()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "PT. Sumber Rejeki", "", nil)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	mitraRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	mitraRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestResolveMitraPenjualan_NamaBaruDibuat(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", "cvberkah").Return(nil, gorm.ErrRecordNotFound).Once()

	var dibuat *models.MitraPengepul
	mitraRepo.On("Create", mock.Anything, mock.MatchedBy(func(m *models.MitraPengepul) bool {
		return m.MitraID != "" && m.NamaMitra == "CV Berkah" && m.BankID == "BSM-001" && m.IsActive
	})).Run(func(args mock.Arguments) {
		dibuat = args.Get(1).(*models.MitraPengepul)
	}).Return(nil).Once()
	repo.On("CreatePenjualan", mock.Anything, mock.MatchedBy(func(p *models.Penjualan) bool {
		return dibuat != nil && p.MitraID == dibuat.MitraID
	})).Return(nil).Once()
	repo.On("CreateDetailPenjualan", mock.Anything, mock.Anything).Return(nil).Once()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"mitra_id"}))
	dbMock.ExpectCommit()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "  CV Berkah  ", "", nil)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	mitraRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestResolveMitraPenjualan_GenerateIDGagal(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", "cvberkah").Return(nil, gorm.ErrRecordNotFound)
	dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("lock timeout"))
	dbMock.ExpectRollback()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "CV Berkah", "", nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
	assert.Contains(t, err.Error(), "generate mitra ID")
	mitraRepo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestResolveMitraPenjualan_DuplikatPakaiYangAda(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", "cvberkah").Return(nil, gorm.ErrRecordNotFound).Once()
	mitraRepo.On("Create", mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey).Once()
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", "cvberkah").
		Return(mtrMitra(mtrID, "CV Berkah", true), nil).Once()
	expectPenjualanDenganMitra(repo, mtrID)
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"mitra_id"}))
	dbMock.ExpectCommit()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "CV Berkah", "", nil)

	assert.NoError(t, err, "mitra yang keburu dibuat request lain dipakai, penjualan tetap jalan")
	assert.NotNil(t, hasil)
	mitraRepo.AssertExpectations(t)
	repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestResolveMitraPenjualan_DuplikatTapiTidakKetemu(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", "cvberkah").Return(nil, gorm.ErrRecordNotFound).Twice()
	mitraRepo.On("Create", mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey).Once()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"mitra_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "CV Berkah", "", nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
	assert.Contains(t, err.Error(), "membuat data mitra baru")
	repo.AssertNotCalled(t, "CreatePenjualan", mock.Anything, mock.Anything)
	mitraRepo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestResolveMitraPenjualan_CreateError(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)
	siapSubmitMitra(dbMock, repo)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", "cvberkah").Return(nil, gorm.ErrRecordNotFound).Once()
	mitraRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"mitra_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "CV Berkah", "", nil)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, mtrKode(t, err))
	repo.AssertNotCalled(t, "CreatePenjualan", mock.Anything, mock.Anything)
	mitraRepo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}
