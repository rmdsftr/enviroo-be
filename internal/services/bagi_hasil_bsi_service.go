package services

import (
	"enviroo-be/internal/models"
	"math"
	"time"

	"gorm.io/gorm"
)

func (s *BagiHasilService) bsuIDs(bsiID string) ([]string, error) {
	bsuList, err := s.repo.GetBSUList(s.db, bsiID)
	if err != nil {
		return nil, errInternal("Gagal mengambil daftar BSU")
	}
	ids := make([]string, 0, len(bsuList))
	for _, bsu := range bsuList {
		ids = append(ids, bsu.BankID)
	}
	return ids, nil
}

func (s *BagiHasilService) bsuIDSet(bsiID string) (map[string]bool, error) {
	ids, err := s.bsuIDs(bsiID)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set, nil
}

func (s *BagiHasilService) previewBSI(penjualan *models.Penjualan, bankID string, detailPenjualans []models.DetailPenjualan) (*PreviewBagiHasilResp, error) {
	bsuIDs, err := s.bsuIDs(bankID)
	if err != nil {
		return nil, err
	}
	return s.hitungPreview(penjualan, detailPenjualans, append([]string{bankID}, bsuIDs...))
}

type perhitunganSisaBSU struct {
	tabunganID   string
	qtyDipakai   float64
	nilaiDipakai float64
}

func (s *BagiHasilService) submitBSI(penjualan *models.Penjualan, bankID string, detailPenjualans []models.DetailPenjualan, adminID string) (*SubmitBagiHasilResp, error) {
	bsuIDs, err := s.bsuIDs(bankID)
	if err != nil {
		return nil, err
	}
	bankIDsNasabah := append([]string{bankID}, bsuIDs...)

	var result *SubmitBagiHasilResp

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		dist, err := s.distribusikanKeNasabah(tx, penjualan, detailPenjualans, bankIDsNasabah, adminID)
		if err != nil {
			return err
		}
		perhitungan, err := s.potongStokBSU(tx, detailPenjualans, bsuIDs)
		if err != nil {
			return err
		}
		bh, err := s.simpanBagiHasil(tx, penjualan, bankID, dist.total, adminID, now)
		if err != nil {
			return err
		}
		for _, ps := range perhitungan {
			if err := s.repo.CreatePerhitunganSisa(tx, &models.PerhitunganSisa{
				BagiHasilID:  bh.BagiHasilID,
				TabunganID:   ps.tabunganID,
				QtyDipakai:   ps.qtyDipakai,
				NilaiDipakai: ps.nilaiDipakai,
			}); err != nil {
				return errInternal("Gagal menyimpan perhitungan sisa BSU")
			}
		}
		nasabahNotifs, err := s.simpanPenerimaNasabah(tx, penjualan, bh.BagiHasilID, dist, adminID, now)
		if err != nil {
			return err
		}
		if penjualan.Reward.NamaReward == models.RewardEnumBarang {
			if err := s.masukkanSisaKeBank(tx, penjualan, bankID, bh.SisaBagiHasil, adminID, now); err != nil {
				return err
			}
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

func (s *BagiHasilService) potongStokBSU(tx *gorm.DB, detailPenjualans []models.DetailPenjualan, bsuIDs []string) ([]perhitunganSisaBSU, error) {
	if len(bsuIDs) == 0 {
		return nil, nil
	}

	var perhitungan []perhitunganSisaBSU
	for _, detail := range detailPenjualans {
		tabunganBSU, err := s.repo.GetTabunganBSU(tx, detail.SampahID, bsuIDs)
		if err != nil {
			return nil, errInternal("Gagal mengambil tabungan BSU")
		}
		sisaQtyBSU := detail.Qty
		for i := range tabunganBSU {
			if sisaQtyBSU <= 0 {
				break
			}
			t := &tabunganBSU[i]
			qtyDiambil := math.Min(t.SisaQty, sisaQtyBSU)
			perhitungan = append(perhitungan, perhitunganSisaBSU{
				tabunganID:   t.TabunganID,
				qtyDipakai:   qtyDiambil,
				nilaiDipakai: qtyDiambil * detail.HargaNasabahSnapshot,
			})
			t.SisaQty -= qtyDiambil
			if err := s.repo.SaveTabungan(tx, t); err != nil {
				return nil, errInternal("Gagal update tabungan BSU")
			}
			sisaQtyBSU -= qtyDiambil
		}
	}
	return perhitungan, nil
}

func (s *BagiHasilService) detailBSI(bh *models.BagiHasil, semuaPenerima []models.PenerimaBagiHasil, resp *DetailBagiHasilResp) (*DetailBagiHasilResp, error) {
	bsuIDSet, err := s.bsuIDSet(bh.Bank.BankID)
	if err != nil {
		return nil, err
	}
	grouped, langsung := groupNasabahByBank(semuaPenerima, bsuIDSet)
	resp.NasabahLangsung = langsung
	resp.Penerima = grouped
	return resp, nil
}

func (s *BagiHasilService) detailBankPusatBSI(bh *models.BagiHasil, resp *DetailBagiHasilBankPusatResp) (*DetailBagiHasilBankPusatResp, error) {
	semuaPenerima, err := s.repo.GetPenerimaBagiHasil(bh.BagiHasilID)
	if err != nil {
		return nil, errInternal("Gagal mengambil data penerima")
	}
	bsuIDSet, err := s.bsuIDSet(bh.Bank.BankID)
	if err != nil {
		return nil, err
	}
	grouped, langsung := groupNasabahByBank(semuaPenerima, bsuIDSet)
	resp.GrossBSI = bh.GrossBank
	resp.Penerima = grouped
	resp.NasabahBSI = langsung
	return resp, nil
}
