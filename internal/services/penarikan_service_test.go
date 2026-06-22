package services_test

import (
	"errors"
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

// ── helpers ───────────────────────────────────────────────────────────────────

func pBankID(s string) *string   { return &s }
func pNasabahID(s string) *string { return &s }
func pRewardID(i int) *int        { return &i }

func penarikanNasabah() *models.Nasabah {
	return &models.Nasabah{
		NasabahID:     "NSB-001",
		BankID:        "BSI-001",
		UserID:        "USR-NSB-001",
		StatusNasabah: models.Aktif,
	}
}

func penarikanRewardUang() *models.Reward {
	return &models.Reward{
		RewardID:   1,
		NamaReward: models.RewardEnumUang,
		Satuan:     string(models.SatuanRewardEnumRp),
	}
}

func penarikanRewardSembako() *models.Reward {
	return &models.Reward{
		RewardID:   2,
		NamaReward: models.RewardEnumSembako,
		Satuan:     string(models.SatuanRewardEnumPoin),
	}
}

func penarikanSaldo(nominal float64) *models.SaldoRekening {
	return &models.SaldoRekening{
		RekeningID:         "RK-001",
		NominalSaldo:       nominal,
		SatuanNominalSaldo: models.SatuanRewardEnumRp,
	}
}

// newPenarikanSvc creates a test service wired to a sqlmock DB and mock repo/notif.
func newPenarikanSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockPenarikanRepo, *mocks.MockNotifSvc, services.PenarikanService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockPenarikanRepo)
	notif := new(mocks.MockNotifSvc)
	svc := services.NewPenarikanService(gdb, repo, notif)
	return dbMock, repo, notif, svc
}

// ── FormatPenarikanNilai ──────────────────────────────────────────────────────

func TestFormatPenarikanNilai_Rp(t *testing.T) {
	assert.Equal(t, "Rp50000", services.FormatPenarikanNilai(50000, models.SatuanRewardEnumRp))
}

func TestFormatPenarikanNilai_Poin(t *testing.T) {
	assert.Equal(t, "150 poin", services.FormatPenarikanNilai(150, models.SatuanRewardEnumPoin))
}

func TestFormatPenarikanNilai_Default(t *testing.T) {
	assert.Equal(t, "3.50 kg", services.FormatPenarikanNilai(3.5, "kg"))
}

// ── Preview ───────────────────────────────────────────────────────────────────

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

func TestPreviewPenarikan_ValidasiSembakoTanpaItem(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindReward", 2).Return(penarikanRewardSembako(), nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 2})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Item sembako")
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

	// nominal > saldo → bukan error, hanya SaldoCukup: false
	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 100000})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.SaldoCukup)
	assert.Equal(t, float64(-50000), result.SaldoSetelah)
}

func TestPreviewPenarikan_Sembako_StokKurang(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	katalog := &models.KatalogSembako{
		SembakoID:     "SMB-001",
		NilaiPoin:     50,
		MasterSembako: models.Sembako{NamaBarang: "Beras"},
	}
	stok := &models.StokSembako{SembakoID: "SMB-001", BankID: "BSI-001", Stok: 1}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindReward", 2).Return(penarikanRewardSembako(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 2).Return(
		&models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 500, SatuanNominalSaldo: models.SatuanRewardEnumPoin}, nil,
	)
	repo.On("FindSembako", "SMB-001", "BSI-001").Return(katalog, nil)
	repo.On("FindStokSembako", "SMB-001", "BSI-001").Return(stok, nil)

	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{
		RewardID:    2,
		ItemSembako: []services.SembakoItemReq{{SembakoID: "SMB-001", Qty: 5}},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Stok")
}

func TestPreviewPenarikan_Sembako_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	katalog1 := &models.KatalogSembako{SembakoID: "SMB-001", NilaiPoin: 50, MasterSembako: models.Sembako{NamaBarang: "Beras"}}
	katalog2 := &models.KatalogSembako{SembakoID: "SMB-002", NilaiPoin: 30, MasterSembako: models.Sembako{NamaBarang: "Minyak"}}
	stok1 := &models.StokSembako{SembakoID: "SMB-001", Stok: 10}
	stok2 := &models.StokSembako{SembakoID: "SMB-002", Stok: 10}
	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 200, SatuanNominalSaldo: models.SatuanRewardEnumPoin}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindReward", 2).Return(penarikanRewardSembako(), nil)
	repo.On("FindSaldoRekening", "NSB-001", 2).Return(saldo, nil)
	repo.On("FindSembako", "SMB-001", "BSI-001").Return(katalog1, nil)
	repo.On("FindSembako", "SMB-002", "BSI-001").Return(katalog2, nil)
	repo.On("FindStokSembako", "SMB-001", "BSI-001").Return(stok1, nil)
	repo.On("FindStokSembako", "SMB-002", "BSI-001").Return(stok2, nil)

	// totalPoin = 2*50 + 3*30 = 190
	result, err := svc.Preview("NSB-001", services.AjukanPenarikanReq{
		RewardID: 2,
		ItemSembako: []services.SembakoItemReq{
			{SembakoID: "SMB-001", Qty: 2},
			{SembakoID: "SMB-002", Qty: 3},
		},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(190), result.NominalPenarikan)
	assert.True(t, result.SaldoCukup)
	assert.Equal(t, float64(10), result.SaldoSetelah)
	assert.Len(t, result.ItemSembako, 2)
	assert.Equal(t, float64(100), result.ItemSembako[0].SubtotalPoin) // 2 * 50
	assert.Equal(t, float64(90), result.ItemSembako[1].SubtotalPoin)  // 3 * 30
}

// ── Ajukan ────────────────────────────────────────────────────────────────────

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

func TestAjukanPenarikan_SaldoKurangDalamTx(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	saldo := penarikanSaldo(50000)

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindReward", 1).Return(penarikanRewardUang(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(saldo, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 100000})

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

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{RewardID: 1, NominalPenarikan: 100000})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.PenarikanID)
	assert.Equal(t, float64(100000), result.Nominal)
	assert.Equal(t, models.StatusPenarikanPending, result.Status)
	repo.AssertExpectations(t)
}

func TestAjukanPenarikan_Sembako_StokKurangDalamTx(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 500, SatuanNominalSaldo: models.SatuanRewardEnumPoin}
	katalog := &models.KatalogSembako{SembakoID: "SMB-001", NilaiPoin: 50, MasterSembako: models.Sembako{NamaBarang: "Beras"}}
	stok := &models.StokSembako{SembakoID: "SMB-001", Stok: 1}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindReward", 2).Return(penarikanRewardSembako(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldo, nil)
	repo.On("LockSembako", mock.Anything, "SMB-001", "BSI-001").Return(katalog, nil)
	repo.On("LockStok", mock.Anything, "SMB-001", "BSI-001").Return(stok, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID:    2,
		ItemSembako: []services.SembakoItemReq{{SembakoID: "SMB-001", Qty: 5}},
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Stok")
}

func TestAjukanPenarikan_Sukses_Sembako(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 500, SatuanNominalSaldo: models.SatuanRewardEnumPoin}
	katalog := &models.KatalogSembako{SembakoID: "SMB-001", NilaiPoin: 50, MasterSembako: models.Sembako{NamaBarang: "Beras"}}
	stok := &models.StokSembako{SembakoID: "SMB-001", Stok: 10}

	repo.On("FindNasabahAktif", "NSB-001").Return(penarikanNasabah(), nil)
	repo.On("FindReward", 2).Return(penarikanRewardSembako(), nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldo, nil)
	repo.On("LockSembako", mock.Anything, "SMB-001", "BSI-001").Return(katalog, nil)
	repo.On("LockStok", mock.Anything, "SMB-001", "BSI-001").Return(stok, nil)
	// 2 * 50 = 100 → saldoSesudah = 500 - 100 = 400
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(400)).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("SavePenarikan", mock.Anything, mock.Anything).Return(nil)
	repo.On("SaveDetailSembako", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpdateStok", mock.Anything, "SMB-001", "BSI-001", float64(-2)).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	result, err := svc.Ajukan("NSB-001", services.AjukanPenarikanReq{
		RewardID:    2,
		ItemSembako: []services.SembakoItemReq{{SembakoID: "SMB-001", Qty: 2}},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, float64(100), result.Nominal) // 2 * 50
	assert.Equal(t, models.StatusPenarikanPending, result.Status)
	repo.AssertExpectations(t)
}

// ── Konfirmasi ────────────────────────────────────────────────────────────────

func TestKonfirmasiPenarikan_AdminTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindAdminTx", mock.Anything, "USR-ADM-001").Return(nil, errors.New("not found"))

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", "https://foto.jpg")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "admin")
}

func TestKonfirmasiPenarikan_PenarikanTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSI-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001"}

	repo.On("FindAdminTx", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-XXX").Return(nil, errors.New("not found"))

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Konfirmasi("RDM-XXX", "USR-ADM-001", "https://foto.jpg")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak ditemukan")
}

func TestKonfirmasiPenarikan_BankMismatch(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	adminBankID := "BSI-001"
	penarikanBankID := "BSI-002"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &adminBankID, UserID: "USR-ADM-001"}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID(penarikanBankID),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindAdminTx", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", "https://foto.jpg")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "akses")
}

func TestKonfirmasiPenarikan_ConflictOfInterest(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSI-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001"}
	// penarikan.NasabahID → nasabah.UserID == adminUserID
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSI-001"),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindAdminTx", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	// raw tx.Where("nasabah_id = ?", ...).First(&nasabah) → user_id == adminUserID
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"nasabah_id", "user_id", "bank_id", "joined_at", "nomor_rekening", "status_nasabah"}).
			AddRow("NSB-001", "USR-ADM-001", "BSI-001", time.Now(), "RK-001", "aktif"))
	dbMock.ExpectRollback()

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", "https://foto.jpg")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sendiri")
}

func TestKonfirmasiPenarikan_StatusBukanPending(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSI-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001"}
	// NasabahID = nil → conflict check dilewati
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       nil,
		BankID:          pBankID("BSI-001"),
		StatusPenarikan: models.StatusPenarikanBerhasil,
	}

	repo.On("FindAdminTx", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", "https://foto.jpg")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "berhasil")
}

func TestKonfirmasiPenarikan_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	bID := "BSI-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001"}
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSI-001"),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindAdminTx", mock.Anything, "USR-ADM-001").Return(admin, nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	// goroutine calls FindPenarikanByID asynchronously — mark Maybe so it doesn't panic
	repo.On("FindPenarikanByID", "RDM-001").Return(nil, errors.New("skip")).Maybe()

	// raw conflict check: user nasabah berbeda dari admin
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").
		WillReturnRows(sqlmock.NewRows([]string{"nasabah_id", "user_id", "bank_id", "joined_at", "nomor_rekening", "status_nasabah"}).
			AddRow("NSB-001", "USR-NASABAH-001", "BSI-001", time.Now(), "RK-001", "aktif"))
	dbMock.ExpectCommit()

	err := svc.Konfirmasi("RDM-001", "USR-ADM-001", "https://foto.jpg")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// ── Batal ─────────────────────────────────────────────────────────────────────

func TestBatalPenarikan_BukanNasabah(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahByUserID", "USR-ADM-001").Return(nil, errors.New("not found"))

	err := svc.Batal("RDM-001", "USR-ADM-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nasabah")
}

func TestBatalPenarikan_PenarikanTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-XXX").Return(nil, errors.New("not found"))

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Batal("RDM-XXX", "USR-NSB-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak ditemukan")
}

func TestBatalPenarikan_BukanMilikNasabah(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	// penarikan milik nasabah lain
	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-LAIN"),
		BankID:          pBankID("BSI-001"),
		StatusPenarikan: models.StatusPenarikanPending,
	}

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil) // nasabahID = NSB-001
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Batal("RDM-001", "USR-NSB-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "akses")
}

func TestBatalPenarikan_StatusBukanPending(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	p := &models.Penarikan{
		PenarikanID:     "RDM-001",
		NasabahID:       pNasabahID("NSB-001"),
		BankID:          pBankID("BSI-001"),
		StatusPenarikan: models.StatusPenarikanBerhasil,
	}

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	err := svc.Batal("RDM-001", "USR-NSB-001")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dibatalkan")
}

func TestBatalPenarikan_Sukses_Uang(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	rewardID := 1
	p := &models.Penarikan{
		PenarikanID:      "RDM-001",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           pBankID("BSI-001"),
		RewardID:         &rewardID,
		NominalPenarikan: 100000,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardUang(),
	}
	saldo := penarikanSaldo(200000)

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-001").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 1).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(300000)).Return(nil) // refund: 200000 + 100000
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	err := svc.Batal("RDM-001", "USR-NSB-001")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestBatalPenarikan_Sukses_Sembako_StokDikembalikan(t *testing.T) {
	dbMock, repo, _, svc := newPenarikanSvc(t)

	rewardID := 2
	p := &models.Penarikan{
		PenarikanID:      "RDM-002",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           pBankID("BSI-001"),
		RewardID:         &rewardID,
		NominalPenarikan: 150,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardSembako(),
	}
	saldo := &models.SaldoRekening{RekeningID: "RK-002", NominalSaldo: 100, SatuanNominalSaldo: models.SatuanRewardEnumPoin}
	details := []models.DetailPenarikanSembako{
		{SembakoID: "SMB-001", Qty: 3, NilaiPoin: 50, SubtotalPoin: 150},
	}

	repo.On("FindNasabahByUserID", "USR-NSB-001").Return(penarikanNasabah(), nil)
	repo.On("LockPenarikan", mock.Anything, "RDM-002").Return(p, nil)
	repo.On("UpdateStatus", mock.Anything, p, mock.Anything).Return(nil)
	repo.On("LockSaldo", mock.Anything, "NSB-001", 2).Return(saldo, nil)
	repo.On("UpdateSaldo", mock.Anything, saldo, float64(250)).Return(nil) // refund: 100 + 150
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindDetailSembako", mock.Anything, "RDM-002").Return(details, nil)
	repo.On("UpdateStok", mock.Anything, "SMB-001", "BSI-001", float64(3)).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	err := svc.Batal("RDM-002", "USR-NSB-001")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

// ── GetList ───────────────────────────────────────────────────────────────────

func TestGetListPenarikan_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	filter := repositories.PenarikanFilter{StartDate: "2026-06-01", EndDate: "2026-06-30"}
	repo.On("FindByNasabahID", "NSB-001", filter).Return([]models.Penarikan{}, nil)

	result, err := svc.GetList("NSB-001", filter)

	assert.NoError(t, err)
	assert.Empty(t, result.Data)
}

// ── GetListByBank ─────────────────────────────────────────────────────────────

func TestGetListByBankPenarikan_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	bID := "BSI-001"
	r := penarikanRewardUang()
	nas := penarikanNasabah()
	nas.User = models.User{UserID: "USR-NSB-001", Nama: "Budi"}
	bank := &models.BankSampah{BankID: "BSI-001", NamaBank: "Bank Satu"}
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
	repo.On("FindByBankID", "BSI-001", filter).Return([]models.Penarikan{p}, nil)

	result, err := svc.GetListByBank("BSI-001", filter)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "RDM-001", result[0].PenarikanID)
	assert.Equal(t, "Budi", result[0].NamaNasabah)
	assert.Equal(t, "Bank Satu", result[0].NamaBank)
}

// ── GetDetail ─────────────────────────────────────────────────────────────────

func TestGetDetailPenarikan_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	repo.On("FindPenarikanByID", "RDM-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetDetail("RDM-XXX", "USR-001", models.SuperAdmin)

	assert.Nil(t, result)
	assert.Error(t, err)
}

func TestGetDetailPenarikan_SuperAdmin_Sukses(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	bID := "BSI-001"
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

	// SuperAdmin: tidak ada raw DB call (tidak perlu akses check)
	result, err := svc.GetDetail("RDM-001", "USR-SUPER-001", models.SuperAdmin)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "RDM-001", result.PenarikanID)
	assert.Equal(t, float64(75000), result.NominalPenarikan)
	assert.Equal(t, string(models.RewardEnumUang), result.NamaReward)
}

func TestGetDetailPenarikan_SuperAdmin_DenganDetailSembako(t *testing.T) {
	_, repo, _, svc := newPenarikanSvc(t)

	bID := "BSI-001"
	katalog := &models.KatalogSembako{
		SembakoID:     "SMB-001",
		NilaiPoin:     50,
		MasterSembako: models.Sembako{NamaBarang: "Beras"},
	}
	p := &models.Penarikan{
		PenarikanID:      "RDM-002",
		NasabahID:        pNasabahID("NSB-001"),
		BankID:           &bID,
		RewardID:         pRewardID(2),
		NominalPenarikan: 100,
		SatuanPenarikan:  models.SatuanRewardEnumPoin,
		StatusPenarikan:  models.StatusPenarikanPending,
		Reward:           penarikanRewardSembako(),
		DetailPenarikanSembako: []models.DetailPenarikanSembako{
			{SembakoID: "SMB-001", Qty: 2, NilaiPoin: 50, SubtotalPoin: 100, KatalogSembako: katalog},
		},
	}

	repo.On("FindPenarikanByID", "RDM-002").Return(p, nil)

	result, err := svc.GetDetail("RDM-002", "USR-SUPER-001", models.SuperAdmin)

	assert.NoError(t, err)
	assert.Len(t, result.DetailSembako, 1)
	assert.Equal(t, "Beras", result.DetailSembako[0].NamaSembako)
	assert.Equal(t, float64(100), result.DetailSembako[0].SubtotalPoin)
}
