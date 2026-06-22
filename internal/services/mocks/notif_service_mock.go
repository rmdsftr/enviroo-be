package mocks

import (
	"context"

	"enviroo-be/internal/services"

	"github.com/stretchr/testify/mock"
)

type MockNotifSvc struct {
	mock.Mock
}

func (m *MockNotifSvc) Kirim(ctx context.Context, req services.KirimNotifRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *MockNotifSvc) GetByUser(ctx context.Context, userID, roleTarget string, page, limit int) (*services.NotifListResponse, error) {
	args := m.Called(ctx, userID, roleTarget, page, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*services.NotifListResponse), args.Error(1)
}

func (m *MockNotifSvc) BacaNotif(ctx context.Context, notifikasiID, userID string) error {
	args := m.Called(ctx, notifikasiID, userID)
	return args.Error(0)
}

func (m *MockNotifSvc) BacaSemua(ctx context.Context, userID, roleTarget string) error {
	args := m.Called(ctx, userID, roleTarget)
	return args.Error(0)
}

func (m *MockNotifSvc) JumlahBelumDibaca(ctx context.Context, userID, roleTarget string) (int64, error) {
	args := m.Called(ctx, userID, roleTarget)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockNotifSvc) Hapus(ctx context.Context, notifikasiID, userID string) error {
	args := m.Called(ctx, notifikasiID, userID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifSetoranBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBank string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, totalJenisSampah, namaBank, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifBagiHasilDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, namaBSU string, namaBSI string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, pesanNilai, namaBSU, namaBSI, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifBagiHasilNasabahDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, pesanNilai, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifPenarikanBerhasil(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, pesanNilai, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifPenarikanDitolak(ctx context.Context, userID, fcmToken string, refID string, alasan string) error {
	args := m.Called(ctx, userID, fcmToken, refID, alasan)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifJatuhTempo(ctx context.Context, userID, fcmToken string, hariLagi int, refID string) error {
	args := m.Called(ctx, userID, fcmToken, hariLagi, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifAkunDiverifikasi(ctx context.Context, userID, fcmToken string) error {
	args := m.Called(ctx, userID, fcmToken)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifPengajuanDiterima(ctx context.Context, userID, fcmToken string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifPengajuanDitolak(ctx context.Context, userID, fcmToken string, refID string, alasan string) error {
	args := m.Called(ctx, userID, fcmToken, refID, alasan)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifPengangkutanBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBSI string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, totalJenisSampah, namaBSI, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifDistribusiSembakoBerhasil(ctx context.Context, userID, fcmToken string, namaBSU string, totalJenisSembako int, namaBSI string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, namaBSU, totalJenisSembako, namaBSI, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifPengajuanPenarikan(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, namaNasabah, pesanNilai, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifReminderPenimbangan(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, namaBank, jamMulai, jamSelesai, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifReminderPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, namaBSU, namaBSI, jamMulai, jamSelesai, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifReminderPenimbanganH1(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, namaBank, jamMulai, jamSelesai, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifReminderPengangkutanH1(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, namaBSU, namaBSI, jamMulai, jamSelesai, refID)
	return args.Error(0)
}

func (m *MockNotifSvc) NotifRequestPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, refID string) error {
	args := m.Called(ctx, userID, fcmToken, namaBSU, refID)
	return args.Error(0)
}
