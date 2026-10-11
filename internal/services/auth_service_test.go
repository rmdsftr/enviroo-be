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
	"enviroo-be/internal/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

const (
	authSecret   = "secret-buat-test-auth-service"
	authPassword = "rahasia123"
)

var (
	authHashOnce sync.Once
	authHashed   string
)

func authPasswordHash(t *testing.T) string {
	t.Helper()
	authHashOnce.Do(func() {
		h, err := bcrypt.GenerateFromPassword([]byte(authPassword), bcrypt.MinCost)
		if err != nil {
			t.Fatalf("GenerateFromPassword: %v", err)
		}
		authHashed = string(h)
	})
	return authHashed
}

func newAuthSvc(t *testing.T) (*mocks.MockAuthRepo, services.AuthService) {
	t.Helper()
	t.Setenv("JWT_SECRET", authSecret)
	repo := new(mocks.MockAuthRepo)
	return repo, services.NewAuthService(repo)
}

func authUser() *models.User {
	return &models.User{
		UserID:   "USR-001",
		Nama:     "Budi Santoso",
		Email:    "budi@mail.com",
		PhotoURL: "https://cdn/foto.png",
	}
}

func authUserDenganPassword(t *testing.T) *models.User {
	t.Helper()
	u := authUser()
	u.Password = authPasswordHash(t)
	return u
}

func authNasabah() *models.Nasabah {
	return &models.Nasabah{
		NasabahID:     "NSB-001",
		BankID:        "BSU-001",
		UserID:        "USR-001",
		StatusNasabah: models.Aktif,
	}
}

func authAdmin(role models.RoleAdmin) *models.Admin {
	bankID := "BSU-001"
	return &models.Admin{
		AdminID:     "ADM-001",
		BankID:      &bankID,
		UserID:      "USR-001",
		Role:        role,
		StatusAdmin: models.Aktif,
	}
}

func authSuperadmin() *models.Admin {
	return &models.Admin{
		AdminID:     "ADM-SA",
		BankID:      nil,
		UserID:      "USR-001",
		Role:        models.SuperAdmin,
		StatusAdmin: models.Aktif,
	}
}

func authClaims(role models.RoleAdmin, platform, sessionID string) *utils.JWTClaims {
	return &utils.JWTClaims{
		UserID:    "USR-001",
		Role:      role,
		BankID:    "BSU-001",
		Platform:  platform,
		SessionID: sessionID,
	}
}

func authLoginInput(role models.RoleUserEnum, platform string) services.LoginInput {
	return services.LoginInput{
		Email:    "budi@mail.com",
		Password: authPassword,
		Platform: platform,
		Role:     role,
	}
}

func authKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func authErrCode(t *testing.T, err error) string {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.ErrCode
}

func authTokenKadaluarsa(t *testing.T) string {
	t.Helper()
	claims := utils.JWTClaims{
		UserID:    "USR-001",
		Role:      models.RoleNasabah,
		BankID:    "BSU-001",
		Platform:  models.PlatformMobile,
		SessionID: "sesi-lama",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "enviroo",
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(authSecret))
	if err != nil {
		t.Fatalf("sign token kadaluarsa: %v", err)
	}
	return signed
}

func TestLogin_EmailTidakDitemukan(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(nil, errors.New("not found"))

	hasil, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformMobile))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Equal(t, "Email atau password salah", err.Error())
}

func TestLogin_PasswordSalah(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)

	in := authLoginInput(models.RoleUserNasabah, models.PlatformMobile)
	in.Password = "password-ngawur"
	hasil, err := svc.Login(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Equal(t, "Email atau password salah", err.Error())
	repo.AssertNotCalled(t, "SetSession", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_RoleTidakValid(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserGlobal, models.PlatformMobile))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, authKode(t, err))
	assert.Contains(t, err.Error(), "role tidak valid")
}

func TestLogin_FCMTokenDiupdate(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("UpdateFCMToken", "USR-001", "fcm-abc").Return(nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)
	repo.On("SetSession", "USR-001", models.PlatformMobile, mock.Anything).Return(nil)

	in := authLoginInput(models.RoleUserNasabah, models.PlatformMobile)
	in.FCMToken = "fcm-abc"
	hasil, err := svc.Login(in)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	repo.AssertExpectations(t)
}

func TestLogin_FCMTokenGagalTidakMenggagalkanLogin(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("UpdateFCMToken", "USR-001", "fcm-abc").Return(errors.New("db error"))
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)
	repo.On("SetSession", "USR-001", models.PlatformMobile, mock.Anything).Return(nil)

	in := authLoginInput(models.RoleUserNasabah, models.PlatformMobile)
	in.FCMToken = "fcm-abc"
	hasil, err := svc.Login(in)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestLogin_FCMTokenKosongTidakUpdate(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)
	repo.On("SetSession", "USR-001", models.PlatformMobile, mock.Anything).Return(nil)

	_, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformMobile))

	assert.NoError(t, err)
	repo.AssertNotCalled(t, "UpdateFCMToken", mock.Anything, mock.Anything)
}

func TestLogin_NasabahDiWebDitolak(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformWeb))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Contains(t, err.Error(), "dashboard web")
	repo.AssertNotCalled(t, "FindNasabahAktif", mock.Anything, mock.Anything)
}

func TestLogin_NasabahAktifTidakAda(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformMobile))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Contains(t, err.Error(), "akun nasabah aktif")
}

func TestLogin_NasabahBankTidakBeroperasi(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(false, nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformMobile))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Contains(t, err.Error(), "berhenti sementara")
	repo.AssertNotCalled(t, "SetSession", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogin_CekBankErrorTetapDiteruskan(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(false, errors.New("db error"))
	repo.On("SetSession", "USR-001", models.PlatformMobile, mock.Anything).Return(nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformMobile))

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestLogin_NasabahSukses(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)
	repo.On("SetSession", "USR-001", models.PlatformMobile, mock.Anything).Return(nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformMobile))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "USR-001", hasil.UserID)
		assert.Equal(t, "Budi Santoso", hasil.Nama)
		assert.Equal(t, "budi@mail.com", hasil.Email)
		assert.Equal(t, models.RoleNasabah, hasil.Role)
		assert.Equal(t, "NSB-001", hasil.IdentityID, "identity nasabah = nasabah_id")
		if assert.NotNil(t, hasil.BankID) {
			assert.Equal(t, "BSU-001", *hasil.BankID)
		}
		assert.NotEmpty(t, hasil.AccessToken)
		assert.NotEmpty(t, hasil.RefreshToken)
	}
}

func TestLogin_AdminAktifTidakAda(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.Login(authLoginInput(models.RoleUserAdmin, models.PlatformWeb))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Contains(t, err.Error(), "Tidak ada akun aktif")
}

func TestLogin_PetugasDiWebDitolak(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.PetugasBSU), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserAdmin, models.PlatformWeb))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Contains(t, err.Error(), "hanya untuk Administrator")
}

func TestLogin_AdminDiMobileDitolak(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.AdminBSU), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserAdmin, models.PlatformMobile))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Contains(t, err.Error(), "Petugas Lapangan")
}

func TestLogin_AdminBSUDiWebSukses(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.AdminBSU), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)
	repo.On("SetSession", "USR-001", models.PlatformWeb, mock.Anything).Return(nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserAdmin, models.PlatformWeb))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.AdminBSU, hasil.Role)
		assert.Equal(t, "ADM-001", hasil.IdentityID, "identity admin = admin_id")
	}
}

func TestLogin_PetugasDiMobileSukses(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.PetugasBSU), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)
	repo.On("SetSession", "USR-001", models.PlatformMobile, mock.Anything).Return(nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserAdmin, models.PlatformMobile))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.PetugasBSU, hasil.Role)
	}
}

func TestLogin_SuperadminTanpaBank(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authSuperadmin(), nil)
	repo.On("SetSession", "USR-001", models.PlatformWeb, mock.Anything).Return(nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserAdmin, models.PlatformWeb))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.SuperAdmin, hasil.Role)
		assert.Nil(t, hasil.BankID)
	}
	repo.AssertNotCalled(t, "IsBankAktif", mock.Anything)
}

func TestLogin_AdminBankTidakBeroperasi(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.AdminBSU), nil)
	repo.On("IsBankAktif", "BSU-001").Return(false, nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserAdmin, models.PlatformWeb))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
}

func TestLogin_SetSessionGagal(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)
	repo.On("SetSession", "USR-001", models.PlatformMobile, mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformMobile))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, authKode(t, err))
	assert.Contains(t, err.Error(), "sesi login")
}

func TestLogin_SessionIDTokenSamaDenganYangDisimpan(t *testing.T) {
	repo, svc := newAuthSvc(t)
	var tersimpan string
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)
	repo.On("SetSession", "USR-001", models.PlatformMobile, mock.Anything).
		Run(func(args mock.Arguments) { tersimpan = args.String(2) }).
		Return(nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserNasabah, models.PlatformMobile))
	assert.NoError(t, err)

	claims, err := utils.ParseJWT(hasil.AccessToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, tersimpan)
	assert.Equal(t, tersimpan, claims.SessionID)
	assert.Equal(t, "USR-001", claims.UserID)
	assert.Equal(t, models.RoleNasabah, claims.Role)
	assert.Equal(t, "BSU-001", claims.BankID)
	assert.Equal(t, models.PlatformMobile, claims.Platform)

	refreshClaims, err := utils.ParseJWT(hasil.RefreshToken)
	assert.NoError(t, err)
	assert.Equal(t, tersimpan, refreshClaims.SessionID, "refresh token pakai sesi yang sama")
}

func TestLogin_TokenSuperadminBankIDKosong(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authSuperadmin(), nil)
	repo.On("SetSession", "USR-001", models.PlatformWeb, mock.Anything).Return(nil)

	hasil, err := svc.Login(authLoginInput(models.RoleUserAdmin, models.PlatformWeb))
	assert.NoError(t, err)

	claims, err := utils.ParseJWT(hasil.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, "", claims.BankID)
}

func TestRefresh_TokenKosong(t *testing.T) {
	_, svc := newAuthSvc(t)

	hasil, err := svc.Refresh("")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Contains(t, err.Error(), "tidak ditemukan")
}

func TestRefresh_TokenTidakValid(t *testing.T) {
	repo, svc := newAuthSvc(t)

	hasil, err := svc.Refresh("bukan.token.valid")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	repo.AssertNotCalled(t, "FindSessionUser", mock.Anything)
}

func TestRefresh_TokenKadaluarsa(t *testing.T) {
	repo, svc := newAuthSvc(t)

	hasil, err := svc.Refresh(authTokenKadaluarsa(t))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Contains(t, err.Error(), "kadaluarsa")
	repo.AssertNotCalled(t, "FindSessionUser", mock.Anything)
}

func TestRefresh_UserSesiTidakTerbaca(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateRefreshToken("USR-001", models.RoleNasabah, "BSU-001", models.PlatformMobile, "sesi-1")
	assert.NoError(t, err)
	repo.On("FindSessionUser", "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.Refresh(token)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Contains(t, err.Error(), "Sesi tidak valid")
}

func TestRefresh_SesiKosongDiDB(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateRefreshToken("USR-001", models.RoleNasabah, "BSU-001", models.PlatformMobile, "sesi-1")
	assert.NoError(t, err)
	repo.On("FindSessionUser", "USR-001").Return(&models.User{UserID: "USR-001"}, nil)

	hasil, err := svc.Refresh(token)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Equal(t, "SESSION_REVOKED", authErrCode(t, err))
}

func TestRefresh_SesiStringKosong(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateRefreshToken("USR-001", models.RoleNasabah, "BSU-001", models.PlatformMobile, "sesi-1")
	assert.NoError(t, err)
	kosong := ""
	repo.On("FindSessionUser", "USR-001").
		Return(&models.User{UserID: "USR-001", DeviceSessionID: &kosong}, nil)

	hasil, err := svc.Refresh(token)

	assert.Nil(t, hasil)
	assert.Equal(t, "SESSION_REVOKED", authErrCode(t, err))
}

func TestRefresh_SesiSudahDipakaiPerangkatLain(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateRefreshToken("USR-001", models.RoleNasabah, "BSU-001", models.PlatformMobile, "sesi-1")
	assert.NoError(t, err)
	lain := "sesi-2"
	repo.On("FindSessionUser", "USR-001").
		Return(&models.User{UserID: "USR-001", DeviceSessionID: &lain}, nil)

	hasil, err := svc.Refresh(token)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Equal(t, "SESSION_REVOKED", authErrCode(t, err))
	assert.Contains(t, err.Error(), "perangkat lain")
}

func TestRefresh_PlatformWebPakaiSlotWeb(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateRefreshToken("USR-001", models.AdminBSU, "BSU-001", models.PlatformWeb, "sesi-web")
	assert.NoError(t, err)
	web := "sesi-web"
	device := "sesi-mobile-beda"
	repo.On("FindSessionUser", "USR-001").
		Return(&models.User{UserID: "USR-001", WebSessionID: &web, DeviceSessionID: &device}, nil)

	hasil, err := svc.Refresh(token)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestRefresh_SuksesReuseSesi(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateRefreshToken("USR-001", models.RoleNasabah, "BSU-001", models.PlatformMobile, "sesi-1")
	assert.NoError(t, err)
	sesi := "sesi-1"
	repo.On("FindSessionUser", "USR-001").
		Return(&models.User{UserID: "USR-001", DeviceSessionID: &sesi}, nil)

	hasil, err := svc.Refresh(token)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.NotEmpty(t, hasil.AccessToken)
		assert.NotEmpty(t, hasil.RefreshToken)

		claims, err := utils.ParseJWT(hasil.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, "sesi-1", claims.SessionID)
		assert.Equal(t, models.PlatformMobile, claims.Platform)
		assert.Equal(t, models.RoleNasabah, claims.Role)
		assert.Equal(t, "BSU-001", claims.BankID)
	}
	repo.AssertNotCalled(t, "SetSession", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogout_TokenKosong(t *testing.T) {
	repo, svc := newAuthSvc(t)

	svc.Logout("")

	repo.AssertNotCalled(t, "ClearSessionMatching", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogout_TokenTidakValid(t *testing.T) {
	repo, svc := newAuthSvc(t)

	svc.Logout("token-ngawur")

	repo.AssertNotCalled(t, "ClearSessionMatching", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogout_TokenTanpaSessionID(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateJWT("USR-001", models.RoleNasabah, "BSU-001", models.PlatformMobile, "")
	assert.NoError(t, err)

	svc.Logout(token)

	repo.AssertNotCalled(t, "ClearSessionMatching", mock.Anything, mock.Anything, mock.Anything)
}

func TestLogout_Sukses(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateJWT("USR-001", models.RoleNasabah, "BSU-001", models.PlatformMobile, "sesi-1")
	assert.NoError(t, err)
	repo.On("ClearSessionMatching", "USR-001", models.PlatformMobile, "sesi-1").Return(nil)

	svc.Logout(token)

	repo.AssertExpectations(t)
}

func TestLogout_RepoErrorTidakPanik(t *testing.T) {
	repo, svc := newAuthSvc(t)
	token, err := utils.GenerateJWT("USR-001", models.RoleNasabah, "BSU-001", models.PlatformMobile, "sesi-1")
	assert.NoError(t, err)
	repo.On("ClearSessionMatching", "USR-001", models.PlatformMobile, "sesi-1").Return(errors.New("db error"))

	assert.NotPanics(t, func() { svc.Logout(token) })
	repo.AssertExpectations(t)
}

func TestMe_UserTidakDitemukan(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.Me(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, authKode(t, err))
}

func TestMe_NasabahAktifTidakAda(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.Me(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, authKode(t, err))
	assert.Contains(t, err.Error(), "nasabah")
}

func TestMe_Nasabah(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)

	hasil, err := svc.Me(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "USR-001", hasil.UserID)
		assert.Equal(t, "Budi Santoso", hasil.Nama)
		assert.Equal(t, "https://cdn/foto.png", hasil.Photo)
		assert.Equal(t, models.RoleNasabah, hasil.Role)
		assert.Equal(t, "NSB-001", hasil.NasabahID)
		assert.Empty(t, hasil.AdminID, "nasabah_id dan admin_id saling eksklusif")
		assert.Equal(t, models.Aktif, hasil.Status)
		if assert.NotNil(t, hasil.BankID) {
			assert.Equal(t, "BSU-001", *hasil.BankID)
		}
	}
	repo.AssertNotCalled(t, "FindAdminAktif", mock.Anything, mock.Anything)
}

func TestMe_AdminAktifTidakAda(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.Me(authClaims(models.AdminBSU, models.PlatformWeb, "sesi-1"))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, authKode(t, err))
	assert.Contains(t, err.Error(), "admin")
}

func TestMe_Admin(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.AdminBSU), nil)

	hasil, err := svc.Me(authClaims(models.AdminBSU, models.PlatformWeb, "sesi-1"))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.AdminBSU, hasil.Role)
		assert.Equal(t, "ADM-001", hasil.AdminID)
		assert.Empty(t, hasil.NasabahID)
		assert.Equal(t, models.Aktif, hasil.Status)
	}
}

func TestMe_PetugasLewatJalurAdmin(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.PetugasBSU), nil)

	hasil, err := svc.Me(authClaims(models.PetugasBSU, models.PlatformMobile, "sesi-1"))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.PetugasBSU, hasil.Role)
		assert.Equal(t, "ADM-001", hasil.AdminID)
	}
}

func TestSwitchRole_TargetTidakValid(t *testing.T) {
	_, svc := newAuthSvc(t)

	hasil, err := svc.SwitchRole(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"), models.RoleUserGlobal)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, authKode(t, err))
	assert.Contains(t, err.Error(), "target_role")
}

func TestSwitchRole_SudahJadiNasabah(t *testing.T) {
	_, svc := newAuthSvc(t)

	hasil, err := svc.SwitchRole(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"), models.RoleUserNasabah)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, authKode(t, err))
	assert.Contains(t, err.Error(), "Sudah login")
}

func TestSwitchRole_SudahJadiPetugas(t *testing.T) {
	_, svc := newAuthSvc(t)

	hasil, err := svc.SwitchRole(authClaims(models.PetugasBSU, models.PlatformMobile, "sesi-1"), models.RoleUserAdmin)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, authKode(t, err))
	assert.Contains(t, err.Error(), "Sudah login")
}

func TestSwitchRole_UserTidakDitemukan(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.SwitchRole(authClaims(models.PetugasBSU, models.PlatformMobile, "sesi-1"), models.RoleUserNasabah)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, authKode(t, err))
}

func TestSwitchRole_KeNasabahTanpaAkunNasabah(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.SwitchRole(authClaims(models.PetugasBSU, models.PlatformMobile, "sesi-1"), models.RoleUserNasabah)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Contains(t, err.Error(), "akun nasabah aktif")
}

func TestSwitchRole_KeNasabahBankTidakBeroperasi(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(false, nil)

	hasil, err := svc.SwitchRole(authClaims(models.PetugasBSU, models.PlatformMobile, "sesi-1"), models.RoleUserNasabah)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Contains(t, err.Error(), "berhenti sementara")
}

func TestSwitchRole_KeNasabahSukses(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)

	hasil, err := svc.SwitchRole(authClaims(models.PetugasBSU, models.PlatformMobile, "sesi-1"), models.RoleUserNasabah)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.RoleNasabah, hasil.Role)
		assert.Equal(t, "NSB-001", hasil.IdentityID)
		assert.Equal(t, "Budi Santoso", hasil.Nama)

		claims, err := utils.ParseJWT(hasil.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, "sesi-1", claims.SessionID, "session id di-reuse dari token lama")
		assert.Equal(t, models.PlatformMobile, claims.Platform)
		assert.Equal(t, models.RoleNasabah, claims.Role)
	}
	repo.AssertNotCalled(t, "SetSession", mock.Anything, mock.Anything, mock.Anything)
}

func TestSwitchRole_KeAdminTanpaAkunAdmin(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.SwitchRole(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"), models.RoleUserAdmin)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Contains(t, err.Error(), "akun petugas aktif")
}

func TestSwitchRole_KeAdminTierDitolak(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.AdminBSU), nil)

	hasil, err := svc.SwitchRole(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"), models.RoleUserAdmin)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Contains(t, err.Error(), "Petugas Lapangan")
}

func TestSwitchRole_KeAdminBankTidakBeroperasi(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.PetugasBSU), nil)
	repo.On("IsBankAktif", "BSU-001").Return(false, nil)

	hasil, err := svc.SwitchRole(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"), models.RoleUserAdmin)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
}

func TestSwitchRole_KePetugasSukses(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByID", mock.Anything, "USR-001").Return(authUser(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.PetugasBSU), nil)
	repo.On("IsBankAktif", "BSU-001").Return(true, nil)

	hasil, err := svc.SwitchRole(authClaims(models.RoleNasabah, models.PlatformMobile, "sesi-1"), models.RoleUserAdmin)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.PetugasBSU, hasil.Role)
		assert.Equal(t, "ADM-001", hasil.IdentityID)

		claims, err := utils.ParseJWT(hasil.AccessToken)
		assert.NoError(t, err)
		assert.Equal(t, "sesi-1", claims.SessionID)
		assert.Equal(t, models.PetugasBSU, claims.Role)
	}
	repo.AssertNotCalled(t, "SetSession", mock.Anything, mock.Anything, mock.Anything)
}

func TestCekUserMobile_UserTidakDitemukan(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(nil, errors.New("not found"))

	hasil, err := svc.CekUserMobile("budi@mail.com", authPassword)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, authKode(t, err))
}

func TestCekUserMobile_PasswordSalah(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("HasNasabahPending", "USR-001").Return(false)
	repo.On("HasAdminPending", "USR-001").Return(false)

	hasil, err := svc.CekUserMobile("budi@mail.com", "salah")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Contains(t, err.Error(), "Password")
}

func TestCekUserMobile_AkunNasabahMasihPending(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("HasNasabahPending", "USR-001").Return(true)

	hasil, err := svc.CekUserMobile("budi@mail.com", "salah")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusUnauthorized, authKode(t, err))
	assert.Contains(t, err.Error(), "aktivasi")
}

func TestCekUserMobile_AkunAdminMasihPending(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("HasNasabahPending", "USR-001").Return(false)
	repo.On("HasAdminPending", "USR-001").Return(true)

	hasil, err := svc.CekUserMobile("budi@mail.com", "salah")

	assert.Nil(t, hasil)
	assert.Contains(t, err.Error(), "aktivasi")
}

func TestCekUserMobile_HanyaNasabah(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.CekUserMobile("budi@mail.com", authPassword)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "USR-001", hasil.UserID)
		assert.False(t, hasil.MultipleRoles)
		assert.Equal(t, []string{"nasabah"}, hasil.Roles)
	}
}

func TestCekUserMobile_HanyaPetugas(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.PetugasBSU), nil)

	hasil, err := svc.CekUserMobile("budi@mail.com", authPassword)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.False(t, hasil.MultipleRoles)
		assert.Equal(t, []string{"admin"}, hasil.Roles)
	}
}

func TestCekUserMobile_NasabahDanPetugas(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.PetugasBSU), nil)

	hasil, err := svc.CekUserMobile("budi@mail.com", authPassword)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.True(t, hasil.MultipleRoles)
		assert.Equal(t, []string{"nasabah", "admin"}, hasil.Roles)
	}
}

func TestCekUserMobile_AdminTierBukanRoleMobile(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.AdminBSU), nil)

	hasil, err := svc.CekUserMobile("budi@mail.com", authPassword)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Equal(t, "Peran akun anda tidak sesuai untuk platform ini", err.Error())
}

func TestCekUserMobile_AdminTierTapiJugaNasabah(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(authNasabah(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(authAdmin(models.AdminBSU), nil)

	hasil, err := svc.CekUserMobile("budi@mail.com", authPassword)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.False(t, hasil.MultipleRoles)
		assert.Equal(t, []string{"nasabah"}, hasil.Roles)
	}
}

func TestCekUserMobile_TidakPunyaRoleAktif(t *testing.T) {
	repo, svc := newAuthSvc(t)
	repo.On("FindUserByEmail", "budi@mail.com").Return(authUserDenganPassword(t), nil)
	repo.On("FindNasabahAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))
	repo.On("FindAdminAktif", mock.Anything, "USR-001").Return(nil, errors.New("not found"))

	hasil, err := svc.CekUserMobile("budi@mail.com", authPassword)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, authKode(t, err))
	assert.Equal(t, "Akun Anda tidak aktif", err.Error())
}
