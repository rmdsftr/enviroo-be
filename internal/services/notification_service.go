package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/pkg/utils"

	"github.com/google/uuid"
)

// NotifikasiService menangani logika bisnis notifikasi:
//   - menyimpan ke database
//   - mengirim push notification via FCM
type NotifikasiService interface {
	// Kirim & simpan notifikasi ke satu user
	Kirim(ctx context.Context, req KirimNotifRequest) error

	// Ambil daftar notifikasi user (paginasi)
	GetByUser(ctx context.Context, userID string, page, limit int) (*NotifListResponse, error)

	// Tandai satu notifikasi sebagai sudah dibaca
	BacaNotif(ctx context.Context, notifikasiID, userID string) error

	// Tandai semua notifikasi user sebagai sudah dibaca
	BacaSemua(ctx context.Context, userID string) error

	// Jumlah notifikasi belum dibaca
	JumlahBelumDibaca(ctx context.Context, userID string) (int64, error)

	// Hapus satu notifikasi milik user
	Hapus(ctx context.Context, notifikasiID, userID string) error

	// ---- Helper per-fitur ----
	NotifSetoranBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBank string, refID string) error
	NotifBagiHasilDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, namaBSU string, namaBSI string, refID string) error
	NotifBagiHasilNasabahDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error
	NotifPenarikanBerhasil(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error
	NotifPenarikanDitolak(ctx context.Context, userID, fcmToken string, refID string, alasan string) error
	NotifJatuhTempo(ctx context.Context, userID, fcmToken string, hariLagi int, refID string) error
	NotifAkunDiverifikasi(ctx context.Context, userID, fcmToken string) error
	NotifPengajuanDiterima(ctx context.Context, userID, fcmToken string, refID string) error
	NotifPengajuanDitolak(ctx context.Context, userID, fcmToken string, refID string, alasan string) error
	NotifPengangkutanBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBSI string, refID string) error
	NotifDistribusiSembakoBerhasil(ctx context.Context, userID, fcmToken string, namaBSU string, totalJenisSembako int, namaBSI string, refID string) error
	NotifPengajuanPenarikan(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error
	NotifReminderPenimbangan(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error
	NotifReminderPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error
	NotifReminderPenimbanganH1(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error
	NotifReminderPengangkutanH1(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error
	NotifRequestPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, refID string) error
}

// KirimNotifRequest adalah parameter umum untuk mengirim notifikasi.
type KirimNotifRequest struct {
	UserID   string
	FCMToken string // boleh kosong jika hanya simpan ke DB

	Judul   string
	Pesan   string
	RefID   *string
	RefType *string

	// Data tambahan untuk deep-link di Flutter
	ExtraData map[string]string
}

type NotifListResponse struct {
	Data         []models.Notifikasi
	Total        int64
	Page         int
	Limit        int
	TotalHalaman int
}

// ----------------------------------------------------------------

type notifikasiService struct {
	repo repositories.NotifikasiRepository
	fcm  *utils.FCMClient // boleh nil jika FCM tidak dikonfigurasi
}

func NewNotifikasiService(
	repo repositories.NotifikasiRepository,
	fcm *utils.FCMClient,
) NotifikasiService {
	return &notifikasiService{repo: repo, fcm: fcm}
}

// ----------------------------------------------------------------
// Core
// ----------------------------------------------------------------

func (s *notifikasiService) Kirim(ctx context.Context, req KirimNotifRequest) error {
	// 1. Simpan ke database
	notif := &models.Notifikasi{
		NotifikasiID: uuid.New().String(),
		UserID:       req.UserID,
		Judul:        req.Judul,
		Pesan:        req.Pesan,
		RefID:        req.RefID,
		RefType:      req.RefType,
		IsRead:       false,
		CreatedAt:    time.Now(),
	}
	if err := s.repo.Create(notif); err != nil {
		return fmt.Errorf("gagal simpan notifikasi: %w", err)
	}

	// 2. Kirim push notification (opsional — jika FCM tersedia & token ada)
	if s.fcm != nil && req.FCMToken != "" {
		data := map[string]string{
			"notifikasi_id": notif.NotifikasiID,
		}
		if req.RefID != nil {
			data["ref_id"] = *req.RefID
		}
		if req.RefType != nil {
			data["ref_type"] = *req.RefType
		}
		for k, v := range req.ExtraData {
			data[k] = v
		}

		_, err := s.fcm.SendToDevice(ctx, utils.FCMPayload{
			Token: req.FCMToken,
			Title: req.Judul,
			Body:  req.Pesan,
			Data:  data,
		})
		if err != nil {
			// Jangan gagalkan seluruh flow hanya karena FCM error
			log.Printf("[NotifikasiService] FCM gagal untuk user %s: %v", req.UserID, err)
		}
	}

	return nil
}

func (s *notifikasiService) GetByUser(ctx context.Context, userID string, page, limit int) (*NotifListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	data, total, err := s.repo.FindByUserID(userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("gagal ambil notifikasi: %w", err)
	}

	totalHalaman := int(total) / limit
	if int(total)%limit != 0 {
		totalHalaman++
	}

	return &NotifListResponse{
		Data:         data,
		Total:        total,
		Page:         page,
		Limit:        limit,
		TotalHalaman: totalHalaman,
	}, nil
}

func (s *notifikasiService) BacaNotif(ctx context.Context, notifikasiID, userID string) error {
	return s.repo.MarkAsRead(notifikasiID, userID)
}

func (s *notifikasiService) BacaSemua(ctx context.Context, userID string) error {
	return s.repo.MarkAllAsRead(userID)
}

func (s *notifikasiService) JumlahBelumDibaca(ctx context.Context, userID string) (int64, error) {
	return s.repo.CountUnread(userID)
}

func (s *notifikasiService) Hapus(ctx context.Context, notifikasiID, userID string) error {
	return s.repo.Delete(notifikasiID, userID)
}

// ----------------------------------------------------------------
// Helper per-fitur
// ----------------------------------------------------------------

func ptr(s string) *string { return &s }

func (s *notifikasiService) NotifSetoranBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBank string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Setoran Sampah Berhasil 🎉",
		Pesan:    fmt.Sprintf("Anda menyetor %d jenis sampah ke %s.", totalJenisSampah, namaBank),
		RefID:    ptr(refID),
		RefType:  ptr("setoran"),
	})
}

func (s *notifikasiService) NotifBagiHasilDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, namaBSU string, namaBSI string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Bagi Hasil Diterima 💰",
		Pesan:    fmt.Sprintf("%s baru saja menerima bagi hasil sebanyak %s dari %s.", namaBSU, pesanNilai, namaBSI),
		RefID:    ptr(refID),
		RefType:  ptr("bagi_hasil"),
	})
}

func (s *notifikasiService) NotifBagiHasilNasabahDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Bagi Hasil Diterima 💰",
		Pesan:    fmt.Sprintf("Anda baru saja menerima bagi hasil sebanyak %s.", pesanNilai),
		RefID:    ptr(refID),
		RefType:  ptr("bagi_hasil"),
	})
}

func (s *notifikasiService) NotifPenarikanBerhasil(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Penarikan Saldo Berhasil ✅",
		Pesan:    fmt.Sprintf("Saldo anda sebanyak %s sudah dicairkan oleh petugas bank sampah.", pesanNilai),
		RefID:    ptr(refID),
		RefType:  ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifPenarikanDitolak(ctx context.Context, userID, fcmToken string, refID string, alasan string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Penarikan Ditolak ❌",
		Pesan:    fmt.Sprintf("Permintaan penarikanmu ditolak. Alasan: %s", alasan),
		RefID:    ptr(refID),
		RefType:  ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifJatuhTempo(ctx context.Context, userID, fcmToken string, hariLagi int, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Pengingat Jatuh Tempo ⏰",
		Pesan:    fmt.Sprintf("Kontrakmu akan jatuh tempo dalam %d hari. Segera lakukan perpanjangan atau penarikan.", hariLagi),
		RefID:    ptr(refID),
		RefType:  ptr("kontrak"),
	})
}

func (s *notifikasiService) NotifAkunDiverifikasi(ctx context.Context, userID, fcmToken string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Akun Terverifikasi ✅",
		Pesan:    "Selamat! Akun kamu telah berhasil diverifikasi. Kamu sudah bisa mulai berinvestasi.",
	})
}

func (s *notifikasiService) NotifPengajuanDiterima(ctx context.Context, userID, fcmToken string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Pengajuan Disetujui 🎉",
		Pesan:    "Pengajuanmu telah disetujui. Silakan cek detailnya di aplikasi.",
		RefID:    ptr(refID),
		RefType:  ptr("pengajuan"),
	})
}

func (s *notifikasiService) NotifPengajuanDitolak(ctx context.Context, userID, fcmToken string, refID string, alasan string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Pengajuan Ditolak ❌",
		Pesan:    fmt.Sprintf("Maaf, pengajuanmu tidak dapat disetujui. Alasan: %s", alasan),
		RefID:    ptr(refID),
		RefType:  ptr("pengajuan"),
	})
}

// ----------------------------------------------------------------
// Util
// ----------------------------------------------------------------

// formatRupiah memformat angka ke format ribuan Indonesia.
// Contoh: 1500000 → "1.500.000"
func formatRupiah(nominal int64) string {
	s := fmt.Sprintf("%d", nominal)
	n := len(s)
	if n <= 3 {
		return s
	}
	result := ""
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result += "."
		}
		result += string(c)
	}
	return result
}

func (s *notifikasiService) NotifPengangkutanBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBSI string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Pengangkutan Sampah Berhasil 🚛",
		Pesan:    fmt.Sprintf("Sampah sebanyak %d jenis sudah diangkut oleh %s.", totalJenisSampah, namaBSI),
		RefID:    ptr(refID),
		RefType:  ptr("pengangkutan"),
	})
}

func (s *notifikasiService) NotifDistribusiSembakoBerhasil(ctx context.Context, userID, fcmToken string, namaBSU string, totalJenisSembako int, namaBSI string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Distribusi Sembako Berhasil 🛒",
		Pesan:    fmt.Sprintf("%s baru saja menerima %d sembako dari %s.", namaBSU, totalJenisSembako, namaBSI),
		RefID:    ptr(refID),
		RefType:  ptr("distribusi_sembako"),
	})
}

func (s *notifikasiService) NotifPengajuanPenarikan(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Pengajuan Penarikan 📥",
		Pesan:    fmt.Sprintf("%s mengajukan penarikan saldo %s.", namaNasabah, pesanNilai),
		RefID:    ptr(refID),
		RefType:  ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifReminderPenimbangan(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Jadwal Penimbangan Hari Ini ⚖️",
		Pesan:    fmt.Sprintf("Jangan lupa! Hari ini ada penimbangan sampah di %s mulai pukul %s - %s.", namaBank, jamMulai, jamSelesai),
		RefID:    ptr(refID),
		RefType:  ptr("jadwal_penimbangan"),
	})
}

func (s *notifikasiService) NotifReminderPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Jadwal Pengangkutan Sampah Hari Ini 🚛",
		Pesan:    fmt.Sprintf("Hari ini ada jadwal pengangkutan sampah dari %s ke %s pada pukul %s - %s.", namaBSU, namaBSI, jamMulai, jamSelesai),
		RefID:    ptr(refID),
		RefType:  ptr("jadwal_pengangkutan"),
	})
}

func (s *notifikasiService) NotifReminderPenimbanganH1(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Reminder Jadwal Penimbangan ⏰",
		Pesan:    fmt.Sprintf("Besok ada jadwal penimbangan sampah di %s mulai pukul %s - %s. Siapkan sampah Anda!", namaBank, jamMulai, jamSelesai),
		RefID:    ptr(refID),
		RefType:  ptr("jadwal_penimbangan"),
	})
}

func (s *notifikasiService) NotifReminderPengangkutanH1(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Reminder Jadwal Pengangkutan ⏰",
		Pesan:    fmt.Sprintf("Besok ada jadwal pengangkutan sampah dari %s ke %s pada pukul %s - %s.", namaBSU, namaBSI, jamMulai, jamSelesai),
		RefID:    ptr(refID),
		RefType:  ptr("jadwal_pengangkutan"),
	})
}

func (s *notifikasiService) NotifRequestPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:   userID,
		FCMToken: fcmToken,
		Judul:    "Pengajuan Pengangkutan Sampah",
		Pesan:    fmt.Sprintf("%s mengajukan pengangkutan sampah di luar jadwal.", namaBSU),
		RefID:    ptr(refID),
		RefType:  ptr("pengajuan_pengangkutan"),
	})
}
