package services_test

import (
	"errors"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func setoranNasabahAktif() *models.Nasabah {
	return &models.Nasabah{
		NasabahID:     "NSB-001",
		BankID:        "BSI-001",
		UserID:        "USR-NSB-001",
		StatusNasabah: models.Aktif,
		User:          models.User{UserID: "USR-NSB-001", Nama: "Budi Nasabah"},
	}
}

func setoranAdmin() *models.Admin {
	bID := "BSI-001"
	return &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-ADM-001"}
}

func setoranPenimbangan() *models.Penimbangan {
	bID := "BSI-001"
	return &models.Penimbangan{
		PenimbanganID:     "PNB-001",
		BankID:            &bID,
		StatusPenimbangan: models.StatusAktif,
	}
}

func newSetoranSvc(t *testing.T) (sqlmock.Sqlmock, *mocks.MockSetoranRepo, *mocks.MockNotifSvc, services.SetoranService) {
	t.Helper()
	gdb, dbMock := newTestDB(t)
	repo := new(mocks.MockSetoranRepo)
	notif := new(mocks.MockNotifSvc)
	svc := services.NewSetoranService(gdb, repo, notif)
	return dbMock, repo, notif, svc
}

// ── Verifikasi ────────────────────────────────────────────────────────────────

func TestVerifikasiSetoran_FormatQRTidakValid(t *testing.T) {
	_, _, _, svc := newSetoranSvc(t)

	result := svc.Verifikasi("bukan-json", "ADM-001")

	assert.Equal(t, "unverified", result.Status)
	assert.Contains(t, result.Message, "tidak valid")
}

func TestVerifikasiSetoran_TypeBukanEnvirooSetoran(t *testing.T) {
	_, _, _, svc := newSetoranSvc(t)

	qr := `{"type":"ENVIROO-LAIN","nasabah_id":"NSB-001","penimbangan_id":"PNB-001"}`
	result := svc.Verifikasi(qr, "ADM-001")

	assert.Equal(t, "unverified", result.Status)
	assert.Contains(t, result.Message, "bukan milik aplikasi ini")
}

func TestVerifikasiSetoran_NasabahTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	qr := `{"type":"ENVIROO-SETORAN","nasabah_id":"NSB-XXX","penimbangan_id":"PNB-001"}`
	repo.On("FindNasabahAktifWithUser", "NSB-XXX").Return(nil, errors.New("not found"))

	result := svc.Verifikasi(qr, "ADM-001")

	assert.Equal(t, "unverified", result.Status)
	assert.Contains(t, result.Message, "Nasabah")
}

func TestVerifikasiSetoran_AdminTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	qr := `{"type":"ENVIROO-SETORAN","nasabah_id":"NSB-001","penimbangan_id":"PNB-001"}`
	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindAdminAktif", "ADM-XXX").Return(nil, errors.New("not found"))

	result := svc.Verifikasi(qr, "ADM-XXX")

	assert.Equal(t, "unverified", result.Status)
	assert.Contains(t, result.Message, "Admin")
}

func TestVerifikasiSetoran_ConflictOfInterest(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	qr := `{"type":"ENVIROO-SETORAN","nasabah_id":"NSB-001","penimbangan_id":"PNB-001"}`
	nasabah := setoranNasabahAktif()
	nasabah.UserID = "USR-SAMA"
	bID := "BSI-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-SAMA"}
	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(nasabah, nil)
	repo.On("FindAdminAktif", "ADM-001").Return(admin, nil)

	result := svc.Verifikasi(qr, "ADM-001")

	assert.Equal(t, "unverified", result.Status)
	assert.Contains(t, result.Message, "sendiri")
}

func TestVerifikasiSetoran_PenimbanganTidakAktif(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	qr := `{"type":"ENVIROO-SETORAN","nasabah_id":"NSB-001","penimbangan_id":"PNB-XXX"}`
	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindAdminAktif", "ADM-001").Return(setoranAdmin(), nil)
	repo.On("FindPenimbanganAktif", "PNB-XXX").Return(nil, errors.New("not found"))

	result := svc.Verifikasi(qr, "ADM-001")

	assert.Equal(t, "unverified", result.Status)
	assert.Contains(t, result.Message, "penimbangan")
}

func TestVerifikasiSetoran_BankIDPenimbanganNil(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	qr := `{"type":"ENVIROO-SETORAN","nasabah_id":"NSB-001","penimbangan_id":"PNB-001"}`
	penimbangan := &models.Penimbangan{PenimbanganID: "PNB-001", BankID: nil}
	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindAdminAktif", "ADM-001").Return(setoranAdmin(), nil)
	repo.On("FindPenimbanganAktif", "PNB-001").Return(penimbangan, nil)

	result := svc.Verifikasi(qr, "ADM-001")

	assert.Equal(t, "unverified", result.Status)
	assert.Contains(t, result.Message, "bank sampah ini")
}

func TestVerifikasiSetoran_BankMismatch(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	qr := `{"type":"ENVIROO-SETORAN","nasabah_id":"NSB-001","penimbangan_id":"PNB-001"}`
	otherBank := "BSI-002"
	penimbangan := &models.Penimbangan{PenimbanganID: "PNB-001", BankID: &otherBank}
	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil) // BankID = BSI-001
	repo.On("FindAdminAktif", "ADM-001").Return(setoranAdmin(), nil)
	repo.On("FindPenimbanganAktif", "PNB-001").Return(penimbangan, nil)

	result := svc.Verifikasi(qr, "ADM-001")

	assert.Equal(t, "unverified", result.Status)
	assert.Contains(t, result.Message, "bank sampah ini")
}

func TestVerifikasiSetoran_Sukses(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	qr := `{"type":"ENVIROO-SETORAN","nasabah_id":"NSB-001","penimbangan_id":"PNB-001"}`
	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindAdminAktif", "ADM-001").Return(setoranAdmin(), nil)
	repo.On("FindPenimbanganAktif", "PNB-001").Return(setoranPenimbangan(), nil)

	result := svc.Verifikasi(qr, "ADM-001")

	assert.Equal(t, "verified", result.Status)
	assert.NotNil(t, result.Data)
	data := result.Data.(map[string]interface{})
	assert.Equal(t, "NSB-001", data["nasabah_id"])
	assert.Equal(t, "Budi Nasabah", data["nama_nasabah"])
}

// ── Preview ───────────────────────────────────────────────────────────────────

func TestPreviewSetoran_NasabahTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	repo.On("FindNasabahWithUser", "NSB-XXX").Return(nil, errors.New("not found"))

	result, err := svc.Preview("PNB-001", "NSB-XXX", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Nasabah")
}

func TestPreviewSetoran_QtyNol(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	repo.On("FindNasabahWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)

	result, err := svc.Preview("PNB-001", "NSB-001", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: 0}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Qty")
}

func TestPreviewSetoran_QtyNegatif(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	repo.On("FindNasabahWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)

	result, err := svc.Preview("PNB-001", "NSB-001", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: -0.5}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Qty")
}

func TestPreviewSetoran_KatalogTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	repo.On("FindNasabahWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindKatalogSampah", "SMP-XXX").Return(nil, errors.New("not found"))

	result, err := svc.Preview("PNB-001", "NSB-001", []services.ItemSetoranReq{{SampahID: "SMP-XXX", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Sampah tidak ditemukan")
}

func TestPreviewSetoran_Sukses_SingleItem(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	katalog := &models.KatalogSampah{
		SampahID: "SMP-001",
		Sarok:    models.Sampah{NamaSampah: "Botol Plastik", Satuan: models.SatuanKG},
		Reward:   models.Reward{NamaReward: models.RewardEnumUang},
	}
	repo.On("FindNasabahWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindKatalogSampah", "SMP-001").Return(katalog, nil)

	result, err := svc.Preview("PNB-001", "NSB-001", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: 2.5}})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "PNB-001", result.PenimbanganID)
	assert.Equal(t, "NSB-001", result.NasabahID)
	assert.Equal(t, "Budi Nasabah", result.NamaNasabah)
	assert.Equal(t, 1, result.TotalItem)
	assert.Len(t, result.Items, 1)
	assert.Equal(t, "SMP-001", result.Items[0].SampahID)
	assert.Equal(t, "Botol Plastik", result.Items[0].NamaSampah)
	assert.Equal(t, float64(2.5), result.Items[0].Qty)
	assert.Equal(t, "kg", result.Items[0].Satuan)
}

func TestPreviewSetoran_Sukses_MultipleItem(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	katalog1 := &models.KatalogSampah{SampahID: "SMP-001", Sarok: models.Sampah{NamaSampah: "Botol Plastik", Satuan: models.SatuanKG}, Reward: models.Reward{NamaReward: models.RewardEnumUang}}
	katalog2 := &models.KatalogSampah{SampahID: "SMP-002", Sarok: models.Sampah{NamaSampah: "Kertas", Satuan: models.SatuanKG}, Reward: models.Reward{NamaReward: models.RewardEnumSembako}}

	repo.On("FindNasabahWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindKatalogSampah", "SMP-001").Return(katalog1, nil)
	repo.On("FindKatalogSampah", "SMP-002").Return(katalog2, nil)

	result, err := svc.Preview("PNB-001", "NSB-001", []services.ItemSetoranReq{
		{SampahID: "SMP-001", Qty: 1},
		{SampahID: "SMP-002", Qty: 3},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 2, result.TotalItem)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, string(models.RewardEnumSembako), result.Items[1].JenisReward)
}

// ── Input ─────────────────────────────────────────────────────────────────────

func TestInputSetoran_NasabahTidakAktif(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	repo.On("FindNasabahAktifWithUser", "NSB-XXX").Return(nil, errors.New("not found"))

	result, err := svc.Input("PNB-001", "NSB-XXX", "ADM-001", "manual", "", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Nasabah")
}

func TestInputSetoran_AdminTidakAktif(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindAdminAktif", "ADM-XXX").Return(nil, errors.New("not found"))

	result, err := svc.Input("PNB-001", "NSB-001", "ADM-XXX", "manual", "", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Admin")
}

func TestInputSetoran_ConflictOfInterest(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	nasabah := setoranNasabahAktif()
	nasabah.UserID = "USR-SAMA"
	bID := "BSI-001"
	admin := &models.Admin{AdminID: "ADM-001", BankID: &bID, UserID: "USR-SAMA"}

	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(nasabah, nil)
	repo.On("FindAdminAktif", "ADM-001").Return(admin, nil)

	result, err := svc.Input("PNB-001", "NSB-001", "ADM-001", "manual", "", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sendiri")
}

func TestInputSetoran_PenimbanganTidakDitemukan(t *testing.T) {
	dbMock, repo, _, svc := newSetoranSvc(t)

	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindAdminAktif", "ADM-001").Return(setoranAdmin(), nil)
	repo.On("FindPenimbanganTx", mock.Anything, "PNB-XXX").Return(nil, errors.New("not found"))

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Input("PNB-XXX", "NSB-001", "ADM-001", "manual", "", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "penimbangan")
}

func TestInputSetoran_NasabahBedaBank(t *testing.T) {
	dbMock, repo, _, svc := newSetoranSvc(t)

	otherBank := "BSI-002"
	penimbangan := &models.Penimbangan{PenimbanganID: "PNB-001", BankID: &otherBank}

	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil) // BankID = BSI-001
	repo.On("FindAdminAktif", "ADM-001").Return(setoranAdmin(), nil)
	repo.On("FindPenimbanganTx", mock.Anything, "PNB-001").Return(penimbangan, nil)

	dbMock.ExpectBegin()
	dbMock.ExpectRollback()

	result, err := svc.Input("PNB-001", "NSB-001", "ADM-001", "manual", "", []services.ItemSetoranReq{{SampahID: "SMP-001", Qty: 1}})

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bank sampah ini")
}

func TestInputSetoran_Sukses_SingleItem(t *testing.T) {
	dbMock, repo, notif, svc := newSetoranSvc(t)

	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindAdminAktif", "ADM-001").Return(setoranAdmin(), nil)
	repo.On("FindPenimbanganTx", mock.Anything, "PNB-001").Return(setoranPenimbangan(), nil)
	repo.On("CreateSetoran", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailSetoran", mock.Anything, mock.Anything).Return(nil)
	repo.On("UpsertStokSampah", mock.Anything, "BSI-001", "SMP-001", float64(2)).Return(nil)
	repo.On("CreateTabunganSampah", mock.Anything, mock.Anything).Return(nil)
	repo.On("FindBankSampah", "BSI-001").Return(&models.BankSampah{BankID: "BSI-001", NamaBank: "BSI Utama"}, nil).Maybe()
	repo.On("FindNasabahWithUser", "NSB-001").Return(setoranNasabahAktif(), nil).Maybe()
	notif.On("NotifSetoranBerhasil", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	result, err := svc.Input("PNB-001", "NSB-001", "ADM-001", "manual", "https://bukti.jpg", []services.ItemSetoranReq{
		{SampahID: "SMP-001", Qty: 2},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.SetoranID)
	assert.Equal(t, 1, result.TotalItem)
	repo.AssertExpectations(t)
}

func TestInputSetoran_Sukses_MultiItem(t *testing.T) {
	dbMock, repo, notif, svc := newSetoranSvc(t)

	repo.On("FindNasabahAktifWithUser", "NSB-001").Return(setoranNasabahAktif(), nil)
	repo.On("FindAdminAktif", "ADM-001").Return(setoranAdmin(), nil)
	repo.On("FindPenimbanganTx", mock.Anything, "PNB-001").Return(setoranPenimbangan(), nil)
	repo.On("CreateSetoran", mock.Anything, mock.Anything).Return(nil)
	repo.On("CreateDetailSetoran", mock.Anything, mock.Anything).Return(nil).Times(2)
	repo.On("UpsertStokSampah", mock.Anything, "BSI-001", "SMP-001", float64(1)).Return(nil)
	repo.On("UpsertStokSampah", mock.Anything, "BSI-001", "SMP-002", float64(3)).Return(nil)
	repo.On("CreateTabunganSampah", mock.Anything, mock.Anything).Return(nil).Times(2)
	repo.On("FindBankSampah", "BSI-001").Return(&models.BankSampah{BankID: "BSI-001", NamaBank: "BSI Utama"}, nil).Maybe()
	repo.On("FindNasabahWithUser", "NSB-001").Return(setoranNasabahAktif(), nil).Maybe()
	notif.On("NotifSetoranBerhasil", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	dbMock.ExpectBegin()
	dbMock.ExpectCommit()

	result, err := svc.Input("PNB-001", "NSB-001", "ADM-001", "manual", "", []services.ItemSetoranReq{
		{SampahID: "SMP-001", Qty: 1},
		{SampahID: "SMP-002", Qty: 3},
	})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.SetoranID)
	assert.Equal(t, 2, result.TotalItem)
	repo.AssertExpectations(t)
}

// ── GetDetail ─────────────────────────────────────────────────────────────────

func TestGetDetailSetoran_HeaderTidakDitemukan(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	repo.On("GetDetailHeader", "STR-XXX").Return(nil, errors.New("not found"))

	result, err := svc.GetDetail("STR-XXX")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "setoran")
}

func TestGetDetailSetoran_ItemsGagal(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	header := &repositories.SetoranDetailHeader{SetoranID: "STR-001", NamaNasabah: "Budi Nasabah"}
	repo.On("GetDetailHeader", "STR-001").Return(header, nil)
	repo.On("GetDetailItems", "STR-001").Return(nil, errors.New("db error"))

	result, err := svc.GetDetail("STR-001")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "detail item")
}

func TestGetDetailSetoran_Sukses(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	header := &repositories.SetoranDetailHeader{
		SetoranID:   "STR-001",
		NamaNasabah: "Budi Nasabah",
		NamaPetugas: "Petugas A",
		TotalItem:   2,
	}
	items := []repositories.SetoranDetailItem{
		{NamaSampah: "Botol Plastik", Satuan: "kg", Qty: 2},
		{NamaSampah: "Kertas", Satuan: "kg", Qty: 1.5},
	}
	repo.On("GetDetailHeader", "STR-001").Return(header, nil)
	repo.On("GetDetailItems", "STR-001").Return(items, nil)

	result, err := svc.GetDetail("STR-001")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "STR-001", result.Header.SetoranID)
	assert.Equal(t, "Budi Nasabah", result.Header.NamaNasabah)
	assert.Equal(t, 2, result.Header.TotalItem)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "Botol Plastik", result.Items[0].NamaSampah)
	assert.Equal(t, float64(2), result.Items[0].Qty)
}

// ── GetRiwayat ────────────────────────────────────────────────────────────────

func TestGetRiwayatSetoran_Error(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	repo.On("GetRiwayat", "NSB-XXX", "", "").Return(nil, errors.New("db error"))

	result, err := svc.GetRiwayat("NSB-XXX", "", "")

	assert.Nil(t, result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "riwayat")
}

func TestGetRiwayatSetoran_Sukses(t *testing.T) {
	_, repo, _, svc := newSetoranSvc(t)

	rows := []repositories.RiwayatSetoranRow{
		{SetoranID: "STR-001", NamaPetugas: "Petugas A", TransaksiTimestamp: time.Now(), TotalItem: 3},
		{SetoranID: "STR-002", NamaPetugas: "Petugas B", TransaksiTimestamp: time.Now(), TotalItem: 1},
	}
	repo.On("GetRiwayat", "NSB-001", "", "").Return(rows, nil)

	result, err := svc.GetRiwayat("NSB-001", "", "")

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "STR-001", result[0].SetoranID)
	assert.Equal(t, "STR-002", result[1].SetoranID)
	assert.Equal(t, 3, result[0].TotalItem)
}
