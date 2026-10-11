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
	nsbBankBSU = "301002001"
	nsbBankBSI = "101002001"
	nsbUserID  = "1371010101010001"
	nsbEmail   = "budi@mail.com"
	nsbAdminID = "23301002001001"
)

func newNsbSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockNasabahRepo, *mocks.MockMailer, services.NasabahService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockNasabahRepo)
	mailerMock := new(mocks.MockMailer)

	mailerMock.On("SendEmail", mock.Anything).Return(nil).Maybe()

	return dbMock, repo, mailerMock, services.NewNasabahService(gdb, repo, mailerMock)
}

func nsbBank(bankID string, jenis models.JenisBank) *models.BankSampah {
	return &models.BankSampah{BankID: bankID, NamaBank: "Bank " + bankID, JenisBank: jenis, IsActive: true}
}

func nsbUser() *models.User {
	return &models.User{UserID: nsbUserID, Nama: "Budi Santoso", Email: nsbEmail, NoWhatsapp: "081234567890"}
}

func nsbNasabah(status models.StatusAkun) *models.Nasabah {
	return &models.Nasabah{
		NasabahID: "433010020010001", BankID: nsbBankBSU, UserID: nsbUserID,
		NomorRekening: "REK-001", StatusNasabah: status,
	}
}

func nsbRewardUang() *models.Reward {
	return &models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang}
}

func nsbRewardBarang() *models.Reward {
	return &models.Reward{RewardID: 2, NamaReward: models.RewardEnumBarang}
}

func nsbBaruInput() services.DaftarNasabahBaruInput {
	return services.DaftarNasabahBaruInput{
		UserID: nsbUserID, Nama: "Budi Santoso", Email: nsbEmail, NoWhatsapp: "081234567890",
		BankID: nsbBankBSU, AdminID: nsbAdminID, SaldoRupiah: 50000, SaldoPoin: 120,
	}
}

func nsbLamaInput() services.DaftarNasabahUserLamaInput {
	return services.DaftarNasabahUserLamaInput{
		UserID: nsbUserID, BankID: nsbBankBSU, AdminID: nsbAdminID,
		SaldoRupiah: 50000, SaldoPoin: 120,
	}
}

func nsbStubPendaftaran(repo *mocks.MockNasabahRepo) {
	repo.On("CreateNasabah", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumUang).Return(nsbRewardUang(), nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumBarang).Return(nsbRewardBarang(), nil)
	repo.On("CreateSaldoRekening", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateRiwayatSaldo", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteAktivasiNasabahBelumDipakai", mock.Anything, nsbUserID).Return(nil)
}

func expectTxPendaftaran(m sqlmock.Sqlmock, commit bool) {
	m.ExpectBegin()
	for i := 0; i < 4; i++ {
		m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	}
	m.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	if commit {
		m.ExpectCommit()
	} else {
		m.ExpectRollback()
	}
}

func nsbKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func TestDaftarNasabahBaru_NIKSudahTerdaftar(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(1), nil)

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, nsbKode(t, err))
	assert.Contains(t, err.Error(), "akun lama")
	repo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
}

func TestDaftarNasabahBaru_CekUserError(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), errors.New("db error"))

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestDaftarNasabahBaru_EmailSudahDipakai(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(1), nil)

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, nsbKode(t, err))
	assert.Contains(t, err.Error(), "Email")
}

func TestDaftarNasabahBaru_CekEmailError(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), errors.New("db error"))

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestDaftarNasabahBaru_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nil, errors.New("not found"))

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, nsbKode(t, err))
}

func TestDaftarNasabahBaru_CreateUserError(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
	assert.Contains(t, err.Error(), "create user")
}

func TestDaftarNasabahBaru_BankIDTidakValidUntukGenerateID(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", "901002001").Return(nsbBank("901002001", models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	expectTxRollback(dbMock)

	in := nsbBaruInput()
	in.BankID = "901002001"
	hasil, err := svc.DaftarNasabahBaru(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
	assert.Contains(t, err.Error(), "generate nasabah ID")
	repo.AssertNotCalled(t, "CreateNasabah", mock.Anything, mock.Anything)
}

func TestDaftarNasabahBaru_CreateNasabahError(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateNasabah", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"nasabah_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
	assert.Contains(t, err.Error(), "create nasabah")
}

func TestDaftarNasabahBaru_RewardTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateNasabah", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumUang).Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"nasabah_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
	assert.Contains(t, err.Error(), "Reward")
}

func TestDaftarNasabahBaru_CreateSaldoError(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateNasabah", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumUang).Return(nsbRewardUang(), nil)
	repo.On("CreateSaldoRekening", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"nasabah_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
	assert.Contains(t, err.Error(), "saldo rupiah")
}

func TestDaftarNasabahBaru_BersihkanTokenLamaGagal(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateNasabah", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumUang).Return(nsbRewardUang(), nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumBarang).Return(nsbRewardBarang(), nil)
	repo.On("CreateSaldoRekening", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateRiwayatSaldo", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteAktivasiNasabahBelumDipakai", mock.Anything, nsbUserID).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	for i := 0; i < 3; i++ {
		dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}))
	}
	dbMock.ExpectRollback()

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
	assert.Contains(t, err.Error(), "old activation")
}

func TestDaftarNasabahBaru_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)

	var userDibuat *models.User
	repo.On("CreateUser", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { userDibuat = args.Get(1).(*models.User) }).Return(nil)
	nsbStubPendaftaran(repo)
	expectTxPendaftaran(dbMock, true)

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "433010020010001", hasil.NasabahID, "nasabah_id = kode 43 (BSU) + bank_id + urutan")
		assert.Equal(t, nsbBankBSU, hasil.BankID)
		assert.Equal(t, nsbUserID, hasil.UserID)
		assert.Equal(t, models.Pending, hasil.StatusNasabah, "pending sampai OTP diaktivasi")
	}
	if assert.NotNil(t, userDibuat) {
		assert.Equal(t, "Budi Santoso", userDibuat.Nama)
		assert.Empty(t, userDibuat.Password, "password di-set saat aktivasi")
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDaftarNasabahBaru_NomorRekeningDefaultIkutNasabahID(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	nsbStubPendaftaran(repo)
	expectTxPendaftaran(dbMock, true)

	hasil, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.NoError(t, err)
	assert.Equal(t, hasil.NasabahID, hasil.NomorRekening)
}

func TestDaftarNasabahBaru_NomorRekeningDariAdminDipakai(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	nsbStubPendaftaran(repo)
	expectTxPendaftaran(dbMock, true)

	in := nsbBaruInput()
	in.NoRekening = "REK-MANUAL-01"
	hasil, err := svc.DaftarNasabahBaru(in)

	assert.NoError(t, err)
	assert.Equal(t, "REK-MANUAL-01", hasil.NomorRekening)
}

func TestDaftarNasabahBaru_DuaRekeningSaldoDibuat(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("CountUser", nsbUserID).Return(int64(0), nil)
	repo.On("CountUserByEmail", nsbEmail).Return(int64(0), nil)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateUser", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateNasabah", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumUang).Return(nsbRewardUang(), nil)
	repo.On("FindRewardByNama", mock.Anything, models.RewardEnumBarang).Return(nsbRewardBarang(), nil)

	var saldo []*models.SaldoRekening
	repo.On("CreateSaldoRekening", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { saldo = append(saldo, args.Get(1).(*models.SaldoRekening)) }).Return(nil)

	var riwayat []*models.RiwayatArusSaldo
	repo.On("CreateRiwayatSaldo", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { riwayat = append(riwayat, args.Get(1).(*models.RiwayatArusSaldo)) }).Return(nil)
	repo.On("DeleteAktivasiNasabahBelumDipakai", mock.Anything, nsbUserID).Return(nil)
	expectTxPendaftaran(dbMock, true)

	_, err := svc.DaftarNasabahBaru(nsbBaruInput())

	assert.NoError(t, err)
	if assert.Len(t, saldo, 2) {
		assert.Equal(t, models.SatuanRewardEnumRp, saldo[0].SatuanNominalSaldo)
		assert.Equal(t, float64(50000), saldo[0].NominalSaldo)
		assert.Equal(t, "1433010020010001", saldo[0].RekeningID, "rekening_id = reward_id + nasabah_id")
		assert.Equal(t, models.SatuanRewardEnumPoin, saldo[1].SatuanNominalSaldo)
		assert.Equal(t, float64(120), saldo[1].NominalSaldo)
		assert.Equal(t, "2433010020010001", saldo[1].RekeningID)
		for _, s := range saldo {
			assert.Equal(t, models.EntitasNasabah, s.Entitas)
			if assert.NotNil(t, s.LastUpdatedBy) {
				assert.Equal(t, nsbAdminID, *s.LastUpdatedBy)
			}
		}
	}
	if assert.Len(t, riwayat, 2) {
		assert.Equal(t, float64(0), riwayat[0].NominalSebelum, "saldo awal selalu mulai dari 0")
		assert.Equal(t, float64(50000), riwayat[0].NominalSesudah)
		assert.Equal(t, float64(120), riwayat[1].NominalSesudah)
		assert.Equal(t, "Saldo awal nasabah", riwayat[0].Keterangan)
	}
}

func TestDaftarNasabahUserLama_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nil, errors.New("not found"))

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, nsbKode(t, err))
}

func TestDaftarNasabahUserLama_SudahPunyaAkunAktif(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(1), nil)

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, nsbKode(t, err))
	assert.Contains(t, err.Error(), "nasabah aktif")
}

func TestDaftarNasabahUserLama_CekAktifError(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), errors.New("db error"))

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestDaftarNasabahUserLama_MasihPendingDiBankLain(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSU).Return(int64(1), nil)

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, nsbKode(t, err))
	assert.Contains(t, err.Error(), "bank sampah lain")
}

func TestDaftarNasabahUserLama_CekPendingError(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSU).Return(int64(0), errors.New("db error"))

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestDaftarNasabahUserLama_PernahTerdaftarDanNonaktif(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSU).Return(int64(0), nil)
	repo.On("FindNasabahByUserAndBank", nsbUserID, nsbBankBSU).Return(nsbNasabah(models.Nonaktif), nil)

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, nsbKode(t, err))
	assert.Contains(t, err.Error(), "fitur aktivasi")
}

func TestDaftarNasabahUserLama_SudahTerdaftarDiBankIni(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSU).Return(int64(0), nil)
	repo.On("FindNasabahByUserAndBank", nsbUserID, nsbBankBSU).Return(nsbNasabah(models.Pending), nil)

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, nsbKode(t, err))
	assert.Contains(t, err.Error(), "sudah terdaftar")
}

func TestDaftarNasabahUserLama_CekExistingErrorLain(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSU).Return(int64(0), nil)
	repo.On("FindNasabahByUserAndBank", nsbUserID, nsbBankBSU).Return(nil, errors.New("db error"))

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
	repo.AssertNotCalled(t, "CreateNasabah", mock.Anything, mock.Anything)
}

func TestDaftarNasabahUserLama_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSU).Return(int64(0), nil)
	repo.On("FindNasabahByUserAndBank", nsbUserID, nsbBankBSU).Return(nil, gorm.ErrRecordNotFound)
	repo.On("FindBank", nsbBankBSU).Return(nil, errors.New("not found"))

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, nsbKode(t, err))
}

func TestDaftarNasabahUserLama_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSU).Return(int64(0), nil)
	repo.On("FindNasabahByUserAndBank", nsbUserID, nsbBankBSU).Return(nil, gorm.ErrRecordNotFound)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	nsbStubPendaftaran(repo)
	expectTxPendaftaran(dbMock, true)

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "433010020010001", hasil.NasabahID)
		assert.Equal(t, models.Pending, hasil.StatusNasabah)
	}
	repo.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestDaftarNasabahUserLama_KodeNasabahIkutJenisBank(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSI).Return(int64(0), nil)
	repo.On("FindNasabahByUserAndBank", nsbUserID, nsbBankBSI).Return(nil, gorm.ErrRecordNotFound)
	repo.On("FindBank", nsbBankBSI).Return(nsbBank(nsbBankBSI, models.BSI), nil)
	nsbStubPendaftaran(repo)
	expectTxPendaftaran(dbMock, true)

	in := nsbLamaInput()
	in.BankID = nsbBankBSI
	hasil, err := svc.DaftarNasabahUserLama(in)

	assert.NoError(t, err)
	assert.Equal(t, "411010020010001", hasil.NasabahID)
}

func TestDaftarNasabahUserLama_CreateNasabahError(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("FindUser", nsbUserID).Return(nsbUser(), nil)
	repo.On("CountNasabahAktif", nsbUserID).Return(int64(0), nil)
	repo.On("CountNasabahPendingDiBankLain", nsbUserID, nsbBankBSU).Return(int64(0), nil)
	repo.On("FindNasabahByUserAndBank", nsbUserID, nsbBankBSU).Return(nil, gorm.ErrRecordNotFound)
	repo.On("FindBank", nsbBankBSU).Return(nsbBank(nsbBankBSU, models.BSU), nil)
	repo.On("CreateNasabah", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"nasabah_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.DaftarNasabahUserLama(nsbLamaInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestHapusNasabah_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindNasabah", "433010020010001").Return(nil, gorm.ErrRecordNotFound)

	err := svc.HapusNasabah("433010020010001")

	assert.Equal(t, http.StatusNotFound, nsbKode(t, err))
}

func TestHapusNasabah_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindNasabah", "433010020010001").Return(nil, errors.New("db error"))

	err := svc.HapusNasabah("433010020010001")

	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestHapusNasabah_CekTransaksiError(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindNasabah", "433010020010001").Return(nsbNasabah(models.Aktif), nil)
	repo.On("CountTransaksiNasabah", "433010020010001").Return(int64(0), errors.New("db error"))

	err := svc.HapusNasabah("433010020010001")

	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestHapusNasabah_MasihPunyaTransaksi(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("FindNasabah", "433010020010001").Return(nsbNasabah(models.Aktif), nil)
	repo.On("CountTransaksiNasabah", "433010020010001").Return(int64(3), nil)

	err := svc.HapusNasabah("433010020010001")

	assert.Equal(t, http.StatusConflict, nsbKode(t, err))
	assert.Contains(t, err.Error(), "riwayat transaksi")
	repo.AssertNotCalled(t, "DeleteNasabah", mock.Anything, mock.Anything)
}

func TestHapusNasabah_PluckRekeningError(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("FindNasabah", "433010020010001").Return(nsbNasabah(models.Aktif), nil)
	repo.On("CountTransaksiNasabah", "433010020010001").Return(int64(0), nil)
	repo.On("PluckRekeningIDs", mock.Anything, "433010020010001").Return(nil, errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.HapusNasabah("433010020010001")

	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestHapusNasabah_DeleteRiwayatError(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("FindNasabah", "433010020010001").Return(nsbNasabah(models.Aktif), nil)
	repo.On("CountTransaksiNasabah", "433010020010001").Return(int64(0), nil)
	repo.On("PluckRekeningIDs", mock.Anything, "433010020010001").
		Return([]string{"1433010020010001"}, nil)
	repo.On("DeleteRiwayatByRekening", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.HapusNasabah("433010020010001")

	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
	repo.AssertNotCalled(t, "DeleteNasabah", mock.Anything, mock.Anything)
}

func TestHapusNasabah_DeleteSaldoError(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	repo.On("FindNasabah", "433010020010001").Return(nsbNasabah(models.Aktif), nil)
	repo.On("CountTransaksiNasabah", "433010020010001").Return(int64(0), nil)
	repo.On("PluckRekeningIDs", mock.Anything, "433010020010001").Return([]string{}, nil)
	repo.On("DeleteRiwayatByRekening", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteSaldoRekening", mock.Anything, "433010020010001").Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.HapusNasabah("433010020010001")

	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestHapusNasabah_UrutanPenghapusan(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	nasabah := nsbNasabah(models.Aktif)
	repo.On("FindNasabah", "433010020010001").Return(nasabah, nil)
	repo.On("CountTransaksiNasabah", "433010020010001").Return(int64(0), nil)
	repo.On("PluckRekeningIDs", mock.Anything, "433010020010001").
		Return([]string{"1433010020010001", "2433010020010001"}, nil)

	urutan := []string{}
	var rekeningDihapus []string
	repo.On("DeleteRiwayatByRekening", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			urutan = append(urutan, "riwayat")
			rekeningDihapus = args.Get(1).([]string)
		}).Return(nil)
	repo.On("DeleteSaldoRekening", mock.Anything, "433010020010001").
		Run(func(mock.Arguments) { urutan = append(urutan, "saldo") }).Return(nil)
	repo.On("DeleteNasabah", mock.Anything, nasabah).
		Run(func(mock.Arguments) { urutan = append(urutan, "nasabah") }).Return(nil)
	expectTx(dbMock, 0)

	err := svc.HapusNasabah("433010020010001")

	assert.NoError(t, err)
	assert.Equal(t, []string{"riwayat", "saldo", "nasabah"}, urutan)
	assert.Equal(t, []string{"1433010020010001", "2433010020010001"}, rekeningDihapus)
}

func TestHapusNasabah_DeleteNasabahError(t *testing.T) {
	dbMock, repo, _, svc := newNsbSvc(t)
	nasabah := nsbNasabah(models.Aktif)
	repo.On("FindNasabah", "433010020010001").Return(nasabah, nil)
	repo.On("CountTransaksiNasabah", "433010020010001").Return(int64(0), nil)
	repo.On("PluckRekeningIDs", mock.Anything, "433010020010001").Return([]string{}, nil)
	repo.On("DeleteRiwayatByRekening", mock.Anything, mock.Anything).Return(nil)
	repo.On("DeleteSaldoRekening", mock.Anything, "433010020010001").Return(nil)
	repo.On("DeleteNasabah", mock.Anything, nasabah).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.HapusNasabah("433010020010001")

	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestGetProfilNasabah_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("GetProfilNasabah", "433010020010001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.GetProfilNasabah("433010020010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, nsbKode(t, err))
}

func TestGetProfilNasabah_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("GetProfilNasabah", "433010020010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetProfilNasabah("433010020010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestGetProfilNasabah_Sukses(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("GetProfilNasabah", "433010020010001").Return(&repositories.ProfilNasabahRow{
		NasabahID: "433010020010001", Nama: "Budi Santoso", NomorRekening: "REK-001",
		StatusNasabah: string(models.Aktif),
	}, nil)

	hasil, err := svc.GetProfilNasabah("433010020010001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Budi Santoso", hasil.Nama)
		assert.Equal(t, "REK-001", hasil.NomorRekening)
	}
}

func TestGetDetailNasabah_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("GetDetailNasabah", "433010020010001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.GetDetailNasabah("433010020010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, nsbKode(t, err))
}

func TestGetDetailNasabah_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("GetDetailNasabah", "433010020010001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetDetailNasabah("433010020010001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, nsbKode(t, err))
}

func TestGetDetailNasabah_Sukses(t *testing.T) {
	_, repo, _, svc := newNsbSvc(t)
	repo.On("GetDetailNasabah", "433010020010001").Return(&repositories.DetailNasabahRow{
		NasabahID: "433010020010001", Nama: "Budi Santoso", NamaBank: "BSU Mawar",
		IsAdmin: true, AdminID: "33301002001001", RoleAdmin: string(models.PetugasBSU),
	}, nil)

	hasil, err := svc.GetDetailNasabah("433010020010001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSU Mawar", hasil.NamaBank)
		assert.True(t, hasil.IsAdmin, "nasabah yang juga petugas ikut terbawa di detail")
	}
}
