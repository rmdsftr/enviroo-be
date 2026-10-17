package services

import (
	"errors"
	"fmt"
	"time"

	"enviroo-be/internal/adapters/mailer"
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"

	"gorm.io/gorm"
)

type NasabahRepoIface interface {
	CountUser(userID string) (int64, error)
	CountUserByEmail(email string) (int64, error)
	FindUser(userID string) (*models.User, error)
	FindBank(bankID string) (*models.BankSampah, error)
	CountNasabahAktif(userID string) (int64, error)
	CountNasabahPendingDiBankLain(userID, bankID string) (int64, error)
	FindNasabahByUserAndBank(userID, bankID string) (*models.Nasabah, error)

	CreateUser(tx *gorm.DB, user *models.User) error
	CreateNasabah(tx *gorm.DB, nasabah *models.Nasabah) error
	FindRewardByNama(tx *gorm.DB, nama models.RewardEnum) (*models.Reward, error)
	CreateSaldoRekening(tx *gorm.DB, saldo *models.SaldoRekening) error
	CreateRiwayatSaldo(tx *gorm.DB, riwayat *models.RiwayatArusSaldo) error
	DeleteAktivasiNasabahBelumDipakai(tx *gorm.DB, userID string) error

	GetProfilNasabah(nasabahID string) (*repositories.ProfilNasabahRow, error)
	GetDetailNasabah(nasabahID string) (*repositories.DetailNasabahRow, error)

	FindNasabah(nasabahID string) (*models.Nasabah, error)
	CountTransaksiNasabah(nasabahID string) (int64, error)
	PluckRekeningIDs(tx *gorm.DB, nasabahID string) ([]string, error)
	DeleteRiwayatByRekening(tx *gorm.DB, rekeningIDs []string) error
	DeleteSaldoRekening(tx *gorm.DB, nasabahID string) error
	DeleteNasabah(tx *gorm.DB, nasabah *models.Nasabah) error
}

type DaftarNasabahBaruInput struct {
	UserID      string
	Nama        string
	Email       string
	NoWhatsapp  string
	BankID      string
	AdminID     string
	NoRekening  string
	SaldoRupiah float64
	SaldoPoin   float64
}

type DaftarNasabahUserLamaInput struct {
	UserID      string
	BankID      string
	AdminID     string
	NoRekening  string
	SaldoRupiah float64
	SaldoPoin   float64
}

type pendaftaranNasabah struct {
	UserID      string
	BankID      string
	AdminID     string
	NoRekening  string
	SaldoRupiah float64
	SaldoPoin   float64
}

type NasabahService interface {
	DaftarNasabahBaru(in DaftarNasabahBaruInput) (*models.Nasabah, error)
	DaftarNasabahUserLama(in DaftarNasabahUserLamaInput) (*models.Nasabah, error)
	HapusNasabah(nasabahID string) error
	GetProfilNasabah(nasabahID string) (*repositories.ProfilNasabahRow, error)
	GetDetailNasabah(nasabahID string) (*repositories.DetailNasabahRow, error)
}

type nasabahService struct {
	db     *gorm.DB
	repo   NasabahRepoIface
	mailer MailerIface
}

func NewNasabahService(db *gorm.DB, repo NasabahRepoIface, mailer MailerIface) NasabahService {
	return &nasabahService{db: db, repo: repo, mailer: mailer}
}

func (s *nasabahService) DaftarNasabahBaru(in DaftarNasabahBaruInput) (*models.Nasabah, error) {
	countUser, err := s.repo.CountUser(in.UserID)
	if err != nil {
		return nil, errInternal("Gagal memvalidasi data user")
	}
	if countUser > 0 {
		return nil, errConflict("User dengan ID ini sudah terdaftar. Gunakan menu pilih akun lama.", nil)
	}

	countEmail, err := s.repo.CountUserByEmail(in.Email)
	if err != nil {
		return nil, errInternal("Gagal memvalidasi data user")
	}
	if countEmail > 0 {
		return nil, errConflict("Email sudah digunakan", nil)
	}

	bank, err := s.repo.FindBank(in.BankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	var nasabah *models.Nasabah
	var otp string
	var expiredAt time.Time

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.CreateUser(tx, &models.User{
			UserID:     in.UserID,
			Nama:       in.Nama,
			Email:      in.Email,
			NoWhatsapp: in.NoWhatsapp,
		}); err != nil {
			return errInternal("Failed to create user: " + err.Error())
		}

		nasabah, otp, expiredAt, err = s.daftarkanNasabah(tx, pendaftaranNasabah{
			UserID:      in.UserID,
			BankID:      in.BankID,
			AdminID:     in.AdminID,
			NoRekening:  in.NoRekening,
			SaldoRupiah: in.SaldoRupiah,
			SaldoPoin:   in.SaldoPoin,
		})
		return err
	})
	if txErr != nil {
		return nil, bungkusTxError(txErr, "Failed to commit transaction")
	}

	s.kirimEmailAktivasi(in.Email, in.Nama, bank.NamaBank, otp, expiredAt)
	return nasabah, nil
}

func (s *nasabahService) DaftarNasabahUserLama(in DaftarNasabahUserLamaInput) (*models.Nasabah, error) {
	user, err := s.repo.FindUser(in.UserID)
	if err != nil {
		return nil, errNotFound("User dengan ID ini tidak ditemukan")
	}

	countAktif, err := s.repo.CountNasabahAktif(in.UserID)
	if err != nil {
		return nil, errInternal("Gagal memvalidasi data nasabah")
	}
	if countAktif > 0 {
		return nil, errConflict("User sudah memiliki akun nasabah aktif. Nonaktifkan akun nasabah yang ada terlebih dahulu.", nil)
	}

	countPendingLain, err := s.repo.CountNasabahPendingDiBankLain(in.UserID, in.BankID)
	if err != nil {
		return nil, errInternal("Gagal memvalidasi data nasabah")
	}
	if countPendingLain > 0 {
		return nil, errConflict("User masih dalam proses pendaftaran di bank sampah lain. Selesaikan atau batalkan pendaftaran tersebut terlebih dahulu.", nil)
	}

	existing, err := s.repo.FindNasabahByUserAndBank(in.UserID, in.BankID)
	if err == nil {
		if existing.StatusNasabah == models.Nonaktif {
			return nil, errConflict("User pernah terdaftar di bank sampah ini. Gunakan fitur aktivasi untuk mengaktifkan kembali.", nil)
		}
		return nil, errConflict("User sudah terdaftar sebagai nasabah di bank sampah ini.", nil)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errInternal("Gagal memvalidasi data nasabah")
	}

	bank, err := s.repo.FindBank(in.BankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	var nasabah *models.Nasabah
	var otp string
	var expiredAt time.Time

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		var err error
		nasabah, otp, expiredAt, err = s.daftarkanNasabah(tx, pendaftaranNasabah{
			UserID:      in.UserID,
			BankID:      in.BankID,
			AdminID:     in.AdminID,
			NoRekening:  in.NoRekening,
			SaldoRupiah: in.SaldoRupiah,
			SaldoPoin:   in.SaldoPoin,
		})
		return err
	})
	if txErr != nil {
		return nil, bungkusTxError(txErr, "Failed to commit transaction")
	}

	s.kirimEmailAktivasi(user.Email, user.Nama, bank.NamaBank, otp, expiredAt)
	return nasabah, nil
}

func (s *nasabahService) HapusNasabah(nasabahID string) error {
	nasabah, err := s.repo.FindNasabah(nasabahID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errNotFound("Nasabah tidak ditemukan")
		}
		return errInternal("Gagal mengambil data nasabah")
	}

	jumlahTransaksi, err := s.repo.CountTransaksiNasabah(nasabahID)
	if err != nil {
		return errInternal("Gagal memvalidasi data nasabah")
	}
	if jumlahTransaksi > 0 {
		return errConflict("Nasabah tidak dapat dihapus karena masih memiliki riwayat transaksi.", nil)
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		rekeningIDs, err := s.repo.PluckRekeningIDs(tx, nasabahID)
		if err != nil {
			return err
		}
		if err := s.repo.DeleteRiwayatByRekening(tx, rekeningIDs); err != nil {
			return err
		}
		if err := s.repo.DeleteSaldoRekening(tx, nasabahID); err != nil {
			return err
		}
		return s.repo.DeleteNasabah(tx, nasabah)
	})
	if txErr != nil {
		return errInternal("Gagal menghapus nasabah")
	}

	return nil
}

func (s *nasabahService) GetProfilNasabah(nasabahID string) (*repositories.ProfilNasabahRow, error) {
	profil, err := s.repo.GetProfilNasabah(nasabahID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Nasabah tidak ditemukan")
		}
		return nil, errInternal("Failed to get nasabah profile: " + err.Error())
	}
	return profil, nil
}

func (s *nasabahService) GetDetailNasabah(nasabahID string) (*repositories.DetailNasabahRow, error) {
	detail, err := s.repo.GetDetailNasabah(nasabahID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Nasabah tidak ditemukan")
		}
		return nil, errInternal("Gagal mengambil detail nasabah: " + err.Error())
	}
	return detail, nil
}

func (s *nasabahService) daftarkanNasabah(tx *gorm.DB, in pendaftaranNasabah) (*models.Nasabah, string, time.Time, error) {
	nasabahID, err := idgen.GenerateNasabahID(tx, in.BankID)
	if err != nil {
		return nil, "", time.Time{}, errInternal("Failed to generate nasabah ID: " + err.Error())
	}

	nomorRekening := in.NoRekening
	if nomorRekening == "" {
		nomorRekening = nasabahID
	}

	nasabah := &models.Nasabah{
		NasabahID:     nasabahID,
		UserID:        in.UserID,
		BankID:        in.BankID,
		NomorRekening: nomorRekening,
		StatusNasabah: models.Pending,
	}
	if err := s.repo.CreateNasabah(tx, nasabah); err != nil {
		return nil, "", time.Time{}, errInternal("Failed to create nasabah: " + err.Error())
	}

	if err := s.buatRekeningAwal(tx, nasabahID, in.AdminID, models.RewardEnumUang,
		models.SatuanRewardEnumRp, in.SaldoRupiah, "rupiah"); err != nil {
		return nil, "", time.Time{}, err
	}
	if err := s.buatRekeningAwal(tx, nasabahID, in.AdminID, models.RewardEnumBarang,
		models.SatuanRewardEnumPoin, in.SaldoPoin, "poin"); err != nil {
		return nil, "", time.Time{}, err
	}

	if err := s.repo.DeleteAktivasiNasabahBelumDipakai(tx, in.UserID); err != nil {
		return nil, "", time.Time{}, errInternal("Failed to clean up old activation: " + err.Error())
	}

	otp, expiredAt, err := createAktivasiToken(tx, in.UserID, models.RoleUserNasabah, &nasabah.NasabahID, nil, in.AdminID)
	if err != nil {
		return nil, "", time.Time{}, errInternal("Failed to create aktivasi akun: " + err.Error())
	}

	return nasabah, otp, expiredAt, nil
}

func (s *nasabahService) buatRekeningAwal(tx *gorm.DB, nasabahID, adminID string, namaReward models.RewardEnum, satuan models.SatuanRewardEnum, nominal float64, label string) error {
	reward, err := s.repo.FindRewardByNama(tx, namaReward)
	if err != nil {
		return errInternal("Reward " + string(namaReward) + " tidak ditemukan: " + err.Error())
	}

	rekeningID, err := idgen.GenerateRekeningID(reward.RewardID, nasabahID)
	if err != nil {
		return errInternal("Gagal generate rekening ID: " + err.Error())
	}

	rewardID := reward.RewardID
	saldo := &models.SaldoRekening{
		RekeningID:         rekeningID,
		NasabahID:          &nasabahID,
		RewardID:           &rewardID,
		Entitas:            models.EntitasNasabah,
		NominalSaldo:       nominal,
		SatuanNominalSaldo: satuan,
		LastUpdatedBy:      &adminID,
	}
	if err := s.repo.CreateSaldoRekening(tx, saldo); err != nil {
		return errInternal("Failed to create saldo " + label + ": " + err.Error())
	}

	riwayatID, err := idgen.GenerateRiwayatSaldoID(tx, saldo.RekeningID)
	if err != nil {
		return errInternal("Gagal generate riwayat saldo ID: " + err.Error())
	}

	if err := s.repo.CreateRiwayatSaldo(tx, &models.RiwayatArusSaldo{
		RiwayatSaldoID: riwayatID,
		RekeningID:     &saldo.RekeningID,
		NominalSebelum: 0,
		NominalSesudah: nominal,
		CreatedBy:      &adminID,
		Keterangan:     models.KeteranganSaldoAwal,
	}); err != nil {
		return errInternal("Failed to create riwayat saldo " + label + ": " + err.Error())
	}

	return nil
}

func (s *nasabahService) kirimEmailAktivasi(email, nama, namaBank, otp string, expiredAt time.Time) {
	paragraf := fmt.Sprintf("Terima kasih telah mendaftar sebagai Nasabah di <b>%s</b>. Untuk menyelesaikan proses aktivasi akun, gunakan kode OTP berikut:", namaBank)
	body := mailer.BuildOTPEmail(nama, paragraf, otp, "24 jam",
		"Mohon untuk tidak membagikan kode OTP ini ke siapapun.", expiredAt)
	kirimEmailAsync(s.mailer, email, "Aktivasi Akun Nasabah Enviroo", body)
}
