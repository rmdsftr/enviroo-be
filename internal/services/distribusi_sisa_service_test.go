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

func splitMandiri(bankID string, qty float64) repositories.TransportSplitRow {
	return repositories.TransportSplitRow{BankID: bankID, QtyMandiri: qty, QtyNonMandiri: 0}
}

func splitBSI(bankID string, qty float64) repositories.TransportSplitRow {
	return repositories.TransportSplitRow{BankID: bankID, QtyMandiri: 0, QtyNonMandiri: qty}
}

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

func TestSubmitDistribusiSisa_SatuBSU_SemuaMandiri(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	transport := []repositories.TransportSplitRow{splitMandiri("BSU-001", 10)}
	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Satu"}}
	setupSubmitMocks(repo, kontribusi, transport, bsuList)
	expectTx(dbMock, 2)

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
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	transport := []repositories.TransportSplitRow{splitBSI("BSU-001", 10)}
	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Satu"}}
	setupSubmitMocks(repo, kontribusi, transport, bsuList)
	expectTx(dbMock, 2)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	assert.Equal(t, float64(40000), result.NominalBSI)
	assert.Equal(t, float64(60000), result.PenerimaBSU[0].Nominal)
	repo.AssertExpectations(t)
}

func TestSubmitDistribusiSisa_SatuBSU_TransportCampuran(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	transport := []repositories.TransportSplitRow{splitMixed("BSU-001", 6, 4)}
	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Satu"}}
	setupSubmitMocks(repo, kontribusi, transport, bsuList)
	expectTx(dbMock, 2)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	assert.Equal(t, float64(34000), result.NominalBSI)
	assert.Equal(t, float64(66000), result.PenerimaBSU[0].Nominal)
	repo.AssertExpectations(t)
}

func TestSubmitDistribusiSisa_DuaBSU_KontribusiProporsional(t *testing.T) {
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
	expectTx(dbMock, 3)

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
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{
		{BankID: "BSU-001", NilaiDipakai: 50000},
		{BankID: "BSU-002", NilaiDipakai: 30000},
		{BankID: "BSU-003", NilaiDipakai: 20000},
	}
	transport := []repositories.TransportSplitRow{
		splitMixed("BSU-001", 3, 1),
		splitBSI("BSU-002", 4),
		splitMandiri("BSU-003", 2),
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
	expectTx(dbMock, 4)

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
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDistribusiSisaRepo)
	svc := services.NewDistribusiSisaServiceWithRepo(gdb, repo)

	kontribusi := []repositories.KontribusiBSURow{{BankID: "BSU-001", NilaiDipakai: 80000}}
	transport := []repositories.TransportSplitRow{}
	bsuList := []models.BankSampah{{BankID: "BSU-001"}}
	setupSubmitMocks(repo, kontribusi, transport, bsuList)
	expectTx(dbMock, 2)

	result, err := svc.Submit("BGH-001", adminReq())

	assert.NoError(t, err)
	assert.Equal(t, float64(40000), result.NominalBSI)
	assert.Equal(t, float64(60000), result.PenerimaBSU[0].Nominal)
	repo.AssertExpectations(t)
}

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
