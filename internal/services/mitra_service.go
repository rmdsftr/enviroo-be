package services

import (
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"errors"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

type MitraRepoIface interface {
	GetBankByID(bankID string) (*models.BankSampah, error)
	ListByBank(bankID string) ([]models.MitraPengepul, error)
	GetByID(db *gorm.DB, bankID, mitraID string) (*models.MitraPengepul, error)
	FindByNormalizedName(db *gorm.DB, bankID, normalizedName string) (*models.MitraPengepul, error)
	Create(db *gorm.DB, m *models.MitraPengepul) error
	Update(db *gorm.DB, m *models.MitraPengepul) error
	IsUsedInPenjualan(db *gorm.DB, mitraID string) (bool, error)
	Delete(db *gorm.DB, m *models.MitraPengepul) error
}

type MitraService struct {
	db   *gorm.DB
	repo MitraRepoIface
}

func NewMitraService(db *gorm.DB, repo MitraRepoIface) *MitraService {
	return &MitraService{db: db, repo: repo}
}

var normalizeMitraRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func normalizeMitraName(nama string) string {
	return normalizeMitraRegex.ReplaceAllString(strings.ToLower(nama), "")
}

type MitraResp struct {
	MitraID   string    `json:"mitra_id"`
	NamaMitra string    `json:"nama_mitra"`
	BankID    string    `json:"bank_id"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

func toMitraResp(m *models.MitraPengepul) MitraResp {
	return MitraResp{
		MitraID:   m.MitraID,
		NamaMitra: m.NamaMitra,
		BankID:    m.BankID,
		IsActive:  m.IsActive,
		CreatedAt: m.CreatedAt,
	}
}

func validateBankUntukMitra(bank *models.BankSampah, err error) (*models.BankSampah, error) {
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank != models.BSI && bank.JenisBank != models.BSM {
		return nil, errForbidden("Fitur ini hanya untuk BSI dan BSM")
	}
	return bank, nil
}

func (s *MitraService) ListMitra(bankID string) ([]MitraResp, error) {
	if _, err := validateBankUntukMitra(s.repo.GetBankByID(bankID)); err != nil {
		return nil, err
	}

	mitra, err := s.repo.ListByBank(bankID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data mitra: " + err.Error())
	}

	resp := make([]MitraResp, 0, len(mitra))
	for i := range mitra {
		resp = append(resp, toMitraResp(&mitra[i]))
	}
	return resp, nil
}

func (s *MitraService) CreateMitra(bankID, nama string) (*MitraResp, error) {
	if _, err := validateBankUntukMitra(s.repo.GetBankByID(bankID)); err != nil {
		return nil, err
	}

	nama = strings.TrimSpace(nama)
	if nama == "" {
		return nil, errBadRequest("Nama mitra wajib diisi")
	}

	normalized := normalizeMitraName(nama)
	if existing, err := s.repo.FindByNormalizedName(s.db, bankID, normalized); err == nil {
		return nil, errBadRequest("Mitra dengan nama serupa sudah terdaftar: " + existing.NamaMitra)
	}

	var mitra *models.MitraPengepul
	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		mitraID, err := idgen.GenerateMitraID(tx, bankID)
		if err != nil {
			return errInternal("Gagal generate mitra ID: " + err.Error())
		}
		mitra = &models.MitraPengepul{
			MitraID:   mitraID,
			NamaMitra: nama,
			BankID:    bankID,
			IsActive:  true,
		}
		if err := s.repo.Create(tx, mitra); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return errBadRequest("Mitra dengan nama serupa sudah terdaftar")
			}
			return errInternal("Gagal membuat data mitra: " + err.Error())
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	resp := toMitraResp(mitra)
	return &resp, nil
}

func (s *MitraService) UpdateMitra(bankID, mitraID string, nama *string, isActive *bool) (*MitraResp, error) {
	if _, err := validateBankUntukMitra(s.repo.GetBankByID(bankID)); err != nil {
		return nil, err
	}

	mitra, err := s.repo.GetByID(s.db, bankID, mitraID)
	if err != nil {
		return nil, errNotFound("Data mitra tidak ditemukan")
	}

	if nama != nil {
		trimmed := strings.TrimSpace(*nama)
		if trimmed == "" {
			return nil, errBadRequest("Nama mitra tidak boleh kosong")
		}
		normalized := normalizeMitraName(trimmed)
		if existing, err := s.repo.FindByNormalizedName(s.db, bankID, normalized); err == nil && existing.MitraID != mitra.MitraID {
			return nil, errBadRequest("Mitra dengan nama serupa sudah terdaftar: " + existing.NamaMitra)
		}
		mitra.NamaMitra = trimmed
	}
	if isActive != nil {
		mitra.IsActive = *isActive
	}

	if err := s.repo.Update(s.db, mitra); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errBadRequest("Mitra dengan nama serupa sudah terdaftar")
		}
		return nil, errInternal("Gagal mengupdate data mitra: " + err.Error())
	}
	resp := toMitraResp(mitra)
	return &resp, nil
}

func (s *MitraService) DeleteMitra(bankID, mitraID string) error {
	if _, err := validateBankUntukMitra(s.repo.GetBankByID(bankID)); err != nil {
		return err
	}

	mitra, err := s.repo.GetByID(s.db, bankID, mitraID)
	if err != nil {
		return errNotFound("Data mitra tidak ditemukan")
	}

	used, err := s.repo.IsUsedInPenjualan(s.db, mitraID)
	if err != nil {
		return errInternal("Gagal mengecek histori penjualan mitra: " + err.Error())
	}

	if used {
		mitra.IsActive = false
		if err := s.repo.Update(s.db, mitra); err != nil {
			return errInternal("Gagal menghapus data mitra: " + err.Error())
		}
		return nil
	}

	if err := s.repo.Delete(s.db, mitra); err != nil {
		return errInternal("Gagal menghapus data mitra: " + err.Error())
	}
	return nil
}

func resolveOrCreateMitra(tx *gorm.DB, repo MitraRepoIface, bankID, mitraID, namaMitra string) (*models.MitraPengepul, error) {
	if mitraID != "" {
		mitra, err := repo.GetByID(tx, bankID, mitraID)
		if err != nil {
			return nil, errBadRequest("Mitra tidak ditemukan")
		}
		return mitra, nil
	}

	namaMitra = strings.TrimSpace(namaMitra)
	if namaMitra == "" {
		return nil, errBadRequest("mitra_id atau nama_mitra wajib diisi")
	}

	normalized := normalizeMitraName(namaMitra)
	if existing, err := repo.FindByNormalizedName(tx, bankID, normalized); err == nil {
		return existing, nil
	}

	mitraID, err := idgen.GenerateMitraID(tx, bankID)
	if err != nil {
		return nil, errInternal("Gagal generate mitra ID: " + err.Error())
	}
	mitra := &models.MitraPengepul{
		MitraID:   mitraID,
		NamaMitra: namaMitra,
		BankID:    bankID,
		IsActive:  true,
	}
	if err := repo.Create(tx, mitra); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			if existing, findErr := repo.FindByNormalizedName(tx, bankID, normalized); findErr == nil {
				return existing, nil
			}
		}
		return nil, errInternal("Gagal membuat data mitra baru: " + err.Error())
	}
	return mitra, nil
}
