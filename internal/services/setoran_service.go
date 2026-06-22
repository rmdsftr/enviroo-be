package services

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/pkg/utils"

	"gorm.io/gorm"
)

// ── Repo interface (diimplementasi oleh repositories.SetoranRepo) ──────────

type SetoranRepoIface interface {
	FindNasabahAktifWithUser(nasabahID string) (*models.Nasabah, error)
	FindNasabahWithUser(nasabahID string) (*models.Nasabah, error)
	FindAdminAktif(adminID string) (*models.Admin, error)
	FindPenimbanganAktif(penimbanganID string) (*models.Penimbangan, error)
	FindKatalogSampah(sampahID string) (*models.KatalogSampah, error)
	FindBankSampah(bankID string) (*models.BankSampah, error)
	GetDetailHeader(setoranID string) (*repositories.SetoranDetailHeader, error)
	GetDetailItems(setoranID string) ([]repositories.SetoranDetailItem, error)
	GetRiwayat(nasabahID, startDate, endDate string) ([]repositories.RiwayatSetoranRow, error)
	// Transactional — caller menyediakan *gorm.DB dari db.Transaction
	FindPenimbanganTx(tx *gorm.DB, penimbanganID string) (*models.Penimbangan, error)
	CreateSetoran(tx *gorm.DB, s *models.SetoranNasabah) error
	CreateDetailSetoran(tx *gorm.DB, d *models.DetailSetoranNasabah) error
	UpsertStokSampah(tx *gorm.DB, bankID, sampahID string, qty float64) error
	CreateTabunganSampah(tx *gorm.DB, t *models.TabunganSampah) error
}

// ── Request / Response types ───────────────────────────────────────────────

type ItemSetoranReq struct {
	SampahID string  `json:"sampah_id"`
	Qty      float64 `json:"qty"`
}

type VerifikasiResult struct {
	Status  string      `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type SetoranPreviewItem struct {
	SampahID    string  `json:"sampah_id"`
	NamaSampah  string  `json:"nama_sampah"`
	JenisReward string  `json:"jenis_reward"`
	Qty         float64 `json:"qty"`
	Satuan      string  `json:"satuan"`
}

type PreviewSetoranResult struct {
	PenimbanganID string               `json:"penimbangan_id"`
	NasabahID     string               `json:"nasabah_id"`
	NamaNasabah   string               `json:"nama_nasabah"`
	TotalItem     int                  `json:"total_item"`
	Items         []SetoranPreviewItem `json:"items"`
}

type InputSetoranResult struct {
	SetoranID string `json:"setoran_id"`
	TotalItem int    `json:"total_item"`
}

type SetoranDetail struct {
	Header repositories.SetoranDetailHeader  `json:"header"`
	Items  []repositories.SetoranDetailItem  `json:"items"`
}

// ── Service interface ──────────────────────────────────────────────────────

type SetoranService interface {
	// Verifikasi selalu mengembalikan result, tidak pernah error — caller menulis 200.
	Verifikasi(qrData, adminID string) *VerifikasiResult
	Preview(penimbanganID, nasabahID string, items []ItemSetoranReq) (*PreviewSetoranResult, error)
	Input(penimbanganID, nasabahID, adminID, via, buktiURL string, items []ItemSetoranReq) (*InputSetoranResult, error)
	GetDetail(setoranID string) (*SetoranDetail, error)
	GetRiwayat(nasabahID, startDate, endDate string) ([]repositories.RiwayatSetoranRow, error)
}

// ── Implementation ─────────────────────────────────────────────────────────

type setoranService struct {
	db       *gorm.DB
	repo     SetoranRepoIface
	notifSvc NotifikasiService
}

func NewSetoranService(db *gorm.DB, repo SetoranRepoIface, notifSvc NotifikasiService) SetoranService {
	return &setoranService{db: db, repo: repo, notifSvc: notifSvc}
}

func (s *setoranService) Verifikasi(qrData, adminID string) *VerifikasiResult {
	var payload struct {
		Type          string `json:"type"`
		NasabahID     string `json:"nasabah_id"`
		PenimbanganID string `json:"penimbangan_id"`
	}
	if err := json.Unmarshal([]byte(qrData), &payload); err != nil {
		return &VerifikasiResult{Status: "unverified", Message: "Format QR code tidak valid"}
	}
	if payload.Type != "ENVIROO-SETORAN" {
		return &VerifikasiResult{Status: "unverified", Message: "QR code bukan milik aplikasi ini"}
	}

	nasabah, err := s.repo.FindNasabahAktifWithUser(payload.NasabahID)
	if err != nil {
		return &VerifikasiResult{Status: "unverified", Message: "Nasabah tidak aktif atau tidak ditemukan"}
	}

	admin, err := s.repo.FindAdminAktif(adminID)
	if err != nil {
		return &VerifikasiResult{Status: "unverified", Message: "Admin tidak aktif atau tidak ditemukan"}
	}

	if nasabah.UserID == admin.UserID {
		return &VerifikasiResult{Status: "unverified", Message: "Petugas tidak boleh mengisi setoran sampah sebagai nasabah sendiri"}
	}

	penimbangan, err := s.repo.FindPenimbanganAktif(payload.PenimbanganID)
	if err != nil {
		return &VerifikasiResult{Status: "unverified", Message: "Sesi penimbangan tidak aktif atau tidak ditemukan"}
	}

	if penimbangan.BankID == nil || nasabah.BankID != *penimbangan.BankID {
		return &VerifikasiResult{Status: "unverified", Message: "Nasabah tidak terdaftar di bank sampah ini"}
	}

	return &VerifikasiResult{
		Status:  "verified",
		Message: "Semua pihak terverifikasi, setoran dapat dilanjutkan",
		Data: map[string]interface{}{
			"nasabah_id":   nasabah.NasabahID,
			"nama_nasabah": nasabah.User.Nama,
			"photo_url":    nasabah.User.PhotoURL,
		},
	}
}

func (s *setoranService) Preview(penimbanganID, nasabahID string, items []ItemSetoranReq) (*PreviewSetoranResult, error) {
	nasabah, err := s.repo.FindNasabahWithUser(nasabahID)
	if err != nil {
		return nil, errNotFound("Nasabah tidak ditemukan")
	}

	var previews []SetoranPreviewItem
	for _, item := range items {
		if item.Qty <= 0 {
			return nil, errBadRequest("Qty harus lebih dari 0")
		}
		katalog, err := s.repo.FindKatalogSampah(item.SampahID)
		if err != nil {
			return nil, errBadRequest("Sampah tidak ditemukan: " + item.SampahID)
		}
		previews = append(previews, SetoranPreviewItem{
			SampahID:    item.SampahID,
			NamaSampah:  katalog.Sarok.NamaSampah,
			JenisReward: string(katalog.Reward.NamaReward),
			Qty:         item.Qty,
			Satuan:      string(katalog.Sarok.Satuan),
		})
	}

	return &PreviewSetoranResult{
		PenimbanganID: penimbanganID,
		NasabahID:     nasabahID,
		NamaNasabah:   nasabah.User.Nama,
		TotalItem:     len(items),
		Items:         previews,
	}, nil
}

func (s *setoranService) Input(penimbanganID, nasabahID, adminID, via, buktiURL string, items []ItemSetoranReq) (*InputSetoranResult, error) {
	nasabah, err := s.repo.FindNasabahAktifWithUser(nasabahID)
	if err != nil {
		return nil, errForbidden("Nasabah tidak aktif atau tidak ditemukan")
	}
	admin, err := s.repo.FindAdminAktif(adminID)
	if err != nil {
		return nil, errForbidden("Admin tidak aktif atau tidak ditemukan")
	}
	if nasabah.UserID == admin.UserID {
		return nil, errForbidden("Petugas tidak boleh mengisi setoran sampah sebagai nasabah sendiri")
	}

	setoranID := utils.GenerateID("STR")
	totalItem := len(items)
	var bankID string

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		penimbangan, err := s.repo.FindPenimbanganTx(tx, penimbanganID)
		if err != nil {
			return errNotFound("Sesi penimbangan tidak aktif atau tidak ditemukan")
		}
		if penimbangan.BankID == nil {
			return errInternal("Penimbangan tidak memiliki bank")
		}
		bankID = *penimbangan.BankID

		if nasabah.BankID != bankID {
			return errForbidden("Nasabah tidak terdaftar di bank sampah ini")
		}

		setoran := &models.SetoranNasabah{
			SetoranID:      setoranID,
			AdminID:        adminID,
			NasabahID:      nasabahID,
			PenimbanganID:  penimbanganID,
			TotalItem:      totalItem,
			StatusSetoran:  models.StatusBerhasil,
			BuktiViaManual: buktiURL,
		}
		if err := s.repo.CreateSetoran(tx, setoran); err != nil {
			return errInternal("Gagal menyimpan setoran")
		}

		for _, item := range items {
			detail := &models.DetailSetoranNasabah{
				SetoranID: setoranID,
				SampahID:  item.SampahID,
				Qty:       item.Qty,
			}
			if err := s.repo.CreateDetailSetoran(tx, detail); err != nil {
				return errInternal("Gagal menyimpan detail setoran")
			}

			if err := s.repo.UpsertStokSampah(tx, bankID, item.SampahID, item.Qty); err != nil {
				return errInternal("Gagal mengupdate stok sampah")
			}

			tabunganID := utils.GenerateID("TBG")
			tabungan := &models.TabunganSampah{
				TabunganID: tabunganID,
				NasabahID:  &nasabahID,
				BankID:     &bankID,
				SampahID:   item.SampahID,
				Entitas:    models.EntitasNasabah,
				Qty:        item.Qty,
				SisaQty:    item.Qty,
				CreatedAt:  time.Now(),
				SourceID:   &setoranID,
			}
			if err := s.repo.CreateTabunganSampah(tx, tabungan); err != nil {
				return errInternal("Gagal menyimpan tabungan sampah")
			}
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	go func() {
		bank, err := s.repo.FindBankSampah(bankID)
		if err != nil {
			log.Printf("[Notif] Gagal ambil bank %s: %v", bankID, err)
			return
		}
		nasabahFull, err := s.repo.FindNasabahWithUser(nasabahID)
		if err != nil {
			log.Printf("[Notif] Gagal ambil nasabah %s: %v", nasabahID, err)
			return
		}
		if err := s.notifSvc.NotifSetoranBerhasil(
			context.Background(),
			nasabahFull.User.UserID,
			nasabahFull.User.FCMToken,
			totalItem,
			bank.NamaBank,
			setoranID,
		); err != nil {
			log.Printf("[Notif] Gagal kirim notif setoran %s: %v", setoranID, err)
		}
	}()

	return &InputSetoranResult{SetoranID: setoranID, TotalItem: totalItem}, nil
}

func (s *setoranService) GetDetail(setoranID string) (*SetoranDetail, error) {
	header, err := s.repo.GetDetailHeader(setoranID)
	if err != nil {
		return nil, errNotFound("Data setoran tidak ditemukan")
	}
	items, err := s.repo.GetDetailItems(setoranID)
	if err != nil {
		return nil, errInternal("Gagal mengambil detail item setoran")
	}
	return &SetoranDetail{Header: *header, Items: items}, nil
}

func (s *setoranService) GetRiwayat(nasabahID, startDate, endDate string) ([]repositories.RiwayatSetoranRow, error) {
	rows, err := s.repo.GetRiwayat(nasabahID, startDate, endDate)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat setoran")
	}
	return rows, nil
}
