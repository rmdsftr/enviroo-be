package services_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/services"
	"enviroo-be/internal/services/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const (
	ntfUser  = "1371010101010001"
	ntfToken = "fcm-token-abc"
	ntfRef   = "REF-001"
)

func newNtfSvc(t *testing.T) (*mocks.MockNotifikasiRepo, services.NotifikasiService) {
	t.Helper()
	repo := new(mocks.MockNotifikasiRepo)
	return repo, services.NewNotifikasiService(repo, nil)
}

func ntfTangkap(repo *mocks.MockNotifikasiRepo) **models.Notifikasi {
	var tersimpan *models.Notifikasi
	repo.On("Create", mock.Anything).
		Run(func(args mock.Arguments) { tersimpan = args.Get(0).(*models.Notifikasi) }).Return(nil)
	return &tersimpan
}

func ntfNotif(id string, dibaca bool) models.Notifikasi {
	return models.Notifikasi{
		NotifikasiID: id, UserID: ntfUser, RoleTarget: "nasabah",
		Judul: "Judul", Pesan: "Pesan", IsRead: dibaca,
		CreatedAt: time.Date(2026, time.September, 1, 8, 0, 0, 0, time.Local),
	}
}

func TestKirimNotif_CreateError(t *testing.T) {
	repo, svc := newNtfSvc(t)
	repo.On("Create", mock.Anything).Return(errors.New("db error"))

	err := svc.Kirim(context.Background(), services.KirimNotifRequest{
		UserID: ntfUser, RoleTarget: "nasabah", Judul: "Judul", Pesan: "Pesan",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gagal simpan notifikasi")
}

func TestKirimNotif_FieldTersimpan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)

	refID, refType := ntfRef, "setoran"
	err := svc.Kirim(context.Background(), services.KirimNotifRequest{
		UserID: ntfUser, FCMToken: ntfToken, RoleTarget: "admin",
		Judul: "Judul Uji", Pesan: "Pesan Uji", RefID: &refID, RefType: &refType,
	})

	assert.NoError(t, err)
	if assert.NotNil(t, *tersimpan) {
		n := *tersimpan
		assert.Equal(t, ntfUser, n.UserID)
		assert.Equal(t, "admin", n.RoleTarget)
		assert.Equal(t, "Judul Uji", n.Judul)
		assert.Equal(t, "Pesan Uji", n.Pesan)
		assert.False(t, n.IsRead, "notifikasi baru selalu belum dibaca")
		assert.NotEmpty(t, n.NotifikasiID, "notifikasi_id digenerate sebagai UUID")
		assert.Len(t, n.NotifikasiID, 36)
		assert.WithinDuration(t, time.Now(), n.CreatedAt, time.Minute)
		if assert.NotNil(t, n.RefID) && assert.NotNil(t, n.RefType) {
			assert.Equal(t, ntfRef, *n.RefID)
			assert.Equal(t, "setoran", *n.RefType)
		}
	}
}

func TestKirimNotif_IDUnikTiapKiriman(t *testing.T) {
	repo, svc := newNtfSvc(t)
	var ids []string
	repo.On("Create", mock.Anything).
		Run(func(args mock.Arguments) {
			ids = append(ids, args.Get(0).(*models.Notifikasi).NotifikasiID)
		}).Return(nil)

	req := services.KirimNotifRequest{UserID: ntfUser, RoleTarget: "nasabah", Judul: "J", Pesan: "P"}
	assert.NoError(t, svc.Kirim(context.Background(), req))
	assert.NoError(t, svc.Kirim(context.Background(), req))

	if assert.Len(t, ids, 2) {
		assert.NotEqual(t, ids[0], ids[1])
	}
}

func TestKirimNotif_TanpaFCMTetapTersimpan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	repo.On("Create", mock.Anything).Return(nil)

	err := svc.Kirim(context.Background(), services.KirimNotifRequest{
		UserID: ntfUser, FCMToken: ntfToken, RoleTarget: "nasabah", Judul: "J", Pesan: "P",
	})

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestKirimNotif_TanpaRefBolehNil(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)

	err := svc.Kirim(context.Background(), services.KirimNotifRequest{
		UserID: ntfUser, RoleTarget: "nasabah", Judul: "J", Pesan: "P",
	})

	assert.NoError(t, err)
	assert.Nil(t, (*tersimpan).RefID)
	assert.Nil(t, (*tersimpan).RefType)
}

func TestGetNotifByUser_RepoError(t *testing.T) {
	repo, svc := newNtfSvc(t)
	repo.On("FindByUserID", ntfUser, "nasabah", 20, 0).Return(nil, int64(0), errors.New("db error"))

	hasil, err := svc.GetByUser(context.Background(), ntfUser, "nasabah", 1, 20)

	assert.Nil(t, hasil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gagal ambil notifikasi")
}

func TestGetNotifByUser_NormalisasiPageLimit(t *testing.T) {
	kasus := []struct {
		nama                               string
		page, limit                        int
		wantLimit, wantOffset, wantHalaman int
	}{
		{"page nol jadi satu", 0, 20, 20, 0, 1},
		{"page negatif jadi satu", -4, 20, 20, 0, 1},
		{"limit nol jadi dua puluh", 1, 0, 20, 0, 1},
		{"limit negatif jadi dua puluh", 1, -5, 20, 0, 1},
		{"halaman ketiga", 3, 10, 10, 20, 3},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			repo, svc := newNtfSvc(t)
			repo.On("FindByUserID", ntfUser, "nasabah", k.wantLimit, k.wantOffset).
				Return([]models.Notifikasi{}, int64(0), nil)

			hasil, err := svc.GetByUser(context.Background(), ntfUser, "nasabah", k.page, k.limit)

			assert.NoError(t, err)
			if assert.NotNil(t, hasil) {
				assert.Equal(t, k.wantHalaman, hasil.Page)
				assert.Equal(t, k.wantLimit, hasil.Limit)
			}
			repo.AssertExpectations(t)
		})
	}
}

func TestGetNotifByUser_TotalHalaman(t *testing.T) {
	kasus := []struct {
		total       int64
		limit, want int
	}{
		{0, 20, 0},
		{1, 20, 1},
		{20, 20, 1},
		{21, 20, 2},
		{45, 20, 3},
	}
	for _, k := range kasus {
		repo, svc := newNtfSvc(t)
		repo.On("FindByUserID", ntfUser, "nasabah", k.limit, 0).
			Return([]models.Notifikasi{}, k.total, nil)

		hasil, err := svc.GetByUser(context.Background(), ntfUser, "nasabah", 1, k.limit)

		assert.NoError(t, err)
		assert.Equal(t, k.want, hasil.TotalHalaman, "total %d, limit %d", k.total, k.limit)
		assert.Equal(t, k.total, hasil.Total)
	}
}

func TestGetNotifByUser_DataDiteruskan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	data := []models.Notifikasi{ntfNotif("NTF-001", false), ntfNotif("NTF-002", true)}
	repo.On("FindByUserID", ntfUser, "admin", 20, 0).Return(data, int64(2), nil)

	hasil, err := svc.GetByUser(context.Background(), ntfUser, "admin", 1, 20)

	assert.NoError(t, err)
	if assert.Len(t, hasil.Data, 2) {
		assert.Equal(t, "NTF-001", hasil.Data[0].NotifikasiID)
		assert.True(t, hasil.Data[1].IsRead)
	}
}

func TestBacaNotif(t *testing.T) {
	t.Run("sukses", func(t *testing.T) {
		repo, svc := newNtfSvc(t)
		repo.On("MarkAsRead", "NTF-001", ntfUser).Return(nil)

		assert.NoError(t, svc.BacaNotif(context.Background(), "NTF-001", ntfUser))
		repo.AssertExpectations(t)
	})

	t.Run("error diteruskan", func(t *testing.T) {
		repo, svc := newNtfSvc(t)
		repo.On("MarkAsRead", "NTF-001", ntfUser).Return(errors.New("db error"))

		assert.Error(t, svc.BacaNotif(context.Background(), "NTF-001", ntfUser))
	})
}

func TestBacaSemuaNotif(t *testing.T) {
	t.Run("sukses", func(t *testing.T) {
		repo, svc := newNtfSvc(t)
		repo.On("MarkAllAsRead", ntfUser, "nasabah").Return(nil)

		assert.NoError(t, svc.BacaSemua(context.Background(), ntfUser, "nasabah"))
		repo.AssertExpectations(t)
	})

	t.Run("error diteruskan", func(t *testing.T) {
		repo, svc := newNtfSvc(t)
		repo.On("MarkAllAsRead", ntfUser, "nasabah").Return(errors.New("db error"))

		assert.Error(t, svc.BacaSemua(context.Background(), ntfUser, "nasabah"))
	})
}

func TestJumlahBelumDibaca(t *testing.T) {
	t.Run("sukses", func(t *testing.T) {
		repo, svc := newNtfSvc(t)
		repo.On("CountUnread", ntfUser, "admin").Return(int64(7), nil)

		jumlah, err := svc.JumlahBelumDibaca(context.Background(), ntfUser, "admin")

		assert.NoError(t, err)
		assert.Equal(t, int64(7), jumlah)
	})

	t.Run("error diteruskan", func(t *testing.T) {
		repo, svc := newNtfSvc(t)
		repo.On("CountUnread", ntfUser, "admin").Return(int64(0), errors.New("db error"))

		_, err := svc.JumlahBelumDibaca(context.Background(), ntfUser, "admin")

		assert.Error(t, err)
	})
}

func TestHapusNotif(t *testing.T) {
	t.Run("sukses", func(t *testing.T) {
		repo, svc := newNtfSvc(t)
		repo.On("Delete", "NTF-001", ntfUser).Return(nil)

		assert.NoError(t, svc.Hapus(context.Background(), "NTF-001", ntfUser))
		repo.AssertExpectations(t)
	})

	t.Run("error diteruskan", func(t *testing.T) {
		repo, svc := newNtfSvc(t)
		repo.On("Delete", "NTF-001", ntfUser).Return(errors.New("db error"))

		assert.Error(t, svc.Hapus(context.Background(), "NTF-001", ntfUser))
	})
}

func TestNotifHelper_RoleTargetDanRefType(t *testing.T) {
	ctx := context.Background()
	tanggal := time.Date(2026, time.September, 17, 0, 0, 0, 0, time.Local)

	kasus := []struct {
		nama        string
		panggil     func(svc services.NotifikasiService) error
		wantRole    string
		wantRefType string
	}{
		{"setoran berhasil", func(s services.NotifikasiService) error {
			return s.NotifSetoranBerhasil(ctx, ntfUser, ntfToken, 3, "BSU Mawar", ntfRef)
		}, "nasabah", "setoran"},

		{"bagi hasil admin", func(s services.NotifikasiService) error {
			return s.NotifBagiHasilDiterima(ctx, ntfUser, ntfToken, "Rp50000", "BSU Mawar", "BSI Utama", ntfRef)
		}, "admin", "bagi_hasil"},

		{"bagi hasil nasabah", func(s services.NotifikasiService) error {
			return s.NotifBagiHasilNasabahDiterima(ctx, ntfUser, ntfToken, "Rp50000", ntfRef)
		}, "nasabah", "bagi_hasil"},

		{"penarikan berhasil", func(s services.NotifikasiService) error {
			return s.NotifPenarikanBerhasil(ctx, ntfUser, ntfToken, "Rp50000", ntfRef)
		}, "nasabah", "penarikan"},

		{"penarikan ditolak", func(s services.NotifikasiService) error {
			return s.NotifPenarikanDitolak(ctx, ntfUser, ntfToken, ntfRef)
		}, "nasabah", "penarikan"},

		{"penarikan disetujui", func(s services.NotifikasiService) error {
			return s.NotifPenarikanDisetujui(ctx, ntfUser, ntfToken, "besok", ntfRef)
		}, "nasabah", "penarikan"},

		{"request expired nasabah", func(s services.NotifikasiService) error {
			return s.NotifPenarikanRequestExpired(ctx, ntfUser, ntfToken, "Rp50000", ntfRef)
		}, "nasabah", "penarikan"},

		{"request expired admin", func(s services.NotifikasiService) error {
			return s.NotifPenarikanRequestExpiredAdmin(ctx, ntfUser, ntfToken, "Budi", "Rp50000", ntfRef)
		}, "admin", "penarikan"},

		{"pickup expired nasabah", func(s services.NotifikasiService) error {
			return s.NotifPenarikanPickupExpired(ctx, ntfUser, ntfToken, "Rp50000", ntfRef)
		}, "nasabah", "penarikan"},

		{"pickup expired admin", func(s services.NotifikasiService) error {
			return s.NotifPenarikanPickupExpiredAdmin(ctx, ntfUser, ntfToken, "Budi", "Rp50000", ntfRef)
		}, "admin", "penarikan"},

		{"jatuh tempo", func(s services.NotifikasiService) error {
			return s.NotifJatuhTempo(ctx, ntfUser, ntfToken, 3, ntfRef)
		}, "nasabah", "kontrak"},

		{"akun diverifikasi", func(s services.NotifikasiService) error {
			return s.NotifAkunDiverifikasi(ctx, ntfUser, ntfToken)
		}, "nasabah", ""},

		{"pengajuan diterima", func(s services.NotifikasiService) error {
			return s.NotifPengajuanDiterima(ctx, ntfUser, ntfToken, ntfRef)
		}, "nasabah", "pengajuan"},

		{"pengajuan ditolak", func(s services.NotifikasiService) error {
			return s.NotifPengajuanDitolak(ctx, ntfUser, ntfToken, ntfRef, "dokumen kurang")
		}, "nasabah", "pengajuan"},

		{"pengangkutan berhasil", func(s services.NotifikasiService) error {
			return s.NotifPengangkutanBerhasil(ctx, ntfUser, ntfToken, 4, "BSI Utama", ntfRef)
		}, "admin", "pengangkutan"},

		{"distribusi barang", func(s services.NotifikasiService) error {
			return s.NotifDistribusiBarangBerhasil(ctx, ntfUser, ntfToken, "BSU Mawar", 2, "BSI Utama", ntfRef)
		}, "admin", "distribusi_barang"},

		{"pengajuan penarikan", func(s services.NotifikasiService) error {
			return s.NotifPengajuanPenarikan(ctx, ntfUser, ntfToken, "Budi", "Rp50000", ntfRef)
		}, "admin", "penarikan"},

		{"reminder pengangkutan", func(s services.NotifikasiService) error {
			return s.NotifReminderPengangkutan(ctx, ntfUser, ntfToken, "BSU Mawar", "BSI Utama", "08:00:00", "11:00:00", ntfRef)
		}, "admin", "jadwal_pengangkutan"},

		{"reminder penimbangan H-1", func(s services.NotifikasiService) error {
			return s.NotifReminderPenimbanganH1(ctx, ntfUser, ntfToken, "BSU Mawar", "08:00:00", "11:00:00", ntfRef)
		}, "nasabah", "jadwal_penimbangan"},

		{"reminder penimbangan admin H-1", func(s services.NotifikasiService) error {
			return s.NotifReminderPenimbanganAdminH1(ctx, ntfUser, ntfToken, "BSU Mawar", "08:00:00", "11:00:00", ntfRef)
		}, "admin", "jadwal_penimbangan"},

		{"reminder pengangkutan H-1", func(s services.NotifikasiService) error {
			return s.NotifReminderPengangkutanH1(ctx, ntfUser, ntfToken, "BSU Mawar", "BSI Utama", "08:00:00", "11:00:00", ntfRef)
		}, "admin", "jadwal_pengangkutan"},

		{"pembatalan penimbangan nasabah", func(s services.NotifikasiService) error {
			return s.NotifPembatalanPenimbangan(ctx, ntfUser, ntfToken, tanggal, "08:00:00", "11:00:00", ntfRef)
		}, "nasabah", "jadwal_penimbangan"},

		{"pembatalan penimbangan admin", func(s services.NotifikasiService) error {
			return s.NotifPembatalanPenimbanganAdmin(ctx, ntfUser, ntfToken, tanggal, "08:00:00", "11:00:00", ntfRef)
		}, "admin", "jadwal_penimbangan"},

		{"sesi dibuka", func(s services.NotifikasiService) error {
			return s.NotifSesiPenimbanganDibuka(ctx, ntfUser, ntfToken, ntfRef)
		}, "nasabah", "penimbangan"},

		{"sesi ditutup", func(s services.NotifikasiService) error {
			return s.NotifSesiPenimbanganDitutup(ctx, ntfUser, ntfToken, ntfRef)
		}, "nasabah", "penimbangan"},

		{"request pengangkutan", func(s services.NotifikasiService) error {
			return s.NotifRequestPengangkutan(ctx, ntfUser, ntfToken, "BSU Mawar", ntfRef)
		}, "admin", "pengajuan_pengangkutan"},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			repo, svc := newNtfSvc(t)
			tersimpan := ntfTangkap(repo)

			assert.NoError(t, k.panggil(svc))

			n := *tersimpan
			if !assert.NotNil(t, n) {
				return
			}
			assert.Equal(t, ntfUser, n.UserID)
			assert.Equal(t, k.wantRole, n.RoleTarget)
			assert.NotEmpty(t, n.Judul, "judul tidak boleh kosong")
			assert.NotEmpty(t, n.Pesan, "pesan tidak boleh kosong")

			if k.wantRefType == "" {
				assert.Nil(t, n.RefType, "notifikasi ini tidak punya tujuan deep-link")
				assert.Nil(t, n.RefID)
				return
			}
			if assert.NotNil(t, n.RefType) && assert.NotNil(t, n.RefID) {
				assert.Equal(t, k.wantRefType, *n.RefType)
				assert.Equal(t, ntfRef, *n.RefID)
			}
		})
	}
}

func TestNotifSetoranBerhasil_IsiPesan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)

	assert.NoError(t, svc.NotifSetoranBerhasil(context.Background(), ntfUser, ntfToken, 3, "BSU Mawar", ntfRef))

	n := *tersimpan
	assert.Equal(t, "Anda menyetor 3 jenis sampah ke BSU Mawar.", n.Pesan)
}

func TestNotifBagiHasilDiterima_IsiPesan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)

	assert.NoError(t, svc.NotifBagiHasilDiterima(context.Background(), ntfUser, ntfToken,
		"Rp50000", "BSU Mawar", "BSI Utama", ntfRef))

	n := *tersimpan
	assert.Equal(t, "BSU Mawar baru saja menerima bagi hasil sebanyak Rp50000 dari BSI Utama.", n.Pesan)
}

func TestNotifPengajuanDitolak_AlasanMasukPesan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)

	assert.NoError(t, svc.NotifPengajuanDitolak(context.Background(), ntfUser, ntfToken, ntfRef, "dokumen kurang"))

	assert.Contains(t, (*tersimpan).Pesan, "dokumen kurang")
}

func TestNotifJatuhTempo_HariMasukPesan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)

	assert.NoError(t, svc.NotifJatuhTempo(context.Background(), ntfUser, ntfToken, 3, ntfRef))

	assert.Contains(t, (*tersimpan).Pesan, "dalam 3 hari")
}

func TestNotifReminder_JamDiformatIndo(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)

	assert.NoError(t, svc.NotifReminderPenimbanganH1(context.Background(), ntfUser, ntfToken,
		"BSU Mawar", "08:00:00", "11:30:00", ntfRef))

	n := *tersimpan
	assert.Contains(t, n.Pesan, "08.00 - 11.30")
	assert.NotContains(t, n.Pesan, "08:00:00", "jam mentah tidak boleh bocor ke pesan")
	assert.Contains(t, n.Pesan, "BSU Mawar")
}

func TestNotifReminderPenimbangan_VersiNasabahDanAdmin(t *testing.T) {
	repoNasabah, svcNasabah := newNtfSvc(t)
	nasabah := ntfTangkap(repoNasabah)
	assert.NoError(t, svcNasabah.NotifReminderPenimbanganH1(context.Background(), ntfUser, ntfToken,
		"BSU Mawar", "08:00:00", "11:00:00", ntfRef))

	repoAdmin, svcAdmin := newNtfSvc(t)
	admin := ntfTangkap(repoAdmin)
	assert.NoError(t, svcAdmin.NotifReminderPenimbanganAdminH1(context.Background(), ntfUser, ntfToken,
		"BSU Mawar", "08:00:00", "11:00:00", ntfRef))

	assert.Equal(t, (*nasabah).Judul, (*admin).Judul, "judulnya sama")
	assert.Contains(t, (*nasabah).Pesan, "Siapkan sampah Anda!")
	assert.Contains(t, (*admin).Pesan, "Siapkan sesi penimbangannya!")

	assert.Equal(t, "nasabah", (*nasabah).RoleTarget)
	assert.Equal(t, "admin", (*admin).RoleTarget)
}

func TestNotifPembatalanPenimbangan_TanggalIndo(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)
	tanggal := time.Date(2026, time.September, 17, 0, 0, 0, 0, time.Local)

	assert.NoError(t, svc.NotifPembatalanPenimbangan(context.Background(), ntfUser, ntfToken,
		tanggal, "08:00:00", "11:00:00", ntfRef))

	n := *tersimpan
	assert.Contains(t, n.Pesan, "17 September 2026")
	assert.Contains(t, n.Pesan, "08.00 - 11.00 WIB")
	assert.NotContains(t, n.Pesan, "2026-09-17")
}

func TestNotifDistribusiBarang_IsiPesan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	tersimpan := ntfTangkap(repo)

	assert.NoError(t, svc.NotifDistribusiBarangBerhasil(context.Background(), ntfUser, ntfToken,
		"BSU Mawar", 2, "BSI Utama", ntfRef))

	assert.Equal(t, "BSU Mawar baru saja menerima 2 barang dari BSI Utama.", (*tersimpan).Pesan)
}

func TestNotifHelper_ErrorSimpanDiteruskan(t *testing.T) {
	repo, svc := newNtfSvc(t)
	repo.On("Create", mock.Anything).Return(errors.New("db error"))

	err := svc.NotifSetoranBerhasil(context.Background(), ntfUser, ntfToken, 1, "BSU Mawar", ntfRef)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "gagal simpan notifikasi")
}
