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

const (
	bnkBSI   = "101002001"
	bnkBSU   = "301002001"
	bnkUser  = "1371010101010001"
	bnkActor = "100001"

	bnkActorUser = "1371010101010099"
)

func bnkPelaku(bankID string) *models.Admin {
	return &models.Admin{AdminID: bnkActor, UserID: bnkActorUser, BankID: &bankID, Role: models.AdminBSU, StatusAdmin: models.Aktif}
}

func newBnkSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockBankRepo, *mocks.MockMailer, services.BankService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBankRepo)
	mailerMock := new(mocks.MockMailer)
	return dbMock, repo, mailerMock, services.NewBankService(gdb, repo, mailerMock)
}

func bnkBank(bankID string, jenis models.JenisBank) *models.BankSampah {
	kec, kel := 10, 20
	return &models.BankSampah{
		BankID: bankID, NamaBank: "Bank " + bankID, JenisBank: jenis, IsActive: true,
		Alamat: "Jl. Mawar 1", Provinsi: "Sumatera Barat", KabupatenKota: "Kota Padang",
		IDKecamatan: &kec, IDKelurahan: &kel,
	}
}

func bnkUserModel() *models.User {
	return &models.User{UserID: bnkUser, Nama: "Budi Santoso", Email: "budi@mail.com"}
}

func bnkAdmin(adminID string, role models.RoleAdmin, status models.StatusAkun) *models.Admin {
	bankID := bnkBSU
	return &models.Admin{
		AdminID: adminID, UserID: bnkUser, BankID: &bankID, Role: role, StatusAdmin: status,
		User: models.User{UserID: bnkUser, Nama: "Budi Santoso", Email: "budi@mail.com"},
		Bank: models.BankSampah{BankID: bnkBSU, NamaBank: "BSU Mawar"},
	}
}

func bnkReward(id int, nama models.RewardEnum, satuan models.SatuanRewardEnum) *models.Reward {
	return &models.Reward{RewardID: id, NamaReward: nama, Satuan: satuan}
}

func bnkCreateInput(jenis models.JenisBank) services.CreateBankInput {
	return services.CreateBankInput{
		JenisBank: jenis, Nama: "Bank Baru", Deskripsi: "deskripsi",
		Provinsi: "Sumatera Barat", KabupatenKota: "Kota Padang",
		IDKecamatan: 10, IDKelurahan: 20, Alamat: "Jl. Mawar 1",
		Latitude: -0.95, Longitude: 100.35, ActorAdminID: bnkActor,
	}
}

func bnkStubReward(repo *mocks.MockBankRepo) {
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumUang).
		Return(bnkReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumBarang).
		Return(bnkReward(2, models.RewardEnumBarang, models.SatuanRewardEnumPoin), nil)
	repo.On("CreateSaldoRekening", mock.Anything, mock.Anything).Return(nil)
}

func bnkStubAdminBaru(repo *mocks.MockBankRepo) {
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(0), nil)
	repo.On("CreateAdmin", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindUserTx", mock.Anything, bnkUser).Return(bnkUserModel(), nil)
}

func expectQueries(m sqlmock.Sqlmock, n int) {
	for i := 0; i < n; i++ {
		m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	}
}

func bnkKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func TestCreateBank_JenisTidakValid(t *testing.T) {
	_, _, _, svc := newBnkSvc(t)

	hasil, err := svc.CreateBank(bnkCreateInput("koperasi"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bnkKode(t, err))
	assert.Contains(t, err.Error(), "Jenis bank")
}

func TestCreateBank_CreateBankError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("CreateBank", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	expectQueries(dbMock, 1)
	dbMock.ExpectRollback()

	hasil, err := svc.CreateBank(bnkCreateInput(models.BSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	assert.Contains(t, err.Error(), "BSI")
}

func TestCreateBank_RewardTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("CreateBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumUang).Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	expectQueries(dbMock, 1)
	dbMock.ExpectRollback()

	hasil, err := svc.CreateBank(bnkCreateInput(models.BSI))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	assert.Contains(t, err.Error(), "Reward")
}

func TestCreateBank_RekeningPerJenisBank(t *testing.T) {
	kasus := []struct {
		jenis        models.JenisBank
		wantRekening int
		wantRole     models.RoleAdmin
	}{
		{models.BSI, 2, models.AdminBSI},
		{models.BSM, 2, models.AdminBSM},
		{models.BSU, 1, models.AdminBSU},
	}
	for _, k := range kasus {
		t.Run(string(k.jenis), func(t *testing.T) {
			dbMock, repo, _, svc := newBnkSvc(t)
			repo.On("CreateBank", mock.Anything, mock.Anything).Return(nil)
			repo.On("FindRewardByNama", mock.Anything, models.RewardEnumUang).
				Return(bnkReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
			repo.On("FindRewardByNama", mock.Anything, models.RewardEnumBarang).
				Return(bnkReward(2, models.RewardEnumBarang, models.SatuanRewardEnumPoin), nil)

			var rekening []*models.SaldoRekening
			repo.On("CreateSaldoRekening", mock.Anything, mock.Anything).
				Run(func(args mock.Arguments) {
					rekening = append(rekening, args.Get(1).(*models.SaldoRekening))
				}).Return(nil)
			expectTx(dbMock, 1)

			hasil, err := svc.CreateBank(bnkCreateInput(k.jenis))

			assert.NoError(t, err)
			assert.NotNil(t, hasil)
			if assert.Len(t, rekening, k.wantRekening) {
				for _, r := range rekening {
					assert.Equal(t, models.EntitasBankSampah, r.Entitas)
					assert.Equal(t, float64(0), r.NominalSaldo, "rekening bank selalu mulai dari 0")
				}
				assert.Equal(t, models.SatuanRewardEnumRp, rekening[0].SatuanNominalSaldo)
			}
		})
	}
}

func TestCreateBank_FormatBankID(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("CreateBank", mock.Anything, mock.Anything).Return(nil)
	bnkStubReward(repo)
	expectTx(dbMock, 1)

	hasil, err := svc.CreateBank(bnkCreateInput(models.BSU))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "310020001", hasil.BankID, "3=BSU, 10=kecamatan, 020=kelurahan, 001=urutan")
		assert.Len(t, hasil.BankID, 9, "bank_id selalu 9 digit (varchar(9) di DB)")
		assert.True(t, hasil.IsActive, "bank baru langsung aktif")
	}
}

func TestCreateBank_SudahPunyaAdminAktif(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("CreateBank", mock.Anything, mock.Anything).Return(nil)
	bnkStubReward(repo)
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(1), nil)
	dbMock.ExpectBegin()
	expectQueries(dbMock, 1)
	dbMock.ExpectRollback()

	in := bnkCreateInput(models.BSI)
	in.UserIDs = []string{bnkUser}
	hasil, err := svc.CreateBank(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, bnkKode(t, err))
	assert.Contains(t, err.Error(), "admin aktif")
	repo.AssertNotCalled(t, "CreateAdmin", mock.Anything, mock.Anything)
}

func TestCreateBank_EmailGagalMembatalkanTransaksi(t *testing.T) {
	dbMock, repo, mailerMock, svc := newBnkSvc(t)
	repo.On("CreateBank", mock.Anything, mock.Anything).Return(nil)
	bnkStubReward(repo)
	bnkStubAdminBaru(repo)
	mailerMock.On("SendEmail", mock.Anything).Return(errors.New("smtp down"))
	dbMock.ExpectBegin()
	expectQueries(dbMock, 3)
	dbMock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectRollback()

	in := bnkCreateInput(models.BSI)
	in.UserIDs = []string{bnkUser}
	hasil, err := svc.CreateBank(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	assert.Contains(t, err.Error(), "send email")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestCreateBank_DenganAdminSukses(t *testing.T) {
	dbMock, repo, mailerMock, svc := newBnkSvc(t)
	repo.On("CreateBank", mock.Anything, mock.Anything).Return(nil)
	bnkStubReward(repo)
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(0), nil)
	repo.On("FindUserTx", mock.Anything, bnkUser).Return(bnkUserModel(), nil)

	var adminDibuat *models.Admin
	repo.On("CreateAdmin", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { adminDibuat = args.Get(1).(*models.Admin) }).Return(nil)
	mailerMock.On("SendEmail", mock.Anything).Return(nil)
	dbMock.ExpectBegin()
	expectQueries(dbMock, 3)
	dbMock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	dbMock.ExpectCommit()

	in := bnkCreateInput(models.BSU)
	in.UserIDs = []string{bnkUser}
	hasil, err := svc.CreateBank(in)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	if assert.NotNil(t, adminDibuat) {
		assert.Equal(t, models.AdminBSU, adminDibuat.Role, "role admin ikut jenis banknya")
		assert.Equal(t, models.Pending, adminDibuat.StatusAdmin, "pending sampai OTP diaktivasi")
		if assert.NotNil(t, adminDibuat.BankID) {
			assert.Equal(t, hasil.BankID, *adminDibuat.BankID)
		}
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestListBankAktif_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBankAktifWithWilayah").Return(nil, errors.New("db error"))

	hasil, err := svc.ListBankAktif()

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListBankAktif_WilayahNilAman(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	bank := *bnkBank(bnkBSU, models.BSU)
	bank.Kecamatan = nil
	bank.Kelurahan = nil
	repo.On("ListBankAktifWithWilayah").Return([]models.BankSampah{bank}, nil)

	hasil, err := svc.ListBankAktif()

	assert.NoError(t, err)
	if assert.Len(t, hasil, 1) {
		assert.Equal(t, "", hasil[0].Kecamatan)
		assert.Equal(t, "", hasil[0].Kelurahan)
		assert.Equal(t, "Jl. Mawar 1", hasil[0].AlamatBank)
	}
}

func TestListBankAktif_WilayahTerisi(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	bank := *bnkBank(bnkBSU, models.BSU)
	bank.Kecamatan = &models.Kecamatan{Kecamatan: "Padang Barat"}
	bank.Kelurahan = &models.Kelurahan{Kelurahan: "Purus"}
	repo.On("ListBankAktifWithWilayah").Return([]models.BankSampah{bank}, nil)

	hasil, err := svc.ListBankAktif()

	assert.NoError(t, err)
	if assert.Len(t, hasil, 1) {
		assert.Equal(t, "Padang Barat", hasil[0].Kecamatan)
		assert.Equal(t, "Purus", hasil[0].Kelurahan)
	}
}

func TestListSemuaBank_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListSemuaBank").Return(nil, errors.New("db error"))

	hasil, err := svc.ListSemuaBank()

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListSemuaBank_Sukses(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListSemuaBank").Return([]models.BankSampah{*bnkBank(bnkBSU, models.BSU)}, nil)

	hasil, err := svc.ListSemuaBank()

	assert.NoError(t, err)
	assert.Len(t, hasil, 1)
}

func TestListNasabahBank_BankIDKosong(t *testing.T) {
	_, _, _, svc := newBnkSvc(t)

	rows, meta, err := svc.ListNasabahBank("", 1)

	assert.Nil(t, rows)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusBadRequest, bnkKode(t, err))
}

func TestListNasabahBank_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListNasabahByBank", bnkBSU, 20, 0).Return(nil, errors.New("db error"))

	rows, meta, err := svc.ListNasabahBank(bnkBSU, 1)

	assert.Nil(t, rows)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListNasabahBank_TanpaPagination(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListNasabahByBank", bnkBSU, 0, 0).Return([]repositories.NasabahBankRow{}, nil)

	rows, meta, err := svc.ListNasabahBank(bnkBSU, 0)

	assert.NoError(t, err)
	assert.NotNil(t, rows)
	assert.Nil(t, meta)
	repo.AssertExpectations(t)
}

func TestListNasabahBank_HalamanKedua(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListNasabahByBank", bnkBSU, 20, 20).
		Return([]repositories.NasabahBankRow{{NasabahID: "NSB-001", TotalCount: 45}}, nil)

	rows, meta, err := svc.ListNasabahBank(bnkBSU, 2)

	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	if assert.NotNil(t, meta) {
		assert.Equal(t, 2, meta.Page)
		assert.Equal(t, 20, meta.Limit)
		assert.Equal(t, 45, meta.Total)
		assert.Equal(t, 3, meta.TotalPages)
	}
}

func TestListBSI_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBSI", 20, 0).Return(nil, errors.New("db error"))

	rows, meta, err := svc.ListBSI(1)

	assert.Nil(t, rows)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListBSI_TotalDariTotalCount(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBSI", 20, 0).Return([]repositories.BSIRow{{JumlahBSU: 3, TotalCount: 21}}, nil)

	_, meta, err := svc.ListBSI(1)

	assert.NoError(t, err)
	assert.Equal(t, 21, meta.Total)
	assert.Equal(t, 2, meta.TotalPages)
}

func TestListBSU_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBSU", 20, 0).Return(nil, errors.New("db error"))

	rows, _, err := svc.ListBSU(1)

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListBSU_HalamanKosongTotalNol(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBSU", 20, 0).Return([]repositories.BSURow{}, nil)

	_, meta, err := svc.ListBSU(1)

	assert.NoError(t, err)
	assert.Equal(t, 0, meta.Total)
	assert.Equal(t, 0, meta.TotalPages)
}

func TestListBSM_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBSM", 20, 0).Return(nil, errors.New("db error"))

	rows, _, err := svc.ListBSM(1)

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListBSM_Sukses(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBSM", 20, 0).Return([]repositories.BSMRow{{BankID: "201002001", TotalCount: 1}}, nil)

	rows, meta, err := svc.ListBSM(1)

	assert.NoError(t, err)
	assert.Len(t, rows, 1)
	assert.Equal(t, 1, meta.Total)
}

func TestListUnitBSI_BankIDKosong(t *testing.T) {
	_, _, _, svc := newBnkSvc(t)

	rows, meta, err := svc.ListUnitBSI("", 1)

	assert.Nil(t, rows)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusBadRequest, bnkKode(t, err))
}

func TestListUnitBSI_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListUnitByParent", bnkBSI, 15, 0).Return(nil, errors.New("db error"))

	rows, _, err := svc.ListUnitBSI(bnkBSI, 1)

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListUnitBSI_PakaiLimitUnit(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListUnitByParent", bnkBSI, 15, 15).
		Return([]repositories.UnitBSURow{{BankID: bnkBSU, TotalCount: 30}}, nil)

	_, meta, err := svc.ListUnitBSI(bnkBSI, 2)

	assert.NoError(t, err)
	if assert.NotNil(t, meta) {
		assert.Equal(t, 15, meta.Limit)
		assert.Equal(t, 2, meta.TotalPages)
	}
	repo.AssertExpectations(t)
}

func TestListBSUByParent_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBSUByParent", bnkBSI).Return(nil, errors.New("db error"))

	rows, err := svc.ListBSUByParent(bnkBSI)

	assert.Nil(t, rows)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListBSUByParent_Sukses(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListBSUByParent", bnkBSI).
		Return([]repositories.BSUSimpleRow{{BankID: bnkBSU, NamaBSU: "BSU Mawar"}}, nil)

	rows, err := svc.ListBSUByParent(bnkBSI)

	assert.NoError(t, err)
	assert.Len(t, rows, 1)
}

func TestToggleAktivasiBank_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(nil, errors.New("not found"))

	hasil, err := svc.ToggleAktivasiBank(bnkBSU, services.AktivasiBankInput{AdminID: bnkActor})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bnkKode(t, err))
}

func TestToggleAktivasiBank_SaveError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.ToggleAktivasiBank(bnkBSU, services.AktivasiBankInput{AdminID: bnkActor})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestToggleAktivasiBank_AktifJadiNonaktif(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(nil)

	var history *models.HistoryAkunBank
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { history = args.Get(1).(*models.HistoryAkunBank) }).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.ToggleAktivasiBank(bnkBSU, services.AktivasiBankInput{
		AdminID: bnkActor, Informasi: "Bank ditutup", Keterangan: "Renovasi",
	})

	assert.NoError(t, err)
	assert.False(t, hasil.IsActive)
	if assert.NotNil(t, history) {
		assert.Equal(t, "UPDATE", history.Action)
		assert.Equal(t, bnkActor, history.CreatedBy)
		assert.Equal(t, "Bank ditutup", history.Informasi)
		assert.JSONEq(t, `{"status_bank":"Aktif"}`, string(history.OldValue))
		assert.JSONEq(t, `{"status_bank":"Non Aktif"}`, string(history.NewValue))
	}
}

func TestToggleAktivasiBank_NonaktifJadiAktif(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	bank := bnkBank(bnkBSU, models.BSU)
	bank.IsActive = false
	repo.On("FindBank", bnkBSU).Return(bank, nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(nil)

	var history *models.HistoryAkunBank
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { history = args.Get(1).(*models.HistoryAkunBank) }).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.ToggleAktivasiBank(bnkBSU, services.AktivasiBankInput{AdminID: bnkActor})

	assert.NoError(t, err)
	assert.True(t, hasil.IsActive)
	assert.JSONEq(t, `{"status_bank":"Non Aktif"}`, string(history.OldValue))
	assert.JSONEq(t, `{"status_bank":"Aktif"}`, string(history.NewValue))
}

func TestToggleAktivasiBank_HistoryError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	expectQueries(dbMock, 1)
	dbMock.ExpectRollback()

	hasil, err := svc.ToggleAktivasiBank(bnkBSU, services.AktivasiBankInput{AdminID: bnkActor})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestUpdateProfilBank_AdminIDKosong(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)

	hasil, err := svc.UpdateProfilBank(bnkBSU, services.UpdateProfilBankInput{NamaBank: "Baru"})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bnkKode(t, err))
	repo.AssertNotCalled(t, "FindBank", mock.Anything)
}

func TestUpdateProfilBank_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(nil, errors.New("not found"))

	hasil, err := svc.UpdateProfilBank(bnkBSU, services.UpdateProfilBankInput{AdminID: bnkActor})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bnkKode(t, err))
}

func TestUpdateProfilBank_FieldKosongTidakMenimpa(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.UpdateProfilBank(bnkBSU, services.UpdateProfilBankInput{AdminID: bnkActor})

	assert.NoError(t, err)
	assert.Equal(t, "Bank "+bnkBSU, hasil.NamaBank)
	assert.Equal(t, "Jl. Mawar 1", hasil.Alamat)
}

func TestUpdateProfilBank_KoordinatNolTetapDisimpan(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	bank := bnkBank(bnkBSU, models.BSU)
	bank.Latitude, bank.Longitude = -0.95, 100.35
	repo.On("FindBank", bnkBSU).Return(bank, nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	nol := 0.0
	hasil, err := svc.UpdateProfilBank(bnkBSU, services.UpdateProfilBankInput{
		AdminID: bnkActor, Latitude: &nol, Longitude: &nol,
	})

	assert.NoError(t, err)
	assert.Equal(t, 0.0, hasil.Latitude)
	assert.Equal(t, 0.0, hasil.Longitude)
}

func TestUpdateProfilBank_SuksesDanCatatHistory(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(nil)

	var history *models.HistoryAkunBank
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { history = args.Get(1).(*models.HistoryAkunBank) }).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.UpdateProfilBank(bnkBSU, services.UpdateProfilBankInput{
		AdminID: bnkActor, NamaBank: "BSU Melati", Alamat: "Jl. Melati 9",
		Deskripsi: "deskripsi baru", FotoURL: "https://cdn/baru.png",
	})

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSU Melati", hasil.NamaBank)
		assert.Equal(t, "Jl. Melati 9", hasil.Alamat)
		assert.Equal(t, "https://cdn/baru.png", hasil.PhotoURL)
	}
	if assert.NotNil(t, history) {
		assert.Equal(t, "UPDATE", history.Action)
		assert.Contains(t, string(history.OldValue), "Jl. Mawar 1", "snapshot lama tersimpan di history")
		assert.Contains(t, string(history.NewValue), "Jl. Melati 9")
	}
}

func TestUpdateProfilBank_HapusFoto(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	bank := bnkBank(bnkBSU, models.BSU)
	bank.PhotoURL = "https://cdn/lama.png"
	repo.On("FindBank", bnkBSU).Return(bank, nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.UpdateProfilBank(bnkBSU, services.UpdateProfilBankInput{
		AdminID: bnkActor, HapusFoto: true,
	})

	assert.NoError(t, err)
	assert.Equal(t, "", hasil.PhotoURL)
}

func TestUpdateProfilBank_FotoBaruMengalahkanHapusFoto(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	bank := bnkBank(bnkBSU, models.BSU)
	bank.PhotoURL = "https://cdn/lama.png"
	repo.On("FindBank", bnkBSU).Return(bank, nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.UpdateProfilBank(bnkBSU, services.UpdateProfilBankInput{
		AdminID: bnkActor, FotoURL: "https://cdn/baru.png", HapusFoto: true,
	})

	assert.NoError(t, err)
	assert.Equal(t, "https://cdn/baru.png", hasil.PhotoURL)
}

func TestUpdateProfilBank_SaveError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("SaveBank", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.UpdateProfilBank(bnkBSU, services.UpdateProfilBankInput{AdminID: bnkActor})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestGetProfilBank_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("GetProfilBank", bnkBSU).Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.GetProfilBank(bnkBSU)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bnkKode(t, err))
}

func TestGetProfilBank_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("GetProfilBank", bnkBSU).Return(nil, errors.New("db error"))

	hasil, err := svc.GetProfilBank(bnkBSU)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestGetProfilBank_Sukses(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("GetProfilBank", bnkBSU).
		Return(&repositories.BankProfilRow{BankID: bnkBSU, NamaBank: "BSU Mawar", IsActive: true}, nil)

	hasil, err := svc.GetProfilBank(bnkBSU)

	assert.NoError(t, err)
	assert.Equal(t, "BSU Mawar", hasil.NamaBank)
}

func TestGetHistoryAkunBank_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("GetHistoryAkunBank", bnkBSU).Return(nil, errors.New("db error"))

	hasil, err := svc.GetHistoryAkunBank(bnkBSU)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestGetHistoryAkunBank_Sukses(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("GetHistoryAkunBank", bnkBSU).
		Return([]repositories.HistoryBankRow{{HistoryBankID: "23010020010001", Action: "UPDATE"}}, nil)

	hasil, err := svc.GetHistoryAkunBank(bnkBSU)

	assert.NoError(t, err)
	assert.Len(t, hasil, 1)
}

func TestDeleteBank_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(nil, gorm.ErrRecordNotFound)

	err := svc.DeleteBank(bnkBSU)

	assert.Equal(t, http.StatusNotFound, bnkKode(t, err))
}

func TestDeleteBank_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindBank", bnkBSU).Return(nil, errors.New("db error"))

	err := svc.DeleteBank(bnkBSU)

	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestDeleteBank_MasihPunyaDataTerkait(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	bank := bnkBank(bnkBSU, models.BSU)
	repo.On("FindBank", bnkBSU).Return(bank, nil)
	repo.On("DeleteBank", bank).Return(repositories.ErrBankMasihDipakai)

	err := svc.DeleteBank(bnkBSU)

	assert.Equal(t, http.StatusConflict, bnkKode(t, err))
	assert.Contains(t, err.Error(), "data terkait")
}

func TestDeleteBank_DeleteError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	bank := bnkBank(bnkBSU, models.BSU)
	repo.On("FindBank", bnkBSU).Return(bank, nil)
	repo.On("DeleteBank", bank).Return(errors.New("db error"))

	err := svc.DeleteBank(bnkBSU)

	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestDeleteBank_Sukses(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	bank := bnkBank(bnkBSU, models.BSU)
	repo.On("FindBank", bnkBSU).Return(bank, nil)
	repo.On("DeleteBank", bank).Return(nil)

	err := svc.DeleteBank(bnkBSU)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetDetailBank_BankIDKosong(t *testing.T) {
	_, _, _, svc := newBnkSvc(t)

	hasil, err := svc.GetDetailBank("")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bnkKode(t, err))
}

func TestGetDetailBank_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindBankWithRelasi", bnkBSU).Return(nil, errors.New("not found"))

	hasil, err := svc.GetDetailBank(bnkBSU)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bnkKode(t, err))
}

func TestGetDetailBank_ErrorStaff(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindBankWithRelasi", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("ListAdminWithUserByBank", bnkBSU).Return(nil, errors.New("db error"))

	hasil, err := svc.GetDetailBank(bnkBSU)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	assert.Contains(t, err.Error(), "staff")
}

func TestGetDetailBank_LabelJenisBank(t *testing.T) {
	kasus := []struct {
		jenis     models.JenisBank
		wantLabel string
		wantIsBSU bool
	}{
		{models.BSI, "Bank Sampah Induk (BSI)", false},
		{models.BSM, "Bank Sampah Mandiri (BSM)", false},
		{models.BSU, "Bank Sampah Unit (BSU)", true},
	}
	for _, k := range kasus {
		t.Run(string(k.jenis), func(t *testing.T) {
			_, repo, _, svc := newBnkSvc(t)
			repo.On("FindBankWithRelasi", bnkBSU).Return(bnkBank(bnkBSU, k.jenis), nil)
			repo.On("ListAdminWithUserByBank", bnkBSU).Return([]models.Admin{}, nil)

			hasil, err := svc.GetDetailBank(bnkBSU)

			assert.NoError(t, err)
			assert.Equal(t, k.wantLabel, hasil.JenisBank)
			assert.Equal(t, k.wantIsBSU, hasil.IsBSU)
		})
	}
}

func TestGetDetailBank_BankIndukDanWilayah(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	bank := bnkBank(bnkBSU, models.BSU)
	bank.ParentBank = &models.BankSampah{BankID: bnkBSI, NamaBank: "BSI Utama"}
	bank.Kecamatan = &models.Kecamatan{Kecamatan: "Padang Barat"}
	bank.Kelurahan = &models.Kelurahan{Kelurahan: "Purus"}
	repo.On("FindBankWithRelasi", bnkBSU).Return(bank, nil)
	repo.On("ListAdminWithUserByBank", bnkBSU).
		Return([]models.Admin{*bnkAdmin("33301002001001", models.PetugasBSU, models.Aktif)}, nil)

	hasil, err := svc.GetDetailBank(bnkBSU)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSI Utama", hasil.BankIndukNama)
		assert.Equal(t, "Padang Barat", hasil.Kecamatan)
		assert.Equal(t, "Purus", hasil.Kelurahan)
		if assert.Len(t, hasil.Admins, 1) {
			assert.Equal(t, "Budi Santoso", hasil.Admins[0].NamaAdmin)
			assert.Equal(t, "Petugas", hasil.Admins[0].RoleAdmin, "role diringkas jadi Admin/Petugas")
		}
	}
}

func TestGetDetailBank_RelasiNilAman(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindBankWithRelasi", bnkBSI).Return(bnkBank(bnkBSI, models.BSI), nil)
	repo.On("ListAdminWithUserByBank", bnkBSI).Return([]models.Admin{}, nil)

	hasil, err := svc.GetDetailBank(bnkBSI)

	assert.NoError(t, err)
	assert.Equal(t, "", hasil.BankIndukNama)
	assert.Equal(t, "", hasil.Kecamatan)
	assert.Len(t, hasil.Admins, 0)
}

func TestListAdminBank_RepoError(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListAdminBank", bnkBSU).Return(nil, errors.New("db error"))

	hasil, err := svc.ListAdminBank(bnkBSU)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestListAdminBank_Sukses(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("ListAdminBank", bnkBSU).
		Return([]repositories.AdminBankRow{{AdminID: "33301002001001", Nama: "Budi Santoso"}}, nil)

	hasil, err := svc.ListAdminBank(bnkBSU)

	assert.NoError(t, err)
	assert.Len(t, hasil, 1)
}

func bnkAddAdminInput(role models.RoleAdmin) services.AddAdminBankInput {
	return services.AddAdminBankInput{
		UserID: bnkUser, BankID: bnkBSU, Role: string(role), ActorAdminID: bnkActor,
	}
}

func TestAddAdminBank_RoleTidakValid(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.SuperAdmin))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bnkKode(t, err))
	assert.Contains(t, err.Error(), "Role tidak valid")
	repo.AssertNotCalled(t, "FindUser", mock.Anything)
}

func TestAddAdminBank_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(nil, errors.New("not found"))

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.PetugasBSU))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bnkKode(t, err))
}

func TestAddAdminBank_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(nil, errors.New("not found"))

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.PetugasBSU))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bnkKode(t, err))
}

func TestAddAdminBank_SudahPunyaAdminAktif(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(1), nil)
	expectTxRollback(dbMock)

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.PetugasBSU))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, bnkKode(t, err))
	assert.Contains(t, err.Error(), "admin aktif")
}

func TestAddAdminBank_PernahTerdaftarDanNonaktif(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(0), nil)
	repo.On("FindAdminByUserAndBankTx", mock.Anything, bnkUser, bnkBSU).
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Nonaktif), nil)
	expectTxRollback(dbMock)

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.PetugasBSU))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, bnkKode(t, err))
	assert.Contains(t, err.Error(), "fitur aktivasi")
}

func TestAddAdminBank_SudahTerdaftarDiBankIni(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(0), nil)
	repo.On("FindAdminByUserAndBankTx", mock.Anything, bnkUser, bnkBSU).
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Pending), nil)
	expectTxRollback(dbMock)

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.PetugasBSU))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, bnkKode(t, err))
	assert.Contains(t, err.Error(), "sudah terdaftar")
}

func TestAddAdminBank_CekExistingErrorLain(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(0), nil)
	repo.On("FindAdminByUserAndBankTx", mock.Anything, bnkUser, bnkBSU).Return(nil, errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.PetugasBSU))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	repo.AssertNotCalled(t, "CreateAdmin", mock.Anything, mock.Anything)
}

func TestAddAdminBank_Sukses(t *testing.T) {
	dbMock, repo, mailerMock, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(0), nil)
	repo.On("FindAdminByUserAndBankTx", mock.Anything, bnkUser, bnkBSU).Return(nil, gorm.ErrRecordNotFound)

	var adminDibuat *models.Admin
	repo.On("CreateAdmin", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { adminDibuat = args.Get(1).(*models.Admin) }).Return(nil)
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, mock.Anything).Return(nil)

	var history *models.HistoryAkunBank
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { history = args.Get(1).(*models.HistoryAkunBank) }).Return(nil)
	mailerMock.On("SendEmail", mock.Anything).Return(nil).Maybe()

	dbMock.ExpectBegin()
	expectQueries(dbMock, 2)
	dbMock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	expectQueries(dbMock, 1)
	dbMock.ExpectCommit()

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.PetugasBSU))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.PetugasBSU, hasil.Role)
		assert.Equal(t, models.Pending, hasil.StatusAdmin)
	}
	assert.NotNil(t, adminDibuat)
	if assert.NotNil(t, history) {
		assert.Equal(t, "CREATE", history.Action)
		assert.Contains(t, history.Keterangan, "Admin Utama", "nama aktor ikut tercatat di audit")
		assert.Contains(t, history.Informasi, "Budi Santoso")
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestAddAdminBank_NamaAktorFallbackKeID(t *testing.T) {
	dbMock, repo, mailerMock, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("")
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(0), nil)
	repo.On("FindAdminByUserAndBankTx", mock.Anything, bnkUser, bnkBSU).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateAdmin", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, mock.Anything).Return(nil)

	var history *models.HistoryAkunBank
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { history = args.Get(1).(*models.HistoryAkunBank) }).Return(nil)
	mailerMock.On("SendEmail", mock.Anything).Return(nil).Maybe()

	dbMock.ExpectBegin()
	expectQueries(dbMock, 2)
	dbMock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	expectQueries(dbMock, 1)
	dbMock.ExpectCommit()

	_, err := svc.AddAdminBank(bnkAddAdminInput(models.AdminBSU))

	assert.NoError(t, err)
	assert.Contains(t, history.Keterangan, bnkActor)
}

func TestAddAdminBank_HistoryError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("CountAdminAktifPending", mock.Anything, bnkUser).Return(int64(0), nil)
	repo.On("FindAdminByUserAndBankTx", mock.Anything, bnkUser, bnkBSU).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateAdmin", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(errors.New("db error"))

	dbMock.ExpectBegin()
	expectQueries(dbMock, 2)
	dbMock.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	expectQueries(dbMock, 1)
	dbMock.ExpectRollback()

	hasil, err := svc.AddAdminBank(bnkAddAdminInput(models.PetugasBSU))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	assert.Contains(t, err.Error(), "audit history")
}

func TestDeleteStaffBank_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bnkKode(t, err))
}

func TestDeleteStaffBank_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").Return(nil, errors.New("db error"))

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestDeleteStaffBank_TanpaBank(t *testing.T) {
	_, repo, _, svc := newBnkSvc(t)
	admin := bnkAdmin("33301002001001", models.PetugasBSU, models.Aktif)
	admin.BankID = nil
	repo.On("FindAdminWithUserBank", "33301002001001").Return(admin, nil)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bnkKode(t, err))
	assert.Contains(t, err.Error(), "tidak terhubung")
}

func TestDeleteStaffBank_PendingHardDelete(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Pending), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("AdminPernahAktif", "33301002001001").Return(false, nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, (*models.TujuanEnum)(nil)).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteAdmin", mock.Anything, "33301002001001").Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Budi Santoso", hasil.StaffName)
		assert.Equal(t, "BSU Mawar", hasil.BankName)
	}
	repo.AssertNotCalled(t, "UpdateStatusAdmin", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestDeleteStaffBank_AktifSoftDelete(t *testing.T) {
	dbMock, repo, mailerMock, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Aktif), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")

	var history *models.HistoryAkunBank
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { history = args.Get(1).(*models.HistoryAkunBank) }).Return(nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "33301002001001", models.Nonaktif).Return(nil)
	mailerMock.On("SendEmail", mock.Anything).Return(nil).Maybe()
	expectTx(dbMock, 1)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertNotCalled(t, "DeleteAdmin", mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "DeleteAktivasiAdminBelumDipakai", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "AdminPernahAktif", mock.Anything)
	if assert.NotNil(t, history) {
		assert.Equal(t, "DELETE", history.Action)
		assert.Contains(t, string(history.OldValue), "Budi Santoso", "snapshot staff tersimpan sebelum dihapus")
		assert.Equal(t, bnkActor, history.CreatedBy, "pelaku diambil dari token, bukan dari body")
	}
}

func TestDeleteStaffBank_FallbackNama(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	admin := bnkAdmin("33301002001001", models.PetugasBSU, models.Pending)
	admin.User = models.User{}
	admin.Bank = models.BankSampah{}
	repo.On("FindAdminWithUserBank", "33301002001001").Return(admin, nil)
	repo.On("FindUser", bnkUser).Return(bnkUserModel(), nil)
	repo.On("FindBank", bnkBSU).Return(bnkBank(bnkBSU, models.BSU), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("AdminPernahAktif", "33301002001001").Return(false, nil)
	repo.On("FindNamaAdmin", bnkActor).Return("")
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteAdmin", mock.Anything, "33301002001001").Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Budi Santoso", hasil.StaffName, "nama diambil ulang dari tabel users")
		assert.Equal(t, "Bank "+bnkBSU, hasil.BankName, "nama bank diambil ulang dari tabel bank")
	}
}

func TestDeleteStaffBank_FallbackNamaKeUserID(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	admin := bnkAdmin("33301002001001", models.PetugasBSU, models.Pending)
	admin.User = models.User{}
	repo.On("FindAdminWithUserBank", "33301002001001").Return(admin, nil)
	repo.On("FindUser", bnkUser).Return(nil, errors.New("not found"))
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("AdminPernahAktif", "33301002001001").Return(false, nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteAdmin", mock.Anything, "33301002001001").Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.NoError(t, err)
	assert.Equal(t, bnkUser, hasil.StaffName)
}

func TestDeleteStaffBank_HapusTokenError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Pending), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("AdminPernahAktif", "33301002001001").Return(false, nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, mock.Anything).
		Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	assert.Contains(t, err.Error(), "clean up activation")
}

func TestDeleteStaffBank_DeleteAdminError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Pending), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("AdminPernahAktif", "33301002001001").Return(false, nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, mock.Anything).Return(nil)
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteAdmin", mock.Anything, "33301002001001").Return(errors.New("db error"))
	dbMock.ExpectBegin()
	expectQueries(dbMock, 1)
	dbMock.ExpectRollback()

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
}

func TestDeleteStaffBank_UpdateStatusError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Aktif), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "33301002001001", models.Nonaktif).
		Return(errors.New("db error"))
	dbMock.ExpectBegin()
	expectQueries(dbMock, 1)
	dbMock.ExpectRollback()

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	assert.Contains(t, err.Error(), "deactivate")
}

func TestDeleteStaffBank_PelakuTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Aktif), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, bnkKode(t, err))
	assert.NoError(t, dbMock.ExpectationsWereMet(), "transaksi gak boleh dibuka")
}

func TestDeleteStaffBank_BankLainDitolak(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Aktif), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSI), nil)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, bnkKode(t, err))
	assert.Contains(t, err.Error(), "bank sampah Anda sendiri")
	repo.AssertNotCalled(t, "CreateHistoryAkunBank", mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "UpdateStatusAdmin", mock.Anything, mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet(), "transaksi gak boleh dibuka")
}

func TestDeleteStaffBank_HapusDiriSendiriDitolak(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", bnkActor).
		Return(bnkAdmin(bnkActor, models.AdminBSU, models.Aktif), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)

	hasil, err := svc.DeleteStaffBank(bnkActor, bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, bnkKode(t, err))
	assert.Contains(t, err.Error(), "akun sendiri")
	repo.AssertNotCalled(t, "UpdateStatusAdmin", mock.Anything, mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet(), "transaksi gak boleh dibuka")
}

func TestDeleteStaffBank_SudahNonaktifDitolak(t *testing.T) {
	dbMock, repo, mailerMock, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Nonaktif), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bnkKode(t, err))
	assert.Contains(t, err.Error(), "sudah nonaktif")
	repo.AssertNotCalled(t, "CreateHistoryAkunBank", mock.Anything, mock.Anything)
	mailerMock.AssertNotCalled(t, "SendEmail", mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet(), "transaksi gak boleh dibuka")
}

func TestDeleteStaffBank_PendingReaktivasiDibatalkan(t *testing.T) {
	dbMock, repo, mailerMock, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Pending), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("AdminPernahAktif", "33301002001001").Return(true, nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("DeleteAktivasiAdminBelumDipakai", mock.Anything, bnkUser, (*models.TujuanEnum)(nil)).Return(nil)
	var history *models.HistoryAkunBank
	repo.On("CreateHistoryAkunBank", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { history = args.Get(1).(*models.HistoryAkunBank) }).Return(nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "33301002001001", models.Nonaktif).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertNotCalled(t, "DeleteAdmin", mock.Anything, mock.Anything)
	mailerMock.AssertNotCalled(t, "SendEmail", mock.Anything)
	if assert.NotNil(t, history) {
		assert.Equal(t, "DELETE", history.Action)
		assert.Contains(t, history.Informasi, "Reaktivasi")
		assert.Contains(t, history.Informasi, "dibatalkan")
		assert.Contains(t, history.Keterangan, "membatalkan reaktivasi")
	}
	repo.AssertExpectations(t)
}

func TestDeleteStaffBank_CekPernahAktifError(t *testing.T) {
	dbMock, repo, _, svc := newBnkSvc(t)
	repo.On("FindAdminWithUserBank", "33301002001001").
		Return(bnkAdmin("33301002001001", models.PetugasBSU, models.Pending), nil)
	repo.On("FindAdminAktifByUser", bnkActorUser).Return(bnkPelaku(bnkBSU), nil)
	repo.On("FindNamaAdmin", bnkActor).Return("Admin Utama")
	repo.On("AdminPernahAktif", "33301002001001").Return(false, errors.New("db error"))

	hasil, err := svc.DeleteStaffBank("33301002001001", bnkActorUser)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bnkKode(t, err))
	repo.AssertNotCalled(t, "DeleteAdmin", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet(), "transaksi gak boleh dibuka")
}
