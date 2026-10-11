package services

import (
	"log"

	"enviroo-be/internal/models"
	"enviroo-be/internal/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthRepoIface interface {
	FindUserByEmail(email string) (*models.User, error)
	FindUserByID(tx *gorm.DB, userID string) (*models.User, error)
	FindSessionUser(userID string) (*models.User, error)
	UpdateFCMToken(userID, token string) error
	UpdatePassword(tx *gorm.DB, userID, hashedPassword string) error
	SetSession(userID, platform, sessionID string) error
	ClearSessionMatching(userID, platform, sessionID string) error
	ClearAllSessions(tx *gorm.DB, userID string) error
	FindNasabahAktif(tx *gorm.DB, userID string) (*models.Nasabah, error)
	FindAdminAktif(tx *gorm.DB, userID string) (*models.Admin, error)
	FindNasabahPending(tx *gorm.DB, userID, nasabahID string) (*models.Nasabah, error)
	FindAdminPending(tx *gorm.DB, userID, adminID string) (*models.Admin, error)
	FindNasabahByID(tx *gorm.DB, nasabahID string) (*models.Nasabah, error)
	FindNasabahByStatus(tx *gorm.DB, userID, nasabahID string, statuses []models.StatusAkun) (*models.Nasabah, error)
	FindAdminByStatus(tx *gorm.DB, userID, adminID string, statuses []models.StatusAkun) (*models.Admin, error)
	FindAdminByIDRole(tx *gorm.DB, adminID string, role models.RoleAdmin) (*models.Admin, error)
	HasNasabahPending(userID string) bool
	HasAdminPending(userID string) bool
	CountNasabahAktif(tx *gorm.DB, userID string) (int64, error)
	CountAdminAktif(tx *gorm.DB, userID string) (int64, error)
	CountNasabahAktifSelain(tx *gorm.DB, userID, exceptNasabahID string) (int64, error)
	CountNasabahAktifAtauPendingSelain(tx *gorm.DB, userID, exceptNasabahID string) (int64, error)
	CountAdminAktifSelain(tx *gorm.DB, userID, exceptAdminID string) (int64, error)
	UpdateStatusNasabah(tx *gorm.DB, nasabahID string, status models.StatusAkun) (int64, error)
	UpdateStatusAdmin(tx *gorm.DB, adminID string, status models.StatusAkun, refreshJoinedAt bool) (int64, error)
	IsBankAktif(bankID string) (bool, error)
	FindBank(tx *gorm.DB, bankID string) (*models.BankSampah, error)
	FindAktivasiValidTerbaru(tx *gorm.DB, userID string, tujuan models.TujuanEnum) (*models.AktivasiAkun, error)
	ListAktivasiValid(tx *gorm.DB, userID string, tujuan models.TujuanEnum) ([]models.AktivasiAkun, error)
	FindAktivasiValidByTarget(tx *gorm.DB, userID string, tujuan models.TujuanEnum, nasabahID, adminID *string) (*models.AktivasiAkun, error)
	CreateAktivasi(tx *gorm.DB, aktivasi *models.AktivasiAkun) error
	UpdateAktivasi(tx *gorm.DB, aktivasiID string, fields map[string]interface{}) error
	MarkAktivasiUsed(tx *gorm.DB, aktivasiID string) error
}

type LoginInput struct {
	Email    string
	Password string
	Platform string
	Role     models.RoleUserEnum
	FCMToken string
}

type SesiResult struct {
	UserID       string           `json:"user_id"`
	Nama         string           `json:"nama"`
	Email        string           `json:"email"`
	Role         models.RoleAdmin `json:"role"`
	BankID       *string          `json:"bank_id"`
	IdentityID   string           `json:"identity_id"`
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token"`
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

type MeResult struct {
	UserID    string            `json:"user_id"`
	Nama      string            `json:"nama"`
	Email     string            `json:"email"`
	Photo     string            `json:"photo"`
	Role      models.RoleAdmin  `json:"role"`
	BankID    *string           `json:"bank_id"`
	NasabahID string            `json:"nasabah_id,omitempty"`
	AdminID   string            `json:"admin_id,omitempty"`
	Status    models.StatusAkun `json:"status"`
}

type CekUserResult struct {
	UserID        string   `json:"user_id"`
	MultipleRoles bool     `json:"multiple_roles"`
	Roles         []string `json:"roles"`
}

type AuthService interface {
	Login(in LoginInput) (*SesiResult, error)
	Refresh(refreshToken string) (*TokenPair, error)
	Logout(rawToken string)
	Me(claims *utils.JWTClaims) (*MeResult, error)
	SwitchRole(claims *utils.JWTClaims, targetRole models.RoleUserEnum) (*SesiResult, error)
	CekUserMobile(email, password string) (*CekUserResult, error)
}

type authService struct {
	repo AuthRepoIface
}

func NewAuthService(repo AuthRepoIface) AuthService {
	return &authService{repo: repo}
}

func (s *authService) Login(in LoginInput) (*SesiResult, error) {
	user, err := s.repo.FindUserByEmail(in.Email)
	if err != nil {
		return nil, errUnauthorized("Email atau password salah")
	}

	if !utils.CheckPassword(user.Password, in.Password) {
		return nil, errUnauthorized("Email atau password salah")
	}

	if in.FCMToken != "" {
		if err := s.repo.UpdateFCMToken(user.UserID, in.FCMToken); err != nil {
			log.Printf("Warning: Gagal update FCM token untuk user %s: %v", user.UserID, err)
		}
	}

	var finalRole models.RoleAdmin
	var identityID string
	var bankID *string

	switch in.Role {
	case models.RoleUserNasabah:
		if in.Platform == models.PlatformWeb {
			return nil, errForbidden("Nasabah tidak memiliki akses ke dashboard web")
		}

		nasabah, err := s.repo.FindNasabahAktif(nil, user.UserID)
		if err != nil {
			return nil, errUnauthorized("Tidak ada akun nasabah aktif untuk akun Anda")
		}
		if err := s.cekBankOperasional(nasabah.BankID); err != nil {
			return nil, err
		}

		finalRole = models.RoleNasabah
		identityID = nasabah.NasabahID
		bankID = &nasabah.BankID

	case models.RoleUserAdmin:
		admin, err := s.repo.FindAdminAktif(nil, user.UserID)
		if err != nil {
			return nil, errUnauthorized("Tidak ada akun aktif untuk akun Anda")
		}
		if admin.BankID != nil {
			if err := s.cekBankOperasional(*admin.BankID); err != nil {
				return nil, err
			}
		}

		if in.Platform == models.PlatformWeb && !isAdminTier(admin.Role) {
			return nil, errForbidden("Akses ditolak. Platform ini hanya untuk Administrator")
		}
		if in.Platform == models.PlatformMobile && !isPetugas(admin.Role) {
			return nil, errForbidden("Akses ditolak. Platform mobile hanya untuk Petugas Lapangan")
		}

		finalRole = admin.Role
		identityID = admin.AdminID
		bankID = admin.BankID

	default:
		return nil, errBadRequest("Tipe role tidak valid")
	}

	sessionID := uuid.New().String()
	if err := s.repo.SetSession(user.UserID, in.Platform, sessionID); err != nil {
		return nil, errInternal("Gagal membuat sesi login")
	}

	tokens, err := s.terbitkanToken(user.UserID, finalRole, derefBankID(bankID), in.Platform, sessionID,
		"Gagal membuat access token", "Gagal membuat refresh token")
	if err != nil {
		return nil, err
	}

	return &SesiResult{
		UserID:       user.UserID,
		Nama:         user.Nama,
		Email:        user.Email,
		Role:         finalRole,
		BankID:       bankID,
		IdentityID:   identityID,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *authService) Refresh(refreshToken string) (*TokenPair, error) {
	if refreshToken == "" {
		return nil, errUnauthorized("Refresh token tidak ditemukan")
	}

	claims, err := utils.ParseJWT(refreshToken)
	if err != nil {
		return nil, errUnauthorized("Refresh token tidak valid atau sudah kadaluarsa")
	}

	if err := s.cekSesiTerbaru(claims); err != nil {
		return nil, err
	}

	return s.terbitkanToken(claims.UserID, claims.Role, claims.BankID, claims.Platform, claims.SessionID,
		"Gagal membuat token baru", "Gagal membuat refresh token baru")
}

func (s *authService) Logout(rawToken string) {
	if rawToken == "" {
		return
	}

	claims, err := utils.ParseJWT(rawToken)
	if err != nil || claims.SessionID == "" {
		return
	}

	if err := s.repo.ClearSessionMatching(claims.UserID, claims.Platform, claims.SessionID); err != nil {
		log.Printf("Warning: Gagal mengosongkan sesi saat logout untuk user %s: %v", claims.UserID, err)
	}
}

func (s *authService) Me(claims *utils.JWTClaims) (*MeResult, error) {
	user, err := s.repo.FindUserByID(nil, claims.UserID)
	if err != nil {
		return nil, errNotFound("User tidak ditemukan")
	}

	if claims.Role == models.RoleNasabah {
		nasabah, err := s.repo.FindNasabahAktif(nil, claims.UserID)
		if err != nil {
			return nil, errNotFound("Akun nasabah aktif tidak ditemukan")
		}
		return &MeResult{
			UserID:    user.UserID,
			Nama:      user.Nama,
			Email:     user.Email,
			Photo:     user.PhotoURL,
			Role:      models.RoleNasabah,
			BankID:    &nasabah.BankID,
			NasabahID: nasabah.NasabahID,
			Status:    nasabah.StatusNasabah,
		}, nil
	}

	admin, err := s.repo.FindAdminAktif(nil, claims.UserID)
	if err != nil {
		return nil, errNotFound("Akun admin aktif tidak ditemukan")
	}
	return &MeResult{
		UserID:  user.UserID,
		Nama:    user.Nama,
		Email:   user.Email,
		Photo:   user.PhotoURL,
		Role:    admin.Role,
		BankID:  admin.BankID,
		AdminID: admin.AdminID,
		Status:  admin.StatusAdmin,
	}, nil
}

func (s *authService) SwitchRole(claims *utils.JWTClaims, targetRole models.RoleUserEnum) (*SesiResult, error) {
	if targetRole != models.RoleUserNasabah && targetRole != models.RoleUserAdmin {
		return nil, errBadRequest("target_role tidak valid, harus 'nasabah' atau 'admin'")
	}

	currentCategory := models.RoleUserAdmin
	if claims.Role == models.RoleNasabah {
		currentCategory = models.RoleUserNasabah
	}
	if targetRole == currentCategory {
		return nil, errBadRequest("Sudah login sebagai role tersebut")
	}

	user, err := s.repo.FindUserByID(nil, claims.UserID)
	if err != nil {
		return nil, errNotFound("User tidak ditemukan")
	}

	var finalRole models.RoleAdmin
	var identityID string
	var bankID *string

	switch targetRole {
	case models.RoleUserNasabah:
		nasabah, err := s.repo.FindNasabahAktif(nil, claims.UserID)
		if err != nil {
			return nil, errForbidden("Tidak ada akun nasabah aktif untuk akun Anda")
		}
		if err := s.cekBankOperasional(nasabah.BankID); err != nil {
			return nil, err
		}
		finalRole = models.RoleNasabah
		identityID = nasabah.NasabahID
		bankID = &nasabah.BankID

	case models.RoleUserAdmin:
		admin, err := s.repo.FindAdminAktif(nil, claims.UserID)
		if err != nil {
			return nil, errForbidden("Tidak ada akun petugas aktif untuk akun Anda")
		}
		if !isPetugas(admin.Role) {
			return nil, errForbidden("Akses ditolak. Switch role mobile hanya untuk Petugas Lapangan")
		}
		if admin.BankID != nil {
			if err := s.cekBankOperasional(*admin.BankID); err != nil {
				return nil, err
			}
		}
		finalRole = admin.Role
		identityID = admin.AdminID
		bankID = admin.BankID
	}

	tokens, err := s.terbitkanToken(claims.UserID, finalRole, derefBankID(bankID), claims.Platform, claims.SessionID,
		"Gagal membuat access token", "Gagal membuat refresh token")
	if err != nil {
		return nil, err
	}

	return &SesiResult{
		UserID:       claims.UserID,
		Nama:         user.Nama,
		Email:        user.Email,
		Role:         finalRole,
		BankID:       bankID,
		IdentityID:   identityID,
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
	}, nil
}

func (s *authService) CekUserMobile(email, password string) (*CekUserResult, error) {
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, errNotFound("User tidak ditemukan")
	}

	if !utils.CheckPassword(user.Password, password) {
		if s.repo.HasNasabahPending(user.UserID) || s.repo.HasAdminPending(user.UserID) {
			return nil, errUnauthorized("Mohon lakukan aktivasi akun terlebih dahulu")
		}
		return nil, errUnauthorized("Password yang Anda masukkan salah")
	}

	var validRoles []string
	if _, err := s.repo.FindNasabahAktif(nil, user.UserID); err == nil {
		validRoles = append(validRoles, string(models.RoleUserNasabah))
	}
	admin, errAdmin := s.repo.FindAdminAktif(nil, user.UserID)
	adaAdminAktif := errAdmin == nil
	if adaAdminAktif && isPetugas(admin.Role) {
		validRoles = append(validRoles, string(models.RoleUserAdmin))
	}

	if len(validRoles) == 0 {
		if adaAdminAktif {
			return nil, errForbidden(pesanPlatformTidakSesuai)
		}
		return nil, errForbidden("Akun Anda tidak aktif")
	}

	return &CekUserResult{
		UserID:        user.UserID,
		MultipleRoles: len(validRoles) > 1,
		Roles:         validRoles,
	}, nil
}

func (s *authService) cekBankOperasional(bankID string) error {
	aktif, err := s.repo.IsBankAktif(bankID)
	if err == nil && !aktif {
		return errForbidden("Operasional bank sampah sedang berhenti sementara")
	}
	return nil
}

func (s *authService) cekSesiTerbaru(claims *utils.JWTClaims) error {
	sessionUser, err := s.repo.FindSessionUser(claims.UserID)
	if err != nil {
		return errUnauthorized("Sesi tidak valid. Silakan login kembali.")
	}
	sessionID := sessionUser.SessionIDFor(claims.Platform)
	if sessionID == nil || *sessionID == "" || *sessionID != claims.SessionID {
		return errUnauthorizedCode(
			"Sesi Anda telah berakhir karena akun digunakan di perangkat lain. Silakan login kembali.",
			"SESSION_REVOKED")
	}
	return nil
}

func (s *authService) terbitkanToken(userID string, role models.RoleAdmin, bankID, platform, sessionID, msgAccess, msgRefresh string) (*TokenPair, error) {
	accessToken, err := utils.GenerateJWT(userID, role, bankID, platform, sessionID)
	if err != nil {
		return nil, errInternal(msgAccess)
	}

	refreshToken, err := utils.GenerateRefreshToken(userID, role, bankID, platform, sessionID)
	if err != nil {
		return nil, errInternal(msgRefresh)
	}

	return &TokenPair{AccessToken: accessToken, RefreshToken: refreshToken}, nil
}

func isAdminTier(role models.RoleAdmin) bool {
	return role == models.SuperAdmin || role == models.AdminBSI || role == models.AdminBSU || role == models.AdminBSM
}

func isPetugas(role models.RoleAdmin) bool {
	return role == models.PetugasBSI || role == models.PetugasBSM || role == models.PetugasBSU
}

func derefBankID(bankID *string) string {
	if bankID == nil {
		return ""
	}
	return *bankID
}
