package services_test

import (
	"errors"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func pgkBSI() *models.BankSampah {
	return &models.BankSampah{BankID: "BSI-001", NamaBank: "BSI Utama", JenisBank: models.BSI, IsActive: true}
}

func pgkBSU() *models.BankSampah {
	parentID := "BSI-001"
	return &models.BankSampah{BankID: "BSU-001", NamaBank: "BSU Mawar", JenisBank: models.BSU, IsActive: true, ParentBankID: &parentID}
}

func pgkBSM() *models.BankSampah {
	return &models.BankSampah{BankID: "BSM-001", NamaBank: "BSM Induk", JenisBank: models.BSM, IsActive: true}
}

func pgkAdmin() *models.Admin {
	bID := "BSI-001"
	return &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001"}
}

func pgkPengangkutan() *models.PengangkutanSampah {
	adminID := "ADM-001"
	return &models.PengangkutanSampah{
		PengangkutanID: "PGK-001",
		BSIID:          "BSI-001",
		BSUId:          "BSU-001",
		AdminBSIID:     &adminID,
	}
}

func pgkKatalog(sampahID string, reward models.RewardEnum) *models.KatalogSampah {
	return &models.KatalogSampah{
		SampahID: sampahID,
		Sarok:    models.Sampah{NamaSampah: "Botol Plastik", Satuan: models.SatuanKG},
		Reward:   models.Reward{NamaReward: reward},
	}
}

func newPgkSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockPengangkutanRepo, *mocks.MockNotifSvc, services.PengangkutanService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockPengangkutanRepo)
	notif := new(mocks.MockNotifSvc)
	svc := services.NewPengangkutanService(gdb, repo, notif)
	return dbMock, repo, notif, svc
}

// ── GetJadwalHariIni ──────────────────────────────────────────────────────────

func TestGetJadwalHariIni_Error(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("GetJadwalHariIni", "BSI-001", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	result, err := svc.GetJadwalHariIni("BSI-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jadwal")
}

func TestGetJadwalHariIni_ReturnNil_Menghasilkan_EmptySlice(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("GetJadwalHariIni", "BSI-001", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, nil)

	result, err := svc.GetJadwalHariIni("BSI-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 0)
}

func TestGetJadwalHariIni_Sukses(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	rows := []repositories.ListJadwalRow{
		{BsuID: "BSU-001", NamaBsu: "BSU Mawar", JadwalID: "JDW-001"},
		{BsuID: "BSU-002", NamaBsu: "BSU Melati", JadwalID: "JDW-002"},
	}
	repo.On("GetJadwalHariIni", "BSI-001", mock.Anything, mock.Anything, mock.Anything).
		Return(rows, nil)

	result, err := svc.GetJadwalHariIni("BSI-001")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "BSU-001", result[0].BsuID)
}

// ── StartSesi ─────────────────────────────────────────────────────────────────

func TestStartSesi_BSITidakAktif(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBankAktif", "BSI-XXX").Return(nil, errors.New("not found"))

	result, err := svc.StartSesi(services.StartSesiReq{BSIID: "BSI-XXX", BSUID: "BSU-001"})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BSI")
}

func TestStartSesi_BSUTidakAktif(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBankAktif", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBSUAktifByParent", "BSU-XXX", "BSI-001").Return(nil, errors.New("not found"))

	result, err := svc.StartSesi(services.StartSesiReq{BSIID: "BSI-001", BSUID: "BSU-XXX"})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BSU")
}

func TestStartSesi_JadwalTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBankAktif", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBSUAktifByParent", "BSU-001", "BSI-001").Return(pgkBSU(), nil)
	repo.On("FindJadwalStart", "BSI-001", "BSU-001", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("not found"))

	result, err := svc.StartSesi(services.StartSesiReq{BSIID: "BSI-001", BSUID: "BSU-001"})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Jadwal")
}

func TestStartSesi_JadwalIDTidakCocok(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	jadwalUUID := uuid.New()
	jadwal := &models.Jadwal{JadwalID: jadwalUUID}

	repo.On("FindBankAktif", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBSUAktifByParent", "BSU-001", "BSI-001").Return(pgkBSU(), nil)
	repo.On("FindJadwalStart", "BSI-001", "BSU-001", mock.Anything, mock.Anything, mock.Anything).
		Return(jadwal, nil)

	result, err := svc.StartSesi(services.StartSesiReq{
		BSIID:    "BSI-001",
		BSUID:    "BSU-001",
		JadwalID: uuid.New().String(), // beda dengan jadwal yang dikembalikan
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jadwal_id")
}

func TestStartSesi_Sukses_StatusOTW(t *testing.T) {
	dbMock, repo, _, svc := newPgkSvc(t)

	jadwalUUID := uuid.New()
	jadwal := &models.Jadwal{JadwalID: jadwalUUID}

	repo.On("FindBankAktif", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBSUAktifByParent", "BSU-001", "BSI-001").Return(pgkBSU(), nil)
	repo.On("FindJadwalStart", "BSI-001", "BSU-001", mock.Anything, mock.Anything, mock.Anything).
		Return(jadwal, nil)
	repo.On("CreatePengangkutan", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	result, err := svc.StartSesi(services.StartSesiReq{
		BSIID:      "BSI-001",
		BSUID:      "BSU-001",
		AdminBSIID: "ADM-001",
		IsMandiri:  false,
		JadwalID:   jadwalUUID.String(),
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.PengangkutanID)
	assert.Equal(t, "BSI-001", result.BSIID)
	assert.Equal(t, "BSU-001", result.BSUId)
	assert.False(t, result.IsMandiri)
	repo.AssertExpectations(t)
}

func TestStartSesi_Sukses_Mandiri(t *testing.T) {
	dbMock, repo, _, svc := newPgkSvc(t)

	jadwalUUID := uuid.New()
	jadwal := &models.Jadwal{JadwalID: jadwalUUID}

	repo.On("FindBankAktif", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBSUAktifByParent", "BSU-001", "BSI-001").Return(pgkBSU(), nil)
	repo.On("FindJadwalStart", "BSI-001", "BSU-001", mock.Anything, mock.Anything, mock.Anything).
		Return(jadwal, nil)
	repo.On("CreatePengangkutan", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	result, err := svc.StartSesi(services.StartSesiReq{
		BSIID:      "BSI-001",
		BSUID:      "BSU-001",
		AdminBSIID: "ADM-001",
		IsMandiri:  true,
		JadwalID:   jadwalUUID.String(),
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsMandiri)
}

// ── GetList ───────────────────────────────────────────────────────────────────

func TestGetList_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBankAktif", "BANK-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetList("BANK-XXX", "", "")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bank")
}

func TestGetList_BSU_Sukses(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	rows := []repositories.PengangkutanListRow{
		{PengangkutanID: "PGK-001", BSIID: "BSI-001", BSUID: "BSU-001"},
	}
	repo.On("FindBankAktif", "BSU-001").Return(pgkBSU(), nil)
	repo.On("GetListByBSU", "BSU-001", "2024-01-01", "2024-01-31").Return(rows, nil)

	result, err := svc.GetList("BSU-001", "2024-01-01", "2024-01-31")

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "PGK-001", result[0].PengangkutanID)
}

func TestGetList_BSI_Sukses(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	rows := []repositories.PengangkutanListRow{
		{PengangkutanID: "PGK-001"},
		{PengangkutanID: "PGK-002"},
	}
	repo.On("FindBankAktif", "BSI-001").Return(pgkBSI(), nil)
	repo.On("GetListByBSI", "BSI-001", "", "").Return(rows, nil)

	result, err := svc.GetList("BSI-001", "", "")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
}

func TestGetList_JenisBankTidakDiizinkan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBankAktif", "BSM-001").Return(pgkBSM(), nil)

	result, err := svc.GetList("BSM-001", "", "")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "role bank")
}

// ── UpdateStatus ──────────────────────────────────────────────────────────────

func TestUpdateStatus_PengangkutanTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-XXX").Return(nil, errors.New("not found"))

	result, err := svc.UpdateStatus("PGK-XXX", "ADM-001", services.UpdateStatusReq{
		CurrentStatus: models.StatusOTW,
		NewStatus:     models.StatusArrived,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pengangkutan")
}

func TestUpdateStatus_AdminTidakBerhak(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindAdminByBank", "ADM-XXX", "BSI-001").Return(nil, errors.New("not found"))

	result, err := svc.UpdateStatus("PGK-001", "ADM-XXX", services.UpdateStatusReq{
		CurrentStatus: models.StatusOTW,
		NewStatus:     models.StatusArrived,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "akses")
}

func TestUpdateStatus_RiwayatGagal(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindAdminByBank", "ADM-001", "BSI-001").Return(pgkAdmin(), nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(nil, errors.New("db error"))

	result, err := svc.UpdateStatus("PGK-001", "ADM-001", services.UpdateStatusReq{
		CurrentStatus: models.StatusOTW,
		NewStatus:     models.StatusArrived,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "riwayat")
}

func TestUpdateStatus_StatusConflict(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	riwayat := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusArrived}

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindAdminByBank", "ADM-001", "BSI-001").Return(pgkAdmin(), nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayat, nil)

	result, err := svc.UpdateStatus("PGK-001", "ADM-001", services.UpdateStatusReq{
		CurrentStatus: models.StatusOTW, // mismatch dengan db (Arrived)
		NewStatus:     models.StatusArrived,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "berubah")
}

func TestUpdateStatus_TransisiTidakValid(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	riwayat := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusOTW}

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindAdminByBank", "ADM-001", "BSI-001").Return(pgkAdmin(), nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayat, nil)

	// OTW → Completed bukan transisi yang valid
	result, err := svc.UpdateStatus("PGK-001", "ADM-001", services.UpdateStatusReq{
		CurrentStatus: models.StatusOTW,
		NewStatus:     models.StatusCompleted,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tidak diizinkan")
}

func TestUpdateStatus_RejectedTanpaNotes(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	riwayat := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusRequested}

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindAdminByBank", "ADM-001", "BSI-001").Return(pgkAdmin(), nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayat, nil)

	result, err := svc.UpdateStatus("PGK-001", "ADM-001", services.UpdateStatusReq{
		CurrentStatus: models.StatusRequested,
		NewStatus:     models.StatusRejected,
		Notes:         "", // kosong
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Catatan")
}

func TestUpdateStatus_CanceledTanpaNotes(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	riwayat := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusOTW}

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindAdminByBank", "ADM-001", "BSI-001").Return(pgkAdmin(), nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayat, nil)

	result, err := svc.UpdateStatus("PGK-001", "ADM-001", services.UpdateStatusReq{
		CurrentStatus: models.StatusOTW,
		NewStatus:     models.StatusCanceled,
		Notes:         "",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Catatan")
}

func TestUpdateStatus_Sukses(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	riwayat := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusOTW}

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindAdminByBank", "ADM-001", "BSI-001").Return(pgkAdmin(), nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayat, nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)

	result, err := svc.UpdateStatus("PGK-001", "ADM-001", services.UpdateStatusReq{
		CurrentStatus: models.StatusOTW,
		NewStatus:     models.StatusArrived,
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, models.StatusArrived, result.StatusPengangkutan)
	assert.Equal(t, "ADM-001", result.ChangedBy)
}

// ── RequestByBSU ──────────────────────────────────────────────────────────────

func TestRequestByBSU_FormatTanggalTidakValid(t *testing.T) {
	_, _, _, svc := newPgkSvc(t)

	result, err := svc.RequestByBSU("BSU-001", "ADM-BSU-001", services.RequestBSUReq{
		Tanggal:  "bukan-tanggal",
		JamMulai: "09:00",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tanggal")
}

func TestRequestByBSU_BSUTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSU-XXX").Return(nil, errors.New("not found"))

	result, err := svc.RequestByBSU("BSU-XXX", "ADM-BSU-001", services.RequestBSUReq{
		Tanggal:  "2024-06-15",
		JamMulai: "09:00",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BSU")
}

func TestRequestByBSU_BSUTidakAktif(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	bsuTidakAktif := pgkBSU()
	bsuTidakAktif.IsActive = false
	repo.On("FindBank", "BSU-001").Return(bsuTidakAktif, nil)

	result, err := svc.RequestByBSU("BSU-001", "ADM-BSU-001", services.RequestBSUReq{
		Tanggal:  "2024-06-15",
		JamMulai: "09:00",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BSU")
}

func TestRequestByBSU_BSUTidakPunyaBSI(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	bsuTanpaParent := &models.BankSampah{
		BankID:       "BSU-001",
		JenisBank:    models.BSU,
		IsActive:     true,
		ParentBankID: nil,
	}
	repo.On("FindBank", "BSU-001").Return(bsuTanpaParent, nil)

	result, err := svc.RequestByBSU("BSU-001", "ADM-BSU-001", services.RequestBSUReq{
		Tanggal:  "2024-06-15",
		JamMulai: "09:00",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BSI induk")
}

func TestRequestByBSU_AdminTidakValid(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindAdminByBank", "ADM-XXX", "BSU-001").Return(nil, errors.New("not found"))

	result, err := svc.RequestByBSU("BSU-001", "ADM-XXX", services.RequestBSUReq{
		Tanggal:  "2024-06-15",
		JamMulai: "09:00",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Admin BSU")
}

func TestRequestByBSU_FormatJamTidakValid(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindAdminByBank", "ADM-BSU-001", "BSU-001").Return(pgkAdmin(), nil)

	result, err := svc.RequestByBSU("BSU-001", "ADM-BSU-001", services.RequestBSUReq{
		Tanggal:  "2024-06-15",
		JamMulai: "jam-tidak-valid",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "jam_mulai")
}

func TestRequestByBSU_KonflikJadwal(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindAdminByBank", "ADM-BSU-001", "BSU-001").Return(pgkAdmin(), nil)
	repo.On("CheckJadwalConflict", "BSI-001", "BSU-001", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(true, nil)

	result, err := svc.RequestByBSU("BSU-001", "ADM-BSU-001", services.RequestBSUReq{
		Tanggal:  "2024-06-15",
		JamMulai: "09:00",
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bertabrakan")
}

func TestRequestByBSU_Sukses(t *testing.T) {
	dbMock, repo, notif, svc := newPgkSvc(t)

	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindAdminByBank", "ADM-BSU-001", "BSU-001").Return(pgkAdmin(), nil)
	repo.On("CheckJadwalConflict", "BSI-001", "BSU-001", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(false, nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreatePengangkutan", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateRiwayat", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindAdminsByBank", "BSI-001").Return([]repositories.PengangkutanAdminUser{}, nil).Maybe()
	notif.On("NotifRequestPengangkutan", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	result, err := svc.RequestByBSU("BSU-001", "ADM-BSU-001", services.RequestBSUReq{
		Tanggal:  "2024-06-15",
		JamMulai: "09:00",
		Notes:    "Request pickup",
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.PengangkutanID)
	assert.Equal(t, "BSI-001", result.BSIID)
	assert.Equal(t, "BSU-001", result.BSUId)
}

// ── Preview ───────────────────────────────────────────────────────────────────

func TestPreview_PengangkutanTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-XXX").Return(nil, errors.New("not found"))

	result, err := svc.Preview("PGK-XXX", []services.ItemPengangkutanReq{{SampahID: "SMP-001", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pengangkutan")
}

func TestPreview_QtyNol(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)

	result, err := svc.Preview("PGK-001", []services.ItemPengangkutanReq{{SampahID: "SMP-001", Qty: 0}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Qty")
}

func TestPreview_QtyNegatif(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)

	result, err := svc.Preview("PGK-001", []services.ItemPengangkutanReq{{SampahID: "SMP-001", Qty: -1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Qty")
}

func TestPreview_SampahTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindKatalogWithAll", "SMP-XXX").Return(nil, errors.New("not found"))

	result, err := svc.Preview("PGK-001", []services.ItemPengangkutanReq{{SampahID: "SMP-XXX", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Sampah tidak ditemukan")
}

func TestPreview_Sukses_CukupStok(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	katalog := pgkKatalog("SMP-001", models.RewardEnumUang)
	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindKatalogWithAll", "SMP-001").Return(katalog, nil)
	repo.On("FindStok", "BSU-001", "SMP-001").Return(float64(10), nil)
	repo.On("FindStok", "BSI-001", "SMP-001").Return(float64(5), nil)

	result, err := svc.Preview("PGK-001", []services.ItemPengangkutanReq{{SampahID: "SMP-001", Qty: 3}})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "PGK-001", result.PengangkutanID)
	assert.Equal(t, "BSI Utama", result.NamaBSI)
	assert.Equal(t, "BSU Mawar", result.NamaBSU)
	assert.Equal(t, 1, result.TotalItem)
	assert.False(t, result.AdaStokKurang)
	assert.Len(t, result.Items, 1)
	assert.True(t, result.Items[0].CukupUntukKirim)
	assert.Equal(t, float64(10), result.Items[0].StokBSUSebelum)
	assert.Equal(t, float64(7), result.Items[0].StokBSUSetelah)
	assert.Equal(t, float64(5), result.Items[0].StokBSISebelum)
	assert.Equal(t, float64(8), result.Items[0].StokBSISetelah)
}

func TestPreview_Sukses_StokKurang(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	katalog := pgkKatalog("SMP-001", models.RewardEnumUang)
	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindKatalogWithAll", "SMP-001").Return(katalog, nil)
	repo.On("FindStok", "BSU-001", "SMP-001").Return(float64(1), nil)
	repo.On("FindStok", "BSI-001", "SMP-001").Return(float64(0), nil)

	result, err := svc.Preview("PGK-001", []services.ItemPengangkutanReq{{SampahID: "SMP-001", Qty: 5}})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.AdaStokKurang)
	assert.False(t, result.Items[0].CukupUntukKirim)
}

func TestPreview_Sukses_MultipleItem(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	katalog1 := pgkKatalog("SMP-001", models.RewardEnumUang)
	katalog2 := pgkKatalog("SMP-002", models.RewardEnumSembako)

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindKatalogWithAll", "SMP-001").Return(katalog1, nil)
	repo.On("FindKatalogWithAll", "SMP-002").Return(katalog2, nil)
	repo.On("FindStok", "BSU-001", "SMP-001").Return(float64(10), nil)
	repo.On("FindStok", "BSI-001", "SMP-001").Return(float64(2), nil)
	repo.On("FindStok", "BSU-001", "SMP-002").Return(float64(5), nil)
	repo.On("FindStok", "BSI-001", "SMP-002").Return(float64(1), nil)

	result, err := svc.Preview("PGK-001", []services.ItemPengangkutanReq{
		{SampahID: "SMP-001", Qty: 2},
		{SampahID: "SMP-002", Qty: 3},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalItem)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, string(models.RewardEnumSembako), result.Items[1].NamaReward)
}

// ── GetDetailSampah ───────────────────────────────────────────────────────────

func TestGetDetailSampah_HeaderTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("GetDetailHeader", "PGK-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetDetailSampah("PGK-XXX")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pengangkutan")
}

func TestGetDetailSampah_ItemsGagal(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	header := &repositories.PengangkutanDetailHeader{PengangkutanID: "PGK-001", NamaBSI: "BSI Utama"}
	repo.On("GetDetailHeader", "PGK-001").Return(header, nil)
	repo.On("GetDetailItems", "PGK-001").Return(nil, errors.New("db error"))

	result, err := svc.GetDetailSampah("PGK-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "detail sampah")
}

func TestGetDetailSampah_Sukses(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	header := &repositories.PengangkutanDetailHeader{
		PengangkutanID: "PGK-001",
		NamaBSI:        "BSI Utama",
		NamaBSU:        "BSU Mawar",
		TotalItem:      2,
	}
	items := []repositories.PengangkutanDetailItem{
		{SampahID: "SMP-001", NamaSampah: "Botol Plastik", Satuan: "kg", Qty: 3},
		{SampahID: "SMP-002", NamaSampah: "Kertas", Satuan: "kg", Qty: 1.5},
	}
	repo.On("GetDetailHeader", "PGK-001").Return(header, nil)
	repo.On("GetDetailItems", "PGK-001").Return(items, nil)

	result, err := svc.GetDetailSampah("PGK-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "PGK-001", result.Header.PengangkutanID)
	assert.Equal(t, "BSI Utama", result.Header.NamaBSI)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "Botol Plastik", result.Items[0].NamaSampah)
}

func TestGetDetailSampah_ItemsNil_Menghasilkan_EmptySlice(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	header := &repositories.PengangkutanDetailHeader{PengangkutanID: "PGK-001"}
	repo.On("GetDetailHeader", "PGK-001").Return(header, nil)
	repo.On("GetDetailItems", "PGK-001").Return(nil, nil)

	result, err := svc.GetDetailSampah("PGK-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Items, 0)
}

// ── GetSampahList ─────────────────────────────────────────────────────────────

func TestGetSampahList_Error(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("GetSampahList", "BSI-001", "BSU-001").Return(nil, errors.New("db error"))

	result, err := svc.GetSampahList("BSI-001", "BSU-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sampah")
}

func TestGetSampahList_Sukses(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	rows := []repositories.SampahPengangkutanRow{
		{SampahID: "SMP-001", NamaSampah: "Botol Plastik", Stok: 10},
		{SampahID: "SMP-002", NamaSampah: "Kertas", Stok: 5},
	}
	repo.On("GetSampahList", "BSI-001", "BSU-001").Return(rows, nil)

	result, err := svc.GetSampahList("BSI-001", "BSU-001")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "SMP-001", result[0].SampahID)
}

// ── CheckSesiAktif ────────────────────────────────────────────────────────────

func TestCheckSesiAktif_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSU-XXX").Return(nil, errors.New("not found"))

	result, err := svc.CheckSesiAktif("BSU-XXX")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Bank")
}

func TestCheckSesiAktif_BukanBSU(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)

	result, err := svc.CheckSesiAktif("BSI-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bukan BSU")
}

func TestCheckSesiAktif_BSUTidakPunyaBSI(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	bsuTanpaParent := &models.BankSampah{BankID: "BSU-001", JenisBank: models.BSU, ParentBankID: nil}
	repo.On("FindBank", "BSU-001").Return(bsuTanpaParent, nil)

	result, err := svc.CheckSesiAktif("BSU-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BSI induk")
}

func TestCheckSesiAktif_TidakAdaPengangkutan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindLatestPengangkutan", "BSU-001", "BSI-001").Return(nil, errors.New("not found"))

	result, err := svc.CheckSesiAktif("BSU-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsActive)
	assert.Equal(t, "BSI-001", result.BSIID)
	assert.Equal(t, "BSI Utama", result.NamaBSI)
}

func TestCheckSesiAktif_StatusTerminal_IsActiveFalse(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	riwayat := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusCompleted}

	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindLatestPengangkutan", "BSU-001", "BSI-001").Return(pgkPengangkutan(), nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayat, nil)

	result, err := svc.CheckSesiAktif("BSU-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.IsActive)
	assert.Equal(t, string(models.StatusCompleted), result.StatusTerkini)
}

func TestCheckSesiAktif_StatusAktif(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	riwayat := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusOTW}

	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindLatestPengangkutan", "BSU-001", "BSI-001").Return(pgkPengangkutan(), nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayat, nil)

	result, err := svc.CheckSesiAktif("BSU-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.IsActive)
	assert.Equal(t, "PGK-001", result.PengangkutanID)
	assert.Equal(t, string(models.StatusOTW), result.StatusTerkini)
}

// ── GetDetailSesiAktif ────────────────────────────────────────────────────────

func TestGetDetailSesiAktif_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetDetailSesiAktif("PGK-XXX")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pengangkutan")
}

func TestGetDetailSesiAktif_RiwayatGagal(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("GetRiwayatDetail", "PGK-001").Return(nil, errors.New("db error"))

	result, err := svc.GetDetailSesiAktif("PGK-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "riwayat")
}

func TestGetDetailSesiAktif_Sukses(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	now := time.Now()
	riwayatRows := []repositories.RiwayatDetailRow{
		{Status: models.StatusOTW, ChangedAt: now, ChangedBy: "ADM-001", NamaPetugas: "Admin Utama", Notes: ""},
		{Status: models.StatusRequested, ChangedAt: now.Add(-time.Hour), ChangedBy: "ADM-BSU-001", NamaPetugas: "", Notes: "Request pickup"},
	}

	repo.On("FindPengangkutan", "PGK-001").Return(pgkPengangkutan(), nil)
	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)
	repo.On("GetRiwayatDetail", "PGK-001").Return(riwayatRows, nil)

	result, err := svc.GetDetailSesiAktif("PGK-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "PGK-001", result.PengangkutanID)
	assert.Equal(t, "BSI Utama", result.NamaBSI)
	assert.Equal(t, "BSU Mawar", result.NamaBSU)
	assert.Equal(t, models.StatusOTW, result.StatusTerkini)
	assert.Len(t, result.Riwayat, 2)
	assert.Equal(t, "Admin Utama", result.Riwayat[0].ChangedBy)
	assert.Equal(t, "ADM-BSU-001", result.Riwayat[1].ChangedBy) // fallback ke ChangedBy jika NamaPetugas kosong
}

// ── GetAllAktif ───────────────────────────────────────────────────────────────

func TestGetAllAktif_BSITidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSI-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetAllAktif("BSI-XXX", "ADM-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BSI")
}

func TestGetAllAktif_BukanBSI(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSU-001").Return(pgkBSU(), nil)

	result, err := svc.GetAllAktif("BSU-001", "ADM-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bukan BSI")
}

func TestGetAllAktif_BSUListKosong(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBSUList", "BSI-001").Return([]models.BankSampah{}, nil)

	result, err := svc.GetAllAktif("BSI-001", "ADM-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "BSI-001", result.BSIID)
	assert.Equal(t, "BSI Utama", result.NamaBSI)
	assert.Len(t, result.Data, 0)
}

func TestGetAllAktif_SkipBSUTerminal(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	bsuList := []models.BankSampah{
		{BankID: "BSU-001", NamaBank: "BSU Mawar"},
		{BankID: "BSU-002", NamaBank: "BSU Melati"},
	}
	pgk1 := &models.PengangkutanSampah{PengangkutanID: "PGK-001", BSIID: "BSI-001", BSUId: "BSU-001"}
	pgk2 := &models.PengangkutanSampah{PengangkutanID: "PGK-002", BSIID: "BSI-001", BSUId: "BSU-002"}

	riwayatAktif := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusOTW}
	riwayatTerminal := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusCompleted}

	jadwal := &models.Jadwal{JamMulai: "09:00", JamSelesai: "11:00"}

	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBSUList", "BSI-001").Return(bsuList, nil)
	repo.On("FindLatestPengangkutan", "BSU-001", "BSI-001").Return(pgk1, nil)
	repo.On("FindLatestPengangkutan", "BSU-002", "BSI-001").Return(pgk2, nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayatAktif, nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-002").Return(riwayatTerminal, nil)
	repo.On("FindJadwal", mock.Anything).Return(jadwal, nil)

	result, err := svc.GetAllAktif("BSI-001", "ADM-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Data, 1) // hanya BSU-001 yang aktif
	assert.Equal(t, "PGK-001", result.Data[0].PengangkutanID)
	assert.Equal(t, models.StatusOTW, result.Data[0].StatusTerkini)
	assert.Equal(t, "09:00", result.Data[0].JamMulai)
}

func TestGetAllAktif_IsActionAllowed(t *testing.T) {
	_, repo, _, svc := newPgkSvc(t)

	bsuList := []models.BankSampah{{BankID: "BSU-001", NamaBank: "BSU Mawar"}}

	adminLain := "ADM-002"
	pgk := &models.PengangkutanSampah{
		PengangkutanID: "PGK-001",
		BSIID:          "BSI-001",
		BSUId:          "BSU-001",
		AdminBSIID:     &adminLain, // bukan admin yang request
	}
	riwayat := &models.RiwayatPengangkutan{StatusPengangkutan: models.StatusOTW}
	jadwal := &models.Jadwal{JamMulai: "09:00", JamSelesai: "11:00"}

	repo.On("FindBank", "BSI-001").Return(pgkBSI(), nil)
	repo.On("FindBSUList", "BSI-001").Return(bsuList, nil)
	repo.On("FindLatestPengangkutan", "BSU-001", "BSI-001").Return(pgk, nil)
	repo.On("FindLastRiwayat", mock.Anything, "PGK-001").Return(riwayat, nil)
	repo.On("FindJadwal", mock.Anything).Return(jadwal, nil)

	result, err := svc.GetAllAktif("BSI-001", "ADM-001") // ADM-001 beda dari ADM-002

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Data, 1)
	assert.False(t, result.Data[0].IsActionAllowed)
}
