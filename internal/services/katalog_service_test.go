package services_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func newKtlSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockKatalogRepo, services.KatalogService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockKatalogRepo)
	return dbMock, repo, services.NewKatalogService(gdb, repo)
}

func ktlBank(bankID string, jenis models.JenisBank) *models.BankSampah {
	return &models.BankSampah{BankID: bankID, NamaBank: "Bank " + bankID, JenisBank: jenis, IsActive: true}
}

func ktlBSU(parentID string) *models.BankSampah {
	bank := ktlBank("BSU-001", models.BSU)
	if parentID != "" {
		bank.ParentBankID = ptrStr(parentID)
	}
	return bank
}

func ktlKatalog(sampahID, bankID string) *models.KatalogSampah {
	return &models.KatalogSampah{
		SampahID:        sampahID,
		SarokID:         12,
		BankID:          bankID,
		KategoriID:      3,
		RewardID:        1,
		PhotoURL:        "https://cdn/botol.png",
		SyaratPemilahan: "Bersih dan kering",
		Sarok:           models.Sampah{SarokID: 12, NamaSampah: "Botol Plastik", SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}},
		Kategori:        models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"},
		Reward:          models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang},
	}
}

func ktlAkses(bankID string, role models.RoleAdmin) services.AksesKatalogInput {
	return services.AksesKatalogInput{ActorBankID: bankID, ActorRole: role}
}

func ktlTambahInput() services.TambahKatalogInput {
	return services.TambahKatalogInput{
		BankID:          "BSI-001",
		NamaSampah:      "Botol Plastik",
		SatuanID:        1,
		KategoriID:      3,
		RewardID:        1,
		SyaratPemilahan: "Bersih dan kering",
		FotoURL:         "https://cdn/botol.png",
	}
}

func ktlKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func TestListMasterSampah_RepoError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("ListMasterSampah", "botol").Return(nil, errors.New("db error"))

	hasil, err := svc.ListMasterSampah("botol")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestListMasterSampah_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	master := []models.Sampah{
		{SarokID: 12, NamaSampah: "Botol Plastik", SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}},
		{SarokID: 13, NamaSampah: "Kardus", SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}},
	}
	repo.On("ListMasterSampah", "").Return(master, nil)

	hasil, err := svc.ListMasterSampah("")

	assert.NoError(t, err)
	assert.Len(t, hasil, 2)
	assert.Equal(t, "Botol Plastik", hasil[0].NamaSampah)
}

func TestTambahKatalog_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(nil, errors.New("not found"))

	hasil, err := svc.TambahKatalog(ktlTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestTambahKatalog_BSUDitolak(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSU-001").Return(ktlBSU("BSI-001"), nil)

	in := ktlTambahInput()
	in.BankID = "BSU-001"
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, ktlKode(t, err))
	assert.Contains(t, err.Error(), "BSI induk")
}

func TestTambahKatalog_SarokIDTidakValid(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 99).Return(nil, errors.New("not found"))

	in := ktlTambahInput()
	in.SarokID = 99
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, ktlKode(t, err))
	assert.Contains(t, err.Error(), "sarok_id")
}

func TestTambahKatalog_SarokIDSatuanBedaPakaiMasterSatuanItu(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "2").Return(&models.Satuan{SatuanID: 2, Satuan: "pcs"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 13).Return(&models.Sampah{SarokID: 13, NamaSampah: "Kardus", SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindOrCreateMasterSampah", mock.Anything, "Kardus", 2).
		Return(&models.Sampah{SarokID: 20, NamaSampah: "Kardus", SatuanID: 2, Satuan: models.Satuan{SatuanID: 2, Satuan: "pcs"}}, nil)
	repo.On("FindKatalogSejenis", 20, "BSI-001").Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateKatalog", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	in := ktlTambahInput()
	in.NamaSampah = "Kardus"
	in.SarokID = 13
	in.SatuanID = 2
	hasil, err := svc.TambahKatalog(in)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, 20, hasil.Katalog.SarokID)
	}
}

func TestTambahKatalog_SarokIDSatuanBedaMasterError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "2").Return(&models.Satuan{SatuanID: 2, Satuan: "pcs"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 13).Return(&models.Sampah{SarokID: 13, NamaSampah: "Kardus", SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindOrCreateMasterSampah", mock.Anything, "Kardus", 2).Return(nil, errors.New("db error"))

	in := ktlTambahInput()
	in.SarokID = 13
	in.SatuanID = 2
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "master sampah")
	repo.AssertNotCalled(t, "CreateKatalog", mock.Anything, mock.Anything)
}

func TestTambahKatalog_SatuanTidakValid(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "99").Return(nil, gorm.ErrRecordNotFound)

	in := ktlTambahInput()
	in.SatuanID = 99
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, ktlKode(t, err))
	assert.Contains(t, err.Error(), "satuan")
	repo.AssertNotCalled(t, "FindMasterSampah", mock.Anything)
	repo.AssertNotCalled(t, "FindOrCreateMasterSampah", mock.Anything, mock.Anything, mock.Anything)
}

func TestTambahKatalog_CekSatuanError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(nil, errors.New("db error"))

	hasil, err := svc.TambahKatalog(ktlTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	repo.AssertNotCalled(t, "AdaNilaiRewardAktif", mock.Anything, mock.Anything)
}

func TestTambahKatalog_SarokIDDipakaiApaAdanya(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, NamaSampah: "Botol Plastik", SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateKatalog", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	in := ktlTambahInput()
	in.SarokID = 12
	hasil, err := svc.TambahKatalog(in)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertNotCalled(t, "FindOrCreateMasterSampah", mock.Anything, mock.Anything, mock.Anything)
}

func TestTambahKatalog_TanpaSarokIDBuatMasterSampah(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindOrCreateMasterSampah", mock.Anything, "Botol Plastik", 1).
		Return(&models.Sampah{SarokID: 7, NamaSampah: "Botol Plastik", SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateKatalog", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	hasil, err := svc.TambahKatalog(ktlTambahInput())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, 7, hasil.Katalog.SarokID)
		assert.Equal(t, "BSI-0010071", hasil.Katalog.SampahID, "sampah_id = bank + sarok(3 digit) + reward")
	}
}

func TestTambahKatalog_MasterSampahError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindOrCreateMasterSampah", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	hasil, err := svc.TambahKatalog(ktlTambahInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "master sampah")
}

func TestTambahKatalog_GenerateIDGagal(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)

	in := ktlTambahInput()
	in.SarokID = 12
	in.RewardID = 42
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "generate sampah ID")
	repo.AssertNotCalled(t, "CreateKatalog", mock.Anything, mock.Anything)
}

func TestTambahKatalog_DuplikatJadiConflict(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateKatalog", mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey)
	expectTxRollback(dbMock)

	in := ktlTambahInput()
	in.SarokID = 12
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	assert.Contains(t, err.Error(), "sudah ada")
	repo.AssertNotCalled(t, "CreateStok", mock.Anything, mock.Anything)
}

func TestTambahKatalog_CreateKatalogError(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateKatalog", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	in := ktlTambahInput()
	in.SarokID = 12
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "membuat katalog")
}

func TestTambahKatalog_CreateStokErrorRollback(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateKatalog", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	in := ktlTambahInput()
	in.SarokID = 12
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "stok awal")
}

func TestTambahKatalog_Sukses(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSM-001").Return(ktlBank("BSM-001", models.BSM), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, NamaSampah: "Botol Plastik", SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)

	var katalogDibuat *models.KatalogSampah
	var stokDibuat *models.StokSampah
	repo.On("FindKatalogSejenis", mock.Anything, mock.Anything).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateKatalog", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { katalogDibuat = args.Get(1).(*models.KatalogSampah) }).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { stokDibuat = args.Get(1).(*models.StokSampah) }).Return(nil)
	expectTx(dbMock, 0)

	in := ktlTambahInput()
	in.BankID = "BSM-001"
	in.SarokID = 12
	hasil, err := svc.TambahKatalog(in)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSM-0010121", hasil.Katalog.SampahID)
		assert.Equal(t, "BSM-001", hasil.Katalog.BankID)
		assert.Equal(t, 3, hasil.Katalog.KategoriID)
		assert.Equal(t, 1, hasil.Katalog.RewardID)
		assert.Equal(t, "Bersih dan kering", hasil.Katalog.SyaratPemilahan)
		assert.Equal(t, "https://cdn/botol.png", hasil.Katalog.PhotoURL)
		assert.Equal(t, float64(0), hasil.Stok.Stok, "stok awal selalu 0, harga terisi saat penjualan")
		assert.Equal(t, "BSM-0010121", hasil.Stok.SampahID)
	}
	assert.NotNil(t, katalogDibuat)
	assert.NotNil(t, stokDibuat)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahKatalog_KategoriBedaDenganKatalogSejenisDitolak(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", 12, "BSI-001").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)

	in := ktlTambahInput()
	in.SarokID = 12
	in.RewardID = 2
	in.KategoriID = 5
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	assert.Contains(t, err.Error(), "Plastik")
	repo.AssertNotCalled(t, "CreateKatalog", mock.Anything, mock.Anything)
}

func TestTambahKatalog_KategoriSamaDenganKatalogSejenisDiterima(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", 12, "BSI-001").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("CreateKatalog", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	in := ktlTambahInput()
	in.SarokID = 12
	in.RewardID = 2
	in.KategoriID = 3
	hasil, err := svc.TambahKatalog(in)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSI-0010122", hasil.Katalog.SampahID)
		assert.Equal(t, 3, hasil.Katalog.KategoriID)
	}
}

func TestTambahKatalog_FotoDanSyaratKosongIkutKatalogSejenis(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", 12, "BSI-001").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	var katalogDibuat *models.KatalogSampah
	repo.On("CreateKatalog", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { katalogDibuat = args.Get(1).(*models.KatalogSampah) }).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	in := ktlTambahInput()
	in.SarokID = 12
	in.RewardID = 2
	in.FotoURL = ""
	in.SyaratPemilahan = "  "
	_, err := svc.TambahKatalog(in)

	assert.NoError(t, err)
	if assert.NotNil(t, katalogDibuat) {
		assert.Equal(t, "https://cdn/botol.png", katalogDibuat.PhotoURL)
		assert.Equal(t, "Bersih dan kering", katalogDibuat.SyaratPemilahan)
	}
}

func TestTambahKatalog_FotoDanSyaratDiRequestDipakai(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", 12, "BSI-001").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	var katalogDibuat *models.KatalogSampah
	repo.On("CreateKatalog", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { katalogDibuat = args.Get(1).(*models.KatalogSampah) }).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	in := ktlTambahInput()
	in.SarokID = 12
	in.RewardID = 2
	in.FotoURL = "https://cdn/baru.png"
	in.SyaratPemilahan = "Dipipihkan"
	_, err := svc.TambahKatalog(in)

	assert.NoError(t, err)
	if assert.NotNil(t, katalogDibuat) {
		assert.Equal(t, "https://cdn/baru.png", katalogDibuat.PhotoURL)
		assert.Equal(t, "Dipipihkan", katalogDibuat.SyaratPemilahan)
	}
}

func TestTambahKatalog_TanpaKatalogSejenisTetapKosong(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", 12, "BSI-001").Return(nil, gorm.ErrRecordNotFound)
	var katalogDibuat *models.KatalogSampah
	repo.On("CreateKatalog", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { katalogDibuat = args.Get(1).(*models.KatalogSampah) }).Return(nil)
	repo.On("CreateStok", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	in := ktlTambahInput()
	in.SarokID = 12
	in.FotoURL = ""
	in.SyaratPemilahan = ""
	_, err := svc.TambahKatalog(in)

	assert.NoError(t, err)
	if assert.NotNil(t, katalogDibuat) {
		assert.Equal(t, "", katalogDibuat.PhotoURL)
		assert.Equal(t, "", katalogDibuat.SyaratPemilahan)
	}
}

func TestTambahKatalog_CekKatalogSejenisError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("AdaNilaiRewardAktif", mock.Anything, mock.Anything).Return(true, nil)
	repo.On("FindMasterSampah", 12).Return(&models.Sampah{SarokID: 12, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}}, nil)
	repo.On("FindKatalogSejenis", 12, "BSI-001").Return(nil, errors.New("db error"))

	in := ktlTambahInput()
	in.SarokID = 12
	hasil, err := svc.TambahKatalog(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	repo.AssertNotCalled(t, "CreateKatalog", mock.Anything, mock.Anything)
}

func ktlEditInput(bankID string, role models.RoleAdmin) services.EditKatalogInput {
	return services.EditKatalogInput{
		SyaratPemilahan: "Harus dicuci dulu",
		Akses:           ktlAkses(bankID, role),
	}
}

func TestEditKatalog_KatalogTidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(nil, errors.New("not found"))

	hasil, err := svc.EditKatalog("BSI-0010121", ktlEditInput("BSI-001", models.AdminBSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestEditKatalog_BankPemilikTidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindBank", "BSI-001").Return(nil, errors.New("not found"))

	hasil, err := svc.EditKatalog("BSI-0010121", ktlEditInput("BSI-001", models.AdminBSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
	assert.Contains(t, err.Error(), "Bank pemilik")
}

func TestEditKatalog_BankLainDitolak(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)

	hasil, err := svc.EditKatalog("BSI-0010121", ktlEditInput("BSM-009", models.AdminBSM))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, ktlKode(t, err))
	assert.Contains(t, err.Error(), "tidak memiliki akses")
}

func TestEditKatalog_BSUDitolak(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSU-0010121").Return(ktlKatalog("BSU-0010121", "BSU-001"), nil)
	repo.On("FindBank", "BSU-001").Return(ktlBSU("BSI-001"), nil)

	hasil, err := svc.EditKatalog("BSU-0010121", ktlEditInput("BSU-001", models.AdminBSU))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, ktlKode(t, err))
	assert.Contains(t, err.Error(), "mengedit katalog")
}

func TestEditKatalog_TanpaFileFotoLamaDipertahankan(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("SaveKatalog", mock.Anything, katalog).Return(nil)
	expectTx(dbMock, 0)

	in := ktlEditInput("BSI-001", models.AdminBSI)
	in.FotoURL = "https://cdn/baru.png"
	in.AdaFileFoto = false
	hasil, err := svc.EditKatalog("BSI-0010121", in)

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn/botol.png", hasil.PhotoURL)
	assert.Equal(t, "Harus dicuci dulu", hasil.SyaratPemilahan)
}

func TestEditKatalog_DenganFileFotoDiganti(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("SaveKatalog", mock.Anything, katalog).Return(nil)
	expectTx(dbMock, 0)

	in := ktlEditInput("BSI-001", models.AdminBSI)
	in.FotoURL = "https://cdn/baru.png"
	in.AdaFileFoto = true
	hasil, err := svc.EditKatalog("BSI-0010121", in)

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn/baru.png", hasil.PhotoURL)
}

func TestEditKatalog_SaveError(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("SaveKatalog", mock.Anything, katalog).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.EditKatalog("BSI-0010121", ktlEditInput("BSI-001", models.AdminBSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestEditKatalog_UbahKategoriIkutKatalogSejenis(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindKategori", "5").Return(&models.KategoriSampah{KategoriID: 5, Kategori: "Karton"}, nil)
	repo.On("SaveKatalog", mock.Anything, katalog).Return(nil)
	repo.On("UpdateKategoriSejenis", mock.Anything, 12, "BSI-001", 5).Return(nil)
	expectTx(dbMock, 0)

	in := ktlEditInput("BSI-001", models.AdminBSI)
	in.KategoriID = 5
	hasil, err := svc.EditKatalog("BSI-0010121", in)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, 5, hasil.KategoriID)
	}
	repo.AssertExpectations(t)
}

func TestEditKatalog_KategoriTidakValid(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindKategori", "99").Return(nil, gorm.ErrRecordNotFound)

	in := ktlEditInput("BSI-001", models.AdminBSI)
	in.KategoriID = 99
	hasil, err := svc.EditKatalog("BSI-0010121", in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, ktlKode(t, err))
	assert.Contains(t, err.Error(), "kategori_id")
	repo.AssertNotCalled(t, "SaveKatalog", mock.Anything, mock.Anything)
}

func TestEditKatalog_KategoriSamaTidakUpdateSejenis(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("SaveKatalog", mock.Anything, katalog).Return(nil)
	expectTx(dbMock, 0)

	in := ktlEditInput("BSI-001", models.AdminBSI)
	in.KategoriID = 3
	_, err := svc.EditKatalog("BSI-0010121", in)

	assert.NoError(t, err)
	repo.AssertNotCalled(t, "FindKategori", mock.Anything)
	repo.AssertNotCalled(t, "UpdateKategoriSejenis", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestEditKatalog_UpdateKategoriSejenisErrorRollback(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("FindKategori", "5").Return(&models.KategoriSampah{KategoriID: 5, Kategori: "Karton"}, nil)
	repo.On("SaveKatalog", mock.Anything, katalog).Return(nil)
	repo.On("UpdateKategoriSejenis", mock.Anything, 12, "BSI-001", 5).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	in := ktlEditInput("BSI-001", models.AdminBSI)
	in.KategoriID = 5
	hasil, err := svc.EditKatalog("BSI-0010121", in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "katalog sejenis")
}

func ktlHapusInput(bankID string, role models.RoleAdmin) services.HapusKatalogInput {
	return services.HapusKatalogInput{SampahID: "BSI-0010121", Akses: ktlAkses(bankID, role)}
}

func TestHapusKatalog_TidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(nil, errors.New("not found"))

	err := svc.HapusKatalog(ktlHapusInput("BSI-001", models.AdminBSI))

	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestHapusKatalog_BankLainDitolak(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)

	err := svc.HapusKatalog(ktlHapusInput("BSM-009", models.AdminBSM))

	assert.Equal(t, http.StatusForbidden, ktlKode(t, err))
	repo.AssertNotCalled(t, "DeleteKatalog", mock.Anything)
}

func TestHapusKatalog_BSUDitolak(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSU-001"), nil)
	repo.On("FindBank", "BSU-001").Return(ktlBSU("BSI-001"), nil)

	err := svc.HapusKatalog(ktlHapusInput("BSU-001", models.AdminBSU))

	assert.Equal(t, http.StatusForbidden, ktlKode(t, err))
	assert.Contains(t, err.Error(), "menghapus katalog")
}

func TestHapusKatalog_CekPemakaianError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("CountPemakaianKatalog", "BSI-0010121").Return("", errors.New("db error"))

	err := svc.HapusKatalog(ktlHapusInput("BSI-001", models.AdminBSI))

	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestHapusKatalog_MasihDipakai(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("CountPemakaianKatalog", "BSI-0010121").Return("setoran nasabah", nil)

	err := svc.HapusKatalog(ktlHapusInput("BSI-001", models.AdminBSI))

	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	assert.Contains(t, err.Error(), "setoran nasabah")
	repo.AssertNotCalled(t, "DeleteKatalog", mock.Anything)
}

func TestHapusKatalog_FKRestrictJadiConflict(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("CountPemakaianKatalog", "BSI-0010121").Return("", nil)
	repo.On("DeleteKatalog", katalog).Return(repositories.ErrKatalogMasihDipakai)

	err := svc.HapusKatalog(ktlHapusInput("BSI-001", models.AdminBSI))

	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	assert.Contains(t, err.Error(), "transaksi")
}

func TestHapusKatalog_DeleteError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("CountPemakaianKatalog", "BSI-0010121").Return("", nil)
	repo.On("DeleteKatalog", katalog).Return(errors.New("db error"))

	err := svc.HapusKatalog(ktlHapusInput("BSI-001", models.AdminBSI))

	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestHapusKatalog_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("CountPemakaianKatalog", "BSI-0010121").Return("", nil)
	repo.On("DeleteKatalog", katalog).Return(nil)

	err := svc.HapusKatalog(ktlHapusInput("BSI-001", models.AdminBSI))

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAksesKatalog_SuperadminLintasBank(t *testing.T) {
	dbMock, repo, svc := newKtlSvc(t)
	katalog := ktlKatalog("BSI-0010121", "BSI-001")
	repo.On("FindKatalog", "BSI-0010121").Return(katalog, nil)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("SaveKatalog", mock.Anything, katalog).Return(nil)
	expectTx(dbMock, 0)

	hasil, err := svc.EditKatalog("BSI-0010121", ktlEditInput("", models.SuperAdmin))

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestAksesKatalog_StaffBSILintasBank(t *testing.T) {
	for _, role := range []models.RoleAdmin{models.AdminBSI, models.PetugasBSI} {
		_, repo, svc := newKtlSvc(t)
		katalog := ktlKatalog("BSM-0010121", "BSM-001")
		repo.On("FindKatalogWithRelasi", "BSM-0010121").Return(katalog, nil)
		repo.On("FindStok", "BSM-001", "BSM-0010121").Return(float64(5), nil)
		repo.On("ListSchemaHarga", "BSM-0010121").Return([]models.SchemaHargaSampah{}, nil)

		hasil, err := svc.DetailKatalog("BSM-0010121", ktlAkses("BSI-001", role))

		assert.NoError(t, err)
		assert.NotNil(t, hasil)
	}
}

func TestListKatalogBank_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-XXX").Return(nil, errors.New("not found"))

	items, meta, err := svc.ListKatalogBank("BSI-XXX", 1)

	assert.Nil(t, items)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestListKatalogBank_BSUTanpaInduk(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSU-001").Return(ktlBSU(""), nil)

	items, meta, err := svc.ListKatalogBank("BSU-001", 1)

	assert.Nil(t, items)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusBadRequest, ktlKode(t, err))
	assert.Contains(t, err.Error(), "BSI induk")
}

func TestListKatalogBank_BSUPakaiKatalogInduk(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	katalogs := []models.KatalogSampah{*ktlKatalog("BSI-0010121", "BSI-001")}
	repo.On("FindBank", "BSU-001").Return(ktlBSU("BSI-001"), nil)
	repo.On("ListKatalogByBank", "BSI-001", 12, 0).Return(katalogs, nil)
	repo.On("ListStokByBank", "BSU-001", []string{"BSI-0010121"}).
		Return(map[string]float64{"BSI-0010121": 7.5}, nil)
	repo.On("CountKatalogByBank", "BSI-001").Return(int64(1), nil)

	items, meta, err := svc.ListKatalogBank("BSU-001", 1)

	assert.NoError(t, err)
	if assert.Len(t, items, 1) {
		assert.Equal(t, 7.5, items[0].Stok, "stok yang dipakai stok BSU, bukan BSI")
		assert.Equal(t, "BSI-001", items[0].BankID)
	}
	assert.NotNil(t, meta)
	repo.AssertExpectations(t)
}

func TestListKatalogBank_ListError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("ListKatalogByBank", "BSI-001", 12, 0).Return(nil, errors.New("db error"))

	items, meta, err := svc.ListKatalogBank("BSI-001", 1)

	assert.Nil(t, items)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestListKatalogBank_StokError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("ListKatalogByBank", "BSI-001", 12, 0).Return([]models.KatalogSampah{}, nil)
	repo.On("ListStokByBank", "BSI-001", []string{}).Return(nil, errors.New("db error"))

	items, _, err := svc.ListKatalogBank("BSI-001", 1)

	assert.Nil(t, items)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "stok")
}

func TestListKatalogBank_CountError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("ListKatalogByBank", "BSI-001", 12, 0).Return([]models.KatalogSampah{}, nil)
	repo.On("ListStokByBank", "BSI-001", []string{}).Return(map[string]float64{}, nil)
	repo.On("CountKatalogByBank", "BSI-001").Return(int64(0), errors.New("db error"))

	items, _, err := svc.ListKatalogBank("BSI-001", 1)

	assert.Nil(t, items)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestListKatalogBank_TanpaPagination(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("ListKatalogByBank", "BSI-001", 0, 0).Return([]models.KatalogSampah{}, nil)
	repo.On("ListStokByBank", "BSI-001", []string{}).Return(map[string]float64{}, nil)

	items, meta, err := svc.ListKatalogBank("BSI-001", 0)

	assert.NoError(t, err)
	assert.NotNil(t, items, "slice kosong harus non-nil supaya JSON-nya []")
	assert.Nil(t, meta, "tanpa pagination tidak ada metadata")
	repo.AssertNotCalled(t, "CountKatalogByBank", mock.Anything)
}

func TestListKatalogBank_OffsetHalamanKedua(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("ListKatalogByBank", "BSI-001", 12, 12).Return([]models.KatalogSampah{}, nil)
	repo.On("ListStokByBank", "BSI-001", []string{}).Return(map[string]float64{}, nil)
	repo.On("CountKatalogByBank", "BSI-001").Return(int64(30), nil)

	_, meta, err := svc.ListKatalogBank("BSI-001", 2)

	assert.NoError(t, err)
	if assert.NotNil(t, meta) {
		assert.Equal(t, 2, meta.Page)
		assert.Equal(t, 12, meta.Limit)
		assert.Equal(t, 30, meta.Total)
		assert.Equal(t, 3, meta.TotalPages)
	}
	repo.AssertExpectations(t)
}

func TestListKatalogBank_Mapping(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	katalogs := []models.KatalogSampah{
		*ktlKatalog("BSI-0010121", "BSI-001"),
		*ktlKatalog("BSI-0010131", "BSI-001"),
	}
	repo.On("FindBank", "BSI-001").Return(ktlBank("BSI-001", models.BSI), nil)
	repo.On("ListKatalogByBank", "BSI-001", 12, 0).Return(katalogs, nil)
	repo.On("ListStokByBank", "BSI-001", []string{"BSI-0010121", "BSI-0010131"}).
		Return(map[string]float64{"BSI-0010121": 3.25}, nil)
	repo.On("CountKatalogByBank", "BSI-001").Return(int64(2), nil)

	items, _, err := svc.ListKatalogBank("BSI-001", 1)

	assert.NoError(t, err)
	if assert.Len(t, items, 2) {
		assert.Equal(t, "Botol Plastik", items[0].NamaSampah, "nama diambil dari master sampah")
		assert.Equal(t, "kg", items[0].Satuan)
		assert.Equal(t, "Plastik", items[0].Kategori.Kategori)
		assert.Equal(t, models.RewardEnumUang, items[0].Reward.NamaReward)
		assert.Equal(t, 3.25, items[0].Stok)
		assert.Equal(t, float64(0), items[1].Stok, "sampah tanpa baris stok dianggap 0")
	}
}

func TestDetailKatalog_TidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalogWithRelasi", "BSI-0010121").Return(nil, errors.New("not found"))

	hasil, err := svc.DetailKatalog("BSI-0010121", ktlAkses("BSI-001", models.AdminBSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestDetailKatalog_BankLainDitolak(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalogWithRelasi", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)

	hasil, err := svc.DetailKatalog("BSI-0010121", ktlAkses("BSM-009", models.AdminBSM))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, ktlKode(t, err))
	repo.AssertNotCalled(t, "FindStok", mock.Anything, mock.Anything)
}

func TestDetailKatalog_StokError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalogWithRelasi", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindStok", "BSI-001", "BSI-0010121").Return(float64(0), errors.New("db error"))

	hasil, err := svc.DetailKatalog("BSI-0010121", ktlAkses("BSI-001", models.AdminBSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "stok")
}

func TestDetailKatalog_SchemaHargaError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalogWithRelasi", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindStok", "BSI-001", "BSI-0010121").Return(float64(5), nil)
	repo.On("ListSchemaHarga", "BSI-0010121").Return(nil, errors.New("db error"))

	hasil, err := svc.DetailKatalog("BSI-0010121", ktlAkses("BSI-001", models.AdminBSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "harga")
}

func TestDetailKatalog_HargaPerLevelKosongBukanNull(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalogWithRelasi", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindStok", "BSI-001", "BSI-0010121").Return(float64(5), nil)
	repo.On("ListSchemaHarga", "BSI-0010121").Return([]models.SchemaHargaSampah{}, nil)

	hasil, err := svc.DetailKatalog("BSI-0010121", ktlAkses("BSI-001", models.AdminBSI))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.NotNil(t, hasil.HargaPerLevel)
	}
}

func TestDetailKatalog_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	schemas := []models.SchemaHargaSampah{
		{SchemaID: "2BSI-0010121", SampahID: "BSI-0010121", LevelUser: models.LevelNasabah, Harga: 3000, SatuanReward: models.SatuanRewardEnumRp},
		{SchemaID: "1BSI-0010121", SampahID: "BSI-0010121", LevelUser: models.LevelEksternal, Harga: 2500, SatuanReward: models.SatuanRewardEnumRp},
	}
	repo.On("FindKatalogWithRelasi", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("FindStok", "BSI-001", "BSI-0010121").Return(12.75, nil)
	repo.On("ListSchemaHarga", "BSI-0010121").Return(schemas, nil)

	hasil, err := svc.DetailKatalog("BSI-0010121", ktlAkses("BSI-001", models.AdminBSI))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSI-0010121", hasil.SampahID)
		assert.Equal(t, 12, hasil.SarokID, "sarok_id diambil dari relasi master sampah")
		assert.Equal(t, "Botol Plastik", hasil.NamaSampah)
		assert.Equal(t, "kg", hasil.Satuan)
		assert.Equal(t, "Bersih dan kering", hasil.SyaratPemilahan)
		assert.Equal(t, "Plastik", hasil.Kategori.Kategori)
		assert.Equal(t, 12.75, hasil.Stok)
		if assert.Len(t, hasil.HargaPerLevel, 2) {
			assert.Equal(t, models.LevelNasabah, hasil.HargaPerLevel[0].LevelUser)
			assert.Equal(t, float64(3000), hasil.HargaPerLevel[0].Harga)
			assert.Equal(t, models.SatuanRewardEnumRp, hasil.HargaPerLevel[0].SatuanReward)
		}
	}
}

func TestHistoryHargaKatalog_TidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(nil, errors.New("not found"))

	hasil, err := svc.HistoryHargaKatalog("BSI-0010121", "", "", ktlAkses("BSI-001", models.AdminBSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestHistoryHargaKatalog_BankLainDitolak(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)

	hasil, err := svc.HistoryHargaKatalog("BSI-0010121", "", "", ktlAkses("BSM-009", models.AdminBSM))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, ktlKode(t, err))
	repo.AssertNotCalled(t, "ListHistoryHarga", mock.Anything, mock.Anything, mock.Anything)
}

func TestHistoryHargaKatalog_RepoError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("ListHistoryHarga", "BSI-0010121", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	hasil, err := svc.HistoryHargaKatalog("BSI-0010121", "", "", ktlAkses("BSI-001", models.AdminBSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
	assert.Contains(t, err.Error(), "riwayat harga")
}

func TestHistoryHargaKatalog_NilJadiSliceKosong(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("ListHistoryHarga", "BSI-0010121", mock.Anything, mock.Anything).Return([]repositories.HistoryHargaRow(nil), nil)

	hasil, err := svc.HistoryHargaKatalog("BSI-0010121", "", "", ktlAkses("BSI-001", models.AdminBSI))

	assert.NoError(t, err)
	assert.NotNil(t, hasil, "nil harus jadi [] supaya JSON-nya bukan null")
	assert.Len(t, hasil, 0)
}

func TestHistoryHargaKatalog_TanpaFilter(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	histories := []repositories.HistoryHargaRow{
		{HistoryID: "H-001", SchemaID: "2BSI-0010121", LevelUser: models.LevelNasabah,
			HargaLama: 2800, HargaBaru: 3000, ChangedByNama: "Admin A"},
	}
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("ListHistoryHarga", "BSI-0010121", (*time.Time)(nil), (*time.Time)(nil)).Return(histories, nil)

	hasil, err := svc.HistoryHargaKatalog("BSI-0010121", "", "", ktlAkses("BSI-001", models.AdminBSI))

	assert.NoError(t, err)
	assert.Len(t, hasil, 1)
	repo.AssertExpectations(t)
}

func TestHistoryHargaKatalog_FilterTanggal(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKatalog", "BSI-0010121").Return(ktlKatalog("BSI-0010121", "BSI-001"), nil)
	repo.On("ListHistoryHarga", "BSI-0010121",
		mock.MatchedBy(func(start *time.Time) bool {
			return start != nil && start.Equal(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
		}),
		mock.MatchedBy(func(end *time.Time) bool {
			return end != nil && end.Equal(time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC))
		}),
	).Return([]repositories.HistoryHargaRow{}, nil)

	hasil, err := svc.HistoryHargaKatalog("BSI-0010121", "2026-01-01", "2026-01-31", ktlAkses("BSI-001", models.AdminBSI))

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertExpectations(t)
}

func TestListKategori_RepoError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("ListKategori").Return(nil, errors.New("db error"))

	hasil, err := svc.ListKategori()

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestListKategori_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("ListKategori").Return([]models.KategoriSampah{
		{KategoriID: 1, Kategori: "Plastik"},
		{KategoriID: 2, Kategori: "Kertas"},
	}, nil)

	hasil, err := svc.ListKategori()

	assert.NoError(t, err)
	assert.Len(t, hasil, 2)
}

func TestTambahKategori_CekNamaError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("CountKategoriDenganNama", "plastik", "").Return(int64(0), errors.New("db error"))

	hasil, err := svc.TambahKategori("Plastik")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestTambahKategori_NamaDinormalisasiSaatCekUnik(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("CountKategoriDenganNama", "plastikdaurulang", "").Return(int64(1), nil)

	hasil, err := svc.TambahKategori("Plastik Daur-Ulang")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	assert.Contains(t, err.Error(), "sudah ada")
	repo.AssertExpectations(t)
}

func TestTambahKategori_CreateError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("CountKategoriDenganNama", "plastik", "").Return(int64(0), nil)
	repo.On("CreateKategori", mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.TambahKategori("Plastik")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestTambahKategori_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("CountKategoriDenganNama", "plastik", "").Return(int64(0), nil)
	repo.On("CreateKategori", mock.Anything).Return(nil)

	hasil, err := svc.TambahKategori("Plastik")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Plastik", hasil.Kategori, "nama disimpan apa adanya, normalisasi cuma buat cek unik")
	}
}

func TestUbahKategori_TidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKategori", "9").Return(nil, errors.New("not found"))

	hasil, err := svc.UbahKategori("9", "Plastik")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestUbahKategori_DirinyaSendiriDikecualikan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKategori", "3").Return(&models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"}, nil)
	repo.On("CountKategoriDenganNama", "plastik", "3").Return(int64(0), nil)
	repo.On("SaveKategori", mock.Anything).Return(nil)

	hasil, err := svc.UbahKategori("3", "Plastik")

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertExpectations(t)
}

func TestUbahKategori_NamaSudahDipakaiKategoriLain(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKategori", "3").Return(&models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"}, nil)
	repo.On("CountKategoriDenganNama", "kertas", "3").Return(int64(1), nil)

	hasil, err := svc.UbahKategori("3", "Kertas")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	repo.AssertNotCalled(t, "SaveKategori", mock.Anything)
}

func TestUbahKategori_SaveError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKategori", "3").Return(&models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"}, nil)
	repo.On("CountKategoriDenganNama", "kertas", "3").Return(int64(0), nil)
	repo.On("SaveKategori", mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.UbahKategori("3", "Kertas")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestUbahKategori_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	kategori := &models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"}
	repo.On("FindKategori", "3").Return(kategori, nil)
	repo.On("CountKategoriDenganNama", "kertas", "3").Return(int64(0), nil)
	repo.On("SaveKategori", kategori).Return(nil)

	hasil, err := svc.UbahKategori("3", "Kertas")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Kertas", hasil.Kategori)
		assert.Equal(t, 3, hasil.KategoriID)
	}
}

func TestHapusKategori_TidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKategori", "9").Return(nil, errors.New("not found"))

	err := svc.HapusKategori("9")

	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestHapusKategori_CekPemakaianError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKategori", "3").Return(&models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"}, nil)
	repo.On("CountKatalogByKategori", "3").Return(int64(0), errors.New("db error"))

	err := svc.HapusKategori("3")

	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestHapusKategori_MasihDipakai(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindKategori", "3").Return(&models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"}, nil)
	repo.On("CountKatalogByKategori", "3").Return(int64(4), nil)

	err := svc.HapusKategori("3")

	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	assert.Contains(t, err.Error(), "masih digunakan")
	repo.AssertNotCalled(t, "DeleteKategori", mock.Anything)
}

func TestHapusKategori_DeleteError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	kategori := &models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"}
	repo.On("FindKategori", "3").Return(kategori, nil)
	repo.On("CountKatalogByKategori", "3").Return(int64(0), nil)
	repo.On("DeleteKategori", kategori).Return(errors.New("db error"))

	err := svc.HapusKategori("3")

	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestHapusKategori_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	kategori := &models.KategoriSampah{KategoriID: 3, Kategori: "Plastik"}
	repo.On("FindKategori", "3").Return(kategori, nil)
	repo.On("CountKatalogByKategori", "3").Return(int64(0), nil)
	repo.On("DeleteKategori", kategori).Return(nil)

	err := svc.HapusKategori("3")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestListSatuan_RepoError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("ListSatuan").Return(nil, errors.New("db error"))

	hasil, err := svc.ListSatuan()

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestListSatuan_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("ListSatuan").Return([]models.Satuan{
		{SatuanID: 1, Satuan: "kg"},
		{SatuanID: 2, Satuan: "pcs"},
	}, nil)

	hasil, err := svc.ListSatuan()

	assert.NoError(t, err)
	assert.Len(t, hasil, 2)
}

func TestTambahSatuan_CekNamaError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("CountSatuanDenganNama", "unit", "").Return(int64(0), errors.New("db error"))

	hasil, err := svc.TambahSatuan("Unit")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestTambahSatuan_NamaDinormalisasiSaatCekUnik(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("CountSatuanDenganNama", "kg", "").Return(int64(1), nil)

	hasil, err := svc.TambahSatuan("K.G")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	assert.Contains(t, err.Error(), "sudah ada")
	repo.AssertExpectations(t)
}

func TestTambahSatuan_CreateError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("CountSatuanDenganNama", "unit", "").Return(int64(0), nil)
	repo.On("CreateSatuan", mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.TambahSatuan("Unit")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestTambahSatuan_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("CountSatuanDenganNama", "unit", "").Return(int64(0), nil)
	repo.On("CreateSatuan", mock.Anything).Return(nil)

	hasil, err := svc.TambahSatuan("Unit")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Unit", hasil.Satuan, "nama disimpan apa adanya, normalisasi cuma buat cek unik")
	}
}

func TestUbahSatuan_TidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindSatuan", "9").Return(nil, errors.New("not found"))

	hasil, err := svc.UbahSatuan("9", "kg")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestUbahSatuan_DirinyaSendiriDikecualikan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("CountSatuanDenganNama", "kg", "1").Return(int64(0), nil)
	repo.On("SaveSatuan", mock.Anything).Return(nil)

	hasil, err := svc.UbahSatuan("1", "Kg")

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertExpectations(t)
}

func TestUbahSatuan_NamaSudahDipakaiSatuanLain(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("CountSatuanDenganNama", "pcs", "1").Return(int64(1), nil)

	hasil, err := svc.UbahSatuan("1", "pcs")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	repo.AssertNotCalled(t, "SaveSatuan", mock.Anything)
}

func TestUbahSatuan_SaveError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("CountSatuanDenganNama", "kilogram", "1").Return(int64(0), nil)
	repo.On("SaveSatuan", mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.UbahSatuan("1", "kilogram")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestUbahSatuan_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	satuan := &models.Satuan{SatuanID: 1, Satuan: "kg"}
	repo.On("FindSatuan", "1").Return(satuan, nil)
	repo.On("CountSatuanDenganNama", "kilogram", "1").Return(int64(0), nil)
	repo.On("SaveSatuan", satuan).Return(nil)

	hasil, err := svc.UbahSatuan("1", "kilogram")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "kilogram", hasil.Satuan)
		assert.Equal(t, 1, hasil.SatuanID)
	}
}

func TestHapusSatuan_TidakDitemukan(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindSatuan", "9").Return(nil, errors.New("not found"))

	err := svc.HapusSatuan("9")

	assert.Equal(t, http.StatusNotFound, ktlKode(t, err))
}

func TestHapusSatuan_CekPemakaianError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("CountSampahBySatuan", "1").Return(int64(0), errors.New("db error"))

	err := svc.HapusSatuan("1")

	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestHapusSatuan_MasihDipakai(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	repo.On("FindSatuan", "1").Return(&models.Satuan{SatuanID: 1, Satuan: "kg"}, nil)
	repo.On("CountSampahBySatuan", "1").Return(int64(5), nil)

	err := svc.HapusSatuan("1")

	assert.Equal(t, http.StatusConflict, ktlKode(t, err))
	assert.Contains(t, err.Error(), "masih digunakan")
	repo.AssertNotCalled(t, "DeleteSatuan", mock.Anything)
}

func TestHapusSatuan_DeleteError(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	satuan := &models.Satuan{SatuanID: 4, Satuan: "unit"}
	repo.On("FindSatuan", "4").Return(satuan, nil)
	repo.On("CountSampahBySatuan", "4").Return(int64(0), nil)
	repo.On("DeleteSatuan", satuan).Return(errors.New("db error"))

	err := svc.HapusSatuan("4")

	assert.Equal(t, http.StatusInternalServerError, ktlKode(t, err))
}

func TestHapusSatuan_Sukses(t *testing.T) {
	_, repo, svc := newKtlSvc(t)
	satuan := &models.Satuan{SatuanID: 4, Satuan: "unit"}
	repo.On("FindSatuan", "4").Return(satuan, nil)
	repo.On("CountSampahBySatuan", "4").Return(int64(0), nil)
	repo.On("DeleteSatuan", satuan).Return(nil)

	err := svc.HapusSatuan("4")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
