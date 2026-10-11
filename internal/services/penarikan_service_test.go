package services_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
	repo.On("FindReward", 99).Return(nil, errors.New("not found"))

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 99, NominalPenarikan: 10000})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reward")
}

func TestPreviewPenarikan_ValidasiNominalNol(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 0})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Nominal")
}

func TestPreviewPenarikan_ValidasiBarangTanpaItem(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindReward", 2).Return(penarikanRewardBarang(), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 2})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Item produk")
}

func TestPreviewPenarikan_SaldoTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
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
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 0})

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestAjukanPenarikan_BarangDuplikat(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
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
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 50000})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deadline_konfirmasi")
}

func TestAjukanPenarikan_DeadlineKonfirmasiFormatSalah(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
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
