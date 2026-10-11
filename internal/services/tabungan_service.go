package services

import (
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
)

type TabunganRepoIface interface {
	ListTabunganNasabah(nasabahID string, start, end *time.Time) ([]models.TabunganSampah, error)
	ListTabunganBank(bankID string, start, end *time.Time) ([]models.TabunganSampah, error)
	ListPencairanTabungan(tabunganIDs []string) ([]repositories.PencairanTabunganRow, error)
	ListTanggalPengangkutanCompleted(pengangkutanIDs []string) (map[string]string, error)
}

type DetailPencairan struct {
	TanggalCair time.Time `json:"tanggal_cair"`
	QtyDipakai  float64   `json:"qty_dipakai"`
	HargaItem   float64   `json:"harga_item"`
	Subtotal    float64   `json:"subtotal"`
}

type ItemTabungan struct {
	TabunganID      string            `json:"tabungan_id"`
	SourceID        string            `json:"source_id"`
	CreatedAt       time.Time         `json:"created_at"`
	Qty             float64           `json:"qty"`
	SisaQty         float64           `json:"sisa_qty"`
	Status          string            `json:"status"`
	DetailPencairan []DetailPencairan `json:"detail_pencairan"`
}

type KelompokSampah struct {
	SampahID       string         `json:"sampah_id"`
	NamaSampah     string         `json:"nama_sampah"`
	SatuanSampah   string         `json:"satuan_sampah"`
	SatuanDiterima string         `json:"satuan_diterima"`
	RewardID       int            `json:"reward_id"`
	NamaReward     string         `json:"nama_reward"`
	Tabungan       []ItemTabungan `json:"tabungan"`
}

type ItemBukuTabungan struct {
	TabunganID string   `json:"tabungan_id"`
	NamaSampah string   `json:"nama_sampah"`
	Satuan     string   `json:"satuan"`
	RewardID   int      `json:"reward_id"`
	NamaReward string   `json:"nama_reward"`
	QtySetoran float64  `json:"qty_setoran"`
	SisaQty    float64  `json:"sisa_qty"`
	HargaItem  *float64 `json:"harga_item"`
	NilaiTotal *float64 `json:"nilai_total"`
	Status     string   `json:"status"`
}

type KelompokBukuTabungan struct {
	SourceID       string             `json:"source_id"`
	TanggalSetoran string             `json:"tanggal_setoran"`
	Items          []ItemBukuTabungan `json:"items"`
}

type BukuTabunganNasabahResult struct {
	NasabahID string           `json:"nasabah_id"`
	Sampah    []KelompokSampah `json:"sampah"`
}

type BukuTabunganBSUResult struct {
	BankID       string                 `json:"bank_id"`
	Pengangkutan []KelompokBukuTabungan `json:"pengangkutan"`
}

type TabunganService interface {
	BukuTabunganNasabah(nasabahID, startDate, endDate string) (*BukuTabunganNasabahResult, error)
	BukuTabunganBSU(bankID, startDate, endDate string) (*BukuTabunganBSUResult, error)
}

type tabunganService struct {
	repo TabunganRepoIface
}

func NewTabunganService(repo TabunganRepoIface) TabunganService {
	return &tabunganService{repo: repo}
}

func (s *tabunganService) BukuTabunganNasabah(nasabahID, startDate, endDate string) (*BukuTabunganNasabahResult, error) {
	if nasabahID == "" {
		return nil, errBadRequest("nasabah_id wajib diisi")
	}

	start, end := rentangTanggal(startDate, endDate)
	tabungans, err := s.repo.ListTabunganNasabah(nasabahID, start, end)
	if err != nil {
		return nil, errInternal("Gagal mengambil data tabungan: " + err.Error())
	}

	if len(tabungans) == 0 {
		return &BukuTabunganNasabahResult{
			NasabahID: nasabahID,
			Sampah:    []KelompokSampah{},
		}, nil
	}

	pencairanMap, err := s.ambilPencairanMap(tabungans)
	if err != nil {
		return nil, err
	}

	return &BukuTabunganNasabahResult{
		NasabahID: nasabahID,
		Sampah:    susunBukuTabunganPerSampah(tabungans, pencairanMap),
	}, nil
}

func (s *tabunganService) BukuTabunganBSU(bankID, startDate, endDate string) (*BukuTabunganBSUResult, error) {
	if bankID == "" {
		return nil, errBadRequest("bsu_id wajib diisi")
	}

	start, end := rentangTanggal(startDate, endDate)
	tabungans, err := s.repo.ListTabunganBank(bankID, start, end)
	if err != nil {
		return nil, errInternal("Gagal mengambil data tabungan: " + err.Error())
	}

	if len(tabungans) == 0 {
		return &BukuTabunganBSUResult{
			BankID:       bankID,
			Pengangkutan: []KelompokBukuTabungan{},
		}, nil
	}

	pencairanMap, err := s.ambilPencairanMap(tabungans)
	if err != nil {
		return nil, err
	}

	tanggalMap, err := s.repo.ListTanggalPengangkutanCompleted(kumpulkanSourceID(tabungans))
	if err != nil {
		return nil, errInternal("Gagal mengambil tanggal pengangkutan: " + err.Error())
	}

	return &BukuTabunganBSUResult{
		BankID:       bankID,
		Pengangkutan: susunBukuTabungan(tabungans, tanggalMap, pencairanMap),
	}, nil
}

func (s *tabunganService) ambilPencairanMap(tabungans []models.TabunganSampah) (map[string][]repositories.PencairanTabunganRow, error) {
	tabunganIDs := make([]string, 0, len(tabungans))
	for _, t := range tabungans {
		tabunganIDs = append(tabunganIDs, t.TabunganID)
	}

	rows, err := s.repo.ListPencairanTabungan(tabunganIDs)
	if err != nil {
		return nil, errInternal("Gagal mengambil data pencairan: " + err.Error())
	}

	pencairanMap := make(map[string][]repositories.PencairanTabunganRow)
	for _, row := range rows {
		pencairanMap[row.TabunganID] = append(pencairanMap[row.TabunganID], row)
	}
	return pencairanMap, nil
}

func susunBukuTabunganPerSampah(
	tabungans []models.TabunganSampah,
	pencairanMap map[string][]repositories.PencairanTabunganRow,
) []KelompokSampah {
	urutan := []string{}
	kelompok := make(map[string][]models.TabunganSampah)

	for _, t := range tabungans {
		if _, ada := kelompok[t.SampahID]; !ada {
			urutan = append(urutan, t.SampahID)
		}
		kelompok[t.SampahID] = append(kelompok[t.SampahID], t)
	}

	hasil := make([]KelompokSampah, 0, len(urutan))
	for _, sampahID := range urutan {
		anggota := kelompok[sampahID]

		namaSampah, satuanSampah, satuanDiterima, namaReward := "", "", "", ""
		rewardID := 0
		if k := anggota[0].KatalogSampah; k != nil {
			namaSampah = k.Sarok.NamaSampah
			satuanSampah = string(k.Sarok.Satuan)
			satuanDiterima = string(k.Reward.Satuan)
			namaReward = string(k.Reward.NamaReward)
			rewardID = k.RewardID
		}

		items := make([]ItemTabungan, 0, len(anggota))
		for _, tab := range anggota {
			sourceID := ""
			if tab.SourceID != nil {
				sourceID = *tab.SourceID
			}

			tranches := pencairanMap[tab.TabunganID]
			detail := make([]DetailPencairan, 0, len(tranches))
			for _, tr := range tranches {
				detail = append(detail, DetailPencairan{
					TanggalCair: tr.TanggalCair,
					QtyDipakai:  tr.QtyDipakai,
					HargaItem:   tr.HargaItem,
					Subtotal:    tr.QtyDipakai * tr.HargaItem,
				})
			}

			items = append(items, ItemTabungan{
				TabunganID:      tab.TabunganID,
				SourceID:        sourceID,
				CreatedAt:       tab.CreatedAt,
				Qty:             tab.Qty,
				SisaQty:         tab.SisaQty,
				Status:          statusCair(tab.Qty, tab.SisaQty),
				DetailPencairan: detail,
			})
		}

		hasil = append(hasil, KelompokSampah{
			SampahID:       sampahID,
			NamaSampah:     namaSampah,
			SatuanSampah:   satuanSampah,
			SatuanDiterima: satuanDiterima,
			RewardID:       rewardID,
			NamaReward:     namaReward,
			Tabungan:       items,
		})
	}

	return hasil
}

func susunBukuTabungan(
	tabungans []models.TabunganSampah,
	tanggalMap map[string]string,
	pencairanMap map[string][]repositories.PencairanTabunganRow,
) []KelompokBukuTabungan {
	urutan := []string{}
	kelompok := make(map[string][]models.TabunganSampah)

	for _, t := range tabungans {
		key := ""
		if t.SourceID != nil {
			key = *t.SourceID
		}
		if _, ada := kelompok[key]; !ada {
			urutan = append(urutan, key)
		}
		kelompok[key] = append(kelompok[key], t)
	}

	hasil := make([]KelompokBukuTabungan, 0, len(urutan))
	for _, sourceID := range urutan {
		anggota := kelompok[sourceID]
		items := make([]ItemBukuTabungan, 0, len(anggota))

		for _, tab := range anggota {
			namaSampah, satuan, namaReward := "", "", ""
			rewardID := 0
			if tab.KatalogSampah != nil {
				namaSampah = tab.KatalogSampah.Sarok.NamaSampah
				satuan = string(tab.KatalogSampah.Sarok.Satuan)
				rewardID = tab.KatalogSampah.RewardID
				namaReward = string(tab.KatalogSampah.Reward.NamaReward)
			}

			var hargaItem, nilaiTotal *float64
			if tranches := pencairanMap[tab.TabunganID]; len(tranches) > 0 {
				harga := tranches[0].HargaItem
				total := 0.0
				for _, tr := range tranches {
					total += tr.QtyDipakai * tr.HargaItem
				}
				hargaItem = &harga
				nilaiTotal = &total
			}

			items = append(items, ItemBukuTabungan{
				TabunganID: tab.TabunganID,
				NamaSampah: namaSampah,
				Satuan:     satuan,
				RewardID:   rewardID,
				NamaReward: namaReward,
				QtySetoran: tab.Qty,
				SisaQty:    tab.SisaQty,
				HargaItem:  hargaItem,
				NilaiTotal: nilaiTotal,
				Status:     statusCair(tab.Qty, tab.SisaQty),
			})
		}

		hasil = append(hasil, KelompokBukuTabungan{
			SourceID:       sourceID,
			TanggalSetoran: tanggalMap[sourceID],
			Items:          items,
		})
	}

	return hasil
}

func kumpulkanSourceID(tabungans []models.TabunganSampah) []string {
	unik := make(map[string]struct{})
	for _, t := range tabungans {
		if t.SourceID != nil && *t.SourceID != "" {
			unik[*t.SourceID] = struct{}{}
		}
	}

	ids := make([]string, 0, len(unik))
	for id := range unik {
		ids = append(ids, id)
	}
	return ids
}

func statusCair(qty, sisaQty float64) string {
	switch {
	case sisaQty == 0:
		return "Cair"
	case sisaQty >= qty:
		return "Belum Cair"
	default:
		return "Cair Sebagian"
	}
}

func rentangTanggal(startDate, endDate string) (*time.Time, *time.Time) {
	var start, end *time.Time
	if t, err := time.Parse("2006-01-02", startDate); err == nil {
		start = &t
	}
	if t, err := time.Parse("2006-01-02", endDate); err == nil {
		akhirHari := t.Add(24*time.Hour - time.Second)
		end = &akhirHari
	}
	return start, end
}
