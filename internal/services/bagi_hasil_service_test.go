package services_test

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"
	"errors"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()
	sqlDB, dbMock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	// GORM MySQL driver queries VERSION() on connect
	dbMock.ExpectQuery("SELECT VERSION()").
		WillReturnRows(sqlmock.NewRows([]string{"VERSION()"}).AddRow("8.0.0"))
	gdb, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return gdb, dbMock
}

func expectTx(m sqlmock.Sqlmock) {
	m.ExpectBegin()
	m.ExpectCommit()
}

func expectTxRollback(m sqlmock.Sqlmock) {
	m.ExpectBegin()
	m.ExpectRollback()
}

// ── helpers ──────────────────────────────────────────────────────────────────

func ptrStr(s string) *string { return &s }

func baseReward() models.Reward {
	return models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang}
}

func basePenjualan(status models.StatusBagiHasilEnum) *models.Penjualan {
	r := baseReward()
	return &models.Penjualan{
		PenjualanID:     "PJL-001",
		BankID:          "BSM-001",
		TotalPenjualan:  500000,
		SatuanReward:    models.SatuanRewardEnumRp,
		RewardID:        1,
		Reward:          r,
		StatusBagiHasil: status,
	}
}

func baseTabunganNasabah(tabunganID, nasabahID, sampahID string, sisaQty float64) models.TabunganSampah {
	return models.TabunganSampah{
		TabunganID: tabunganID,
		NasabahID:  ptrStr(nasabahID),
		SampahID:   sampahID,
		Entitas:    models.EntitasNasabah,
		Qty:        sisaQty,
		SisaQty:    sisaQty,
		CreatedAt:  time.Now(),
	}
}

func baseTabunganBSU(tabunganID, bankID, sampahID string, sisaQty float64) models.TabunganSampah {
	return models.TabunganSampah{
		TabunganID: tabunganID,
		BankID:     ptrStr(bankID),
		SampahID:   sampahID,
		Entitas:    models.EntitasBankSampah,
		Qty:        sisaQty,
		SisaQty:    sisaQty,
		CreatedAt:  time.Now(),
	}
}

// ── Tests: validasi awal ──────────────────────────────────────────────────────

func TestSubmitBagiHasil_SudahDilakukan(t *testing.T) {
	gdb, _ := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(basePenjualan(models.BagiHasilBerhasil), nil)

	_, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sudah dilakukan")
}

func TestSubmitBagiHasil_BSUTidakBisaBagiHasil(t *testing.T) {
	gdb, _ := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSU-001", JenisBank: models.BSU}
	repo.On("GetPenjualanWithReward", "PJL-001", "BSU-001").Return(basePenjualan(models.BagiHasilPending), nil)
	repo.On("GetBankByID", "BSU-001").Return(bank, nil)

	_, err := svc.Submit("PJL-001", "BSU-001", "ADM-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BSU tidak dapat")
}

func TestSubmitBagiHasil_PenjualanTidakDitemukan(t *testing.T) {
	gdb, _ := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	repo.On("GetPenjualanWithReward", "PJL-XXX", "BSM-001").Return(nil, errors.New("not found"))

	_, err := svc.Submit("PJL-XXX", "BSM-001", "ADM-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Penjualan tidak ditemukan")
}

// ── Tests: BSM sukses ─────────────────────────────────────────────────────────

func TestSubmitBagiHasil_BSM_Sukses(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 10, HargaNasabahSnapshot: 5000},
	}
	nasabahID := "NSB-001"
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", nasabahID, "SMP-001", 10),
	}
	nasabah := &models.Nasabah{
		NasabahID: nasabahID,
		User:      models.User{UserID: "USR-001", Nama: "Budi"},
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(basePenjualan(models.BagiHasilPending), nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, "SMP-001", float64(5000), models.SatuanRewardEnumRp, "ADM-001").Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", nasabahID).Return(nasabah, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(500000), result.GrossBank)
	assert.Equal(t, float64(50000), result.TotalDistribusiNasabah) // 10 * 5000
	assert.Equal(t, float64(450000), result.SisaBagiHasil)
	repo.AssertExpectations(t)
}

// ── Tests: stok tidak cukup ───────────────────────────────────────────────────

func TestSubmitBagiHasil_StokTidakCukup(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 10, HargaNasabahSnapshot: 5000},
	}
	// hanya 5kg tersedia, butuh 10kg
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", "NSB-001", "SMP-001", 5),
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(basePenjualan(models.BagiHasilPending), nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)

	expectTxRollback(dbMock)

	_, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.Error(t, err)
	se, ok := err.(*services.ServiceError)
	assert.True(t, ok)
	assert.Equal(t, 422, se.Code)
}

// ── Tests: BSI dengan BSU ─────────────────────────────────────────────────────

func TestSubmitBagiHasil_BSI_DenganBSU(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	r := baseReward()
	penjualan := &models.Penjualan{
		PenjualanID:     "PJL-002",
		BankID:          "BSI-001",
		TotalPenjualan:  200000,
		SatuanReward:    models.SatuanRewardEnumRp,
		RewardID:        1,
		Reward:          r,
		StatusBagiHasil: models.BagiHasilPending,
	}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI}
	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Satu"}}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 10, HargaNasabahSnapshot: 5000},
	}
	// 6kg nasabah BSI, 4kg nasabah BSU
	tabunganNasabah := []models.TabunganSampah{
		baseTabunganNasabah("TBG-BSI", "NSB-BSI", "SMP-001", 6),
		baseTabunganNasabah("TBG-BSU-NSB", "NSB-BSU", "SMP-001", 4),
	}
	// stok fisik BSU: 4kg
	tabunganBSU := []models.TabunganSampah{
		baseTabunganBSU("TBG-BSU-STK", "BSU-001", "SMP-001", 4),
	}

	repo.On("GetPenjualanWithReward", "PJL-002", "BSI-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSI-001").Return(bank, nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-002").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSI-001", "BSU-001"}).Return(tabunganNasabah, nil)
	repo.On("GetTabunganBSU", mock.Anything, "SMP-001", []string{"BSU-001"}).Return(tabunganBSU, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, "SMP-001", float64(5000), models.SatuanRewardEnumRp, "ADM-001").Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-BSI").Return(&models.Nasabah{NasabahID: "NSB-BSI", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-BSU").Return(&models.Nasabah{NasabahID: "NSB-BSU", User: models.User{UserID: "USR-2"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-002", "BSI-001", "ADM-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	// total distribusi: (6+4) * 5000 = 50000
	assert.Equal(t, float64(50000), result.TotalDistribusiNasabah)
	assert.Equal(t, float64(150000), result.SisaBagiHasil)
	assert.Len(t, result.NasabahNotifs, 2)
	// CreatePerhitunganSisa dipanggil 1x untuk BSU stok 4kg
	repo.AssertNumberOfCalls(t, "CreatePerhitunganSisa", 1)
	repo.AssertExpectations(t)
}

// ── Tests: BSM saldo bank diupdate ───────────────────────────────────────────

func TestSubmitBagiHasil_BSM_SisaMasukSaldoBank(t *testing.T) {
	// BSM selalu dapat sisa bagi hasil → UpdateSaldoRekening dipanggil untuk bank juga
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	// nasabah hanya dapat 30000 dari total 100000 → sisa 70000 masuk bank
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 6, HargaNasabahSnapshot: 5000},
	}
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", "NSB-001", "SMP-001", 6),
	}
	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 100000

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-001").Return(&models.Nasabah{NasabahID: "NSB-001", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(30000), result.TotalDistribusiNasabah) // 6 * 5000
	assert.Equal(t, float64(70000), result.SisaBagiHasil)
	// UpdateSaldoRekening: 1x nasabah + 1x bank (sisa)
	repo.AssertNumberOfCalls(t, "UpdateSaldoRekening", 2)
}

func TestSubmitBagiHasil_BSI_Uang_SisaTidakMasukBank(t *testing.T) {
	// BSI dengan reward Uang → bankGetsSisa = false → saldo bank tidak diupdate
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	r := models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang}
	penjualan := &models.Penjualan{
		PenjualanID:     "PJL-001",
		BankID:          "BSI-001",
		TotalPenjualan:  100000,
		SatuanReward:    models.SatuanRewardEnumRp,
		RewardID:        1,
		Reward:          r,
		StatusBagiHasil: models.BagiHasilPending,
	}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 5, HargaNasabahSnapshot: 5000},
	}
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", "NSB-001", "SMP-001", 5),
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSI-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSI-001").Return(bank, nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return([]models.BankSampah{}, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSI-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-001").Return(&models.Nasabah{NasabahID: "NSB-001", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSI-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(75000), result.SisaBagiHasil) // 100000 - 25000
	// hanya 1x UpdateSaldoRekening untuk nasabah, bank tidak dapat sisa
	repo.AssertNumberOfCalls(t, "UpdateSaldoRekening", 1)
}

func TestSubmitBagiHasil_BSI_Sembako_SisaMasukBank(t *testing.T) {
	// BSI dengan reward Sembako → bankGetsSisa = true
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	r := models.Reward{RewardID: 2, NamaReward: models.RewardEnumSembako}
	penjualan := &models.Penjualan{
		PenjualanID:     "PJL-001",
		BankID:          "BSI-001",
		TotalPenjualan:  1000,
		SatuanReward:    models.SatuanRewardEnumPoin,
		RewardID:        2,
		Reward:          r,
		StatusBagiHasil: models.BagiHasilPending,
	}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 4, HargaNasabahSnapshot: 10},
	}
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", "NSB-001", "SMP-001", 4),
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSI-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSI-001").Return(bank, nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return([]models.BankSampah{}, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSI-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-001").Return(&models.Nasabah{NasabahID: "NSB-001", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSI-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(40), result.TotalDistribusiNasabah) // 4 * 10
	assert.Equal(t, float64(960), result.SisaBagiHasil)
	// 1x nasabah + 1x bank (sisa)
	repo.AssertNumberOfCalls(t, "UpdateSaldoRekening", 2)
}

// ── Tests: BSM multiple sampah ────────────────────────────────────────────────

func TestSubmitBagiHasil_BSM_MultipleSampah(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	// dua jenis sampah dalam satu penjualan
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 5, HargaNasabahSnapshot: 2000},
		{SampahID: "SMP-002", Qty: 3, HargaNasabahSnapshot: 3000},
	}
	tabSmp1 := []models.TabunganSampah{
		baseTabunganNasabah("TBG-A1", "NSB-A", "SMP-001", 5),
	}
	tabSmp2 := []models.TabunganSampah{
		baseTabunganNasabah("TBG-B1", "NSB-B", "SMP-002", 3),
	}
	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 50000

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabSmp1, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-002", []string{"BSM-001"}).Return(tabSmp2, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-A").Return(&models.Nasabah{NasabahID: "NSB-A", User: models.User{UserID: "USR-A"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-B").Return(&models.Nasabah{NasabahID: "NSB-B", User: models.User{UserID: "USR-B"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	// NSB-A: 5*2000=10000, NSB-B: 3*3000=9000 → total 19000
	assert.Equal(t, float64(19000), result.TotalDistribusiNasabah)
	assert.Equal(t, float64(31000), result.SisaBagiHasil)
	// GetTabunganNasabah dipanggil 2x (per sampah)
	repo.AssertNumberOfCalls(t, "GetTabunganNasabah", 2)
}

// ── Tests: partial FIFO ───────────────────────────────────────────────────────

func TestSubmitBagiHasil_PartialFIFO(t *testing.T) {
	// tabungan NSB-A punya 20kg, penjualan hanya 8kg → sisa 12kg tetap di tabungan
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 8, HargaNasabahSnapshot: 1000},
	}
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", "NSB-001", "SMP-001", 20),
	}
	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 20000

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.MatchedBy(func(t *models.TabunganSampah) bool {
		return t.SisaQty == 12 // 20 - 8 = 12 tersisa
	})).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-001").Return(&models.Nasabah{NasabahID: "NSB-001", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(8000), result.TotalDistribusiNasabah) // 8 * 1000
	repo.AssertExpectations(t)
}

// ── Tests: kerugian angkut BSU ────────────────────────────────────────────────

func TestSubmitBagiHasil_BSI_KerugianAngkutBSU(t *testing.T) {
	// Nasabah BSU setor 10kg → tabungan nasabah 10kg
	// Pengangkutan sah hanya 8kg → tabungan BSU (bank_sampah) 8kg
	// CreatePerhitunganSisa hanya untuk 8kg (bukan 10kg)
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	r := baseReward()
	penjualan := &models.Penjualan{
		PenjualanID:     "PJL-001",
		BankID:          "BSI-001",
		TotalPenjualan:  50000,
		SatuanReward:    models.SatuanRewardEnumRp,
		RewardID:        1,
		Reward:          r,
		StatusBagiHasil: models.BagiHasilPending,
	}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI}
	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Satu"}}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 10, HargaNasabahSnapshot: 2000},
	}
	// FIFO nasabah: 10kg dari nasabah BSU (karena nasabah BSU lebih tua)
	tabunganNasabah := []models.TabunganSampah{
		baseTabunganNasabah("TBG-NSB", "NSB-BSU", "SMP-001", 10),
	}
	// stok fisik BSU hanya 8kg (kerugian 2kg saat angkut)
	tabunganBSU := []models.TabunganSampah{
		baseTabunganBSU("TBG-STK", "BSU-001", "SMP-001", 8),
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSI-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSI-001").Return(bank, nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSI-001", "BSU-001"}).Return(tabunganNasabah, nil)
	repo.On("GetTabunganBSU", mock.Anything, "SMP-001", []string{"BSU-001"}).Return(tabunganBSU, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	// CreatePerhitunganSisa hanya 1x dengan qty 8 (bukan 10)
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.MatchedBy(func(ps *models.PerhitunganSisa) bool {
		return ps.QtyDipakai == 8 && ps.NilaiDipakai == 16000 // 8 * 2000
	})).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-BSU").Return(&models.Nasabah{NasabahID: "NSB-BSU", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	_, err := svc.Submit("PJL-001", "BSI-001", "ADM-001")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// ── Tests: BSI multiple BSU ───────────────────────────────────────────────────

func TestSubmitBagiHasil_BSI_MultipleBSU(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	r := baseReward()
	penjualan := &models.Penjualan{
		PenjualanID:     "PJL-001",
		BankID:          "BSI-001",
		TotalPenjualan:  100000,
		SatuanReward:    models.SatuanRewardEnumRp,
		RewardID:        1,
		Reward:          r,
		StatusBagiHasil: models.BagiHasilPending,
	}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI}
	bsuList := []models.BankSampah{
		{BankID: "BSU-001", NamaBank: "BSU Satu"},
		{BankID: "BSU-002", NamaBank: "BSU Dua"},
	}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 10, HargaNasabahSnapshot: 3000},
	}
	tabunganNasabah := []models.TabunganSampah{
		baseTabunganNasabah("TBG-N1", "NSB-BSU1", "SMP-001", 6),
		baseTabunganNasabah("TBG-N2", "NSB-BSU2", "SMP-001", 4),
	}
	// stok fisik: BSU-001 6kg, BSU-002 4kg
	tabunganBSU := []models.TabunganSampah{
		baseTabunganBSU("TBG-S1", "BSU-001", "SMP-001", 6),
		baseTabunganBSU("TBG-S2", "BSU-002", "SMP-001", 4),
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSI-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSI-001").Return(bank, nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSI-001", "BSU-001", "BSU-002"}).Return(tabunganNasabah, nil)
	repo.On("GetTabunganBSU", mock.Anything, "SMP-001", []string{"BSU-001", "BSU-002"}).Return(tabunganBSU, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-BSU1").Return(&models.Nasabah{NasabahID: "NSB-BSU1", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-BSU2").Return(&models.Nasabah{NasabahID: "NSB-BSU2", User: models.User{UserID: "USR-2"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSI-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(30000), result.TotalDistribusiNasabah) // 10 * 3000
	// CreatePerhitunganSisa 2x: satu untuk BSU-001, satu untuk BSU-002
	repo.AssertNumberOfCalls(t, "CreatePerhitunganSisa", 2)
	repo.AssertExpectations(t)
}

// ── Tests: BSI banyak BSU, tiap BSU banyak nasabah ───────────────────────────

func TestSubmitBagiHasil_BSI_DuaBSU_BanyakNasabah(t *testing.T) {
	// Skenario:
	//   BSI-001 punya 2 BSU (BSU-001, BSU-002)
	//   BSU-001: NSB-A1 (3kg), NSB-A2 (5kg) — stok fisik 6kg (kerugian 2kg)
	//   BSU-002: NSB-B1 (4kg), NSB-B2 (4kg) — stok fisik 8kg (pas)
	//   BSI direct: NSB-BSI (4kg)
	//   Penjualan: SMP-001, 20kg, harga 1500, total penjualan 80000
	//
	// Ekspektasi:
	//   SaveTabungan 7x     (5 nasabah + 2 BSU physical)
	//   CreatePerhitunganSisa 2x  (TBG-S1 qty=6, TBG-S2 qty=8)
	//   CreatePencairanTabungan 5x (satu per nasabah tabungan)
	//   UpdateSaldoRekening 5x   (BSI reward Uang → bankGetsSisa=false)
	//   NasabahNotifs 5 entries

	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	r := baseReward() // RewardEnumUang
	penjualan := &models.Penjualan{
		PenjualanID:     "PJL-001",
		BankID:          "BSI-001",
		TotalPenjualan:  80000,
		SatuanReward:    models.SatuanRewardEnumRp,
		RewardID:        1,
		Reward:          r,
		StatusBagiHasil: models.BagiHasilPending,
	}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI}
	bsuList := []models.BankSampah{
		{BankID: "BSU-001", NamaBank: "BSU Satu"},
		{BankID: "BSU-002", NamaBank: "BSU Dua"},
	}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 20, HargaNasabahSnapshot: 1500},
	}

	// FIFO nasabah diurutkan created_at ASC (oldest first)
	t0 := time.Now()
	tabunganNasabah := []models.TabunganSampah{
		{TabunganID: "TBG-BSI-N", NasabahID: ptrStr("NSB-BSI"), SampahID: "SMP-001", Entitas: models.EntitasNasabah, Qty: 4, SisaQty: 4, CreatedAt: t0.Add(-5 * time.Hour)},
		{TabunganID: "TBG-A1-N", NasabahID: ptrStr("NSB-A1"), SampahID: "SMP-001", Entitas: models.EntitasNasabah, Qty: 3, SisaQty: 3, CreatedAt: t0.Add(-4 * time.Hour)},
		{TabunganID: "TBG-A2-N", NasabahID: ptrStr("NSB-A2"), SampahID: "SMP-001", Entitas: models.EntitasNasabah, Qty: 5, SisaQty: 5, CreatedAt: t0.Add(-3 * time.Hour)},
		{TabunganID: "TBG-B1-N", NasabahID: ptrStr("NSB-B1"), SampahID: "SMP-001", Entitas: models.EntitasNasabah, Qty: 4, SisaQty: 4, CreatedAt: t0.Add(-2 * time.Hour)},
		{TabunganID: "TBG-B2-N", NasabahID: ptrStr("NSB-B2"), SampahID: "SMP-001", Entitas: models.EntitasNasabah, Qty: 4, SisaQty: 4, CreatedAt: t0.Add(-1 * time.Hour)},
	}
	// Stok fisik BSU (bank_sampah):
	// BSU-001: 6kg (< 8kg nasabah → kerugian 2kg ditanggung BSU-001)
	// BSU-002: 8kg (pas dengan total nasabah BSU-002)
	tabunganBSU := []models.TabunganSampah{
		{TabunganID: "TBG-S1", BankID: ptrStr("BSU-001"), SampahID: "SMP-001", Entitas: models.EntitasBankSampah, Qty: 6, SisaQty: 6, CreatedAt: t0.Add(-4 * time.Hour)},
		{TabunganID: "TBG-S2", BankID: ptrStr("BSU-002"), SampahID: "SMP-001", Entitas: models.EntitasBankSampah, Qty: 8, SisaQty: 8, CreatedAt: t0.Add(-2 * time.Hour)},
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSI-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSI-001").Return(bank, nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSI-001", "BSU-001", "BSU-002"}).Return(tabunganNasabah, nil)
	repo.On("GetTabunganBSU", mock.Anything, "SMP-001", []string{"BSU-001", "BSU-002"}).Return(tabunganBSU, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	// PerhitunganSisa harus mencerminkan stok fisik, bukan stok nasabah
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.MatchedBy(func(ps *models.PerhitunganSisa) bool {
		return ps.TabunganID == "TBG-S1" && ps.QtyDipakai == 6 && ps.NilaiDipakai == 9000 // 6*1500
	})).Return(nil)
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.MatchedBy(func(ps *models.PerhitunganSisa) bool {
		return ps.TabunganID == "TBG-S2" && ps.QtyDipakai == 8 && ps.NilaiDipakai == 12000 // 8*1500
	})).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-BSI").Return(&models.Nasabah{NasabahID: "NSB-BSI", User: models.User{UserID: "USR-0"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-A1").Return(&models.Nasabah{NasabahID: "NSB-A1", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-A2").Return(&models.Nasabah{NasabahID: "NSB-A2", User: models.User{UserID: "USR-2"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-B1").Return(&models.Nasabah{NasabahID: "NSB-B1", User: models.User{UserID: "USR-3"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-B2").Return(&models.Nasabah{NasabahID: "NSB-B2", User: models.User{UserID: "USR-4"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSI-001", "ADM-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(30000), result.TotalDistribusiNasabah) // 20 * 1500
	assert.Equal(t, float64(50000), result.SisaBagiHasil)          // 80000 - 30000
	assert.Len(t, result.NasabahNotifs, 5)

	// 5 nasabah tabungan + 2 BSU physical tabungan = 7 SaveTabungan
	repo.AssertNumberOfCalls(t, "SaveTabungan", 7)
	// 2 BSU physical entries → 2 PerhitunganSisa (nilai mencerminkan kerugian BSU-001)
	repo.AssertNumberOfCalls(t, "CreatePerhitunganSisa", 2)
	// Satu PencairanTabungan per nasabah tabungan yang dikonsumsi
	repo.AssertNumberOfCalls(t, "CreatePencairanTabungan", 5)
	// BSI uang → bankGetsSisa=false → hanya nasabah (5x)
	repo.AssertNumberOfCalls(t, "UpdateSaldoRekening", 5)
	repo.AssertExpectations(t)
}

// ── Tests: BSI banyak BSU, banyak nasabah, dua jenis sampah ──────────────────

func TestSubmitBagiHasil_BSI_DuaBSU_DuaSampah(t *testing.T) {
	// Seperti test sebelumnya tapi ada dua jenis sampah.
	// SMP-001: 10kg harga 2000 — NSB-A1 (4kg, BSU-001), NSB-B1 (6kg, BSU-002)
	// SMP-002: 6kg  harga 3000 — NSB-A2 (3kg, BSU-001), NSB-B2 (3kg, BSU-002)
	// BSU-001 stok: SMP-001=4kg, SMP-002=3kg (pas)
	// BSU-002 stok: SMP-001=5kg (< 6kg → kerugian 1kg), SMP-002=3kg (pas)
	// TotalPenjualan: 100000

	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	r := baseReward()
	penjualan := &models.Penjualan{
		PenjualanID:     "PJL-001",
		BankID:          "BSI-001",
		TotalPenjualan:  100000,
		SatuanReward:    models.SatuanRewardEnumRp,
		RewardID:        1,
		Reward:          r,
		StatusBagiHasil: models.BagiHasilPending,
	}
	bank := &models.BankSampah{BankID: "BSI-001", JenisBank: models.BSI}
	bsuList := []models.BankSampah{
		{BankID: "BSU-001"}, {BankID: "BSU-002"},
	}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 10, HargaNasabahSnapshot: 2000},
		{SampahID: "SMP-002", Qty: 6, HargaNasabahSnapshot: 3000},
	}

	tabSmp1 := []models.TabunganSampah{
		baseTabunganNasabah("TBG-A1-1", "NSB-A1", "SMP-001", 4),
		baseTabunganNasabah("TBG-B1-1", "NSB-B1", "SMP-001", 6),
	}
	tabSmp2 := []models.TabunganSampah{
		baseTabunganNasabah("TBG-A2-2", "NSB-A2", "SMP-002", 3),
		baseTabunganNasabah("TBG-B2-2", "NSB-B2", "SMP-002", 3),
	}
	bsuTabSmp1 := []models.TabunganSampah{
		{TabunganID: "TBG-S1-1", BankID: ptrStr("BSU-001"), SampahID: "SMP-001", Entitas: models.EntitasBankSampah, Qty: 4, SisaQty: 4},
		{TabunganID: "TBG-S2-1", BankID: ptrStr("BSU-002"), SampahID: "SMP-001", Entitas: models.EntitasBankSampah, Qty: 5, SisaQty: 5}, // kerugian 1kg
	}
	bsuTabSmp2 := []models.TabunganSampah{
		{TabunganID: "TBG-S1-2", BankID: ptrStr("BSU-001"), SampahID: "SMP-002", Entitas: models.EntitasBankSampah, Qty: 3, SisaQty: 3},
		{TabunganID: "TBG-S2-2", BankID: ptrStr("BSU-002"), SampahID: "SMP-002", Entitas: models.EntitasBankSampah, Qty: 3, SisaQty: 3},
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSI-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSI-001").Return(bank, nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSI-001", "BSU-001", "BSU-002"}).Return(tabSmp1, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-002", []string{"BSI-001", "BSU-001", "BSU-002"}).Return(tabSmp2, nil)
	repo.On("GetTabunganBSU", mock.Anything, "SMP-001", []string{"BSU-001", "BSU-002"}).Return(bsuTabSmp1, nil)
	repo.On("GetTabunganBSU", mock.Anything, "SMP-002", []string{"BSU-001", "BSU-002"}).Return(bsuTabSmp2, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	// SMP-001: BSU-001=4kg*2000=8000, BSU-002=5kg*2000=10000
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.MatchedBy(func(ps *models.PerhitunganSisa) bool {
		return ps.TabunganID == "TBG-S1-1" && ps.QtyDipakai == 4 && ps.NilaiDipakai == 8000
	})).Return(nil)
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.MatchedBy(func(ps *models.PerhitunganSisa) bool {
		return ps.TabunganID == "TBG-S2-1" && ps.QtyDipakai == 5 && ps.NilaiDipakai == 10000
	})).Return(nil)
	// SMP-002: BSU-001=3kg*3000=9000, BSU-002=3kg*3000=9000
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.MatchedBy(func(ps *models.PerhitunganSisa) bool {
		return ps.TabunganID == "TBG-S1-2" && ps.QtyDipakai == 3 && ps.NilaiDipakai == 9000
	})).Return(nil)
	repo.On("CreatePerhitunganSisa", mock.Anything, mock.MatchedBy(func(ps *models.PerhitunganSisa) bool {
		return ps.TabunganID == "TBG-S2-2" && ps.QtyDipakai == 3 && ps.NilaiDipakai == 9000
	})).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-A1").Return(&models.Nasabah{NasabahID: "NSB-A1", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-B1").Return(&models.Nasabah{NasabahID: "NSB-B1", User: models.User{UserID: "USR-2"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-A2").Return(&models.Nasabah{NasabahID: "NSB-A2", User: models.User{UserID: "USR-3"}}, nil)
	repo.On("GetNasabahWithUser", "NSB-B2").Return(&models.Nasabah{NasabahID: "NSB-B2", User: models.User{UserID: "USR-4"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSI-001", "ADM-001")

	assert.NoError(t, err)
	// SMP-001: (4+6)*2000=20000, SMP-002: (3+3)*3000=18000 → total 38000
	assert.Equal(t, float64(38000), result.TotalDistribusiNasabah)
	assert.Equal(t, float64(62000), result.SisaBagiHasil) // 100000 - 38000
	assert.Len(t, result.NasabahNotifs, 4)

	// 4 nasabah tabungan (SMP-001: 2, SMP-002: 2) + 4 BSU physical (2 per sampah) = 8x SaveTabungan
	repo.AssertNumberOfCalls(t, "SaveTabungan", 8)
	// 4 BSU physical entries (2 sampah × 2 BSU) → 4x CreatePerhitunganSisa
	repo.AssertNumberOfCalls(t, "CreatePerhitunganSisa", 4)
	// 4 nasabah tabungan dikonsumsi → 4x PencairanTabungan
	repo.AssertNumberOfCalls(t, "CreatePencairanTabungan", 4)
	repo.AssertExpectations(t)
}

// ── Tests: edge cases ─────────────────────────────────────────────────────────

func TestSubmitBagiHasil_SkipItemHargaNol(t *testing.T) {
	// item dengan HargaNasabahSnapshot=0 dilewati, tidak mengurangi tabungan
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 5, HargaNasabahSnapshot: 1000},
		{SampahID: "SMP-002", Qty: 3, HargaNasabahSnapshot: 0}, // dilewati
	}
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", "NSB-001", "SMP-001", 5),
	}
	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 10000

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-001").Return(&models.Nasabah{NasabahID: "NSB-001", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(5000), result.TotalDistribusiNasabah) // hanya SMP-001
	// GetTabunganNasabah hanya dipanggil 1x (SMP-002 dilewati karena harga=0)
	repo.AssertNumberOfCalls(t, "GetTabunganNasabah", 1)
}

func TestSubmitBagiHasil_SaveTabunganError_Rollback(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 5, HargaNasabahSnapshot: 1000},
	}
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", "NSB-001", "SMP-001", 5),
	}

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(basePenjualan(models.BagiHasilPending), nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(errors.New("db error"))

	expectTxRollback(dbMock)

	_, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Gagal update tabungan")
	// CreateBagiHasil tidak pernah dipanggil karena rollback
	repo.AssertNotCalled(t, "CreateBagiHasil")
}

// ── Tests: BSI FIFO konsumsi dari dua tabungan ────────────────────────────────

func TestSubmitBagiHasil_FIFODuaTabungan(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 15, HargaNasabahSnapshot: 1000},
	}
	// dua nasabah: NSB-A punya 10kg (lebih tua), NSB-B punya 10kg
	// FIFO: ambil 10kg dari NSB-A dulu, sisanya 5kg dari NSB-B
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-A", "NSB-A", "SMP-001", 10),
		baseTabunganNasabah("TBG-B", "NSB-B", "SMP-001", 10),
	}
	nasabahA := &models.Nasabah{NasabahID: "NSB-A", User: models.User{UserID: "USR-A"}}
	nasabahB := &models.Nasabah{NasabahID: "NSB-B", User: models.User{UserID: "USR-B"}}

	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 30000

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabungan, nil)
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-A").Return(nasabahA, nil)
	repo.On("GetNasabahWithUser", "NSB-B").Return(nasabahB, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	// total: 15 * 1000 = 15000
	assert.Equal(t, float64(15000), result.TotalDistribusiNasabah)
	// dua nasabah dapat notif
	assert.Len(t, result.NasabahNotifs, 2)
	// SaveTabungan dipanggil 2x (dua tabungan dikonsumsi)
	repo.AssertNumberOfCalls(t, "SaveTabungan", 2)
	repo.AssertExpectations(t)
}
