package services_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"
	"enviroo-be/internal/utils"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func pBankID(s string) *string    { return &s }
func pNasabahID(s string) *string { return &s }
func pRewardID(i int) *int        { return &i }

func validDeadlineStr() string {
	return time.Now().Add(2 * time.Hour).Format(time.RFC3339)
}

func nasabahMilik(nasabahID, userID, bankID string) *models.Nasabah {
	return &models.Nasabah{
		NasabahID:     nasabahID,
		UserID:        userID,
		BankID:        bankID,
		NomorRekening: "RK-001",
		StatusNasabah: models.Aktif,
	}
}

func penarikanNasabah() *models.Nasabah {
	return &models.Nasabah{
		NasabahID:     "NSB-001",
		BankID:        "BSM-001",
		UserID:        "USR-NSB-001",
		StatusNasabah: models.Aktif,
	}
}

func penarikanRewardUang() *models.Reward {
	return &models.Reward{
		RewardID:   1,
		NamaReward: models.RewardEnumUang,
		Satuan:     models.SatuanRewardEnumRp,
	}
}

func penarikanRewardBarang() *models.Reward {
	return &models.Reward{
		RewardID:   2,
		NamaReward: models.RewardEnumBarang,
		Satuan:     models.SatuanRewardEnumPoin,
	}
}

func penarikanSaldo(nominal float64) *models.SaldoRekening {
	return &models.SaldoRekening{
		RekeningID:         "RK-001",
		NominalSaldo:       nominal,
		SatuanNominalSaldo: models.SatuanRewardEnumRp,
	}
}

func pnrKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func pnrFloat(f float64) *float64 { return &f }

// Syarat saldo diisi 0 (tanpa minimal) supaya test yang fokus ke syarat waktu tidak terganggu.
func pnrSetting(minWaktu int, satuan models.SatuanWaktuEnum) *models.SettingPenarikan {
	return &models.SettingPenarikan{SettingID: "SPBSM-001", BankID: "BSM-001", MinWaktu: &minWaktu, SatuanWaktu: &satuan,
		MinRupiah: pnrFloat(0), MinPoin: pnrFloat(0)}
}

// Nasabah NSB-001 sudah memenuhi syarat: setting 1 bulan, setoran pertama 3 bulan lalu.
func penarikanBolehMenarik(repo *mocks.MockPenarikanRepo) {
	pertama := time.Now().AddDate(0, -3, 0)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(&pertama, nil)
}

func newPenarikanSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockPenarikanRepo, *mocks.MockNotifSvc, services.PenarikanService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockPenarikanRepo)
	notif := new(mocks.MockNotifSvc)
	svc := services.NewPenarikanService(gdb, repo, notif)

	repo.On("FindNasabahWithUser", mock.Anything).
		Return(nil, errors.New("tidak distub")).Maybe()

	return dbMock, repo, notif, svc
}

func TestFormatPenarikanNilai_Rp(t *testing.T) {
	assert.Equal(t, "Rp50000", services.FormatPenarikanNilai(50000, models.SatuanRewardEnumRp))
}

func TestFormatPenarikanNilai_Poin(t *testing.T) {
	assert.Equal(t, "150 poin", services.FormatPenarikanNilai(150, models.SatuanRewardEnumPoin))
}

func TestFormatPenarikanNilai_Default(t *testing.T) {
	assert.Equal(t, "3.50 kg", services.FormatPenarikanNilai(3.5, "kg"))
}

func TestPreviewPenarikan_NasabahTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-XXX").Return(nil, errors.New("not found"))

	result, err := svc.Preview("NSB-XXX", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 10000})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak ditemukan")
}

func TestPreviewPenarikan_RewardTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 99).Return(nil, errors.New("not found"))

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 99, NominalPenarikan: 10000})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reward")
}

func TestPreviewPenarikan_ValidasiNominalNol(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 0})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Nominal")
}

func TestPreviewPenarikan_ValidasiBarangTanpaItem(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 2})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Item produk")
}

func TestPreviewPenarikan_SaldoTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 1).Return(nil, errors.New("not found"))

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 50000})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Rekening saldo")
}

func TestPreviewPenarikan_Uang_SaldoCukup(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 1).Return(penarikanSaldo(200000), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 150000})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.SaldoCukup)
	assert.Equal(t, float64(200000), result.SaldoSekarang)
	assert.Equal(t, float64(50000), result.SaldoSetelah)
	assert.Equal(t, float64(150000), result.NominalPenarikan)
}

func TestPreviewPenarikan_Uang_SaldoKurang(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 1).Return(penarikanSaldo(50000), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 100000})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.SaldoCukup)
	assert.Equal(t, float64(-50000), result.SaldoSetelah)
}

func TestPreviewPenarikan_Barang_StokKurang(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	katalog := &models.KatalogBarang{
		ProdukID:     "SMB-001",
		NilaiPoin:     50,
		MasterBarang: models.Barang{NamaBarang: "Beras"},
	}
	stok := &models.StokBarang{ProdukID: "SMB-001", BankID: "BSM-001", Stok: 1}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 2).Return(
		&models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 500, SatuanNominalSaldo: models.SatuanRewardEnumPoin}, nil,
	)
	repo.On("FindProduk", "SMB-001", "BSM-001").Return(katalog, nil)
	repo.On("FindStokBarang", "SMB-001", "BSM-001").Return(stok, nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{
		RewardID:    2,
		ItemBarang: []services.BarangItemReq{{ProdukID: "SMB-001", Qty: 5}},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Stok")
}

func TestPreviewPenarikan_Barang_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	katalog1 := &models.KatalogBarang{ProdukID: "SMB-001", NilaiPoin: 50, MasterBarang: models.Barang{NamaBarang: "Beras"}}
	katalog2 := &models.KatalogBarang{ProdukID: "SMB-002", NilaiPoin: 30, MasterBarang: models.Barang{NamaBarang: "Minyak"}}
	stok1 := &models.StokBarang{ProdukID: "SMB-001", Stok: 10}
	stok2 := &models.StokBarang{ProdukID: "SMB-002", Stok: 10}
	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 200, SatuanNominalSaldo: models.SatuanRewardEnumPoin}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 2).Return(saldo, nil)
	repo.On("FindProduk", "SMB-001", "BSM-001").Return(katalog1, nil)
	repo.On("FindProduk", "SMB-002", "BSM-001").Return(katalog2, nil)
	repo.On("FindStokBarang", "SMB-001", "BSM-001").Return(stok1, nil)
	repo.On("FindStokBarang", "SMB-002", "BSM-001").Return(stok2, nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{
		RewardID: 2,
		ItemBarang: []services.BarangItemReq{
			{ProdukID: "SMB-001", Qty: 2},
			{ProdukID: "SMB-002", Qty: 3},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(190), result.NominalPenarikan)
	assert.True(t, result.SaldoCukup)
	assert.Equal(t, float64(10), result.SaldoSetelah)
	assert.Len(t, result.ItemBarang, 2)
	assert.Equal(t, float64(100), result.ItemBarang[0].SubtotalPoin)
	assert.Equal(t, float64(90), result.ItemBarang[1].SubtotalPoin)
}

func TestAjukanPenarikan_NasabahTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-XXX").Return(nil, errors.New("not found"))

	result, err := svc.Ajukan("NSB-XXX", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 10000})

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestAjukanPenarikan_ValidasiGagal(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 0})

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestAjukanPenarikan_BarangDuplikat(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID: 2,
		ItemBarang: []services.BarangItemReq{
			{ProdukID: "SMB-001", Qty: 2},
			{ProdukID: "SMB-001", Qty: 3},
		},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplikat")
}

func TestAjukanPenarikan_DeadlineKonfirmasiKosong(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 50000})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline_konfirmasi")
}

func TestAjukanPenarikan_DeadlineKonfirmasiFormatSalah(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID: 1, NominalPenarikan: 50000, DeadlineKonfirmasi: "bukan-tanggal-valid",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Format deadline_konfirmasi")
}

func TestAjukanPenarikan_DeadlineKonfirmasiKurangDariSatuJam(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID: 1, NominalPenarikan: 50000,
		DeadlineKonfirmasi: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "minimal 1 jam")
}

func TestAjukanPenarikan_DeadlineKonfirmasiLebihDariTigaHari(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID: 1, NominalPenarikan: 50000,
		DeadlineKonfirmasi: time.Now().Add(4 * 24 * time.Hour).Format(time.RFC3339),
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maksimal 3 hari")
}

func TestAjukanPenarikan_CatatanTerlaluPanjang(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	panjang := strings.Repeat("a", 256)
	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID:           1,
		NominalPenarikan:   50000,
		DeadlineKonfirmasi: validDeadlineStr(),
		Catatan:            &panjang,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "255 karakter")
}

func TestAjukanPenarikan_SaldoKurangDalamTx(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	saldo := penarikanSaldo(50000)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(saldo, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID: 1, NominalPenarikan: 100000, DeadlineKonfirmasi: validDeadlineStr(),
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Saldo tidak mencukupi")
}

func TestAjukanPenarikan_Sukses_Uang(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	saldo := penarikanSaldo(300000)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(200000)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("SavePenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"penarikan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectCommit()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID: 1, NominalPenarikan: 100000, DeadlineKonfirmasi: validDeadlineStr(),
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.PenarikanID)
	assert.Equal(t, float64(100000), result.Nominal)
	assert.Equal(t, models.StatusPenarikanPending, result.Status)
	repo.AssertExpectations(t)
}

func TestAjukanPenarikan_Barang_StokKurangDalamTx(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 500, SatuanNominalSaldo: models.SatuanRewardEnumPoin}
	katalog := &models.KatalogBarang{ProdukID: "SMB-001", NilaiPoin: 50, MasterBarang: models.Barang{NamaBarang: "Beras"}}
	stok := &models.StokBarang{ProdukID: "SMB-001", Stok: 1}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldo, nil)
	repo.On("LockProduk", mock.Anything, "SMB-001", "BSM-001").Return(katalog, nil)
	repo.On("LockStok", mock.Anything, "SMB-001", "BSM-001").Return(stok, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID:           2,
		ItemBarang:        []services.BarangItemReq{{ProdukID: "SMB-001", Qty: 5}},
		DeadlineKonfirmasi: validDeadlineStr(),
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Stok")
}

func TestAjukanPenarikan_Sukses_Barang(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 500, SatuanNominalSaldo: models.SatuanRewardEnumPoin}
	katalog := &models.KatalogBarang{ProdukID: "SMB-001", NilaiPoin: 50, MasterBarang: models.Barang{NamaBarang: "Beras"}}
	stok := &models.StokBarang{ProdukID: "SMB-001", Stok: 10}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldo, nil)
	repo.On("LockProduk", mock.Anything, "SMB-001", "BSM-001").Return(katalog, nil)
	repo.On("LockStok", mock.Anything, "SMB-001", "BSM-001").Return(stok, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(400)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("SavePenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("SaveDetailBarang", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateStok", mock.Anything, "SMB-001", "BSM-001", float64(-2)).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"penarikan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectCommit()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID:           2,
		ItemBarang:        []services.BarangItemReq{{ProdukID: "SMB-001", Qty: 2}},
		DeadlineKonfirmasi: validDeadlineStr(),
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(100), result.Nominal)
	assert.Equal(t, models.StatusPenarikanPending, result.Status)
	repo.AssertExpectations(t)
}

func TestKonfirmasiPenarikan_StatusInvalid(t *testing.T) {
	_, _, _, svc := newPenarikanSvc(t)

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", services.KonfirmasiPenarikanReq{Status: "selesai"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "approved")
}

func TestKonfirmasiPenarikan_CatatanKosongPasReject(t *testing.T) {
	_, _, _, svc := newPenarikanSvc(t)

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", services.KonfirmasiPenarikanReq{Status: "rejected"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "catatan wajib")
}

func TestKonfirmasiPenarikan_DeadlineJemputKosongPasApprove(t *testing.T) {
	_, _, _, svc := newPenarikanSvc(t)

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", services.KonfirmasiPenarikanReq{Status: "approved"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline_jemput")
}

func TestKonfirmasiPenarikan_AdminTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(nil, errors.New("not found"))

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")}
	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "petugas")
}

func TestKonfirmasiPenarikan_RoleBukanPetugas(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.AdminBSM}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")}
	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "petugas")
}

func TestKonfirmasiPenarikan_PenarikanTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-XXX").Return(nil, errors.New("not found"))

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")}
	err := svc.Konfirmasi("RDM-XXX", "USR-ADM-001", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak ditemukan")
}

func TestKonfirmasiPenarikan_BankMismatch(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	adminBankID := "BSM-001"
	penarikanBankID := "BSM-002"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &adminBankID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID(penarikanBankID),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")}
	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "akses")
}

func TestKonfirmasiPenarikan_ConflictOfInterest(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	repo.On("FindNasabah", mock.Anything, "NSB-001").
		Return(nasabahMilik("NSB-001", "USR-ADM-001", "BSM-001"), nil)
	dbMock.ExpectRollback()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")}
	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sendiri")
}

func TestKonfirmasiPenarikan_StatusBukanPending(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       nil,
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanApproved,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")}
	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "approved")
}

func TestKonfirmasiPenarikan_DeadlineKonfirmasiSudahLewat(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	past := time.Now().Add(-1 * time.Hour)
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:        "RDM-001",
		NasabahID:          nil,
		BankID:             pBankID("BSM-001"),
		StatusPenarikan:    models.StatusPenarikanPending,
		DeadlineKonfirmasi: &past,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")}
	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", req)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "konfirmasi")
	assert.Contains(t, err.Error(), "lewat")
}

func TestKonfirmasiPenarikan_Sukses_Approve(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindPenarikanByID", "RDM-001").Return(nil, errors.New("skip")).Maybe()

	dbMock.ExpectBegin()
	repo.On("FindNasabah", mock.Anything, "NSB-001").
		Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectCommit()

	req := services.KonfirmasiPenarikanReq{Status: "approved", DeadlineJemput: validDeadlineStr()}
	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", req)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestKonfirmasiPenarikan_Sukses_Reject_Uang(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	rewardID := 1
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:      "RDM-002",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           pBankID("BSM-001"),
		RewardID:         &rewardID,
		NominalPenarikan: 100000,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardUang(),
	}
	saldo := penarikanSaldo(50000)

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-002").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(150000)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindPenarikanByID", "RDM-002").Return(nil, errors.New("skip")).Maybe()

	dbMock.ExpectBegin()
	repo.On("FindNasabah", mock.Anything, "NSB-001").
		Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	dbMock.ExpectCommit()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("Stok gak sesuai")}
	err := svc.Konfirmasi("RDM-002", "USR-ADM-001", req)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestKonfirmasiPenarikan_Sukses_Reject_Barang_StokDikembalikan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	rewardID := 2
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:      "RDM-003",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           pBankID("BSM-001"),
		RewardID:         &rewardID,
		NominalPenarikan: 150,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardBarang(),
	}
	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 100, SatuanNominalSaldo: models.SatuanRewardEnumPoin}
	details := []models.DetailPenarikanBarang{
		{ProdukID: "SMB-001", Qty: 3, NilaiPoin: 50, SubtotalPoin: 150},
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-003").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(250)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindDetailBarang", mock.Anything, "RDM-003").Return(details, nil)
	repo.On("UpdateStok", mock.Anything, "SMB-001", "BSM-001", float64(3)).Return(nil)
	repo.On("FindPenarikanByID", "RDM-003").Return(nil, errors.New("skip")).Maybe()

	dbMock.ExpectBegin()
	repo.On("FindNasabah", mock.Anything, "NSB-001").
		Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	dbMock.ExpectCommit()

	req := services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("Barang gak sesuai")}
	err := svc.Konfirmasi("RDM-003", "USR-ADM-001", req)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSelesaiPenarikan_QR_FormatInvalid(t *testing.T) {
	_, _, _, svc := newPenarikanSvc(t)

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{QRData: "bukan json"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Format QR code")
}

func TestSelesaiPenarikan_QR_TypeSalah(t *testing.T) {
	_, _, _, svc := newPenarikanSvc(t)

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		QRData: `{"type":"ENVIROO-SETORAN","nasabah_id":"NSB-001","penarikan_id":"RDM-001"}`,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bukan milik aplikasi")
}

func TestSelesaiPenarikan_Manual_BuktiFotoKosong(t *testing.T) {
	_, _, _, svc := newPenarikanSvc(t)

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		NasabahID: "NSB-001", PenarikanID: "RDM-001",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bukti foto wajib")
}

func TestSelesaiPenarikan_Manual_CatatanKosong(t *testing.T) {
	_, _, _, svc := newPenarikanSvc(t)

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		NasabahID: "NSB-001", PenarikanID: "RDM-001", BuktiFoto: "https://foto.jpg",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Catatan wajib")
}

func TestSelesaiPenarikan_RoleBukanPetugas(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.AdminBSM}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		NasabahID: "NSB-001", PenarikanID: "RDM-001", BuktiFoto: "https://foto.jpg", Catatan: ptrStr("Diambil langsung"),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "petugas")
}

func TestSelesaiPenarikan_NasabahTidakSesuai(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-LAIN"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanApproved,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		NasabahID: "NSB-001", PenarikanID: "RDM-001", BuktiFoto: "https://foto.jpg", Catatan: ptrStr("Diambil langsung"),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak sesuai")
}

func TestSelesaiPenarikan_StatusBukanApproved(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	repo.On("FindNasabah", mock.Anything, "NSB-001").
		Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		NasabahID: "NSB-001", PenarikanID: "RDM-001", BuktiFoto: "https://foto.jpg", Catatan: ptrStr("Diambil langsung"),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "diselesaikan")
}

func TestSelesaiPenarikan_DeadlineJemputSudahLewat(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	past := time.Now().Add(-1 * time.Hour)
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanApproved,
		DeadlineJemput:  &past,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	repo.On("FindNasabah", mock.Anything, "NSB-001").
		Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		NasabahID: "NSB-001", PenarikanID: "RDM-001", BuktiFoto: "https://foto.jpg", Catatan: ptrStr("Diambil langsung"),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "penjemputan")
}

func TestSelesaiPenarikan_Sukses_Manual(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanApproved,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindPenarikanByID", "RDM-001").Return(nil, errors.New("skip")).Maybe()

	dbMock.ExpectBegin()
	repo.On("FindNasabah", mock.Anything, "NSB-001").
		Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectCommit()

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		NasabahID: "NSB-001", PenarikanID: "RDM-001", BuktiFoto: "https://foto.jpg", Catatan: ptrStr("Diambil langsung"),
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestSelesaiPenarikan_Sukses_QR(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001", Role: models.PetugasBSM}
	p := &models.Penarikan{
		PenarikanID:     "RDM-002",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanApproved,
	}

	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-002").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindPenarikanByID", "RDM-002").Return(nil, errors.New("skip")).Maybe()

	dbMock.ExpectBegin()
	repo.On("FindNasabah", mock.Anything, "NSB-001").
		Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectCommit()

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{
		QRData: `{"type":"ENVIROO-PENARIKAN","nasabah_id":"NSB-001","penarikan_id":"RDM-002"}`,
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBatalPenarikan_CatatanTerlaluPanjang(t *testing.T) {
	_, _, _, svc := newPenarikanSvc(t)

	panjang := strings.Repeat("a", 256)
	err := svc.Batal("RDM-001", "USR-NSB-001", panjang)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "255 karakter")
}

func TestBatalPenarikan_BukanNasabah(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahByUserID", "USR-ADM-001").Return(nil, errors.New("not found"))

	err := svc.Batal("RDM-001", "USR-ADM-001", "batal dulu")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nasabah")
}

func TestBatalPenarikan_PenarikanTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-XXX").Return(nil, errors.New("not found"))

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Batal("RDM-XXX", "USR-NSB-001", "batal dulu")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak ditemukan")
}

func TestBatalPenarikan_BukanMilikNasabah(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-LAIN"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Batal("RDM-001", "USR-NSB-001", "batal dulu")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "akses")
}

func TestBatalPenarikan_StatusBukanPending(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanApproved,
	}

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Batal("RDM-001", "USR-NSB-001", "batal dulu")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dibatalkan")
}

func TestBatalPenarikan_Sukses_Uang(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	rewardID := 1
	p := &models.Penarikan{
		PenarikanID:      "RDM-001",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           pBankID("BSM-001"),
		RewardID:         &rewardID,
		NominalPenarikan: 100000,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardUang(),
	}
	saldo := penarikanSaldo(200000)

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(300000)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	dbMock.ExpectCommit()

	err := svc.Batal("RDM-001", "USR-NSB-001", "gak jadi ambil")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBatalPenarikan_Sukses_Barang_StokDikembalikan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	rewardID := 2
	p := &models.Penarikan{
		PenarikanID:      "RDM-002",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           pBankID("BSM-001"),
		RewardID:         &rewardID,
		NominalPenarikan: 150,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardBarang(),
	}
	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 100, SatuanNominalSaldo: models.SatuanRewardEnumPoin}
	details := []models.DetailPenarikanBarang{
		{ProdukID: "SMB-001", Qty: 3, NilaiPoin: 50, SubtotalPoin: 150},
	}

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-002").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(250)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindDetailBarang", mock.Anything, "RDM-002").Return(details, nil)
	repo.On("UpdateStok", mock.Anything, "SMB-001", "BSM-001", float64(3)).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	dbMock.ExpectCommit()

	err := svc.Batal("RDM-002", "USR-NSB-001", "gak jadi ambil")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestGetListPenarikan_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	filter := repositories.PenarikanFilter{StartDate: "2026-06-01", EndDate: "2026-06-30"}
	repo.On("FindByNasabahID", "NSB-001", filter).Return([]repositories.PenarikanListRow{}, nil)

	result, err := svc.GetList("NSB-001", filter)

	assert.NoError(t, err)
	assert.Empty(t, result.Data)
}

func TestGetListPenarikan_MappingField(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	createdAt := time.Now()
	rows := []repositories.PenarikanListRow{
		{
			PenarikanID:      "RDM-001",
			CreatedAt:        &createdAt,
			StatusPenarikan:  models.StatusPenarikanCompleted,
			NominalPenarikan: 150000,
			SatuanPenarikan:  models.SatuanRewardEnumRp,
			RewardID:         pRewardID(1),
			NamaReward:       "uang",
		},
	}
	repo.On("FindByNasabahID", "NSB-001", repositories.PenarikanFilter{}).Return(rows, nil)

	result, err := svc.GetList("NSB-001", repositories.PenarikanFilter{})

	assert.NoError(t, err)
	assert.Len(t, result.Data, 1)
	item := result.Data[0]
	assert.Equal(t, "RDM-001", item.PenarikanID)
	assert.Equal(t, &createdAt, item.CreatedAt)
	assert.Equal(t, models.StatusPenarikanCompleted, item.StatusPenarikan)
	assert.Equal(t, float64(150000), item.NominalPenarikan)
	assert.Equal(t, models.SatuanRewardEnumRp, item.SatuanPenarikan)
	assert.Equal(t, "uang", item.NamaReward)
}

func TestGetListByBankPenarikan_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	r := penarikanRewardUang()
	nas := penarikanNasabah()
	nas.User = models.User{UserID: "USR-NSB-001", Nama: "Budi"}
	bank := &models.BankSampah{BankID: "BSM-001", NamaBank: "Bank Satu"}
	p := models.Penarikan{
		PenarikanID:      "RDM-001",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           &bID,
		RewardID:         pRewardID(1),
		NominalPenarikan: 50000,
		SatuanPenarikan:  models.SatuanRewardEnumRp,
		StatusPenarikan:  models.StatusPenarikanPending,
		Nasabah:          nas,
		Bank:             bank,
		Reward:           r,
	}

	filter := repositories.PenarikanFilter{}
	repo.On("FindByBankID", "BSM-001", filter).Return([]models.Penarikan{p}, nil)

	result, err := svc.GetListByBank("BSM-001", filter)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "RDM-001", result[0].PenarikanID)
	assert.Equal(t, "Budi", result[0].NamaNasabah)
	assert.Equal(t, "Bank Satu", result[0].NamaBank)
}

func TestGetDetailPenarikan_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindPenarikanByID", "RDM-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetDetail("RDM-XXX", "USR-001", models.SuperAdmin)

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGetDetailPenarikan_SuperAdmin_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	p := &models.Penarikan{
		PenarikanID:      "RDM-001",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           &bID,
		RewardID:         pRewardID(1),
		NominalPenarikan: 75000,
		SatuanPenarikan:  models.SatuanRewardEnumRp,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardUang(),
	}

	repo.On("FindPenarikanByID", "RDM-001").Return(p, nil)
	repo.On("FindRiwayatPenarikan", "RDM-001").Return([]repositories.RiwayatPenarikanRow{}, nil)

	result, err := svc.GetDetail("RDM-001", "USR-SUPER-001", models.SuperAdmin)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "RDM-001", result.Header.PenarikanID)
	assert.Equal(t, float64(75000), result.Header.NominalPenarikan)
	assert.Equal(t, string(models.RewardEnumUang), result.Header.NamaReward)
	assert.Empty(t, result.DetailPenarikan)
	assert.Empty(t, result.Riwayat)
}

func TestGetDetailPenarikan_SuperAdmin_DenganDetailBarang(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	katalog := &models.KatalogBarang{
		ProdukID:     "SMB-001",
		NilaiPoin:     50,
		MasterBarang: models.Barang{NamaBarang: "Beras"},
	}
	p := &models.Penarikan{
		PenarikanID:      "RDM-002",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           &bID,
		RewardID:         pRewardID(2),
		NominalPenarikan: 100,
		SatuanPenarikan:  models.SatuanRewardEnumPoin,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardBarang(),
		DetailPenarikanBarang: []models.DetailPenarikanBarang{
			{ProdukID: "SMB-001", Qty: 2, NilaiPoin: 50, SubtotalPoin: 100, KatalogBarang: katalog},
		},
	}

	repo.On("FindPenarikanByID", "RDM-002").Return(p, nil)
	repo.On("FindRiwayatPenarikan", "RDM-002").Return([]repositories.RiwayatPenarikanRow{}, nil)

	result, err := svc.GetDetail("RDM-002", "USR-SUPER-001", models.SuperAdmin)

	assert.NoError(t, err)
	assert.Len(t, result.DetailPenarikan, 1)
	assert.Equal(t, "Beras", result.DetailPenarikan[0].NamaBarang)
	assert.Equal(t, float64(100), result.DetailPenarikan[0].SubtotalPoin)
}

func TestGetDetailPenarikan_RiwayatDenganNamaResolve(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	bID := "BSM-001"
	p := &models.Penarikan{
		PenarikanID:     "RDM-003",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          &bID,
		RewardID:        pRewardID(1),
		StatusPenarikan: models.StatusPenarikanCompleted,
		Reward:          penarikanRewardUang(),
	}
	repo.On("FindPenarikanByID", "RDM-003").Return(p, nil)

	nama := "Budi Santoso"
	rows := []repositories.RiwayatPenarikanRow{
		{
			RiwayatPenarikanID: "RWP-001",
			StatusPenarikan:    models.StatusPenarikanPending,
			CreatedByRaw:       "USR-NSB-001",
			CreatedByNama:      &nama,
			CreatedAt:          time.Now(),
		},
		{
			RiwayatPenarikanID: "RWP-002",
			StatusPenarikan:    models.StatusPenarikanRequestExpired,
			CreatedByRaw:       "sistem",
			CreatedByNama:      nil,
			CreatedAt:          time.Now(),
		},
	}
	repo.On("FindRiwayatPenarikan", "RDM-003").Return(rows, nil)

	result, err := svc.GetDetail("RDM-003", "USR-SUPER-001", models.SuperAdmin)

	assert.NoError(t, err)
	assert.Len(t, result.Riwayat, 2)
	assert.Equal(t, "Budi Santoso", result.Riwayat[0].CreatedBy)
	assert.Equal(t, "Sistem", result.Riwayat[1].CreatedBy)
}

func previewUang(svc services.PenarikanService) (*services.PreviewResult, error) {
	return svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 10000})
}

func TestPreviewPenarikan_SettingBelumAdaDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(nil, gorm.ErrRecordNotFound)

	result, err := previewUang(svc)

	assert.Nil(t, result)
	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "belum mengatur")
	repo.AssertNotCalled(t, "FindReward", mock.Anything)
}

func TestPreviewPenarikan_CekSettingError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(nil, errors.New("db error"))

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

func TestPreviewPenarikan_BelumPernahMenabungDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(nil, nil)

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "Belum ada setoran")
}

func TestPreviewPenarikan_BelumCukupLamaDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	pertama := time.Now().AddDate(0, 0, -10)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(&pertama, nil)

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), utils.FormatTanggalSajaIndo(pertama.AddDate(0, 1, 0)))
	assert.Contains(t, err.Error(), "minimal menabung 1 bulan")
	repo.AssertNotCalled(t, "FindReward", mock.Anything)
}

func TestPreviewPenarikan_SatuanMingguSudahCukupLolos(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	pertama := time.Now().AddDate(0, 0, -15)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(2, models.SatuanWaktuMinggu), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(&pertama, nil)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 1).Return(penarikanSaldo(50000), nil)

	result, err := previewUang(svc)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestPreviewPenarikan_SatuanMingguBelumCukupDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	pertama := time.Now().AddDate(0, 0, -10)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(2, models.SatuanWaktuMinggu), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(&pertama, nil)

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), utils.FormatTanggalSajaIndo(pertama.AddDate(0, 0, 14)))
	assert.Contains(t, err.Error(), "minimal menabung 2 minggu")
}

func TestPreviewPenarikan_SaldoAwalPakaiJoinedAt(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	nasabah := penarikanNasabah()
	nasabah.JoinedAt = time.Now().AddDate(0, -2, 0)
	repo.On("FindNasabahAktif", "NSB-001").Return(nasabah, nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(true, nil)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 1).Return(penarikanSaldo(50000), nil)

	result, err := previewUang(svc)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	repo.AssertNotCalled(t, "FindTanggalSetoranPertama", mock.Anything)
}

func TestPreviewPenarikan_SaldoAwalJoinedAtBaruDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	nasabah := penarikanNasabah()
	nasabah.JoinedAt = time.Now().AddDate(0, 0, -7)
	repo.On("FindNasabahAktif", "NSB-001").Return(nasabah, nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(true, nil)

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), utils.FormatTanggalSajaIndo(nasabah.JoinedAt.AddDate(0, 1, 0)))
}

func TestPreviewPenarikan_CekSaldoAwalError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, errors.New("db error"))

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

func TestPreviewPenarikan_CekSetoranPertamaError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(nil, errors.New("db error"))

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

func TestAjukanPenarikan_BarangBelumCukupLamaDitolak(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	pertama := time.Now().AddDate(0, 0, -10)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(&pertama, nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID:           2,
		ItemBarang:         []services.BarangItemReq{{ProdukID: "PRD-001", Qty: 1}},
		DeadlineKonfirmasi: validDeadlineStr(),
	})

	assert.Nil(t, result)
	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	repo.AssertNotCalled(t, "FindReward", mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet(), "transaksi gak boleh dibuka")
}

func pnrAdmin(bankID string) *models.Admin {
	return &models.Admin{AdminID: "ADM-001", UserID: "USR-ADM-001", BankID: &bankID, Role: models.AdminBSM}
}

func TestSimpanSettingWaktu_MinWaktuTidakValid(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	_, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 0, SatuanWaktu: models.SatuanWaktuBulan})

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	repo.AssertNotCalled(t, "FindAdminAktif", mock.Anything, mock.Anything)
}

func TestSimpanSettingWaktu_SatuanTidakValid(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	_, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 3, SatuanWaktu: "hari"})

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	repo.AssertNotCalled(t, "FindAdminAktif", mock.Anything, mock.Anything)
}

func TestSimpanSettingWaktu_AdminTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(nil, gorm.ErrRecordNotFound)

	_, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 1, SatuanWaktu: models.SatuanWaktuBulan})

	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
}

func TestSimpanSettingWaktu_AdminBankLainDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-999"), nil)

	_, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 1, SatuanWaktu: models.SatuanWaktuBulan})

	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	repo.AssertNotCalled(t, "FindSettingPenarikan", mock.Anything)
}

func TestSimpanSettingWaktu_BuatBaru(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-001"), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(nil, gorm.ErrRecordNotFound)
	var dibuat *models.SettingPenarikan
	repo.On("CreateSettingPenarikan", mock.Anything).
		Run(func(args mock.Arguments) { dibuat = args.Get(0).(*models.SettingPenarikan) }).Return(nil)

	hasil, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 1, SatuanWaktu: models.SatuanWaktuBulan})

	assert.NoError(t, err)
	if assert.NotNil(t, dibuat) {
		assert.Equal(t, "SPBSM-001", dibuat.SettingID)
		assert.Equal(t, "BSM-001", dibuat.BankID)
		assert.Equal(t, 1, *dibuat.MinWaktu)
		assert.Equal(t, models.SatuanWaktuBulan, *dibuat.SatuanWaktu)
		assert.Nil(t, dibuat.MinRupiah, "syarat saldo belum diatur")
		assert.Nil(t, dibuat.MinPoin, "syarat saldo belum diatur")
		assert.Equal(t, "ADM-001", dibuat.CreatedBy)
		assert.Equal(t, "ADM-001", dibuat.UpdatedBy)
	}
	assert.Equal(t, dibuat, hasil)
	repo.AssertNotCalled(t, "SaveSettingPenarikan", mock.Anything)
}

func TestSimpanSettingWaktu_UpdateYangAda(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	lama := pnrSetting(3, models.SatuanWaktuBulan)
	lama.MinRupiah = pnrFloat(25000)
	lama.CreatedBy, lama.UpdatedBy = "ADM-LAMA", "ADM-LAMA"
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-001"), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(lama, nil)
	repo.On("SaveSettingPenarikan", lama).Return(nil)

	hasil, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 2, SatuanWaktu: models.SatuanWaktuMinggu})

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, 2, *hasil.MinWaktu)
		assert.Equal(t, models.SatuanWaktuMinggu, *hasil.SatuanWaktu)
		assert.Equal(t, 25000.0, *hasil.MinRupiah, "syarat saldo gak boleh ikut berubah")
		assert.Equal(t, 0.0, *hasil.MinPoin, "syarat saldo gak boleh ikut berubah")
		assert.Equal(t, "ADM-LAMA", hasil.CreatedBy, "pembuat awal gak boleh berubah")
		assert.Equal(t, "ADM-001", hasil.UpdatedBy)
	}
	repo.AssertNotCalled(t, "CreateSettingPenarikan", mock.Anything)
}

func TestSimpanSettingWaktu_SimpanError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-001"), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateSettingPenarikan", mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 1, SatuanWaktu: models.SatuanWaktuBulan})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

func TestSimpanSettingSaldo_DuaDuanyaKosong(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	_, err := svc.SimpanSettingSaldoPenarikan("BSM-001", "USR-ADM-001", services.SimpanSettingSaldoReq{})

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "salah satu")
	repo.AssertNotCalled(t, "FindAdminAktif", mock.Anything, mock.Anything)
}

func TestSimpanSettingSaldo_NilaiNegatif(t *testing.T) {
	cases := map[string]services.SimpanSettingSaldoReq{
		"min_rupiah": {MinRupiah: pnrFloat(-1)},
		"min_poin":   {MinRupiah: pnrFloat(25000), MinPoin: pnrFloat(-5)},
	}
	for field, req := range cases {
		t.Run(field, func(t *testing.T) {
			_, repo, _, svc := newPenarikanSvc(t)

			_, err := svc.SimpanSettingSaldoPenarikan("BSM-001", "USR-ADM-001", req)

			assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
			assert.Contains(t, err.Error(), field)
			repo.AssertNotCalled(t, "FindAdminAktif", mock.Anything, mock.Anything)
		})
	}
}

func TestSimpanSettingSaldo_AdminBankLainDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-999"), nil)

	_, err := svc.SimpanSettingSaldoPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingSaldoReq{MinRupiah: pnrFloat(25000)})

	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	repo.AssertNotCalled(t, "FindSettingPenarikan", mock.Anything)
}

func TestSimpanSettingSaldo_BuatBaruCumaRupiah(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-001"), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(nil, gorm.ErrRecordNotFound)
	var dibuat *models.SettingPenarikan
	repo.On("CreateSettingPenarikan", mock.Anything).
		Run(func(args mock.Arguments) { dibuat = args.Get(0).(*models.SettingPenarikan) }).Return(nil)

	hasil, err := svc.SimpanSettingSaldoPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingSaldoReq{MinRupiah: pnrFloat(25000)})

	assert.NoError(t, err)
	if assert.NotNil(t, dibuat) {
		assert.Equal(t, "SPBSM-001", dibuat.SettingID)
		assert.Equal(t, 25000.0, *dibuat.MinRupiah)
		assert.Nil(t, dibuat.MinPoin, "bank ini gak melayani penarikan poin")
		assert.Nil(t, dibuat.MinWaktu, "syarat waktu belum diatur")
		assert.Nil(t, dibuat.SatuanWaktu, "syarat waktu belum diatur")
		assert.Equal(t, "ADM-001", dibuat.CreatedBy)
	}
	assert.Equal(t, dibuat, hasil)
	repo.AssertNotCalled(t, "SaveSettingPenarikan", mock.Anything)
}

func TestSimpanSettingSaldo_UpdateMenggantiSyaratSaldo(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	lama := pnrSetting(1, models.SatuanWaktuBulan)
	lama.MinRupiah, lama.MinPoin = pnrFloat(10000), pnrFloat(100)
	lama.CreatedBy, lama.UpdatedBy = "ADM-LAMA", "ADM-LAMA"
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-001"), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(lama, nil)
	repo.On("SaveSettingPenarikan", lama).Return(nil)

	hasil, err := svc.SimpanSettingSaldoPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingSaldoReq{MinRupiah: pnrFloat(25000)})

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, 25000.0, *hasil.MinRupiah)
		assert.Nil(t, hasil.MinPoin, "min_poin yang gak dikirim ikut terhapus")
		assert.Equal(t, 1, *hasil.MinWaktu, "syarat waktu gak boleh ikut berubah")
		assert.Equal(t, models.SatuanWaktuBulan, *hasil.SatuanWaktu, "syarat waktu gak boleh ikut berubah")
		assert.Equal(t, "ADM-LAMA", hasil.CreatedBy)
		assert.Equal(t, "ADM-001", hasil.UpdatedBy)
	}
	repo.AssertNotCalled(t, "CreateSettingPenarikan", mock.Anything)
}

// Nasabah NSB-001 sudah memenuhi syarat waktu; syarat saldo diatur per test.
func penarikanBolehMenarikDenganSaldo(repo *mocks.MockPenarikanRepo, minRupiah, minPoin *float64) {
	setting := pnrSetting(1, models.SatuanWaktuBulan)
	setting.MinRupiah, setting.MinPoin = minRupiah, minPoin
	pertama := time.Now().AddDate(0, -3, 0)
	repo.On("FindSettingPenarikan", "BSM-001").Return(setting, nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(&pertama, nil)
}

func TestPreviewPenarikan_SyaratWaktuBelumDiaturDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	setting := &models.SettingPenarikan{SettingID: "SPBSM-001", BankID: "BSM-001", MinRupiah: pnrFloat(25000)}
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(setting, nil)

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "syarat waktu")
	repo.AssertNotCalled(t, "PunyaSaldoAwal", mock.Anything)
	repo.AssertNotCalled(t, "FindReward", mock.Anything)
}

func TestPreviewPenarikan_MinRupiahBelumDiaturDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarikDenganSaldo(repo, nil, pnrFloat(100))
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	_, err := previewUang(svc)

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "syarat saldo penarikan uang")
	repo.AssertNotCalled(t, "FindSaldoRekening", mock.Anything, mock.Anything)
}

func TestPreviewPenarikan_MinPoinBelumDiaturDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarikDenganSaldo(repo, pnrFloat(25000), nil)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)

	_, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{
		RewardID:   2,
		ItemBarang: []services.BarangItemReq{{ProdukID: "SMB-001", Qty: 1}},
	})

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "syarat saldo penarikan poin")
	repo.AssertNotCalled(t, "FindProduk", mock.Anything, mock.Anything)
}

func TestPreviewPenarikan_SaldoDiBawahMinimalDitolak(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarikDenganSaldo(repo, pnrFloat(25000), nil)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 1).Return(penarikanSaldo(20000), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 10000})

	assert.Nil(t, result)
	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "Saldo minimal untuk penarikan Rp25000, saldo Anda Rp20000")
}

func TestPreviewPenarikan_SaldoPasMinimalLolos(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarikDenganSaldo(repo, pnrFloat(25000), nil)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 1).Return(penarikanSaldo(25000), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 25000})

	assert.NoError(t, err)
	if assert.NotNil(t, result) {
		assert.Equal(t, 25000.0, result.NominalPenarikan)
		assert.Equal(t, 0.0, result.SaldoSetelah, "saldo boleh ditarik sampai habis")
	}
}

func TestPreviewPenarikan_NominalKecilSaldoCukupLolos(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarikDenganSaldo(repo, pnrFloat(25000), nil)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 1).Return(penarikanSaldo(200000), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 5000})

	assert.NoError(t, err, "syarat minimal dicek ke saldo, bukan ke nominal penarikan")
	if assert.NotNil(t, result) {
		assert.Equal(t, 5000.0, result.NominalPenarikan)
	}
}

func TestAjukanPenarikan_SaldoDiBawahMinimalDitolak(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarikDenganSaldo(repo, pnrFloat(25000), nil)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(penarikanSaldo(20000), nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID: 1, NominalPenarikan: 10000, DeadlineKonfirmasi: validDeadlineStr(),
	})

	assert.Nil(t, result)
	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "Saldo minimal untuk penarikan Rp25000, saldo Anda Rp20000")
	repo.AssertNotCalled(t, "UpdateSaldo", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "SavePenarikan", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestAjukanPenarikan_SaldoPoinDiBawahMinimalDitolak(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 150, SatuanNominalSaldo: models.SatuanRewardEnumPoin}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarikDenganSaldo(repo, nil, pnrFloat(200))
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldo, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID:           2,
		ItemBarang:         []services.BarangItemReq{{ProdukID: "SMB-001", Qty: 2}},
		DeadlineKonfirmasi: validDeadlineStr(),
	})

	assert.Nil(t, result)
	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "Saldo minimal untuk penarikan 200 poin, saldo Anda 150 poin")
	repo.AssertNotCalled(t, "LockProduk", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertNotCalled(t, "SavePenarikan", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGetSettingPenarikan_BelumAda(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindSettingPenarikan", "BSM-001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.GetSettingPenarikan("BSM-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, pnrKode(t, err))
}

func TestGetSettingPenarikan_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)

	hasil, err := svc.GetSettingPenarikan("BSM-001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, 1, *hasil.MinWaktu)
		assert.Equal(t, models.SatuanWaktuBulan, *hasil.SatuanWaktu)
		assert.Equal(t, 0.0, *hasil.MinRupiah)
	}
}

// ===== Helper: transaksi "gagal di langkah ke-N" & goroutine notifikasi =====

// langkahTx adalah satu langkah di dalam transaksi. Test "gagal di langkah ke-N" menjalankan
// semua langkah sebelum N dengan sukses lalu menggagalkan langkah N.
type langkahTx struct {
	nama          string
	sukses, gagal func()
}

func langkahRepo(nama string, call func() *mock.Call) langkahTx {
	return langkahTx{
		nama:   nama,
		sukses: func() { call().Return(nil).Once() },
		gagal:  func() { call().Return(errors.New("db error")).Once() },
	}
}

func langkahGenerateID(nama string, dbMock sqlmock.Sqlmock, kolom string) langkahTx {
	return langkahTx{
		nama:   nama,
		sukses: func() { dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{kolom})) },
		gagal:  func() { dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("lock timeout")) },
	}
}

func jalankanSampaiGagal(langkah []langkahTx, gagalDi int) {
	for _, l := range langkah[:gagalDi] {
		l.sukses()
	}
	langkah[gagalDi].gagal()
}

func kirimSinyal(sinyal chan<- struct{}) func(mock.Arguments) {
	return func(mock.Arguments) { sinyal <- struct{}{} }
}

func tungguSinyal(t *testing.T, sinyal <-chan struct{}, jumlah int) {
	t.Helper()
	for i := 0; i < jumlah; i++ {
		select {
		case <-sinyal:
		case <-time.After(time.Second):
			t.Fatalf("goroutine notifikasi ke-%d tidak jalan dalam 1 detik", i+1)
		}
	}
}

// newPenarikanSvc men-stub FindNasabahWithUser supaya goroutine notifikasi langsung berhenti;
// test yang memeriksa notifikasi melepas stub itu dulu.
func lepasStub(m *mock.Mock, method string) {
	var target []*mock.Call
	for _, c := range m.ExpectedCalls {
		if c.Method == method {
			target = append(target, c)
		}
	}
	for _, c := range target {
		c.Unset()
	}
}

func pnrPetugas() *models.Admin {
	bankID := "BSM-001"
	return &models.Admin{AdminID: "ADM-001", UserID: "USR-ADM-001", BankID: &bankID, Role: models.PetugasBSM}
}

func katalogBeras() *models.KatalogBarang {
	return &models.KatalogBarang{ProdukID: "SMB-001", NilaiPoin: 50, MasterBarang: models.Barang{NamaBarang: "Beras"}}
}

func saldoPoin(nominal float64) *models.SaldoRekening {
	return &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: nominal, SatuanNominalSaldo: models.SatuanRewardEnumPoin}
}

func reqAjukanBarang(qty float64) services.AjukanPenarikanReq {
	return services.AjukanPenarikanReq{
		RewardID:           2,
		ItemBarang:         []services.BarangItemReq{{ProdukID: "SMB-001", Qty: qty}},
		DeadlineKonfirmasi: validDeadlineStr(),
	}
}

func reqAjukanUang(nominal float64) services.AjukanPenarikanReq {
	return services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: nominal, DeadlineKonfirmasi: validDeadlineStr()}
}

// Penarikan barang 3 x 50 poin milik NSB-001 di BSM-001 yang masih pending.
func penarikanBarangPending(penarikanID string) *models.Penarikan {
	return &models.Penarikan{
		PenarikanID:      penarikanID,
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           pBankID("BSM-001"),
		RewardID:         pRewardID(2),
		NominalPenarikan: 150,
		SatuanPenarikan:  models.SatuanRewardEnumPoin,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardBarang(),
	}
}

// Data penarikan yang dibaca ulang goroutine notifikasi.
func penarikanUntukNotif(penarikanID string) *models.Penarikan {
	return &models.Penarikan{
		PenarikanID:      penarikanID,
		NasabahID:        pNasabahID("NSB-001"),
		NominalPenarikan: 100000,
		SatuanPenarikan:  models.SatuanRewardEnumRp,
		Nasabah:          &models.Nasabah{NasabahID: "NSB-001", User: models.User{UserID: "USR-NSB-001", FCMToken: "tok-nsb"}},
	}
}

// Langkah pengembalian saldo dan stok saat penarikan barang ditolak atau dibatalkan.
func langkahPengembalianBarang(dbMock sqlmock.Sqlmock, repo *mocks.MockPenarikanRepo, p *models.Penarikan) []langkahTx {
	return []langkahTx{
		langkahRepo("ubah status", func() *mock.Call { return repo.On("UpdateStatus", mock.Anything, p, mock.Anything) }),
		langkahGenerateID("generate riwayat penarikan", dbMock, "riwayat_penarikan_id"),
		langkahRepo("simpan riwayat penarikan", func() *mock.Call { return repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything) }),
		{
			nama:   "kunci rekening saldo",
			sukses: func() { repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldoPoin(100), nil).Once() },
			gagal:  func() { repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(nil, errors.New("not found")).Once() },
		},
		langkahRepo("kembalikan saldo", func() *mock.Call { return repo.On("UpdateSaldo", mock.Anything, mock.Anything, float64(250)) }),
		langkahGenerateID("generate riwayat saldo", dbMock, "riwayat_saldo_id"),
		langkahRepo("catat riwayat saldo", func() *mock.Call { return repo.On("CreateRiwayat", mock.Anything, mock.Anything) }),
		{
			nama: "ambil detail barang",
			sukses: func() {
				repo.On("FindDetailBarang", mock.Anything, p.PenarikanID).
					Return([]models.DetailPenarikanBarang{{ProdukID: "SMB-001", Qty: 3}}, nil).Once()
			},
			gagal: func() {
				repo.On("FindDetailBarang", mock.Anything, p.PenarikanID).
					Return([]models.DetailPenarikanBarang(nil), errors.New("db error")).Once()
			},
		},
		langkahRepo("kembalikan stok barang", func() *mock.Call { return repo.On("UpdateStok", mock.Anything, "SMB-001", "BSM-001", float64(3)) }),
	}
}

// ===== Preview =====

func TestPreviewPenarikan_Barang_NasabahBSUPakaiKatalogBSI(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	nasabah := penarikanNasabah()
	nasabah.BankID = "BSU-001"
	nasabah.Bank = models.BankSampah{BankID: "BSU-001", JenisBank: models.BSU, ParentBankID: pBankID("BSI-001")}
	pertama := time.Now().AddDate(0, -3, 0)
	repo.On("FindNasabahAktif", "NSB-001").Return(nasabah, nil)
	repo.On("FindSettingPenarikan", "BSU-001").Return(pnrSetting(1, models.SatuanWaktuBulan), nil)
	repo.On("PunyaSaldoAwal", "NSB-001").Return(false, nil)
	repo.On("FindTanggalSetoranPertama", "NSB-001").Return(&pertama, nil)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 2).Return(saldoPoin(500), nil)
	repo.On("FindProduk", "SMB-001", "BSI-001").Return(katalogBeras(), nil).Once()
	repo.On("FindStokBarang", "SMB-001", "BSU-001").Return(&models.StokBarang{ProdukID: "SMB-001", Stok: 10}, nil).Once()

	result, err := svc.Preview("NSB-001", reqAjukanBarang(2))

	assert.NoError(t, err)
	if assert.NotNil(t, result) {
		assert.Equal(t, 100.0, result.NominalPenarikan)
	}
	repo.AssertExpectations(t)
}

func TestPreviewPenarikan_Barang_ProdukTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 2).Return(saldoPoin(500), nil)
	repo.On("FindProduk", "SMB-001", "BSM-001").Return(nil, errors.New("not found"))

	result, err := svc.Preview("NSB-001", reqAjukanBarang(2))

	assert.Nil(t, result)
	assert.Equal(t, http.StatusNotFound, pnrKode(t, err))
	assert.Contains(t, err.Error(), "SMB-001")
}

func TestPreviewPenarikan_Barang_StokTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 2).Return(saldoPoin(500), nil)
	repo.On("FindProduk", "SMB-001", "BSM-001").Return(katalogBeras(), nil)
	repo.On("FindStokBarang", "SMB-001", "BSM-001").Return(nil, errors.New("not found"))

	result, err := svc.Preview("NSB-001", reqAjukanBarang(2))

	assert.Nil(t, result)
	assert.Equal(t, http.StatusNotFound, pnrKode(t, err))
	assert.Contains(t, err.Error(), "Stok produk")
}

// ===== Ajukan =====

func TestAjukanPenarikan_RewardTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(nil, errors.New("not found"))

	result, err := svc.Ajukan("NSB-001", reqAjukanUang(10000))

	assert.Nil(t, result)
	assert.Equal(t, http.StatusNotFound, pnrKode(t, err))
}

func TestAjukanPenarikan_SyaratSaldoBelumDiatur(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarikDenganSaldo(repo, nil, pnrFloat(0))
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Ajukan("NSB-001", reqAjukanUang(10000))

	assert.Nil(t, result)
	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "syarat saldo penarikan uang")
	repo.AssertNotCalled(t, "LockSaldo", mock.Anything, mock.Anything, mock.Anything)
}

func TestAjukanPenarikan_RekeningTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", reqAjukanUang(10000))

	assert.Nil(t, result)
	assert.Equal(t, http.StatusNotFound, pnrKode(t, err))
	assert.Contains(t, err.Error(), "Rekening saldo")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestAjukanPenarikan_Barang_ProdukTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldoPoin(500), nil)
	repo.On("LockProduk", mock.Anything, "SMB-001", "BSM-001").Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", reqAjukanBarang(2))

	assert.Nil(t, result)
	assert.Equal(t, http.StatusNotFound, pnrKode(t, err))
	assert.Contains(t, err.Error(), "Barang produk")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestAjukanPenarikan_Barang_StokTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldoPoin(500), nil)
	repo.On("LockProduk", mock.Anything, "SMB-001", "BSM-001").Return(katalogBeras(), nil)
	repo.On("LockStok", mock.Anything, "SMB-001", "BSM-001").Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", reqAjukanBarang(2))

	assert.Nil(t, result)
	assert.Equal(t, http.StatusNotFound, pnrKode(t, err))
	assert.Contains(t, err.Error(), "Stok produk")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func langkahAjukanBarang(dbMock sqlmock.Sqlmock, repo *mocks.MockPenarikanRepo) []langkahTx {
	return []langkahTx{
		langkahRepo("kurangi saldo", func() *mock.Call { return repo.On("UpdateSaldo", mock.Anything, mock.Anything, float64(400)) }),
		langkahGenerateID("generate riwayat saldo", dbMock, "riwayat_saldo_id"),
		langkahRepo("catat riwayat saldo", func() *mock.Call { return repo.On("CreateRiwayat", mock.Anything, mock.Anything) }),
		langkahGenerateID("generate penarikan_id", dbMock, "penarikan_id"),
		langkahRepo("simpan penarikan", func() *mock.Call { return repo.On("SavePenarikan", mock.Anything, mock.Anything) }),
		langkahGenerateID("generate riwayat penarikan", dbMock, "riwayat_penarikan_id"),
		langkahRepo("simpan riwayat penarikan", func() *mock.Call { return repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything) }),
		langkahRepo("simpan detail barang", func() *mock.Call { return repo.On("SaveDetailBarang", mock.Anything, mock.Anything) }),
		langkahRepo("kurangi stok barang", func() *mock.Call { return repo.On("UpdateStok", mock.Anything, "SMB-001", "BSM-001", float64(-2)) }),
	}
}

func TestAjukanPenarikan_GagalDiDalamTx(t *testing.T) {
	for i, l := range langkahAjukanBarang(nil, nil) {
		t.Run(l.nama, func(t *testing.T) {
			dbMock, repo, notif, svc := newPenarikanSvc(t)
			repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
			penarikanBolehMenarik(repo)
			repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)
			repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldoPoin(500), nil)
			repo.On("LockProduk", mock.Anything, "SMB-001", "BSM-001").Return(katalogBeras(), nil)
			repo.On("LockStok", mock.Anything, "SMB-001", "BSM-001").Return(&models.StokBarang{ProdukID: "SMB-001", Stok: 10}, nil)
			dbMock.ExpectBegin()
			jalankanSampaiGagal(langkahAjukanBarang(dbMock, repo), i)
			dbMock.ExpectRollback()

			result, err := svc.Ajukan("NSB-001", reqAjukanBarang(2))

			assert.Nil(t, result)
			assert.Error(t, err)
			notif.AssertNotCalled(t, "NotifPengajuanPenarikan",
				mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

// Pengajuan uang Rp100.000 dari saldo Rp300.000 yang berhasil sampai commit.
func siapAjukanUangSukses(dbMock sqlmock.Sqlmock, repo *mocks.MockPenarikanRepo) {
	saldo := penarikanSaldo(300000)
	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	penarikanBolehMenarik(repo)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(200000)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("SavePenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindNasabahWithUser", "NSB-001").
		Return(&models.Nasabah{NasabahID: "NSB-001", User: models.User{UserID: "USR-NSB-001", Nama: "Budi"}}, nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"penarikan_id"}))
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectCommit()
}

func TestAjukanPenarikan_NotifKeSemuaPetugasBank(t *testing.T) {
	dbMock, repo, notif, svc := newPenarikanSvc(t)
	lepasStub(&repo.Mock, "FindNasabahWithUser")
	siapAjukanUangSukses(dbMock, repo)
	repo.On("FindAdminsByBankID", "BSM-001").Return([]repositories.AdminInfo{
		{UserID: "USR-PTG-1", FCMToken: "tok-1"},
		{UserID: "USR-PTG-2", FCMToken: "tok-2"},
	}, nil)
	terkirim := make(chan struct{}, 2)
	notif.On("NotifPengajuanPenarikan", mock.Anything, "USR-PTG-1", "tok-1", mock.Anything, mock.Anything, mock.Anything).
		Run(kirimSinyal(terkirim)).Return(errors.New("fcm down")).Once()
	notif.On("NotifPengajuanPenarikan", mock.Anything, "USR-PTG-2", "tok-2", mock.Anything, mock.Anything, mock.Anything).
		Run(kirimSinyal(terkirim)).Return(nil).Once()

	result, err := svc.Ajukan("NSB-001", reqAjukanUang(100000))

	assert.NoError(t, err)
	tungguSinyal(t, terkirim, 2)
	notif.AssertExpectations(t)
	for _, c := range notif.Calls {
		assert.Equal(t, "Budi", c.Arguments.String(3))
		assert.Equal(t, "Rp100000", c.Arguments.String(4))
		assert.Equal(t, result.PenarikanID, c.Arguments.String(5), "notif merujuk ke penarikan yang baru diajukan")
	}
}

func TestAjukanPenarikan_NotifPetugasGagalDiambil(t *testing.T) {
	dbMock, repo, notif, svc := newPenarikanSvc(t)
	lepasStub(&repo.Mock, "FindNasabahWithUser")
	siapAjukanUangSukses(dbMock, repo)
	dicari := make(chan struct{}, 1)
	repo.On("FindAdminsByBankID", "BSM-001").Run(kirimSinyal(dicari)).
		Return([]repositories.AdminInfo(nil), errors.New("db error"))

	_, err := svc.Ajukan("NSB-001", reqAjukanUang(100000))

	assert.NoError(t, err, "gagal ambil daftar petugas tidak membatalkan pengajuan")
	tungguSinyal(t, dicari, 1)
	notif.AssertNotCalled(t, "NotifPengajuanPenarikan",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ===== Konfirmasi =====

func TestKonfirmasiPenarikan_CatatanTerlaluPanjang(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001",
		services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr(strings.Repeat("a", 256))})

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	repo.AssertNotCalled(t, "FindAdminAktif", mock.Anything, mock.Anything)
}

func TestKonfirmasiPenarikan_AdminTanpaBank(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	admin := pnrPetugas()
	admin.BankID = nil
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001",
		services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")})

	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	repo.AssertNotCalled(t, "LockPenarikan", mock.Anything, mock.Anything)
}

func TestKonfirmasiPenarikan_TolakTanpaRewardID(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	p := penarikanBarangPending("RDM-003")
	p.RewardID = nil
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-003").Return(p, nil)
	repo.On("FindNasabah", mock.Anything, "NSB-001").Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectRollback()

	err := svc.Konfirmasi("RDM-003", "USR-ADM-001",
		services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("alasan")})

	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
	repo.AssertNotCalled(t, "LockSaldo", mock.Anything, mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestKonfirmasiPenarikan_TolakGagalDiDalamTx(t *testing.T) {
	for i, l := range langkahPengembalianBarang(nil, nil, nil) {
		t.Run(l.nama, func(t *testing.T) {
			dbMock, repo, _, svc := newPenarikanSvc(t)
			p := penarikanBarangPending("RDM-003")
			repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
			repo.On("LockPenarikan", mock.Anything, "RDM-003").Return(p, nil)
			repo.On("FindNasabah", mock.Anything, "NSB-001").Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
			dbMock.ExpectBegin()
			jalankanSampaiGagal(langkahPengembalianBarang(dbMock, repo, p), i)
			dbMock.ExpectRollback()

			err := svc.Konfirmasi("RDM-003", "USR-ADM-001",
				services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("Barang gak sesuai")})

			assert.Error(t, err)
			repo.AssertNotCalled(t, "FindPenarikanByID", mock.Anything)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

func TestKonfirmasiPenarikan_NotifDisetujui(t *testing.T) {
	dbMock, repo, notif, svc := newPenarikanSvc(t)
	p := &models.Penarikan{
		PenarikanID: "RDM-001", NasabahID: pNasabahID("NSB-001"), BankID: pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanPending,
	}
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
	repo.On("FindNasabah", mock.Anything, "NSB-001").Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindPenarikanByID", "RDM-001").Return(penarikanUntukNotif("RDM-001"), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectCommit()
	terkirim := make(chan struct{}, 1)
	notif.On("NotifPenarikanDisetujui", mock.Anything, "USR-NSB-001", "tok-nsb", mock.Anything, "RDM-001").
		Run(kirimSinyal(terkirim)).Return(errors.New("fcm down")).Once()

	deadline := validDeadlineStr()
	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", services.KonfirmasiPenarikanReq{Status: "approved", DeadlineJemput: deadline})

	assert.NoError(t, err)
	tungguSinyal(t, terkirim, 1)
	waktuJemput, _ := utils.ParseFlexibleDate(deadline)
	assert.Equal(t, utils.FormatTanggalIndo(waktuJemput), notif.Calls[0].Arguments.String(3), "isi notif = batas jemput")
	notif.AssertNotCalled(t, "NotifPenarikanDitolak", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestKonfirmasiPenarikan_NotifDitolak(t *testing.T) {
	dbMock, repo, notif, svc := newPenarikanSvc(t)
	p := &models.Penarikan{
		PenarikanID: "RDM-002", NasabahID: pNasabahID("NSB-001"), BankID: pBankID("BSM-001"),
		RewardID: pRewardID(1), NominalPenarikan: 100000, StatusPenarikan: models.StatusPenarikanPending,
		Reward: penarikanRewardUang(),
	}
	saldo := penarikanSaldo(50000)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-002").Return(p, nil)
	repo.On("FindNasabah", mock.Anything, "NSB-001").Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(150000)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindPenarikanByID", "RDM-002").Return(penarikanUntukNotif("RDM-002"), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	dbMock.ExpectCommit()
	terkirim := make(chan struct{}, 1)
	notif.On("NotifPenarikanDitolak", mock.Anything, "USR-NSB-001", "tok-nsb", "RDM-002").
		Run(kirimSinyal(terkirim)).Return(errors.New("fcm down")).Once()

	err := svc.Konfirmasi("RDM-002", "USR-ADM-001",
		services.KonfirmasiPenarikanReq{Status: "rejected", Catatan: ptrStr("Stok gak sesuai")})

	assert.NoError(t, err)
	tungguSinyal(t, terkirim, 1)
	notif.AssertExpectations(t)
	notif.AssertNotCalled(t, "NotifPenarikanDisetujui", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// ===== Selesai =====

func reqSelesaiManual() services.SelesaiPenarikanReq {
	return services.SelesaiPenarikanReq{
		NasabahID: "NSB-001", PenarikanID: "RDM-001", BuktiFoto: "https://foto.jpg", Catatan: ptrStr("Diambil langsung"),
	}
}

func penarikanApproved() *models.Penarikan {
	return &models.Penarikan{
		PenarikanID: "RDM-001", NasabahID: pNasabahID("NSB-001"), BankID: pBankID("BSM-001"),
		StatusPenarikan: models.StatusPenarikanApproved,
	}
}

func TestSelesaiPenarikan_QR_TanpaID(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	err := svc.Selesai("USR-ADM-001", services.SelesaiPenarikanReq{QRData: `{"type":"ENVIROO-PENARIKAN"}`})

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "wajib diisi")
	repo.AssertNotCalled(t, "FindAdminAktif", mock.Anything, mock.Anything)
}

func TestSelesaiPenarikan_CatatanTerlaluPanjang(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	req := reqSelesaiManual()
	req.Catatan = ptrStr(strings.Repeat("a", 256))

	err := svc.Selesai("USR-ADM-001", req)

	assert.Equal(t, http.StatusBadRequest, pnrKode(t, err))
	assert.Contains(t, err.Error(), "255 karakter")
	repo.AssertNotCalled(t, "FindAdminAktif", mock.Anything, mock.Anything)
}

func TestSelesaiPenarikan_AdminTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", reqSelesaiManual())

	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
}

func TestSelesaiPenarikan_AdminTanpaBank(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	admin := pnrPetugas()
	admin.BankID = nil
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", reqSelesaiManual())

	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	repo.AssertNotCalled(t, "LockPenarikan", mock.Anything, mock.Anything)
}

func TestSelesaiPenarikan_PenarikanTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(nil, errors.New("not found"))
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", reqSelesaiManual())

	assert.Equal(t, http.StatusNotFound, pnrKode(t, err))
}

func TestSelesaiPenarikan_BankMismatch(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	p := penarikanApproved()
	p.BankID = pBankID("BSM-002")
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", reqSelesaiManual())

	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	assert.Contains(t, err.Error(), "akses")
}

func TestSelesaiPenarikan_ConflictOfInterest(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(penarikanApproved(), nil)
	repo.On("FindNasabah", mock.Anything, "NSB-001").Return(nasabahMilik("NSB-001", "USR-ADM-001", "BSM-001"), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Selesai("USR-ADM-001", reqSelesaiManual())

	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	assert.Contains(t, err.Error(), "sendiri")
	repo.AssertNotCalled(t, "UpdateStatus", mock.Anything, mock.Anything, mock.Anything)
}

func TestSelesaiPenarikan_GagalDiDalamTx(t *testing.T) {
	langkah := func(dbMock sqlmock.Sqlmock, repo *mocks.MockPenarikanRepo, p *models.Penarikan) []langkahTx {
		return []langkahTx{
			langkahRepo("ubah status", func() *mock.Call { return repo.On("UpdateStatus", mock.Anything, p, mock.Anything) }),
			langkahGenerateID("generate riwayat penarikan", dbMock, "riwayat_penarikan_id"),
			langkahRepo("simpan riwayat penarikan", func() *mock.Call { return repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything) }),
		}
	}
	for i, l := range langkah(nil, nil, nil) {
		t.Run(l.nama, func(t *testing.T) {
			dbMock, repo, _, svc := newPenarikanSvc(t)
			p := penarikanApproved()
			repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
			repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
			repo.On("FindNasabah", mock.Anything, "NSB-001").Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
			dbMock.ExpectBegin()
			jalankanSampaiGagal(langkah(dbMock, repo, p), i)
			dbMock.ExpectRollback()

			err := svc.Selesai("USR-ADM-001", reqSelesaiManual())

			assert.Error(t, err)
			repo.AssertNotCalled(t, "FindPenarikanByID", mock.Anything)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

func TestSelesaiPenarikan_NotifBerhasil(t *testing.T) {
	dbMock, repo, notif, svc := newPenarikanSvc(t)
	p := penarikanApproved()
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
	repo.On("FindNasabah", mock.Anything, "NSB-001").Return(nasabahMilik("NSB-001", "USR-NASABAH-001", "BSM-001"), nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindPenarikanByID", "RDM-001").Return(penarikanUntukNotif("RDM-001"), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectCommit()
	terkirim := make(chan struct{}, 1)
	notif.On("NotifPenarikanBerhasil", mock.Anything, "USR-NSB-001", "tok-nsb", mock.Anything, "RDM-001").
		Run(kirimSinyal(terkirim)).Return(errors.New("fcm down")).Once()

	err := svc.Selesai("USR-ADM-001", reqSelesaiManual())

	assert.NoError(t, err)
	tungguSinyal(t, terkirim, 1)
	assert.Equal(t, "Rp100000", notif.Calls[0].Arguments.String(3), "isi notif = nominal yang dicairkan")
}

// ===== Batal =====

func TestBatalPenarikan_TanpaRewardID(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	p := penarikanBarangPending("RDM-003")
	p.RewardID = nil
	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-003").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("SaveRiwayatPenarikan", mock.Anything, mock.Anything).Return(nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_penarikan_id"}))
	dbMock.ExpectRollback()

	err := svc.Batal("RDM-003", "USR-NSB-001", "gak jadi")

	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
	assert.Contains(t, err.Error(), "reward")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestBatalPenarikan_BarangTanpaBankID(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)
	p := penarikanBarangPending("RDM-003")
	p.BankID = nil
	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-003").Return(p, nil)
	dbMock.ExpectBegin()
	for _, l := range langkahPengembalianBarang(dbMock, repo, p)[:7] {
		l.sukses()
	}
	dbMock.ExpectRollback()

	err := svc.Batal("RDM-003", "USR-NSB-001", "gak jadi")

	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
	assert.Contains(t, err.Error(), "bank")
	repo.AssertNotCalled(t, "FindDetailBarang", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestBatalPenarikan_GagalDiDalamTx(t *testing.T) {
	for i, l := range langkahPengembalianBarang(nil, nil, nil) {
		t.Run(l.nama, func(t *testing.T) {
			dbMock, repo, _, svc := newPenarikanSvc(t)
			p := penarikanBarangPending("RDM-003")
			repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
			repo.On("LockPenarikan", mock.Anything, "RDM-003").Return(p, nil)
			dbMock.ExpectBegin()
			jalankanSampaiGagal(langkahPengembalianBarang(dbMock, repo, p), i)
			dbMock.ExpectRollback()

			err := svc.Batal("RDM-003", "USR-NSB-001", "gak jadi")

			assert.Error(t, err)
			assert.NoError(t, dbMock.ExpectationsWereMet())
		})
	}
}

// ===== GetList / GetListByBank =====

func TestGetListPenarikan_RepoError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindByNasabahID", "NSB-001", repositories.PenarikanFilter{}).
		Return([]repositories.PenarikanListRow(nil), errors.New("db error"))

	result, err := svc.GetList("NSB-001", repositories.PenarikanFilter{})

	assert.Nil(t, result)
	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

func TestGetListByBankPenarikan_RepoError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindByBankID", "BSM-001", repositories.PenarikanFilter{}).
		Return([]models.Penarikan(nil), errors.New("db error"))

	result, err := svc.GetListByBank("BSM-001", repositories.PenarikanFilter{})

	assert.Nil(t, result)
	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

// ===== GetDetail =====

func penarikanUntukDetail() *models.Penarikan {
	return &models.Penarikan{
		PenarikanID: "RDM-001", NasabahID: pNasabahID("NSB-001"), BankID: pBankID("BSM-001"),
		RewardID: pRewardID(1), NominalPenarikan: 75000, SatuanPenarikan: models.SatuanRewardEnumRp,
		StatusPenarikan: models.StatusPenarikanPending, Reward: penarikanRewardUang(),
	}
}

func TestGetDetailPenarikan_Nasabah_PenarikanTanpaNasabah(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	p := penarikanUntukDetail()
	p.NasabahID = nil
	repo.On("FindPenarikanByID", "RDM-001").Return(p, nil)

	result, err := svc.GetDetail("RDM-001", "USR-NSB-001", models.RoleNasabah)

	assert.Nil(t, result)
	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	repo.AssertNotCalled(t, "CountNasabahMilikUser", mock.Anything, mock.Anything)
}

func TestGetDetailPenarikan_Nasabah_BukanMiliknya(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindPenarikanByID", "RDM-001").Return(penarikanUntukDetail(), nil)
	repo.On("CountNasabahMilikUser", "USR-NSB-LAIN", "NSB-001").Return(int64(0), nil)

	result, err := svc.GetDetail("RDM-001", "USR-NSB-LAIN", models.RoleNasabah)

	assert.Nil(t, result)
	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	repo.AssertNotCalled(t, "FindRiwayatPenarikan", mock.Anything)
}

func TestGetDetailPenarikan_Nasabah_Miliknya(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindPenarikanByID", "RDM-001").Return(penarikanUntukDetail(), nil)
	repo.On("CountNasabahMilikUser", "USR-NSB-001", "NSB-001").Return(int64(1), nil)
	repo.On("FindRiwayatPenarikan", "RDM-001").Return([]repositories.RiwayatPenarikanRow{}, nil)

	result, err := svc.GetDetail("RDM-001", "USR-NSB-001", models.RoleNasabah)

	assert.NoError(t, err)
	if assert.NotNil(t, result) {
		assert.Equal(t, "RDM-001", result.Header.PenarikanID)
	}
}

func TestGetDetailPenarikan_Petugas_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindPenarikanByID", "RDM-001").Return(penarikanUntukDetail(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(nil, errors.New("not found"))

	result, err := svc.GetDetail("RDM-001", "USR-ADM-001", models.PetugasBSM)

	assert.Nil(t, result)
	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
}

func TestGetDetailPenarikan_Petugas_BankLain(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	admin := pnrPetugas()
	admin.BankID = pBankID("BSM-002")
	repo.On("FindPenarikanByID", "RDM-001").Return(penarikanUntukDetail(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(admin, nil)

	result, err := svc.GetDetail("RDM-001", "USR-ADM-001", models.PetugasBSM)

	assert.Nil(t, result)
	assert.Equal(t, http.StatusForbidden, pnrKode(t, err))
	repo.AssertNotCalled(t, "FindRiwayatPenarikan", mock.Anything)
}

func TestGetDetailPenarikan_Petugas_BankSama(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindPenarikanByID", "RDM-001").Return(penarikanUntukDetail(), nil)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrPetugas(), nil)
	repo.On("FindRiwayatPenarikan", "RDM-001").Return([]repositories.RiwayatPenarikanRow{}, nil)

	result, err := svc.GetDetail("RDM-001", "USR-ADM-001", models.PetugasBSM)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestGetDetailPenarikan_RiwayatError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindPenarikanByID", "RDM-001").Return(penarikanUntukDetail(), nil)
	repo.On("FindRiwayatPenarikan", "RDM-001").Return([]repositories.RiwayatPenarikanRow(nil), errors.New("db error"))

	result, err := svc.GetDetail("RDM-001", "USR-SUPER-001", models.SuperAdmin)

	assert.Nil(t, result)
	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

// ===== Setting =====

func TestGetSettingPenarikan_RepoError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindSettingPenarikan", "BSM-001").Return(nil, errors.New("db error"))

	hasil, err := svc.GetSettingPenarikan("BSM-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

func TestSimpanSettingWaktu_UpdateSimpanError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-001"), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(pnrSetting(3, models.SatuanWaktuBulan), nil)
	repo.On("SaveSettingPenarikan", mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 2, SatuanWaktu: models.SatuanWaktuMinggu})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
}

func TestSimpanSettingWaktu_CekSettingError(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)
	repo.On("FindAdminAktif", mock.Anything, "USR-ADM-001").Return(pnrAdmin("BSM-001"), nil)
	repo.On("FindSettingPenarikan", "BSM-001").Return(nil, errors.New("db error"))

	hasil, err := svc.SimpanSettingWaktuPenarikan("BSM-001", "USR-ADM-001",
		services.SimpanSettingWaktuReq{MinWaktu: 2, SatuanWaktu: models.SatuanWaktuMinggu})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnrKode(t, err))
	repo.AssertNotCalled(t, "CreateSettingPenarikan", mock.Anything)
}
