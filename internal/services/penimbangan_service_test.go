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
	"enviroo-be/internal/utils"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func pnbBool(b bool) *bool { return &b }

func pnbBank() *models.BankSampah {
	return &models.BankSampah{BankID: "BSM-001", NamaBank: "BSM Mawar", JenisBank: models.BSM, IsActive: true}
}

func pnbTanggalOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func pnbHariIni() time.Time { return pnbTanggalOnly(time.Now()) }
func pnbBesok() time.Time   { return pnbTanggalOnly(time.Now().AddDate(0, 0, 1)) }
func pnbKemarin() time.Time { return pnbTanggalOnly(time.Now().AddDate(0, 0, -1)) }
func pnbStrHariIni() string { return time.Now().Format("2006-01-02") }

func pnbTanggalParsed(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func pnbSesi(status models.StatusPenimbangan) *models.Penimbangan {
	jadwalID := "JDW-001"
	bankID := "BSM-001"
	return &models.Penimbangan{
		PenimbanganID:     "PNB-001",
		JadwalID:          &jadwalID,
		BankID:            &bankID,
		StatusPenimbangan: status,
		TanggalSesi:       pnbHariIni(),
	}
}

func pnbJadwalRutin(tanggal time.Time) *models.Jadwal {
	hari := utils.HariOf(tanggal)
	minggu := utils.WeekOfMonth(tanggal)
	return &models.Jadwal{
		JadwalID:    "JDW-001",
		BankID:      "BSM-001",
		Hari:        &hari,
		MingguKe:    &minggu,
		JamMulai:    "08:00:00",
		JamSelesai:  "11:00:00",
		JenisJadwal: models.JadwalPenimbangan,
		IsActive:    pnbBool(true),
		IsRutin:     pnbBool(true),
	}
}

func pnbJadwalSpesial(tanggal time.Time) *models.Jadwal {
	tgl := tanggal
	return &models.Jadwal{
		JadwalID:          "JDW-002",
		BankID:            "BSM-001",
		Tanggal:           &tgl,
		JamMulai:          "08:00:00",
		JamSelesai:        "11:00:00",
		JenisJadwal:       models.JadwalPenimbangan,
		IsActive:          pnbBool(true),
		IsRutin:           pnbBool(false),
		NamaJadwalSpesial: "Timbang Massal",
	}
}

func pnbKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func newPnbSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockPenimbanganRepo, *mocks.MockNotifSvc, services.PenimbanganService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockPenimbanganRepo)
	notif := new(mocks.MockNotifSvc)
	svc := services.NewPenimbanganService(gdb, repo, notif)

	repo.On("FindNasabahTargets", mock.Anything).Return(nil, errors.New("tidak distub")).Maybe()
	repo.On("FindAdminTargets", mock.Anything).Return(nil, errors.New("tidak distub")).Maybe()

	return dbMock, repo, notif, svc
}

func TestListSesiHariIni_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-XXX").Return(nil, errors.New("not found"))

	items, err := svc.ListSesiHariIni("BSM-XXX")

	assert.Nil(t, items)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "Bank sampah tidak ditemukan")
}

func TestListSesiHariIni_RepoError(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListSesiTanggal", "BSM-001", mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	items, err := svc.ListSesiHariIni("BSM-001")

	assert.Nil(t, items)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestListSesiHariIni_QueryTanpaFilterStatus(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	var semuaStatus []models.StatusPenimbangan
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListSesiTanggal", "BSM-001", pnbStrHariIni(), semuaStatus).
		Return([]repositories.SesiHariIniRow{}, nil)

	items, err := svc.ListSesiHariIni("BSM-001")

	assert.NoError(t, err)
	assert.NotNil(t, items, "slice kosong harus tetap non-nil supaya JSON-nya [] bukan null")
	assert.Len(t, items, 0)
	repo.AssertExpectations(t)
}

func TestListSesiHariIni_MappingDanNamaJadwal(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	tanggal := pnbHariIni()
	rows := []repositories.SesiHariIniRow{
		{
			PenimbanganID: "PNB-001", TanggalSesi: tanggal,
			JamMulai: "08:00:00", JamSelesai: "11:00:00",
			NamaJadwalSpesial: "harusnya diabaikan", IsRutin: pnbBool(true),
			StatusPenimbangan: models.StatusTimbangPending,
		},
		{
			PenimbanganID: "PNB-002", TanggalSesi: tanggal,
			JamMulai: "13:00:00", JamSelesai: "15:00:00",
			NamaJadwalSpesial: "Timbang Massal", IsRutin: pnbBool(false),
			StatusPenimbangan: models.StatusAktif,
		},
		{
			PenimbanganID: "PNB-003", TanggalSesi: tanggal,
			JamMulai: "16:00:00", JamSelesai: "17:00:00",
			NamaJadwalSpesial: "", IsRutin: nil,
			StatusPenimbangan: models.StatusSelesai,
		},
	}
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListSesiTanggal", "BSM-001", mock.Anything, mock.Anything).Return(rows, nil)

	items, err := svc.ListSesiHariIni("BSM-001")

	assert.NoError(t, err)
	assert.Len(t, items, 3)
	assert.Equal(t, "Jadwal penimbangan rutin", items[0].NamaJadwalSpesial)
	assert.Equal(t, "Timbang Massal", items[1].NamaJadwalSpesial)
	assert.Equal(t, "Jadwal penimbangan rutin", items[2].NamaJadwalSpesial, "is_rutin nil dianggap rutin")

	assert.Equal(t, "PNB-001", items[0].PenimbanganID)
	assert.Equal(t, tanggal, items[0].TanggalSesi)
	assert.Equal(t, "08:00:00", items[0].JamMulai)
	assert.Equal(t, "11:00:00", items[0].JamSelesai)
	assert.Equal(t, models.StatusTimbangPending, items[0].StatusPenimbangan)
	assert.Equal(t, models.StatusAktif, items[1].StatusPenimbangan)
}

func TestCekSesiAktif_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-XXX").Return(nil, errors.New("not found"))

	result, err := svc.CekSesiAktif("BSM-XXX")

	assert.Nil(t, result)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
}

func TestCekSesiAktif_RepoError(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListSesiTanggal", "BSM-001", mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	result, err := svc.CekSesiAktif("BSM-001")

	assert.Nil(t, result)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestCekSesiAktif_QueryHanyaPendingDanAktif(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListSesiTanggal", "BSM-001", pnbStrHariIni(),
		[]models.StatusPenimbangan{models.StatusTimbangPending, models.StatusAktif}).
		Return([]repositories.SesiHariIniRow{}, nil)

	_, err := svc.CekSesiAktif("BSM-001")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestCekSesiAktif_TidakAdaSesiSamaSekali(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListSesiTanggal", "BSM-001", mock.Anything, mock.Anything).
		Return([]repositories.SesiHariIniRow{}, nil)

	result, err := svc.CekSesiAktif("BSM-001")

	assert.NoError(t, err)
	assert.False(t, result.IsActive)
	assert.Nil(t, result.Detail)
	assert.Equal(t, 0, result.PendingSessions)
}

func TestCekSesiAktif_AdaPendingTapiBelumDibuka(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	rows := []repositories.SesiHariIniRow{
		{PenimbanganID: "PNB-001", StatusPenimbangan: models.StatusTimbangPending, IsRutin: pnbBool(true)},
		{PenimbanganID: "PNB-002", StatusPenimbangan: models.StatusTimbangPending, IsRutin: pnbBool(true)},
	}
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListSesiTanggal", "BSM-001", mock.Anything, mock.Anything).Return(rows, nil)

	result, err := svc.CekSesiAktif("BSM-001")

	assert.NoError(t, err)
	assert.False(t, result.IsActive)
	assert.Nil(t, result.Detail)
	assert.Equal(t, 2, result.PendingSessions)
}

func TestCekSesiAktif_AdaSesiAktif(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	tanggal := pnbHariIni()
	rows := []repositories.SesiHariIniRow{
		{PenimbanganID: "PNB-001", StatusPenimbangan: models.StatusTimbangPending, IsRutin: pnbBool(true)},
		{
			PenimbanganID: "PNB-002", TanggalSesi: tanggal,
			JamMulai: "13:00:00", JamSelesai: "15:00:00",
			NamaJadwalSpesial: "Timbang Massal", IsRutin: pnbBool(false),
			StatusPenimbangan: models.StatusAktif,
		},
	}
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListSesiTanggal", "BSM-001", mock.Anything, mock.Anything).Return(rows, nil)

	result, err := svc.CekSesiAktif("BSM-001")

	assert.NoError(t, err)
	assert.True(t, result.IsActive)
	assert.Equal(t, 1, result.PendingSessions, "sesi aktif tidak ikut dihitung sebagai pending")
	if assert.NotNil(t, result.Detail) {
		assert.Equal(t, "PNB-002", result.Detail.PenimbanganID)
		assert.Equal(t, tanggal, result.Detail.TanggalSesi)
		assert.Equal(t, "13:00:00", result.Detail.JamMulai)
		assert.Equal(t, "15:00:00", result.Detail.JamSelesai)
		assert.Equal(t, "Timbang Massal", result.Detail.NamaJadwalSpesial)
	}
}

func pnbUbahStatusInput(status models.StatusPenimbangan) services.UbahStatusSesiInput {
	return services.UbahStatusSesiInput{
		PenimbanganID: "PNB-001",
		StatusTujuan:  status,
		ActorUserID:   "USR-ADM-001",
		ActorBankID:   "BSM-001",
	}
}

func TestUbahStatusSesi_StatusTujuanDibatalkan_Ditolak(t *testing.T) {
	_, _, _, svc := newPnbSvc(t)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusDibatalkan))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "/penimbangan/batal")
}

func TestUbahStatusSesi_StatusTujuanPending_Ditolak(t *testing.T) {
	_, _, _, svc := newPnbSvc(t)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusTimbangPending))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
}

func TestUbahStatusSesi_PenimbanganTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindPenimbangan", "PNB-001").Return(nil, errors.New("not found"))

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusAktif))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, pnbKode(t, err))
}

func TestUbahStatusSesi_BukanMilikBankPemanggil(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindPenimbangan", "PNB-001").Return(pnbSesi(models.StatusTimbangPending), nil)

	in := pnbUbahStatusInput(models.StatusAktif)
	in.ActorBankID = "BSM-999"
	hasil, err := svc.UbahStatusSesi(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, pnbKode(t, err))
}

func TestUbahStatusSesi_BankIDSesiNil_Ditolak(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusTimbangPending)
	sesi.BankID = nil
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusAktif))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, pnbKode(t, err))
}

func TestUbahStatusSesi_MulaiSesi_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusTimbangPending)
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusTimbangPending).Return(sesi, nil)
	repo.On("CountSesiAktif", mock.Anything, "BSM-001", pnbStrHariIni()).Return(int64(0), nil)
	repo.On("SavePenimbangan", mock.Anything, sesi).Return(nil)
	expectTx(dbMock, 0)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusAktif))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.StatusAktif, hasil.StatusPenimbangan)
		assert.NotNil(t, hasil.StartedAt)
		if assert.NotNil(t, hasil.StartedBy) {
			assert.Equal(t, "USR-ADM-001", *hasil.StartedBy)
		}
		assert.Nil(t, hasil.EndedAt, "membuka sesi tidak boleh mengisi ended_at")
		assert.Nil(t, hasil.EndedBy)
	}
	repo.AssertExpectations(t)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestUbahStatusSesi_TutupSesi_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusAktif)
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusAktif).Return(sesi, nil)
	repo.On("SavePenimbangan", mock.Anything, sesi).Return(nil)
	expectTx(dbMock, 0)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusSelesai))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.StatusSelesai, hasil.StatusPenimbangan)
		assert.NotNil(t, hasil.EndedAt)
		if assert.NotNil(t, hasil.EndedBy) {
			assert.Equal(t, "USR-ADM-001", *hasil.EndedBy)
		}
	}
	repo.AssertNotCalled(t, "CountSesiAktif", mock.Anything, mock.Anything, mock.Anything)
	repo.AssertExpectations(t)
}

func TestUbahStatusSesi_MulaiSesi_MasihAdaSesiAktifLain(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusTimbangPending)
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusTimbangPending).Return(sesi, nil)
	repo.On("CountSesiAktif", mock.Anything, "BSM-001", mock.Anything).Return(int64(1), nil)
	expectTxRollback(dbMock)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusAktif))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, pnbKode(t, err))
	assert.Contains(t, err.Error(), "sesi penimbangan lain")
	repo.AssertNotCalled(t, "SavePenimbangan", mock.Anything, mock.Anything)
}

func TestUbahStatusSesi_MulaiSesi_DuplicateKeyJadiConflict(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusTimbangPending)
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusTimbangPending).Return(sesi, nil)
	repo.On("CountSesiAktif", mock.Anything, "BSM-001", mock.Anything).Return(int64(0), nil)
	repo.On("SavePenimbangan", mock.Anything, sesi).Return(gorm.ErrDuplicatedKey)
	expectTxRollback(dbMock)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusAktif))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, pnbKode(t, err))
}

func TestUbahStatusSesi_MulaiSesi_CountError(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusTimbangPending)
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusTimbangPending).Return(sesi, nil)
	repo.On("CountSesiAktif", mock.Anything, "BSM-001", mock.Anything).Return(int64(0), errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusAktif))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestUbahStatusSesi_MulaiSesi_StatusSudahBukanPending(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("FindPenimbangan", "PNB-001").Return(pnbSesi(models.StatusTimbangPending), nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusTimbangPending).
		Return(nil, gorm.ErrRecordNotFound)
	expectTxRollback(dbMock)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusAktif))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, pnbKode(t, err))
	assert.Contains(t, err.Error(), "bukan berstatus pending")
}

func TestUbahStatusSesi_TutupSesi_StatusSudahBukanAktif(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("FindPenimbangan", "PNB-001").Return(pnbSesi(models.StatusAktif), nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusAktif).
		Return(nil, gorm.ErrRecordNotFound)
	expectTxRollback(dbMock)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusSelesai))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, pnbKode(t, err))
	assert.Contains(t, err.Error(), "bukan berstatus aktif")
}

func TestUbahStatusSesi_SaveError(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusAktif)
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusAktif).Return(sesi, nil)
	repo.On("SavePenimbangan", mock.Anything, sesi).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.UbahStatusSesi(pnbUbahStatusInput(models.StatusSelesai))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestBatalkanSesi_AlasanKosong(t *testing.T) {
	_, _, _, svc := newPnbSvc(t)

	hasil, err := svc.BatalkanSesi(services.BatalkanSesiInput{
		PenimbanganID: "PNB-001",
		ActorBankID:   "BSM-001",
	})

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "alasan_pembatalan")
}

func pnbBatalPendingInput() services.BatalkanSesiInput {
	return services.BatalkanSesiInput{
		PenimbanganID:    "PNB-001",
		AlasanPembatalan: "Hujan deras",
		ActorUserID:      "USR-ADM-001",
		ActorBankID:      "BSM-001",
	}
}

func TestBatalkanSesiPending_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindPenimbangan", "PNB-001").Return(nil, errors.New("not found"))

	hasil, err := svc.BatalkanSesi(pnbBatalPendingInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, pnbKode(t, err))
}

func TestBatalkanSesiPending_BukanMilikBankPemanggil(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindPenimbangan", "PNB-001").Return(pnbSesi(models.StatusTimbangPending), nil)

	in := pnbBatalPendingInput()
	in.ActorBankID = "BSM-999"
	hasil, err := svc.BatalkanSesi(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, pnbKode(t, err))
}

func TestBatalkanSesiPending_JadwalIDNil(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusTimbangPending)
	sesi.JadwalID = nil
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)

	hasil, err := svc.BatalkanSesi(pnbBatalPendingInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
	assert.Contains(t, err.Error(), "jadwal")
}

func TestBatalkanSesiPending_JadwalTidakTerambil(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindPenimbangan", "PNB-001").Return(pnbSesi(models.StatusTimbangPending), nil)
	repo.On("FindJadwal", "JDW-001").Return(nil, errors.New("db error"))

	hasil, err := svc.BatalkanSesi(pnbBatalPendingInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestBatalkanSesiPending_StatusSudahBukanPending(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("FindPenimbangan", "PNB-001").Return(pnbSesi(models.StatusAktif), nil)
	repo.On("FindJadwal", "JDW-001").Return(pnbJadwalRutin(pnbHariIni()), nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusTimbangPending).
		Return(nil, gorm.ErrRecordNotFound)
	expectTxRollback(dbMock)

	hasil, err := svc.BatalkanSesi(pnbBatalPendingInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, pnbKode(t, err))
	assert.Contains(t, err.Error(), "bukan berstatus pending")
}

func TestBatalkanSesiPending_SaveError(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusTimbangPending)
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)
	repo.On("FindJadwal", "JDW-001").Return(pnbJadwalRutin(pnbHariIni()), nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusTimbangPending).Return(sesi, nil)
	repo.On("SavePenimbangan", mock.Anything, sesi).Return(errors.New("db error"))
	expectTxRollback(dbMock)

	hasil, err := svc.BatalkanSesi(pnbBatalPendingInput())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestBatalkanSesiPending_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	sesi := pnbSesi(models.StatusTimbangPending)
	repo.On("FindPenimbangan", "PNB-001").Return(sesi, nil)
	repo.On("FindJadwal", "JDW-001").Return(pnbJadwalRutin(pnbHariIni()), nil)
	repo.On("FindPenimbanganLock", mock.Anything, "PNB-001", models.StatusTimbangPending).Return(sesi, nil)
	repo.On("SavePenimbangan", mock.Anything, sesi).Return(nil)
	expectTx(dbMock, 0)

	hasil, err := svc.BatalkanSesi(pnbBatalPendingInput())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.False(t, hasil.Dibuat, "row-nya sudah ada, bukan baru dibuat")
		assert.Equal(t, models.StatusDibatalkan, hasil.Penimbangan.StatusPenimbangan)
		if assert.NotNil(t, hasil.Penimbangan.AlasanPembatalan) {
			assert.Equal(t, "Hujan deras", *hasil.Penimbangan.AlasanPembatalan)
		}
		assert.NotNil(t, hasil.Penimbangan.EndedAt)
		if assert.NotNil(t, hasil.Penimbangan.EndedBy) {
			assert.Equal(t, "USR-ADM-001", *hasil.Penimbangan.EndedBy)
		}
		assert.Nil(t, hasil.Penimbangan.StartedAt, "sesi yang dibatalkan tidak pernah dimulai")
	}
	repo.AssertExpectations(t)
}

func pnbBatalBaruInput(tanggal time.Time) services.BatalkanSesiInput {
	return services.BatalkanSesiInput{
		JadwalID:         "JDW-001",
		TanggalSesi:      tanggal.Format("2006-01-02"),
		AlasanPembatalan: "Libur nasional",
		ActorUserID:      "USR-ADM-001",
		ActorBankID:      "BSM-001",
	}
}

func TestBatalkanSesiBaru_JadwalIDKosong(t *testing.T) {
	_, _, _, svc := newPnbSvc(t)
	in := pnbBatalBaruInput(pnbBesok())
	in.JadwalID = ""

	hasil, err := svc.BatalkanSesi(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "jadwal_id")
}

func TestBatalkanSesiBaru_TanggalKosong(t *testing.T) {
	_, _, _, svc := newPnbSvc(t)
	in := pnbBatalBaruInput(pnbBesok())
	in.TanggalSesi = ""

	hasil, err := svc.BatalkanSesi(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
}

func TestBatalkanSesiBaru_FormatTanggalSalah(t *testing.T) {
	_, _, _, svc := newPnbSvc(t)
	in := pnbBatalBaruInput(pnbBesok())
	in.TanggalSesi = "27-09-2026"

	hasil, err := svc.BatalkanSesi(in)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestBatalkanSesiBaru_JadwalTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(nil, errors.New("not found"))

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, pnbKode(t, err))
}

func TestBatalkanSesiBaru_BankLain_DicekSebelumValidasiBisnis(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	jadwal := pnbJadwalRutin(pnbBesok())
	jadwal.BankID = "BSM-999"
	jadwal.IsActive = pnbBool(false)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(jadwal, nil)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, pnbKode(t, err))
}

func TestBatalkanSesiBaru_JadwalTidakAktif(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	jadwal := pnbJadwalRutin(pnbBesok())
	jadwal.IsActive = pnbBool(false)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(jadwal, nil)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "tidak aktif")
}

func TestBatalkanSesiBaru_TanggalSudahLewat(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalRutin(pnbKemarin()), nil)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbKemarin()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "sudah lewat")
}

func TestBatalkanSesiBaru_TanggalHariIniMasihBoleh(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalRutin(pnbHariIni()), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbTanggalParsed(pnbHariIni())).Return(int64(0), nil)
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil).Maybe()
	expectTx(dbMock, 1)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbHariIni()))

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestBatalkanSesiBaru_TanggalTidakCocokPolaRutin(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	lusa := pnbTanggalOnly(time.Now().AddDate(0, 0, 2))
	jadwal := pnbJadwalRutin(pnbBesok())
	hariBesok := utils.HariOf(pnbBesok())
	jadwal.Hari = &hariBesok
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(jadwal, nil)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(lusa))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "pola jadwal rutin")
}

func TestBatalkanSesiBaru_JadwalRutinTanpaPola(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	jadwal := pnbJadwalRutin(pnbBesok())
	jadwal.Hari = nil
	jadwal.MingguKe = nil
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(jadwal, nil)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "pola jadwal rutin")
}

func TestBatalkanSesiBaru_JadwalSpesialTanggalBeda(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	lusa := pnbTanggalOnly(time.Now().AddDate(0, 0, 2))
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalSpesial(pnbBesok()), nil)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(lusa))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
	assert.Contains(t, err.Error(), "tanggal jadwal ini")
}

func TestBatalkanSesiBaru_JadwalSpesialTanggalNil(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	jadwal := pnbJadwalSpesial(pnbBesok())
	jadwal.Tanggal = nil
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(jadwal, nil)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
}

func TestBatalkanSesiBaru_CekSesiExistingError(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalRutin(pnbBesok()), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbTanggalParsed(pnbBesok())).Return(int64(0), errors.New("db error"))

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestBatalkanSesiBaru_SesiSudahAda(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalRutin(pnbBesok()), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbTanggalParsed(pnbBesok())).Return(int64(1), nil)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, pnbKode(t, err))
	assert.Contains(t, err.Error(), "Sudah ada sesi penimbangan")
	repo.AssertNotCalled(t, "CreatePenimbangan", mock.Anything, mock.Anything)
}

func TestBatalkanSesiBaru_DuplicateKeySaatInsert(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalRutin(pnbBesok()), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbTanggalParsed(pnbBesok())).Return(int64(0), nil)
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"penimbangan_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, pnbKode(t, err))
}

func TestBatalkanSesiBaru_InsertError(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalRutin(pnbBesok()), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbTanggalParsed(pnbBesok())).Return(int64(0), nil)
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"penimbangan_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestBatalkanSesiBaru_GenerateIDGagal(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalRutin(pnbBesok()), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbTanggalParsed(pnbBesok())).Return(int64(0), nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("lock timeout"))
	dbMock.ExpectRollback()

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(pnbBesok()))

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
	assert.Contains(t, err.Error(), "generate penimbangan ID")
	repo.AssertNotCalled(t, "CreatePenimbangan", mock.Anything, mock.Anything)
}

func TestBatalkanSesiBaru_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	besok := pnbTanggalParsed(pnbBesok())
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalRutin(besok), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", besok).Return(int64(0), nil)

	var dibuat *models.Penimbangan
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { dibuat = args.Get(1).(*models.Penimbangan) }).
		Return(nil)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil).Maybe()
	expectTx(dbMock, 1)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(besok))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.True(t, hasil.Dibuat, "row baru dibuat, controller harus balas 201")
		assert.Equal(t, models.StatusDibatalkan, hasil.Penimbangan.StatusPenimbangan)
		assert.Equal(t, besok, hasil.Penimbangan.TanggalSesi)
	}
	if assert.NotNil(t, dibuat) {
		assert.Equal(t, "JDW-001", *dibuat.JadwalID)
		assert.Equal(t, "BSM-001", *dibuat.BankID)
		if assert.NotNil(t, dibuat.AlasanPembatalan) {
			assert.Equal(t, "Libur nasional", *dibuat.AlasanPembatalan)
		}
		assert.Nil(t, dibuat.StartedAt, "sesi yang tidak pernah terjadi tidak punya started_at")
		assert.Nil(t, dibuat.EndedAt)
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestBatalkanSesiBaru_JadwalSpesialSukses(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	besok := pnbTanggalParsed(pnbBesok())
	repo.On("FindJadwalPenimbangan", "JDW-001").Return(pnbJadwalSpesial(besok), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", besok).Return(int64(0), nil)
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil).Maybe()
	expectTx(dbMock, 1)

	hasil, err := svc.BatalkanSesi(pnbBatalBaruInput(besok))

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.True(t, hasil.Dibuat)
		assert.Equal(t, models.StatusDibatalkan, hasil.Penimbangan.StatusPenimbangan)
	}
}

func TestRiwayatPenimbangan_BankTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-XXX").Return(nil, errors.New("not found"))

	items, err := svc.RiwayatPenimbangan("BSM-XXX", "", "")

	assert.Nil(t, items)
	assert.Equal(t, http.StatusBadRequest, pnbKode(t, err))
}

func TestRiwayatPenimbangan_RepoError(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListRiwayat", "BSM-001", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	items, err := svc.RiwayatPenimbangan("BSM-001", "", "")

	assert.Nil(t, items)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestRiwayatPenimbangan_TanggalKosongJadiTanpaFilter(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListRiwayat", "BSM-001", (*time.Time)(nil), (*time.Time)(nil)).
		Return([]repositories.RiwayatPenimbanganRow{}, nil)

	items, err := svc.RiwayatPenimbangan("BSM-001", "", "")

	assert.NoError(t, err)
	assert.NotNil(t, items)
	repo.AssertExpectations(t)
}

func TestRiwayatPenimbangan_TanggalSalahFormatDiabaikan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListRiwayat", "BSM-001", (*time.Time)(nil), (*time.Time)(nil)).
		Return([]repositories.RiwayatPenimbanganRow{}, nil)

	items, err := svc.RiwayatPenimbangan("BSM-001", "01-09-2026", "bukan tanggal")

	assert.NoError(t, err)
	assert.NotNil(t, items)
	repo.AssertExpectations(t)
}

func TestRiwayatPenimbangan_TanggalValidDiteruskan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 30, 0, 0, 0, 0, time.UTC)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListRiwayat", "BSM-001",
		mock.MatchedBy(func(p *time.Time) bool { return p != nil && p.Equal(start) }),
		mock.MatchedBy(func(p *time.Time) bool { return p != nil && p.Equal(end) })).
		Return([]repositories.RiwayatPenimbanganRow{}, nil)

	_, err := svc.RiwayatPenimbangan("BSM-001", "2026-09-01", "2026-09-30")

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestRiwayatPenimbangan_Mapping(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	tanggal := pnbHariIni()
	rows := []repositories.RiwayatPenimbanganRow{
		{PenimbanganID: "PNB-002", TanggalSesi: tanggal, StatusPenimbangan: models.StatusSelesai, JumlahSetoran: 7},
		{PenimbanganID: "PNB-001", TanggalSesi: tanggal, StatusPenimbangan: models.StatusDibatalkan, JumlahSetoran: 0},
	}
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil)
	repo.On("ListRiwayat", "BSM-001", mock.Anything, mock.Anything).Return(rows, nil)

	items, err := svc.RiwayatPenimbangan("BSM-001", "", "")

	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "PNB-002", items[0].PenimbanganID)
	assert.Equal(t, models.StatusSelesai, items[0].StatusPenimbangan)
	assert.Equal(t, int64(7), items[0].JumlahSetoran)
	assert.Equal(t, tanggal, items[0].TanggalSesi)
	assert.Equal(t, models.StatusDibatalkan, items[1].StatusPenimbangan)
	assert.Equal(t, int64(0), items[1].JumlahSetoran)
}

func pnbHeader() *repositories.HeaderSesiRow {
	return &repositories.HeaderSesiRow{
		PenimbanganID:     "PNB-001",
		BankID:            "BSM-001",
		NamaBank:          "BSM Mawar",
		TanggalSesi:       pnbHariIni(),
		StatusPenimbangan: string(models.StatusSelesai),
	}
}

func TestDetailSesi_TidakDitemukan(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindHeaderSesi", "PNB-XXX").Return(nil, errors.New("not found"))

	header, err := svc.DetailSesi("PNB-XXX", services.AksesSesiInput{
		ActorBankID: "BSM-001", ActorRole: models.AdminBSM,
	})

	assert.Nil(t, header)
	assert.Equal(t, http.StatusNotFound, pnbKode(t, err))
}

func TestDetailSesi_BankLainDitolak(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindHeaderSesi", "PNB-001").Return(pnbHeader(), nil)

	header, err := svc.DetailSesi("PNB-001", services.AksesSesiInput{
		ActorBankID: "BSM-999", ActorRole: models.AdminBSM,
	})

	assert.Nil(t, header)
	assert.Equal(t, http.StatusForbidden, pnbKode(t, err))
	repo.AssertNotCalled(t, "ListSetoranSesi", mock.Anything)
}

func TestDetailSesi_ListSetoranError(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindHeaderSesi", "PNB-001").Return(pnbHeader(), nil)
	repo.On("ListSetoranSesi", "PNB-001").Return(nil, errors.New("db error"))

	header, err := svc.DetailSesi("PNB-001", services.AksesSesiInput{
		ActorBankID: "BSM-001", ActorRole: models.AdminBSM,
	})

	assert.Nil(t, header)
	assert.Equal(t, http.StatusInternalServerError, pnbKode(t, err))
}

func TestDetailSesi_SetoranNilJadiSliceKosong(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("FindHeaderSesi", "PNB-001").Return(pnbHeader(), nil)
	repo.On("ListSetoranSesi", "PNB-001").Return([]repositories.SetoranSesiRow(nil), nil)

	header, err := svc.DetailSesi("PNB-001", services.AksesSesiInput{
		ActorBankID: "BSM-001", ActorRole: models.AdminBSM,
	})

	assert.NoError(t, err)
	if assert.NotNil(t, header) {
		assert.NotNil(t, header.ListSetoran, "nil harus jadi [] supaya JSON-nya bukan null")
		assert.Len(t, header.ListSetoran, 0)
	}
}

func TestDetailSesi_Sukses(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	setoran := []repositories.SetoranSesiRow{
		{SetoranID: "STR-001", NamaPetugas: "Petugas A", NasabahID: "NSB-001", NamaNasabah: "Budi", TotalItem: 3},
	}
	repo.On("FindHeaderSesi", "PNB-001").Return(pnbHeader(), nil)
	repo.On("ListSetoranSesi", "PNB-001").Return(setoran, nil)

	header, err := svc.DetailSesi("PNB-001", services.AksesSesiInput{
		ActorBankID: "BSM-001", ActorRole: models.AdminBSM,
	})

	assert.NoError(t, err)
	if assert.NotNil(t, header) {
		assert.Equal(t, "PNB-001", header.PenimbanganID)
		assert.Equal(t, "BSM Mawar", header.NamaBank)
		assert.Len(t, header.ListSetoran, 1)
		assert.Equal(t, "STR-001", header.ListSetoran[0].SetoranID)
	}
}

func TestJamBuatSesiPending(t *testing.T) {
	_, _, _, svc := newPnbSvc(t)
	assert.Equal(t, 20, svc.JamBuatSesiPending(), "worker membuat sesi pending jam 20:00 H-1")
}

func pnbJadwalBesok(jadwalID, bankID string) models.Jadwal {
	j := pnbJadwalRutin(pnbBesok())
	j.JadwalID = jadwalID
	j.BankID = bankID
	return *j
}

func TestBuatSesiPendingBesok_QueryPakaiPolaBesok(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	now := time.Date(2026, 9, 23, 20, 0, 0, 0, time.Local)
	besok := time.Date(2026, 9, 24, 0, 0, 0, 0, time.Local)
	repo.On("ListJadwalJatuhTempo", utils.HariOf(besok), utils.WeekOfMonth(besok), "2026-09-24").
		Return([]models.Jadwal{}, nil)

	dibuat, sukses := svc.BuatSesiPendingBesok(now)

	assert.Equal(t, 0, dibuat)
	assert.True(t, sukses)
	repo.AssertExpectations(t)
}

func TestBuatSesiPendingBesok_QueryJadwalGagal(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("db error"))

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 0, dibuat)
	assert.False(t, sukses, "worker harus tahu ini gagal supaya bisa retry")
}

func TestBuatSesiPendingBesok_TidakAdaJadwal(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return([]models.Jadwal{}, nil)

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 0, dibuat)
	assert.True(t, sukses)
}

func TestBuatSesiPendingBesok_SesiSudahAdaDilewati(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return([]models.Jadwal{pnbJadwalBesok("JDW-001", "BSM-001")}, nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbBesok()).Return(int64(1), nil)

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 0, dibuat)
	assert.True(t, sukses)
	repo.AssertNotCalled(t, "CreatePenimbangan", mock.Anything, mock.Anything)
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestBuatSesiPendingBesok_CekSesiExistingGagal(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return([]models.Jadwal{pnbJadwalBesok("JDW-001", "BSM-001")}, nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbBesok()).Return(int64(0), errors.New("db error"))

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 0, dibuat)
	assert.False(t, sukses)
}

func TestBuatSesiPendingBesok_DuplicateKeyBukanKegagalan(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return([]models.Jadwal{pnbJadwalBesok("JDW-001", "BSM-001")}, nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbBesok()).Return(int64(0), nil)
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).Return(gorm.ErrDuplicatedKey)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"penimbangan_id"}))
	dbMock.ExpectRollback()

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 0, dibuat)
	assert.True(t, sukses)
}

func TestBuatSesiPendingBesok_InsertGagal(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return([]models.Jadwal{pnbJadwalBesok("JDW-001", "BSM-001")}, nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbBesok()).Return(int64(0), nil)
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"penimbangan_id"}))
	dbMock.ExpectRollback()

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 0, dibuat)
	assert.False(t, sukses)
}

func TestBuatSesiPendingBesok_Sukses(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	besok := pnbBesok()
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return([]models.Jadwal{
			pnbJadwalBesok("JDW-001", "BSM-001"),
			pnbJadwalBesok("JDW-002", "BSM-001"),
		}, nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", besok).Return(int64(0), nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-002", besok).Return(int64(0), nil)

	var terbuat []*models.Penimbangan
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) { terbuat = append(terbuat, args.Get(1).(*models.Penimbangan)) }).
		Return(nil)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil).Maybe()
	expectTx(dbMock, 1)
	expectTx(dbMock, 1)

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 2, dibuat)
	assert.True(t, sukses)
	if assert.Len(t, terbuat, 2) {
		for _, p := range terbuat {
			assert.Equal(t, models.StatusTimbangPending, p.StatusPenimbangan)
			assert.Equal(t, besok, p.TanggalSesi)
			assert.Nil(t, p.AlasanPembatalan, "sesi pending belum punya alasan pembatalan")
			assert.Nil(t, p.StartedAt)
		}
		assert.Equal(t, "JDW-001", *terbuat[0].JadwalID)
		assert.Equal(t, "JDW-002", *terbuat[1].JadwalID)
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestBuatSesiPendingBesok_SebagianGagalTetapLanjut(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	besok := pnbBesok()
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return([]models.Jadwal{
			pnbJadwalBesok("JDW-001", "BSM-001"),
			pnbJadwalBesok("JDW-002", "BSM-001"),
		}, nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", besok).Return(int64(0), errors.New("db error"))
	repo.On("CountSesiByJadwalTanggal", "JDW-002", besok).Return(int64(0), nil)
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindBank", "BSM-001").Return(pnbBank(), nil).Maybe()
	expectTx(dbMock, 1)

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 1, dibuat, "jadwal kedua tetap diproses")
	assert.False(t, sukses)
}

func TestBuatSesiPendingBesok_BankTidakDitemukanSesiTetapDibuat(t *testing.T) {
	dbMock, repo, _, svc := newPnbSvc(t)
	repo.On("ListJadwalJatuhTempo", mock.Anything, mock.Anything, mock.Anything).
		Return([]models.Jadwal{pnbJadwalBesok("JDW-001", "BSM-001")}, nil)
	repo.On("CountSesiByJadwalTanggal", "JDW-001", pnbBesok()).Return(int64(0), nil)
	repo.On("CreatePenimbangan", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindBank", "BSM-001").Return(nil, errors.New("not found"))
	expectTx(dbMock, 1)

	dibuat, sukses := svc.BuatSesiPendingBesok(time.Now())

	assert.Equal(t, 1, dibuat)
	assert.True(t, sukses)
}

func TestTutupSesiTerlewat_Error(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("TutupSesiTerlewat").Return(int64(0), errors.New("db error"))

	rows, err := svc.TutupSesiTerlewat()

	assert.Equal(t, int64(0), rows)
	assert.Error(t, err)
}

func TestTutupSesiTerlewat_Sukses(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("TutupSesiTerlewat").Return(int64(3), nil)

	rows, err := svc.TutupSesiTerlewat()

	assert.NoError(t, err)
	assert.Equal(t, int64(3), rows)
}

func TestTutupSesiTerlewat_TidakAdaYangDitutup(t *testing.T) {
	_, repo, _, svc := newPnbSvc(t)
	repo.On("TutupSesiTerlewat").Return(int64(0), nil)

	rows, err := svc.TutupSesiTerlewat()

	assert.NoError(t, err)
	assert.Equal(t, int64(0), rows)
}
