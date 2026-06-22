package services

import (
	"context"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/pkg/utils"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// ── Repo interface yang dibutuhkan service ─────────────────────────────────

type PenarikanRepoIface interface {
	FindNasabahAktif(nasabahID string) (*models.Nasabah, error)
	FindNasabahByUserID(userID string) (*models.Nasabah, error)
	FindReward(rewardID int) (*models.Reward, error)
	FindSaldoRekening(nasabahID string, rewardID int) (*models.SaldoRekening, error)
	FindSembako(sembakoID, bankID string) (*models.KatalogSembako, error)
	FindStokSembako(sembakoID, bankID string) (*models.StokSembako, error)
	FindPenarikanByID(penarikanID string) (*models.Penarikan, error)
	FindByNasabahID(nasabahID string, f repositories.PenarikanFilter) ([]models.Penarikan, error)
	FindByBankID(bankID string, f repositories.PenarikanFilter) ([]models.Penarikan, error)
	FindAdminsByBankID(bankID string) ([]repositories.AdminInfo, error)

	LockSaldo(tx *gorm.DB, nasabahID string, rewardID int) (*models.SaldoRekening, error)
	LockSembako(tx *gorm.DB, sembakoID, bankID string) (*models.KatalogSembako, error)
	LockStok(tx *gorm.DB, sembakoID, bankID string) (*models.StokSembako, error)
	LockPenarikan(tx *gorm.DB, penarikanID string) (*models.Penarikan, error)
	FindAdminTx(tx *gorm.DB, userID string) (*models.Admin, error)
	FindDetailSembako(tx *gorm.DB, penarikanID string) ([]models.DetailPenarikanSembako, error)
	UpdateSaldo(tx *gorm.DB, saldo *models.SaldoRekening, nominal float64) error
	CreateRiwayat(tx *gorm.DB, rw *models.RiwayatArusSaldo) error
	SavePenarikan(tx *gorm.DB, p *models.Penarikan) error
	SaveDetailSembako(tx *gorm.DB, d *models.DetailPenarikanSembako) error
	UpdateStok(tx *gorm.DB, sembakoID, bankID string, delta float64) error
	UpdateStatus(tx *gorm.DB, p *models.Penarikan, updates map[string]interface{}) error
}

// ── Request / Response types ───────────────────────────────────────────────

type SembakoItemReq struct {
	SembakoID string  `json:"sembako_id" binding:"required"`
	Qty       float64 `json:"qty"        binding:"required,gt=0"`
}

type AjukanPenarikanReq struct {
	RewardID         int              `json:"reward_id"          binding:"required"`
	NominalPenarikan float64          `json:"nominal_penarikan"`
	ItemSembako      []SembakoItemReq `json:"item_sembako"`
}

type PreviewSembakoItem struct {
	SembakoID    string  `json:"sembako_id"`
	NamaSembako  string  `json:"nama_sembako"`
	Qty          float64 `json:"qty"`
	NilaiPoin    float64 `json:"nilai_poin"`
	SubtotalPoin float64 `json:"subtotal_poin"`
}

type PreviewResult struct {
	NasabahID        string                  `json:"nasabah_id"`
	NamaReward       models.RewardEnum       `json:"nama_reward"`
	Satuan           string                  `json:"satuan"`
	SaldoSekarang    float64                 `json:"saldo_sekarang"`
	SatuanSaldo      models.SatuanRewardEnum `json:"satuan_saldo"`
	NominalPenarikan float64                 `json:"nominal_penarikan"`
	SaldoSetelah     float64                 `json:"saldo_setelah"`
	SaldoCukup       bool                    `json:"saldo_cukup"`
	ItemSembako      []PreviewSembakoItem    `json:"item_sembako,omitempty"`
}

type AjukanResult struct {
	PenarikanID string                     `json:"penarikan_id"`
	Nominal     float64                    `json:"nominal"`
	Status      models.StatusPenarikanEnum `json:"status"`
}

type PenarikanSummary struct {
	PenarikanID      string                     `json:"penarikan_id"`
	NasabahID        *string                    `json:"nasabah_id"`
	NamaNasabah      string                     `json:"nama_nasabah"`
	BankID           *string                    `json:"bank_id"`
	NamaBank         string                     `json:"nama_bank"`
	RewardID         *int                       `json:"reward_id"`
	NamaReward       string                     `json:"nama_reward"`
	NominalPenarikan float64                    `json:"nominal_penarikan"`
	SatuanPenarikan  models.SatuanRewardEnum    `json:"satuan_penarikan"`
	StatusPenarikan  models.StatusPenarikanEnum `json:"status_penarikan"`
	KadaluarsaAt     *time.Time                 `json:"kadaluarsa_at"`
	BuktiFoto        *string                    `json:"bukti_foto"`
	CreatedAt        time.Time                  `json:"created_at"`
	UpdatedAt        time.Time                  `json:"updated_at"`
}

type DetailSembakoItem struct {
	SembakoID    string  `json:"sembako_id"`
	NamaSembako  string  `json:"nama_sembako"`
	PhotoURL     *string `json:"photo_url"`
	Qty          float64 `json:"qty"`
	NilaiPoin    float64 `json:"nilai_poin"`
	SubtotalPoin float64 `json:"subtotal_poin"`
}

type PenarikanDetail struct {
	PenarikanSummary
	DetailSembako []DetailSembakoItem `json:"detail_sembako,omitempty"`
}

type ListPenarikanResult struct {
	Data []PenarikanSummary `json:"data"`
}

// ── Service interface ──────────────────────────────────────────────────────

type PenarikanService interface {
	Preview(nasabahID string, req AjukanPenarikanReq) (*PreviewResult, error)
	Ajukan(nasabahID string, req AjukanPenarikanReq) (*AjukanResult, error)
	Konfirmasi(penarikanID, adminUserID, fotoURL string) error
	Batal(penarikanID, nasabahUserID string) error
	GetList(nasabahID string, f repositories.PenarikanFilter) (*ListPenarikanResult, error)
	GetListByBank(bankID string, f repositories.PenarikanFilter) ([]PenarikanSummary, error)
	GetDetail(penarikanID, userID string, role models.RoleAdmin) (*PenarikanDetail, error)
}

// ── Implementation ─────────────────────────────────────────────────────────

type penarikanService struct {
	db       *gorm.DB
	repo     PenarikanRepoIface
	notifSvc NotifikasiService
}

func NewPenarikanService(db *gorm.DB, repo PenarikanRepoIface, notifSvc NotifikasiService) PenarikanService {
	return &penarikanService{db: db, repo: repo, notifSvc: notifSvc}
}

// ── Preview ────────────────────────────────────────────────────────────────

func (s *penarikanService) Preview(nasabahID string, req AjukanPenarikanReq) (*PreviewResult, error) {
	nasabah, err := s.repo.FindNasabahAktif(nasabahID)
	if err != nil {
		return nil, errNotFound("Nasabah tidak ditemukan atau tidak aktif")
	}

	reward, err := s.repo.FindReward(req.RewardID)
	if err != nil {
		return nil, errNotFound("Jenis reward tidak ditemukan")
	}

	if err := validatePenarikanReq(reward, req); err != nil {
		return nil, err
	}

	saldo, err := s.repo.FindSaldoRekening(nasabahID, req.RewardID)
	if err != nil {
		return nil, errNotFound("Rekening saldo untuk reward ini tidak ditemukan")
	}

	finalNominal := req.NominalPenarikan
	var itemsSembako []PreviewSembakoItem

	if reward.NamaReward == models.RewardEnumSembako {
		var totalPoin float64
		for _, item := range req.ItemSembako {
			sembako, err := s.repo.FindSembako(item.SembakoID, nasabah.BankID)
			if err != nil {
				return nil, errNotFound("Barang sembako tidak ditemukan: " + item.SembakoID)
			}
			stok, err := s.repo.FindStokSembako(item.SembakoID, nasabah.BankID)
			if err != nil {
				return nil, errNotFound("Stok sembako tidak ditemukan: " + item.SembakoID)
			}
			if stok.Stok < item.Qty {
				return nil, errBadRequest("Stok barang '" + sembako.MasterSembako.NamaBarang + "' tidak mencukupi")
			}
			subtotal := item.Qty * sembako.NilaiPoin
			totalPoin += subtotal
			itemsSembako = append(itemsSembako, PreviewSembakoItem{
				SembakoID:    item.SembakoID,
				NamaSembako:  sembako.MasterSembako.NamaBarang,
				Qty:          item.Qty,
				NilaiPoin:    sembako.NilaiPoin,
				SubtotalPoin: subtotal,
			})
		}
		finalNominal = totalPoin
	}

	saldoSetelah := saldo.NominalSaldo - finalNominal

	return &PreviewResult{
		NasabahID:        nasabahID,
		NamaReward:       reward.NamaReward,
		Satuan:           reward.Satuan,
		SaldoSekarang:    saldo.NominalSaldo,
		SatuanSaldo:      saldo.SatuanNominalSaldo,
		NominalPenarikan: finalNominal,
		SaldoSetelah:     saldoSetelah,
		SaldoCukup:       saldoSetelah >= 0,
		ItemSembako:      itemsSembako,
	}, nil
}

// ── Ajukan ─────────────────────────────────────────────────────────────────

func (s *penarikanService) Ajukan(nasabahID string, req AjukanPenarikanReq) (*AjukanResult, error) {
	nasabah, err := s.repo.FindNasabahAktif(nasabahID)
	if err != nil {
		return nil, errNotFound("Nasabah tidak ditemukan atau tidak aktif")
	}

	reward, err := s.repo.FindReward(req.RewardID)
	if err != nil {
		return nil, errNotFound("Jenis reward tidak ditemukan")
	}

	if err := validatePenarikanReq(reward, req); err != nil {
		return nil, err
	}

	var penarikanID string
	var finalNominal float64

	err = s.db.Transaction(func(tx *gorm.DB) error {
		saldo, err := s.repo.LockSaldo(tx, nasabahID, req.RewardID)
		if err != nil {
			return errNotFound("Rekening saldo untuk reward ini tidak ditemukan")
		}

		nominalTx := req.NominalPenarikan
		var itemsToSave []models.DetailPenarikanSembako

		if reward.NamaReward == models.RewardEnumSembako {
			var totalPoin float64
			for _, item := range req.ItemSembako {
				sembako, err := s.repo.LockSembako(tx, item.SembakoID, nasabah.BankID)
				if err != nil {
					return errNotFound("Barang sembako tidak ditemukan: " + item.SembakoID)
				}
				stok, err := s.repo.LockStok(tx, item.SembakoID, nasabah.BankID)
				if err != nil {
					return errNotFound("Stok sembako tidak ditemukan: " + item.SembakoID)
				}
				if stok.Stok < item.Qty {
					return errBadRequest("Stok barang '" + sembako.MasterSembako.NamaBarang + "' tidak mencukupi")
				}
				subtotal := item.Qty * sembako.NilaiPoin
				totalPoin += subtotal
				itemsToSave = append(itemsToSave, models.DetailPenarikanSembako{
					SembakoID:    item.SembakoID,
					Qty:          item.Qty,
					NilaiPoin:    sembako.NilaiPoin,
					SubtotalPoin: subtotal,
				})
			}
			nominalTx = totalPoin
		}

		if saldo.NominalSaldo < nominalTx {
			return errBadRequest("Saldo tidak mencukupi untuk melakukan penarikan")
		}

		saldoSebelum := saldo.NominalSaldo
		saldoSesudah := saldoSebelum - nominalTx

		if err := s.repo.UpdateSaldo(tx, saldo, saldoSesudah); err != nil {
			return err
		}

		if err := s.repo.CreateRiwayat(tx, &models.RiwayatArusSaldo{
			RiwayatSaldoID: utils.GenerateID("RS"),
			RekeningID:     &saldo.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			CreatedBy:      &nasabahID,
		}); err != nil {
			return err
		}

		penarikanID = utils.GenerateID("RDM")
		now := time.Now()
		kadaluarsa := now.Add(2 * time.Hour)

		p := &models.Penarikan{
			PenarikanID:      penarikanID,
			NasabahID:        &nasabahID,
			BankID:           &nasabah.BankID,
			RewardID:         &req.RewardID,
			NominalPenarikan: nominalTx,
			SatuanPenarikan:  models.SatuanRewardEnum(reward.Satuan),
			StatusPenarikan:  models.StatusPenarikanPending,
			KadaluarsaAt:     &kadaluarsa,
			CreatedAt:        now,
			CreatedBy:        &nasabahID,
		}
		if err := s.repo.SavePenarikan(tx, p); err != nil {
			return err
		}

		for i := range itemsToSave {
			itemsToSave[i].PenarikanID = penarikanID
			if err := s.repo.SaveDetailSembako(tx, &itemsToSave[i]); err != nil {
				return err
			}
			if err := s.repo.UpdateStok(tx, itemsToSave[i].SembakoID, nasabah.BankID, -itemsToSave[i].Qty); err != nil {
				return err
			}
		}

		finalNominal = nominalTx
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Notifikasi ke semua admin bank (fire-and-forget)
	snapshotBankID := nasabah.BankID
	snapshotNasabahID := nasabahID
	snapshotPenarikanID := penarikanID
	snapshotNominal := finalNominal
	snapshotSatuan := models.SatuanRewardEnum(reward.Satuan)

	go func() {
		var nas models.Nasabah
		// Preload user untuk ambil nama nasabah
		if err := s.db.Preload("User").Where("nasabah_id = ?", snapshotNasabahID).First(&nas).Error; err != nil {
			return
		}
		pesanNilai := FormatPenarikanNilai(snapshotNominal, snapshotSatuan)
		admins, err := s.repo.FindAdminsByBankID(snapshotBankID)
		if err != nil {
			return
		}
		for _, au := range admins {
			if err := s.notifSvc.NotifPengajuanPenarikan(
				context.Background(), au.UserID, au.FCMToken,
				nas.User.Nama, pesanNilai, snapshotPenarikanID,
			); err != nil {
				fmt.Printf("[Notif] Gagal kirim notif pengajuan penarikan ke user %s: %v\n", au.UserID, err)
			}
		}
	}()

	return &AjukanResult{
		PenarikanID: penarikanID,
		Nominal:     finalNominal,
		Status:      models.StatusPenarikanPending,
	}, nil
}

// ── Konfirmasi ─────────────────────────────────────────────────────────────

func (s *penarikanService) Konfirmasi(penarikanID, adminUserID, fotoURL string) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		admin, err := s.repo.FindAdminTx(tx, adminUserID)
		if err != nil {
			return errForbidden("Hanya admin/petugas yang dapat melakukan konfirmasi")
		}
		if admin.BankID == nil {
			return errForbidden("Admin tidak terhubung ke bank manapun")
		}

		p, err := s.repo.LockPenarikan(tx, penarikanID)
		if err != nil {
			return errNotFound("Penarikan tidak ditemukan")
		}

		if p.BankID == nil || *p.BankID != *admin.BankID {
			return errForbidden("Tidak memiliki akses untuk mengkonfirmasi penarikan ini")
		}

		// Cegah petugas melayani penarikan miliknya sendiri sebagai nasabah
		if p.NasabahID != nil {
			var nasabah models.Nasabah
			if err := tx.Where("nasabah_id = ?", *p.NasabahID).First(&nasabah).Error; err == nil {
				if nasabah.UserID == adminUserID {
					return errForbidden("Petugas tidak boleh melayani penarikan saldo nasabah sendiri")
				}
			}
		}

		if p.StatusPenarikan != models.StatusPenarikanPending {
			return errConflict(
				"Penarikan tidak dapat dikonfirmasi, status saat ini: "+string(p.StatusPenarikan),
				nil,
			)
		}

		return s.repo.UpdateStatus(tx, p, map[string]interface{}{
			"status_penarikan": models.StatusPenarikanBerhasil,
			"bukti_foto":       fotoURL,
			"updated_at":       time.Now(),
			"updated_by":       admin.AdminID,
		})
	})

	if err != nil {
		return err
	}

	// Notifikasi ke nasabah (fire-and-forget)
	snapshotPenarikanID := penarikanID
	go func() {
		p, err := s.repo.FindPenarikanByID(snapshotPenarikanID)
		if err != nil || p.NasabahID == nil || p.Nasabah == nil {
			return
		}
		pesanNilai := FormatPenarikanNilai(p.NominalPenarikan, p.SatuanPenarikan)
		if err := s.notifSvc.NotifPenarikanBerhasil(
			context.Background(),
			p.Nasabah.User.UserID,
			p.Nasabah.User.FCMToken,
			pesanNilai,
			snapshotPenarikanID,
		); err != nil {
			fmt.Printf("[Notif] Gagal kirim notif penarikan berhasil ke user %s: %v\n", p.Nasabah.User.UserID, err)
		}
	}()

	return nil
}

// ── Batal ──────────────────────────────────────────────────────────────────

func (s *penarikanService) Batal(penarikanID, nasabahUserID string) error {
	nasabah, err := s.repo.FindNasabahByUserID(nasabahUserID)
	if err != nil {
		return errForbidden("Hanya nasabah yang dapat membatalkan pengajuan")
	}
	nasabahID := nasabah.NasabahID

	return s.db.Transaction(func(tx *gorm.DB) error {
		p, err := s.repo.LockPenarikan(tx, penarikanID)
		if err != nil {
			return errNotFound("Pengajuan penarikan tidak ditemukan")
		}
		if p.NasabahID == nil || *p.NasabahID != nasabahID {
			return errForbidden("Anda tidak memiliki akses untuk membatalkan pengajuan ini")
		}
		if p.StatusPenarikan != models.StatusPenarikanPending {
			return errBadRequest("Pengajuan tidak dapat dibatalkan (sudah diproses/kadaluarsa/batal)")
		}

		if err := s.repo.UpdateStatus(tx, p, map[string]interface{}{
			"status_penarikan": models.StatusPenarikanDibatalkan,
			"updated_at":       time.Now(),
			"updated_by":       nasabahID,
		}); err != nil {
			return err
		}

		// Kembalikan saldo
		saldo, err := s.repo.LockSaldo(tx, nasabahID, *p.RewardID)
		if err != nil {
			return err
		}
		saldoSebelum := saldo.NominalSaldo
		saldoSesudah := saldoSebelum + p.NominalPenarikan

		if err := s.repo.UpdateSaldo(tx, saldo, saldoSesudah); err != nil {
			return err
		}
		if err := s.repo.CreateRiwayat(tx, &models.RiwayatArusSaldo{
			RiwayatSaldoID: utils.GenerateID("RS"),
			RekeningID:     &saldo.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			CreatedBy:      &nasabahID,
		}); err != nil {
			return err
		}

		// Kembalikan stok sembako jika ada
		if p.Reward != nil && p.Reward.NamaReward == models.RewardEnumSembako {
			details, err := s.repo.FindDetailSembako(tx, p.PenarikanID)
			if err != nil {
				return err
			}
			for _, d := range details {
				if err := s.repo.UpdateStok(tx, d.SembakoID, *p.BankID, d.Qty); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// ── Query ──────────────────────────────────────────────────────────────────

func (s *penarikanService) GetList(nasabahID string, f repositories.PenarikanFilter) (*ListPenarikanResult, error) {
	list, err := s.repo.FindByNasabahID(nasabahID, f)
	if err != nil {
		return nil, errInternal("Gagal mengambil data penarikan")
	}

	summaries := make([]PenarikanSummary, 0, len(list))
	for _, p := range list {
		summaries = append(summaries, mapPenarikanSummary(p))
	}

	return &ListPenarikanResult{Data: summaries}, nil
}

func (s *penarikanService) GetListByBank(bankID string, f repositories.PenarikanFilter) ([]PenarikanSummary, error) {
	list, err := s.repo.FindByBankID(bankID, f)
	if err != nil {
		return nil, errInternal("Gagal mengambil data penarikan")
	}
	summaries := make([]PenarikanSummary, 0, len(list))
	for _, p := range list {
		summaries = append(summaries, mapPenarikanSummary(p))
	}
	return summaries, nil
}

func (s *penarikanService) GetDetail(penarikanID, userID string, role models.RoleAdmin) (*PenarikanDetail, error) {
	p, err := s.repo.FindPenarikanByID(penarikanID)
	if err != nil {
		return nil, errNotFound("Penarikan tidak ditemukan")
	}

	// Validasi akses
	if role == models.RoleNasabah {
		if p.NasabahID == nil {
			return nil, errForbidden("Anda tidak memiliki akses ke data ini")
		}
		var count int64
		s.db.Model(&models.Nasabah{}).
			Where("user_id = ? AND nasabah_id = ?", userID, *p.NasabahID).
			Count(&count)
		if count == 0 {
			return nil, errForbidden("Anda tidak memiliki akses ke data ini")
		}
	} else if role != models.SuperAdmin {
		var admin models.Admin
		if err := s.db.Where("user_id = ?", userID).First(&admin).Error; err != nil {
			return nil, errForbidden("Data petugas tidak ditemukan")
		}
		if admin.BankID == nil || p.BankID == nil || *admin.BankID != *p.BankID {
			return nil, errForbidden("Anda tidak memiliki akses ke data bank ini")
		}
	}

	detail := &PenarikanDetail{PenarikanSummary: mapPenarikanSummary(*p)}

	if len(p.DetailPenarikanSembako) > 0 {
		items := make([]DetailSembakoItem, 0, len(p.DetailPenarikanSembako))
		for _, d := range p.DetailPenarikanSembako {
			item := DetailSembakoItem{
				SembakoID:    d.SembakoID,
				Qty:          d.Qty,
				NilaiPoin:    d.NilaiPoin,
				SubtotalPoin: d.SubtotalPoin,
			}
			if d.KatalogSembako != nil {
				item.NamaSembako = d.KatalogSembako.MasterSembako.NamaBarang
				item.PhotoURL = d.KatalogSembako.PhotoURL
			}
			items = append(items, item)
		}
		detail.DetailSembako = items
	}

	return detail, nil
}

// ── Helpers ────────────────────────────────────────────────────────────────

func validatePenarikanReq(reward *models.Reward, req AjukanPenarikanReq) error {
	if reward.NamaReward != models.RewardEnumSembako && req.NominalPenarikan <= 0 {
		return errBadRequest("Nominal penarikan harus lebih dari 0")
	}
	if reward.NamaReward == models.RewardEnumSembako && len(req.ItemSembako) == 0 {
		return errBadRequest("Item sembako harus dipilih")
	}
	return nil
}

func mapPenarikanSummary(p models.Penarikan) PenarikanSummary {
	s := PenarikanSummary{
		PenarikanID:      p.PenarikanID,
		NasabahID:        p.NasabahID,
		BankID:           p.BankID,
		RewardID:         p.RewardID,
		NominalPenarikan: p.NominalPenarikan,
		SatuanPenarikan:  p.SatuanPenarikan,
		StatusPenarikan:  p.StatusPenarikan,
		KadaluarsaAt:     p.KadaluarsaAt,
		BuktiFoto:        p.BuktiFoto,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
	if p.Nasabah != nil && p.Nasabah.User.UserID != "" {
		s.NamaNasabah = p.Nasabah.User.Nama
	}
	if p.Bank != nil {
		s.NamaBank = p.Bank.NamaBank
	}
	if p.Reward != nil {
		s.NamaReward = string(p.Reward.NamaReward)
	}
	return s
}

// FormatPenarikanNilai memformat nilai penarikan sesuai satuannya.
// Diekspor agar controller dapat memakainya jika diperlukan.
func FormatPenarikanNilai(nominal float64, satuan models.SatuanRewardEnum) string {
	switch satuan {
	case models.SatuanRewardEnumRp:
		return fmt.Sprintf("Rp%.0f", nominal)
	case models.SatuanRewardEnumPoin:
		return fmt.Sprintf("%.0f poin", nominal)
	default:
		return fmt.Sprintf("%.2f %s", nominal, satuan)
	}
}

