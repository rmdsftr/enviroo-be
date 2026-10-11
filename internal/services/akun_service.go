package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"enviroo-be/internal/adapters/mailer"
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/utils"

	"gorm.io/gorm"
)

type AktivasiAkunInput struct {
	UserID   string
	OTP      string
	Password string
	Platform string
}

type GenerateReaktivasiInput struct {
	UserID string
	Role   models.RoleUserEnum

	RoleID string

	AdminID string

	ActorBankID string
}

type ReaktivasiTokenResult struct {
	AktivasiID string    `json:"aktivasi_id"`
	OTP        string    `json:"otp"`
	ExpiredAt  time.Time `json:"expired_at"`
}

type DeaktivasiInput struct {
	UserID      string
	Role        models.RoleUserEnum
	ActorBankID string
}

type ToggleStatusNasabahInput struct {
	NasabahID   string
	ActorBankID string
}

type NonaktifkanSuperadminInput struct {
	TargetAdminID string
	ActorUserID   string
}

type ChangePasswordInput struct {
	PasswordLama           string
	PasswordBaru           string
	KonfirmasiPasswordBaru string
}

type ResetOTPResult struct {
	AktivasiID string    `json:"aktivasi_id"`
	ExpiredAt  time.Time `json:"expired_at"`
}

type VerifikasiOTPResult struct {
	UserID     string `json:"user_id"`
	AktivasiID string `json:"aktivasi_id"`
}

type ResetPasswordInput struct {
	Email                  string
	OTP                    string
	PasswordBaru           string
	KonfirmasiPasswordBaru string
}

type AkunService interface {
	AktivasiAkun(in AktivasiAkunInput) (pesanSukses string, err error)
	GenerateReaktivasi(in GenerateReaktivasiInput) (*ReaktivasiTokenResult, error)
	Deaktivasi(in DeaktivasiInput) error
	ToggleStatusNasabah(in ToggleStatusNasabahInput) (*models.Nasabah, error)
	NonaktifkanSuperadmin(in NonaktifkanSuperadminInput) error
	ChangePassword(userID string, in ChangePasswordInput) error
	KirimOTPResetPassword(email string) (*ResetOTPResult, error)
	VerifikasiOTPReset(email, otp string) (*VerifikasiOTPResult, error)
	ResetPassword(in ResetPasswordInput) error
}

type akunService struct {
	db     *gorm.DB
	repo   AuthRepoIface
	mailer MailerIface
}

func NewAkunService(db *gorm.DB, repo AuthRepoIface, mailer MailerIface) AkunService {
	return &akunService{db: db, repo: repo, mailer: mailer}
}

func cocokkanOTP(kandidat []models.AktivasiAkun, otp string) *models.AktivasiAkun {
	for i := range kandidat {
		if utils.VerifyOTP(otp, kandidat[i].Token) {
			return &kandidat[i]
		}
	}
	return nil
}

func (s *akunService) AktivasiAkun(in AktivasiAkunInput) (string, error) {
	kandidat, err := s.repo.ListAktivasiValid(nil, in.UserID, models.TujuanAktivasi)
	if err != nil || len(kandidat) == 0 {
		return "", errNotFound("Kode aktivasi tidak valid, kadaluarsa, atau sudah digunakan")
	}

	aktivasi := cocokkanOTP(kandidat, in.OTP)
	if aktivasi == nil {
		return "", errBadRequest("OTP tidak valid atau salah")
	}

	user, err := s.repo.FindUserByID(nil, in.UserID)
	if err != nil {
		return "", errNotFound("User tidak ditemukan")
	}

	var nasabah *models.Nasabah
	var admin *models.Admin

	switch aktivasi.AsRole {
	case models.RoleUserNasabah:
		if aktivasi.NasabahID == nil {
			return "", errInternal("Data aktivasi tidak valid (nasabah_id kosong)")
		}
		nasabah, err = s.repo.FindNasabahPending(nil, in.UserID, *aktivasi.NasabahID)
		if err != nil {
			return "", errNotFound("Tidak ada akun nasabah pending untuk diaktivasi")
		}
		if in.Platform != models.PlatformMobile {
			return "", errForbidden(pesanPlatformTidakSesuai)
		}
	case models.RoleUserAdmin:
		if aktivasi.AdminID == nil {
			return "", errInternal("Data aktivasi tidak valid (admin_id kosong)")
		}
		admin, err = s.repo.FindAdminPending(nil, in.UserID, *aktivasi.AdminID)
		if err != nil {
			return "", errNotFound("Tidak ada akun admin/petugas pending untuk diaktivasi")
		}
		if !platformSesuaiAdmin(admin.Role, in.Platform) {
			return "", errForbidden(pesanPlatformTidakSesuai)
		}
	default:
		return "", errBadRequest("Tipe role aktivasi tidak diketahui")
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {

		if user.Password == "" {
			if in.Password == "" {
				return errBadRequestCode("Password wajib dibuat untuk aktivasi pertama", "PASSWORD_REQUIRED")
			}
			if err := utils.ValidasiKekuatanPassword(in.Password); err != nil {
				return errBadRequest(err.Error())
			}
			hashed, err := utils.HashPassword(in.Password)
			if err != nil {
				return errInternal("Gagal memproses password")
			}
			if err := s.repo.UpdatePassword(tx, user.UserID, hashed); err != nil {
				return errInternal("Gagal mengupdate password user")
			}
		}

		if nasabah != nil {
			if err := s.pastikanTidakAdaNasabahAktifLain(tx, in.UserID, nasabah.NasabahID); err != nil {
				return err
			}
			if _, err := s.repo.UpdateStatusNasabah(tx, nasabah.NasabahID, models.Aktif); err != nil {
				return errInternal("Gagal mengupdate status nasabah")
			}
		} else {
			if err := s.pastikanTidakAdaAdminAktifLain(tx, in.UserID, admin.AdminID); err != nil {
				return err
			}
			if _, err := s.repo.UpdateStatusAdmin(tx, admin.AdminID, models.Aktif, false); err != nil {
				return errInternal("Gagal mengupdate status admin")
			}
		}

		if err := s.repo.MarkAktivasiUsed(tx, aktivasi.AktivasiID); err != nil {
			return errInternal("Gagal mengupdate status aktivasi")
		}
		return nil
	})
	if txErr != nil {
		return "", bungkusTxError(txErr, "Gagal menyimpan perubahan ke database")
	}

	if nasabah != nil {
		return "Aktivasi nasabah berhasil. Anda sekarang dapat login menggunakan password.", nil
	}
	return "Aktivasi staff berhasil. Anda sekarang dapat login menggunakan password.", nil
}

func (s *akunService) GenerateReaktivasi(in GenerateReaktivasiInput) (*ReaktivasiTokenResult, error) {
	otp, err := utils.GenerateOTP()
	if err != nil {
		return nil, errInternal("Gagal generate OTP")
	}
	hashedOTP, err := utils.HashOTP(otp)
	if err != nil {
		return nil, errInternal("Gagal hash OTP")
	}

	expiredAt := time.Now().Add(24 * time.Hour)

	var user *models.User
	var namaBank, successRoleName, aktivasiID string

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		user, err = s.repo.FindUserByID(tx, in.UserID)
		if err != nil {
			return errNotFound("User tidak ditemukan")
		}

		var targetNasabahID, targetAdminID *string
		var bankID string

		switch in.Role {
		case models.RoleUserNasabah:
			count, err := s.repo.CountNasabahAktif(tx, in.UserID)
			if err != nil {
				return errInternal("Gagal memvalidasi data nasabah")
			}
			if count > 0 {
				return errConflict("User sudah memiliki akun nasabah aktif. Nonaktifkan akun tersebut terlebih dahulu sebelum mengaktifkan kembali akun lain.", nil)
			}

			nasabah, err := s.repo.FindNasabahByStatus(tx, in.UserID, in.RoleID,
				[]models.StatusAkun{models.Nonaktif, models.Pending})
			if err != nil {
				return errNotFound("Tidak ada akun nasabah nonaktif/pending dengan nasabah_id tersebut yang dapat diaktifkan kembali")
			}
			if err := cekAksesBank(in.ActorBankID, &nasabah.BankID); err != nil {
				return err
			}

			bankID = nasabah.BankID
			successRoleName = "nasabah"
			targetNasabahID = &nasabah.NasabahID

			if _, err := s.repo.UpdateStatusNasabah(tx, nasabah.NasabahID, models.Pending); err != nil {
				return errInternal("Gagal mengupdate status nasabah")
			}

		case models.RoleUserAdmin:
			count, err := s.repo.CountAdminAktif(tx, in.UserID)
			if err != nil {
				return errInternal("Gagal memvalidasi data admin")
			}
			if count > 0 {
				return errConflict("User sudah memiliki akun admin aktif. Nonaktifkan akun tersebut terlebih dahulu sebelum mengaktifkan kembali akun lain.", nil)
			}

			admin, err := s.repo.FindAdminByStatus(tx, in.UserID, in.RoleID,
				[]models.StatusAkun{models.Nonaktif, models.Pending})
			if err != nil {
				return errNotFound("Tidak ada akun staff nonaktif/pending dengan admin_id tersebut yang dapat diaktifkan kembali")
			}
			if err := cekAksesBank(in.ActorBankID, admin.BankID); err != nil {
				return err
			}

			if admin.BankID != nil {
				bankID = *admin.BankID
			}
			successRoleName = "staff"
			targetAdminID = &admin.AdminID

			if _, err := s.repo.UpdateStatusAdmin(tx, admin.AdminID, models.Pending, true); err != nil {
				return errInternal("Gagal mengupdate status admin")
			}

		default:
			return errBadRequest("Role tidak valid")
		}

		namaBank = "Enviroo System"
		if bankID != "" {
			if bank, err := s.repo.FindBank(tx, bankID); err == nil {
				namaBank = bank.NamaBank
			}
		}

		lama, err := s.repo.FindAktivasiValidByTarget(tx, in.UserID, models.TujuanAktivasi, targetNasabahID, targetAdminID)
		if err == nil && lama.AktivasiID != "" {
			aktivasiID = lama.AktivasiID
			if err := s.repo.UpdateAktivasi(tx, lama.AktivasiID, map[string]interface{}{
				"token":        hashedOTP,
				"expired_at":   expiredAt,
				"generated_by": in.AdminID,
				"as_role":      in.Role,
				"nasabah_id":   targetNasabahID,
				"admin_id":     targetAdminID,
			}); err != nil {
				return errInternal("Gagal mengupdate record aktivasi")
			}
			return nil
		}

		newID, err := idgen.GenerateAktivasiID(tx, in.UserID, in.Role, models.TujuanAktivasi)
		if err != nil {
			return errInternal("Gagal generate aktivasi ID")
		}
		aktivasiID = newID

		if err := s.repo.CreateAktivasi(tx, &models.AktivasiAkun{
			AktivasiID:  newID,
			UserID:      in.UserID,
			AsRole:      in.Role,
			NasabahID:   targetNasabahID,
			AdminID:     targetAdminID,
			IsUsed:      false,
			Token:       hashedOTP,
			ExpiredAt:   expiredAt,
			GeneratedBy: in.AdminID,
			Tujuan:      models.TujuanAktivasi,
		}); err != nil {
			return errInternal("Gagal membuat record aktivasi baru")
		}
		return nil
	})
	if txErr != nil {
		return nil, bungkusTxError(txErr, "Gagal menyimpan perubahan ke database")
	}

	paragraf := fmt.Sprintf("Untuk mengaktifkan kembali akun %s Anda di <b>%s</b>, gunakan kode OTP berikut:", successRoleName, namaBank)
	body := mailer.BuildOTPEmail(user.Nama, paragraf, otp, "24 jam",
		"Mohon untuk tidak membagikan kode OTP ini ke siapapun.", expiredAt)
	kirimEmailAsync(s.mailer, user.Email, "Aktivasi Kembali Akun Enviroo", body)

	return &ReaktivasiTokenResult{
		AktivasiID: aktivasiID,
		OTP:        otp,
		ExpiredAt:  expiredAt,
	}, nil
}

func (s *akunService) Deaktivasi(in DeaktivasiInput) error {
	user, err := s.repo.FindUserByID(nil, in.UserID)
	if err != nil {
		return errNotFound("User tidak ditemukan")
	}

	var namaBank, roleLabel string
	var rows int64

	switch in.Role {
	case models.RoleUserNasabah:
		nasabah, err := s.repo.FindNasabahAktif(nil, in.UserID)
		if err != nil {
			return errNotFound("Data akun nasabah aktif tidak ditemukan")
		}
		if err := cekAksesBank(in.ActorBankID, &nasabah.BankID); err != nil {
			return err
		}
		if bank, err := s.repo.FindBank(nil, nasabah.BankID); err == nil {
			namaBank = bank.NamaBank
		}
		roleLabel = "Nasabah"

		rows, err = s.repo.UpdateStatusNasabah(nil, nasabah.NasabahID, models.Nonaktif)
		if err != nil {
			return errInternal("Gagal menonaktifkan akun: " + err.Error())
		}

	case models.RoleUserAdmin:
		admin, err := s.repo.FindAdminAktif(nil, in.UserID)
		if err != nil {
			return errNotFound("Data akun admin aktif tidak ditemukan")
		}
		if err := cekAksesBank(in.ActorBankID, admin.BankID); err != nil {
			return err
		}
		if admin.BankID != nil {
			if bank, err := s.repo.FindBank(nil, *admin.BankID); err == nil {
				namaBank = bank.NamaBank
			}
		}
		roleLabel = "Staff/Admin"

		rows, err = s.repo.UpdateStatusAdmin(nil, admin.AdminID, models.Nonaktif, false)
		if err != nil {
			return errInternal("Gagal menonaktifkan akun: " + err.Error())
		}

	default:
		return errBadRequest("Role tidak valid")
	}

	if rows == 0 {
		return errNotFound("Data akun tidak ditemukan")
	}

	body := mailer.BuildNotifEmail(user.Nama, fmt.Sprintf(
		"Akun <b>%s</b> Anda di <b>%s</b> telah dinonaktifkan. Jika Anda merasa ini adalah kesalahan, silakan hubungi administrator.",
		roleLabel, namaBank))
	kirimEmailAsync(s.mailer, user.Email, "Notifikasi Penonaktifan Akun Enviroo", body)

	return nil
}

func (s *akunService) ToggleStatusNasabah(in ToggleStatusNasabahInput) (*models.Nasabah, error) {
	nasabah, err := s.repo.FindNasabahByID(nil, in.NasabahID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Nasabah tidak ditemukan")
		}
		return nil, errInternal("Failed to get nasabah: " + err.Error())
	}

	if err := cekAksesBank(in.ActorBankID, &nasabah.BankID); err != nil {
		return nil, err
	}

	targetStatus := models.Nonaktif
	if nasabah.StatusNasabah == models.Nonaktif || nasabah.StatusNasabah == models.Pending {
		targetStatus = models.Aktif
	}

	if targetStatus == models.Aktif {
		count, err := s.repo.CountNasabahAktifAtauPendingSelain(nil, nasabah.UserID, nasabah.NasabahID)
		if err != nil {
			return nil, errInternal("Gagal memvalidasi data nasabah")
		}
		if count > 0 {
			return nil, errConflict("User sudah memiliki akun nasabah aktif atau sedang pending di bank sampah lain. Selesaikan atau batalkan proses tersebut terlebih dahulu.", nil)
		}
	}

	if _, err := s.repo.UpdateStatusNasabah(nil, nasabah.NasabahID, targetStatus); err != nil {
		return nil, errInternal("Failed to update nasabah status: " + err.Error())
	}

	nasabah.StatusNasabah = targetStatus
	return nasabah, nil
}

func (s *akunService) NonaktifkanSuperadmin(in NonaktifkanSuperadminInput) error {
	target, err := s.repo.FindAdminByIDRole(nil, in.TargetAdminID, models.SuperAdmin)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errNotFound("Superadmin tidak ditemukan")
		}
		return errInternal("Gagal mengambil data: " + err.Error())
	}

	if target.UserID == in.ActorUserID {
		return errForbidden("Tidak dapat menonaktifkan akun sendiri")
	}

	if target.StatusAdmin == models.Nonaktif {
		return errBadRequest("Akun superadmin ini sudah nonaktif")
	}

	if _, err := s.repo.UpdateStatusAdmin(nil, target.AdminID, models.Nonaktif, false); err != nil {
		return errInternal("Gagal menonaktifkan superadmin: " + err.Error())
	}
	return nil
}

func (s *akunService) ChangePassword(userID string, in ChangePasswordInput) error {
	if in.PasswordBaru != in.KonfirmasiPasswordBaru {
		return errBadRequest("Konfirmasi password baru tidak cocok")
	}
	if err := utils.ValidasiKekuatanPassword(in.PasswordBaru); err != nil {
		return errBadRequest(err.Error())
	}

	user, err := s.repo.FindUserByID(nil, userID)
	if err != nil {
		return errNotFound("User tidak ditemukan")
	}

	if !utils.CheckPassword(user.Password, in.PasswordLama) {
		return errUnauthorized("Password lama salah")
	}

	hashed, err := utils.HashPassword(in.PasswordBaru)
	if err != nil {
		return errInternal("Gagal mengamankan password baru")
	}

	if err := s.repo.UpdatePassword(nil, user.UserID, hashed); err != nil {
		return errInternal("Gagal memperbarui password")
	}

	if err := s.repo.ClearAllSessions(nil, user.UserID); err != nil {
		log.Printf("Warning: Gagal mengosongkan sesi setelah ganti password untuk user %s: %v", user.UserID, err)
	}

	body := mailer.BuildNotifEmail(user.Nama,
		"Password akun Enviroo Anda baru saja berhasil diubah. Jika Anda tidak melakukan perubahan ini, segera hubungi administrator.")
	kirimEmailAsync(s.mailer, user.Email, "Notifikasi Perubahan Password Enviroo", body)

	return nil
}

func (s *akunService) KirimOTPResetPassword(email string) (*ResetOTPResult, error) {
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, errNotFound("Pengguna tidak ditemukan")
	}

	otp, err := utils.GenerateOTP()
	if err != nil {
		return nil, errInternal("Gagal generate OTP")
	}
	hashedOTP, err := utils.HashOTP(otp)
	if err != nil {
		return nil, errInternal("Gagal hash OTP")
	}

	expiredAt := time.Now().Add(10 * time.Minute)
	var aktivasiID string

	lama, errLama := s.repo.FindAktivasiValidTerbaru(nil, user.UserID, models.TujuanResetPassword)
	if errLama == nil {
		aktivasiID = lama.AktivasiID
		if err := s.repo.UpdateAktivasi(nil, lama.AktivasiID, map[string]interface{}{
			"token":        hashedOTP,
			"expired_at":   expiredAt,
			"generated_by": user.UserID,
		}); err != nil {
			return nil, errInternal("Gagal mengupdate record aktivasi")
		}
	} else {

		txErr := s.db.Transaction(func(tx *gorm.DB) error {
			newID, err := idgen.GenerateAktivasiID(tx, user.UserID, models.RoleUserGlobal, models.TujuanResetPassword)
			if err != nil {
				return err
			}
			aktivasiID = newID

			return s.repo.CreateAktivasi(tx, &models.AktivasiAkun{
				AktivasiID:  newID,
				UserID:      user.UserID,
				AsRole:      models.RoleUserGlobal,
				IsUsed:      false,
				Token:       hashedOTP,
				ExpiredAt:   expiredAt,
				GeneratedBy: user.UserID,
				Tujuan:      models.TujuanResetPassword,
			})
		})
		if txErr != nil {
			return nil, errInternal("Gagal membuat record aktivasi baru")
		}
	}

	paragraf := fmt.Sprintf("Kami menerima permintaan reset password untuk akun email <b>%s</b>. Gunakan kode OTP berikut untuk melanjutkan:", user.Email)
	body := mailer.BuildOTPEmail(user.Nama, paragraf, otp, "10 menit",
		"Jika Anda tidak meminta reset password, abaikan email ini.", expiredAt)
	kirimEmailAsync(s.mailer, user.Email, "Reset Password Enviroo APP", body)

	return &ResetOTPResult{AktivasiID: aktivasiID, ExpiredAt: expiredAt}, nil
}

func (s *akunService) VerifikasiOTPReset(email, otp string) (*VerifikasiOTPResult, error) {
	user, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, errNotFound("Pengguna tidak ditemukan")
	}

	aktivasi, err := s.repo.FindAktivasiValidTerbaru(nil, user.UserID, models.TujuanResetPassword)
	if err != nil {
		return nil, errNotFound("Kode OTP tidak ditemukan atau sudah kadaluarsa")
	}

	if !utils.VerifyOTP(otp, aktivasi.Token) {
		return nil, errBadRequest("Kode OTP salah")
	}

	return &VerifikasiOTPResult{UserID: user.UserID, AktivasiID: aktivasi.AktivasiID}, nil
}

func (s *akunService) ResetPassword(in ResetPasswordInput) error {
	if in.PasswordBaru != in.KonfirmasiPasswordBaru {
		return errBadRequest("Konfirmasi password baru tidak cocok")
	}
	if err := utils.ValidasiKekuatanPassword(in.PasswordBaru); err != nil {
		return errBadRequest(err.Error())
	}

	user, err := s.repo.FindUserByEmail(in.Email)
	if err != nil {
		return errNotFound("Pengguna tidak ditemukan")
	}

	aktivasi, err := s.repo.FindAktivasiValidTerbaru(nil, user.UserID, models.TujuanResetPassword)
	if err != nil {
		return errNotFound("Kode OTP tidak valid atau sudah kadaluarsa")
	}

	if !utils.VerifyOTP(in.OTP, aktivasi.Token) {
		return errBadRequest("Kode OTP salah")
	}

	hashed, err := utils.HashPassword(in.PasswordBaru)
	if err != nil {
		return errInternal("Gagal memproses password baru")
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.UpdatePassword(tx, user.UserID, hashed); err != nil {
			return errInternal("Gagal mengupdate password")
		}

		if err := s.repo.ClearAllSessions(tx, user.UserID); err != nil {
			return errInternal("Gagal mengakhiri sesi lama")
		}

		if err := s.repo.MarkAktivasiUsed(tx, aktivasi.AktivasiID); err != nil {
			return errInternal("Gagal menyelesaikan proses reset password")
		}
		return nil
	})
	if txErr != nil {
		return bungkusTxError(txErr, "Gagal menyimpan data")
	}

	body := mailer.BuildNotifEmail(user.Nama,
		"Password akun Enviroo Anda berhasil direset. Silakan login menggunakan password baru Anda. Jika Anda tidak merasa melakukan reset password, segera hubungi administrator.")
	kirimEmailAsync(s.mailer, user.Email, "Konfirmasi Reset Password Enviroo", body)

	return nil
}

func (s *akunService) pastikanTidakAdaNasabahAktifLain(tx *gorm.DB, userID, nasabahID string) error {
	count, err := s.repo.CountNasabahAktifSelain(tx, userID, nasabahID)
	if err != nil {
		return errInternal("Gagal memvalidasi data nasabah")
	}
	if count > 0 {
		return errConflict("User sudah memiliki akun nasabah aktif di bank lain. Nonaktifkan akun tersebut terlebih dahulu.", nil)
	}
	return nil
}

func (s *akunService) pastikanTidakAdaAdminAktifLain(tx *gorm.DB, userID, adminID string) error {
	count, err := s.repo.CountAdminAktifSelain(tx, userID, adminID)
	if err != nil {
		return errInternal("Gagal memvalidasi data admin")
	}
	if count > 0 {
		return errConflict("User sudah memiliki akun admin aktif. Nonaktifkan akun tersebut terlebih dahulu.", nil)
	}
	return nil
}

const pesanPlatformTidakSesuai = "Peran akun anda tidak sesuai untuk platform ini"

func platformSesuaiAdmin(role models.RoleAdmin, platform string) bool {
	switch platform {
	case models.PlatformMobile:
		return isPetugas(role)
	case models.PlatformWeb:
		return isAdminTier(role)
	}
	return false
}

func cekAksesBank(actorBankID string, targetBankID *string) error {
	if actorBankID == "" {
		return nil
	}
	if targetBankID == nil || *targetBankID != actorBankID {
		return errForbidden("Akses ditolak. Anda tidak memiliki akses ke akun di bank sampah ini.")
	}
	return nil
}

func bungkusTxError(err error, pesanDefault string) error {
	if err == nil {
		return nil
	}
	if _, ok := err.(*ServiceError); ok {
		return err
	}
	return errInternal(pesanDefault)
}

func kirimEmailAsync(m MailerIface, to, subject, body string) {
	go func() {
		if err := m.SendEmail(mailer.Params{To: to, Subject: subject, Body: body}); err != nil {
			log.Printf("[Email] Gagal kirim '%s' ke %s: %v", subject, to, err)
		}
	}()
}
