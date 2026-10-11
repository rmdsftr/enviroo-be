package services_test

import (
	"errors"
	"testing"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func newPenjualanSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockPenjualanRepo, *mocks.MockMitraRepo, *services.PenjualanService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockPenjualanRepo)
	mitraRepo := new(mocks.MockMitraRepo)
	svc := services.NewPenjualanService(gdb, repo, mitraRepo)
	return dbMock, repo, mitraRepo, svc
}

func penjualanBankBSM() *models.BankSampah {
	return &models.BankSampah{BankID: "BSM-001", NamaBank: "BSM Utama", JenisBank: models.BSM}
}

func rewardUang() *models.Reward {
	return &models.Reward{RewardID: 1, NamaReward: models.RewardEnumUang, Satuan: models.SatuanRewardEnumRp}
}

func katalogBotol() *models.KatalogSampah {
	return &models.KatalogSampah{
		SampahID: "SMP-001",
		RewardID: 1,
		Sarok:    models.Sampah{NamaSampah: "Botol Plastik", Satuan: models.SatuanKG},
	}
}

func stokCukup() *models.StokSampah {
	return &models.StokSampah{BankID: "BSM-001", SampahID: "SMP-001", Stok: 100}
}

func stokKosong() *models.StokSampah {
	return &models.StokSampah{BankID: "BSM-001", SampahID: "SMP-001", Stok: 0.5}
}

func TestPreviewPenjualan_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-XXX").Return(nil, errors.New("not found"))

	result, err := svc.PreviewPenjualan("BSM-XXX", 1, []services.ItemSampahDijual{{SampahID: "SMP-001", Qty: 1, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bank")
}

func TestPreviewPenjualan_RewardTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 99).Return(float64(30), nil)
	repo.On("GetRewardByID", mock.Anything, 99).Return(nil, errors.New("not found"))

	result, err := svc.PreviewPenjualan("BSM-001", 99, []services.ItemSampahDijual{{SampahID: "SMP-001", Qty: 1, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reward")
}

func TestPreviewPenjualan_BagiHasilBelumDiatur(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).
		Return(float64(0), gorm.ErrRecordNotFound)

	result, err := svc.PreviewPenjualan("BSM-001", 1, []services.ItemSampahDijual{{SampahID: "SMP-001", Qty: 1, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "belum mengatur bagi hasil")
}

func TestSubmitPenjualan_BagiHasilBelumDiatur(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).
		Return(float64(0), gorm.ErrRecordNotFound)

	result, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "MTR-001", "", "",
		[]services.ItemSampahDijual{{SampahID: "SMP-001", Qty: 1, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "belum mengatur bagi hasil")
}

func TestPreviewPenjualan_SampahTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-XXX").Return(nil, errors.New("not found"))

	result, err := svc.PreviewPenjualan("BSM-001", 1, []services.ItemSampahDijual{{SampahID: "SMP-XXX", Qty: 1, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sampah")
}

func TestPreviewPenjualan_StokTidakAda(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-001").Return(katalogBotol(), nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-001").Return(nil, errors.New("not found"))

	result, err := svc.PreviewPenjualan("BSM-001", 1, []services.ItemSampahDijual{{SampahID: "SMP-001", Qty: 1, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestPreviewPenjualan_StokTidakCukup(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-001").Return(katalogBotol(), nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-001").Return(stokKosong(), nil)

	result, err := svc.PreviewPenjualan("BSM-001", 1, []services.ItemSampahDijual{{SampahID: "SMP-001", Qty: 5, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak mencukupi")
}

func TestPreviewPenjualan_Sukses_SingleItem(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-001").Return(katalogBotol(), nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-001").Return(stokCukup(), nil)

	result, err := svc.PreviewPenjualan("BSM-001", 1, []services.ItemSampahDijual{
		{SampahID: "SMP-001", Qty: 2, HargaJual: 5000},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(10000), result.TotalPenjualan)
	assert.Equal(t, models.SatuanRewardEnumRp, result.Satuan)
	assert.Equal(t, float64(30), result.PersenNasabah)
	assert.Len(t, result.DetailItems, 1)
	assert.Equal(t, float64(10000), result.DetailItems[0].Subtotal)
	assert.Equal(t, float64(1500), result.DetailItems[0].HargaNasabahSnapshot)
}

func TestPreviewPenjualan_Sukses_MultipleItem(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	katalog2 := &models.KatalogSampah{SampahID: "SMP-002", RewardID: 1, Sarok: models.Sampah{NamaSampah: "Kertas"}}
	stok2 := &models.StokSampah{BankID: "BSM-001", SampahID: "SMP-002", Stok: 50}

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-001").Return(katalogBotol(), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-002").Return(katalog2, nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-001").Return(stokCukup(), nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-002").Return(stok2, nil)

	result, err := svc.PreviewPenjualan("BSM-001", 1, []services.ItemSampahDijual{
		{SampahID: "SMP-001", Qty: 2, HargaJual: 5000},
		{SampahID: "SMP-002", Qty: 1, HargaJual: 3000},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(13000), result.TotalPenjualan)
	assert.Len(t, result.DetailItems, 2)
}

func TestSubmitPenjualan_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-XXX").Return(nil, errors.New("not found"))

	result, err := svc.SubmitPenjualan("BSM-XXX", "ADM-001", 1, "", "Pembeli A", "", []services.ItemSampahDijual{{SampahID: "SMP-001", Qty: 1, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bank")
}

func TestSubmitPenjualan_RewardTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetRewardByID", mock.Anything, 99).Return(nil, errors.New("not found"))

	result, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 99, "", "Pembeli A", "", []services.ItemSampahDijual{{SampahID: "SMP-001", Qty: 1, HargaJual: 5000}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Reward")
}

func TestSubmitPenjualan_StokTidakCukup_Rollback(t *testing.T) {
	dbMock, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-001").Return(katalogBotol(), nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-001").Return(stokKosong(), nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"penjualan_id"}))
	dbMock.ExpectRollback()

	result, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "Pembeli A", "", []services.ItemSampahDijual{
		{SampahID: "SMP-001", Qty: 5, HargaJual: 5000},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak mencukupi")
}

func TestSubmitPenjualan_Sukses_SchemaSamHarga(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)

	schema := &models.SchemaHargaSampah{SchemaID: "1SMP-001", SampahID: "SMP-001", LevelUser: models.LevelEksternal, Harga: 5000}

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-001").Return(katalogBotol(), nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-001").Return(stokCukup(), nil)
	repo.On("GetSchemaHarga", mock.Anything, "SMP-001", models.LevelEksternal).Return(schema, nil)
	repo.On("DecrStokSampah", mock.Anything, "BSM-001", "SMP-001", float64(98)).Return(nil)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", mock.Anything).Return(nil, errors.New("not found"))
	mitraRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenjualan", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailPenjualan", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"penjualan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"mitra_id"}))
	dbMock.ExpectCommit()

	result, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "Pembeli A", "", []services.ItemSampahDijual{
		{SampahID: "SMP-001", Qty: 2, HargaJual: 5000},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.PenjualanID)
	assert.Equal(t, float64(10000), result.TotalPenjualan)
	assert.Equal(t, models.SatuanRewardEnumRp, result.Satuan)
	assert.Equal(t, models.BagiHasilPending, result.StatusBagiHasil)
	repo.AssertExpectations(t)
}

func TestSubmitPenjualan_Sukses_SchemaBelumAda(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-001").Return(katalogBotol(), nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-001").Return(stokCukup(), nil)
	repo.On("GetSchemaHarga", mock.Anything, "SMP-001", models.LevelEksternal).Return(nil, errors.New("not found"))
	repo.On("CreateSchemaHarga", mock.Anything, mock.Anything).Return(nil)
	repo.On("DecrStokSampah", mock.Anything, "BSM-001", "SMP-001", float64(98)).Return(nil)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", mock.Anything).Return(nil, errors.New("not found"))
	mitraRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenjualan", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailPenjualan", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"penjualan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"mitra_id"}))
	dbMock.ExpectCommit()

	result, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "Pembeli A", "", []services.ItemSampahDijual{
		{SampahID: "SMP-001", Qty: 2, HargaJual: 5000},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.PenjualanID)
	repo.AssertExpectations(t)
}

func TestSubmitPenjualan_Sukses_SchemaBerubahHarga(t *testing.T) {
	dbMock, repo, mitraRepo, svc := newPenjualanSvc(t)

	schema := &models.SchemaHargaSampah{SchemaID: "1SMP-001", SampahID: "SMP-001", LevelUser: models.LevelEksternal, Harga: 4000}

	repo.On("GetBankByID", "BSM-001").Return(penjualanBankBSM(), nil)
	repo.On("GetRewardByID", mock.Anything, 1).Return(rewardUang(), nil)
	repo.On("GetPersenNasabah", mock.Anything, "BSM-001", 1).Return(float64(30), nil)
	repo.On("GetKatalogSampah", mock.Anything, "SMP-001").Return(katalogBotol(), nil)
	repo.On("GetStokSampah", mock.Anything, "BSM-001", "SMP-001").Return(stokCukup(), nil)
	repo.On("GetSchemaHarga", mock.Anything, "SMP-001", models.LevelEksternal).Return(schema, nil)
	repo.On("CreateHistoryHarga", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSchemaHarga", mock.Anything, mock.Anything).Return(nil)
	repo.On("DecrStokSampah", mock.Anything, "BSM-001", "SMP-001", float64(98)).Return(nil)
	mitraRepo.On("FindByNormalizedName", mock.Anything, "BSM-001", mock.Anything).Return(nil, errors.New("not found"))
	mitraRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenjualan", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailPenjualan", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"penjualan_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"history_sampah_id"}))
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"mitra_id"}))
	dbMock.ExpectCommit()

	result, err := svc.SubmitPenjualan("BSM-001", "ADM-001", 1, "", "Pembeli A", "", []services.ItemSampahDijual{
		{SampahID: "SMP-001", Qty: 2, HargaJual: 5000},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(10000), result.TotalPenjualan)
	repo.AssertExpectations(t)
}

func TestGetDetailPenjualan_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetPenjualan", "PJL-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetDetail("PJL-XXX")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "penjualan")
}

func TestGetDetailPenjualan_DetailItemsGagal(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	p := &models.Penjualan{
		PenjualanID: "PJL-001",
		BankID:      "BSM-001",
		BankSampah:  *penjualanBankBSM(),
		Reward:      *rewardUang(),
	}
	repo.On("GetPenjualan", "PJL-001").Return(p, nil)
	repo.On("GetDetailItems", "PJL-001").Return(nil, errors.New("db error"))

	result, err := svc.GetDetail("PJL-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "detail item")
}

func TestGetDetailPenjualan_Sukses(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	p := &models.Penjualan{
		PenjualanID:      "PJL-001",
		BankID:           "BSM-001",
		MitraID:          "MTR-001",
		TotalItem:        1,
		TotalPenjualan:   10000,
		SatuanReward:     models.SatuanRewardEnumRp,
		SoldBy:           "ADM-001",
		StatusBagiHasil:  models.BagiHasilPending,
		BankSampah:       *penjualanBankBSM(),
		Reward:           *rewardUang(),
		MitraPengepul:    models.MitraPengepul{MitraID: "MTR-001", NamaMitra: "Pembeli A"},
	}
	details := []models.DetailPenjualan{
		{
			SampahID:          "SMP-001",
			Qty:               2,
			HargaJual:         5000,
			SubtotalPenjualan: 10000,
			Sampah:            models.KatalogSampah{Sarok: models.Sampah{NamaSampah: "Botol Plastik"}},
		},
	}

	repo.On("GetPenjualan", "PJL-001").Return(p, nil)
	repo.On("GetDetailItems", "PJL-001").Return(details, nil)
	repo.On("GetAdminNama", "ADM-001").Return("Admin Satu")

	result, err := svc.GetDetail("PJL-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "PJL-001", result.PenjualanID)
	assert.Equal(t, "Pembeli A", result.NamaMitra)
	assert.Equal(t, float64(10000), result.TotalPenjualan)
	assert.Equal(t, "Admin Satu", result.AdminName)
	assert.Len(t, result.ItemsSampah, 1)
	assert.Equal(t, "Botol Plastik", result.ItemsSampah[0].NamaSampah)
	assert.Equal(t, float64(10000), result.ItemsSampah[0].SubtotalPenjualan)
}

func TestGetRiwayat_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenjualanSvc(t)

	repo.On("GetBankByID", "BSM-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetRiwayat("BSM-XXX", "", "")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bank")
}
