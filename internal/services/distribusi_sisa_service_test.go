package services_test

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ── Helpers ──────────────────────────────────────────────────────────────────

func baseBagiHasil(sisaBagiHasil float64) *models.BagiHasil {
	penjualanID := "PJL-001"
	bankID := "BSI-001"
	return &models.BagiHasil{
		BagiHasilID:     "BGH-001",
		PenjualanID:     &penjualanID,
		BankID:          &bankID,
		SisaBagiHasil:   sisaBagiHasil,
		SatuanBagiHasil: models.SatuanRewardEnumRp,
		Bank: &models.BankSampah{
			BankID:    "BSI-001",
			NamaBank:  "BSI Utama",
			JenisBank: models.BSI,
		},
	}
}

func baseSetting() *models.SettingSisaBagiHasil {
	return &models.SettingSisaBagiHasil{
		BankID:         "BSI-001",
		PorsiBSU:       60,
		PorsiBSI:       30,
		PorsiTransport: 10,
	}
}

func baseRewardUang() *models.Reward {
	return &models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang}
}

// splitMandiri builds a TransportSplitRow where all qty is mandiri (BSU delivers itself).
func splitMandiri(bankID string, qty float64) repositories.TransportSplitRow {
	return repositories.TransportSplitRow{BankID: bankID, QtyMandiri: qty, QtyNonMandiri: 0}
}

// splitBSI builds a TransportSplitRow where all qty is non-mandiri (BSI picks up).
func splitBSI(bankID string, qty float64) repositories.TransportSplitRow {
	return repositories.TransportSplitRow{BankID: bankID, QtyMandiri: 0, QtyNonMandiri: qty}
}

// splitMixed builds a TransportSplitRow with a mix of mandiri and non-mandiri qty.
func splitMixed(bankID string, qtyMandiri, qtyNonMandiri float64) repositories.TransportSplitRow {
	return repositories.TransportSplitRow{BankID: bankID, QtyMandiri: qtyMandiri, QtyNonMandiri: qtyNonMandiri}
}

func setupSubmitMocks(
	repo *mocks.MockDistribusiSisaRepo,
	kontribusi []repositories.KontribusiBSURow,
	transportSplit []repositories.TransportSplitRow,
	bsuList []models.BankSampah,
) {
	repo.On("GetBagiHasilWithBank", "BGH-001").Return(baseBagiHasil(100000), nil)
	repo.On("FindDistribusiByBagiHasil", "BGH-001").Return(nil, false)
	repo.On("GetSetting", "BSI-001").Return(baseSetting(), nil)
	repo.On("GetRewardByNama", models.RewardEnumUang).Return(baseRewardUang(), nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetKontribusiPerBSU", mock.Anything, "BGH-001").Return(kontribusi, nil)
	repo.On("GetTransportSplitPerBSU", mock.Anything, "BGH-001").Return(transportSplit, nil)
	repo.On("CreateDistribusiSisa", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaSisa", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("SetPenerimaSisaID", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
}

func adminReq() services.SubmitDistribusiReq {
	return services.SubmitDistribusiReq{AdminID: "ADM-001"}
}

// ── Tests: validasi awal ──────────────────────────────────────────────────────

func TestSubmitDistribusiSisa_BukanBSI(t *testing.T) {
	gdb, _ := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	bh := baseBagiHasil(100000)
	bh.Bank.JenisBank = models.BSM
	repo.On("GetBagiHasilWithBank", "BGH-001").Return(bh, nil)
	repo.On("FindDistribusiByBagiHasil", "BGH-001").Return(nil, false)

	_, err := svc.Submit("BGH-001", adminReq())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hanya berlaku untuk bagi hasil BSI")
}

func TestSubmitDistribusiSisa_SudahDidistribusikan(t *testing.T) {
	gdb, _ := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	existing := &models.DistribusiSisa{DistribusiID: "DS-EXISTING"}
	repo.On("GetBagiHasilWithBank", "BGH-001").Return(baseBagiHasil(100000), nil)
	repo.On("FindDistribusiByBagiHasil", "BGH-001").Return(existing, true)

	_, err := svc.Submit("BGH-001", adminReq())

	se, ok := err.(*services.ServiceError)
	assert.True(t, ok)
	assert.Equal(t, 400, se.Code)
	assert.Contains(t, err.Error(), "sudah pernah didistribusikan")
}

func TestSubmitDistribusiSisa_SisaNol(t *testing.T) {
	gdb, _ := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	repo.On("GetBagiHasilWithBank", "BGH-001").Return(baseBagiHasil(0), nil)
	repo.On("FindDistribusiByBagiHasil", "BGH-001").Return(nil, false)

	_, err := svc.Submit("BGH-001", adminReq())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Tidak ada sisa")
}

// ── Tests: happy path — transport split dari is_mandiri ───────────────────────

func TestSubmitDistribusiSisa_SatuBSU_SemuaMandiri(t *testing.T) {
	// Semua pengangkutan is_mandiri=true → BSU dapat seluruh transport
	// totalSisa=100000, setting 60/30/10, BSU-001 100% kontribusi
	// BSU: 60000 + 10000 transport = 70000
	// BSI: 30000 pokok + 0 transport = 30000
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	transport := []repositories.TransportSplitRow{splitMandiri("BSU-001", 10)}
	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Satu"}}
	setupSubmitMocks(repo, kontribusi, transport, bsuList)
	expectTx(dbMock)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	assert.Equal(t, float64(100000), result.TotalSisa)
	assert.Equal(t, float64(30000), result.NominalBSI)
	assert.Len(t, result.PenerimaBSU, 1)
	assert.Equal(t, float64(70000), result.PenerimaBSU[0].Nominal)
	repo.AssertNumberOfCalls(t, "CreatePenerimaSisa", 2)
	repo.AssertNumberOfCalls(t, "UpdateSaldoRekening", 2)
	repo.AssertNumberOfCalls(t, "SetPenerimaSisaID", 1)
	repo.AssertExpectations(t)
}

func TestSubmitDistribusiSisa_SatuBSU_SemuaNonMandiri(t *testing.T) {
	// Semua pengangkutan is_mandiri=false → BSI dapat seluruh transport
	// BSU: 60000 pokok saja
	// BSI: 30000 + 10000 transport = 40000
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	transport := []repositories.TransportSplitRow{splitBSI("BSU-001", 10)}
	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Satu"}}
	setupSubmitMocks(repo, kontribusi, transport, bsuList)
	expectTx(dbMock)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	assert.Equal(t, float64(40000), result.NominalBSI)
	assert.Equal(t, float64(60000), result.PenerimaBSU[0].Nominal)
	repo.AssertExpectations(t)
}

func TestSubmitDistribusiSisa_SatuBSU_TransportCampuran(t *testing.T) {
	// BSU-001: pgk-1 mandiri 6kg, pgk-2 non-mandiri 4kg → 60% mandiri
	// totalSisa=100000, setting 60/30/10, BSU-001 100% kontribusi
	// bagianTransport = 100000 * 0.10 = 10000
	// transportBSU = 10000 * 0.6 = 6000
	// transportBSI = 10000 * 0.4 = 4000
	// BSU: 60000 + 6000 = 66000
	// BSI: 30000 + 4000 = 34000
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	transport := []repositories.TransportSplitRow{splitMixed("BSU-001", 6, 4)}
	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Satu"}}
	setupSubmitMocks(repo, kontribusi, transport, bsuList)
	expectTx(dbMock)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	assert.Equal(t, float64(34000), result.NominalBSI)
	assert.Equal(t, float64(66000), result.PenerimaBSU[0].Nominal)
	repo.AssertExpectations(t)
}

func TestSubmitDistribusiSisa_DuaBSU_KontribusiProporsional(t *testing.T) {
	// BSU-001 60% kontribusi, mandiri semua
	// BSU-002 40% kontribusi, non-mandiri semua
	//
	// BSU-001 (60%):
	//   pokok = 100000*0.6*0.6 = 36000
	//   transport = 100000*0.1*0.6 * 1.0 = 6000  → total 42000
	// BSU-002 (40%):
	//   pokok = 100000*0.6*0.4 = 24000
	//   transport = 100000*0.1*0.4 * 0 = 0       → total 24000
	// BSI:
	//   pokok = (18000+12000) = 30000
	//   transport = 0 + 4000 = 4000               → total 34000
	// Cek: 42000+24000+34000 = 100000 ✓
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{
		{BankID: "BSU-001", NilaiDipakai: 60000},
		{BankID: "BSU-002", NilaiDipakai: 40000},
	}
	transport := []repositories.TransportSplitRow{
		splitMandiri("BSU-001", 6),
		splitBSI("BSU-002", 4),
	}
	bsuList := []models.BankSampah{
		{BankID: "BSU-001", NamaBank: "BSU Satu"},
		{BankID: "BSU-002", NamaBank: "BSU Dua"},
	}

	repo.On("GetBagiHasilWithBank", "BGH-001").Return(baseBagiHasil(100000), nil)
	repo.On("FindDistribusiByBagiHasil", "BGH-001").Return(nil, false)
	repo.On("GetSetting", "BSI-001").Return(baseSetting(), nil)
	repo.On("GetRewardByNama", models.RewardEnumUang).Return(baseRewardUang(), nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetKontribusiPerBSU", mock.Anything, "BGH-001").Return(kontribusi, nil)
	repo.On("GetTransportSplitPerBSU", mock.Anything, "BGH-001").Return(transport, nil)
	repo.On("CreateDistribusiSisa", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaSisa", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("SetPenerimaSisaID", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	assert.Equal(t, float64(34000), result.NominalBSI)
	assert.Len(t, result.PenerimaBSU, 2)

	nominalByID := make(map[string]float64)
	for _, p := range result.PenerimaBSU {
		nominalByID[p.BankID] = p.Nominal
	}
	assert.Equal(t, float64(42000), nominalByID["BSU-001"])
	assert.Equal(t, float64(24000), nominalByID["BSU-002"])

	repo.AssertNumberOfCalls(t, "CreatePenerimaSisa", 3)
	repo.AssertNumberOfCalls(t, "UpdateSaldoRekening", 3)
	repo.AssertNumberOfCalls(t, "SetPenerimaSisaID", 2)
	repo.AssertExpectations(t)
}

func TestSubmitDistribusiSisa_TigaBSU_CampuranMandiri(t *testing.T) {
	// BSU-001 50% kontribusi, 75% mandiri
	// BSU-002 30% kontribusi, 0% mandiri
	// BSU-003 20% kontribusi, 100% mandiri
	// totalSisa=100000, setting 60/30/10
	//
	// BSU-001 (50%): pokok=30000, transport=10000*0.5*0.75=3750  → 33750
	// BSU-002 (30%): pokok=18000, transport=0                     → 18000
	// BSU-003 (20%): pokok=12000, transport=10000*0.2*1.0=2000    → 14000
	// BSI: pokok=(15000+9000+6000)=30000, transport=(1250+3000+0)=4250 → 34250
	// Cek: 33750+18000+14000+34250 = 100000 ✓
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{
		{BankID: "BSU-001", NilaiDipakai: 50000},
		{BankID: "BSU-002", NilaiDipakai: 30000},
		{BankID: "BSU-003", NilaiDipakai: 20000},
	}
	transport := []repositories.TransportSplitRow{
		splitMixed("BSU-001", 3, 1), // 75% mandiri
		splitBSI("BSU-002", 4),      // 0% mandiri
		splitMandiri("BSU-003", 2),  // 100% mandiri
	}
	bsuList := []models.BankSampah{
		{BankID: "BSU-001"}, {BankID: "BSU-002"}, {BankID: "BSU-003"},
	}

	repo.On("GetBagiHasilWithBank", "BGH-001").Return(baseBagiHasil(100000), nil)
	repo.On("FindDistribusiByBagiHasil", "BGH-001").Return(nil, false)
	repo.On("GetSetting", "BSI-001").Return(baseSetting(), nil)
	repo.On("GetRewardByNama", models.RewardEnumUang).Return(baseRewardUang(), nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetKontribusiPerBSU", mock.Anything, "BGH-001").Return(kontribusi, nil)
	repo.On("GetTransportSplitPerBSU", mock.Anything, "BGH-001").Return(transport, nil)
	repo.On("CreateDistribusiSisa", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaSisa", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("SetPenerimaSisaID", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	assert.InDelta(t, 34250, result.NominalBSI, 0.01)
	assert.Len(t, result.PenerimaBSU, 3)

	nominalByID := make(map[string]float64)
	for _, p := range result.PenerimaBSU {
		nominalByID[p.BankID] = p.Nominal
	}
	assert.InDelta(t, 33750, nominalByID["BSU-001"], 0.01)
	assert.InDelta(t, 18000, nominalByID["BSU-002"], 0.01)
	assert.InDelta(t, 14000, nominalByID["BSU-003"], 0.01)

	repo.AssertNumberOfCalls(t, "CreatePenerimaSisa", 4)
	repo.AssertNumberOfCalls(t, "UpdateSaldoRekening", 4)
	repo.AssertNumberOfCalls(t, "SetPenerimaSisaID", 3)
	repo.AssertExpectations(t)
}

func TestSubmitDistribusiSisa_TidakAdaTransportSplit_FallbackNonMandiri(t *testing.T) {
	// BSU-001 tidak ada di transport split (mungkin data pengangkutan belum ada is_mandiri)
	// QtyMandiri=0, QtyNonMandiri=0 → persenMandiri=0 → semua transport ke BSI
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	// BSU-001 tidak ada di transport split
	transport := []repositories.TransportSplitRow{}
	bsuList := []models.BankSampah{{BankID: "BSU-001"}}
	setupSubmitMocks(repo, kontribusi, transport, bsuList)
	expectTx(dbMock)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	// persenMandiri=0 → BSU hanya dapat pokok (60000), BSI dapat pokok+transport (40000)
	assert.Equal(t, float64(40000), result.NominalBSI)
	assert.Equal(t, float64(60000), result.PenerimaBSU[0].Nominal)
	repo.AssertExpectations(t)
}

// ── Tests: error path ─────────────────────────────────────────────────────────

func TestSubmitDistribusiSisa_CreateDistribusiError_Rollback(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	transport := []repositories.TransportSplitRow{splitMandiri("BSU-001", 10)}
	bsuList := []models.BankSampah{{BankID: "BSU-001"}}

	repo.On("GetBagiHasilWithBank", "BGH-001").Return(baseBagiHasil(100000), nil)
	repo.On("FindDistribusiByBagiHasil", "BGH-001").Return(nil, false)
	repo.On("GetSetting", "BSI-001").Return(baseSetting(), nil)
	repo.On("GetRewardByNama", models.RewardEnumUang).Return(baseRewardUang(), nil)
	repo.On("GetBSUList", mock.Anything, "BSI-001").Return(bsuList, nil)
	repo.On("GetKontribusiPerBSU", mock.Anything, "BGH-001").Return(kontribusi, nil)
	repo.On("GetTransportSplitPerBSU", mock.Anything, "BGH-001").Return(transport, nil)
	repo.On("CreateDistribusiSisa", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	_, err := svc.Submit("BGH-001", adminReq())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Gagal menyimpan data distribusi")
	repo.AssertNotCalled(t, "CreatePenerimaSisa")
	repo.AssertNotCalled(t, "UpdateSaldoRekening")
}
