package services

import (
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"errors"
	"math"
	"time"

	"gorm.io/gorm"
)

type BagiHasilRepoIface interface {
	GetBankByID(bankID string) (*models.BankSampah, error)
	GetPenjualanWithReward(penjualanID, bankID string) (*models.Penjualan, error)
	GetDetailPenjualan(db *gorm.DB, penjualanID string) ([]models.DetailPenjualan, error)
	GetBSUList(db *gorm.DB, parentBankID string) ([]models.BankSampah, error)
	GetTabunganNasabah(db *gorm.DB, sampahID string, bankIDs []string) ([]models.TabunganSampah, error)
	GetTabunganBSU(db *gorm.DB, sampahID string, bsuIDs []string) ([]models.TabunganSampah, error)
	SaveTabungan(db *gorm.DB, t *models.TabunganSampah) error
	UpsertSchemaHargaNasabah(db *gorm.DB, sampahID string, harga float64, satuan models.SatuanRewardEnum, adminID string) error
	CreateBagiHasil(db *gorm.DB, bh *models.BagiHasil) error
	CreatePerhitunganSisa(db *gorm.DB, ps *models.PerhitunganSisa) error
	CreatePenerimaBagiHasil(db *gorm.DB, p *models.PenerimaBagiHasil) error
	CreateDetailBagiHasil(db *gorm.DB, d *models.DetailBagiHasil) error
	CreatePencairanTabungan(db *gorm.DB, p *models.PencairanTabungan) error
	UpdatePenjualanStatus(db *gorm.DB, penjualan *models.Penjualan, status models.StatusBagiHasilEnum) error
	UpdateSaldoRekening(db *gorm.DB, bankID *string, nasabahID *string, rewardID int, reward models.Reward, nilai float64, entitas models.EntitasEnum, adminID string, now time.Time) error
	GetNasabahWithUser(nasabahID string) (*models.Nasabah, error)
	GetNasabahInfoBatch(nasabahIDs []string) ([]repositories.NasabahRow, error)
	GetBagiHasilByPenjualan(penjualanID string) (*models.BagiHasil, error)
	GetBagiHasilWithAll(bagiHasilID string) (*models.BagiHasil, error)
	GetPenerimaBagiHasil(bagiHasilID string) ([]models.PenerimaBagiHasil, error)
	GetPenerimaNasabahFull(bagiHasilID string) ([]models.PenerimaBagiHasil, error)
	GetPenerimaByNasabah(nasabahID, startDate, endDate string) ([]models.PenerimaBagiHasil, error)
	GetPenerimaBagiHasilNasabah(penerimaID string) (*models.PenerimaBagiHasil, error)
	GetDetailBagiHasilItems(penerimaID string) ([]models.DetailBagiHasil, error)
	GetBagiHasilByBankWithFilter(bankID string, startTime, endTime time.Time) ([]models.BagiHasil, error)
	GetNamaPetugas(adminID string) string
	FindDistribusiSisaID(bagiHasilID string) *string
}

type BagiHasilService struct {
	db   *gorm.DB
	repo BagiHasilRepoIface
}

func NewBagiHasilService(db *gorm.DB, repo *repositories.BagiHasilRepo) *BagiHasilService {
	return &BagiHasilService{db: db, repo: repo}
}

func NewBagiHasilServiceWithRepo(db *gorm.DB, repo BagiHasilRepoIface) *BagiHasilService {
	return &BagiHasilService{db: db, repo: repo}
}

type NasabahPenerimaItem struct {
	PenerimaID     string                  `json:"penerima_id"`
	NasabahID      string                  `json:"nasabah_id"`
	NamaNasabah    string                  `json:"nama_nasabah"`
	TotalDiterima  float64                 `json:"total_diterima"`
	SatuanDiterima models.SatuanRewardEnum `json:"satuan_diterima"`
}

type BankPenerimaGroup struct {
	BankID          string                `json:"bank_id"`
	NamaBank        string                `json:"nama_bank"`
	NasabahPenerima []NasabahPenerimaItem `json:"nasabah_penerima"`
}

type NasabahNotifItem struct {
	PenerimaID string
	NasabahID  string
	UserID     string
	FCMToken   string
	Total      float64
	Satuan     models.SatuanRewardEnum
}

func (s *BagiHasilService) siapkanBagiHasil(penjualanID, bankID string) (*models.Penjualan, *models.BankSampah, []models.DetailPenjualan, error) {
	penjualan, err := s.repo.GetPenjualanWithReward(penjualanID, bankID)
	if err != nil {
		return nil, nil, nil, errNotFound("Penjualan tidak ditemukan")
	}
	if penjualan.StatusBagiHasil == models.BagiHasilBerhasil {
		return nil, nil, nil, errBadRequest("Bagi hasil untuk penjualan ini sudah dilakukan")
	}

	bank, err := s.repo.GetBankByID(bankID)
	if err != nil {
		return nil, nil, nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank == models.BSU {
		return nil, nil, nil, errForbidden("BSU tidak dapat melakukan bagi hasil")
	}

	detailPenjualans, err := s.repo.GetDetailPenjualan(s.db, penjualanID)
	if err != nil {
		return nil, nil, nil, errInternal("Gagal mengambil detail penjualan")
	}
	return penjualan, bank, detailPenjualans, nil
}

type PreviewBagiHasilSummary struct {
	GrossBank              float64 `json:"gross_bank"`
	TotalDistribusiNasabah float64 `json:"total_distribusi_nasabah"`
	SisaBagiHasil          float64 `json:"sisa_bagi_hasil"`
}

type NasabahPenerimaPreview struct {
	NasabahID     string  `json:"nasabah_id"`
	NamaNasabah   string  `json:"nama_nasabah"`
	TotalDiterima float64 `json:"total_diterima"`
}

type BankPenerimaPreview struct {
	BankID          string                   `json:"bank_id"`
	NamaBank        string                   `json:"nama_bank"`
	NasabahPenerima []NasabahPenerimaPreview `json:"nasabah_penerima"`
}

type PreviewBagiHasilResp struct {
	PenjualanID string                  `json:"penjualan_id"`
	Reward      string                  `json:"reward"`
	Summary     PreviewBagiHasilSummary `json:"summary"`
	Penerima    []BankPenerimaPreview   `json:"penerima"`
}

func (s *BagiHasilService) Preview(penjualanID, bankID string) (*PreviewBagiHasilResp, error) {
	penjualan, bank, detailPenjualans, err := s.siapkanBagiHasil(penjualanID, bankID)
	if err != nil {
		return nil, err
	}
	if bank.JenisBank == models.BSI {
		return s.previewBSI(penjualan, bankID, detailPenjualans)
	}

	return s.hitungPreview(penjualan, detailPenjualans, []string{bankID})
}

func (s *BagiHasilService) hitungPreview(penjualan *models.Penjualan, detailPenjualans []models.DetailPenjualan, bankIDsNasabah []string) (*PreviewBagiHasilResp, error) {
	totalDibagikanKeNasabah := make(map[string]float64)

	for _, detail := range detailPenjualans {
		if detail.Qty <= 0 {
			continue
		}
		if detail.HargaNasabahSnapshot <= 0 {
			return nil, errBadRequest("Penjualan ini dibuat tanpa persentase bagi hasil yang sah untuk sampah " +
				detail.SampahID + ", jadi bagi hasilnya tidak bisa dihitung")
		}
		tabungan, err := s.repo.GetTabunganNasabah(s.db, detail.SampahID, bankIDsNasabah)
		if err != nil {
			return nil, errInternal("Gagal mengambil tabungan nasabah")
		}
		sisaQty := detail.Qty
		for _, t := range tabungan {
			if sisaQty <= 0 {
				break
			}
			qtyDiambil := math.Min(t.SisaQty, sisaQty)
			totalDibagikanKeNasabah[*t.NasabahID] += qtyDiambil * detail.HargaNasabahSnapshot
			sisaQty -= qtyDiambil
		}
		if sisaQty > 0 {
			return nil, &ServiceError{
				Code:    422,
				Message: "Stok tabungan nasabah tidak mencukupi",
				Data:    map[string]interface{}{"sampah_id": detail.SampahID, "kurang_qty": sisaQty},
			}
		}
	}

	grossBank := penjualan.TotalPenjualan
	var totalDistribusiNasabah float64
	for _, v := range totalDibagikanKeNasabah {
		totalDistribusiNasabah += v
	}

	nasabahIDs := make([]string, 0, len(totalDibagikanKeNasabah))
	for id := range totalDibagikanKeNasabah {
		nasabahIDs = append(nasabahIDs, id)
	}
	type nasabahInfo struct {
		NamaNasabah string
		NamaBank    string
		BankID      string
	}
	infoCache := make(map[string]nasabahInfo)
	if len(nasabahIDs) > 0 {
		rows, _ := s.repo.GetNasabahInfoBatch(nasabahIDs)
		for _, r := range rows {
			infoCache[r.NasabahID] = nasabahInfo{
				NamaNasabah: r.NamaNasabah,
				NamaBank:    r.NamaBank,
				BankID:      r.BankID,
			}
		}
	}

	bankMap := make(map[string]*BankPenerimaPreview)
	var bankOrder []string
	for nasabahID, total := range totalDibagikanKeNasabah {
		info := infoCache[nasabahID]
		if _, ada := bankMap[info.BankID]; !ada {
			bankMap[info.BankID] = &BankPenerimaPreview{
				BankID:          info.BankID,
				NamaBank:        info.NamaBank,
				NasabahPenerima: []NasabahPenerimaPreview{},
			}
			bankOrder = append(bankOrder, info.BankID)
		}
		bankMap[info.BankID].NasabahPenerima = append(bankMap[info.BankID].NasabahPenerima, NasabahPenerimaPreview{
			NasabahID:     nasabahID,
			NamaNasabah:   info.NamaNasabah,
			TotalDiterima: total,
		})
	}
	penerima := make([]BankPenerimaPreview, 0, len(bankOrder))
	for _, bID := range bankOrder {
		penerima = append(penerima, *bankMap[bID])
	}

	return &PreviewBagiHasilResp{
		PenjualanID: penjualan.PenjualanID,
		Reward:      string(penjualan.Reward.NamaReward),
		Summary: PreviewBagiHasilSummary{
			GrossBank:              grossBank,
			TotalDistribusiNasabah: totalDistribusiNasabah,
			SisaBagiHasil:          grossBank - totalDistribusiNasabah,
		},
		Penerima: penerima,
	}, nil
}

type SubmitBagiHasilResp struct {
	BagiHasilID            string             `json:"bagi_hasil_id"`
	GrossBank              float64            `json:"gross_bank"`
	TotalDistribusiNasabah float64            `json:"total_distribusi_nasabah"`
	SisaBagiHasil          float64            `json:"sisa_bagi_hasil"`
	NasabahNotifs          []NasabahNotifItem `json:"-"`
}

func (s *BagiHasilService) Submit(penjualanID, bankID, adminID string) (*SubmitBagiHasilResp, error) {
	penjualan, bank, detailPenjualans, err := s.siapkanBagiHasil(penjualanID, bankID)
	if err != nil {
		return nil, err
	}
	if bank.JenisBank == models.BSI {
		return s.submitBSI(penjualan, bankID, detailPenjualans, adminID)
	}

	var result *SubmitBagiHasilResp

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		dist, err := s.distribusikanKeNasabah(tx, penjualan, detailPenjualans, []string{bankID}, adminID)
		if err != nil {
			return err
		}
		bh, err := s.simpanBagiHasil(tx, penjualan, bankID, dist.total, adminID, now)
		if err != nil {
			return err
		}
		nasabahNotifs, err := s.simpanPenerimaNasabah(tx, penjualan, bh.BagiHasilID, dist, adminID, now)
		if err != nil {
			return err
		}
		if err := s.masukkanSisaKeBank(tx, penjualan, bankID, bh.SisaBagiHasil, adminID, now); err != nil {
			return err
		}
		if err := s.repo.UpdatePenjualanStatus(tx, penjualan, models.BagiHasilBerhasil); err != nil {
			return errInternal("Gagal update status penjualan")
		}

		result = &SubmitBagiHasilResp{
			BagiHasilID:            bh.BagiHasilID,
			GrossBank:              bh.GrossBank,
			TotalDistribusiNasabah: bh.TotalDistribusiNasabah,
			SisaBagiHasil:          bh.SisaBagiHasil,
			NasabahNotifs:          nasabahNotifs,
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

type itemBagiHasil struct {
	sampahID string
	qty      float64
	harga    float64
	subtotal float64
}

type distribusiNasabah struct {
	items    map[string]map[string]*itemBagiHasil
	tabungan map[string]map[string]float64
	totals   map[string]float64
	total    float64
}

func (s *BagiHasilService) distribusikanKeNasabah(tx *gorm.DB, penjualan *models.Penjualan, detailPenjualans []models.DetailPenjualan, bankIDsNasabah []string, adminID string) (*distribusiNasabah, error) {
	dist := &distribusiNasabah{
		items:    make(map[string]map[string]*itemBagiHasil),
		tabungan: make(map[string]map[string]float64),
		totals:   make(map[string]float64),
	}

	for _, detail := range detailPenjualans {
		if detail.Qty <= 0 {
			return nil, errBadRequest("qty tidak valid untuk sampah_id: " + detail.SampahID)
		}
		if detail.HargaNasabahSnapshot <= 0 {
			return nil, errBadRequest("Penjualan ini dibuat tanpa persentase bagi hasil yang sah untuk sampah " +
				detail.SampahID + ", jadi bagi hasilnya tidak bisa diproses")
		}

		tabunganList, err := s.repo.GetTabunganNasabah(tx, detail.SampahID, bankIDsNasabah)
		if err != nil {
			return nil, errInternal("Gagal mengambil tabungan nasabah")
		}

		sisaQty := detail.Qty
		for i := range tabunganList {
			if sisaQty <= 0 {
				break
			}
			t := &tabunganList[i]
			nasabahID := *t.NasabahID
			qtyDiambil := math.Min(t.SisaQty, sisaQty)
			nilaiDapat := qtyDiambil * detail.HargaNasabahSnapshot

			if dist.items[nasabahID] == nil {
				dist.items[nasabahID] = make(map[string]*itemBagiHasil)
			}
			if dist.items[nasabahID][detail.SampahID] == nil {
				dist.items[nasabahID][detail.SampahID] = &itemBagiHasil{
					sampahID: detail.SampahID,
					harga:    detail.HargaNasabahSnapshot,
				}
			}
			dist.items[nasabahID][detail.SampahID].qty += qtyDiambil
			dist.items[nasabahID][detail.SampahID].subtotal += nilaiDapat

			if dist.tabungan[nasabahID] == nil {
				dist.tabungan[nasabahID] = make(map[string]float64)
			}
			dist.tabungan[nasabahID][t.TabunganID] += qtyDiambil
			dist.totals[nasabahID] += nilaiDapat
			sisaQty -= qtyDiambil

			t.SisaQty -= qtyDiambil
			if err := s.repo.SaveTabungan(tx, t); err != nil {
				return nil, errInternal("Gagal update tabungan nasabah")
			}
		}
		if sisaQty > 0 {
			return nil, &ServiceError{
				Code:    422,
				Message: "Stok tabungan nasabah tidak mencukupi",
				Data:    map[string]interface{}{"sampah_id": detail.SampahID, "kurang_qty": sisaQty},
			}
		}

		if err := s.repo.UpsertSchemaHargaNasabah(tx, detail.SampahID,
			detail.HargaNasabahSnapshot, penjualan.SatuanReward, adminID); err != nil {
			return nil, errInternal("Gagal update schema harga nasabah: " + err.Error())
		}
	}

	for _, v := range dist.totals {
		dist.total += v
	}
	return dist, nil
}

func (s *BagiHasilService) simpanBagiHasil(tx *gorm.DB, penjualan *models.Penjualan, bankID string, totalDistribusiNasabah float64, adminID string, now time.Time) (*models.BagiHasil, error) {
	penjualanID := penjualan.PenjualanID
	grossBank := penjualan.TotalPenjualan
	bh := &models.BagiHasil{
		BagiHasilID:            idgen.GenerateBagiHasilID(penjualanID),
		BankID:                 &bankID,
		PenjualanID:            &penjualanID,
		GrossBank:              grossBank,
		TotalDistribusiNasabah: totalDistribusiNasabah,
		SisaBagiHasil:          grossBank - totalDistribusiNasabah,
		SatuanBagiHasil:        penjualan.SatuanReward,
		CreatedAt:              now,
		CreatedBy:              &adminID,
	}
	if err := s.repo.CreateBagiHasil(tx, bh); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, errBadRequest("Bagi hasil untuk penjualan ini sudah dilakukan")
		}
		return nil, errInternal("Gagal menyimpan bagi hasil")
	}
	return bh, nil
}

func (s *BagiHasilService) simpanPenerimaNasabah(tx *gorm.DB, penjualan *models.Penjualan, bagiHasilID string, dist *distribusiNasabah, adminID string, now time.Time) ([]NasabahNotifItem, error) {
	var nasabahNotifs []NasabahNotifItem
	for nasabahID, sampahMap := range dist.items {
		nasabahIDCopy := nasabahID
		total := dist.totals[nasabahID]
		penerimaID, err := idgen.GeneratePenerimaBagiHasilID(tx, penjualan.PenjualanID)
		if err != nil {
			return nil, errInternal("Gagal generate penerima bagi hasil ID: " + err.Error())
		}

		pbh := &models.PenerimaBagiHasil{
			PenerimaID:     penerimaID,
			BagiHasilID:    &bagiHasilID,
			RewardID:       &penjualan.RewardID,
			NasabahID:      &nasabahIDCopy,
			TotalItem:      len(sampahMap),
			TotalDiterima:  total,
			SatuanDiterima: penjualan.SatuanReward,
			CreatedAt:      now,
			CreatedBy:      &adminID,
		}
		if err := s.repo.CreatePenerimaBagiHasil(tx, pbh); err != nil {
			return nil, errInternal("Gagal menyimpan penerima bagi hasil nasabah")
		}
		for _, item := range sampahMap {
			if err := s.repo.CreateDetailBagiHasil(tx, &models.DetailBagiHasil{
				PenerimaID:    penerimaID,
				SampahID:      item.sampahID,
				Qty:           item.qty,
				HargaItem:     item.harga,
				SubtotalHarga: item.subtotal,
			}); err != nil {
				return nil, errInternal("Gagal menyimpan detail bagi hasil nasabah")
			}
		}
		for tabunganID, qtyDipakai := range dist.tabungan[nasabahID] {
			if err := s.repo.CreatePencairanTabungan(tx, &models.PencairanTabungan{
				PenerimaID: penerimaID,
				TabunganID: tabunganID,
				QtyDipakai: qtyDipakai,
			}); err != nil {
				return nil, errInternal("Gagal menyimpan pencairan tabungan nasabah")
			}
		}

		if err := s.repo.UpdateSaldoRekening(tx, nil, &nasabahIDCopy,
			penjualan.RewardID, penjualan.Reward,
			total, models.EntitasNasabah, adminID, now); err != nil {
			return nil, errInternal("Gagal update saldo nasabah: " + err.Error())
		}

		nasabah, err := s.repo.GetNasabahWithUser(nasabahIDCopy)
		if err == nil {
			nasabahNotifs = append(nasabahNotifs, NasabahNotifItem{
				PenerimaID: penerimaID,
				NasabahID:  nasabahIDCopy,
				UserID:     nasabah.User.UserID,
				FCMToken:   nasabah.User.FCMToken,
				Total:      total,
				Satuan:     penjualan.SatuanReward,
			})
		}
	}
	return nasabahNotifs, nil
}

func (s *BagiHasilService) masukkanSisaKeBank(tx *gorm.DB, penjualan *models.Penjualan, bankID string, sisaBagiHasil float64, adminID string, now time.Time) error {
	if sisaBagiHasil <= 0 {
		return nil
	}
	if err := s.repo.UpdateSaldoRekening(tx, &bankID, nil,
		penjualan.RewardID, penjualan.Reward,
		sisaBagiHasil, models.EntitasBankSampah, adminID, now); err != nil {
		return errInternal("Gagal update saldo bank: " + err.Error())
	}
	return nil
}

type DetailBagiHasilResp struct {
	BagiHasilID            string                  `json:"bagi_hasil_id"`
	Tanggal                time.Time               `json:"tanggal"`
	PenjualanID            string                  `json:"penjualan_id"`
	NamaPetugas            string                  `json:"nama_petugas"`
	Reward                 string                  `json:"reward"`
	GrossBank              float64                 `json:"gross_bank"`
	TotalDistribusiNasabah float64                 `json:"total_distribusi_nasabah"`
	SisaBagiHasil          float64                 `json:"sisa_bagi_hasil"`
	Satuan                 models.SatuanRewardEnum `json:"satuan"`
	DistribusiID           *string                 `json:"distribusi_id"`
	NasabahLangsung        []NasabahPenerimaItem   `json:"nasabah_langsung"`
	Penerima               []BankPenerimaGroup     `json:"penerima,omitempty"`
}

func (s *BagiHasilService) GetDetail(penjualanID string) (*DetailBagiHasilResp, error) {
	bh, err := s.repo.GetBagiHasilByPenjualan(penjualanID)
	if err != nil {
		return nil, errNotFound("Data bagi hasil tidak ditemukan")
	}

	var namaPetugas string
	if bh.CreatedBy != nil {
		namaPetugas = s.repo.GetNamaPetugas(*bh.CreatedBy)
	}

	semuaPenerima, err := s.repo.GetPenerimaBagiHasil(bh.BagiHasilID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data penerima")
	}

	resp := &DetailBagiHasilResp{
		BagiHasilID:            bh.BagiHasilID,
		Tanggal:                bh.CreatedAt,
		PenjualanID:            penjualanID,
		NamaPetugas:            namaPetugas,
		Reward:                 string(bh.Penjualan.Reward.NamaReward),
		GrossBank:              bh.GrossBank,
		TotalDistribusiNasabah: bh.TotalDistribusiNasabah,
		SisaBagiHasil:          bh.SisaBagiHasil,
		Satuan:                 bh.SatuanBagiHasil,
		DistribusiID:           s.repo.FindDistribusiSisaID(bh.BagiHasilID),
	}
	if bh.Bank != nil && bh.Bank.JenisBank == models.BSI {
		return s.detailBSI(bh, semuaPenerima, resp)
	}

	resp.NasabahLangsung = make([]NasabahPenerimaItem, 0, len(semuaPenerima))
	for _, p := range semuaPenerima {
		if p.Nasabah == nil {
			continue
		}
		resp.NasabahLangsung = append(resp.NasabahLangsung, toNasabahPenerimaItem(p))
	}
	return resp, nil
}

type ListBagiHasilNasabahItem struct {
	PenerimaID     string                  `json:"penerima_id"`
	BagiHasilID    string                  `json:"bagi_hasil_id"`
	Reward         string                  `json:"reward"`
	Tanggal        string                  `json:"tanggal"`
	TotalDiterima  float64                 `json:"total_diterima"`
	SatuanDiterima models.SatuanRewardEnum `json:"satuan_diterima"`
}

type ListBagiHasilNasabahResp struct {
	NasabahID        string                     `json:"nasabah_id"`
	NamaNasabah      string                     `json:"nama_nasabah"`
	RiwayatBagiHasil []ListBagiHasilNasabahItem `json:"riwayat_bagi_hasil"`
}

func (s *BagiHasilService) GetListNasabah(nasabahID, startDate, endDate string) (*ListBagiHasilNasabahResp, error) {
	nasabah, err := s.repo.GetNasabahWithUser(nasabahID)
	if err != nil {
		return nil, errNotFound("Nasabah tidak ditemukan")
	}

	penerimas, err := s.repo.GetPenerimaByNasabah(nasabahID, startDate, endDate)
	if err != nil {
		return nil, errInternal("Gagal mengambil data bagi hasil")
	}

	list := make([]ListBagiHasilNasabahItem, 0, len(penerimas))
	for _, p := range penerimas {
		if p.BagiHasil == nil || p.BagiHasil.Penjualan == nil {
			continue
		}
		list = append(list, ListBagiHasilNasabahItem{
			PenerimaID:     p.PenerimaID,
			BagiHasilID:    *p.BagiHasilID,
			Reward:         string(p.BagiHasil.Penjualan.Reward.NamaReward),
			Tanggal:        p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			TotalDiterima:  p.TotalDiterima,
			SatuanDiterima: p.SatuanDiterima,
		})
	}

	return &ListBagiHasilNasabahResp{
		NasabahID:        nasabah.NasabahID,
		NamaNasabah:      nasabah.User.Nama,
		RiwayatBagiHasil: list,
	}, nil
}

type DetailBagiHasilItemResp struct {
	SampahID      string  `json:"sampah_id"`
	NamaSampah    string  `json:"nama_sampah"`
	Qty           float64 `json:"qty"`
	HargaItem     float64 `json:"harga_item"`
	SubtotalHarga float64 `json:"subtotal_harga"`
}

type DetailBagiHasilNasabahResp struct {
	PenerimaID     string                    `json:"penerima_id"`
	BagiHasilID    string                    `json:"bagi_hasil_id"`
	NasabahID      string                    `json:"nasabah_id"`
	NamaNasabah    string                    `json:"nama_nasabah"`
	Reward         string                    `json:"reward"`
	Tanggal        string                    `json:"tanggal"`
	TotalDiterima  float64                   `json:"total_diterima"`
	SatuanDiterima models.SatuanRewardEnum   `json:"satuan_diterima"`
	DetailItem     []DetailBagiHasilItemResp `json:"detail_item"`
}

func (s *BagiHasilService) GetDetailNasabah(penerimaID string) (*DetailBagiHasilNasabahResp, error) {
	penerima, err := s.repo.GetPenerimaBagiHasilNasabah(penerimaID)
	if err != nil {
		return nil, errNotFound("Data bagi hasil nasabah tidak ditemukan")
	}
	if penerima.BagiHasil == nil || penerima.BagiHasil.Penjualan == nil {
		return nil, errInternal("Data bagi hasil tidak lengkap")
	}

	details, err := s.repo.GetDetailBagiHasilItems(penerimaID)
	if err != nil {
		return nil, errInternal("Gagal mengambil detail bagi hasil")
	}

	items := make([]DetailBagiHasilItemResp, 0, len(details))
	for _, d := range details {
		namaSampah := ""
		if d.KatalogSampah != nil {
			namaSampah = d.KatalogSampah.Sarok.NamaSampah
		}
		items = append(items, DetailBagiHasilItemResp{
			SampahID:      d.SampahID,
			NamaSampah:    namaSampah,
			Qty:           d.Qty,
			HargaItem:     d.HargaItem,
			SubtotalHarga: d.SubtotalHarga,
		})
	}

	return &DetailBagiHasilNasabahResp{
		PenerimaID:     penerima.PenerimaID,
		BagiHasilID:    *penerima.BagiHasilID,
		NasabahID:      penerima.Nasabah.NasabahID,
		NamaNasabah:    penerima.Nasabah.User.Nama,
		Reward:         string(penerima.BagiHasil.Penjualan.Reward.NamaReward),
		Tanggal:        penerima.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalDiterima:  penerima.TotalDiterima,
		SatuanDiterima: penerima.SatuanDiterima,
		DetailItem:     items,
	}, nil
}

type ListBagiHasilBankPusatItem struct {
	BagiHasilID      string `json:"bagi_hasil_id"`
	RewardID         int    `json:"reward_id"`
	NamaReward       string `json:"nama_reward"`
	TanggalBagiHasil string `json:"tanggal_bagi_hasil"`
}

type ListBagiHasilBankPusatResp struct {
	BankID           string                       `json:"bank_id"`
	NamaBank         string                       `json:"nama_bank"`
	JenisBank        models.JenisBank             `json:"jenis_bank"`
	RiwayatBagiHasil []ListBagiHasilBankPusatItem `json:"riwayat_bagi_hasil"`
}

func (s *BagiHasilService) GetListBankPusat(bankID string, startTime, endTime time.Time) (*ListBagiHasilBankPusatResp, error) {
	bank, err := s.repo.GetBankByID(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank == models.BSU {
		return nil, errBadRequest("Bank bukan bertipe Bank Pusat")
	}

	bagiHasils, err := s.repo.GetBagiHasilByBankWithFilter(bankID, startTime, endTime)
	if err != nil {
		return nil, errInternal("Gagal mengambil data bagi hasil")
	}

	list := make([]ListBagiHasilBankPusatItem, 0, len(bagiHasils))
	for _, bh := range bagiHasils {
		if bh.Penjualan == nil {
			continue
		}
		list = append(list, ListBagiHasilBankPusatItem{
			BagiHasilID:      bh.BagiHasilID,
			RewardID:         bh.Penjualan.RewardID,
			NamaReward:       string(bh.Penjualan.Reward.NamaReward),
			TanggalBagiHasil: bh.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	return &ListBagiHasilBankPusatResp{
		BankID:           bank.BankID,
		NamaBank:         bank.NamaBank,
		JenisBank:        bank.JenisBank,
		RiwayatBagiHasil: list,
	}, nil
}

type DetailBagiHasilBankPusatResp struct {
	BagiHasilID            string                `json:"bagi_hasil_id"`
	PenjualanID            string                `json:"penjualan_id"`
	RewardID               int                   `json:"reward_id"`
	NamaReward             string                `json:"nama_reward"`
	Tanggal                string                `json:"tanggal"`
	GrossBSI               float64               `json:"gross_bsi,omitempty"`
	GrossBSM               float64               `json:"gross_bsm,omitempty"`
	TotalDistribusiNasabah float64               `json:"total_distribusi_nasabah"`
	SisaBagiHasil          float64               `json:"sisa_bagi_hasil"`
	DistribusiID           *string               `json:"distribusi_id"`
	Penerima               []BankPenerimaGroup   `json:"penerima"`
	NasabahBSI             []NasabahPenerimaItem `json:"nasabah_bsi,omitempty"`
	NasabahBSM             []NasabahPenerimaItem `json:"nasabah_bsm,omitempty"`
}

func (s *BagiHasilService) GetDetailBankPusat(bagiHasilID string) (*DetailBagiHasilBankPusatResp, error) {
	bh, err := s.repo.GetBagiHasilWithAll(bagiHasilID)
	if err != nil {
		return nil, errNotFound("Data bagi hasil tidak ditemukan")
	}
	if bh.Bank == nil || bh.Penjualan == nil {
		return nil, errInternal("Data bagi hasil tidak lengkap")
	}
	if bh.Bank.JenisBank == models.BSU {
		return nil, errBadRequest("Bank bukan bertipe Bank Pusat")
	}

	distribusiID := s.repo.FindDistribusiSisaID(bagiHasilID)

	resp := &DetailBagiHasilBankPusatResp{
		BagiHasilID:            bh.BagiHasilID,
		PenjualanID:            *bh.PenjualanID,
		RewardID:               bh.Penjualan.RewardID,
		NamaReward:             string(bh.Penjualan.Reward.NamaReward),
		Tanggal:                bh.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		TotalDistribusiNasabah: bh.TotalDistribusiNasabah,
		SisaBagiHasil:          bh.SisaBagiHasil,
		DistribusiID:           distribusiID,
		Penerima:               []BankPenerimaGroup{},
	}
	if bh.Bank.JenisBank == models.BSI {
		return s.detailBankPusatBSI(bh, resp)
	}

	penerimaNasabah, err := s.repo.GetPenerimaNasabahFull(bagiHasilID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data nasabah penerima")
	}
	grouped, _ := groupNasabahByBank(penerimaNasabah, nil)
	nasabahList := make([]NasabahPenerimaItem, 0, len(penerimaNasabah))
	for _, p := range penerimaNasabah {
		nasabahList = append(nasabahList, toNasabahPenerimaItem(p))
	}
	resp.GrossBSM = bh.GrossBank
	resp.Penerima = grouped
	resp.NasabahBSM = nasabahList
	return resp, nil
}

func toNasabahPenerimaItem(p models.PenerimaBagiHasil) NasabahPenerimaItem {
	nasabahID, nama := "", ""
	if p.Nasabah != nil {
		nasabahID = p.Nasabah.NasabahID
		nama = p.Nasabah.User.Nama
	}
	return NasabahPenerimaItem{
		PenerimaID:     p.PenerimaID,
		NasabahID:      nasabahID,
		NamaNasabah:    nama,
		TotalDiterima:  p.TotalDiterima,
		SatuanDiterima: p.SatuanDiterima,
	}
}

func groupNasabahByBank(penerima []models.PenerimaBagiHasil, bsuIDSet map[string]bool) (grouped []BankPenerimaGroup, langsung []NasabahPenerimaItem) {
	bankMap := make(map[string]*BankPenerimaGroup)
	var bankOrder []string
	langsung = []NasabahPenerimaItem{}

	for _, p := range penerima {
		if p.Nasabah == nil {
			continue
		}
		item := toNasabahPenerimaItem(p)
		bankID := p.Nasabah.BankID

		inGroup := bsuIDSet == nil || bsuIDSet[bankID]

		if inGroup {
			namaBank := p.Nasabah.Bank.NamaBank
			if namaBank == "" {
				namaBank = "Bank Sampah"
			}
			if _, ada := bankMap[bankID]; !ada {
				bankMap[bankID] = &BankPenerimaGroup{
					BankID:          bankID,
					NamaBank:        namaBank,
					NasabahPenerima: []NasabahPenerimaItem{},
				}
				bankOrder = append(bankOrder, bankID)
			}
			bankMap[bankID].NasabahPenerima = append(bankMap[bankID].NasabahPenerima, item)
		} else {
			langsung = append(langsung, item)
		}
	}

	grouped = make([]BankPenerimaGroup, 0, len(bankOrder))
	for _, bID := range bankOrder {
		grouped = append(grouped, *bankMap[bID])
	}
	return grouped, langsung
}
