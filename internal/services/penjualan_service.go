package services

import (
	"enviroo-be/internal/idgen"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

type PenjualanRepoIface interface {
	GetBankByID(bankID string) (*models.BankSampah, error)
	GetRewardByID(db *gorm.DB, rewardID int) (*models.Reward, error)
	GetPersenNasabah(db *gorm.DB, bankID string, rewardID int) (float64, error)
	GetKatalogSampah(db *gorm.DB, sampahID string) (*models.KatalogSampah, error)
	GetStokSampah(db *gorm.DB, bankID, sampahID string) (*models.StokSampah, error)
	GetSchemaHarga(db *gorm.DB, sampahID string, level models.LevelUser) (*models.SchemaHargaSampah, error)
	CreateSchemaHarga(db *gorm.DB, s *models.SchemaHargaSampah) error
	UpdateSchemaHarga(db *gorm.DB, s *models.SchemaHargaSampah) error
	CreateHistoryHarga(db *gorm.DB, h *models.KatalogSampahHistory) error
	DecrStokSampah(db *gorm.DB, bankID, sampahID string, newStok float64) error
	CreatePenjualan(db *gorm.DB, p *models.Penjualan) error
	CreateDetailPenjualan(db *gorm.DB, details []models.DetailPenjualan) error
	GetRiwayatPenjualan(bankID string, start, end *time.Time) ([]repositories.RiwayatPenjualanRow, error)
	GetPenjualan(penjualanID string) (*models.Penjualan, error)
	GetDetailItems(penjualanID string) ([]models.DetailPenjualan, error)
	GetAdminNama(adminID string) string
}

type PenjualanService struct {
	db        *gorm.DB
	repo      PenjualanRepoIface
	mitraRepo MitraRepoIface
}

func NewPenjualanService(db *gorm.DB, repo PenjualanRepoIface, mitraRepo MitraRepoIface) *PenjualanService {
	return &PenjualanService{db: db, repo: repo, mitraRepo: mitraRepo}
}

type ItemSampahDijual struct {
	SampahID  string  `json:"sampah_id" binding:"required"`
	Qty       float64 `json:"qty" binding:"required"`
	HargaJual float64 `json:"harga_jual" binding:"required"`
}

type DetailKalkulasiItem struct {
	SampahID             string  `json:"sampah_id"`
	NamaSampah           string  `json:"nama_sampah"`
	Satuan               models.SatuanEnum  `json:"satuan"`
	Qty                  float64 `json:"qty"`
	HargaJual            float64 `json:"harga_jual"`
	Subtotal             float64 `json:"subtotal"`
	HargaNasabahSnapshot float64 `json:"harga_nasabah_snapshot"`
}

type PreviewPenjualanResp struct {
	TotalPenjualan float64                 `json:"total_penjualan"`
	Satuan         models.SatuanRewardEnum `json:"satuan"`
	PersenNasabah  float64                 `json:"persen_nasabah"`
	DetailItems    []DetailKalkulasiItem   `json:"detail_items"`
}

type SubmitPenjualanResp struct {
	PenjualanID     string                     `json:"penjualan_id"`
	TotalPenjualan  float64                    `json:"total_penjualan"`
	Satuan          models.SatuanRewardEnum    `json:"satuan"`
	StatusBagiHasil models.StatusBagiHasilEnum `json:"status_bagi_hasil"`
}

type RiwayatPenjualanItem struct {
	PenjualanID     string                     `json:"penjualan_id"`
	MitraID         string                     `json:"mitra_id"`
	NamaMitra       string                     `json:"nama_mitra"`
	TotalItem       int                        `json:"total_item"`
	TotalPenjualan  float64                    `json:"total_penjualan"`
	NamaReward      string                     `json:"nama_reward"`
	SatuanReward    models.SatuanRewardEnum    `json:"satuan_reward"`
	BuktiFoto       string                     `json:"bukti_foto"`
	CreatedAt       time.Time                  `json:"created_at"`
	AdminName       string                     `json:"admin_name"`
	StatusBagiHasil models.StatusBagiHasilEnum `json:"status_bagi_hasil"`
}

type SampahDetailItem struct {
	SampahID             string            `json:"sampah_id"`
	NamaSampah           string            `json:"nama_sampah"`
	Satuan               models.SatuanEnum `json:"satuan"`
	Qty                  float64           `json:"qty"`
	HargaJual            float64           `json:"harga_jual"`
	SubtotalPenjualan    float64           `json:"subtotal_penjualan"`
	HargaNasabahSnapshot float64           `json:"harga_nasabah_snapshot"`
}

type DetailPenjualanResp struct {
	PenjualanID     string                     `json:"penjualan_id"`
	BankID          string                     `json:"bank_id"`
	NamaBank        string                     `json:"nama_bank"`
	MitraID         string                     `json:"mitra_id"`
	NamaMitra       string                     `json:"nama_mitra"`
	TotalItem       int                        `json:"total_item"`
	TotalPenjualan  float64                    `json:"total_penjualan"`
	SatuanReward    models.SatuanRewardEnum    `json:"satuan_reward"`
	NamaReward      models.RewardEnum          `json:"nama_reward"`
	BuktiFoto       string                     `json:"bukti_foto"`
	CreatedAt       time.Time                  `json:"created_at"`
	AdminName       string                     `json:"admin_name"`
	StatusBagiHasil models.StatusBagiHasilEnum `json:"status_bagi_hasil"`
	ItemsSampah     []SampahDetailItem         `json:"items_sampah"`
}

func satuanFromReward(r *models.Reward) models.SatuanRewardEnum {
	if r.NamaReward == models.RewardEnumBarang {
		return models.SatuanRewardEnumPoin
	}
	return models.SatuanRewardEnumRp
}

func (s *PenjualanService) hitungKalkulasi(
	db *gorm.DB,
	bankID string,
	rewardID int,
	items []ItemSampahDijual,
) (*PreviewPenjualanResp, error) {
	persenNasabah, err := s.repo.GetPersenNasabah(db, bankID, rewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errBadRequest("Bank belum mengatur bagi hasil untuk jenis insentif ini. Atur dulu di menu nilai reward sebelum melakukan penjualan")
		}
		return nil, errInternal("Gagal mengambil persentase bagi hasil: " + err.Error())
	}

	reward, err := s.repo.GetRewardByID(db, rewardID)
	if err != nil {
		return nil, fmt.Errorf("reward tidak ditemukan")
	}
	satuan := satuanFromReward(reward)

	var totalPenjualan float64
	detailItems := make([]DetailKalkulasiItem, 0, len(items))

	for _, item := range items {
		sampah, err := s.repo.GetKatalogSampah(db, item.SampahID)
		if err != nil {
			return nil, fmt.Errorf("sampah tidak ditemukan: %s", item.SampahID)
		}

		stok, err := s.repo.GetStokSampah(db, bankID, item.SampahID)
		if err != nil {
			return nil, fmt.Errorf("stok sampah tidak ada di bank: %s", sampah.Sarok.NamaSampah)
		}
		if stok.Stok < item.Qty {
			return nil, fmt.Errorf("stok %s tidak mencukupi (stok: %s, dibutuhkan: %s)",
				sampah.Sarok.NamaSampah, utils.FormatFloat(stok.Stok), utils.FormatFloat(item.Qty))
		}

		subtotal := item.HargaJual * item.Qty
		totalPenjualan += subtotal

		detailItems = append(detailItems, DetailKalkulasiItem{
			SampahID:             item.SampahID,
			NamaSampah:           sampah.Sarok.NamaSampah,
			Satuan:               sampah.Sarok.Satuan,
			Qty:                  item.Qty,
			HargaJual:            item.HargaJual,
			Subtotal:             subtotal,
			HargaNasabahSnapshot: item.HargaJual * (persenNasabah / 100),
		})
	}

	return &PreviewPenjualanResp{
		TotalPenjualan: totalPenjualan,
		Satuan:         satuan,
		PersenNasabah:  persenNasabah,
		DetailItems:    detailItems,
	}, nil
}

func (s *PenjualanService) PreviewPenjualan(bankID string, rewardID int, items []ItemSampahDijual) (*PreviewPenjualanResp, error) {
	bank, err := s.repo.GetBankByID(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank == models.BSU {
		return nil, errForbidden("BSU tidak diizinkan melakukan penjualan eksternal. Hanya BSI/BSM.")
	}

	result, err := s.hitungKalkulasi(s.db, bankID, rewardID, items)
	if err != nil {
		return nil, errBadRequest(err.Error())
	}
	return result, nil
}

func (s *PenjualanService) SubmitPenjualan(
	bankID, adminID string,
	rewardID int,
	mitraID, namaMitra, fotoURL string,
	items []ItemSampahDijual,
) (*SubmitPenjualanResp, error) {
	bank, err := s.repo.GetBankByID(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank == models.BSU {
		return nil, errForbidden("BSU tidak diizinkan melakukan penjualan eksternal. Hanya BSI/BSM.")
	}

	reward, err := s.repo.GetRewardByID(s.db, rewardID)
	if err != nil {
		return nil, errBadRequest("Reward tidak ditemukan")
	}
	satuan := satuanFromReward(reward)
	persenNasabah, err := s.repo.GetPersenNasabah(s.db, bankID, rewardID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errBadRequest("Bank belum mengatur bagi hasil untuk jenis insentif ini. Atur dulu di menu nilai reward sebelum melakukan penjualan")
		}
		return nil, errInternal("Gagal mengambil persentase bagi hasil: " + err.Error())
	}

	var result *SubmitPenjualanResp

	txErr := s.db.Transaction(func(tx *gorm.DB) error {
		penjualanID, err := idgen.GeneratePenjualanID(tx, rewardID, bankID)
		if err != nil {
			return errInternal("Gagal generate penjualan ID: " + err.Error())
		}
		var totalPenjualan float64
		detailPenjualans := make([]models.DetailPenjualan, 0, len(items))

		for _, item := range items {
			sampah, err := s.repo.GetKatalogSampah(tx, item.SampahID)
			if err != nil {
				return errBadRequest("Sampah tidak ditemukan: " + item.SampahID)
			}

			stok, err := s.repo.GetStokSampah(tx, bankID, item.SampahID)
			if err != nil {
				return errBadRequest("Stok sampah tidak ada di bank: " + sampah.Sarok.NamaSampah)
			}
			if stok.Stok < item.Qty {
				return errBadRequest(fmt.Sprintf("Stok %s tidak mencukupi (stok: %s, dibutuhkan: %s)",
					sampah.Sarok.NamaSampah, utils.FormatFloat(stok.Stok), utils.FormatFloat(item.Qty)))
			}

			schema, schemaErr := s.repo.GetSchemaHarga(tx, item.SampahID, models.LevelEksternal)
			if schemaErr != nil {
				rewardSampah, _ := s.repo.GetRewardByID(tx, sampah.RewardID)
				schemaID, genErr := idgen.GenerateSchemaHargaID(item.SampahID, models.LevelEksternal)
				if genErr != nil {
					return errInternal("Gagal generate schema harga ID: " + genErr.Error())
				}
				newSchema := &models.SchemaHargaSampah{
					SchemaID:  schemaID,
					SampahID:  item.SampahID,
					LevelUser: models.LevelEksternal,
					Harga:     item.HargaJual,
				}
				if rewardSampah != nil {
					newSchema.SatuanReward = models.SatuanRewardEnum(rewardSampah.Satuan)
				}
				if err := s.repo.CreateSchemaHarga(tx, newSchema); err != nil {
					return errInternal("Gagal membuat schema harga: " + err.Error())
				}
			} else if schema.Harga != item.HargaJual {
				historyID, genErr := idgen.GenerateHistorySampahID(tx, schema.SchemaID)
				if genErr != nil {
					return errInternal("Gagal generate history sampah ID: " + genErr.Error())
				}
				history := &models.KatalogSampahHistory{
					HistorySampahID: historyID,
					SchemaID:        schema.SchemaID,
					HargaLama:       schema.Harga,
					HargaBaru:       item.HargaJual,
					ChangedBy:       adminID,
				}
				if err := s.repo.CreateHistoryHarga(tx, history); err != nil {
					return errInternal("Gagal mencatat history harga: " + err.Error())
				}
				schema.Harga = item.HargaJual
				if err := s.repo.UpdateSchemaHarga(tx, schema); err != nil {
					return errInternal("Gagal update schema harga: " + err.Error())
				}
			}

			if err := s.repo.DecrStokSampah(tx, bankID, item.SampahID, stok.Stok-item.Qty); err != nil {
				return errInternal("Gagal mengurangi stok sampah")
			}

			subtotal := item.HargaJual * item.Qty
			totalPenjualan += subtotal

			detailPenjualans = append(detailPenjualans, models.DetailPenjualan{
				PenjualanID:          penjualanID,
				SampahID:             item.SampahID,
				Qty:                  item.Qty,
				HargaJual:            item.HargaJual,
				SubtotalPenjualan:    subtotal,
				HargaNasabahSnapshot: item.HargaJual * (persenNasabah / 100),
			})
		}

		mitra, err := resolveOrCreateMitra(tx, s.mitraRepo, bankID, mitraID, namaMitra)
		if err != nil {
			return err
		}

		penjualan := &models.Penjualan{
			PenjualanID:     penjualanID,
			BankID:          bankID,
			RewardID:        rewardID,
			MitraID:         mitra.MitraID,
			TotalItem:       len(items),
			TotalPenjualan:  totalPenjualan,
			SatuanReward:    satuan,
			SoldBy:          adminID,
			BuktiFoto:       fotoURL,
			CreatedAt:       time.Now(),
			StatusBagiHasil: models.BagiHasilPending,
		}
		if err := s.repo.CreatePenjualan(tx, penjualan); err != nil {
			return errInternal("Gagal menyimpan data penjualan: " + err.Error())
		}
		if err := s.repo.CreateDetailPenjualan(tx, detailPenjualans); err != nil {
			return errInternal("Gagal menyimpan detail penjualan: " + err.Error())
		}

		result = &SubmitPenjualanResp{
			PenjualanID:     penjualanID,
			TotalPenjualan:  totalPenjualan,
			Satuan:          satuan,
			StatusBagiHasil: models.BagiHasilPending,
		}
		return nil
	})

	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

func (s *PenjualanService) GetRiwayat(bankID, startDate, endDate string) ([]RiwayatPenjualanItem, error) {
	bank, err := s.repo.GetBankByID(bankID)
	if err != nil {
		return nil, errNotFound("Bank tidak ditemukan")
	}
	if bank.JenisBank != models.BSI && bank.JenisBank != models.BSM {
		return nil, errForbidden("Fitur ini hanya untuk BSI dan BSM")
	}

	start, end := rentangTanggal(startDate, endDate)
	rows, err := s.repo.GetRiwayatPenjualan(bankID, start, end)
	if err != nil {
		return nil, errInternal("Gagal mengambil riwayat penjualan")
	}

	riwayat := make([]RiwayatPenjualanItem, 0, len(rows))
	for _, row := range rows {
		riwayat = append(riwayat, RiwayatPenjualanItem{
			PenjualanID:     row.PenjualanID,
			MitraID:         row.MitraID,
			NamaMitra:       row.NamaMitra,
			TotalItem:       row.TotalItem,
			TotalPenjualan:  row.TotalPenjualan,
			NamaReward:      row.NamaReward,
			SatuanReward:    row.SatuanReward,
			BuktiFoto:       row.BuktiFoto,
			CreatedAt:       row.CreatedAt,
			AdminName:       row.AdminName,
			StatusBagiHasil: row.StatusBagiHasil,
		})
	}
	return riwayat, nil
}

func (s *PenjualanService) GetDetail(penjualanID string) (*DetailPenjualanResp, error) {
	penjualan, err := s.repo.GetPenjualan(penjualanID)
	if err != nil {
		return nil, errNotFound("Data penjualan tidak ditemukan")
	}
	if penjualan.BankSampah.JenisBank != models.BSI && penjualan.BankSampah.JenisBank != models.BSM {
		return nil, errForbidden("Fitur ini hanya untuk BSI dan BSM")
	}

	rawDetails, err := s.repo.GetDetailItems(penjualanID)
	if err != nil {
		return nil, errInternal("Gagal mengambil detail item")
	}

	items := make([]SampahDetailItem, 0, len(rawDetails))
	for _, d := range rawDetails {
		namaSampah := ""
		if d.Sampah.Sarok.NamaSampah != "" {
			namaSampah = d.Sampah.Sarok.NamaSampah
		}
		items = append(items, SampahDetailItem{
			SampahID:             d.SampahID,
			NamaSampah:           namaSampah,
			Satuan:               d.Sampah.Sarok.Satuan,
			Qty:                  d.Qty,
			HargaJual:            d.HargaJual,
			SubtotalPenjualan:    d.SubtotalPenjualan,
			HargaNasabahSnapshot: d.HargaNasabahSnapshot,
		})
	}

	adminName := s.repo.GetAdminNama(penjualan.SoldBy)

	return &DetailPenjualanResp{
		PenjualanID:     penjualan.PenjualanID,
		BankID:          penjualan.BankID,
		NamaBank:        penjualan.BankSampah.NamaBank,
		MitraID:         penjualan.MitraID,
		NamaMitra:       penjualan.MitraPengepul.NamaMitra,
		TotalItem:       penjualan.TotalItem,
		TotalPenjualan:  penjualan.TotalPenjualan,
		SatuanReward:    penjualan.SatuanReward,
		NamaReward:      penjualan.Reward.NamaReward,
		BuktiFoto:       penjualan.BuktiFoto,
		CreatedAt:       penjualan.CreatedAt,
		AdminName:       adminName,
		StatusBagiHasil: penjualan.StatusBagiHasil,
		ItemsSampah:     items,
	}, nil
}
