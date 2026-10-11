package services

import (
	"context"
	"encoding/json"
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type PenarikanRepoIface interface {
	FindNasabahAktif(nasabahID string) (*models.Nasabah, error)
	FindNasabah(tx *gorm.DB, nasabahID string) (*models.Nasabah, error)
	FindNasabahWithUser(nasabahID string) (*models.Nasabah, error)
	CountNasabahMilikUser(userID, nasabahID string) (int64, error)
	FindNasabahByUserID(userID string) (*models.Nasabah, error)
	FindReward(rewardID int) (*models.Reward, error)
	FindSaldoRekening(nasabahID string, rewardID int) (*models.SaldoRekening, error)
	FindProduk(produkID, bankID string) (*models.KatalogBarang, error)
	FindStokBarang(produkID, bankID string) (*models.StokBarang, error)
	FindPenarikanByID(penarikanID string) (*models.Penarikan, error)
	FindRiwayatPenarikan(penarikanID string) ([]repositories.RiwayatPenarikanRow, error)
	FindByNasabahID(nasabahID string, f repositories.PenarikanFilter) ([]repositories.PenarikanListRow, error)
	FindByBankID(bankID string, f repositories.PenarikanFilter) ([]models.Penarikan, error)
	FindAdminsByBankID(bankID string) ([]repositories.AdminInfo, error)

	LockSaldo(tx *gorm.DB, nasabahID string, rewardID int) (*models.SaldoRekening, error)
	LockProduk(tx *gorm.DB, produkID, bankID string) (*models.KatalogBarang, error)
	LockStok(tx *gorm.DB, produkID, bankID string) (*models.StokBarang, error)
	LockPenarikan(tx *gorm.DB, penarikanID string) (*models.Penarikan, error)
	FindAdminAktif(tx *gorm.DB, userID string) (*models.Admin, error)
	FindDetailBarang(tx *gorm.DB, penarikanID string) ([]models.DetailPenarikanBarang, error)
	UpdateSaldo(tx *gorm.DB, saldo *models.SaldoRekening, nominal float64) error
	CreateRiwayat(tx *gorm.DB, rw *models.RiwayatArusSaldo) error
	SavePenarikan(tx *gorm.DB, p *models.Penarikan) error
	SaveRiwayatPenarikan(tx *gorm.DB, rw *models.RiwayatPenarikan) error
	SaveDetailBarang(tx *gorm.DB, d *models.DetailPenarikanBarang) error
	UpdateStok(tx *gorm.DB, produkID, bankID string, delta float64) error
	UpdateStatus(tx *gorm.DB, p *models.Penarikan, updates map[string]interface{}) error
}

type BarangItemReq struct {
	ProdukID string  `json:"produk_id" binding:"required"`
	Qty      float64 `json:"qty"        binding:"required,gt=0"`
}

type AjukanPenarikanReq struct {
	RewardID         int             `json:"reward_id"          binding:"required"`
	NominalPenarikan float64         `json:"nominal_penarikan"`
	ItemBarang       []BarangItemReq `json:"item_barang"`

	DeadlineKonfirmasi string  `json:"deadline_konfirmasi"`
	Catatan            *string `json:"catatan"`
}

type KonfirmasiPenarikanReq struct {
	Status  string  `json:"status"`
	Catatan *string `json:"catatan"`

	DeadlineJemput string `json:"deadline_jemput"`
}

type SelesaiPenarikanReq struct {
	QRData      string  `json:"qr_data"`
	NasabahID   string  `json:"nasabah_id"`
	PenarikanID string  `json:"penarikan_id"`
	BuktiFoto   string  `json:"bukti_foto"`
	Catatan     *string `json:"catatan"`
}

type PreviewBarangItem struct {
	ProdukID     string  `json:"produk_id"`
	NamaBarang   string  `json:"nama_barang"`
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
	ItemBarang       []PreviewBarangItem     `json:"item_barang,omitempty"`
}

type AjukanResult struct {
	PenarikanID string                     `json:"penarikan_id"`
	Nominal     float64                    `json:"nominal"`
	Status      models.StatusPenarikanEnum `json:"status"`
}

type PenarikanSummary struct {
	PenarikanID        string                     `json:"penarikan_id"`
	NasabahID          *string                    `json:"nasabah_id"`
	NamaNasabah        string                     `json:"nama_nasabah"`
	BankID             *string                    `json:"bank_id"`
	NamaBank           string                     `json:"nama_bank"`
	RewardID           *int                       `json:"reward_id"`
	NamaReward         string                     `json:"nama_reward"`
	NominalPenarikan   float64                    `json:"nominal_penarikan"`
	SatuanPenarikan    models.SatuanRewardEnum    `json:"satuan_penarikan"`
	StatusPenarikan    models.StatusPenarikanEnum `json:"status_penarikan"`
	DeadlineKonfirmasi *time.Time                 `json:"deadline_konfirmasi"`
	DeadlineJemput     *time.Time                 `json:"deadline_jemput"`
	BuktiFoto          *string                    `json:"bukti_foto"`
}

type DetailBarangItem struct {
	ProdukID     string  `json:"produk_id"`
	NamaBarang   string  `json:"nama_barang"`
	PhotoURL     *string `json:"photo_url"`
	Qty          float64 `json:"qty"`
	NilaiPoin    float64 `json:"nilai_poin"`
	SubtotalPoin float64 `json:"subtotal_poin"`
}

type RiwayatPenarikanItem struct {
	RiwayatPenarikanID string                     `json:"riwayat_penarikan_id"`
	StatusPenarikan    models.StatusPenarikanEnum `json:"status_penarikan"`
	Catatan            *string                    `json:"catatan"`
	CreatedBy          string                     `json:"created_by"`
	CreatedAt          time.Time                  `json:"created_at"`
}

type PenarikanDetail struct {
	Header          PenarikanSummary       `json:"header"`
	Riwayat         []RiwayatPenarikanItem `json:"riwayat"`
	DetailPenarikan []DetailBarangItem     `json:"detail_penarikan"`
}

type PenarikanListItem struct {
	PenarikanID      string                     `json:"penarikan_id"`
	CreatedAt        *time.Time                 `json:"created_at"`
	StatusPenarikan  models.StatusPenarikanEnum `json:"status_penarikan"`
	NominalPenarikan float64                    `json:"nominal_penarikan"`
	SatuanPenarikan  models.SatuanRewardEnum    `json:"satuan_penarikan"`
	RewardID         *int                       `json:"reward_id"`
	NamaReward       string                     `json:"nama_reward"`
}

type ListPenarikanResult struct {
	Data []PenarikanListItem `json:"data"`
}

type PenarikanService interface {
	Preview(nasabahID string, req AjukanPenarikanReq) (*PreviewResult, error)
	Ajukan(nasabahID string, req AjukanPenarikanReq) (*AjukanResult, error)
	Konfirmasi(penarikanID, adminUserID string, req KonfirmasiPenarikanReq) error
	Selesai(adminUserID string, req SelesaiPenarikanReq) error
	Batal(penarikanID, nasabahUserID, catatan string) error
	GetList(nasabahID string, f repositories.PenarikanFilter) (*ListPenarikanResult, error)
	GetListByBank(bankID string, f repositories.PenarikanFilter) ([]PenarikanSummary, error)
	GetDetail(penarikanID, userID string, role models.RoleAdmin) (*PenarikanDetail, error)
}

type penarikanService struct {
	db       *gorm.DB
	repo     PenarikanRepoIface
	notifSvc NotifikasiService
}

func NewPenarikanService(db *gorm.DB, repo PenarikanRepoIface, notifSvc NotifikasiService) PenarikanService {
	return &penarikanService{db: db, repo: repo, notifSvc: notifSvc}
}

func resolveKatalogBankID(nasabah *models.Nasabah) string {
	if nasabah.Bank.JenisBank == models.BSU && nasabah.Bank.ParentBankID != nil {
		return *nasabah.Bank.ParentBankID
	}
	return nasabah.BankID
}

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
	var itemsBarang []PreviewBarangItem

	if reward.NamaReward == models.RewardEnumBarang {
		katalogBankID := resolveKatalogBankID(nasabah)
		var totalPoin float64
		for _, item := range req.ItemBarang {
			produk, err := s.repo.FindProduk(item.ProdukID, katalogBankID)
			if err != nil {
				return nil, errNotFound("Barang produk tidak ditemukan: " + item.ProdukID)
			}
			stok, err := s.repo.FindStokBarang(item.ProdukID, nasabah.BankID)
			if err != nil {
				return nil, errNotFound("Stok produk tidak ditemukan: " + item.ProdukID)
			}
			if stok.Stok < item.Qty {
				return nil, errBadRequest("Stok barang '" + produk.MasterBarang.NamaBarang + "' tidak mencukupi")
			}
			subtotal := item.Qty * produk.NilaiPoin
			totalPoin += subtotal
			itemsBarang = append(itemsBarang, PreviewBarangItem{
				ProdukID:     item.ProdukID,
				NamaBarang:   produk.MasterBarang.NamaBarang,
				Qty:          item.Qty,
				NilaiPoin:    produk.NilaiPoin,
				SubtotalPoin: subtotal,
			})
		}
		finalNominal = totalPoin
	}

	saldoSetelah := saldo.NominalSaldo - finalNominal

	return &PreviewResult{
		NasabahID:        nasabahID,
		NamaReward:       reward.NamaReward,
		Satuan:           string(reward.Satuan),
		SaldoSekarang:    saldo.NominalSaldo,
		SatuanSaldo:      saldo.SatuanNominalSaldo,
		NominalPenarikan: finalNominal,
		SaldoSetelah:     saldoSetelah,
		SaldoCukup:       saldoSetelah >= 0,
		ItemBarang:       itemsBarang,
	}, nil
}

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

	if err := validateNoDuplicateBarang(req.ItemBarang); err != nil {
		return nil, err
	}

	deadlineKonfirmasi, err := validateDeadline(req.DeadlineKonfirmasi, "deadline_konfirmasi")
	if err != nil {
		return nil, err
	}

	if req.Catatan != nil && len(*req.Catatan) > 255 {
		return nil, errBadRequest("catatan maksimal 255 karakter")
	}

	var penarikanID string
	var finalNominal float64

	err = s.db.Transaction(func(tx *gorm.DB) error {
		saldo, err := s.repo.LockSaldo(tx, nasabahID, req.RewardID)
		if err != nil {
			return errNotFound("Rekening saldo untuk reward ini tidak ditemukan")
		}

		nominalTx := req.NominalPenarikan
		var itemsToSave []models.DetailPenarikanBarang

		if reward.NamaReward == models.RewardEnumBarang {
			katalogBankID := resolveKatalogBankID(nasabah)
			var totalPoin float64
			for _, item := range req.ItemBarang {
				produk, err := s.repo.LockProduk(tx, item.ProdukID, katalogBankID)
				if err != nil {
					return errNotFound("Barang produk tidak ditemukan: " + item.ProdukID)
				}
				stok, err := s.repo.LockStok(tx, item.ProdukID, nasabah.BankID)
				if err != nil {
					return errNotFound("Stok produk tidak ditemukan: " + item.ProdukID)
				}
				if stok.Stok < item.Qty {
					return errBadRequest("Stok barang '" + produk.MasterBarang.NamaBarang + "' tidak mencukupi")
				}
				subtotal := item.Qty * produk.NilaiPoin
				totalPoin += subtotal
				itemsToSave = append(itemsToSave, models.DetailPenarikanBarang{
					ProdukID:     item.ProdukID,
					Qty:          item.Qty,
					NilaiPoin:    produk.NilaiPoin,
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

		riwayatSaldoID, err := idgen.GenerateRiwayatSaldoID(tx, saldo.RekeningID)
		if err != nil {
			return errInternal("Gagal generate riwayat saldo ID: " + err.Error())
		}
		if err := s.repo.CreateRiwayat(tx, &models.RiwayatArusSaldo{
			RiwayatSaldoID: riwayatSaldoID,
			RekeningID:     &saldo.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			CreatedBy:      &nasabahID,
			Keterangan:     "Pengajuan penarikan",
		}); err != nil {
			return err
		}

		penarikanID, err = idgen.GeneratePenarikanID(tx, req.RewardID, nasabahID)
		if err != nil {
			return errInternal("Gagal generate penarikan ID: " + err.Error())
		}

		p := &models.Penarikan{
			PenarikanID:        penarikanID,
			NasabahID:          &nasabahID,
			BankID:             &nasabah.BankID,
			RewardID:           &req.RewardID,
			NominalPenarikan:   nominalTx,
			SatuanPenarikan:    models.SatuanRewardEnum(reward.Satuan),
			StatusPenarikan:    models.StatusPenarikanPending,
			DeadlineKonfirmasi: &deadlineKonfirmasi,
		}
		if err := s.repo.SavePenarikan(tx, p); err != nil {
			return err
		}

		riwayatID, err := idgen.GenerateRiwayatPenarikanID(tx, penarikanID)
		if err != nil {
			return errInternal("Gagal generate riwayat penarikan ID: " + err.Error())
		}
		if err := s.repo.SaveRiwayatPenarikan(tx, &models.RiwayatPenarikan{
			RiwayatPenarikanID: riwayatID,
			PenarikanID:        penarikanID,
			StatusPenarikan:    models.StatusPenarikanPending,
			Catatan:            req.Catatan,
			CreatedBy:          &nasabah.UserID,
		}); err != nil {
			return err
		}

		for i := range itemsToSave {
			itemsToSave[i].PenarikanID = penarikanID
			if err := s.repo.SaveDetailBarang(tx, &itemsToSave[i]); err != nil {
				return err
			}
			if err := s.repo.UpdateStok(tx, itemsToSave[i].ProdukID, nasabah.BankID, -itemsToSave[i].Qty); err != nil {
				return err
			}
		}

		finalNominal = nominalTx
		return nil
	})

	if err != nil {
		return nil, err
	}

	snapshotBankID := nasabah.BankID
	snapshotNasabahID := nasabahID
	snapshotPenarikanID := penarikanID
	snapshotNominal := finalNominal
	snapshotSatuan := models.SatuanRewardEnum(reward.Satuan)

	go func() {
		nas, err := s.repo.FindNasabahWithUser(snapshotNasabahID)
		if err != nil {
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

func (s *penarikanService) Konfirmasi(penarikanID, adminUserID string, req KonfirmasiPenarikanReq) error {
	status := models.StatusPenarikanEnum(req.Status)
	if status != models.StatusPenarikanApproved && status != models.StatusPenarikanRejected {
		return errBadRequest("status harus 'approved' atau 'rejected'")
	}

	if status == models.StatusPenarikanRejected && (req.Catatan == nil || strings.TrimSpace(*req.Catatan) == "") {
		return errBadRequest("catatan wajib diisi untuk penolakan")
	}
	if req.Catatan != nil && len(*req.Catatan) > 255 {
		return errBadRequest("catatan maksimal 255 karakter")
	}

	var deadlineJemput time.Time
	if status == models.StatusPenarikanApproved {
		var err error
		deadlineJemput, err = validateDeadline(req.DeadlineJemput, "deadline_jemput")
		if err != nil {
			return err
		}
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		admin, err := s.repo.FindAdminAktif(tx, adminUserID)
		if err != nil {
			return errForbidden("Hanya petugas yang dapat melakukan konfirmasi")
		}
		if admin.BankID == nil {
			return errForbidden("Admin tidak terhubung ke bank manapun")
		}
		if admin.Role != models.PetugasBSI && admin.Role != models.PetugasBSM && admin.Role != models.PetugasBSU {
			return errForbidden("Hanya petugas yang dapat melakukan konfirmasi")
		}

		p, err := s.repo.LockPenarikan(tx, penarikanID)
		if err != nil {
			return errNotFound("Penarikan tidak ditemukan")
		}

		if p.BankID == nil || *p.BankID != *admin.BankID {
			return errForbidden("Tidak memiliki akses untuk mengkonfirmasi penarikan ini")
		}

		if p.NasabahID != nil {
			if nasabah, err := s.repo.FindNasabah(tx, *p.NasabahID); err == nil {
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

		if p.DeadlineKonfirmasi != nil && time.Now().After(*p.DeadlineKonfirmasi) {
			return errConflict("Batas waktu konfirmasi untuk penarikan ini sudah lewat", nil)
		}

		updates := map[string]interface{}{
			"status_penarikan": status,
		}
		if status == models.StatusPenarikanApproved {
			updates["deadline_jemput"] = deadlineJemput
		}
		if err := s.repo.UpdateStatus(tx, p, updates); err != nil {
			return err
		}

		riwayatID, err := idgen.GenerateRiwayatPenarikanID(tx, penarikanID)
		if err != nil {
			return errInternal("Gagal generate riwayat penarikan ID: " + err.Error())
		}
		if err := s.repo.SaveRiwayatPenarikan(tx, &models.RiwayatPenarikan{
			RiwayatPenarikanID: riwayatID,
			PenarikanID:        penarikanID,
			StatusPenarikan:    status,
			Catatan:            req.Catatan,
			CreatedBy:          &adminUserID,
		}); err != nil {
			return err
		}

		if status == models.StatusPenarikanRejected {
			if p.NasabahID == nil || p.RewardID == nil {
				return errInternal("Data penarikan ini tidak valid")
			}

			saldo, err := s.repo.LockSaldo(tx, *p.NasabahID, *p.RewardID)
			if err != nil {
				return errNotFound("Rekening saldo untuk reward ini tidak ditemukan")
			}
			saldoSebelum := saldo.NominalSaldo
			saldoSesudah := saldoSebelum + p.NominalPenarikan

			if err := s.repo.UpdateSaldo(tx, saldo, saldoSesudah); err != nil {
				return err
			}
			riwayatSaldoID, err := idgen.GenerateRiwayatSaldoID(tx, saldo.RekeningID)
			if err != nil {
				return err
			}
			if err := s.repo.CreateRiwayat(tx, &models.RiwayatArusSaldo{
				RiwayatSaldoID: riwayatSaldoID,
				RekeningID:     &saldo.RekeningID,
				NominalSebelum: saldoSebelum,
				NominalSesudah: saldoSesudah,
				CreatedBy:      &admin.AdminID,
				Keterangan:     "Pengembalian saldo",
			}); err != nil {
				return err
			}

			if p.Reward != nil && p.Reward.NamaReward == models.RewardEnumBarang {
				if p.BankID == nil {
					return errInternal("Data bank pada penarikan ini tidak valid")
				}
				details, err := s.repo.FindDetailBarang(tx, p.PenarikanID)
				if err != nil {
					return err
				}
				for _, d := range details {
					if err := s.repo.UpdateStok(tx, d.ProdukID, *p.BankID, d.Qty); err != nil {
						return err
					}
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	snapshotPenarikanID := penarikanID
	snapshotStatus := status
	snapshotDeadlineJemput := deadlineJemput

	go func() {
		p, err := s.repo.FindPenarikanByID(snapshotPenarikanID)
		if err != nil || p.NasabahID == nil || p.Nasabah == nil {
			return
		}

		if snapshotStatus == models.StatusPenarikanApproved {
			pesanDeadline := utils.FormatTanggalIndo(snapshotDeadlineJemput)
			if err := s.notifSvc.NotifPenarikanDisetujui(
				context.Background(),
				p.Nasabah.User.UserID,
				p.Nasabah.User.FCMToken,
				pesanDeadline,
				snapshotPenarikanID,
			); err != nil {
				fmt.Printf("[Notif] Gagal kirim notif penarikan disetujui ke user %s: %v\n", p.Nasabah.User.UserID, err)
			}
			return
		}

		if err := s.notifSvc.NotifPenarikanDitolak(
			context.Background(),
			p.Nasabah.User.UserID,
			p.Nasabah.User.FCMToken,
			snapshotPenarikanID,
		); err != nil {
			fmt.Printf("[Notif] Gagal kirim notif penarikan ditolak ke user %s: %v\n", p.Nasabah.User.UserID, err)
		}
	}()

	return nil
}

func (s *penarikanService) Selesai(adminUserID string, req SelesaiPenarikanReq) error {
	nasabahID := req.NasabahID
	penarikanID := req.PenarikanID
	buktiFoto := req.BuktiFoto

	if req.QRData != "" {
		var payload struct {
			Type        string `json:"type"`
			NasabahID   string `json:"nasabah_id"`
			PenarikanID string `json:"penarikan_id"`
		}
		if err := json.Unmarshal([]byte(req.QRData), &payload); err != nil {
			return errBadRequest("Format QR code tidak valid")
		}
		if payload.Type != "ENVIROO-PENARIKAN" {
			return errBadRequest("QR code bukan milik aplikasi ini")
		}
		nasabahID = payload.NasabahID
		penarikanID = payload.PenarikanID
		buktiFoto = ""
	} else {
		if buktiFoto == "" {
			return errBadRequest("Bukti foto wajib diunggah jika tidak menggunakan QR code")
		}
		if req.Catatan == nil || strings.TrimSpace(*req.Catatan) == "" {
			return errBadRequest("Catatan wajib diisi jika tidak menggunakan QR code")
		}
	}

	if nasabahID == "" || penarikanID == "" {
		return errBadRequest("nasabah_id dan penarikan_id wajib diisi")
	}

	if req.Catatan != nil && len(*req.Catatan) > 255 {
		return errBadRequest("catatan maksimal 255 karakter")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		admin, err := s.repo.FindAdminAktif(tx, adminUserID)
		if err != nil {
			return errForbidden("Hanya petugas yang dapat menyelesaikan penarikan")
		}
		if admin.BankID == nil {
			return errForbidden("Admin tidak terhubung ke bank manapun")
		}
		if admin.Role != models.PetugasBSI && admin.Role != models.PetugasBSM && admin.Role != models.PetugasBSU {
			return errForbidden("Hanya petugas yang dapat menyelesaikan penarikan")
		}

		p, err := s.repo.LockPenarikan(tx, penarikanID)
		if err != nil {
			return errNotFound("Penarikan tidak ditemukan")
		}

		if p.BankID == nil || *p.BankID != *admin.BankID {
			return errForbidden("Tidak memiliki akses untuk menyelesaikan penarikan ini")
		}

		if p.NasabahID == nil || *p.NasabahID != nasabahID {
			return errBadRequest("Nasabah tidak sesuai dengan penarikan ini")
		}

		if nasabah, err := s.repo.FindNasabah(tx, nasabahID); err == nil {
			if nasabah.UserID == adminUserID {
				return errForbidden("Petugas tidak boleh melayani penarikan saldo nasabah sendiri")
			}
		}

		if p.StatusPenarikan != models.StatusPenarikanApproved {
			return errConflict(
				"Penarikan tidak dapat diselesaikan, status saat ini: "+string(p.StatusPenarikan),
				nil,
			)
		}

		if p.DeadlineJemput != nil && time.Now().After(*p.DeadlineJemput) {
			return errConflict("Batas waktu penjemputan untuk penarikan ini sudah lewat", nil)
		}

		updates := map[string]interface{}{
			"status_penarikan": models.StatusPenarikanCompleted,
		}
		if buktiFoto != "" {
			updates["bukti_foto"] = buktiFoto
		}
		if err := s.repo.UpdateStatus(tx, p, updates); err != nil {
			return err
		}

		riwayatID, err := idgen.GenerateRiwayatPenarikanID(tx, penarikanID)
		if err != nil {
			return errInternal("Gagal generate riwayat penarikan ID: " + err.Error())
		}
		if err := s.repo.SaveRiwayatPenarikan(tx, &models.RiwayatPenarikan{
			RiwayatPenarikanID: riwayatID,
			PenarikanID:        penarikanID,
			StatusPenarikan:    models.StatusPenarikanCompleted,
			Catatan:            req.Catatan,
			CreatedBy:          &adminUserID,
		}); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

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

func (s *penarikanService) Batal(penarikanID, nasabahUserID, catatan string) error {
	if len(catatan) > 255 {
		return errBadRequest("catatan maksimal 255 karakter")
	}

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
			"status_penarikan": models.StatusPenarikanCanceled,
		}); err != nil {
			return err
		}

		riwayatID, err := idgen.GenerateRiwayatPenarikanID(tx, penarikanID)
		if err != nil {
			return errInternal("Gagal generate riwayat penarikan ID: " + err.Error())
		}
		if err := s.repo.SaveRiwayatPenarikan(tx, &models.RiwayatPenarikan{
			RiwayatPenarikanID: riwayatID,
			PenarikanID:        penarikanID,
			StatusPenarikan:    models.StatusPenarikanCanceled,
			Catatan:            &catatan,
			CreatedBy:          &nasabahUserID,
		}); err != nil {
			return err
		}

		if p.RewardID == nil {
			return errInternal("Data reward pada penarikan ini tidak valid")
		}

		saldo, err := s.repo.LockSaldo(tx, nasabahID, *p.RewardID)
		if err != nil {
			return errNotFound("Rekening saldo untuk reward ini tidak ditemukan")
		}
		saldoSebelum := saldo.NominalSaldo
		saldoSesudah := saldoSebelum + p.NominalPenarikan

		if err := s.repo.UpdateSaldo(tx, saldo, saldoSesudah); err != nil {
			return err
		}
		riwayatSaldoID, err := idgen.GenerateRiwayatSaldoID(tx, saldo.RekeningID)
		if err != nil {
			return err
		}
		if err := s.repo.CreateRiwayat(tx, &models.RiwayatArusSaldo{
			RiwayatSaldoID: riwayatSaldoID,
			RekeningID:     &saldo.RekeningID,
			NominalSebelum: saldoSebelum,
			NominalSesudah: saldoSesudah,
			CreatedBy:      &nasabahID,
			Keterangan:     "Pengembalian saldo",
		}); err != nil {
			return err
		}

		if p.Reward != nil && p.Reward.NamaReward == models.RewardEnumBarang {
			if p.BankID == nil {
				return errInternal("Data bank pada penarikan ini tidak valid")
			}
			details, err := s.repo.FindDetailBarang(tx, p.PenarikanID)
			if err != nil {
				return err
			}
			for _, d := range details {
				if err := s.repo.UpdateStok(tx, d.ProdukID, *p.BankID, d.Qty); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

func (s *penarikanService) GetList(nasabahID string, f repositories.PenarikanFilter) (*ListPenarikanResult, error) {
	rows, err := s.repo.FindByNasabahID(nasabahID, f)
	if err != nil {
		return nil, errInternal("Gagal mengambil data penarikan")
	}

	items := make([]PenarikanListItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, PenarikanListItem{
			PenarikanID:      row.PenarikanID,
			CreatedAt:        row.CreatedAt,
			StatusPenarikan:  row.StatusPenarikan,
			NominalPenarikan: row.NominalPenarikan,
			SatuanPenarikan:  row.SatuanPenarikan,
			RewardID:         row.RewardID,
			NamaReward:       row.NamaReward,
		})
	}

	return &ListPenarikanResult{Data: items}, nil
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

	if role == models.RoleNasabah {
		if p.NasabahID == nil {
			return nil, errForbidden("Anda tidak memiliki akses ke data ini")
		}
		count, err := s.repo.CountNasabahMilikUser(userID, *p.NasabahID)
		if err != nil || count == 0 {
			return nil, errForbidden("Anda tidak memiliki akses ke data ini")
		}
	} else if role != models.SuperAdmin {
		admin, err := s.repo.FindAdminAktif(nil, userID)
		if err != nil {
			return nil, errForbidden("Data petugas tidak ditemukan")
		}
		if admin.BankID == nil || p.BankID == nil || *admin.BankID != *p.BankID {
			return nil, errForbidden("Anda tidak memiliki akses ke data bank ini")
		}
	}

	detail := &PenarikanDetail{
		Header:          mapPenarikanSummary(*p),
		DetailPenarikan: []DetailBarangItem{},
	}

	if len(p.DetailPenarikanBarang) > 0 {
		items := make([]DetailBarangItem, 0, len(p.DetailPenarikanBarang))
		for _, d := range p.DetailPenarikanBarang {
			item := DetailBarangItem{
				ProdukID:     d.ProdukID,
				Qty:          d.Qty,
				NilaiPoin:    d.NilaiPoin,
				SubtotalPoin: d.SubtotalPoin,
			}
			if d.KatalogBarang != nil {
				item.NamaBarang = d.KatalogBarang.MasterBarang.NamaBarang
				item.PhotoURL = d.KatalogBarang.PhotoURL
			}
			items = append(items, item)
		}
		detail.DetailPenarikan = items
	}

	riwayatRows, err := s.repo.FindRiwayatPenarikan(penarikanID)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat penarikan")
	}
	riwayat := make([]RiwayatPenarikanItem, 0, len(riwayatRows))
	for _, row := range riwayatRows {
		createdBy := row.CreatedByRaw
		if row.CreatedByRaw == "sistem" {
			createdBy = "Sistem"
		} else if row.CreatedByNama != nil && *row.CreatedByNama != "" {
			createdBy = *row.CreatedByNama
		}
		riwayat = append(riwayat, RiwayatPenarikanItem{
			RiwayatPenarikanID: row.RiwayatPenarikanID,
			StatusPenarikan:    row.StatusPenarikan,
			Catatan:            row.Catatan,
			CreatedBy:          createdBy,
			CreatedAt:          row.CreatedAt,
		})
	}
	detail.Riwayat = riwayat

	return detail, nil
}

func validatePenarikanReq(reward *models.Reward, req AjukanPenarikanReq) error {
	if reward.NamaReward != models.RewardEnumBarang && req.NominalPenarikan <= 0 {
		return errBadRequest("Nominal penarikan harus lebih dari 0")
	}
	if reward.NamaReward == models.RewardEnumBarang && len(req.ItemBarang) == 0 {
		return errBadRequest("Item produk harus dipilih")
	}
	return nil
}

func validateNoDuplicateBarang(items []BarangItemReq) error {
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		if seen[item.ProdukID] {
			return errBadRequest("Barang produk duplikat dalam satu pengajuan: " + item.ProdukID)
		}
		seen[item.ProdukID] = true
	}
	return nil
}

func validateDeadline(raw, fieldName string) (time.Time, error) {
	if raw == "" {
		return time.Time{}, errBadRequest(fieldName + " wajib diisi")
	}

	deadline, err := utils.ParseFlexibleDate(raw)
	if err != nil {
		return time.Time{}, errBadRequest("Format " + fieldName + " tidak valid")
	}

	now := time.Now()
	if deadline.Before(now.Add(1 * time.Hour)) {
		return time.Time{}, errBadRequest(fieldName + " minimal 1 jam dari waktu sekarang")
	}
	if deadline.After(now.Add(3 * 24 * time.Hour)) {
		return time.Time{}, errBadRequest(fieldName + " maksimal 3 hari dari waktu sekarang")
	}

	return deadline, nil
}

func mapPenarikanSummary(p models.Penarikan) PenarikanSummary {
	s := PenarikanSummary{
		PenarikanID:        p.PenarikanID,
		NasabahID:          p.NasabahID,
		BankID:             p.BankID,
		RewardID:           p.RewardID,
		NominalPenarikan:   p.NominalPenarikan,
		SatuanPenarikan:    p.SatuanPenarikan,
		StatusPenarikan:    p.StatusPenarikan,
		DeadlineKonfirmasi: p.DeadlineKonfirmasi,
		DeadlineJemput:     p.DeadlineJemput,
		BuktiFoto:          p.BuktiFoto,
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
