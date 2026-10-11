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

func newJdwSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockJadwalRepo, services.JadwalService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockJadwalRepo)
	return dbMock, repo, services.NewJadwalService(gdb, repo)
}

func jdwBool(b bool) *bool { return &b }
func jdwInt(i int) *int    { return &i }

func jdwTanggal(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}

func jdwBesok() time.Time {
	besok := time.Now().AddDate(0, 0, 1)
	return jdwTanggal(besok.Year(), besok.Month(), besok.Day())
}

func jdwBank(bankID string, jenis models.JenisBank) *models.BankSampah {
	return &models.BankSampah{BankID: bankID, NamaBank: "Bank " + bankID, JenisBank: jenis, IsActive: true}
}

func jdwAkses() services.AksesJadwalInput {
	return services.AksesJadwalInput{
		ActorUserID: "USR-001",
		ActorBankID: "BSU-001",
		ActorRole:   models.AdminBSU,
	}
}

func jdwRutinDari(jadwalID string, tanggal time.Time, jamMulai, jamSelesai string) models.Jadwal {
	hari := utils.HariOf(tanggal)
	minggu := utils.WeekOfMonth(tanggal)
	return models.Jadwal{
		JadwalID:    jadwalID,
		BankID:      "BSU-001",
		Hari:        &hari,
		MingguKe:    &minggu,
		JamMulai:    jamMulai,
		JamSelesai:  jamSelesai,
		JenisJadwal: models.JadwalPenimbangan,
		IsActive:    jdwBool(true),
		IsRutin:     jdwBool(true),
		CreatedAt:   jdwTanggal(2020, time.January, 1),
	}
}

func jdwKhusus(jadwalID string, tanggal time.Time, jamMulai, jamSelesai, nama string) models.Jadwal {
	tgl := tanggal
	return models.Jadwal{
		JadwalID:          jadwalID,
		BankID:            "BSU-001",
		Tanggal:           &tgl,
		JamMulai:          jamMulai,
		JamSelesai:        jamSelesai,
		JenisJadwal:       models.JadwalPenimbangan,
		IsActive:          jdwBool(true),
		IsRutin:           jdwBool(false),
		NamaJadwalSpesial: nama,
		CreatedAt:         jdwTanggal(2020, time.January, 1),
	}
}

func jdwKode(t *testing.T, err error) int {
	t.Helper()
	se, ok := err.(*services.ServiceError)
	if !ok {
		t.Fatalf("expected *services.ServiceError, got %T (%v)", err, err)
	}
	return se.Code
}

func jdwTambahRutin() services.TambahJadwalInput {
	return services.TambahJadwalInput{
		Hari:        models.Senin,
		MingguKe:    2,
		JamMulai:    "08:00",
		JamSelesai:  "11:00",
		JenisJadwal: models.JadwalPenimbangan,
		IsRutin:     true,
	}
}

func jdwTambahKhusus() services.TambahJadwalInput {
	return services.TambahJadwalInput{
		JamMulai:          "08:00",
		JamSelesai:        "11:00",
		JenisJadwal:       models.JadwalPenimbangan,
		IsRutin:           false,
		Tanggal:           jdwBesok().Format("2006-01-02"),
		NamaJadwalSpesial: "Timbang Massal",
	}
}

func TestTambahJadwal_BankIDKosong(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	hasil, err := svc.TambahJadwal("", jdwTambahRutin(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "bank_id")
}

func TestTambahJadwal_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.TambahJadwal("BSU-XXX", jdwTambahRutin(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, jdwKode(t, err))
}

func TestTambahJadwal_JenisJadwalTidakValid(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahRutin()
	in.JenisJadwal = "pembakaran"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "jenis_jadwal")
}

func TestTambahJadwal_PengangkutanTanpaTargetBank(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSI-001").Return(jdwBank("BSI-001", models.BSI), nil)

	in := jdwTambahRutin()
	in.JenisJadwal = models.JadwalPengangkutan
	hasil, err := svc.TambahJadwal("BSI-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "target_bank_id")
}

func TestTambahJadwal_BSUTidakBisaBuatPengangkutan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahRutin()
	in.JenisJadwal = models.JadwalPengangkutan
	in.TargetBankID = "BSU-002"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "target_bank_id")
}

func TestTambahJadwal_JamKosong(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahRutin()
	in.JamMulai = ""
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "wajib diisi")
}

func TestTambahJadwal_JamFormatSalah(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahRutin()
	in.JamMulai = "8 pagi"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "jam_mulai")
}

func TestTambahJadwal_JamSelesaiTidakLebihBesar(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahRutin()
	in.JamMulai, in.JamSelesai = "11:00", "08:00"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "lebih besar")
}

func TestTambahJadwal_JamDenganDetikDinormalisasi(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahRutin()
	in.JamMulai, in.JamSelesai = "08:00:00", "11:00:00"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "08:00", hasil.JamMulai)
		assert.Equal(t, "11:00", hasil.JamSelesai)
	}
}

func TestTambahJadwal_RutinHariTidakValid(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahRutin()
	in.Hari = "sabtu kliwon"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "hari harus")
}

func TestTambahJadwal_RutinMingguKeDiLuarRentang(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil).Times(2)

	for _, minggu := range []int{0, 6} {
		in := jdwTambahRutin()
		in.MingguKe = minggu
		hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

		assert.Nil(t, hasil)
		assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
		assert.Contains(t, err.Error(), "minggu_ke")
	}
}

func TestTambahJadwal_KhususTanpaTanggal(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahKhusus()
	in.Tanggal = ""
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "tanggal wajib diisi")
}

func TestTambahJadwal_KhususTanggalFormatSalah(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahKhusus()
	in.Tanggal = "17-09-2026"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestTambahJadwal_KhususTanggalSudahLewat(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

	in := jdwTambahKhusus()
	in.Tanggal = time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "minimal besok")
}

func TestTambahJadwal_KhususHariIniDitolak(t *testing.T) {
	kasus := [][2]string{{"00:00", "00:01"}, {"17:00", "21:00"}, {"21:00", "23:00"}}
	for _, jam := range kasus {
		t.Run(jam[0]+"-"+jam[1], func(t *testing.T) {
			_, repo, svc := newJdwSvc(t)
			repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)

			in := jdwTambahKhusus()
			in.Tanggal = time.Now().Format("2006-01-02")
			in.JamMulai, in.JamSelesai = jam[0], jam[1]
			hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

			assert.Nil(t, hasil)
			assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
			assert.Contains(t, err.Error(), "minimal besok")
		})
	}
}

func TestTambahJadwal_ListJadwalError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(nil, errors.New("db error"))

	hasil, err := svc.TambahJadwal("BSU-001", jdwTambahRutin(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestTambahJadwal_BentrokRutinDenganRutin(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	existing := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "09:00", "12:00")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{existing}, nil)

	in := jdwTambahRutin()
	in.Hari = *existing.Hari
	in.MingguKe = *existing.MingguKe
	in.JamMulai, in.JamSelesai = "11:00", "13:00"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
	assert.Contains(t, err.Error(), "bentrok")
}

func TestTambahJadwal_JamBersebelahanTidakBentrok(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	existing := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{existing}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahRutin()
	in.Hari = *existing.Hari
	in.MingguKe = *existing.MingguKe
	in.JamMulai, in.JamSelesai = "11:00", "13:00"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestTambahJadwal_MingguBerbedaTidakBentrok(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	existing := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{existing}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahRutin()
	in.Hari = *existing.Hari
	in.MingguKe = *existing.MingguKe%5 + 1
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestTambahJadwal_KhususBentrokDenganPolaRutin(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	tanggal := jdwBesok()
	existing := jdwRutinDari("RTBSU-0010001", tanggal, "08:00", "11:00")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{existing}, nil)

	hasil, err := svc.TambahJadwal("BSU-001", jdwTambahKhusus(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
}

func TestTambahJadwal_KhususBentrokDenganKhususTanggalSama(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	existing := jdwKhusus("KTBSU-0010001", jdwBesok(), "09:00", "12:00", "Lama")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{existing}, nil)

	hasil, err := svc.TambahJadwal("BSU-001", jdwTambahKhusus(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
}

func TestTambahJadwal_KhususTanggalBerbedaTidakBentrok(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	existing := jdwKhusus("KTBSU-0010001", jdwBesok().AddDate(0, 0, 1), "08:00", "11:00", "Lama")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{existing}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	hasil, err := svc.TambahJadwal("BSU-001", jdwTambahKhusus(), jdwAkses())

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestTambahJadwal_PengangkutanRuteBerbedaTidakBentrok(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	existing := jdwRutinDari("RABSU-0020001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	existing.BankID = "BSI-001"
	existing.JenisJadwal = models.JadwalPengangkutan
	existing.TargetBankID = ptrStr("BSU-002")

	repo.On("FindBank", "BSI-001").Return(jdwBank("BSI-001", models.BSI), nil)
	repo.On("ListJadwalBank", "BSI-001", models.JadwalPengangkutan).Return([]models.Jadwal{existing}, nil)
	repo.On("FindAdminIDByUserBank", mock.Anything, mock.Anything).Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahRutin()
	in.JenisJadwal = models.JadwalPengangkutan
	in.TargetBankID = "BSU-003"
	in.Hari = *existing.Hari
	in.MingguKe = *existing.MingguKe

	akses := jdwAkses()
	akses.ActorBankID = "BSI-001"
	hasil, err := svc.TambahJadwal("BSI-001", in, akses)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestTambahJadwal_PengangkutanRuteSamaBentrok(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	existing := jdwRutinDari("RABSU-0020001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	existing.BankID = "BSI-001"
	existing.JenisJadwal = models.JadwalPengangkutan
	existing.TargetBankID = ptrStr("BSU-002")

	repo.On("FindBank", "BSI-001").Return(jdwBank("BSI-001", models.BSI), nil)
	repo.On("ListJadwalBank", "BSI-001", models.JadwalPengangkutan).Return([]models.Jadwal{existing}, nil)

	in := jdwTambahRutin()
	in.JenisJadwal = models.JadwalPengangkutan
	in.TargetBankID = "BSU-002"
	in.Hari = *existing.Hari
	in.MingguKe = *existing.MingguKe

	akses := jdwAkses()
	akses.ActorBankID = "BSI-001"
	hasil, err := svc.TambahJadwal("BSI-001", in, akses)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
}

func TestTambahJadwal_AdminLookupError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("", errors.New("db error"))

	hasil, err := svc.TambahJadwal("BSU-001", jdwTambahRutin(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestTambahJadwal_AdminTidakDitemukan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("", nil)

	hasil, err := svc.TambahJadwal("BSU-001", jdwTambahRutin(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, jdwKode(t, err))
	assert.Contains(t, err.Error(), "admin aktif")
}

func TestTambahJadwal_GenerateIDGagal(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnError(errors.New("lock timeout"))
	dbMock.ExpectRollback()

	hasil, err := svc.TambahJadwal("BSU-001", jdwTambahRutin(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
	assert.Contains(t, err.Error(), "generate jadwal ID")
	repo.AssertNotCalled(t, "CreateJadwal", mock.Anything, mock.Anything)
}

func TestTambahJadwal_CreateError(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"jadwal_id"}))
	dbMock.ExpectRollback()

	hasil, err := svc.TambahJadwal("BSU-001", jdwTambahRutin(), jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
	assert.Contains(t, err.Error(), "menyimpan jadwal")
}

func TestTambahJadwal_RutinSukses(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahRutin()
	in.NamaJadwalSpesial = "harusnya diabaikan"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "RTBSU-0010001", hasil.JadwalID, "rutin+penimbangan = prefix RT + bank sendiri")
		assert.Equal(t, "BSU-001", hasil.BankID)
		assert.Equal(t, "ADM-001", hasil.CreatedBy, "created_by diambil dari token, bukan body")
		if assert.NotNil(t, hasil.Hari) {
			assert.Equal(t, models.Senin, *hasil.Hari)
		}
		if assert.NotNil(t, hasil.MingguKe) {
			assert.Equal(t, 2, *hasil.MingguKe)
		}
		assert.Nil(t, hasil.Tanggal, "jadwal rutin tidak menyimpan tanggal")
		assert.Equal(t, "", hasil.NamaJadwalSpesial, "nama khusus diabaikan untuk jadwal rutin")
		assert.Nil(t, hasil.TargetBankID)
		if assert.NotNil(t, hasil.IsActive) {
			assert.True(t, *hasil.IsActive, "tanpa is_active dianggap aktif, ikut DEFAULT true di DB")
		}
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahJadwal_KhususSukses(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahKhusus()
	in.Hari = models.Rabu
	in.MingguKe = 3
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "KTBSU-0010001", hasil.JadwalID, "khusus = prefix KT")
		assert.Nil(t, hasil.Hari, "jadwal khusus tidak menyimpan turunan hari")
		assert.Nil(t, hasil.MingguKe)
		assert.Equal(t, "Timbang Massal", hasil.NamaJadwalSpesial)
		if assert.NotNil(t, hasil.Tanggal) {
			assert.Equal(t, jdwBesok().Format("2006-01-02"), hasil.Tanggal.Format("2006-01-02"))
		}
	}
}

func TestTambahJadwal_IsActiveFalseDihormati(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahRutin()
	in.IsActive = jdwBool(false)
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil.IsActive) {
		assert.False(t, *hasil.IsActive)
	}
}

func TestTambahJadwal_PengangkutanIDPakaiTargetBank(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSI-001").Return(jdwBank("BSI-001", models.BSI), nil)
	repo.On("ListJadwalBank", "BSI-001", models.JadwalPengangkutan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", mock.Anything, mock.Anything).Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahRutin()
	in.JenisJadwal = models.JadwalPengangkutan
	in.TargetBankID = "BSU-002"

	akses := jdwAkses()
	akses.ActorBankID = "BSI-001"
	hasil, err := svc.TambahJadwal("BSI-001", in, akses)

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "RABSU-0020001", hasil.JadwalID)
		assert.Equal(t, "BSI-001", hasil.BankID)
		if assert.NotNil(t, hasil.TargetBankID) {
			assert.Equal(t, "BSU-002", *hasil.TargetBankID)
		}
	}
}

func TestTambahJadwal_PenimbanganAbaikanTargetBank(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 1)

	in := jdwTambahRutin()
	in.TargetBankID = "BSU-009"
	hasil, err := svc.TambahJadwal("BSU-001", in, jdwAkses())

	assert.NoError(t, err)
	assert.Nil(t, hasil.TargetBankID)
}

func jdwBatchItem(hari models.HariEnum, minggu int, mulai, selesai string) services.BatchJadwalItem {
	return services.BatchJadwalItem{Hari: hari, MingguKe: minggu, JamMulai: mulai, JamSelesai: selesai}
}

func TestTambahJadwalBatch_BankIDKosong(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	hasil, err := svc.TambahJadwalBatch("", []services.BatchJadwalItem{jdwBatchItem(models.Senin, 1, "08:00", "11:00")}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
}

func TestTambahJadwalBatch_ItemKosong(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	hasil, err := svc.TambahJadwalBatch("BSU-001", nil, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "Minimal satu jadwal")
}

func TestTambahJadwalBatch_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.TambahJadwalBatch("BSU-XXX", []services.BatchJadwalItem{jdwBatchItem(models.Senin, 1, "08:00", "11:00")}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, jdwKode(t, err))
}

func TestTambahJadwalBatch_ListJadwalError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(nil, errors.New("db error"))

	hasil, err := svc.TambahJadwalBatch("BSU-001", []services.BatchJadwalItem{jdwBatchItem(models.Senin, 1, "08:00", "11:00")}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestTambahJadwalBatch_ItemTidakValid(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)

	items := []services.BatchJadwalItem{
		jdwBatchItem(models.Senin, 1, "08:00", "11:00"),
		jdwBatchItem(models.Selasa, 9, "08:00", "11:00"),
	}
	hasil, err := svc.TambahJadwalBatch("BSU-001", items, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "minggu_ke")
}

func TestTambahJadwalBatch_BentrokDenganJadwalExisting(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	existing := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{existing}, nil)

	items := []services.BatchJadwalItem{jdwBatchItem(*existing.Hari, *existing.MingguKe, "09:00", "12:00")}
	hasil, err := svc.TambahJadwalBatch("BSU-001", items, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
	assert.Contains(t, err.Error(), "bentrok")
}

func TestTambahJadwalBatch_BentrokAntarItem(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)

	items := []services.BatchJadwalItem{
		jdwBatchItem(models.Senin, 1, "08:00", "11:00"),
		jdwBatchItem(models.Senin, 1, "10:00", "12:00"),
	}
	hasil, err := svc.TambahJadwalBatch("BSU-001", items, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
	repo.AssertNotCalled(t, "CreateJadwal", mock.Anything, mock.Anything)
}

func TestTambahJadwalBatch_Sukses(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(nil)
	expectTx(dbMock, 2)

	items := []services.BatchJadwalItem{
		jdwBatchItem(models.Senin, 1, "08:00", "11:00"),
		jdwBatchItem(models.Rabu, 3, "13:00", "15:00"),
	}
	hasil, err := svc.TambahJadwalBatch("BSU-001", items, jdwAkses())

	assert.NoError(t, err)
	if assert.Len(t, hasil, 2) {
		for _, j := range hasil {
			assert.Equal(t, models.JadwalPenimbangan, j.JenisJadwal)
			assert.Equal(t, "ADM-001", j.CreatedBy)
			assert.Equal(t, "BSU-001", j.BankID)
			if assert.NotNil(t, j.IsRutin) {
				assert.True(t, *j.IsRutin, "batch hanya untuk jadwal rutin")
			}
			if assert.NotNil(t, j.IsActive) {
				assert.True(t, *j.IsActive)
			}
			assert.Nil(t, j.Tanggal)
		}
		assert.Equal(t, models.Senin, *hasil[0].Hari)
		assert.Equal(t, "13:00", hasil[1].JamMulai)
	}
	assert.NoError(t, dbMock.ExpectationsWereMet())
}

func TestTambahJadwalBatch_CreateErrorRollback(t *testing.T) {
	dbMock, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("FindAdminIDByUserBank", "USR-001", "BSU-001").Return("ADM-001", nil)
	repo.On("CreateJadwal", mock.Anything, mock.Anything).Return(errors.New("db error"))
	dbMock.ExpectBegin()
	dbMock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"jadwal_id"}))
	dbMock.ExpectRollback()

	items := []services.BatchJadwalItem{jdwBatchItem(models.Senin, 1, "08:00", "11:00")}
	hasil, err := svc.TambahJadwalBatch("BSU-001", items, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestJadwalBank_BankIDKosong(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	hasil, err := svc.JadwalBank("", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
}

func TestJadwalBank_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.JadwalBank("BSU-XXX", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, jdwKode(t, err))
}

func TestJadwalBank_ErrorPenimbangan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBankWithNama", "BSU-001", models.JadwalPenimbangan, false, 9, 2026).
		Return(nil, errors.New("db error"))

	hasil, err := svc.JadwalBank("BSU-001", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
	assert.Contains(t, err.Error(), "penimbangan")
}

func TestJadwalBank_ErrorPengangkutan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBankWithNama", "BSU-001", models.JadwalPenimbangan, false, 9, 2026).
		Return([]repositories.JadwalBankRow{}, nil)
	repo.On("ListJadwalBankWithNama", "BSU-001", models.JadwalPengangkutan, true, 9, 2026).
		Return(nil, errors.New("db error"))

	hasil, err := svc.JadwalBank("BSU-001", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
	assert.Contains(t, err.Error(), "pengangkutan")
}

func TestJadwalBank_BSUDicariSebagaiTarget(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBankWithNama", "BSU-001", models.JadwalPenimbangan, false, 9, 2026).
		Return([]repositories.JadwalBankRow{}, nil)
	repo.On("ListJadwalBankWithNama", "BSU-001", models.JadwalPengangkutan, true, 9, 2026).
		Return([]repositories.JadwalBankRow{}, nil)

	hasil, err := svc.JadwalBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	assert.NotNil(t, hasil.Penimbangan, "slice kosong harus non-nil supaya JSON-nya []")
	assert.NotNil(t, hasil.Pengangkutan)
	repo.AssertExpectations(t)
}

func TestJadwalBank_BSIDicariSebagaiPemilik(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSI-001").Return(jdwBank("BSI-001", models.BSI), nil)
	repo.On("ListJadwalBankWithNama", "BSI-001", models.JadwalPenimbangan, false, 9, 2026).
		Return([]repositories.JadwalBankRow{}, nil)
	repo.On("ListJadwalBankWithNama", "BSI-001", models.JadwalPengangkutan, false, 9, 2026).
		Return([]repositories.JadwalBankRow{}, nil)

	_, err := svc.JadwalBank("BSI-001", 9, 2026)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestJadwalBank_Mapping(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	penimbangan := []repositories.JadwalBankRow{
		{Jadwal: jdwRutinDari("RTBSI-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00"), BankName: "BSI Utama"},
	}
	pengangkutan := []repositories.JadwalBankRow{
		{Jadwal: jdwRutinDari("RABSU-0020001", jdwTanggal(2026, time.September, 15), "13:00", "15:00"),
			BankName: "BSI Utama", TargetBankName: "BSU Mawar"},
	}
	repo.On("FindBank", "BSI-001").Return(jdwBank("BSI-001", models.BSI), nil)
	repo.On("ListJadwalBankWithNama", "BSI-001", models.JadwalPenimbangan, false, 9, 2026).Return(penimbangan, nil)
	repo.On("ListJadwalBankWithNama", "BSI-001", models.JadwalPengangkutan, false, 9, 2026).Return(pengangkutan, nil)

	hasil, err := svc.JadwalBank("BSI-001", 9, 2026)

	assert.NoError(t, err)
	if assert.Len(t, hasil.Penimbangan, 1) {
		assert.Equal(t, "RTBSI-0010001", hasil.Penimbangan[0].JadwalID)
		assert.Equal(t, "BSI Utama", hasil.Penimbangan[0].BankName)
	}
	if assert.Len(t, hasil.Pengangkutan, 1) {
		assert.Equal(t, "BSU Mawar", hasil.Pengangkutan[0].TargetBankName)
	}
}

func TestJadwalPenimbanganBank_BankIDKosong(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	hasil, err := svc.JadwalPenimbanganBank("", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
}

func TestJadwalPenimbanganBank_PeriodeTidakValid(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	for _, tc := range []struct{ month, year int }{{0, 2026}, {13, 2026}, {9, 0}} {
		hasil, err := svc.JadwalPenimbanganBank("BSU-001", tc.month, tc.year)
		assert.Nil(t, hasil)
		assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	}
}

func TestJadwalPenimbanganBank_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.JadwalPenimbanganBank("BSU-XXX", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, jdwKode(t, err))
}

func TestJadwalPenimbanganBank_ListJadwalError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(nil, errors.New("db error"))

	hasil, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestJadwalPenimbanganBank_StatusSesiError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("ListStatusSesi", mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	hasil, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
	assert.Contains(t, err.Error(), "status sesi")
}

func TestJadwalPenimbanganBank_RutinJadiTanggalKonkret(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	target := jdwTanggal(2026, time.September, 14)
	rutin := jdwRutinDari("RTBSU-0010001", target, "08:00", "11:00")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{rutin}, nil)
	repo.On("ListStatusSesi", mock.Anything, mock.Anything, mock.Anything).Return(map[string]string{}, nil)

	items, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	if assert.Len(t, items, 1) {
		assert.Equal(t, "2026-09-14", items[0].Tanggal)
		assert.Equal(t, "Penimbangan rutin", items[0].NamaJadwal, "jadwal rutin pakai label generik")
		assert.True(t, items[0].IsRutin)
		assert.True(t, items[0].IsActive)
		assert.Equal(t, "upcoming", items[0].StatusJadwal, "belum ada baris penimbangan = upcoming")
	}
}

func TestJadwalPenimbanganBank_StatusDariSesiDipakai(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	target := jdwTanggal(2026, time.September, 14)
	rutin := jdwRutinDari("RTBSU-0010001", target, "08:00", "11:00")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{rutin}, nil)
	repo.On("ListStatusSesi", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]string{"RTBSU-0010001|2026-09-14": "selesai"}, nil)

	items, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	if assert.Len(t, items, 1) {
		assert.Equal(t, "selesai", items[0].StatusJadwal)
	}
}

func TestJadwalPenimbanganBank_OccurrenceSebelumCreatedAtDilewati(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	target := jdwTanggal(2026, time.September, 14)
	rutin := jdwRutinDari("RTBSU-0010001", target, "08:00", "11:00")
	rutin.CreatedAt = jdwTanggal(2026, time.September, 20)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{rutin}, nil)
	repo.On("ListStatusSesi", []string{}, mock.Anything, mock.Anything).Return(map[string]string{}, nil)

	items, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestJadwalPenimbanganBank_KhususPakaiNamaAsli(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	khusus := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "Timbang Massal")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{khusus}, nil)
	repo.On("ListStatusSesi", mock.Anything, mock.Anything, mock.Anything).Return(map[string]string{}, nil)

	items, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	if assert.Len(t, items, 1) {
		assert.Equal(t, "Timbang Massal", items[0].NamaJadwal)
		assert.False(t, items[0].IsRutin)
	}
}

func TestJadwalPenimbanganBank_KhususDiLuarBulanTidakMuncul(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	khusus := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.October, 5), "08:00", "11:00", "Bulan depan")
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{khusus}, nil)
	repo.On("ListStatusSesi", mock.Anything, mock.Anything, mock.Anything).Return(map[string]string{}, nil)

	items, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestJadwalPenimbanganBank_DiurutkanByTanggal(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwals := []models.Jadwal{
		jdwKhusus("KTBSU-0010002", jdwTanggal(2026, time.September, 25), "08:00", "11:00", "Akhir"),
		jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 3), "08:00", "11:00", "Awal"),
	}
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(jadwals, nil)
	repo.On("ListStatusSesi", mock.Anything, mock.Anything, mock.Anything).Return(map[string]string{}, nil)

	items, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	if assert.Len(t, items, 2) {
		assert.Equal(t, "2026-09-03", items[0].Tanggal)
		assert.Equal(t, "2026-09-25", items[1].Tanggal)
	}
}

func TestJadwalPenimbanganBank_NonaktifTanpaSesiTidakMuncul(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	rutin := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	rutin.IsActive = jdwBool(false)
	khusus := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "Timbang Massal")
	khusus.IsActive = jdwBool(false)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{rutin, khusus}, nil)
	repo.On("ListStatusSesi", mock.Anything, mock.Anything, mock.Anything).Return(map[string]string{}, nil)

	items, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestJadwalPenimbanganBank_NonaktifYangSudahAdaSesiTetapMuncul(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	rutin := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	rutin.IsActive = jdwBool(false)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{rutin}, nil)
	repo.On("ListStatusSesi", mock.Anything, mock.Anything, mock.Anything).
		Return(map[string]string{"RTBSU-0010001|2026-09-14": "selesai"}, nil)

	items, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	if assert.Len(t, items, 1) {
		assert.Equal(t, "2026-09-14", items[0].Tanggal)
		assert.False(t, items[0].IsActive)
		assert.Equal(t, "selesai", items[0].StatusJadwal)
	}
}

func TestJadwalPenimbanganBank_RentangSatuBulan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("ListStatusSesi", mock.Anything,
		jdwTanggal(2026, time.September, 1), jdwTanggal(2026, time.September, 30)).
		Return(map[string]string{}, nil)

	_, err := svc.JadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestMasterJadwal_BankIDKosong(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	hasil, err := svc.MasterJadwalPenimbanganBank("", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
}

func TestMasterJadwal_BankTidakDitemukan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-XXX").Return(nil, errors.New("not found"))

	hasil, err := svc.MasterJadwalPenimbanganBank("BSU-XXX", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, jdwKode(t, err))
}

func TestMasterJadwal_RepoError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(nil, errors.New("db error"))

	hasil, err := svc.MasterJadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestMasterJadwal_RutinTidakIkutDifilterBulan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwals := []models.Jadwal{
		jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00"),
		jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.October, 5), "08:00", "11:00", "Bulan depan"),
	}
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(jadwals, nil)

	hasil, err := svc.MasterJadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	assert.Len(t, hasil.Rutin, 1)
	assert.Len(t, hasil.Khusus, 0, "jadwal khusus bulan Oktober tidak muncul di filter September")
}

func TestMasterJadwal_KhususIkutSaatBulanCocok(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwals := []models.Jadwal{
		jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "Timbang Massal"),
	}
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(jadwals, nil)

	hasil, err := svc.MasterJadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	if assert.Len(t, hasil.Khusus, 1) {
		assert.Equal(t, "Timbang Massal", hasil.Khusus[0].NamaJadwal)
		assert.Equal(t, "2026-09-17", hasil.Khusus[0].Tanggal)
	}
}

func TestMasterJadwal_TanpaFilterSemuaKhususMuncul(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwals := []models.Jadwal{
		jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "September"),
		jdwKhusus("KTBSU-0010002", jdwTanggal(2026, time.October, 5), "08:00", "11:00", "Oktober"),
	}
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(jadwals, nil)

	hasil, err := svc.MasterJadwalPenimbanganBank("BSU-001", 0, 0)

	assert.NoError(t, err)
	assert.Len(t, hasil.Khusus, 2)
}

func TestMasterJadwal_KhususTanggalNilDilewati(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	rusak := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "Rusak")
	rusak.Tanggal = nil
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{rusak}, nil)

	hasil, err := svc.MasterJadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	assert.NotNil(t, hasil.Khusus)
	assert.Len(t, hasil.Khusus, 0)
}

func TestMasterJadwal_RutinTanpaPolaTidakPanik(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	rusak := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	rusak.Hari = nil
	rusak.MingguKe = nil
	repo.On("FindBank", "BSU-001").Return(jdwBank("BSU-001", models.BSU), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{rusak}, nil)

	hasil, err := svc.MasterJadwalPenimbanganBank("BSU-001", 9, 2026)

	assert.NoError(t, err)
	if assert.Len(t, hasil.Rutin, 1) {
		assert.Equal(t, models.HariEnum(""), hasil.Rutin[0].Hari)
		assert.Equal(t, 0, hasil.Rutin[0].MingguKe)
	}
}

func TestSemuaJadwal_TypeTidakValid(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	hasil, err := svc.SemuaJadwal("semua")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "type")
}

func TestSemuaJadwal_RepoError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("ListSemuaJadwal", models.JadwalPenimbangan).Return(nil, errors.New("db error"))

	hasil, err := svc.SemuaJadwal("penimbangan")

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestSemuaJadwal_PenimbanganDikelompokkanPerJenisBank(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	rows := []repositories.JadwalLengkapRow{
		{JadwalID: "RTBSI-0010001", BankID: "BSI-001", NamaBank: "BSI Utama", JenisBank: models.BSI, IsRutin: jdwBool(true)},
		{JadwalID: "KTBSU-0010001", BankID: "BSU-001", NamaBank: "BSU Mawar", JenisBank: models.BSU, IsRutin: jdwBool(false)},
		{JadwalID: "RTBSM-0010001", BankID: "BSM-001", NamaBank: "BSM Induk", JenisBank: models.BSM, IsRutin: jdwBool(true)},
	}
	repo.On("ListSemuaJadwal", models.JadwalPenimbangan).Return(rows, nil)

	hasil, err := svc.SemuaJadwal("penimbangan")

	assert.NoError(t, err)
	rekap, ok := hasil.(*services.SemuaJadwalPenimbanganResult)
	if assert.True(t, ok) {
		assert.Len(t, rekap.BSI, 1)
		assert.Len(t, rekap.BSM, 1)
		if assert.Len(t, rekap.BSU, 1) {
			assert.Len(t, rekap.BSU[0].TidakRutin, 1, "is_rutin false masuk ke tidak_rutin")
			assert.Len(t, rekap.BSU[0].Rutin, 0)
		}
		assert.Len(t, rekap.BSI[0].Rutin, 1)
	}
}

func TestSemuaJadwal_JenisBankTidakDikenalDilewati(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	rows := []repositories.JadwalLengkapRow{
		{JadwalID: "RTXXX-0010001", BankID: "XXX-001", JenisBank: "koperasi", IsRutin: jdwBool(true)},
	}
	repo.On("ListSemuaJadwal", models.JadwalPenimbangan).Return(rows, nil)

	hasil, err := svc.SemuaJadwal("penimbangan")

	assert.NoError(t, err)
	rekap := hasil.(*services.SemuaJadwalPenimbanganResult)
	assert.Len(t, rekap.BSI, 0)
	assert.Len(t, rekap.BSM, 0)
	assert.Len(t, rekap.BSU, 0)
}

func TestSemuaJadwal_PengangkutanDikelompokkanPerRute(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	rows := []repositories.JadwalLengkapRow{
		{JadwalID: "RABSU-0020001", BankID: "BSI-001", NamaBank: "BSI Utama",
			TargetBankID: "BSU-002", TargetBankName: "BSU Mawar", IsRutin: jdwBool(true)},
		{JadwalID: "RABSU-0030001", BankID: "BSI-001", NamaBank: "BSI Utama",
			TargetBankID: "BSU-003", TargetBankName: "BSU Melati", IsRutin: jdwBool(false)},
	}
	repo.On("ListSemuaJadwal", models.JadwalPengangkutan).Return(rows, nil)

	hasil, err := svc.SemuaJadwal("pengangkutan")

	assert.NoError(t, err)
	rute, ok := hasil.([]services.RuteBSIJadwal)
	if assert.True(t, ok) && assert.Len(t, rute, 1) {
		assert.Equal(t, "BSI-001", rute[0].BSIID)
		assert.Equal(t, "BSI Utama", rute[0].NamaBSI)
		assert.Len(t, rute[0].RuteBSU, 2, "dua BSU tujuan berbeda jadi dua rute")
	}
}

func TestUbahJadwal_JadwalIDKosong(t *testing.T) {
	_, _, svc := newJdwSvc(t)

	hasil, err := svc.UbahJadwal("", services.UbahJadwalInput{}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
}

func TestUbahJadwal_TidakDitemukan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(nil, gorm.ErrRecordNotFound)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusNotFound, jdwKode(t, err))
}

func TestUbahJadwal_ErrorDBLain(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(nil, errors.New("db error"))

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestUbahJadwal_BankLainDitolak(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)

	akses := jdwAkses()
	akses.ActorBankID = "BSU-999"
	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{IsActive: jdwBool(false)}, akses)

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusForbidden, jdwKode(t, err))
}

func TestUbahJadwal_SuperadminBolehLintasBank(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	akses := services.AksesJadwalInput{ActorUserID: "USR-SA", ActorBankID: "", ActorRole: models.SuperAdmin}
	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{IsActive: jdwBool(false)}, akses)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestUbahJadwal_BankTargetDiterima(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RABSU-0020001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	jadwal.BankID = "BSI-001"
	jadwal.JenisJadwal = models.JadwalPengangkutan
	jadwal.TargetBankID = ptrStr("BSU-002")
	repo.On("FindJadwal", mock.Anything, "RABSU-0020001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RABSU-0020001").Return(int64(0), nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	akses := jdwAkses()
	akses.ActorBankID = "BSU-002"
	hasil, err := svc.UbahJadwal("RABSU-0020001", services.UbahJadwalInput{IsActive: jdwBool(false)}, akses)

	assert.NoError(t, err)
	assert.NotNil(t, hasil)
}

func TestUbahJadwal_CekHistoriError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), errors.New("db error"))

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
	assert.Contains(t, err.Error(), "histori")
}

func TestUbahJadwal_AdaHistoriTolakUbahWaktuRutin(t *testing.T) {
	kasus := []struct {
		nama string
		in   services.UbahJadwalInput
	}{
		{"ubah hari", services.UbahJadwalInput{Hari: models.Kamis}},
		{"ubah minggu", services.UbahJadwalInput{MingguKe: 4}},
		{"ubah jam mulai", services.UbahJadwalInput{JamMulai: "09:00"}},
		{"ubah jam selesai", services.UbahJadwalInput{JamSelesai: "12:00"}},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			_, repo, svc := newJdwSvc(t)
			jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
			repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
			repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(3), nil)

			hasil, err := svc.UbahJadwal("RTBSU-0010001", k.in, jdwAkses())

			assert.Nil(t, hasil)
			assert.Equal(t, http.StatusConflict, jdwKode(t, err))
			assert.Contains(t, err.Error(), "histori penimbangan")
			repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
		})
	}
}

func TestUbahJadwal_AdaHistoriTolakUbahTanggalKhusus(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "Lama")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(1), nil)

	hasil, err := svc.UbahJadwal("KTBSU-0010001", services.UbahJadwalInput{Tanggal: "2026-09-18"}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
}

func TestUbahJadwal_AdaHistoriMasihBolehUbahIsActive(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(5), nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{IsActive: jdwBool(false)}, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil.IsActive) {
		assert.False(t, *hasil.IsActive)
	}
}

func TestUbahJadwal_HariTidakValid(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{Hari: "pahing"}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
}

func TestUbahJadwal_MingguKeTidakValid(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{MingguKe: 7}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
}

func TestUbahJadwal_JamMulaiSajaDivalidasiDenganJamLama(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{JamMulai: "12:00"}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "lebih besar")
}

func TestUbahJadwal_TanggalFormatSalah(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "Lama")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)

	hasil, err := svc.UbahJadwal("KTBSU-0010001", services.UbahJadwalInput{Tanggal: "18-09-2026"}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "YYYY-MM-DD")
}

func TestUbahJadwal_HariDiabaikanUntukJadwalKhusus(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "Lama")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	hasil, err := svc.UbahJadwal("KTBSU-0010001",
		services.UbahJadwalInput{Hari: models.Kamis, MingguKe: 2}, jdwAkses())

	assert.NoError(t, err)
	assert.Nil(t, hasil.Hari)
	assert.Nil(t, hasil.MingguKe)
}

func TestUbahJadwal_TanggalDanNamaDiabaikanUntukJadwalRutin(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001",
		services.UbahJadwalInput{Tanggal: "2026-09-18", NamaJadwalSpesial: "Baru"}, jdwAkses())

	assert.NoError(t, err)
	assert.Nil(t, hasil.Tanggal)
	assert.Equal(t, "", hasil.NamaJadwalSpesial)
}

func TestUbahJadwal_SaveError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(errors.New("db error"))

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{IsActive: jdwBool(false)}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestUbahJadwal_SuksesUbahPolaDanJam(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	asalCreatedBy := "ADM-LAMA"
	jadwal.CreatedBy = asalCreatedBy
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{
		Hari:       models.Kamis,
		MingguKe:   4,
		JamMulai:   "13:00",
		JamSelesai: "15:00",
	}, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, models.Kamis, *hasil.Hari)
		assert.Equal(t, 4, *hasil.MingguKe)
		assert.Equal(t, "13:00", hasil.JamMulai)
		assert.Equal(t, "15:00", hasil.JamSelesai)
		assert.Equal(t, asalCreatedBy, hasil.CreatedBy, "created_by tidak ditimpa saat update")
	}
}

func TestUbahJadwal_SuksesUbahKhusus(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00", "11:00", "Lama")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{}, nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	tanggalBaru := jdwBesok().Format("2006-01-02")
	hasil, err := svc.UbahJadwal("KTBSU-0010001", services.UbahJadwalInput{
		Tanggal:           tanggalBaru,
		NamaJadwalSpesial: "Timbang Massal",
	}, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, tanggalBaru, hasil.Tanggal.Format("2006-01-02"))
		assert.Equal(t, "Timbang Massal", hasil.NamaJadwalSpesial)
	}
}

func TestUbahJadwal_RutinBentrokDenganRutin(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	existing := jdwRutinDari("RTBSU-0010002", jdwTanggal(2026, time.September, 14), "09:00", "16:00")
	sabtu, minggu4 := models.Sabtu, 4
	existing.Hari, existing.MingguKe = &sabtu, &minggu4
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{jadwal, existing}, nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{
		Hari: models.Sabtu, MingguKe: 4, JamMulai: "10:00", JamSelesai: "17:00",
	}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
	assert.Contains(t, err.Error(), "bentrok")
	repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
}

func TestUbahJadwal_RutinBentrokDenganKhusus(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	khusus := jdwKhusus("KTBSU-0010001", jdwBesok(), "09:00", "12:00", "Timbang Massal")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{jadwal, khusus}, nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{
		Hari: utils.HariOf(jdwBesok()), MingguKe: utils.WeekOfMonth(jdwBesok()),
		JamMulai: "09:30", JamSelesai: "11:30",
	}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
	repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
}

func TestUbahJadwal_KhususBentrokDenganRutin(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	lusa := jdwBesok().AddDate(0, 0, 1)
	jadwal := jdwKhusus("KTBSU-0010001", jdwBesok(), "08:00", "11:00", "Lama")
	rutin := jdwRutinDari("RTBSU-0010001", lusa, "10:00", "12:00")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{jadwal, rutin}, nil)

	hasil, err := svc.UbahJadwal("KTBSU-0010001",
		services.UbahJadwalInput{Tanggal: lusa.Format("2006-01-02")}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
	repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
}

func TestUbahJadwal_KhususBentrokDenganKhusus(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	lusa := jdwBesok().AddDate(0, 0, 1)
	jadwal := jdwKhusus("KTBSU-0010001", jdwBesok(), "08:00", "11:00", "Lama")
	lain := jdwKhusus("KTBSU-0010002", lusa, "09:00", "12:00", "Lain")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{jadwal, lain}, nil)

	hasil, err := svc.UbahJadwal("KTBSU-0010001",
		services.UbahJadwalInput{Tanggal: lusa.Format("2006-01-02")}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
	repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
}

func TestUbahJadwal_TidakBentrokDenganDirinyaSendiri(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return([]models.Jadwal{jadwal}, nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{JamSelesai: "12:00"}, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "12:00", hasil.JamSelesai)
	}
}

func TestUbahJadwal_KhususTanggalSudahLewat(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwKhusus("KTBSU-0010001", jdwBesok(), "08:00", "11:00", "Lama")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)

	kemarin := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	hasil, err := svc.UbahJadwal("KTBSU-0010001", services.UbahJadwalInput{Tanggal: kemarin}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "minimal besok")
	repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
}

func TestUbahJadwal_KhususDiubahKeHariIniDitolak(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwKhusus("KTBSU-0010001", jdwBesok(), "21:00", "23:00", "Lama")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)

	hariIni := time.Now().Format("2006-01-02")
	hasil, err := svc.UbahJadwal("KTBSU-0010001", services.UbahJadwalInput{Tanggal: hariIni}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "minimal besok")
	repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
}

func TestUbahJadwal_KhususHariIniUbahJamDitolak(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	hariIni := jdwTanggal(time.Now().Year(), time.Now().Month(), time.Now().Day())
	jadwal := jdwKhusus("KTBSU-0010001", hariIni, "08:00:00", "11:00:00", "Lama")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)

	hasil, err := svc.UbahJadwal("KTBSU-0010001", services.UbahJadwalInput{JamSelesai: "12:00"}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusBadRequest, jdwKode(t, err))
	assert.Contains(t, err.Error(), "minimal besok")
	repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
}

func TestUbahJadwal_NilaiSamaTidakDicekUlang(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwKhusus("KTBSU-0010001", jdwTanggal(2026, time.September, 17), "08:00:00", "11:00:00", "Lama")
	repo.On("FindJadwal", mock.Anything, "KTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "KTBSU-0010001").Return(int64(0), nil)
	repo.On("SaveJadwal", mock.Anything, mock.Anything).Return(nil)

	hasil, err := svc.UbahJadwal("KTBSU-0010001", services.UbahJadwalInput{
		Tanggal: "2026-09-17", JamMulai: "08:00", JamSelesai: "11:00", NamaJadwalSpesial: "Baru",
	}, jdwAkses())

	assert.NoError(t, err)
	if assert.NotNil(t, hasil) {
		assert.Equal(t, "Baru", hasil.NamaJadwalSpesial)
	}
	repo.AssertNotCalled(t, "ListJadwalBank", mock.Anything, mock.Anything)
}

func TestUbahJadwal_ListJadwalError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountSesiPenimbangan", "RTBSU-0010001").Return(int64(0), nil)
	repo.On("ListJadwalBank", "BSU-001", models.JadwalPenimbangan).Return(nil, errors.New("db error"))

	hasil, err := svc.UbahJadwal("RTBSU-0010001", services.UbahJadwalInput{JamSelesai: "12:00"}, jdwAkses())

	assert.Nil(t, hasil)
	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
	repo.AssertNotCalled(t, "SaveJadwal", mock.Anything, mock.Anything)
}

func TestHapusJadwal_TidakDitemukan(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(nil, gorm.ErrRecordNotFound)

	err := svc.HapusJadwal("RTBSU-0010001", jdwAkses())

	assert.Equal(t, http.StatusNotFound, jdwKode(t, err))
}

func TestHapusJadwal_BankLainDitolak(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)

	akses := jdwAkses()
	akses.ActorBankID = "BSU-999"
	err := svc.HapusJadwal("RTBSU-0010001", akses)

	assert.Equal(t, http.StatusForbidden, jdwKode(t, err))
	repo.AssertNotCalled(t, "DeleteJadwal", mock.Anything, mock.Anything)
}

func TestHapusJadwal_CekPemakaianError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountPemakaianJadwal", "RTBSU-0010001").Return(nil, errors.New("db error"))

	err := svc.HapusJadwal("RTBSU-0010001", jdwAkses())

	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestHapusJadwal_MasihDipakai(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountPemakaianJadwal", "RTBSU-0010001").Return([]repositories.PemakaianJadwal{
		{Label: "sesi penimbangan", Jumlah: 3},
		{Label: "pengangkutan sampah", Jumlah: 1},
	}, nil)

	err := svc.HapusJadwal("RTBSU-0010001", jdwAkses())

	assert.Equal(t, http.StatusConflict, jdwKode(t, err))
	assert.Contains(t, err.Error(), "3 sesi penimbangan")
	assert.Contains(t, err.Error(), "1 pengangkutan sampah")
	repo.AssertNotCalled(t, "DeleteJadwal", mock.Anything, mock.Anything)
}

func TestHapusJadwal_DeleteError(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountPemakaianJadwal", "RTBSU-0010001").Return([]repositories.PemakaianJadwal{}, nil)
	repo.On("DeleteJadwal", mock.Anything, &jadwal).Return(errors.New("db error"))

	err := svc.HapusJadwal("RTBSU-0010001", jdwAkses())

	assert.Equal(t, http.StatusInternalServerError, jdwKode(t, err))
}

func TestHapusJadwal_Sukses(t *testing.T) {
	_, repo, svc := newJdwSvc(t)
	jadwal := jdwRutinDari("RTBSU-0010001", jdwTanggal(2026, time.September, 14), "08:00", "11:00")
	repo.On("FindJadwal", mock.Anything, "RTBSU-0010001").Return(&jadwal, nil)
	repo.On("CountPemakaianJadwal", "RTBSU-0010001").Return([]repositories.PemakaianJadwal{}, nil)
	repo.On("DeleteJadwal", mock.Anything, &jadwal).Return(nil)

	err := svc.HapusJadwal("RTBSU-0010001", jdwAkses())

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
