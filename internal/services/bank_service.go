package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"enviroo-be/internal/adapters/mailer"
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BankRepoIface interface {
	FindBank(bankID string) (*models.BankSampah, error)
	ListBankAktifWithWilayah() ([]models.BankSampah, error)
	ListSemuaBank() ([]models.BankSampah, error)
	ListNasabahByBank(bankID string, limit, offset int) ([]repositories.NasabahBankRow, error)
	ListBSI(limit, offset int) ([]repositories.BSIRow, error)
	ListBSU(limit, offset int) ([]repositories.BSURow, error)
	ListBSM(limit, offset int) ([]repositories.BSMRow, error)
	ListUnitByParent(parentBankID string, limit, offset int) ([]repositories.UnitBSURow, error)
	ListBSUByParent(parentBankID string) ([]repositories.BSUSimpleRow, error)
	FindRewardByNama(tx *gorm.DB, nama models.RewardEnum) (*models.Reward, error)
	FindUserTx(tx *gorm.DB, userID string) (*models.User, error)
	CountAdminAktifPending(tx *gorm.DB, userID string) (int64, error)
	CreateBank(tx *gorm.DB, b *models.BankSampah) error
	SaveBank(tx *gorm.DB, b *models.BankSampah) error
	CreateSaldoRekening(tx *gorm.DB, s *models.SaldoRekening) error
	CreateAdmin(tx *gorm.DB, a *models.Admin) error
	CreateHistoryAkunBank(tx *gorm.DB, h *models.HistoryAkunBank) error
	GetProfilBank(bankID string) (*repositories.BankProfilRow, error)
	GetHistoryAkunBank(bankID string) ([]repositories.HistoryBankRow, error)
	FindBankWithRelasi(bankID string) (*models.BankSampah, error)
	ListAdminWithUserByBank(bankID string) ([]models.Admin, error)
	ListAdminBank(bankID string) ([]repositories.AdminBankRow, error)
	DeleteBank(bank *models.BankSampah) error
	FindUser(userID string) (*models.User, error)
	FindAdminWithUserBank(adminID string) (*models.Admin, error)
	FindNamaAdmin(adminID string) string
	FindAdminByUserAndBankTx(tx *gorm.DB, userID, bankID string) (*models.Admin, error)
	FindAdminAktifByUser(userID string) (*models.Admin, error)
	AdminPernahAktif(adminID string) (bool, error)
	DeleteAktivasiAdminBelumDipakai(tx *gorm.DB, userID string, tujuan *models.TujuanEnum) error
	DeleteAdmin(tx *gorm.DB, adminID string) error
	UpdateStatusAdmin(tx *gorm.DB, adminID string, status models.StatusAkun) error
}

type MailerIface interface {
	SendEmail(params mailer.Params) error
}

const (
	bankPageLimit = 20
	unitPageLimit = 15
)

type CreateBankInput struct {
	JenisBank     models.JenisBank
	ParentBankID  *string
	Nama          string
	Deskripsi     string
	Provinsi      string
	KabupatenKota string
	IDKecamatan   int
	IDKelurahan   int
	Alamat        string
	Latitude      float64
	Longitude     float64
	FotoURL       string
	UserIDs       []string
	ActorAdminID  string
}

type AktivasiBankInput struct {
	AdminID    string
	Informasi  string
	Keterangan string
}

type UpdateProfilBankInput struct {
	AdminID   string
	NamaBank  string
	Alamat    string
	Deskripsi string
	Latitude  *float64
	Longitude *float64
	FotoURL   string
	HapusFoto bool
}

type BankPublicItem struct {
	BankID        string  `json:"bank_id"`
	NamaBank      string  `json:"nama_bank"`
	AlamatBank    string  `json:"alamat_bank"`
	Kecamatan     string  `json:"kecamatan"`
	JenisBank     string  `json:"jenis_bank"`
	Kelurahan     string  `json:"kelurahan"`
	KabupatenKota string  `json:"kabupaten_kota"`
	Provinsi      string  `json:"provinsi"`
	PhotoURL      string  `json:"photo_url"`
	Latitude      float64 `json:"latitude"`
	Longitude     float64 `json:"longitude"`
}

type bankJenisConfig struct {
	label      string
	rewards    []models.RewardEnum
	adminRole  models.RoleAdmin
	emailLabel string
}

func configFor(jenis models.JenisBank) (bankJenisConfig, bool) {
	switch jenis {
	case models.BSI:
		return bankJenisConfig{
			label:     "BSI",
			rewards:   []models.RewardEnum{models.RewardEnumUang, models.RewardEnumBarang},
			adminRole: models.AdminBSI,
		}, true
	case models.BSM:
		return bankJenisConfig{
			label:      "BSM",
			rewards:    []models.RewardEnum{models.RewardEnumUang, models.RewardEnumBarang},
			adminRole:  models.AdminBSM,
			emailLabel: " (BSM)",
		}, true
	case models.BSU:
		return bankJenisConfig{
			label:      "BSU",
			rewards:    []models.RewardEnum{models.RewardEnumUang},
			adminRole:  models.AdminBSU,
			emailLabel: " (BSU)",
		}, true
	default:
		return bankJenisConfig{}, false
	}
}

type BankService interface {
	CreateBank(in CreateBankInput) (*models.BankSampah, error)

	ListBankAktif() ([]BankPublicItem, error)
	ListSemuaBank() ([]models.BankSampah, error)
	ListNasabahBank(bankID string, page int) ([]repositories.NasabahBankRow, *utils.PageMeta, error)
	ListBSI(page int) ([]repositories.BSIRow, *utils.PageMeta, error)
	ListBSU(page int) ([]repositories.BSURow, *utils.PageMeta, error)
	ListBSM(page int) ([]repositories.BSMRow, *utils.PageMeta, error)
	ListUnitBSI(parentBankID string, page int) ([]repositories.UnitBSURow, *utils.PageMeta, error)
	ListBSUByParent(parentBankID string) ([]repositories.BSUSimpleRow, error)

	ToggleAktivasiBank(bankID string, in AktivasiBankInput) (*models.BankSampah, error)
	UpdateProfilBank(bankID string, in UpdateProfilBankInput) (*models.BankSampah, error)

	GetProfilBank(bankID string) (*repositories.BankProfilRow, error)
	GetDetailBank(bankID string) (*DetailBankItem, error)
	GetHistoryAkunBank(bankID string) ([]repositories.HistoryBankRow, error)
	DeleteBank(bankID string) error

	ListAdminBank(bankID string) ([]repositories.AdminBankRow, error)
	AddAdminBank(in AddAdminBankInput) (*models.Admin, error)
	DeleteStaffBank(adminID, actorUserID string) (*DeleteStaffResult, error)
}

type bankService struct {
	db     *gorm.DB
	repo   BankRepoIface
	mailer MailerIface
}

func NewBankService(db *gorm.DB, repo BankRepoIface, mailer MailerIface) BankService {
	return &bankService{db: db, repo: repo, mailer: mailer}
}

func (s *bankService) CreateBank(in CreateBankInput) (*models.BankSampah, error) {
	cfg, ok := configFor(in.JenisBank)
	if !ok {
		return nil, errBadRequest("Jenis bank tidak valid")
	}

	newBank := &models.BankSampah{
		NamaBank:      in.Nama,
		Deskripsi:     in.Deskripsi,
		ParentBankID:  in.ParentBankID,
		PhotoURL:      in.FotoURL,
		Provinsi:      in.Provinsi,
		KabupatenKota: in.KabupatenKota,
		IDKecamatan:   &in.IDKecamatan,
		IDKelurahan:   &in.IDKelurahan,
		Alamat:        in.Alamat,
		Latitude:      in.Latitude,
		Longitude:     in.Longitude,
		JenisBank:     in.JenisBank,
		IsActive:      true,
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		bankID, err := idgen.GenerateBankID(tx, in.JenisBank, in.IDKecamatan, in.IDKelurahan)
		if err != nil {
			return errInternal("Gagal generate Bank ID: " + err.Error())
		}
		newBank.BankID = bankID

		if err := s.repo.CreateBank(tx, newBank); err != nil {
			return errInternal("Failed to create " + cfg.label + ": " + err.Error())
		}

		if err := s.createRekeningBank(tx, newBank.BankID, cfg.rewards); err != nil {
			return err
		}

		for _, userID := range in.UserIDs {
			if err := s.createAdminBank(tx, newBank, cfg, userID, in.ActorAdminID); err != nil {
				return err
			}
		}

		return nil
	})

	if txErr != nil {
		if _, ok := txErr.(*ServiceError); ok {
			return nil, txErr
		}
		return nil, errInternal("Failed to commit transaction: " + txErr.Error())
	}

	return newBank, nil
}

func (s *bankService) createRekeningBank(tx *gorm.DB, bankID string, rewards []models.RewardEnum) error {
	for _, namaReward := range rewards {
		reward, err := s.repo.FindRewardByNama(tx, namaReward)
		if err != nil {
			return errInternal("Reward " + string(namaReward) + " tidak ditemukan: " + err.Error())
		}

		satuan := models.SatuanRewardEnumRp
		if reward.Satuan == models.SatuanRewardEnumPoin {
			satuan = models.SatuanRewardEnumPoin
		}

		rekeningID, err := idgen.GenerateRekeningID(reward.RewardID, bankID)
		if err != nil {
			return errInternal("Gagal generate rekening ID: " + err.Error())
		}

		rewardID := reward.RewardID
		rekening := &models.SaldoRekening{
			RekeningID:         rekeningID,
			BankID:             &bankID,
			RewardID:           &rewardID,
			Entitas:            models.EntitasBankSampah,
			NominalSaldo:       0,
			SatuanNominalSaldo: satuan,
		}
		if err := s.repo.CreateSaldoRekening(tx, rekening); err != nil {
			return errInternal("Gagal membuat rekening bank: " + err.Error())
		}
	}
	return nil
}

func (s *bankService) createAdminBank(tx *gorm.DB, bank *models.BankSampah, cfg bankJenisConfig, userID, actorAdminID string) error {
	countAdmin, err := s.repo.CountAdminAktifPending(tx, userID)
	if err != nil {
		return errInternal("Gagal memvalidasi data admin: " + err.Error())
	}
	if countAdmin > 0 {
		return errConflict("User sudah memiliki akun admin aktif. Nonaktifkan akun admin yang ada terlebih dahulu.", nil)
	}

	adminID, err := idgen.GenerateAdminID(tx, cfg.adminRole, bank.BankID)
	if err != nil {
		return errInternal("Failed to generate admin ID: " + err.Error())
	}

	newAdmin := &models.Admin{
		AdminID:     adminID,
		UserID:      userID,
		BankID:      &bank.BankID,
		Role:        cfg.adminRole,
		StatusAdmin: models.Pending,
	}
	if err := s.repo.CreateAdmin(tx, newAdmin); err != nil {
		return errInternal("Failed to add admin: " + err.Error())
	}

	existingUser, err := s.repo.FindUserTx(tx, userID)
	if err != nil {
		return errInternal("Failed to get user data for activation")
	}

	otp, expiredAt, err := createAktivasiToken(tx, userID, models.RoleUserAdmin, nil, &newAdmin.AdminID, actorAdminID)
	if err != nil {
		return errInternal("Failed to create aktivasi akun: " + err.Error())
	}

	emailBody := mailer.BuildAktivasiAdminEmail(existingUser.Nama, "Administrator", bank.NamaBank, cfg.emailLabel, otp, expiredAt)
	subject := "Aktivasi Akun Administrator Enviroo"
	if cfg.emailLabel != "" {
		subject = "Aktivasi Akun Administrator" + cfg.emailLabel + " Enviroo"
	}

	if err := s.mailer.SendEmail(mailer.Params{
		To:      existingUser.Email,
		Subject: subject,
		Body:    emailBody,
	}); err != nil {
		return errInternal("Failed to send email: " + err.Error())
	}

	return nil
}

func (s *bankService) ListBankAktif() ([]BankPublicItem, error) {
	banks, err := s.repo.ListBankAktifWithWilayah()
	if err != nil {
		return nil, errInternal("Failed to get all banks: " + err.Error())
	}

	items := make([]BankPublicItem, 0, len(banks))
	for _, bank := range banks {
		kecamatanStr := ""
		if bank.Kecamatan != nil {
			kecamatanStr = bank.Kecamatan.Kecamatan
		}
		kelurahanStr := ""
		if bank.Kelurahan != nil {
			kelurahanStr = bank.Kelurahan.Kelurahan
		}

		items = append(items, BankPublicItem{
			BankID:        bank.BankID,
			NamaBank:      bank.NamaBank,
			AlamatBank:    bank.Alamat,
			JenisBank:     string(bank.JenisBank),
			Kecamatan:     kecamatanStr,
			Kelurahan:     kelurahanStr,
			KabupatenKota: bank.KabupatenKota,
			Provinsi:      bank.Provinsi,
			PhotoURL:      bank.PhotoURL,
			Latitude:      bank.Latitude,
			Longitude:     bank.Longitude,
		})
	}

	return items, nil
}

func (s *bankService) ListSemuaBank() ([]models.BankSampah, error) {
	banks, err := s.repo.ListSemuaBank()
	if err != nil {
		return nil, errInternal("Failed to get afiliasi: " + err.Error())
	}
	return banks, nil
}

func (s *bankService) ListNasabahBank(bankID string, page int) ([]repositories.NasabahBankRow, *utils.PageMeta, error) {
	if bankID == "" {
		return nil, nil, errBadRequest("Bank ID is required")
	}

	limit, offset := pageBounds(page, bankPageLimit)
	rows, err := s.repo.ListNasabahByBank(bankID, limit, offset)
	if err != nil {
		return nil, nil, errInternal("Failed to get nasabah: " + err.Error())
	}
	if page <= 0 {
		return rows, nil, nil
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}
	return rows, utils.NewPageMeta(page, limit, total), nil
}

func (s *bankService) ListBSI(page int) ([]repositories.BSIRow, *utils.PageMeta, error) {
	limit, offset := pageBounds(page, bankPageLimit)
	rows, err := s.repo.ListBSI(limit, offset)
	if err != nil {
		return nil, nil, errInternal("Failed to get BSI: " + err.Error())
	}
	if page <= 0 {
		return rows, nil, nil
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}
	return rows, utils.NewPageMeta(page, limit, total), nil
}

func (s *bankService) ListBSU(page int) ([]repositories.BSURow, *utils.PageMeta, error) {
	limit, offset := pageBounds(page, bankPageLimit)
	rows, err := s.repo.ListBSU(limit, offset)
	if err != nil {
		return nil, nil, errInternal("Failed to get BSU: " + err.Error())
	}
	if page <= 0 {
		return rows, nil, nil
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}
	return rows, utils.NewPageMeta(page, limit, total), nil
}

func (s *bankService) ListBSM(page int) ([]repositories.BSMRow, *utils.PageMeta, error) {
	limit, offset := pageBounds(page, bankPageLimit)
	rows, err := s.repo.ListBSM(limit, offset)
	if err != nil {
		return nil, nil, errInternal("Failed to get BSM: " + err.Error())
	}
	if page <= 0 {
		return rows, nil, nil
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}
	return rows, utils.NewPageMeta(page, limit, total), nil
}

func (s *bankService) ListUnitBSI(parentBankID string, page int) ([]repositories.UnitBSURow, *utils.PageMeta, error) {
	if parentBankID == "" {
		return nil, nil, errBadRequest("Bank ID is required")
	}

	limit, offset := pageBounds(page, unitPageLimit)
	rows, err := s.repo.ListUnitByParent(parentBankID, limit, offset)
	if err != nil {
		return nil, nil, errInternal("Failed to get BSU: " + err.Error())
	}
	if page <= 0 {
		return rows, nil, nil
	}

	total := 0
	if len(rows) > 0 {
		total = rows[0].TotalCount
	}
	return rows, utils.NewPageMeta(page, limit, total), nil
}

func (s *bankService) ListBSUByParent(parentBankID string) ([]repositories.BSUSimpleRow, error) {
	rows, err := s.repo.ListBSUByParent(parentBankID)
	if err != nil {
		return nil, errInternal("Failed to get BSU: " + err.Error())
	}
	return rows, nil
}

func (s *bankService) ToggleAktivasiBank(bankID string, in AktivasiBankInput) (*models.BankSampah, error) {
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	oldStatus := statusBankLabel(bank.IsActive)
	bank.IsActive = !bank.IsActive
	newStatus := statusBankLabel(bank.IsActive)

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.SaveBank(tx, bank); err != nil {
			return err
		}
		return s.writeHistory(tx, bankID, "UPDATE",
			map[string]string{"status_bank": oldStatus},
			map[string]string{"status_bank": newStatus},
			in.Informasi, in.Keterangan, in.AdminID)
	})
	if txErr != nil {
		return nil, errInternal("Gagal memperbarui status bank")
	}

	return bank, nil
}

func (s *bankService) UpdateProfilBank(bankID string, in UpdateProfilBankInput) (*models.BankSampah, error) {
	if in.AdminID == "" {
		return nil, errBadRequest("Admin ID wajib diisi")
	}

	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	oldValue := profilBankSnapshot(bank)

	if in.NamaBank != "" {
		bank.NamaBank = in.NamaBank
	}
	if in.Alamat != "" {
		bank.Alamat = in.Alamat
	}
	if in.Deskripsi != "" {
		bank.Deskripsi = in.Deskripsi
	}
	if in.Latitude != nil {
		bank.Latitude = *in.Latitude
	}
	if in.Longitude != nil {
		bank.Longitude = *in.Longitude
	}
	if in.FotoURL != "" {
		bank.PhotoURL = in.FotoURL
	} else if in.HapusFoto {
		bank.PhotoURL = ""
	}

	newValue := profilBankSnapshot(bank)

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.SaveBank(tx, bank); err != nil {
			return err
		}
		return s.writeHistory(tx, bankID, "UPDATE", oldValue, newValue,
			"Update profil bank sampah",
			"Admin memperbarui informasi profil bank sampah",
			in.AdminID)
	})
	if txErr != nil {
		return nil, errInternal("Gagal memperbarui profil bank")
	}

	return bank, nil
}

func pageBounds(page, limit int) (int, int) {
	if page <= 0 {
		return 0, 0
	}
	return limit, utils.Offset(page, limit)
}

func statusBankLabel(isActive bool) string {
	if isActive {
		return "Aktif"
	}
	return "Non Aktif"
}

func profilBankSnapshot(bank *models.BankSampah) map[string]interface{} {
	return map[string]interface{}{
		"nama_bank": bank.NamaBank,
		"photo_url": bank.PhotoURL,
		"alamat":    bank.Alamat,
		"latitude":  bank.Latitude,
		"longitude": bank.Longitude,
		"deskripsi": bank.Deskripsi,
	}
}

func (s *bankService) writeHistory(tx *gorm.DB, bankID, action string, oldValue, newValue interface{}, informasi, keterangan, createdBy string) error {
	historyBankID, err := idgen.GenerateHistoryBankID(tx, action, bankID)
	if err != nil {
		return err
	}

	var oldVal, newVal datatypes.JSON
	if oldValue != nil {
		b, _ := json.Marshal(oldValue)
		oldVal = datatypes.JSON(b)
	}
	if newValue != nil {
		b, _ := json.Marshal(newValue)
		newVal = datatypes.JSON(b)
	}

	return s.repo.CreateHistoryAkunBank(tx, &models.HistoryAkunBank{
		HistoryBankID: historyBankID,
		BankID:        bankID,
		Action:        action,
		OldValue:      oldVal,
		NewValue:      newVal,
		Informasi:     informasi,
		Keterangan:    keterangan,
		CreatedBy:     createdBy,
	})
}

type AddAdminBankInput struct {
	UserID       string
	BankID       string
	Role         string
	ActorAdminID string
}

type DetailBankAdminItem struct {
	AdminID   string `json:"admin_id"`
	NamaAdmin string `json:"nama_admin"`
	PhotoURL  string `json:"photo_url"`
	RoleAdmin string `json:"role_admin"`
}

type DetailBankItem struct {
	BankID        string                `json:"bank_id"`
	NamaBank      string                `json:"nama_bank"`
	PhotoURL      string                `json:"photo_url"`
	IsBSU         bool                  `json:"is_bsu"`
	BankIndukNama string                `json:"bank_induk_nama"`
	JenisBank     string                `json:"jenis_bank"`
	Alamat        string                `json:"alamat"`
	Provinsi      string                `json:"provinsi"`
	KabupatenKota string                `json:"kabupaten_kota"`
	Kecamatan     string                `json:"kecamatan"`
	Kelurahan     string                `json:"kelurahan"`
	Deskripsi     string                `json:"deskripsi"`
	IsActive      bool                  `json:"is_active"`
	JoinedAt      time.Time             `json:"joined_at"`
	Latitude      float64               `json:"latitude"`
	Longitude     float64               `json:"longitude"`
	Admins        []DetailBankAdminItem `json:"admins"`
}

type DeleteStaffResult struct {
	StaffName string
	BankName  string
}

var validAdminRoles = map[models.RoleAdmin]bool{
	models.AdminBSI:   true,
	models.AdminBSM:   true,
	models.AdminBSU:   true,
	models.PetugasBSI: true,
	models.PetugasBSM: true,
	models.PetugasBSU: true,
}

func (s *bankService) GetProfilBank(bankID string) (*repositories.BankProfilRow, error) {
	profil, err := s.repo.GetProfilBank(bankID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Bank sampah tidak ditemukan")
		}
		return nil, errInternal("Failed to get bank sampah profile: " + err.Error())
	}
	return profil, nil
}

func (s *bankService) GetDetailBank(bankID string) (*DetailBankItem, error) {
	if bankID == "" {
		return nil, errBadRequest("Bank ID wajib diisi")
	}

	bank, err := s.repo.FindBankWithRelasi(bankID)
	if err != nil {
		return nil, errNotFound("Bank Sampah tidak ditemukan")
	}

	admins, err := s.repo.ListAdminWithUserByBank(bankID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data staff")
	}

	var listAdmins []DetailBankAdminItem
	for _, a := range admins {
		listAdmins = append(listAdmins, DetailBankAdminItem{
			AdminID:   a.AdminID,
			NamaAdmin: a.User.Nama,
			PhotoURL:  a.User.PhotoURL,
			RoleAdmin: roleLabel(a.Role),
		})
	}

	jenisBankStr := string(bank.JenisBank)
	isBSU := false
	switch bank.JenisBank {
	case models.BSU:
		jenisBankStr = "Bank Sampah Unit (BSU)"
		isBSU = true
	case models.BSM:
		jenisBankStr = "Bank Sampah Mandiri (BSM)"
	case models.BSI:
		jenisBankStr = "Bank Sampah Induk (BSI)"
	}

	bankIndukNama := ""
	if bank.ParentBank != nil {
		bankIndukNama = bank.ParentBank.NamaBank
	}
	kecamatanStr := ""
	if bank.Kecamatan != nil {
		kecamatanStr = bank.Kecamatan.Kecamatan
	}
	kelurahanStr := ""
	if bank.Kelurahan != nil {
		kelurahanStr = bank.Kelurahan.Kelurahan
	}

	return &DetailBankItem{
		BankID:        bank.BankID,
		NamaBank:      bank.NamaBank,
		PhotoURL:      bank.PhotoURL,
		IsBSU:         isBSU,
		BankIndukNama: bankIndukNama,
		JenisBank:     jenisBankStr,
		Alamat:        bank.Alamat,
		Provinsi:      bank.Provinsi,
		KabupatenKota: bank.KabupatenKota,
		Kecamatan:     kecamatanStr,
		Kelurahan:     kelurahanStr,
		Deskripsi:     bank.Deskripsi,
		IsActive:      bank.IsActive,
		JoinedAt:      bank.CreatedAt,
		Latitude:      bank.Latitude,
		Longitude:     bank.Longitude,
		Admins:        listAdmins,
	}, nil
}

func (s *bankService) GetHistoryAkunBank(bankID string) ([]repositories.HistoryBankRow, error) {
	rows, err := s.repo.GetHistoryAkunBank(bankID)
	if err != nil {
		return nil, errInternal("Failed to get history: " + err.Error())
	}
	return rows, nil
}

func (s *bankService) DeleteBank(bankID string) error {
	bank, err := s.repo.FindBank(bankID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errNotFound("Bank sampah tidak ditemukan")
		}
		return errInternal("Failed to get bank sampah: " + err.Error())
	}

	if err := s.repo.DeleteBank(bank); err != nil {
		if errors.Is(err, repositories.ErrBankMasihDipakai) {
			return errConflict("Bank sampah tidak dapat dihapus karena masih memiliki data terkait", nil)
		}
		return errInternal("Failed to delete bank sampah: " + err.Error())
	}
	return nil
}

func (s *bankService) ListAdminBank(bankID string) ([]repositories.AdminBankRow, error) {
	rows, err := s.repo.ListAdminBank(bankID)
	if err != nil {
		return nil, errInternal("Failed to get admin: " + err.Error())
	}
	return rows, nil
}

func (s *bankService) AddAdminBank(in AddAdminBankInput) (*models.Admin, error) {
	role := models.RoleAdmin(in.Role)
	if !validAdminRoles[role] {
		return nil, errBadRequest("Role tidak valid")
	}

	existingUser, err := s.repo.FindUser(in.UserID)
	if err != nil {
		return nil, errNotFound("User tidak ditemukan")
	}

	bank, err := s.repo.FindBank(in.BankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}

	roleTitle := roleLabel(role)
	actorName := s.repo.FindNamaAdmin(in.ActorAdminID)
	if actorName == "" {
		actorName = in.ActorAdminID
	}

	var newAdmin *models.Admin
	var otp string
	var expiredAt time.Time

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		countAdmin, err := s.repo.CountAdminAktifPending(tx, in.UserID)
		if err != nil {
			return errInternal("Gagal memvalidasi data admin")
		}
		if countAdmin > 0 {
			return errConflict("User sudah memiliki akun admin aktif. Nonaktifkan akun admin yang ada terlebih dahulu.", nil)
		}

		existingAdmin, err := s.repo.FindAdminByUserAndBankTx(tx, in.UserID, in.BankID)
		if err == nil {
			if existingAdmin.StatusAdmin == models.Nonaktif {
				return errConflict("User pernah terdaftar sebagai admin di bank sampah ini. Gunakan fitur aktivasi untuk mengaktifkan kembali.", nil)
			}
			return errConflict("User sudah terdaftar sebagai admin di bank sampah ini.", nil)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return errInternal("Gagal memvalidasi data admin: " + err.Error())
		}

		adminID, err := idgen.GenerateAdminID(tx, role, in.BankID)
		if err != nil {
			return errInternal("Failed to generate admin ID: " + err.Error())
		}

		bankID := in.BankID
		newAdmin = &models.Admin{
			AdminID:     adminID,
			UserID:      in.UserID,
			BankID:      &bankID,
			Role:        role,
			StatusAdmin: models.Pending,
		}
		if err := s.repo.CreateAdmin(tx, newAdmin); err != nil {
			return errInternal("Failed to create admin: " + err.Error())
		}

		tujuan := models.TujuanAktivasi
		if err := s.repo.DeleteAktivasiAdminBelumDipakai(tx, in.UserID, &tujuan); err != nil {
			return errInternal("Failed to clean up old activation: " + err.Error())
		}

		otp, expiredAt, err = createAktivasiToken(tx, in.UserID, models.RoleUserAdmin, nil, &newAdmin.AdminID, in.ActorAdminID)
		if err != nil {
			return errInternal("Failed to create aktivasi akun: " + err.Error())
		}

		if err := s.writeHistory(tx, in.BankID, "CREATE", nil, *newAdmin,
			fmt.Sprintf("Staff %s (%s) ditambahkan ke %s", existingUser.Nama, roleTitle, bank.NamaBank),
			fmt.Sprintf("%s menambahkan %s (%s) ke %s", actorName, existingUser.Nama, roleTitle, bank.NamaBank),
			in.ActorAdminID); err != nil {
			return errInternal("Failed to create audit history: " + err.Error())
		}

		return nil
	})

	if txErr != nil {
		if _, ok := txErr.(*ServiceError); ok {
			return nil, txErr
		}
		return nil, errInternal("Failed to commit transaction")
	}

	emailBody := mailer.BuildAktivasiAdminEmail(existingUser.Nama, roleTitle, bank.NamaBank, "", otp, expiredAt)
	go func() {
		if err := s.mailer.SendEmail(mailer.Params{
			To:      existingUser.Email,
			Subject: fmt.Sprintf("Aktivasi Akun %s Enviroo", roleTitle),
			Body:    emailBody,
		}); err != nil {
			log.Printf("[Email] Gagal kirim aktivasi ke %s: %v", existingUser.Email, err)
		}
	}()

	return newAdmin, nil
}

func (s *bankService) DeleteStaffBank(adminID, actorUserID string) (*DeleteStaffResult, error) {
	admin, err := s.repo.FindAdminWithUserBank(adminID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errNotFound("Staff not found")
		}
		return nil, errInternal("Failed to fetch staff data: " + err.Error())
	}
	if admin.BankID == nil {
		return nil, errBadRequest("Staff tidak terhubung dengan bank sampah")
	}

	actor, err := s.repo.FindAdminAktifByUser(actorUserID)
	if err != nil {
		return nil, errForbidden("Akun Anda tidak memiliki akses untuk menghapus staff")
	}
	if actor.AdminID == admin.AdminID {
		return nil, errForbidden("Tidak dapat menghapus akun sendiri")
	}
	if actor.BankID == nil || *actor.BankID != *admin.BankID {
		return nil, errForbidden("Anda hanya dapat menghapus staff di bank sampah Anda sendiri")
	}
	if admin.StatusAdmin == models.Nonaktif {
		return nil, errBadRequest("Staff sudah nonaktif")
	}
	deletedBy := actor.AdminID

	staffName := admin.User.Nama
	if staffName == "" {
		if user, err := s.repo.FindUser(admin.UserID); err == nil && user.Nama != "" {
			staffName = user.Nama
		} else {
			staffName = admin.UserID
		}
	}

	bankName := "Unknown Bank"
	if admin.Bank.NamaBank != "" {
		bankName = admin.Bank.NamaBank
	} else if bank, err := s.repo.FindBank(*admin.BankID); err == nil && bank.NamaBank != "" {
		bankName = bank.NamaBank
	}

	actorName := s.repo.FindNamaAdmin(deletedBy)
	if actorName == "" {
		actorName = deletedBy
	}

	snapshot := struct {
		AdminID     string            `json:"admin_id"`
		BankID      *string           `json:"bank_id"`
		UserID      string            `json:"user_id"`
		Nama        string            `json:"nama"`
		Role        models.RoleAdmin  `json:"role"`
		StatusAdmin models.StatusAkun `json:"status_admin"`
	}{
		AdminID:     admin.AdminID,
		BankID:      admin.BankID,
		UserID:      admin.UserID,
		Nama:        staffName,
		Role:        admin.Role,
		StatusAdmin: admin.StatusAdmin,
	}

	isPending := admin.StatusAdmin == models.Pending
	staffRole := roleLabel(admin.Role)

	// Pending bisa berarti akun baru atau akun lama yang sedang direaktivasi.
	// Akun yang pernah aktif tidak boleh dihapus permanen: reaktivasinya dibatalkan saja.
	hapusPermanen := false
	if isPending {
		pernahAktif, err := s.repo.AdminPernahAktif(adminID)
		if err != nil {
			return nil, errInternal("Failed to check activation history: " + err.Error())
		}
		hapusPermanen = !pernahAktif
	}

	informasi := fmt.Sprintf("Staff %s (%s) dihapus dari %s", staffName, staffRole, bankName)
	keterangan := fmt.Sprintf("%s menghapus %s (%s) dari %s", actorName, staffName, staffRole, bankName)
	if isPending && !hapusPermanen {
		informasi = fmt.Sprintf("Reaktivasi staff %s (%s) di %s dibatalkan", staffName, staffRole, bankName)
		keterangan = fmt.Sprintf("%s membatalkan reaktivasi %s (%s) di %s", actorName, staffName, staffRole, bankName)
	}

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		if isPending {
			if err := s.repo.DeleteAktivasiAdminBelumDipakai(tx, admin.UserID, nil); err != nil {
				return errInternal("Failed to clean up activation: " + err.Error())
			}
		}

		if err := s.writeHistory(tx, *admin.BankID, "DELETE", snapshot, nil, informasi, keterangan, deletedBy); err != nil {
			return errInternal("Failed to create audit history: " + err.Error())
		}

		if hapusPermanen {
			if err := s.repo.DeleteAdmin(tx, adminID); err != nil {
				return errInternal("Failed to delete staff: " + err.Error())
			}
			return nil
		}
		if err := s.repo.UpdateStatusAdmin(tx, adminID, models.Nonaktif); err != nil {
			return errInternal("Failed to deactivate staff: " + err.Error())
		}
		return nil
	})

	if txErr != nil {
		if _, ok := txErr.(*ServiceError); ok {
			return nil, txErr
		}
		return nil, errInternal("Failed to commit transaction: " + txErr.Error())
	}

	if !isPending {
		staffEmail := admin.User.Email
		if staffEmail == "" {
			if u, err := s.repo.FindUser(admin.UserID); err == nil {
				staffEmail = u.Email
			}
		}
		if staffEmail != "" {
			emailBody := fmt.Sprintf(`
				<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
					<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
					<p style="font-size: 14px; color: #333; line-height: 1.5;">
						Akun <b>%s</b> Anda di <b>%s</b> telah dinonaktifkan oleh administrator.
						Jika Anda merasa ini adalah kesalahan, silakan hubungi administrator terkait.
					</p>
					<hr style="border: 0; height: 1px; background: #ddd; margin: 25px 0;">
					<p style="font-size: 12px; color: #999; text-align: center; margin: 0;">&copy; Enviroo APP</p>
				</div>
			`, staffName, admin.Role, bankName)
			go func() {
				_ = s.mailer.SendEmail(mailer.Params{
					To:      staffEmail,
					Subject: "Notifikasi Penonaktifan Akun Staff Enviroo",
					Body:    emailBody,
				})
			}()
		}
	}

	return &DeleteStaffResult{StaffName: staffName, BankName: bankName}, nil
}

func roleLabel(role models.RoleAdmin) string {
	switch role {
	case models.AdminBSI, models.AdminBSM, models.AdminBSU:
		return "Admin"
	case models.PetugasBSI, models.PetugasBSM, models.PetugasBSU:
		return "Petugas"
	case models.SuperAdmin:
		return "Superadmin"
	default:
		return string(role)
	}
}
