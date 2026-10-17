package services_test

import (
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"
	"errors"
	"fmt"
	"net/http"
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

func expectTx(m sqlmock.Sqlmock, penerimaCount int) {
	m.ExpectBegin()
	for i := 0; i < penerimaCount; i++ {
		m.ExpectQuery("SELECT").
			WillReturnRows(sqlmock.NewRows([]string{"penerima_id"}))
	}
	m.ExpectCommit()
}

func expectTxRollback(m sqlmock.Sqlmock) {
	m.ExpectBegin()
	m.ExpectRollback()
}

func expectTxBagiHasil(m sqlmock.Sqlmock, penjualanID string, penerimaCount int) {
	m.ExpectBegin()
	for i := 0; i < penerimaCount; i++ {
		rows := sqlmock.NewRows([]string{"penerima_id"})
		if i > 0 {
			rows.AddRow(fmt.Sprintf("P%s%03d", penjualanID, i))
		}
		m.ExpectQuery("SELECT").WillReturnRows(rows)
	}
	m.ExpectCommit()
}

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

type skenarioBSM struct {
	repo    *mocks.MockBagiHasilRepo
	pemilik map[string]string
}

func skenarioBaruBSM(penjualan *models.Penjualan, detailPenjualan []models.DetailPenjualan) *skenarioBSM {
	repo := new(mocks.MockBagiHasilRepo)
	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSM-001").Return(&models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	return &skenarioBSM{repo: repo, pemilik: map[string]string{}}
}

func (sk *skenarioBSM) sediakanTabungan(sampahID string, tabungan ...models.TabunganSampah) {
	sk.repo.On("GetTabunganNasabah", mock.Anything, sampahID, []string{"BSM-001"}).Return(tabungan, nil).Once()
}

func (sk *skenarioBSM) expectTabunganTersimpan(tabunganID string, sisaQty float64) {
	sk.repo.On("SaveTabungan", mock.Anything, mock.MatchedBy(func(t *models.TabunganSampah) bool {
		return t.TabunganID == tabunganID && t.SisaQty == sisaQty
	})).Return(nil).Once()
}

func (sk *skenarioBSM) expectHargaNasabah(sampahID string, harga float64) {
	sk.repo.On("UpsertSchemaHargaNasabah", mock.Anything, sampahID, harga, models.SatuanRewardEnumRp, "ADM-001").
		Return(nil).Once()
}

func (sk *skenarioBSM) expectBagiHasil(gross, totalDistribusi, sisa float64) {
	sk.repo.On("CreateBagiHasil", mock.Anything, mock.MatchedBy(func(bh *models.BagiHasil) bool {
		return bh.BankID != nil && *bh.BankID == "BSM-001" &&
			bh.PenjualanID != nil && *bh.PenjualanID == "PJL-001" &&
			bh.GrossBank == gross && bh.TotalDistribusiNasabah == totalDistribusi && bh.SisaBagiHasil == sisa
	})).Return(nil).Once()
}

func (sk *skenarioBSM) expectNasabahMenerima(nasabahID string, jumlahItem int, total float64) {
	sk.repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.MatchedBy(func(p *models.PenerimaBagiHasil) bool {
		return p.NasabahID != nil && *p.NasabahID == nasabahID &&
			p.TotalItem == jumlahItem && p.TotalDiterima == total
	})).Run(func(args mock.Arguments) {
		p := args.Get(1).(*models.PenerimaBagiHasil)
		sk.pemilik[p.PenerimaID] = nasabahID
	}).Return(nil).Once()

	sk.repo.On("UpdateSaldoRekening", mock.Anything, (*string)(nil), ptrBernilai(nasabahID),
		1, mock.Anything, total, models.EntitasNasabah, "ADM-001", mock.Anything).Return(nil).Once()

	sk.repo.On("GetNasabahWithUser", nasabahID).
		Return(&models.Nasabah{NasabahID: nasabahID, User: models.User{UserID: "USR-" + nasabahID}}, nil).Once()
}

func (sk *skenarioBSM) expectDetail(nasabahID, sampahID string, qty, harga, subtotal float64) {
	sk.repo.On("CreateDetailBagiHasil", mock.Anything, mock.MatchedBy(func(d *models.DetailBagiHasil) bool {
		return sk.pemilik[d.PenerimaID] == nasabahID && d.SampahID == sampahID &&
			d.Qty == qty && d.HargaItem == harga && d.SubtotalHarga == subtotal
	})).Return(nil).Once()
}

func (sk *skenarioBSM) expectPencairan(nasabahID, tabunganID string, qty float64) {
	sk.repo.On("CreatePencairanTabungan", mock.Anything, mock.MatchedBy(func(p *models.PencairanTabungan) bool {
		return sk.pemilik[p.PenerimaID] == nasabahID && p.TabunganID == tabunganID && p.QtyDipakai == qty
	})).Return(nil).Once()
}

func (sk *skenarioBSM) expectSisaMasukBank(sisa float64) {
	sk.repo.On("UpdateSaldoRekening", mock.Anything, ptrBernilai("BSM-001"), (*string)(nil),
		1, mock.Anything, sisa, models.EntitasBankSampah, "ADM-001", mock.Anything).Return(nil).Once()
}

func (sk *skenarioBSM) expectPenjualanSelesai() {
	sk.repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil).Once()
}

func ptrBernilai(s string) interface{} {
	return mock.MatchedBy(func(p *string) bool { return p != nil && *p == s })
}

func totalNotifPerNasabah(notifs []services.NasabahNotifItem) map[string]float64 {
	total := make(map[string]float64, len(notifs))
	for _, n := range notifs {
		total[n.NasabahID] = n.Total
	}
	return total
}

func TestSubmitBagiHasil_SudahDilakukan(t *testing.T) {
	gdb, _ := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(basePenjualan(models.BagiHasilBerhasil), nil)

	_, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sudah dilakukan")
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

	expectTx(dbMock, 1)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(500000), result.GrossBank)
	assert.Equal(t, float64(50000), result.TotalDistribusiNasabah)
	assert.Equal(t, float64(450000), result.SisaBagiHasil)
	repo.AssertExpectations(t)
}

func TestSubmitBagiHasil_StokTidakCukup(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 10, HargaNasabahSnapshot: 5000},
	}
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

func TestSubmitBagiHasil_StokTidakCukup_DiSampahKetiga(t *testing.T) {
	gdb, dbMock := newTestDB(t)

	sk := skenarioBaruBSM(basePenjualan(models.BagiHasilPending), []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 5, HargaNasabahSnapshot: 3000},
		{SampahID: "SMP-002", Qty: 4, HargaNasabahSnapshot: 1500},
		{SampahID: "SMP-003", Qty: 8, HargaNasabahSnapshot: 4000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)

	sk.sediakanTabungan("SMP-001",
		baseTabunganNasabah("TBG-21", "NSB-001", "SMP-001", 5),
	)
	sk.sediakanTabungan("SMP-002",
		baseTabunganNasabah("TBG-22", "NSB-002", "SMP-002", 2.5),
		baseTabunganNasabah("TBG-23", "NSB-003", "SMP-002", 1.5),
	)
	sk.sediakanTabungan("SMP-003",
		baseTabunganNasabah("TBG-24", "NSB-001", "SMP-003", 3),
		baseTabunganNasabah("TBG-25", "NSB-002", "SMP-003", 2.5),
	)
	for _, id := range []string{"TBG-21", "TBG-22", "TBG-23", "TBG-24", "TBG-25"} {
		sk.expectTabunganTersimpan(id, 0)
	}
	sk.expectHargaNasabah("SMP-001", 3000)
	sk.expectHargaNasabah("SMP-002", 1500)
	expectTxRollback(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.Nil(t, result)
	var se *services.ServiceError
	if assert.ErrorAs(t, err, &se) {
		assert.Equal(t, 422, se.Code)
		assert.Equal(t, "Stok tabungan nasabah tidak mencukupi", se.Message)
		assert.Equal(t, map[string]interface{}{"sampah_id": "SMP-003", "kurang_qty": 2.5}, se.Data)
	}
	sk.repo.AssertNotCalled(t, "UpsertSchemaHargaNasabah", mock.Anything, "SMP-003", mock.Anything, mock.Anything, mock.Anything)
	sk.repo.AssertNotCalled(t, "CreateBagiHasil", mock.Anything, mock.Anything)
	sk.repo.AssertNotCalled(t, "CreatePenerimaBagiHasil", mock.Anything, mock.Anything)
	sk.repo.AssertNotCalled(t, "UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	sk.repo.AssertNotCalled(t, "UpdatePenjualanStatus", mock.Anything, mock.Anything, mock.Anything)
	sk.repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestSubmitBagiHasil_BSM_SisaMasukSaldoBank(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
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

	expectTx(dbMock, 1)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(30000), result.TotalDistribusiNasabah)
	assert.Equal(t, float64(70000), result.SisaBagiHasil)
	repo.AssertNumberOfCalls(t, "UpdateSaldoRekening", 2)
}

func TestSubmitBagiHasil_BSM_MultipleSampah(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
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

	expectTx(dbMock, 2)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(19000), result.TotalDistribusiNasabah)
	assert.Equal(t, float64(31000), result.SisaBagiHasil)
	repo.AssertNumberOfCalls(t, "GetTabunganNasabah", 2)
}

func TestSubmitBagiHasil_BSM_EmpatSampahLimaNasabah(t *testing.T) {
	gdb, dbMock := newTestDB(t)

	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 169000
	sk := skenarioBaruBSM(penjualan, []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 16, HargaNasabahSnapshot: 3000},
		{SampahID: "SMP-002", Qty: 20, HargaNasabahSnapshot: 1500},
		{SampahID: "SMP-003", Qty: 7.5, HargaNasabahSnapshot: 4000},
		{SampahID: "SMP-004", Qty: 11, HargaNasabahSnapshot: 2000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)

	sk.sediakanTabungan("SMP-001",
		baseTabunganNasabah("TBG-01", "NSB-001", "SMP-001", 5),
		baseTabunganNasabah("TBG-02", "NSB-002", "SMP-001", 3.5),
		baseTabunganNasabah("TBG-03", "NSB-001", "SMP-001", 2.5),
		baseTabunganNasabah("TBG-04", "NSB-004", "SMP-001", 5),
	)
	sk.sediakanTabungan("SMP-002",
		baseTabunganNasabah("TBG-05", "NSB-003", "SMP-002", 8),
		baseTabunganNasabah("TBG-06", "NSB-005", "SMP-002", 12),
	)
	sk.sediakanTabungan("SMP-003",
		baseTabunganNasabah("TBG-07", "NSB-002", "SMP-003", 4),
		baseTabunganNasabah("TBG-08", "NSB-003", "SMP-003", 3.5),
	)
	sk.sediakanTabungan("SMP-004",
		baseTabunganNasabah("TBG-09", "NSB-001", "SMP-004", 6),
		baseTabunganNasabah("TBG-10", "NSB-005", "SMP-004", 5),
	)

	for _, id := range []string{"TBG-01", "TBG-02", "TBG-03", "TBG-04", "TBG-05",
		"TBG-06", "TBG-07", "TBG-08", "TBG-09", "TBG-10"} {
		sk.expectTabunganTersimpan(id, 0)
	}
	sk.expectHargaNasabah("SMP-001", 3000)
	sk.expectHargaNasabah("SMP-002", 1500)
	sk.expectHargaNasabah("SMP-003", 4000)
	sk.expectHargaNasabah("SMP-004", 2000)
	sk.expectBagiHasil(169000, 130000, 39000)

	sk.expectNasabahMenerima("NSB-001", 2, 34500)
	sk.expectDetail("NSB-001", "SMP-001", 7.5, 3000, 22500)
	sk.expectDetail("NSB-001", "SMP-004", 6, 2000, 12000)
	sk.expectPencairan("NSB-001", "TBG-01", 5)
	sk.expectPencairan("NSB-001", "TBG-03", 2.5)
	sk.expectPencairan("NSB-001", "TBG-09", 6)

	sk.expectNasabahMenerima("NSB-002", 2, 26500)
	sk.expectDetail("NSB-002", "SMP-001", 3.5, 3000, 10500)
	sk.expectDetail("NSB-002", "SMP-003", 4, 4000, 16000)
	sk.expectPencairan("NSB-002", "TBG-02", 3.5)
	sk.expectPencairan("NSB-002", "TBG-07", 4)

	sk.expectNasabahMenerima("NSB-003", 2, 26000)
	sk.expectDetail("NSB-003", "SMP-002", 8, 1500, 12000)
	sk.expectDetail("NSB-003", "SMP-003", 3.5, 4000, 14000)
	sk.expectPencairan("NSB-003", "TBG-05", 8)
	sk.expectPencairan("NSB-003", "TBG-08", 3.5)

	sk.expectNasabahMenerima("NSB-004", 1, 15000)
	sk.expectDetail("NSB-004", "SMP-001", 5, 3000, 15000)
	sk.expectPencairan("NSB-004", "TBG-04", 5)

	sk.expectNasabahMenerima("NSB-005", 2, 28000)
	sk.expectDetail("NSB-005", "SMP-002", 12, 1500, 18000)
	sk.expectDetail("NSB-005", "SMP-004", 5, 2000, 10000)
	sk.expectPencairan("NSB-005", "TBG-06", 12)
	sk.expectPencairan("NSB-005", "TBG-10", 5)

	sk.expectSisaMasukBank(39000)
	sk.expectPenjualanSelesai()
	expectTxBagiHasil(dbMock, "PJL-001", 5)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(169000), result.GrossBank)
	assert.Equal(t, float64(130000), result.TotalDistribusiNasabah)
	assert.Equal(t, float64(39000), result.SisaBagiHasil)
	assert.Equal(t, map[string]float64{
		"NSB-001": 34500, "NSB-002": 26500, "NSB-003": 26000, "NSB-004": 15000, "NSB-005": 28000,
	}, totalNotifPerNasabah(result.NasabahNotifs))
	sk.repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestSubmitBagiHasil_PartialFIFO(t *testing.T) {
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
		return t.SisaQty == 12
	})).Return(nil)
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePenerimaBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailBagiHasil", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePencairanTabungan", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateSaldoRekening", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	repo.On("GetNasabahWithUser", "NSB-001").Return(&models.Nasabah{NasabahID: "NSB-001", User: models.User{UserID: "USR-1"}}, nil)
	repo.On("UpdatePenjualanStatus", mock.Anything, mock.Anything, models.BagiHasilBerhasil).Return(nil)

	expectTx(dbMock, 1)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(8000), result.TotalDistribusiNasabah)
	repo.AssertExpectations(t)
}

func TestSubmitBagiHasil_PartialFIFO_TigaSampah(t *testing.T) {
	gdb, dbMock := newTestDB(t)

	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 68500
	sk := skenarioBaruBSM(penjualan, []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 9, HargaNasabahSnapshot: 3000},
		{SampahID: "SMP-002", Qty: 10, HargaNasabahSnapshot: 1500},
		{SampahID: "SMP-003", Qty: 2.5, HargaNasabahSnapshot: 4000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)

	sisaPenjualanLalu := baseTabunganNasabah("TBG-11", "NSB-001", "SMP-001", 4)
	sisaPenjualanLalu.Qty = 10
	sk.sediakanTabungan("SMP-001",
		sisaPenjualanLalu,
		baseTabunganNasabah("TBG-12", "NSB-002", "SMP-001", 3),
		baseTabunganNasabah("TBG-13", "NSB-001", "SMP-001", 6),
		baseTabunganNasabah("TBG-14", "NSB-003", "SMP-001", 5),
	)
	sk.sediakanTabungan("SMP-002",
		baseTabunganNasabah("TBG-15", "NSB-002", "SMP-002", 7.5),
		baseTabunganNasabah("TBG-16", "NSB-004", "SMP-002", 6),
		baseTabunganNasabah("TBG-17", "NSB-005", "SMP-002", 4),
	)
	sk.sediakanTabungan("SMP-003",
		baseTabunganNasabah("TBG-18", "NSB-003", "SMP-003", 4),
		baseTabunganNasabah("TBG-19", "NSB-005", "SMP-003", 3),
	)

	sk.expectTabunganTersimpan("TBG-11", 0)
	sk.expectTabunganTersimpan("TBG-12", 0)
	sk.expectTabunganTersimpan("TBG-13", 4)
	sk.expectTabunganTersimpan("TBG-15", 0)
	sk.expectTabunganTersimpan("TBG-16", 3.5)
	sk.expectTabunganTersimpan("TBG-18", 1.5)
	sk.expectHargaNasabah("SMP-001", 3000)
	sk.expectHargaNasabah("SMP-002", 1500)
	sk.expectHargaNasabah("SMP-003", 4000)
	sk.expectBagiHasil(68500, 52000, 16500)

	sk.expectNasabahMenerima("NSB-001", 1, 18000)
	sk.expectDetail("NSB-001", "SMP-001", 6, 3000, 18000)
	sk.expectPencairan("NSB-001", "TBG-11", 4)
	sk.expectPencairan("NSB-001", "TBG-13", 2)

	sk.expectNasabahMenerima("NSB-002", 2, 20250)
	sk.expectDetail("NSB-002", "SMP-001", 3, 3000, 9000)
	sk.expectDetail("NSB-002", "SMP-002", 7.5, 1500, 11250)
	sk.expectPencairan("NSB-002", "TBG-12", 3)
	sk.expectPencairan("NSB-002", "TBG-15", 7.5)

	sk.expectNasabahMenerima("NSB-003", 1, 10000)
	sk.expectDetail("NSB-003", "SMP-003", 2.5, 4000, 10000)
	sk.expectPencairan("NSB-003", "TBG-18", 2.5)

	sk.expectNasabahMenerima("NSB-004", 1, 3750)
	sk.expectDetail("NSB-004", "SMP-002", 2.5, 1500, 3750)
	sk.expectPencairan("NSB-004", "TBG-16", 2.5)

	sk.expectSisaMasukBank(16500)
	sk.expectPenjualanSelesai()
	expectTxBagiHasil(dbMock, "PJL-001", 4)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(68500), result.GrossBank)
	assert.Equal(t, float64(52000), result.TotalDistribusiNasabah)
	assert.Equal(t, float64(16500), result.SisaBagiHasil)
	assert.Equal(t, map[string]float64{
		"NSB-001": 18000, "NSB-002": 20250, "NSB-003": 10000, "NSB-004": 3750,
	}, totalNotifPerNasabah(result.NasabahNotifs))
	sk.repo.AssertNotCalled(t, "GetNasabahWithUser", "NSB-005")
	sk.repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestSubmitBagiHasil_TolakItemHargaNol(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 5, HargaNasabahSnapshot: 1000},
		{SampahID: "SMP-002", Qty: 3, HargaNasabahSnapshot: 0},
	}
	tabungan := []models.TabunganSampah{
		baseTabunganNasabah("TBG-001", "NSB-001", "SMP-001", 5),
	}
	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 10000

	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(penjualan, nil)
	repo.On("GetBankByID", "BSM-001").Return(bank, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return(detailPenjualan, nil)
	repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).Return(tabungan, nil).Maybe()
	repo.On("SaveTabungan", mock.Anything, mock.Anything).Return(nil).Maybe()
	repo.On("UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "SMP-002")
	repo.AssertNotCalled(t, "UpdatePenjualanStatus", mock.Anything, mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
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
	repo.AssertNotCalled(t, "CreateBagiHasil")
}

func TestSubmitBagiHasil_FIFODuaTabungan(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	svc := services.NewBagiHasilServiceWithRepo(gdb, repo)

	bank := &models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}
	detailPenjualan := []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 15, HargaNasabahSnapshot: 1000},
	}
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

	expectTx(dbMock, 2)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.NoError(t, err)
	assert.Equal(t, float64(15000), result.TotalDistribusiNasabah)
	assert.Len(t, result.NasabahNotifs, 2)
	repo.AssertNumberOfCalls(t, "SaveTabungan", 2)
	repo.AssertExpectations(t)
}

func bhKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func newBHSvc(t *testing.T) (*mocks.MockBagiHasilRepo, *services.BagiHasilService) {
	t.Helper()
	gdb, _ := newTestDB(t)
	repo := new(mocks.MockBagiHasilRepo)
	return repo, services.NewBagiHasilServiceWithRepo(gdb, repo)
}

var bhTanggal = time.Date(2026, 3, 10, 9, 0, 0, 0, time.UTC)

func bhBagiHasil() *models.BagiHasil {
	return &models.BagiHasil{
		BagiHasilID:            "BH-PJL-001",
		BankID:                 ptrStr("BSM-001"),
		PenjualanID:            ptrStr("PJL-001"),
		GrossBank:              100000,
		TotalDistribusiNasabah: 34000,
		SisaBagiHasil:          66000,
		SatuanBagiHasil:        models.SatuanRewardEnumRp,
		CreatedAt:              bhTanggal,
		CreatedBy:              ptrStr("ADM-001"),
		Bank:                   &models.BankSampah{BankID: "BSM-001", NamaBank: "BSM Satu", JenisBank: models.BSM},
		Penjualan:              basePenjualan(models.BagiHasilBerhasil),
	}
}

func bhPenerima(penerimaID, nasabahID, bankID, namaBank, nama string, total float64) models.PenerimaBagiHasil {
	return models.PenerimaBagiHasil{
		PenerimaID:     penerimaID,
		BagiHasilID:    ptrStr("BH-PJL-001"),
		NasabahID:      ptrStr(nasabahID),
		TotalDiterima:  total,
		SatuanDiterima: models.SatuanRewardEnumRp,
		CreatedAt:      bhTanggal,
		Nasabah: &models.Nasabah{
			NasabahID: nasabahID,
			BankID:    bankID,
			Bank:      models.BankSampah{BankID: bankID, NamaBank: namaBank},
			User:      models.User{Nama: nama},
		},
	}
}

func bhItem(p models.PenerimaBagiHasil) services.NasabahPenerimaItem {
	return services.NasabahPenerimaItem{
		PenerimaID:     p.PenerimaID,
		NasabahID:      p.Nasabah.NasabahID,
		NamaNasabah:    p.Nasabah.User.Nama,
		TotalDiterima:  p.TotalDiterima,
		SatuanDiterima: p.SatuanDiterima,
	}
}

func TestPreviewBagiHasil_BSM_Sukses(t *testing.T) {
	gdb, _ := newTestDB(t)
	penjualan := basePenjualan(models.BagiHasilPending)
	penjualan.TotalPenjualan = 100000
	sk := skenarioBaruBSM(penjualan, []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 8, HargaNasabahSnapshot: 3000},
		{SampahID: "SMP-002", Qty: 5, HargaNasabahSnapshot: 2000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)

	sk.sediakanTabungan("SMP-001",
		baseTabunganNasabah("TBG-01", "NSB-001", "SMP-001", 5),
		baseTabunganNasabah("TBG-02", "NSB-002", "SMP-001", 6),
	)
	sk.sediakanTabungan("SMP-002",
		baseTabunganNasabah("TBG-03", "NSB-003", "SMP-002", 5),
	)
	sk.repo.On("GetNasabahInfoBatch", mock.Anything).Return([]repositories.NasabahRow{
		{NasabahID: "NSB-001", NamaNasabah: "Budi", BankID: "BSM-001", NamaBank: "BSM Satu"},
		{NasabahID: "NSB-002", NamaNasabah: "Sari", BankID: "BSM-001", NamaBank: "BSM Satu"},
		{NasabahID: "NSB-003", NamaNasabah: "Joko", BankID: "BSM-001", NamaBank: "BSM Satu"},
	}, nil).Once()

	hasil, err := svc.Preview("PJL-001", "BSM-001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "PJL-001", hasil.PenjualanID)
		assert.Equal(t, string(models.RewardEnumUang), hasil.Reward)
		assert.Equal(t, services.PreviewBagiHasilSummary{
			GrossBank: 100000, TotalDistribusiNasabah: 34000, SisaBagiHasil: 66000,
		}, hasil.Summary)
		if assert.Len(t, hasil.Penerima, 1) {
			assert.Equal(t, "BSM-001", hasil.Penerima[0].BankID)
			assert.Equal(t, "BSM Satu", hasil.Penerima[0].NamaBank)
			assert.ElementsMatch(t, []services.NasabahPenerimaPreview{
				{NasabahID: "NSB-001", NamaNasabah: "Budi", TotalDiterima: 15000},
				{NasabahID: "NSB-002", NamaNasabah: "Sari", TotalDiterima: 9000},
				{NasabahID: "NSB-003", NamaNasabah: "Joko", TotalDiterima: 10000},
			}, hasil.Penerima[0].NasabahPenerima)
		}
	}
	sk.repo.AssertNotCalled(t, "SaveTabungan", mock.Anything, mock.Anything)
	sk.repo.AssertNotCalled(t, "UpsertSchemaHargaNasabah", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	sk.repo.AssertNotCalled(t, "CreateBagiHasil", mock.Anything, mock.Anything)
	sk.repo.AssertExpectations(t)
}

func TestPreviewBagiHasil_TanpaPenerima(t *testing.T) {
	gdb, _ := newTestDB(t)
	sk := skenarioBaruBSM(basePenjualan(models.BagiHasilPending), []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 0, HargaNasabahSnapshot: 3000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)

	hasil, err := svc.Preview("PJL-001", "BSM-001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, services.PreviewBagiHasilSummary{
			GrossBank: 500000, TotalDistribusiNasabah: 0, SisaBagiHasil: 500000,
		}, hasil.Summary)
		assert.NotNil(t, hasil.Penerima, "harus [] supaya JSON-nya bukan null")
		assert.Len(t, hasil.Penerima, 0)
	}
	sk.repo.AssertNotCalled(t, "GetTabunganNasabah", mock.Anything, mock.Anything, mock.Anything)
	sk.repo.AssertNotCalled(t, "GetNasabahInfoBatch", mock.Anything)
}

func TestPreviewBagiHasil_HargaNolDitolak(t *testing.T) {
	gdb, _ := newTestDB(t)
	sk := skenarioBaruBSM(basePenjualan(models.BagiHasilPending), []models.DetailPenjualan{
		{SampahID: "SMP-009", Qty: 2, HargaNasabahSnapshot: 0},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)

	hasil, err := svc.Preview("PJL-001", "BSM-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bhKode(t, err))
	assert.Contains(t, err.Error(), "SMP-009")
	sk.repo.AssertNotCalled(t, "GetTabunganNasabah", mock.Anything, mock.Anything, mock.Anything)
}

func TestPreviewBagiHasil_TabunganError(t *testing.T) {
	gdb, _ := newTestDB(t)
	sk := skenarioBaruBSM(basePenjualan(models.BagiHasilPending), []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 2, HargaNasabahSnapshot: 3000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)
	sk.repo.On("GetTabunganNasabah", mock.Anything, "SMP-001", []string{"BSM-001"}).
		Return([]models.TabunganSampah(nil), errors.New("db error"))

	hasil, err := svc.Preview("PJL-001", "BSM-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
}

func TestPreviewBagiHasil_StokTidakCukup(t *testing.T) {
	gdb, _ := newTestDB(t)
	sk := skenarioBaruBSM(basePenjualan(models.BagiHasilPending), []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 10, HargaNasabahSnapshot: 1000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)
	sk.sediakanTabungan("SMP-001", baseTabunganNasabah("TBG-01", "NSB-001", "SMP-001", 4))

	hasil, err := svc.Preview("PJL-001", "BSM-001")

	assert.Nil(t, hasil)
	var se *services.ServiceError
	if assert.ErrorAs(t, err, &se) {
		assert.Equal(t, 422, se.Code)
		assert.Equal(t, map[string]interface{}{"sampah_id": "SMP-001", "kurang_qty": 6.0}, se.Data)
	}
	sk.repo.AssertNotCalled(t, "GetNasabahInfoBatch", mock.Anything)
}

func TestPreviewBagiHasil_BankTidakDitemukan(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(basePenjualan(models.BagiHasilPending), nil)
	repo.On("GetBankByID", "BSM-001").Return(nil, errors.New("not found"))

	hasil, err := svc.Preview("PJL-001", "BSM-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bhKode(t, err))
}

func TestPreviewBagiHasil_BSUDitolak(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetPenjualanWithReward", "PJL-001", "BSU-001").Return(basePenjualan(models.BagiHasilPending), nil)
	repo.On("GetBankByID", "BSU-001").Return(&models.BankSampah{BankID: "BSU-001", JenisBank: models.BSU}, nil)

	hasil, err := svc.Preview("PJL-001", "BSU-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, bhKode(t, err))
	repo.AssertNotCalled(t, "GetDetailPenjualan", mock.Anything, mock.Anything)
}

func TestPreviewBagiHasil_DetailPenjualanError(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetPenjualanWithReward", "PJL-001", "BSM-001").Return(basePenjualan(models.BagiHasilPending), nil)
	repo.On("GetBankByID", "BSM-001").Return(&models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}, nil)
	repo.On("GetDetailPenjualan", mock.Anything, "PJL-001").Return([]models.DetailPenjualan(nil), errors.New("db error"))

	hasil, err := svc.Preview("PJL-001", "BSM-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
}

func TestSubmitBagiHasil_CreateBagiHasilDuplikat(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	sk := skenarioBaruBSM(basePenjualan(models.BagiHasilPending), []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 5, HargaNasabahSnapshot: 1000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)
	sk.sediakanTabungan("SMP-001", baseTabunganNasabah("TBG-01", "NSB-001", "SMP-001", 5))
	sk.expectTabunganTersimpan("TBG-01", 0)
	sk.expectHargaNasabah("SMP-001", 1000)
	sk.repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey).Once()
	expectTxRollback(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.Nil(t, result)
	assert.Equal(t, http.StatusBadRequest, bhKode(t, err))
	assert.Contains(t, err.Error(), "sudah dilakukan")
	sk.repo.AssertNotCalled(t, "CreatePenerimaBagiHasil", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestSubmitBagiHasil_CreateBagiHasilError(t *testing.T) {
	gdb, dbMock := newTestDB(t)
	sk := skenarioBaruBSM(basePenjualan(models.BagiHasilPending), []models.DetailPenjualan{
		{SampahID: "SMP-001", Qty: 5, HargaNasabahSnapshot: 1000},
	})
	svc := services.NewBagiHasilServiceWithRepo(gdb, sk.repo)
	sk.sediakanTabungan("SMP-001", baseTabunganNasabah("TBG-01", "NSB-001", "SMP-001", 5))
	sk.expectTabunganTersimpan("TBG-01", 0)
	sk.expectHargaNasabah("SMP-001", 1000)
	sk.repo.On("CreateBagiHasil", mock.Anything, mock.Anything).Return(errors.New("db error")).Once()
	expectTxRollback(dbMock)

	result, err := svc.Submit("PJL-001", "BSM-001", "ADM-001")

	assert.Nil(t, result)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
	assert.Contains(t, err.Error(), "Gagal menyimpan bagi hasil")
	sk.repo.AssertNotCalled(t, "CreatePenerimaBagiHasil", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestGetDetailBagiHasil_TidakDitemukan(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetBagiHasilByPenjualan", "PJL-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.GetDetail("PJL-XXX")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bhKode(t, err))
}

func TestGetDetailBagiHasil_PenerimaError(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetBagiHasilByPenjualan", "PJL-001").Return(bhBagiHasil(), nil)
	repo.On("GetNamaPetugas", "ADM-001").Return("Petugas A")
	repo.On("GetPenerimaBagiHasil", "BH-PJL-001").Return([]models.PenerimaBagiHasil(nil), errors.New("db error"))

	hasil, err := svc.GetDetail("PJL-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
}

func TestGetDetailBagiHasil_BSM_Sukses(t *testing.T) {
	repo, svc := newBHSvc(t)
	budi := bhPenerima("P-001", "NSB-001", "BSM-001", "BSM Satu", "Budi", 15000)
	sari := bhPenerima("P-002", "NSB-002", "BSM-001", "BSM Satu", "Sari", 19000)
	tanpaNasabah := models.PenerimaBagiHasil{PenerimaID: "P-003", TotalDiterima: 500}
	repo.On("GetBagiHasilByPenjualan", "PJL-001").Return(bhBagiHasil(), nil)
	repo.On("GetNamaPetugas", "ADM-001").Return("Petugas A").Once()
	repo.On("GetPenerimaBagiHasil", "BH-PJL-001").Return([]models.PenerimaBagiHasil{budi, tanpaNasabah, sari}, nil)
	repo.On("FindDistribusiSisaID", "BH-PJL-001").Return(ptrStr("DS-001"))

	hasil, err := svc.GetDetail("PJL-001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BH-PJL-001", hasil.BagiHasilID)
		assert.Equal(t, bhTanggal, hasil.Tanggal)
		assert.Equal(t, "PJL-001", hasil.PenjualanID)
		assert.Equal(t, "Petugas A", hasil.NamaPetugas)
		assert.Equal(t, string(models.RewardEnumUang), hasil.Reward)
		assert.Equal(t, 100000.0, hasil.GrossBank)
		assert.Equal(t, 34000.0, hasil.TotalDistribusiNasabah)
		assert.Equal(t, 66000.0, hasil.SisaBagiHasil)
		assert.Equal(t, models.SatuanRewardEnumRp, hasil.Satuan)
		if assert.NotNil(t, hasil.DistribusiID) {
			assert.Equal(t, "DS-001", *hasil.DistribusiID)
		}
		assert.Equal(t, []services.NasabahPenerimaItem{bhItem(budi), bhItem(sari)}, hasil.NasabahLangsung,
			"penerima tanpa nasabah dilewati")
		assert.Nil(t, hasil.Penerima)
	}
	repo.AssertExpectations(t)
}

func TestGetDetailBagiHasil_TanpaCreatedBy(t *testing.T) {
	repo, svc := newBHSvc(t)
	bh := bhBagiHasil()
	bh.CreatedBy = nil
	repo.On("GetBagiHasilByPenjualan", "PJL-001").Return(bh, nil)
	repo.On("GetPenerimaBagiHasil", "BH-PJL-001").Return([]models.PenerimaBagiHasil{}, nil)
	repo.On("FindDistribusiSisaID", "BH-PJL-001").Return(nil)

	hasil, err := svc.GetDetail("PJL-001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "", hasil.NamaPetugas)
		assert.Nil(t, hasil.DistribusiID)
		assert.NotNil(t, hasil.NasabahLangsung)
		assert.Len(t, hasil.NasabahLangsung, 0)
	}
	repo.AssertNotCalled(t, "GetNamaPetugas", mock.Anything)
}

func TestGetListBagiHasilNasabah_NasabahTidakDitemukan(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetNasabahWithUser", "NSB-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.GetListNasabah("NSB-XXX", "", "")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bhKode(t, err))
	repo.AssertNotCalled(t, "GetPenerimaByNasabah", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetListBagiHasilNasabah_RepoError(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetNasabahWithUser", "NSB-001").Return(&models.Nasabah{NasabahID: "NSB-001"}, nil)
	repo.On("GetPenerimaByNasabah", "NSB-001", "", "").Return([]models.PenerimaBagiHasil(nil), errors.New("db error"))

	hasil, err := svc.GetListNasabah("NSB-001", "", "")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
}

func TestGetListBagiHasilNasabah_Sukses(t *testing.T) {
	repo, svc := newBHSvc(t)
	valid := bhPenerima("P-001", "NSB-001", "BSM-001", "BSM Satu", "Budi", 15000)
	valid.BagiHasil = bhBagiHasil()
	tanpaBagiHasil := bhPenerima("P-002", "NSB-001", "BSM-001", "BSM Satu", "Budi", 7000)
	tanpaPenjualan := bhPenerima("P-003", "NSB-001", "BSM-001", "BSM Satu", "Budi", 3000)
	tanpaPenjualan.BagiHasil = &models.BagiHasil{BagiHasilID: "BH-PJL-002"}

	repo.On("GetNasabahWithUser", "NSB-001").
		Return(&models.Nasabah{NasabahID: "NSB-001", User: models.User{Nama: "Budi"}}, nil)
	repo.On("GetPenerimaByNasabah", "NSB-001", "2026-03-01", "2026-03-31").
		Return([]models.PenerimaBagiHasil{valid, tanpaBagiHasil, tanpaPenjualan}, nil).Once()

	hasil, err := svc.GetListNasabah("NSB-001", "2026-03-01", "2026-03-31")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "NSB-001", hasil.NasabahID)
		assert.Equal(t, "Budi", hasil.NamaNasabah)
		assert.Equal(t, []services.ListBagiHasilNasabahItem{{
			PenerimaID:     "P-001",
			BagiHasilID:    "BH-PJL-001",
			Reward:         string(models.RewardEnumUang),
			Tanggal:        "2026-03-10T09:00:00Z",
			TotalDiterima:  15000,
			SatuanDiterima: models.SatuanRewardEnumRp,
		}}, hasil.RiwayatBagiHasil, "penerima tanpa bagi hasil/penjualan dilewati")
	}
	repo.AssertExpectations(t)
}

func TestGetDetailBagiHasilNasabah_TidakDitemukan(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetPenerimaBagiHasilNasabah", "P-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.GetDetailNasabah("P-XXX")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bhKode(t, err))
}

func TestGetDetailBagiHasilNasabah_DataTidakLengkap(t *testing.T) {
	repo, svc := newBHSvc(t)
	penerima := bhPenerima("P-001", "NSB-001", "BSM-001", "BSM Satu", "Budi", 15000)
	repo.On("GetPenerimaBagiHasilNasabah", "P-001").Return(&penerima, nil)

	hasil, err := svc.GetDetailNasabah("P-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
	repo.AssertNotCalled(t, "GetDetailBagiHasilItems", mock.Anything)
}

func TestGetDetailBagiHasilNasabah_DetailError(t *testing.T) {
	repo, svc := newBHSvc(t)
	penerima := bhPenerima("P-001", "NSB-001", "BSM-001", "BSM Satu", "Budi", 15000)
	penerima.BagiHasil = bhBagiHasil()
	repo.On("GetPenerimaBagiHasilNasabah", "P-001").Return(&penerima, nil)
	repo.On("GetDetailBagiHasilItems", "P-001").Return([]models.DetailBagiHasil(nil), errors.New("db error"))

	hasil, err := svc.GetDetailNasabah("P-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
}

func TestGetDetailBagiHasilNasabah_Sukses(t *testing.T) {
	repo, svc := newBHSvc(t)
	penerima := bhPenerima("P-001", "NSB-001", "BSM-001", "BSM Satu", "Budi", 21000)
	penerima.BagiHasil = bhBagiHasil()
	repo.On("GetPenerimaBagiHasilNasabah", "P-001").Return(&penerima, nil)
	repo.On("GetDetailBagiHasilItems", "P-001").Return([]models.DetailBagiHasil{
		{PenerimaID: "P-001", SampahID: "SMP-001", Qty: 5, HargaItem: 3000, SubtotalHarga: 15000,
			KatalogSampah: &models.KatalogSampah{Sarok: models.Sampah{NamaSampah: "Botol Plastik"}}},
		{PenerimaID: "P-001", SampahID: "SMP-002", Qty: 3, HargaItem: 2000, SubtotalHarga: 6000},
	}, nil)

	hasil, err := svc.GetDetailNasabah("P-001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "P-001", hasil.PenerimaID)
		assert.Equal(t, "BH-PJL-001", hasil.BagiHasilID)
		assert.Equal(t, "NSB-001", hasil.NasabahID)
		assert.Equal(t, "Budi", hasil.NamaNasabah)
		assert.Equal(t, string(models.RewardEnumUang), hasil.Reward)
		assert.Equal(t, "2026-03-10T09:00:00Z", hasil.Tanggal)
		assert.Equal(t, 21000.0, hasil.TotalDiterima)
		assert.Equal(t, models.SatuanRewardEnumRp, hasil.SatuanDiterima)
		assert.Equal(t, []services.DetailBagiHasilItemResp{
			{SampahID: "SMP-001", NamaSampah: "Botol Plastik", Qty: 5, HargaItem: 3000, SubtotalHarga: 15000},
			{SampahID: "SMP-002", NamaSampah: "", Qty: 3, HargaItem: 2000, SubtotalHarga: 6000},
		}, hasil.DetailItem)
	}
}

func TestGetListBagiHasilBankPusat_BankTidakDitemukan(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetBankByID", "BSM-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.GetListBankPusat("BSM-XXX", time.Time{}, time.Time{})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bhKode(t, err))
}

func TestGetListBagiHasilBankPusat_BSUDitolak(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetBankByID", "BSU-001").Return(&models.BankSampah{BankID: "BSU-001", JenisBank: models.BSU}, nil)

	hasil, err := svc.GetListBankPusat("BSU-001", time.Time{}, time.Time{})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bhKode(t, err))
	repo.AssertNotCalled(t, "GetBagiHasilByBankWithFilter", mock.Anything, mock.Anything, mock.Anything)
}

func TestGetListBagiHasilBankPusat_RepoError(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetBankByID", "BSM-001").Return(&models.BankSampah{BankID: "BSM-001", JenisBank: models.BSM}, nil)
	repo.On("GetBagiHasilByBankWithFilter", "BSM-001", mock.Anything, mock.Anything).
		Return([]models.BagiHasil(nil), errors.New("db error"))

	hasil, err := svc.GetListBankPusat("BSM-001", time.Time{}, time.Time{})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
}

func TestGetListBagiHasilBankPusat_Sukses(t *testing.T) {
	repo, svc := newBHSvc(t)
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	repo.On("GetBankByID", "BSM-001").
		Return(&models.BankSampah{BankID: "BSM-001", NamaBank: "BSM Satu", JenisBank: models.BSM}, nil)
	repo.On("GetBagiHasilByBankWithFilter", "BSM-001", start, end).
		Return([]models.BagiHasil{*bhBagiHasil(), {BagiHasilID: "BH-PJL-002"}}, nil).Once()

	hasil, err := svc.GetListBankPusat("BSM-001", start, end)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BSM-001", hasil.BankID)
		assert.Equal(t, "BSM Satu", hasil.NamaBank)
		assert.Equal(t, models.BSM, hasil.JenisBank)
		assert.Equal(t, []services.ListBagiHasilBankPusatItem{{
			BagiHasilID:      "BH-PJL-001",
			RewardID:         1,
			NamaReward:       string(models.RewardEnumUang),
			TanggalBagiHasil: "2026-03-10T09:00:00Z",
		}}, hasil.RiwayatBagiHasil, "bagi hasil tanpa penjualan dilewati")
	}
	repo.AssertExpectations(t)
}

func TestGetDetailBagiHasilBankPusat_TidakDitemukan(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetBagiHasilWithAll", "BH-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.GetDetailBankPusat("BH-XXX")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, bhKode(t, err))
}

func TestGetDetailBagiHasilBankPusat_DataTidakLengkap(t *testing.T) {
	repo, svc := newBHSvc(t)
	bh := bhBagiHasil()
	bh.Bank = nil
	repo.On("GetBagiHasilWithAll", "BH-PJL-001").Return(bh, nil)

	hasil, err := svc.GetDetailBankPusat("BH-PJL-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
}

func TestGetDetailBagiHasilBankPusat_BSUDitolak(t *testing.T) {
	repo, svc := newBHSvc(t)
	bh := bhBagiHasil()
	bh.Bank.JenisBank = models.BSU
	repo.On("GetBagiHasilWithAll", "BH-PJL-001").Return(bh, nil)

	hasil, err := svc.GetDetailBankPusat("BH-PJL-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, bhKode(t, err))
	repo.AssertNotCalled(t, "FindDistribusiSisaID", mock.Anything)
}

func TestGetDetailBagiHasilBankPusat_PenerimaError(t *testing.T) {
	repo, svc := newBHSvc(t)
	repo.On("GetBagiHasilWithAll", "BH-PJL-001").Return(bhBagiHasil(), nil)
	repo.On("FindDistribusiSisaID", "BH-PJL-001").Return(nil)
	repo.On("GetPenerimaNasabahFull", "BH-PJL-001").Return([]models.PenerimaBagiHasil(nil), errors.New("db error"))

	hasil, err := svc.GetDetailBankPusat("BH-PJL-001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, bhKode(t, err))
}

func TestGetDetailBagiHasilBankPusat_BSM_Sukses(t *testing.T) {
	repo, svc := newBHSvc(t)
	budi := bhPenerima("P-001", "NSB-001", "BSM-001", "BSM Satu", "Budi", 15000)
	sari := bhPenerima("P-002", "NSB-002", "BSM-002", "", "Sari", 9000)
	joko := bhPenerima("P-003", "NSB-003", "BSM-001", "BSM Satu", "Joko", 10000)
	repo.On("GetBagiHasilWithAll", "BH-PJL-001").Return(bhBagiHasil(), nil)
	repo.On("FindDistribusiSisaID", "BH-PJL-001").Return(ptrStr("DS-001"))
	repo.On("GetPenerimaNasabahFull", "BH-PJL-001").Return([]models.PenerimaBagiHasil{budi, sari, joko}, nil)

	hasil, err := svc.GetDetailBankPusat("BH-PJL-001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "BH-PJL-001", hasil.BagiHasilID)
		assert.Equal(t, "PJL-001", hasil.PenjualanID)
		assert.Equal(t, 1, hasil.RewardID)
		assert.Equal(t, string(models.RewardEnumUang), hasil.NamaReward)
		assert.Equal(t, "2026-03-10T09:00:00Z", hasil.Tanggal)
		assert.Equal(t, 100000.0, hasil.GrossBSM, "gross BSM diambil dari gross_bank")
		assert.Equal(t, 0.0, hasil.GrossBSI)
		assert.Equal(t, 34000.0, hasil.TotalDistribusiNasabah)
		assert.Equal(t, 66000.0, hasil.SisaBagiHasil)
		if assert.NotNil(t, hasil.DistribusiID) {
			assert.Equal(t, "DS-001", *hasil.DistribusiID)
		}
		assert.Equal(t, []services.BankPenerimaGroup{
			{BankID: "BSM-001", NamaBank: "BSM Satu", NasabahPenerima: []services.NasabahPenerimaItem{bhItem(budi), bhItem(joko)}},
			{BankID: "BSM-002", NamaBank: "Bank Sampah", NasabahPenerima: []services.NasabahPenerimaItem{bhItem(sari)}},
		}, hasil.Penerima, "dikelompokkan per bank sesuai urutan muncul, nama bank kosong pakai fallback")
		assert.Equal(t, []services.NasabahPenerimaItem{bhItem(budi), bhItem(sari), bhItem(joko)}, hasil.NasabahBSM)
		assert.Nil(t, hasil.NasabahBSI)
	}
}
