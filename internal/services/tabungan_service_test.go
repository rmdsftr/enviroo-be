package services_test

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func tabunganKatalog(sampahID, namaSampah string) *models.KatalogSampah {
	return &models.KatalogSampah{
		SampahID: sampahID,
		RewardID: 1,
		Sarok:    models.Sampah{NamaSampah: namaSampah, SatuanID: 1, Satuan: models.Satuan{SatuanID: 1, Satuan: "kg"}},
		Reward:   models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnumRp},
	}
}

func tabunganBaris(tabunganID, sampahID, sourceID string, qty, sisaQty float64) models.TabunganSampah {
	src := sourceID
	return models.TabunganSampah{
		TabunganID:    tabunganID,
		SampahID:      sampahID,
		SourceID:      &src,
		Qty:           qty,
		SisaQty:       sisaQty,
		CreatedAt:     time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC),
		KatalogSampah: tabunganKatalog(sampahID, "Botol Plastik"),
	}
}

func tranche(tabunganID string, hari int, qtyDipakai, harga float64) repositories.PencairanTabunganRow {
	return repositories.PencairanTabunganRow{
		TabunganID:  tabunganID,
		TanggalCair: time.Date(2026, 8, hari, 14, 0, 0, 0, time.UTC),
		QtyDipakai:  qtyDipakai,
		HargaItem:   harga,
	}
}

func siapkanBukuNasabah(t *testing.T, rows []models.TabunganSampah, tranches []repositories.PencairanTabunganRow) *services.BukuTabunganNasabahResult {
	t.Helper()

	repo := new(mocks.MockTabunganRepo)
	repo.On("ListTabunganNasabah", "NSB-001", mock.Anything, mock.Anything).Return(rows, nil)
	repo.On("ListPencairanTabungan", mock.Anything).Return(tranches, nil)

	hasil, err := services.NewTabunganService(repo).BukuTabunganNasabah("NSB-001", "", "")
	assert.NoError(t, err)
	assert.NotNil(t, hasil)
	return hasil
}

func TestBukuTabunganNasabah_BelumCair(t *testing.T) {
	rows := []models.TabunganSampah{tabunganBaris("TAB-001", "SMP-001", "STR-001", 10, 10)}

	hasil := siapkanBukuNasabah(t, rows, nil)

	assert.Len(t, hasil.Sampah, 1)
	kelompok := hasil.Sampah[0]
	assert.Equal(t, "SMP-001", kelompok.SampahID)
	assert.Equal(t, "Botol Plastik", kelompok.NamaSampah)
	assert.Equal(t, "kg", kelompok.SatuanSampah)
	assert.Equal(t, "Rp", kelompok.SatuanDiterima)
	assert.NotNil(t, kelompok.Antrian, "antrian harus [] di JSON, bukan null")
	assert.Empty(t, kelompok.Antrian)

	assert.Len(t, kelompok.Tabungan, 1)
	item := kelompok.Tabungan[0]
	assert.Equal(t, "Belum Cair", item.Status)
	assert.Equal(t, "STR-001", item.SourceID)
	assert.Empty(t, item.DetailPencairan)
}

func TestBukuTabunganNasabah_CairSebagianPakaiQtyDipakai(t *testing.T) {
	rows := []models.TabunganSampah{tabunganBaris("TAB-001", "SMP-001", "STR-001", 10, 6)}
	tranches := []repositories.PencairanTabunganRow{tranche("TAB-001", 5, 4, 2000)}

	hasil := siapkanBukuNasabah(t, rows, tranches)

	item := hasil.Sampah[0].Tabungan[0]
	assert.Equal(t, "Cair Sebagian", item.Status)
	assert.Len(t, item.DetailPencairan, 1)
	assert.Equal(t, 4.0, item.DetailPencairan[0].QtyDipakai)
	assert.Equal(t, 2000.0, item.DetailPencairan[0].HargaItem)
	assert.Equal(t, 8000.0, item.DetailPencairan[0].Subtotal)
}

func TestBukuTabunganNasabah_BeberapaTrancheHargaBerbeda(t *testing.T) {
	rows := []models.TabunganSampah{tabunganBaris("TAB-001", "SMP-001", "STR-001", 10, 0)}
	tranches := []repositories.PencairanTabunganRow{
		tranche("TAB-001", 5, 4, 2000),
		tranche("TAB-001", 20, 6, 2500),
	}

	hasil := siapkanBukuNasabah(t, rows, tranches)

	item := hasil.Sampah[0].Tabungan[0]
	assert.Equal(t, "Cair", item.Status)
	assert.Len(t, item.DetailPencairan, 2)
	assert.Equal(t, 8000.0, item.DetailPencairan[0].Subtotal)
	assert.Equal(t, 15000.0, item.DetailPencairan[1].Subtotal)
}

func TestBukuTabunganNasabah_DuaTabunganSampahSama(t *testing.T) {
	rows := []models.TabunganSampah{
		tabunganBaris("TAB-001", "SMP-001", "STR-001", 10, 0),
		tabunganBaris("TAB-002", "SMP-001", "STR-002", 5, 5),
	}
	tranches := []repositories.PencairanTabunganRow{tranche("TAB-001", 5, 10, 2000)}

	hasil := siapkanBukuNasabah(t, rows, tranches)

	assert.Len(t, hasil.Sampah, 1)
	assert.Len(t, hasil.Sampah[0].Tabungan, 2)

	pertama := hasil.Sampah[0].Tabungan[0]
	assert.Equal(t, "TAB-001", pertama.TabunganID)
	assert.Len(t, pertama.DetailPencairan, 1)
	assert.Equal(t, 20000.0, pertama.DetailPencairan[0].Subtotal)

	kedua := hasil.Sampah[0].Tabungan[1]
	assert.Equal(t, "TAB-002", kedua.TabunganID)
	assert.Equal(t, "Belum Cair", kedua.Status)
	assert.Empty(t, kedua.DetailPencairan)
}

func TestBukuTabunganNasabah_SampahBerbedaJadiKelompokTerpisah(t *testing.T) {
	rows := []models.TabunganSampah{
		tabunganBaris("TAB-001", "SMP-001", "STR-001", 10, 10),
		tabunganBaris("TAB-002", "SMP-002", "STR-001", 3, 3),
	}

	hasil := siapkanBukuNasabah(t, rows, nil)

	assert.Len(t, hasil.Sampah, 2)
	assert.Equal(t, "SMP-001", hasil.Sampah[0].SampahID)
	assert.Equal(t, "SMP-002", hasil.Sampah[1].SampahID)
}

func TestBukuTabunganNasabah_TanpaTabunganMengembalikanSliceKosong(t *testing.T) {
	repo := new(mocks.MockTabunganRepo)
	repo.On("ListTabunganNasabah", "NSB-001", mock.Anything, mock.Anything).
		Return([]models.TabunganSampah{}, nil)

	hasil, err := services.NewTabunganService(repo).BukuTabunganNasabah("NSB-001", "", "")

	assert.NoError(t, err)
	assert.NotNil(t, hasil.Sampah)
	assert.Empty(t, hasil.Sampah)
	repo.AssertNotCalled(t, "ListPencairanTabungan", mock.Anything)
}

func TestBukuTabunganBSU_BentukTetapPerSourceID(t *testing.T) {
	rows := []models.TabunganSampah{tabunganBaris("TAB-001", "SMP-001", "PGK-001", 10, 0)}
	tranches := []repositories.PencairanTabunganRow{
		tranche("TAB-001", 5, 4, 2000),
		tranche("TAB-001", 20, 6, 2500),
	}

	repo := new(mocks.MockTabunganRepo)
	repo.On("ListTabunganBank", "BSU-001", mock.Anything, mock.Anything).Return(rows, nil)
	repo.On("ListPencairanTabungan", mock.Anything).Return(tranches, nil)
	repo.On("ListTanggalPengangkutanCompleted", mock.Anything).
		Return(map[string]string{"PGK-001": "2026-08-20T14:00:00+07:00"}, nil)

	hasil, err := services.NewTabunganService(repo).BukuTabunganBSU("BSU-001", "", "")
	assert.NoError(t, err)

	assert.Len(t, hasil.Pengangkutan, 1)
	kelompok := hasil.Pengangkutan[0]
	assert.Equal(t, "PGK-001", kelompok.SourceID)
	assert.Equal(t, "2026-08-20T14:00:00+07:00", kelompok.TanggalSetoran)

	assert.Len(t, kelompok.Items, 1)
	item := kelompok.Items[0]
	assert.Equal(t, 2000.0, *item.HargaItem)
	assert.Equal(t, 23000.0, *item.NilaiTotal)
}

func TestBukuTabunganBSU_BelumCairHargaNil(t *testing.T) {
	rows := []models.TabunganSampah{tabunganBaris("TAB-001", "SMP-001", "PGK-001", 10, 10)}

	repo := new(mocks.MockTabunganRepo)
	repo.On("ListTabunganBank", "BSU-001", mock.Anything, mock.Anything).Return(rows, nil)
	repo.On("ListPencairanTabungan", mock.Anything).
		Return([]repositories.PencairanTabunganRow{}, nil)
	repo.On("ListTanggalPengangkutanCompleted", mock.Anything).
		Return(map[string]string{}, nil)

	hasil, err := services.NewTabunganService(repo).BukuTabunganBSU("BSU-001", "", "")
	assert.NoError(t, err)

	item := hasil.Pengangkutan[0].Items[0]
	assert.Nil(t, item.HargaItem)
	assert.Nil(t, item.NilaiTotal)
	assert.Equal(t, "Belum Cair", item.Status)
}

func tabKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func tabBank(jenis models.JenisBank) *models.BankSampah {
	return &models.BankSampah{BankID: "BNK-001", NamaBank: "Bank Uji", JenisBank: jenis, IsActive: true}
}

func tabunganBarisBank(tabunganID, sampahID string, qty, sisaQty float64) models.TabunganSampah {
	baris := tabunganBaris(tabunganID, sampahID, "STR-001", qty, sisaQty)
	bankID := "BNK-001"
	baris.BankID = &bankID
	return baris
}

func siapkanRepoAntrian(rows []models.TabunganSampah) *mocks.MockTabunganRepo {
	repo := new(mocks.MockTabunganRepo)
	repo.On("ListTabunganNasabah", "NSB-001", mock.Anything, mock.Anything).Return(rows, nil)
	repo.On("ListPencairanTabungan", mock.Anything).Return(nil, nil)
	return repo
}

func TestBukuTabunganNasabah_AntrianBSM(t *testing.T) {
	rows := []models.TabunganSampah{
		tabunganBarisBank("TAB-004", "SMP-001", 1.5, 1.5),
		tabunganBarisBank("TAB-010", "SMP-002", 4, 0),
	}
	tanggal := func(hari int) time.Time { return time.Date(2026, 10, hari, 8, 0, 0, 0, time.UTC) }
	antrian := []repositories.AntrianSampahRow{
		{SampahID: "SMP-001", Nomor: 1, NasabahID: "NSB-ANI", NamaNasabah: "Ani", SisaQty: 2, TanggalSetor: tanggal(1)},
		{SampahID: "SMP-001", Nomor: 2, NasabahID: "NSB-BUDI", NamaNasabah: "Budi", SisaQty: 3, TanggalSetor: tanggal(2)},
		{SampahID: "SMP-001", Nomor: 3, NasabahID: "NSB-BUDI", NamaNasabah: "Budi", SisaQty: 1, TanggalSetor: tanggal(3)},
		{SampahID: "SMP-001", Nomor: 4, NasabahID: "NSB-001", NamaNasabah: "Levi", SisaQty: 1.5, TanggalSetor: tanggal(4)},
	}

	repo := siapkanRepoAntrian(rows)
	repo.On("FindBank", "BNK-001").Return(tabBank(models.BSM), nil)
	repo.On("ListAntrianSampaiNasabah", "BNK-001", "NSB-001").Return(antrian, nil)

	hasil, err := services.NewTabunganService(repo).BukuTabunganNasabah("NSB-001", "", "")

	assert.NoError(t, err)
	assert.Len(t, hasil.Sampah, 2)

	botol := hasil.Sampah[0].Antrian
	if assert.Len(t, botol, 4) {
		for i, item := range botol {
			assert.Equal(t, i+1, item.Nomor)
		}
		assert.Equal(t, "NSB-BUDI", botol[1].NasabahID)
		assert.Equal(t, "NSB-BUDI", botol[2].NasabahID, "satu baris = satu tabungan, nasabah lain boleh muncul berkali-kali")

		terakhir := botol[3]
		assert.Equal(t, "NSB-001", terakhir.NasabahID)
		assert.Equal(t, "Levi", terakhir.NamaNasabah)
		assert.Equal(t, 1.5, terakhir.SisaQty)
		assert.Equal(t, tanggal(4), terakhir.TanggalSetor)
	}

	assert.NotNil(t, hasil.Sampah[1].Antrian, "sampah yang sudah cair semua tetap [] di JSON, bukan null")
	assert.Empty(t, hasil.Sampah[1].Antrian)
}

func TestBukuTabunganNasabah_AntrianBukanBSM(t *testing.T) {
	repo := siapkanRepoAntrian([]models.TabunganSampah{tabunganBarisBank("TAB-001", "SMP-001", 10, 10)})
	repo.On("FindBank", "BNK-001").Return(tabBank(models.BSI), nil)

	hasil, err := services.NewTabunganService(repo).BukuTabunganNasabah("NSB-001", "", "")

	assert.NoError(t, err)
	assert.NotNil(t, hasil.Sampah[0].Antrian)
	assert.Empty(t, hasil.Sampah[0].Antrian)
	repo.AssertNotCalled(t, "ListAntrianSampaiNasabah", mock.Anything, mock.Anything)
}

func TestBukuTabunganNasabah_AntrianCekBankError(t *testing.T) {
	repo := siapkanRepoAntrian([]models.TabunganSampah{tabunganBarisBank("TAB-001", "SMP-001", 10, 10)})
	repo.On("FindBank", "BNK-001").Return(nil, errors.New("db error"))

	hasil, err := services.NewTabunganService(repo).BukuTabunganNasabah("NSB-001", "", "")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, tabKode(t, err))
	repo.AssertNotCalled(t, "ListAntrianSampaiNasabah", mock.Anything, mock.Anything)
}

func TestBukuTabunganNasabah_AntrianQueryError(t *testing.T) {
	repo := siapkanRepoAntrian([]models.TabunganSampah{tabunganBarisBank("TAB-001", "SMP-001", 10, 10)})
	repo.On("FindBank", "BNK-001").Return(tabBank(models.BSM), nil)
	repo.On("ListAntrianSampaiNasabah", "BNK-001", "NSB-001").Return(nil, errors.New("db error"))

	hasil, err := services.NewTabunganService(repo).BukuTabunganNasabah("NSB-001", "", "")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, tabKode(t, err))
}
