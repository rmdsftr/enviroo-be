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

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

const (
	dshBank    = "301002001"
	dshNasabah = "433010020010001"
	dshUser    = "1371010101010001"
	dshAdmin   = "23301002001001"
)

func newDshSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockDashboardRepo, services.DashboardService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockDashboardRepo)
	return dbMock, repo, services.NewDashboardService(gdb, repo)
}

func dshBankModel(bankID string, jenis models.JenisBank) *models.BankSampah {
	return &models.BankSampah{
		BankID: bankID, NamaBank: "Bank " + bankID, JenisBank: jenis, IsActive: true,
		Alamat: "Jl. Mawar 1", PhotoURL: "https://cdn/bank.png",
	}
}

func dshBSUModel() *models.BankSampah {
	bank := dshBankModel(dshBank, models.BSU)
	bank.ParentBankID = ptrStr("101002001")
	return bank
}

func dshSaldo(uang, poin float64) repositories.SaldoPerSatuan {
	return repositories.SaldoPerSatuan{Uang: uang, Poin: poin}
}

func dshReward(id int, nama models.RewardEnum, satuan models.SatuanRewardEnum) *models.Reward {
	return &models.Reward{RewardID: id, NamaReward: nama, Satuan: satuan}
}

func dshRekeningBank(rekeningID string, nominal float64) *models.SaldoRekening {
	bankID, rewardID := dshBank, 1
	return &models.SaldoRekening{
		RekeningID: rekeningID, BankID: &bankID, RewardID: &rewardID,
		Entitas: models.EntitasBankSampah, NominalSaldo: nominal,
		SatuanNominalSaldo: models.SatuanRewardEnumRp,
	}
}

func dshArus(sebelum, sesudah float64, keterangan string) models.RiwayatArusSaldo {
	return models.RiwayatArusSaldo{
		RiwayatSaldoID: "RWY-001",
		NominalSebelum: sebelum,
		NominalSesudah: sesudah,
		Keterangan:     keterangan,
		CreatedAt:      time.Date(2026, time.September, 10, 8, 0, 0, 0, time.Local),
	}
}

func dshAkses() services.AksesDashboardInput {
	return services.AksesDashboardInput{
		ActorUserID: dshUser, ActorBankID: dshBank, ActorRole: models.AdminBSU,
	}
}

func dshCatatInput(tipe, jenis string, nominal float64) services.CatatMutasiInput {
	return services.CatatMutasiInput{
		TipeMutasi: tipe, JenisReward: jenis, Nominal: nominal, Keterangan: "Koreksi kas",
	}
}

func expectTxMutasi(m sqlmock.Sqlmock, commit bool) {
	m.ExpectBegin()
	m.ExpectExec("UPDATE").WillReturnResult(sqlmock.NewResult(1, 1))
	m.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"riwayat_saldo_id"}))
	m.ExpectExec("INSERT").WillReturnResult(sqlmock.NewResult(1, 1))
	if commit {
		m.ExpectCommit()
	} else {
		m.ExpectRollback()
	}
}

func dshKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func TestDashboardPetugas_BankIDKosong(t *testing.T) {
	_, _, svc := newDshSvc(t)

	hasil, err := svc.DashboardPetugas("")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, dshKode(t, err))
}

func TestDashboardPetugas_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(nil, errors.New("not found"))

	hasil, err := svc.DashboardPetugas(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestDashboardPetugas_JenisBankTidakDikenali(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, "koperasi"), nil)

	hasil, err := svc.DashboardPetugas(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, dshKode(t, err))
	assert.Contains(t, err.Error(), "Jenis bank")
}

func TestDashboardPetugas_HitungNasabahError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("CountNasabahAktif", dshBank).Return(int64(0), errors.New("db error"))

	hasil, err := svc.DashboardPetugas(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	assert.Contains(t, err.Error(), "nasabah")
}

func TestDashboardPetugas_HitungStaffError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("CountNasabahAktif", dshBank).Return(int64(5), nil)
	repo.On("CountAdminAktif", dshBank).Return(int64(0), errors.New("db error"))

	hasil, err := svc.DashboardPetugas(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	assert.Contains(t, err.Error(), "staff")
}

func TestDashboardPetugas_KasError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("CountNasabahAktif", dshBank).Return(int64(5), nil)
	repo.On("CountAdminAktif", dshBank).Return(int64(2), nil)
	repo.On("SumSaldoBank", dshBank).Return(dshSaldo(0, 0), errors.New("db error"))

	hasil, err := svc.DashboardPetugas(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	assert.Contains(t, err.Error(), "kas")
}

func TestDashboardPetugas_BSUTanpaPoinDenganIndukI(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	bank := dshBankModel(dshBank, models.BSU)
	bank.ParentBankID = ptrStr("101002001")
	repo.On("FindBank", dshBank).Return(bank, nil)
	repo.On("CountNasabahAktif", dshBank).Return(int64(5), nil)
	repo.On("CountAdminAktif", dshBank).Return(int64(2), nil)
	repo.On("SumSaldoBank", dshBank).Return(dshSaldo(250000, 40), nil)
	repo.On("FindBank", "101002001").Return(dshBankModel("101002001", models.BSI), nil)

	hasil, err := svc.DashboardPetugas(dshBank)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, float64(250000), hasil.Kas.TotalUang)
		assert.Nil(t, hasil.Kas.TotalPoin, "BSU tidak mengirim key total_poin sama sekali")
		assert.Nil(t, hasil.JumlahBSU, "BSU tidak punya BSU di bawahnya")
		if assert.NotNil(t, hasil.NamaBankPusat) {
			assert.Equal(t, "Bank 101002001", *hasil.NamaBankPusat)
		}
		assert.Equal(t, int64(5), hasil.JumlahNasabah)
		assert.Equal(t, int64(2), hasil.JumlahStaff)
	}
}

func TestDashboardPetugas_BSUTanpaInduk(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("CountNasabahAktif", dshBank).Return(int64(0), nil)
	repo.On("CountAdminAktif", dshBank).Return(int64(0), nil)
	repo.On("SumSaldoBank", dshBank).Return(dshSaldo(0, 0), nil)

	hasil, err := svc.DashboardPetugas(dshBank)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil.NamaBankPusat) {
		assert.Equal(t, "", *hasil.NamaBankPusat)
	}
}

func TestDashboardPetugas_BSIDenganJumlahBSU(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", "101002001").Return(dshBankModel("101002001", models.BSI), nil)
	repo.On("CountNasabahAktif", "101002001").Return(int64(12), nil)
	repo.On("CountAdminAktif", "101002001").Return(int64(4), nil)
	repo.On("SumSaldoBank", "101002001").Return(dshSaldo(900000, 320), nil)
	repo.On("CountBSU", "101002001").Return(int64(3), nil)

	hasil, err := svc.DashboardPetugas("101002001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		if assert.NotNil(t, hasil.Kas.TotalPoin) {
			assert.Equal(t, float64(320), *hasil.Kas.TotalPoin)
		}
		if assert.NotNil(t, hasil.JumlahBSU) {
			assert.Equal(t, int64(3), *hasil.JumlahBSU)
		}
		assert.Nil(t, hasil.NamaBankPusat, "BSI tidak punya bank pusat")
	}
}

func TestDashboardPetugas_BSIHitungBSUError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", "101002001").Return(dshBankModel("101002001", models.BSI), nil)
	repo.On("CountNasabahAktif", "101002001").Return(int64(0), nil)
	repo.On("CountAdminAktif", "101002001").Return(int64(0), nil)
	repo.On("SumSaldoBank", "101002001").Return(dshSaldo(0, 0), nil)
	repo.On("CountBSU", "101002001").Return(int64(0), errors.New("db error"))

	hasil, err := svc.DashboardPetugas("101002001")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	assert.Contains(t, err.Error(), "BSU")
}

func TestDashboardPetugas_BSM(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", "201002001").Return(dshBankModel("201002001", models.BSM), nil)
	repo.On("CountNasabahAktif", "201002001").Return(int64(7), nil)
	repo.On("CountAdminAktif", "201002001").Return(int64(1), nil)
	repo.On("SumSaldoBank", "201002001").Return(dshSaldo(75000, 15), nil)

	hasil, err := svc.DashboardPetugas("201002001")

	assert.NoError(t, err)
	assert.NotNil(t, hasil.Kas.TotalPoin)
	assert.Nil(t, hasil.JumlahBSU)
	assert.Nil(t, hasil.NamaBankPusat)
	repo.AssertNotCalled(t, "CountBSU", mock.Anything)
}

func TestSaldoBank_BankIDKosong(t *testing.T) {
	_, _, svc := newDshSvc(t)

	hasil, err := svc.SaldoBank("")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, dshKode(t, err))
}

func TestSaldoBank_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(nil, errors.New("not found"))

	hasil, err := svc.SaldoBank(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestSaldoBank_RepoError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("SumSaldoBank", dshBank).Return(dshSaldo(0, 0), errors.New("db error"))

	hasil, err := svc.SaldoBank(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
}

func TestSaldoBank_BSUTanpaPoin(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("SumSaldoBank", dshBank).Return(dshSaldo(250000, 40), nil)

	hasil, err := svc.SaldoBank(dshBank)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, float64(250000), hasil.Uang.TotalUang)
		assert.Equal(t, models.SatuanRewardEnumRp, hasil.Uang.SatuanUang)
		assert.Nil(t, hasil.Poin, "BSU tidak mengelola poin")
	}
}

func TestSaldoBank_BSIDenganPoin(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", "101002001").Return(dshBankModel("101002001", models.BSI), nil)
	repo.On("SumSaldoBank", "101002001").Return(dshSaldo(500000, 80), nil)

	hasil, err := svc.SaldoBank("101002001")

	assert.NoError(t, err)
	if assert.NotNil(t, hasil.Poin) {
		assert.Equal(t, float64(80), hasil.Poin.TotalPoin)
		assert.Equal(t, models.SatuanRewardEnumPoin, hasil.Poin.SatuanPoin)
	}
}

func TestSaldoNasabah_TidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindNasabah", dshNasabah).Return(nil, errors.New("not found"))

	hasil, err := svc.SaldoNasabah(dshNasabah)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestSaldoNasabah_RepoError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindNasabah", dshNasabah).Return(&models.Nasabah{NasabahID: dshNasabah}, nil)
	repo.On("SumSaldoNasabah", dshNasabah).Return(dshSaldo(0, 0), errors.New("db error"))

	hasil, err := svc.SaldoNasabah(dshNasabah)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
}

func TestSaldoNasabah_SelaluDuaSatuan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindNasabah", dshNasabah).Return(&models.Nasabah{NasabahID: dshNasabah}, nil)
	repo.On("SumSaldoNasabah", dshNasabah).Return(dshSaldo(50000, 120), nil)

	hasil, err := svc.SaldoNasabah(dshNasabah)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, float64(50000), hasil.Uang.TotalUang)
		assert.Equal(t, models.SatuanRewardEnumRp, hasil.Uang.SatuanUang)
		assert.Equal(t, float64(120), hasil.Poin.TotalPoin)
		assert.Equal(t, models.SatuanRewardEnumPoin, hasil.Poin.SatuanPoin)
	}
}

func TestTotalSaldoSemuaNasabah_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(nil, errors.New("not found"))

	hasil, err := svc.TotalSaldoSemuaNasabah(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestTotalSaldoSemuaNasabah_RepoError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("SumSaldoSemuaNasabah", dshBank).Return(dshSaldo(0, 0), errors.New("db error"))

	hasil, err := svc.TotalSaldoSemuaNasabah(dshBank)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
}

func TestTotalSaldoSemuaNasabah_Sukses(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("SumSaldoSemuaNasabah", dshBank).Return(dshSaldo(1250000, 430), nil)

	hasil, err := svc.TotalSaldoSemuaNasabah(dshBank)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, float64(1250000), hasil.TotalUang)
		assert.Equal(t, float64(430), hasil.TotalPoin)
	}
}

func TestListSaldoSemuaNasabah_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(nil, errors.New("not found"))

	items, meta, err := svc.ListSaldoSemuaNasabah(dshBank, 1)

	assert.Nil(t, items)
	assert.Nil(t, meta)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestListSaldoSemuaNasabah_CountError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("CountNasabahBank", dshBank).Return(int64(0), errors.New("db error"))

	items, _, err := svc.ListSaldoSemuaNasabah(dshBank, 1)

	assert.Nil(t, items)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
}

func TestListSaldoSemuaNasabah_ListError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("CountNasabahBank", dshBank).Return(int64(10), nil)
	repo.On("ListNasabahDenganSaldo", dshBank, 20, 0).Return(nil, errors.New("db error"))

	items, _, err := svc.ListSaldoSemuaNasabah(dshBank, 1)

	assert.Nil(t, items)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
}

func TestListSaldoSemuaNasabah_PageDinormalisasi(t *testing.T) {
	for _, page := range []int{0, -3} {
		_, repo, svc := newDshSvc(t)
		repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
		repo.On("CountNasabahBank", dshBank).Return(int64(0), nil)
		repo.On("ListNasabahDenganSaldo", dshBank, 20, 0).Return([]repositories.NasabahSaldoRow{}, nil)

		items, meta, err := svc.ListSaldoSemuaNasabah(dshBank, page)

		assert.NoError(t, err)
		assert.NotNil(t, items, "slice kosong harus non-nil supaya JSON-nya []")
		if assert.NotNil(t, meta) {
			assert.Equal(t, 1, meta.Page)
			assert.Equal(t, 20, meta.Limit)
		}
		repo.AssertExpectations(t)
	}
}

func TestListSaldoSemuaNasabah_HalamanKeduaDanMapping(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	rows := []repositories.NasabahSaldoRow{
		{NasabahID: dshNasabah, NamaNasabah: "Budi Santoso", StatusNasabah: models.Aktif,
			SaldoUang: 50000, SaldoPoin: 120},
	}
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("CountNasabahBank", dshBank).Return(int64(45), nil)
	repo.On("ListNasabahDenganSaldo", dshBank, 20, 20).Return(rows, nil)

	items, meta, err := svc.ListSaldoSemuaNasabah(dshBank, 2)

	assert.NoError(t, err)
	if assert.Len(t, items, 1) {
		assert.Equal(t, "Budi Santoso", items[0].NamaNasabah)
		assert.Equal(t, models.Aktif, items[0].StatusNasabah)
		assert.Equal(t, float64(50000), items[0].SaldoUang)
		assert.Equal(t, float64(120), items[0].SaldoPoin)
	}
	if assert.NotNil(t, meta) {
		assert.Equal(t, 45, meta.Total)
		assert.Equal(t, 3, meta.TotalPages)
	}
}

func TestMutasiBank_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(nil, errors.New("not found"))

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestMutasiBank_TanpaRewardIDPakaiRewardUang(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindRewardByNama", models.RewardEnumUang).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningBank", mock.Anything, dshBank, 1).Return(dshRekeningBank("1"+dshBank, 100000), nil)
	repo.On("ListArusSaldo", "1"+dshBank, mock.Anything, mock.Anything).
		Return([]models.RiwayatArusSaldo{}, nil)

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{})

	assert.NoError(t, err)
	assert.Equal(t, models.RewardEnumUang, hasil.NamaReward)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "FindReward", mock.Anything)
}

func TestMutasiBank_RewardUangBelumTerdaftar(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindRewardByNama", models.RewardEnumUang).Return(nil, errors.New("not found"))

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
	assert.Contains(t, err.Error(), "Uang")
}

func TestMutasiBank_RewardIDTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 9).Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{RewardID: 9})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestMutasiBank_RewardErrorDBLain(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 9).Return(nil, errors.New("db error"))

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{RewardID: 9})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
}

func TestMutasiBank_RekeningBelumAdaJadiMutasiKosong(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 2).Return(dshReward(2, models.RewardEnumBarang, models.SatuanRewardEnumPoin), nil)
	repo.On("FindRekeningBank", mock.Anything, dshBank, 2).Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{RewardID: 2})

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.RewardEnumBarang, hasil.NamaReward, "jenis reward tetap dilaporkan")
		assert.Equal(t, models.SatuanRewardEnumPoin, hasil.SatuanReward)
		assert.NotNil(t, hasil.MutasiItems)
		assert.Len(t, hasil.MutasiItems, 0)
		assert.Equal(t, float64(0), hasil.TotalDebit)
	}
	repo.AssertNotCalled(t, "ListArusSaldo", mock.Anything, mock.Anything, mock.Anything)
}

func TestMutasiBank_CariRekeningErrorLain(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 1).Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningBank", mock.Anything, dshBank, 1).Return(nil, errors.New("db error"))

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{RewardID: 1})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
}

func TestMutasiBank_ArusSaldoError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 1).Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningBank", mock.Anything, dshBank, 1).Return(dshRekeningBank("1"+dshBank, 0), nil)
	repo.On("ListArusSaldo", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{RewardID: 1})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	assert.Contains(t, err.Error(), "arus saldo")
}

func TestMutasiBank_HitungDebitKredit(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	arus := []models.RiwayatArusSaldo{
		dshArus(0, 100000, "Setoran"),
		dshArus(100000, 60000, "Penarikan"),
		dshArus(60000, 85000, "Bagi hasil"),
	}
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 1).Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningBank", mock.Anything, dshBank, 1).Return(dshRekeningBank("1"+dshBank, 85000), nil)
	repo.On("ListArusSaldo", mock.Anything, mock.Anything, mock.Anything).Return(arus, nil)

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{RewardID: 1})

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, float64(125000), hasil.TotalKredit, "100000 + 25000")
		assert.Equal(t, float64(40000), hasil.TotalDebit)
		if assert.Len(t, hasil.MutasiItems, 3) {
			assert.True(t, hasil.MutasiItems[0].IsPositive)
			assert.Equal(t, float64(100000), hasil.MutasiItems[0].Nominal)
			assert.Equal(t, "Setoran", hasil.MutasiItems[0].Keterangan)
			assert.False(t, hasil.MutasiItems[1].IsPositive)
			assert.Equal(t, float64(40000), hasil.MutasiItems[1].Nominal)
		}
	}
}

func TestMutasiBank_NominalTidakBerubahDilewati(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	arus := []models.RiwayatArusSaldo{
		dshArus(50000, 50000, "Blokir saldo"),
		dshArus(50000, 70000, "Setoran"),
	}
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 1).Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningBank", mock.Anything, dshBank, 1).Return(dshRekeningBank("1"+dshBank, 70000), nil)
	repo.On("ListArusSaldo", mock.Anything, mock.Anything, mock.Anything).Return(arus, nil)

	hasil, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{RewardID: 1})

	assert.NoError(t, err)
	assert.Len(t, hasil.MutasiItems, 1)
	assert.Equal(t, "Setoran", hasil.MutasiItems[0].Keterangan)
}

func TestMutasiBank_RentangTanggalSampaiAkhirHari(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 1).Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningBank", mock.Anything, dshBank, 1).Return(dshRekeningBank("1"+dshBank, 0), nil)
	repo.On("ListArusSaldo", mock.Anything,
		mock.MatchedBy(func(p *time.Time) bool {
			return p != nil && p.Format("2006-01-02 15:04:05") == "2026-09-01 00:00:00"
		}),
		mock.MatchedBy(func(p *time.Time) bool {
			return p != nil && p.Format("2006-01-02 15:04:05") == "2026-09-30 23:59:59"
		})).Return([]models.RiwayatArusSaldo{}, nil)

	_, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{
		RewardID: 1, StartDate: "2026-09-01", EndDate: "2026-09-30",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMutasiBank_TanggalSalahFormatDiabaikan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBankModel(dshBank, models.BSU), nil)
	repo.On("FindReward", 1).Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningBank", mock.Anything, dshBank, 1).Return(dshRekeningBank("1"+dshBank, 0), nil)
	repo.On("ListArusSaldo", mock.Anything, (*time.Time)(nil), (*time.Time)(nil)).
		Return([]models.RiwayatArusSaldo{}, nil)

	_, err := svc.MutasiBank(dshBank, services.FilterMutasiInput{
		RewardID: 1, StartDate: "01-09-2026", EndDate: "bukan tanggal",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMutasiNasabah_TidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindNasabah", dshNasabah).Return(nil, errors.New("not found"))

	hasil, err := svc.MutasiNasabah(dshNasabah, services.FilterMutasiInput{})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestMutasiNasabah_RekeningBelumAdaJadiMutasiKosong(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindNasabah", dshNasabah).Return(&models.Nasabah{NasabahID: dshNasabah}, nil)
	repo.On("FindReward", 2).Return(dshReward(2, models.RewardEnumBarang, models.SatuanRewardEnumPoin), nil)
	repo.On("FindRekeningNasabah", dshNasabah, 2).Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.MutasiNasabah(dshNasabah, services.FilterMutasiInput{RewardID: 2})

	assert.NoError(t, err)
	assert.Len(t, hasil.MutasiItems, 0)
	assert.Equal(t, models.RewardEnumBarang, hasil.NamaReward)
}

func TestMutasiNasabah_CariRekeningErrorLain(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindNasabah", dshNasabah).Return(&models.Nasabah{NasabahID: dshNasabah}, nil)
	repo.On("FindReward", 1).Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningNasabah", dshNasabah, 1).Return(nil, errors.New("db error"))

	hasil, err := svc.MutasiNasabah(dshNasabah, services.FilterMutasiInput{RewardID: 1})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	assert.Contains(t, err.Error(), "rekening nasabah")
}

func TestMutasiNasabah_HitungDebitKredit(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	arus := []models.RiwayatArusSaldo{
		dshArus(0, 30000, "Saldo awal nasabah"),
		dshArus(30000, 5000, "Penarikan"),
	}
	rekening := &models.SaldoRekening{RekeningID: "1" + dshNasabah}
	repo.On("FindNasabah", dshNasabah).Return(&models.Nasabah{NasabahID: dshNasabah}, nil)
	repo.On("FindReward", 1).Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindRekeningNasabah", dshNasabah, 1).Return(rekening, nil)
	repo.On("ListArusSaldo", "1"+dshNasabah, mock.Anything, mock.Anything).Return(arus, nil)

	hasil, err := svc.MutasiNasabah(dshNasabah, services.FilterMutasiInput{RewardID: 1})

	assert.NoError(t, err)
	assert.Equal(t, float64(30000), hasil.TotalKredit)
	assert.Equal(t, float64(25000), hasil.TotalDebit)
	assert.Len(t, hasil.MutasiItems, 2)
}

func TestCatatMutasi_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(nil, errors.New("not found"))

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 1000), dshAkses())

	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestCatatMutasi_ValidasiInput(t *testing.T) {
	kasus := []struct {
		nama  string
		in    services.CatatMutasiInput
		pesan string
	}{
		{"tipe mutasi invalid", dshCatatInput("transfer", "uang", 1000), "tipe_mutasi"},
		{"jenis reward invalid", dshCatatInput("kredit", "emas", 1000), "jenis_reward"},
		{"nominal nol", dshCatatInput("kredit", "uang", 0), "nominal"},
		{"nominal negatif", dshCatatInput("kredit", "uang", -500), "nominal"},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			_, repo, svc := newDshSvc(t)
			repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)

			err := svc.CatatManualMutasiBank(dshBank, k.in, dshAkses())

			assert.Equal(t, http.StatusBadRequest, dshKode(t, err))
			assert.Contains(t, err.Error(), k.pesan)
		})
	}
}

func TestCatatMutasi_BSUTolakPoin(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "poin", 50), dshAkses())

	assert.Equal(t, http.StatusBadRequest, dshKode(t, err))
	assert.Contains(t, err.Error(), "BSU tidak mengelola saldo poin")
	repo.AssertNotCalled(t, "FindRewardByJenis", mock.Anything, mock.Anything)
}

func TestCatatMutasi_BSIBolehPoin(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	repo.On("FindBank", "101002001").Return(dshBankModel("101002001", models.BSI), nil)
	repo.On("FindRewardByJenis", models.RewardEnumBarang, models.SatuanRewardEnumPoin).
		Return(dshReward(2, models.RewardEnumBarang, models.SatuanRewardEnumPoin), nil)
	repo.On("AdaNilaiRewardAktif", "101002001", 2).Return(true, nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, "101002001", 2).
		Return(&models.SaldoRekening{RekeningID: "2101002001", NominalSaldo: 100}, nil)
	expectTxMutasi(dbMock, true)

	err := svc.CatatManualMutasiBank("101002001", dshCatatInput("kredit", "poin", 50), dshAkses())

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCatatMutasi_RewardTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).Return(nil, errors.New("not found"))

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 1000), dshAkses())

	assert.Equal(t, http.StatusNotFound, dshKode(t, err))
}

func TestCatatMutasi_AdminLookupError(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("AdaNilaiRewardAktif", "101002001", 1).Return(true, nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return("", errors.New("db error"))

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 1000), dshAkses())

	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
}

func TestCatatMutasi_AdminTidakDitemukan(t *testing.T) {
	_, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("AdaNilaiRewardAktif", "101002001", 1).Return(true, nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return("", nil)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 1000), dshAkses())

	assert.Equal(t, http.StatusForbidden, dshKode(t, err))
	assert.Contains(t, err.Error(), "admin aktif")
}

func TestCatatMutasi_RekeningDibuatOtomatis(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("AdaNilaiRewardAktif", "101002001", 1).Return(true, nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, dshBank, 1).Return(nil, gorm.ErrRecordNotFound)

	var dibuat models.SaldoRekening
	var adaYangDibuat bool
	repo.On("CreateRekening", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			dibuat, adaYangDibuat = *args.Get(1).(*models.SaldoRekening), true
		}).Return(nil)
	expectTxMutasi(dbMock, true)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 1000), dshAkses())

	assert.NoError(t, err)
	if assert.True(t, adaYangDibuat) {
		assert.Equal(t, "1"+dshBank, dibuat.RekeningID, "rekening_id = reward_id + bank_id")
		assert.Equal(t, models.EntitasBankSampah, dibuat.Entitas)
		assert.Equal(t, float64(0), dibuat.NominalSaldo, "rekening baru selalu mulai dari 0")
		if assert.NotNil(t, dibuat.LastUpdatedBy) {
			assert.Equal(t, dshAdmin, *dibuat.LastUpdatedBy, "audit diisi dari token, bukan body")
		}
	}
}

func TestCatatMutasi_BuatRekeningError(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("AdaNilaiRewardAktif", "101002001", 1).Return(true, nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, dshBank, 1).Return(nil, gorm.ErrRecordNotFound)
	repo.On("CreateRekening", mock.Anything, mock.Anything).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 1000), dshAkses())

	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	assert.Contains(t, err.Error(), "rekening bank")
}

func TestCatatMutasi_CariRekeningErrorLain(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("AdaNilaiRewardAktif", "101002001", 1).Return(true, nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, dshBank, 1).Return(nil, errors.New("db error"))
	expectTxRollback(dbMock)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 1000), dshAkses())

	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	repo.AssertNotCalled(t, "CreateRekening", mock.Anything, mock.Anything)
}

func TestCatatMutasi_DebitSaldoTidakCukup(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, dshBank, 1).Return(dshRekeningBank("1"+dshBank, 500), nil)
	expectTxRollback(dbMock)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("debit", "uang", 1000), dshAkses())

	assert.Equal(t, http.StatusBadRequest, dshKode(t, err))
	assert.Contains(t, err.Error(), "saldo tidak mencukupi")
}

func TestCatatMutasi_DebitMengurangiSaldo(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	rekening := dshRekeningBank("1"+dshBank, 5000)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, dshBank, 1).Return(rekening, nil)
	expectTxMutasi(dbMock, true)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("debit", "uang", 2000), dshAkses())

	assert.NoError(t, err)
	assert.Equal(t, float64(3000), rekening.NominalSaldo, "5000 - 2000")
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestCatatMutasi_KreditMenambahSaldo(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	rekening := dshRekeningBank("1"+dshBank, 5000)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("AdaNilaiRewardAktif", "101002001", 1).Return(true, nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, dshBank, 1).Return(rekening, nil)
	expectTxMutasi(dbMock, true)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 2000), dshAkses())

	assert.NoError(t, err)
	assert.Equal(t, float64(7000), rekening.NominalSaldo, "5000 + 2000")
	if assert.NotNil(t, rekening.LastUpdatedBy) {
		assert.Equal(t, dshAdmin, *rekening.LastUpdatedBy)
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestCatatMutasi_DebitSamaDenganSaldoBoleh(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	rekening := dshRekeningBank("1"+dshBank, 2000)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, dshBank, 1).Return(rekening, nil)
	expectTxMutasi(dbMock, true)

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("debit", "uang", 2000), dshAkses())

	assert.NoError(t, err)
	assert.Equal(t, float64(0), rekening.NominalSaldo)
}

func TestCatatMutasi_SimpanMutasiGagal(t *testing.T) {
	dbMock, repo, svc := newDshSvc(t)
	repo.On("FindBank", dshBank).Return(dshBSUModel(), nil)
	repo.On("FindRewardByJenis", models.RewardEnumUang, models.SatuanRewardEnumRp).
		Return(dshReward(1, models.RewardEnumUang, models.SatuanRewardEnumRp), nil)
	repo.On("AdaNilaiRewardAktif", "101002001", 1).Return(true, nil)
	repo.On("FindAdminIDByUserBank", dshUser, dshBank).Return(dshAdmin, nil)
	repo.On("LockRekeningBank", mock.Anything, dshBank, 1).Return(dshRekeningBank("1"+dshBank, 5000), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectExec("UPDATE").WillReturnError(errors.New("db error"))
	dbMock.ExpectRollback()

	err := svc.CatatManualMutasiBank(dshBank, dshCatatInput("kredit", "uang", 2000), dshAkses())

	assert.Equal(t, http.StatusInternalServerError, dshKode(t, err))
	assert.Contains(t, err.Error(), "mencatat mutasi")
}
