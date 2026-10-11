package services_test

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const (
	aknOTP      = "483920"
	aknOTPSalah = "999999"
	aknPassword = "rahasia123"
)

var (
	aknHashOnce   sync.Once
	aknHashedOTP  string
	aknHashedPass string
)

func aknHash(t *testing.T) (otpHash, passHash string) {
	t.Helper()
	aknHashOnce.Do(func() {
		h1, err := bcrypt.GenerateFromPassword([]byte(aknOTP), bcrypt.MinCost)
		if err != nil {
			t.Fatalf("hash OTP: %v", err)
		}
		h2, err := bcrypt.GenerateFromPassword([]byte(aknPassword), bcrypt.MinCost)
		if err != nil {
			t.Fatalf("hash password: %v", err)
		}
		aknHashedOTP, aknHashedPass = string(h1), string(h2)
	})
	return aknHashedOTP, aknHashedPass
}

func newAknSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockAuthRepo, *mocks.MockMailer, services.AkunService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockAuthRepo)
	mailerMock := new(mocks.MockMailer)

	mailerMock.On("SendEmail", mock.Anything).Return(nil).Maybe()

	return dbMock, repo, mailerMock, services.NewAkunService(gdb, repo, mailerMock)
}

func aknUser() *models.User {
	return &models.User{UserID: "USR-001", Nama: "Budi Santoso", Email: "budi@mail.com"}
}

func aknUserBerpassword(t *testing.T) *models.User {
	t.Helper()
	_, pass := aknHash(t)
	u := aknUser()
	u.Password = pass
	return u
}

func aknNasabah(status models.StatusAkun) *models.Nasabah {
	return &models.Nasabah{
		NasabahID: "NSB-001", BankID: "BSU-001", UserID: "USR-001", StatusNasabah: status,
	}
}

func aknAdmin(status models.StatusAkun) *models.Admin {
	bankID := "BSU-001"
	return &models.Admin{
		AdminID: "ADM-001", BankID: &bankID, UserID: "USR-001",
		Role: models.PetugasBSU, StatusAdmin: status,
	}
}

func aknAktivasiNasabah(t *testing.T) models.AktivasiAkun {
	t.Helper()
	otpHash, _ := aknHash(t)
	return models.AktivasiAkun{
		AktivasiID: "USR-0012111", UserID: "USR-001", AsRole: models.RoleUserNasabah,
		NasabahID: ptrStr("NSB-001"), Token: otpHash, Tujuan: models.TujuanAktivasi,
		ExpiredAt: time.Now().Add(time.Hour),
	}
}

func aknAktivasiAdmin(t *testing.T) models.AktivasiAkun {
	t.Helper()
	otpHash, _ := aknHash(t)
	return models.AktivasiAkun{
		AktivasiID: "USR-0011111", UserID: "USR-001", AsRole: models.RoleUserAdmin,
		AdminID: ptrStr("ADM-001"), Token: otpHash, Tujuan: models.TujuanAktivasi,
		ExpiredAt: time.Now().Add(time.Hour),
	}
}

func aknAktivasiReset(t *testing.T) *models.AktivasiAkun {
	t.Helper()
	otpHash, _ := aknHash(t)
	return &models.AktivasiAkun{
		AktivasiID: "USR-0013211", UserID: "USR-001", AsRole: models.RoleUserGlobal,
		Token: otpHash, Tujuan: models.TujuanResetPassword,
		ExpiredAt: time.Now().Add(10 * time.Minute),
	}
}

func aknBank() *models.BankSampah {
	return &models.BankSampah{BankID: "BSU-001", NamaBank: "BSU Mawar", JenisBank: models.BSU}
}

func aknKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func TestAktivasiAkun_TidakAdaKandidat(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{}, nil)

	pesan, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Empty(t, pesan)
	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestAktivasiAkun_ListError(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return(nil, errors.New("db error"))

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestAktivasiAkun_OTPSalah(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTPSalah, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
	assert.Contains(t, err.Error(), "OTP tidak valid")
}

func TestAktivasiAkun_OTPMemilihKandidatYangCocok(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	otpLain, err := bcrypt.GenerateFromPassword([]byte("111111"), bcrypt.MinCost)
	assert.NoError(t, err)

	duluan := aknAktivasiAdmin(t)
	duluan.Token = string(otpLain)
	kandidat := []models.AktivasiAkun{duluan, aknAktivasiNasabah(t)}

	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).Return(kandidat, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
	repo.On("CountNasabahAktifSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(0), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Aktif).Return(int64(1), nil)
	repo.On("MarkAktivasiUsed", mock.Anything, "USR-0012111").Return(nil)
	expectTx(dbMock, 0)

	pesan, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.NoError(t, err)
	assert.Contains(t, pesan, "nasabah")
	repo.AssertExpectations(t)
}

func TestAktivasiAkun_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestAktivasiAkun_NasabahIDKosong(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	rusak := aknAktivasiNasabah(t)
	rusak.NasabahID = nil
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{rusak}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
	assert.Contains(t, err.Error(), "nasabah_id kosong")
}

func TestAktivasiAkun_NasabahPendingTidakAda(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(nil, errors.New("not found"))

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
	assert.Contains(t, err.Error(), "nasabah pending")
}

func TestAktivasiAkun_AdminIDKosong(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	rusak := aknAktivasiAdmin(t)
	rusak.AdminID = nil
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{rusak}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
	assert.Contains(t, err.Error(), "admin_id kosong")
}

func TestAktivasiAkun_AdminPendingTidakAda(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiAdmin(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindAdminPending", mock.Anything, "USR-001", "ADM-001").Return(nil, errors.New("not found"))

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestAktivasiAkun_RoleTidakDikenal(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	rusak := aknAktivasiNasabah(t)
	rusak.AsRole = models.RoleUserGlobal
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{rusak}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
	assert.Contains(t, err.Error(), "Tipe role aktivasi")
}

func TestAktivasiAkun_PasswordKosong_PasswordRequired(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
	expectTxRollback(dbMock)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
	assert.Equal(t, "PASSWORD_REQUIRED", authErrCode(t, err))
	repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "MarkAktivasiUsed", mock.Anything, mock.Anything)
}

func TestAktivasiAkun_PasswordLemah(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
	expectTxRollback(dbMock)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Password: "pendek", Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
	assert.Contains(t, err.Error(), "tidak memenuhi persyaratan")
	assert.Empty(t, authErrCode(t, err))
	repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
}

func TestAktivasiAkun_PasswordDiSetSaatAktivasiPertama(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
	repo.On("UpdatePassword", mock.Anything, "USR-001", mock.Anything).Return(nil)
	repo.On("CountNasabahAktifSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(0), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Aktif).Return(int64(1), nil)
	repo.On("MarkAktivasiUsed", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{
		UserID: "USR-001", OTP: aknOTP, Password: "PasswordBaru123", Platform: models.PlatformMobile,
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestAktivasiAkun_PasswordLamaTidakDitimpa(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
	repo.On("CountNasabahAktifSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(0), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Aktif).Return(int64(1), nil)
	repo.On("MarkAktivasiUsed", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 0)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{
		UserID: "USR-001", OTP: aknOTP, Password: "PasswordBaru123", Platform: models.PlatformMobile,
	})

	assert.NoError(t, err)
	repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
}

func TestAktivasiAkun_AdaNasabahAktifLain(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
	repo.On("CountNasabahAktifSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(1), nil)
	expectTxRollback(dbMock)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusConflict, aknKode(t, err))
	assert.Contains(t, err.Error(), "nasabah aktif di bank lain")
	repo.AssertNotCalled(t, "UpdateStatusNasabah", mock.Anything, mock.Anything, mock.Anything)
}

func TestAktivasiAkun_AdaAdminAktifLain(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiAdmin(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindAdminPending", mock.Anything, "USR-001", "ADM-001").Return(aknAdmin(models.Pending), nil)
	repo.On("CountAdminAktifSelain", mock.Anything, "USR-001", "ADM-001").Return(int64(1), nil)
	expectTxRollback(dbMock)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusConflict, aknKode(t, err))
	assert.Contains(t, err.Error(), "admin aktif")
}

func TestAktivasiAkun_CountError(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
	repo.On("CountNasabahAktifSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(0), errors.New("db error"))
	expectTxRollback(dbMock)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestAktivasiAkun_MarkUsedError(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
	repo.On("CountNasabahAktifSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(0), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Aktif).Return(int64(1), nil)
	repo.On("MarkAktivasiUsed", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestAktivasiAkun_SuksesAdmin(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiAdmin(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindAdminPending", mock.Anything, "USR-001", "ADM-001").Return(aknAdmin(models.Pending), nil)
	repo.On("CountAdminAktifSelain", mock.Anything, "USR-001", "ADM-001").Return(int64(0), nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "ADM-001", models.Aktif, false).Return(int64(1), nil)
	repo.On("MarkAktivasiUsed", mock.Anything, "USR-0011111").Return(nil)
	expectTx(dbMock, 0)

	pesan, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformMobile})

	assert.NoError(t, err)
	assert.Contains(t, pesan, "staff")
	repo.AssertExpectations(t)
}

func TestAktivasiAkun_AdminLewatWeb_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	admin := aknAdmin(models.Pending)
	admin.Role = models.AdminBSM
	repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
		Return([]models.AktivasiAkun{aknAktivasiAdmin(t)}, nil)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("FindAdminPending", mock.Anything, "USR-001", "ADM-001").Return(admin, nil)
	repo.On("CountAdminAktifSelain", mock.Anything, "USR-001", "ADM-001").Return(int64(0), nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "ADM-001", models.Aktif, false).Return(int64(1), nil)
	repo.On("MarkAktivasiUsed", mock.Anything, "USR-0011111").Return(nil)
	expectTx(dbMock, 0)

	pesan, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: models.PlatformWeb})

	assert.NoError(t, err)
	assert.Contains(t, pesan, "staff")
	repo.AssertExpectations(t)
}

func TestAktivasiAkun_PlatformTidakSesuai_Ditolak(t *testing.T) {
	kasus := []struct {
		nama      string
		adminRole models.RoleAdmin
		platform  string
	}{
		{"nasabah lewat web", "", models.PlatformWeb},
		{"nasabah tanpa platform", "", ""},
		{"petugas lewat web", models.PetugasBSU, models.PlatformWeb},
		{"admin bsm lewat mobile", models.AdminBSM, models.PlatformMobile},
		{"superadmin lewat mobile", models.SuperAdmin, models.PlatformMobile},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			_, repo, _, svc := newAknSvc(t)
			repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
			if k.adminRole == "" {
				repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
					Return([]models.AktivasiAkun{aknAktivasiNasabah(t)}, nil)
				repo.On("FindNasabahPending", mock.Anything, "USR-001", "NSB-001").Return(aknNasabah(models.Pending), nil)
			} else {
				admin := aknAdmin(models.Pending)
				admin.Role = k.adminRole
				repo.On("ListAktivasiValid", mock.Anything, "USR-001", models.TujuanAktivasi).
					Return([]models.AktivasiAkun{aknAktivasiAdmin(t)}, nil)
				repo.On("FindAdminPending", mock.Anything, "USR-001", "ADM-001").Return(admin, nil)
			}

			_, err := svc.AktivasiAkun(services.AktivasiAkunInput{UserID: "USR-001", OTP: aknOTP, Platform: k.platform})

			assert.Equal(t, http.StatusForbidden, aknKode(t, err))
			assert.Equal(t, "Peran akun anda tidak sesuai untuk platform ini", err.Error())
			repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
			repo.AssertNotCalled(t, "MarkAktivasiUsed", mock.Anything, mock.Anything)
		})
	}
}

func aknReaktivasiInput(role models.RoleUserEnum, roleID, actorBankID string) services.GenerateReaktivasiInput {
	return services.GenerateReaktivasiInput{
		UserID: "USR-001", Role: role, RoleID: roleID,
		AdminID: "ADM-PEMBUAT", ActorBankID: actorBankID,
	}
}

func TestGenerateReaktivasi_UserTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(nil, errors.New("not found"))
	expectTxRollback(dbMock)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", "BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestGenerateReaktivasi_RoleTidakValid(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	expectTxRollback(dbMock)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserGlobal, "NSB-001", ""))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
}

func TestGenerateReaktivasi_SudahPunyaNasabahAktif(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountNasabahAktif", mock.Anything, "USR-001").Return(int64(1), nil)
	expectTxRollback(dbMock)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", "BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, aknKode(t, err))
	assert.Contains(t, err.Error(), "nasabah aktif")
}

func TestGenerateReaktivasi_TargetNasabahTidakAda(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountNasabahAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindNasabahByStatus", mock.Anything, "USR-001", "NSB-001",
		[]models.StatusAkun{models.Nonaktif, models.Pending}).Return(nil, errors.New("not found"))
	expectTxRollback(dbMock)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", "BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestGenerateReaktivasi_BankLainDitolak(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountNasabahAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindNasabahByStatus", mock.Anything, "USR-001", "NSB-001", mock.Anything).
		Return(aknNasabah(models.Nonaktif), nil)
	expectTxRollback(dbMock)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", "BSU-999"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, aknKode(t, err))
	repo.AssertNotCalled(t, "UpdateStatusNasabah", mock.Anything, mock.Anything, mock.Anything)
}

func TestGenerateReaktivasi_SudahPunyaAdminAktif(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountAdminAktif", mock.Anything, "USR-001").Return(int64(1), nil)
	expectTxRollback(dbMock)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserAdmin, "ADM-001", "BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, aknKode(t, err))
	assert.Contains(t, err.Error(), "admin aktif")
}

func TestGenerateReaktivasi_TargetAdminTidakAda(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountAdminAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindAdminByStatus", mock.Anything, "USR-001", "ADM-001", mock.Anything).
		Return(nil, errors.New("not found"))
	expectTxRollback(dbMock)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserAdmin, "ADM-001", "BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestGenerateReaktivasi_AdminRefreshJoinedAt(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountAdminAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindAdminByStatus", mock.Anything, "USR-001", "ADM-001", mock.Anything).
		Return(aknAdmin(models.Nonaktif), nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "ADM-001", models.Pending, true).Return(int64(1), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("FindAktivasiValidByTarget", mock.Anything, "USR-001", models.TujuanAktivasi,
		(*string)(nil), ptrStr("ADM-001")).Return(nil, errors.New("not found"))
	repo.On("CreateAktivasi", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserAdmin, "ADM-001", "BSU-001"))

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertExpectations(t)
}

func TestGenerateReaktivasi_ReuseRowLama(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	lama := aknAktivasiNasabah(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountNasabahAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindNasabahByStatus", mock.Anything, "USR-001", "NSB-001", mock.Anything).
		Return(aknNasabah(models.Nonaktif), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Pending).Return(int64(1), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("FindAktivasiValidByTarget", mock.Anything, "USR-001", models.TujuanAktivasi,
		ptrStr("NSB-001"), (*string)(nil)).Return(&lama, nil)

	var fields map[string]interface{}
	repo.On("UpdateAktivasi", mock.Anything, "USR-0012111", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(2).(map[string]interface{}) }).Return(nil)
	expectTx(dbMock, 0)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", "BSU-001"))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "USR-0012111", hasil.AktivasiID, "row lama dipakai ulang")
	}
	assert.Equal(t, "ADM-PEMBUAT", fields["generated_by"])
	assert.Equal(t, models.RoleUserNasabah, fields["as_role"])
	repo.AssertNotCalled(t, "CreateAktivasi", mock.Anything, mock.Anything)
}

func TestGenerateReaktivasi_UpdateAktivasiError(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	lama := aknAktivasiNasabah(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountNasabahAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindNasabahByStatus", mock.Anything, "USR-001", "NSB-001", mock.Anything).
		Return(aknNasabah(models.Nonaktif), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Pending).Return(int64(1), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("FindAktivasiValidByTarget", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&lama, nil)
	repo.On("UpdateAktivasi", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", "BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestGenerateReaktivasi_CreateAktivasiError(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountNasabahAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindNasabahByStatus", mock.Anything, "USR-001", "NSB-001", mock.Anything).
		Return(aknNasabah(models.Nonaktif), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Pending).Return(int64(1), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("FindAktivasiValidByTarget", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))
	repo.On("CreateAktivasi", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"aktivasi_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", "BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestGenerateReaktivasi_GenerateIDGagal(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountNasabahAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindNasabahByStatus", mock.Anything, "USR-001", "NSB-001", mock.Anything).
		Return(aknNasabah(models.Nonaktif), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Pending).Return(int64(1), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("FindAktivasiValidByTarget", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("lock timeout"))
	dbMock.ExpectRollback()

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", "BSU-001"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
	repo.AssertNotCalled(t, "CreateAktivasi", mock.Anything, mock.Anything)
}

func TestGenerateReaktivasi_SuperadminLintasBank(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("CountNasabahAktif", mock.Anything, "USR-001").Return(int64(0), nil)
	repo.On("FindNasabahByStatus", mock.Anything, "USR-001", "NSB-001", mock.Anything).
		Return(aknNasabah(models.Nonaktif), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Pending).Return(int64(1), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("FindAktivasiValidByTarget", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	var dibuat *models.AktivasiAkun
	repo.On("CreateAktivasi", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { dibuat = args.Get(1).(*models.AktivasiAkun) }).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.GenerateReaktivasi(aknReaktivasiInput(models.RoleUserNasabah, "NSB-001", ""))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Len(t, hasil.OTP, 6, "OTP 6 karakter")
		assert.WithinDuration(t, time.Now().Add(24*time.Hour), hasil.ExpiredAt, time.Minute)
	}
	if assert.NotNil(t, dibuat) {
		assert.Equal(t, models.TujuanAktivasi, dibuat.Tujuan)
		assert.Equal(t, "ADM-PEMBUAT", dibuat.GeneratedBy)
		assert.False(t, dibuat.IsUsed)
		if assert.NotNil(t, dibuat.NasabahID) {
			assert.Equal(t, "NSB-001", *dibuat.NasabahID, "record menunjuk target spesifik")
		}
		assert.Nil(t, dibuat.AdminID)
		assert.NotEqual(t, hasil.OTP, dibuat.Token, "token disimpan sebagai hash, bukan OTP mentah")
	}
}

func TestDeaktivasi_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	err := svc.Deaktivasi(services.DeaktivasiInput{UserID: "USR-001", Role: models.RoleUserNasabah})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestDeaktivasi_RoleTidakValid(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)

	err := svc.Deaktivasi(services.DeaktivasiInput{UserID: "USR-001", Role: models.RoleUserGlobal})

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
}

func TestDeaktivasi_NasabahAktifTidakAda(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	err := svc.Deaktivasi(services.DeaktivasiInput{UserID: "USR-001", Role: models.RoleUserNasabah})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestDeaktivasi_BankLainDitolak(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(aknNasabah(models.Aktif), nil)

	err := svc.Deaktivasi(services.DeaktivasiInput{
		UserID: "USR-001", Role: models.RoleUserNasabah, ActorBankID: "BSU-999",
	})

	assert.Equal(t, http.StatusForbidden, aknKode(t, err))
	repo.AssertNotCalled(t, "UpdateStatusNasabah", mock.Anything, mock.Anything, mock.Anything)
}

func TestDeaktivasi_AdminAktifTidakAda(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	err := svc.Deaktivasi(services.DeaktivasiInput{UserID: "USR-001", Role: models.RoleUserAdmin})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestDeaktivasi_UpdateError(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(aknNasabah(models.Aktif), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Nonaktif).
		Return(int64(0), errors.New("db error"))

	err := svc.Deaktivasi(services.DeaktivasiInput{
		UserID: "USR-001", Role: models.RoleUserNasabah, ActorBankID: "BSU-001",
	})

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestDeaktivasi_NolBarisTerupdate(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(aknNasabah(models.Aktif), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Nonaktif).Return(int64(0), nil)

	err := svc.Deaktivasi(services.DeaktivasiInput{
		UserID: "USR-001", Role: models.RoleUserNasabah, ActorBankID: "BSU-001",
	})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestDeaktivasi_SuksesNasabah(t *testing.T) {
	_, repo, mailerMock, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(aknNasabah(models.Aktif), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Nonaktif).Return(int64(1), nil)

	err := svc.Deaktivasi(services.DeaktivasiInput{
		UserID: "USR-001", Role: models.RoleUserNasabah, ActorBankID: "BSU-001",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	_ = mailerMock
}

func TestDeaktivasi_SuksesAdmin(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(aknAdmin(models.Aktif), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "ADM-001", models.Nonaktif, false).Return(int64(1), nil)

	err := svc.Deaktivasi(services.DeaktivasiInput{
		UserID: "USR-001", Role: models.RoleUserAdmin, ActorBankID: "BSU-001",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestDeaktivasi_SuperadminLintasBank(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(aknNasabah(models.Aktif), nil)
	repo.On("FindBank", mock.Anything, "BSU-001").Return(aknBank(), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Nonaktif).Return(int64(1), nil)

	err := svc.Deaktivasi(services.DeaktivasiInput{
		UserID: "USR-001", Role: models.RoleUserNasabah, ActorBankID: "",
	})

	assert.NoError(t, err)
}

func TestToggleStatusNasabah_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{NasabahID: "NSB-001"})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestToggleStatusNasabah_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(nil, errors.New("db error"))

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{NasabahID: "NSB-001"})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestToggleStatusNasabah_BankLainDitolak(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(aknNasabah(models.Aktif), nil)

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{
		NasabahID: "NSB-001", ActorBankID: "BSU-999",
	})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, aknKode(t, err))
}

func TestToggleStatusNasabah_AktifJadiNonaktif(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(aknNasabah(models.Aktif), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Nonaktif).Return(int64(1), nil)

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{
		NasabahID: "NSB-001", ActorBankID: "BSU-001",
	})

	assert.NoError(t, err)
	assert.Equal(t, models.Nonaktif, hasil.StatusNasabah)
	repo.AssertNotCalled(t, "CountNasabahAktifAtauPendingSelain", mock.Anything, mock.Anything, mock.Anything)
}

func TestToggleStatusNasabah_NonaktifJadiAktif(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(aknNasabah(models.Nonaktif), nil)
	repo.On("CountNasabahAktifAtauPendingSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(0), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Aktif).Return(int64(1), nil)

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{
		NasabahID: "NSB-001", ActorBankID: "BSU-001",
	})

	assert.NoError(t, err)
	assert.Equal(t, models.Aktif, hasil.StatusNasabah)
}

func TestToggleStatusNasabah_PendingJadiAktif(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(aknNasabah(models.Pending), nil)
	repo.On("CountNasabahAktifAtauPendingSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(0), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Aktif).Return(int64(1), nil)

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{
		NasabahID: "NSB-001", ActorBankID: "BSU-001",
	})

	assert.NoError(t, err)
	assert.Equal(t, models.Aktif, hasil.StatusNasabah)
}

func TestToggleStatusNasabah_AdaAkunLainDiBankLain(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(aknNasabah(models.Nonaktif), nil)
	repo.On("CountNasabahAktifAtauPendingSelain", mock.Anything, "USR-001", "NSB-001").Return(int64(1), nil)

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{
		NasabahID: "NSB-001", ActorBankID: "BSU-001",
	})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, aknKode(t, err))
	assert.Contains(t, err.Error(), "bank sampah lain")
	repo.AssertNotCalled(t, "UpdateStatusNasabah", mock.Anything, mock.Anything, mock.Anything)
}

func TestToggleStatusNasabah_CountError(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(aknNasabah(models.Nonaktif), nil)
	repo.On("CountNasabahAktifAtauPendingSelain", mock.Anything, "USR-001", "NSB-001").
		Return(int64(0), errors.New("db error"))

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{
		NasabahID: "NSB-001", ActorBankID: "BSU-001",
	})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestToggleStatusNasabah_UpdateError(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindNasabahByID", mock.Anything, "NSB-001").Return(aknNasabah(models.Aktif), nil)
	repo.On("UpdateStatusNasabah", mock.Anything, "NSB-001", models.Nonaktif).
		Return(int64(0), errors.New("db error"))

	hasil, err := svc.ToggleStatusNasabah(services.ToggleStatusNasabahInput{
		NasabahID: "NSB-001", ActorBankID: "BSU-001",
	})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func aknSuperadmin(status models.StatusAkun) *models.Admin {
	return &models.Admin{
		AdminID: "ADM-SA-02", UserID: "USR-SA-02", Role: models.SuperAdmin, StatusAdmin: status,
	}
}

func TestNonaktifkanSuperadmin_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindAdminByIDRole", mock.Anything, "ADM-SA-02", models.SuperAdmin).
		Return(nil, gorm.ErrRecordNotFound)

	err := svc.NonaktifkanSuperadmin(services.NonaktifkanSuperadminInput{
		TargetAdminID: "ADM-SA-02", ActorUserID: "USR-SA-01",
	})

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestNonaktifkanSuperadmin_ErrorDBLain(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindAdminByIDRole", mock.Anything, "ADM-SA-02", models.SuperAdmin).
		Return(nil, errors.New("db error"))

	err := svc.NonaktifkanSuperadmin(services.NonaktifkanSuperadminInput{
		TargetAdminID: "ADM-SA-02", ActorUserID: "USR-SA-01",
	})

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestNonaktifkanSuperadmin_DiriSendiri(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindAdminByIDRole", mock.Anything, "ADM-SA-02", models.SuperAdmin).
		Return(aknSuperadmin(models.Aktif), nil)

	err := svc.NonaktifkanSuperadmin(services.NonaktifkanSuperadminInput{
		TargetAdminID: "ADM-SA-02", ActorUserID: "USR-SA-02",
	})

	assert.Equal(t, http.StatusForbidden, aknKode(t, err))
	assert.Contains(t, err.Error(), "akun sendiri")
	repo.AssertNotCalled(t, "UpdateStatusAdmin", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestNonaktifkanSuperadmin_SudahNonaktif(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindAdminByIDRole", mock.Anything, "ADM-SA-02", models.SuperAdmin).
		Return(aknSuperadmin(models.Nonaktif), nil)

	err := svc.NonaktifkanSuperadmin(services.NonaktifkanSuperadminInput{
		TargetAdminID: "ADM-SA-02", ActorUserID: "USR-SA-01",
	})

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
}

func TestNonaktifkanSuperadmin_UpdateError(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindAdminByIDRole", mock.Anything, "ADM-SA-02", models.SuperAdmin).
		Return(aknSuperadmin(models.Aktif), nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "ADM-SA-02", models.Nonaktif, false).
		Return(int64(0), errors.New("db error"))

	err := svc.NonaktifkanSuperadmin(services.NonaktifkanSuperadminInput{
		TargetAdminID: "ADM-SA-02", ActorUserID: "USR-SA-01",
	})

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestNonaktifkanSuperadmin_Sukses(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindAdminByIDRole", mock.Anything, "ADM-SA-02", models.SuperAdmin).
		Return(aknSuperadmin(models.Aktif), nil)
	repo.On("UpdateStatusAdmin", mock.Anything, "ADM-SA-02", models.Nonaktif, false).Return(int64(1), nil)

	err := svc.NonaktifkanSuperadmin(services.NonaktifkanSuperadminInput{
		TargetAdminID: "ADM-SA-02", ActorUserID: "USR-SA-01",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func aknChangeInput() services.ChangePasswordInput {
	return services.ChangePasswordInput{
		PasswordLama:           aknPassword,
		PasswordBaru:           "PasswordBaru123",
		KonfirmasiPasswordBaru: "PasswordBaru123",
	}
}

func TestChangePassword_KonfirmasiTidakCocok(t *testing.T) {
	_, _, _, svc := newAknSvc(t)

	in := aknChangeInput()
	in.KonfirmasiPasswordBaru = "beda"
	err := svc.ChangePassword("USR-001", in)

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
	assert.Contains(t, err.Error(), "Konfirmasi")
}

func TestChangePassword_PasswordBaruLemah(t *testing.T) {
	_, _, _, svc := newAknSvc(t)

	in := aknChangeInput()
	in.PasswordBaru = "passwordbaru123"
	in.KonfirmasiPasswordBaru = "passwordbaru123"
	err := svc.ChangePassword("USR-001", in)

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
	assert.Contains(t, err.Error(), "tidak memenuhi persyaratan")
}

func TestChangePassword_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	err := svc.ChangePassword("USR-001", aknChangeInput())

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestChangePassword_PasswordLamaSalah(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)

	in := aknChangeInput()
	in.PasswordLama = "salah-banget"
	err := svc.ChangePassword("USR-001", in)

	assert.Equal(t, http.StatusUnauthorized, aknKode(t, err))
	repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
}

func TestChangePassword_UpdateError(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("UpdatePassword", mock.Anything, "USR-001", mock.Anything).Return(errors.New("db error"))

	err := svc.ChangePassword("USR-001", aknChangeInput())

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestChangePassword_SemuaSesiDimatikan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("UpdatePassword", mock.Anything, "USR-001", mock.Anything).Return(nil)
	repo.On("ClearAllSessions", mock.Anything, "USR-001").Return(nil)

	err := svc.ChangePassword("USR-001", aknChangeInput())

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestChangePassword_ClearSesiGagalTetapSukses(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(aknUserBerpassword(t), nil)
	repo.On("UpdatePassword", mock.Anything, "USR-001", mock.Anything).Return(nil)
	repo.On("ClearAllSessions", mock.Anything, "USR-001").Return(errors.New("db error"))

	err := svc.ChangePassword("USR-001", aknChangeInput())

	assert.NoError(t, err)
}

func TestKirimOTPReset_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(nil, errors.New("not found"))

	hasil, err := svc.KirimOTPResetPassword("budi@mail.com")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestKirimOTPReset_ReuseRowLama(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)

	var fields map[string]interface{}
	repo.On("UpdateAktivasi", mock.Anything, "USR-0013211", mock.Anything).
		Run(func(args mock.Arguments) { fields = args.Get(2).(map[string]interface{}) }).Return(nil)

	hasil, err := svc.KirimOTPResetPassword("budi@mail.com")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "USR-0013211", hasil.AktivasiID)
		assert.WithinDuration(t, time.Now().Add(10*time.Minute), hasil.ExpiredAt, time.Minute,
			"OTP reset password berlaku 10 menit")
	}
	assert.Equal(t, "USR-001", fields["generated_by"])
	assert.NoError(t, dbMock.ExpectationsWereMet(), "jalur reuse tidak membuka transaksi")
	repo.AssertNotCalled(t, "CreateAktivasi", mock.Anything, mock.Anything)
}

func TestKirimOTPReset_UpdateError(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)
	repo.On("UpdateAktivasi", mock.Anything, mock.Anything, mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.KirimOTPResetPassword("budi@mail.com")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestKirimOTPReset_BuatRowBaru(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(nil, errors.New("not found"))

	var dibuat *models.AktivasiAkun
	repo.On("CreateAktivasi", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { dibuat = args.Get(1).(*models.AktivasiAkun) }).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.KirimOTPResetPassword("budi@mail.com")

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	if assert.NotNil(t, dibuat) {
		assert.Equal(t, models.TujuanResetPassword, dibuat.Tujuan)
		assert.Equal(t, models.RoleUserGlobal, dibuat.AsRole, "reset password tidak terikat role tertentu")
		assert.Nil(t, dibuat.NasabahID)
		assert.Nil(t, dibuat.AdminID)
		assert.False(t, dibuat.IsUsed)
	}
}

func TestKirimOTPReset_CreateError(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(nil, errors.New("not found"))
	repo.On("CreateAktivasi", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"aktivasi_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.KirimOTPResetPassword("budi@mail.com")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestVerifikasiOTPReset_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(nil, errors.New("not found"))

	hasil, err := svc.VerifikasiOTPReset("budi@mail.com", aknOTP)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestVerifikasiOTPReset_AktivasiTidakAda(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(nil, errors.New("not found"))

	hasil, err := svc.VerifikasiOTPReset("budi@mail.com", aknOTP)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestVerifikasiOTPReset_OTPSalah(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)

	hasil, err := svc.VerifikasiOTPReset("budi@mail.com", aknOTPSalah)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
}

func TestVerifikasiOTPReset_Sukses(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)

	hasil, err := svc.VerifikasiOTPReset("budi@mail.com", aknOTP)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "USR-001", hasil.UserID)
		assert.Equal(t, "USR-0013211", hasil.AktivasiID)
	}
}

func aknResetInput() services.ResetPasswordInput {
	return services.ResetPasswordInput{
		Email:                  "budi@mail.com",
		OTP:                    aknOTP,
		PasswordBaru:           "PasswordBaru123",
		KonfirmasiPasswordBaru: "PasswordBaru123",
	}
}

func TestResetPassword_KonfirmasiTidakCocok(t *testing.T) {
	_, _, _, svc := newAknSvc(t)

	in := aknResetInput()
	in.KonfirmasiPasswordBaru = "beda"
	err := svc.ResetPassword(in)

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
}

func TestResetPassword_PasswordBaruLemah(t *testing.T) {
	_, _, _, svc := newAknSvc(t)

	in := aknResetInput()
	in.PasswordBaru = "PasswordBaru"
	in.KonfirmasiPasswordBaru = "PasswordBaru"
	err := svc.ResetPassword(in)

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
	assert.Contains(t, err.Error(), "tidak memenuhi persyaratan")
}

func TestResetPassword_UserTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(nil, errors.New("not found"))

	err := svc.ResetPassword(aknResetInput())

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestResetPassword_AktivasiTidakAda(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(nil, errors.New("not found"))

	err := svc.ResetPassword(aknResetInput())

	assert.Equal(t, http.StatusNotFound, aknKode(t, err))
}

func TestResetPassword_OTPSalah(t *testing.T) {
	_, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)

	in := aknResetInput()
	in.OTP = aknOTPSalah
	err := svc.ResetPassword(in)

	assert.Equal(t, http.StatusBadRequest, aknKode(t, err))
	repo.AssertNotCalled(t, "UpdatePassword", mock.Anything, mock.Anything, mock.Anything)
}

func TestResetPassword_UpdatePasswordError(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)
	repo.On("UpdatePassword", mock.Anything, "USR-001", mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.ResetPassword(aknResetInput())

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestResetPassword_ClearSesiGagalMembatalkan(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)
	repo.On("UpdatePassword", mock.Anything, "USR-001", mock.Anything).Return(nil)
	repo.On("ClearAllSessions", mock.Anything, "USR-001").Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.ResetPassword(aknResetInput())

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
	assert.Contains(t, err.Error(), "sesi lama")
	repo.AssertNotCalled(t, "MarkAktivasiUsed", mock.Anything, mock.Anything)
}

func TestResetPassword_MarkUsedError(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)
	repo.On("UpdatePassword", mock.Anything, "USR-001", mock.Anything).Return(nil)
	repo.On("ClearAllSessions", mock.Anything, "USR-001").Return(nil)
	repo.On("MarkAktivasiUsed", mock.Anything, "USR-0013211").Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.ResetPassword(aknResetInput())

	assert.Equal(t, http.StatusInternalServerError, aknKode(t, err))
}

func TestResetPassword_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newAknSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(aknUser(), nil)
	repo.On("FindAktivasiValidTerbaru", mock.Anything, "USR-001", models.TujuanResetPassword).
		Return(aknAktivasiReset(t), nil)
	repo.On("UpdatePassword", mock.Anything, "USR-001", mock.Anything).Return(nil)
	repo.On("ClearAllSessions", mock.Anything, "USR-001").Return(nil)
	repo.On("MarkAktivasiUsed", mock.Anything, "USR-0013211").Return(nil)
	expectTx(dbMock, 0)

	err := svc.ResetPassword(aknResetInput())

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
