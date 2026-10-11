package services

import (
	"errors"
	"time"

	"enviroo-be/internal/adapters/mailer"
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"

	"gorm.io/gorm"
)

type UserRepoIface interface {
	GetProfilUser(userID string) (*repositories.ProfilUserRow, error)
	GetDetailPetugas(petugasID string) (*repositories.DetailPetugasRow, error)
	FindUser(userID string) (*models.User, error)
	FindUserRow(userID string) (*repositories.UserRow, error)
	FindUserRingkas(userID string) (*repositories.UserRingkasRow, error)
	CountUserByID(userID string) (int64, error)
	CountUserByEmail(email string) (int64, error)
	CountUser() (int64, error)
	ListUserBukanAdmin() ([]repositories.UserRow, error)
	ListUserBukanNasabah() ([]repositories.UserRow, error)
	ListUserWithAkun(limit, offset int) ([]models.User, error)
	FindUserWithAkun(userID string) (*models.User, error)
	ListNasabahWithBank(userID string) ([]models.Nasabah, error)
	ListAdminWithBank(userID string) ([]models.Admin, error)
	CountNasabahByUser(userID string) (int64, error)
	CountAdminByUser(userID string) (int64, error)
	CreateUser(tx *gorm.DB, user *models.User) error
	UpdateUser(userID string, fields map[string]interface{}) error
	DeleteUser(user *models.User) error
	UpdateFCMToken(userID, token string) error
	CreateAdmin(tx *gorm.DB, admin *models.Admin) error
	FindActiveAdmin(adminID string) (*repositories.ActiveAdminRow, error)
	FindActivePetugas(adminID string) (*repositories.ActivePetugasRow, error)
	ListSuperadmin() ([]repositories.SuperadminRow, error)
}

type TambahUserInput struct {
	UserID     string
	Nama       string
	Email      string
	NoWhatsapp string
}

type UpdateProfilUserInput struct {
	Nama        string
	NoWhatsapp  string
	FotoURL     string
	HapusFoto   bool
	AdaFileFoto bool
}

type TambahSuperadminInput struct {
	UserID     string
	Nama       string
	Email      string
	NoWhatsapp string
}

type SuperadminBaruResult struct {
	UserID  string           `json:"user_id"`
	AdminID string           `json:"admin_id"`
	Nama    string           `json:"nama"`
	Email   string           `json:"email"`
	Role    models.RoleAdmin `json:"role"`
}

type AkunNasabahItem struct {
	NasabahID string            `json:"nasabah_id"`
	JoinedAt  time.Time         `json:"joined_at"`
	Status    models.StatusAkun `json:"status"`
	Afiliasi  string            `json:"afiliasi"`
}

type AkunAdminItem struct {
	AdminID  string            `json:"admin_id"`
	Role     string            `json:"role"`
	JoinedAt time.Time         `json:"joined_at"`
	Status   models.StatusAkun `json:"status"`
	Afiliasi string            `json:"afiliasi"`
}

type LogAkunResult struct {
	UserID      string            `json:"user_id"`
	Nama        string            `json:"nama"`
	AkunNasabah []AkunNasabahItem `json:"akun_nasabah"`
	AkunAdmin   []AkunAdminItem   `json:"akun_admin"`
}

type AkunNasabahDetailItem struct {
	NasabahID     string            `json:"nasabah_id"`
	NomorRekening string            `json:"nomor_rekening"`
	Status        models.StatusAkun `json:"status"`
	Afiliasi      string            `json:"afiliasi"`
	JoinedAt      time.Time         `json:"joined_at"`
}

type DetailUserResult struct {
	UserID      string                  `json:"user_id"`
	Nama        string                  `json:"nama"`
	Email       string                  `json:"email"`
	NoWhatsapp  string                  `json:"no_whatsapp"`
	PhotoURL    string                  `json:"photo_url"`
	CreatedAt   time.Time               `json:"created_at"`
	AkunNasabah []AkunNasabahDetailItem `json:"akun_nasabah"`
	AkunAdmin   []AkunAdminItem         `json:"akun_admin"`
}

type UserRolesItem struct {
	UserID   string   `json:"user_id"`
	NamaUser string   `json:"nama_user"`
	Email    string   `json:"email"`
	Foto     string   `json:"foto"`
	Roles    []string `json:"roles"`
}

type UserPageMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"total_items"`
	TotalPages int   `json:"total_pages"`
}

type UserService interface {
	GetProfilUser(userID string) (*repositories.ProfilUserRow, error)
	GetDetailPetugas(petugasID string) (*repositories.DetailPetugasRow, error)

	TambahUser(in TambahUserInput) (*repositories.UserRow, error)
	UpdateProfil(userID string, in UpdateProfilUserInput) (*repositories.UserRow, error)
	ListNonAdmin() ([]repositories.UserRow, error)
	ListNonNasabah() ([]repositories.UserRow, error)
	GetUserRingkas(userID string) (*repositories.UserRingkasRow, error)
	UpdateFCMToken(userID, token string) error
	GetLogAkun(userID string) (*LogAkunResult, error)
	GetDetailUser(userID string) (*DetailUserResult, error)
	HapusUser(userID string) error
	ListSemuaUser(page, limit int) ([]UserRolesItem, *UserPageMeta, error)

	GetActiveAdmin(adminID string) (*repositories.ActiveAdminRow, error)
	GetActivePetugas(adminID string) (*repositories.ActivePetugasRow, error)

	TambahSuperadmin(in TambahSuperadminInput) (*SuperadminBaruResult, error)
	ListSuperadmin() ([]repositories.SuperadminRow, error)
}

type userService struct {
	db     *gorm.DB
	repo   UserRepoIface
	mailer MailerIface
}

func NewUserService(db *gorm.DB, repo UserRepoIface, mailer MailerIface) UserService {
	return &userService{db: db, repo: repo, mailer: mailer}
}

func (s *userService) GetProfilUser(userID string) (*repositories.ProfilUserRow, error) {
	profil, err := s.repo.GetProfilUser(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("User tidak ditemukan")
		}
		return nil, errInternal("Failed to get user: " + err.Error())
	}
	return profil, nil
}

func (s *userService) GetDetailPetugas(petugasID string) (*repositories.DetailPetugasRow, error) {
	if petugasID == "" {
		return nil, errBadRequest("Petugas ID wajib diisi")
	}

	detail, err := s.repo.GetDetailPetugas(petugasID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Petugas tidak ditemukan")
		}
		return nil, errInternal("Gagal mengambil detail petugas: " + err.Error())
	}
	return detail, nil
}

func (s *userService) TambahUser(in TambahUserInput) (*repositories.UserRow, error) {
	if err := s.pastikanUserBelumAda(in.UserID, in.Email); err != nil {
		return nil, err
	}

	user := &models.User{
		UserID:     in.UserID,
		Nama:       in.Nama,
		Email:      in.Email,
		NoWhatsapp: in.NoWhatsapp,
	}
	if err := s.repo.CreateUser(nil, user); err != nil {
		return nil, errInternal("Failed to create user: " + err.Error())
	}

	return s.ambilUserRow(in.UserID, "Failed to create user")
}

func (s *userService) UpdateProfil(userID string, in UpdateProfilUserInput) (*repositories.UserRow, error) {
	if in.Nama == "" && in.NoWhatsapp == "" && !in.AdaFileFoto && !in.HapusFoto {
		return nil, errBadRequest("Minimal satu field harus diisi (nama, no_whatsapp, photo_profile, atau delete_photo)")
	}

	if _, err := s.repo.FindUser(userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("User tidak ditemukan")
		}
		return nil, errInternal("Gagal mengambil data user: " + err.Error())
	}

	updates := map[string]interface{}{}
	if in.Nama != "" {
		updates["nama"] = in.Nama
	}
	if in.NoWhatsapp != "" {
		updates["no_whatsapp"] = in.NoWhatsapp
	}
	if in.AdaFileFoto {
		updates["photo_url"] = in.FotoURL
	} else if in.HapusFoto {
		updates["photo_url"] = ""
	}

	if err := s.repo.UpdateUser(userID, updates); err != nil {
		return nil, errInternal("Gagal update profil: " + err.Error())
	}

	return s.ambilUserRow(userID, "Gagal update profil")
}

func (s *userService) ListNonAdmin() ([]repositories.UserRow, error) {
	rows, err := s.repo.ListUserBukanAdmin()
	if err != nil {
		return nil, errInternal("Failed to get users: " + err.Error())
	}
	return rows, nil
}

func (s *userService) ListNonNasabah() ([]repositories.UserRow, error) {
	rows, err := s.repo.ListUserBukanNasabah()
	if err != nil {
		return nil, errInternal("Failed to get users: " + err.Error())
	}
	return rows, nil
}

func (s *userService) GetUserRingkas(userID string) (*repositories.UserRingkasRow, error) {
	row, err := s.repo.FindUserRingkas(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("User tidak ditemukan")
		}
		return nil, errInternal("Gagal mengambil data user")
	}
	return row, nil
}

func (s *userService) UpdateFCMToken(userID, token string) error {
	if err := s.repo.UpdateFCMToken(userID, token); err != nil {
		return errInternal("Gagal update FCM token: " + err.Error())
	}
	return nil
}

func (s *userService) GetLogAkun(userID string) (*LogAkunResult, error) {
	user, err := s.repo.FindUser(userID)
	if err != nil {
		return nil, errNotFound("User tidak ditemukan")
	}

	nasabahList, err := s.repo.ListNasabahWithBank(userID)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat akun nasabah: " + err.Error())
	}
	adminList, err := s.repo.ListAdminWithBank(userID)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat akun admin: " + err.Error())
	}

	akunNasabah := make([]AkunNasabahItem, 0, len(nasabahList))
	for _, n := range nasabahList {
		akunNasabah = append(akunNasabah, AkunNasabahItem{
			NasabahID: n.NasabahID,
			JoinedAt:  n.JoinedAt,
			Status:    n.StatusNasabah,
			Afiliasi:  n.Bank.NamaBank,
		})
	}

	return &LogAkunResult{
		UserID:      userID,
		Nama:        user.Nama,
		AkunNasabah: akunNasabah,
		AkunAdmin:   susunAkunAdmin(adminList),
	}, nil
}

func (s *userService) GetDetailUser(userID string) (*DetailUserResult, error) {
	user, err := s.repo.FindUserWithAkun(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("User tidak ditemukan")
		}
		return nil, errInternal("Gagal mengambil data user: " + err.Error())
	}

	akunNasabah := make([]AkunNasabahDetailItem, 0, len(user.Nasabahs))
	for _, n := range user.Nasabahs {
		akunNasabah = append(akunNasabah, AkunNasabahDetailItem{
			NasabahID:     n.NasabahID,
			NomorRekening: n.NomorRekening,
			Status:        n.StatusNasabah,
			Afiliasi:      n.Bank.NamaBank,
			JoinedAt:      n.JoinedAt,
		})
	}

	return &DetailUserResult{
		UserID:      user.UserID,
		Nama:        user.Nama,
		Email:       user.Email,
		NoWhatsapp:  user.NoWhatsapp,
		PhotoURL:    user.PhotoURL,
		CreatedAt:   user.CreatedAt,
		AkunNasabah: akunNasabah,
		AkunAdmin:   susunAkunAdmin(user.Admins),
	}, nil
}

func (s *userService) HapusUser(userID string) error {
	user, err := s.repo.FindUser(userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errNotFound("User tidak ditemukan")
		}
		return errInternal("Gagal mengambil data user: " + err.Error())
	}

	nasabahCount, err := s.repo.CountNasabahByUser(userID)
	if err != nil {
		return errInternal("Gagal memeriksa data nasabah: " + err.Error())
	}
	if nasabahCount > 0 {
		return errConflict("User tidak dapat dihapus karena sudah pernah terdaftar sebagai nasabah", nil)
	}

	adminCount, err := s.repo.CountAdminByUser(userID)
	if err != nil {
		return errInternal("Gagal memeriksa data admin: " + err.Error())
	}
	if adminCount > 0 {
		return errConflict("User tidak dapat dihapus karena sudah pernah terdaftar sebagai admin", nil)
	}

	if err := s.repo.DeleteUser(user); err != nil {
		return errInternal("Gagal menghapus user: " + err.Error())
	}
	return nil
}

func (s *userService) ListSemuaUser(page, limit int) ([]UserRolesItem, *UserPageMeta, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 25
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	total, err := s.repo.CountUser()
	if err != nil {
		return nil, nil, errInternal("Gagal menghitung data pengguna: " + err.Error())
	}

	users, err := s.repo.ListUserWithAkun(limit, offset)
	if err != nil {
		return nil, nil, errInternal("Gagal mengambil data pengguna: " + err.Error())
	}

	items := make([]UserRolesItem, 0, len(users))
	for _, user := range users {
		items = append(items, UserRolesItem{
			UserID:   user.UserID,
			NamaUser: user.Nama,
			Email:    user.Email,
			Foto:     user.PhotoURL,
			Roles:    roleUserAktif(user),
		})
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return items, &UserPageMeta{
		Page:       page,
		Limit:      limit,
		TotalItems: total,
		TotalPages: totalPages,
	}, nil
}

func (s *userService) GetActiveAdmin(adminID string) (*repositories.ActiveAdminRow, error) {
	row, err := s.repo.FindActiveAdmin(adminID)
	if err != nil {
		return nil, errInternal("Failed to get admin: " + err.Error())
	}
	if row.BankID == "" {
		return nil, errNotFound("Admin tidak ditemukan")
	}
	return row, nil
}

func (s *userService) GetActivePetugas(adminID string) (*repositories.ActivePetugasRow, error) {
	row, err := s.repo.FindActivePetugas(adminID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data petugas: " + err.Error())
	}
	if row.AdminID == "" {
		return nil, errNotFound("Petugas tidak ditemukan")
	}
	return row, nil
}

func (s *userService) TambahSuperadmin(in TambahSuperadminInput) (*SuperadminBaruResult, error) {
	if err := s.pastikanUserBelumAda(in.UserID, in.Email); err != nil {
		return nil, err
	}

	user := &models.User{
		UserID:     in.UserID,
		Nama:       utils.ToTitleCase(in.Nama),
		Email:      in.Email,
		NoWhatsapp: in.NoWhatsapp,
	}
	var admin *models.Admin

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateUser(tx, user); err != nil {
			return errInternal("Gagal membuat user: " + err.Error())
		}

		adminID, err := idgen.GenerateAdminID(tx, models.SuperAdmin, "")
		if err != nil {
			return errInternal("Gagal generate admin ID: " + err.Error())
		}

		admin = &models.Admin{
			AdminID:     adminID,
			UserID:      in.UserID,
			BankID:      nil,
			Role:        models.SuperAdmin,
			StatusAdmin: models.Pending,
		}
		if err := s.repo.CreateAdmin(tx, admin); err != nil {
			return errInternal("Gagal membuat admin: " + err.Error())
		}

		otp, expiredAt, err := createAktivasiToken(tx, in.UserID, models.RoleUserAdmin, nil, &admin.AdminID, "system")
		if err != nil {
			return errInternal("Gagal membuat aktivasi akun: " + err.Error())
		}

		body := mailer.BuildOTPEmail(user.Nama,
			"Anda telah ditunjuk sebagai <b>Superadmin</b> dalam aplikasi Enviroo. Untuk menyelesaikan proses aktivasi akun, gunakan kode OTP berikut:",
			otp, "24 jam", "Mohon untuk tidak membagikan kode OTP ini ke siapapun.", expiredAt)

		if err := s.mailer.SendEmail(mailer.Params{
			To:      user.Email,
			Subject: "Aktivasi Akun Superadmin Enviroo",
			Body:    body,
		}); err != nil {
			return errInternal("Gagal mengirim email: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusTxError(txErr, "Gagal menyimpan data")
	}

	return &SuperadminBaruResult{
		UserID:  user.UserID,
		AdminID: admin.AdminID,
		Nama:    user.Nama,
		Email:   user.Email,
		Role:    admin.Role,
	}, nil
}

func (s *userService) ListSuperadmin() ([]repositories.SuperadminRow, error) {
	list, err := s.repo.ListSuperadmin()
	if err != nil {
		return nil, errInternal("Gagal mengambil daftar superadmin: " + err.Error())
	}
	return list, nil
}

func (s *userService) pastikanUserBelumAda(userID, email string) error {
	countID, err := s.repo.CountUserByID(userID)
	if err != nil {
		return errInternal("Gagal memvalidasi data user")
	}
	if countID > 0 {
		return errConflict("User dengan ID tersebut sudah terdaftar", nil)
	}

	countEmail, err := s.repo.CountUserByEmail(email)
	if err != nil {
		return errInternal("Gagal memvalidasi data user")
	}
	if countEmail > 0 {
		return errConflict("Email sudah digunakan", nil)
	}
	return nil
}

func (s *userService) ambilUserRow(userID, pesanGagal string) (*repositories.UserRow, error) {
	row, err := s.repo.FindUserRow(userID)
	if err != nil {
		return nil, errInternal(pesanGagal + ": " + err.Error())
	}
	return row, nil
}

func susunAkunAdmin(adminList []models.Admin) []AkunAdminItem {
	items := make([]AkunAdminItem, 0, len(adminList))
	for _, a := range adminList {
		items = append(items, AkunAdminItem{
			AdminID:  a.AdminID,
			Role:     roleLabelDetail(a.Role),
			JoinedAt: a.JoinedAt,
			Status:   a.StatusAdmin,
			Afiliasi: afiliasiAdmin(a),
		})
	}
	return items
}

func afiliasiAdmin(a models.Admin) string {
	if a.Role == models.SuperAdmin {
		return "Dinas Lingkungan Hidup Kota Padang"
	}
	if a.BankID != nil {
		return a.Bank.NamaBank
	}
	return ""
}

func roleLabelDetail(role models.RoleAdmin) string {
	switch role {
	case models.SuperAdmin:
		return "Super Admin"
	case models.AdminBSI:
		return "Admin BSI"
	case models.AdminBSM:
		return "Admin BSM"
	case models.AdminBSU:
		return "Admin BSU"
	case models.PetugasBSI:
		return "Petugas BSI"
	case models.PetugasBSM:
		return "Petugas BSM"
	case models.PetugasBSU:
		return "Petugas BSU"
	default:
		return string(role)
	}
}

func roleUserAktif(user models.User) []string {
	var roles []string

	for _, nasabah := range user.Nasabahs {
		if nasabah.StatusNasabah == models.Aktif {
			roles = append(roles, "Nasabah")
			break
		}
	}

	for _, admin := range user.Admins {
		if admin.StatusAdmin != models.Aktif {
			continue
		}
		roles = append(roles, roleLabelDetail(admin.Role))
		break
	}

	if len(roles) == 0 {
		roles = append(roles, "Belum Terdaftar")
	}
	return roles
}
