package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"enviroo-be/internal/adapters/fcm"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/utils"

	"github.com/google/uuid"
)

type NotifikasiService interface {
	Kirim(ctx context.Context, req KirimNotifRequest) error

	GetByUser(ctx context.Context, userID, roleTarget string, page, limit int) (*NotifListResponse, error)

	BacaNotif(ctx context.Context, notifikasiID, userID string) error

	BacaSemua(ctx context.Context, userID, roleTarget string) error

	JumlahBelumDibaca(ctx context.Context, userID, roleTarget string) (int64, error)

	Hapus(ctx context.Context, notifikasiID, userID string) error

	NotifSetoranBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBank string, refID string) error
	NotifBagiHasilDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, namaBSU string, namaBSI string, refID string) error
	NotifBagiHasilNasabahDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error
	NotifPenarikanBerhasil(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error
	NotifPenarikanDitolak(ctx context.Context, userID, fcmToken string, refID string) error
	NotifPenarikanDisetujui(ctx context.Context, userID, fcmToken string, pesanDeadlineJemput string, refID string) error
	NotifPenarikanRequestExpired(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error
	NotifPenarikanRequestExpiredAdmin(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error
	NotifPenarikanPickupExpired(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error
	NotifPenarikanPickupExpiredAdmin(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error
	NotifJatuhTempo(ctx context.Context, userID, fcmToken string, hariLagi int, refID string) error
	NotifAkunDiverifikasi(ctx context.Context, userID, fcmToken string) error
	NotifPengajuanDiterima(ctx context.Context, userID, fcmToken string, refID string) error
	NotifPengajuanDitolak(ctx context.Context, userID, fcmToken string, refID string, alasan string) error
	NotifPengangkutanBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBSI string, refID string) error
	NotifDistribusiBarangBerhasil(ctx context.Context, userID, fcmToken string, namaBSU string, totalJenisBarang int, namaBSI string, refID string) error
	NotifPengajuanPenarikan(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error
	NotifReminderPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error
	NotifReminderPenimbanganH1(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error
	NotifReminderPenimbanganAdminH1(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error
	NotifReminderPengangkutanH1(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error
	NotifPembatalanPenimbangan(ctx context.Context, userID, fcmToken string, tanggalSesi time.Time, jamMulai string, jamSelesai string, refID string) error
	NotifPembatalanPenimbanganAdmin(ctx context.Context, userID, fcmToken string, tanggalSesi time.Time, jamMulai string, jamSelesai string, refID string) error
	NotifSesiPenimbanganDibuka(ctx context.Context, userID, fcmToken string, refID string) error
	NotifSesiPenimbanganDitutup(ctx context.Context, userID, fcmToken string, refID string) error
	NotifRequestPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, refID string) error
}

type KirimNotifRequest struct {
	UserID     string
	FCMToken   string
	RoleTarget string

	Judul   string
	Pesan   string
	RefID   *string
	RefType *string

	ExtraData map[string]string
}

type NotifListResponse struct {
	Data         []models.Notifikasi
	Total        int64
	Page         int
	Limit        int
	TotalHalaman int
}

type notifikasiService struct {
	repo      repositories.NotifikasiRepository
	fcmClient *fcm.Client
}

func NewNotifikasiService(
	repo repositories.NotifikasiRepository,
	fcmClient *fcm.Client,
) NotifikasiService {
	return &notifikasiService{repo: repo, fcmClient: fcmClient}
}

func (s *notifikasiService) Kirim(ctx context.Context, req KirimNotifRequest) error {
	notif := &models.Notifikasi{
		NotifikasiID: uuid.New().String(),
		UserID:       req.UserID,
		RoleTarget:   req.RoleTarget,
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

	if s.fcmClient != nil && req.FCMToken != "" {
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

		_, err := s.fcmClient.SendToDevice(ctx, fcm.Payload{
			Token: req.FCMToken,
			Title: req.Judul,
			Body:  req.Pesan,
			Data:  data,
		})
		if err != nil {
			log.Printf("[NotifikasiService] FCM gagal untuk user %s: %v", req.UserID, err)
		}
	}

	return nil
}

func (s *notifikasiService) GetByUser(ctx context.Context, userID, roleTarget string, page, limit int) (*NotifListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	offset := (page - 1) * limit

	data, total, err := s.repo.FindByUserID(userID, roleTarget, limit, offset)
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

func (s *notifikasiService) BacaSemua(ctx context.Context, userID, roleTarget string) error {
	return s.repo.MarkAllAsRead(userID, roleTarget)
}

func (s *notifikasiService) JumlahBelumDibaca(ctx context.Context, userID, roleTarget string) (int64, error) {
	return s.repo.CountUnread(userID, roleTarget)
}

func (s *notifikasiService) Hapus(ctx context.Context, notifikasiID, userID string) error {
	return s.repo.Delete(notifikasiID, userID)
}

func ptr(s string) *string { return &s }

func (s *notifikasiService) NotifSetoranBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBank string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Setoran Sampah Berhasil 🎉",
		Pesan:      fmt.Sprintf("Anda menyetor %d jenis sampah ke %s.", totalJenisSampah, namaBank),
		RefID:      ptr(refID),
		RefType:    ptr("setoran"),
	})
}

func (s *notifikasiService) NotifBagiHasilDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, namaBSU string, namaBSI string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Bagi Hasil Diterima 💰",
		Pesan:      fmt.Sprintf("%s baru saja menerima bagi hasil sebanyak %s dari %s.", namaBSU, pesanNilai, namaBSI),
		RefID:      ptr(refID),
		RefType:    ptr("bagi_hasil"),
	})
}

func (s *notifikasiService) NotifBagiHasilNasabahDiterima(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Bagi Hasil Diterima 💰",
		Pesan:      fmt.Sprintf("Anda baru saja menerima bagi hasil sebanyak %s.", pesanNilai),
		RefID:      ptr(refID),
		RefType:    ptr("bagi_hasil"),
	})
}

func (s *notifikasiService) NotifPenarikanBerhasil(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Penarikan Saldo Berhasil ✅",
		Pesan:      fmt.Sprintf("Saldo anda sebanyak %s sudah dicairkan oleh petugas bank sampah.", pesanNilai),
		RefID:      ptr(refID),
		RefType:    ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifPenarikanDitolak(ctx context.Context, userID, fcmToken string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Penarikan Ditolak ❌",
		Pesan:      "Maaf, pengajuan penarikan tidak disetujui oleh pihak bank sampah.",
		RefID:      ptr(refID),
		RefType:    ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifPenarikanDisetujui(ctx context.Context, userID, fcmToken string, pesanDeadlineJemput string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Penarikan Disetujui ✅",
		Pesan:      fmt.Sprintf("Pengajuan penarikan telah disetujui, silahkan jemput sebelum %s.", pesanDeadlineJemput),
		RefID:      ptr(refID),
		RefType:    ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifPenarikanRequestExpired(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Pengajuan Penarikan Kadaluarsa ⏰",
		Pesan:      fmt.Sprintf("Pengajuanmu senilai %s kadaluarsa karena tidak dikonfirmasi tepat waktu. Saldo telah dikembalikan.", pesanNilai),
		RefID:      ptr(refID),
		RefType:    ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifPenarikanRequestExpiredAdmin(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Pengajuan Penarikan Kadaluarsa ⏰",
		Pesan:      fmt.Sprintf("Pengajuan penarikan %s dari %s kadaluarsa karena tidak dikonfirmasi tepat waktu.", pesanNilai, namaNasabah),
		RefID:      ptr(refID),
		RefType:    ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifPenarikanPickupExpired(ctx context.Context, userID, fcmToken string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Batas Penjemputan Terlewat ⏰",
		Pesan:      fmt.Sprintf("Penjemputan penarikanmu senilai %s terlewat batas waktu. Saldo telah dikembalikan.", pesanNilai),
		RefID:      ptr(refID),
		RefType:    ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifPenarikanPickupExpiredAdmin(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Batas Penjemputan Terlewat ⏰",
		Pesan:      fmt.Sprintf("Penjemputan penarikan %s oleh %s terlewat batas waktu.", pesanNilai, namaNasabah),
		RefID:      ptr(refID),
		RefType:    ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifJatuhTempo(ctx context.Context, userID, fcmToken string, hariLagi int, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Pengingat Jatuh Tempo ⏰",
		Pesan:      fmt.Sprintf("Kontrakmu akan jatuh tempo dalam %d hari. Segera lakukan perpanjangan atau penarikan.", hariLagi),
		RefID:      ptr(refID),
		RefType:    ptr("kontrak"),
	})
}

func (s *notifikasiService) NotifAkunDiverifikasi(ctx context.Context, userID, fcmToken string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Akun Terverifikasi ✅",
		Pesan:      "Selamat! Akun kamu telah berhasil diverifikasi. Kamu sudah bisa mulai berinvestasi.",
	})
}

func (s *notifikasiService) NotifPengajuanDiterima(ctx context.Context, userID, fcmToken string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Pengajuan Disetujui 🎉",
		Pesan:      "Pengajuanmu telah disetujui. Silakan cek detailnya di aplikasi.",
		RefID:      ptr(refID),
		RefType:    ptr("pengajuan"),
	})
}

func (s *notifikasiService) NotifPengajuanDitolak(ctx context.Context, userID, fcmToken string, refID string, alasan string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Pengajuan Ditolak ❌",
		Pesan:      fmt.Sprintf("Maaf, pengajuanmu tidak dapat disetujui. Alasan: %s", alasan),
		RefID:      ptr(refID),
		RefType:    ptr("pengajuan"),
	})
}

func (s *notifikasiService) NotifPengangkutanBerhasil(ctx context.Context, userID, fcmToken string, totalJenisSampah int, namaBSI string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Pengangkutan Sampah Berhasil 🚛",
		Pesan:      fmt.Sprintf("Sampah sebanyak %d jenis sudah diangkut oleh %s.", totalJenisSampah, namaBSI),
		RefID:      ptr(refID),
		RefType:    ptr("pengangkutan"),
	})
}

func (s *notifikasiService) NotifDistribusiBarangBerhasil(ctx context.Context, userID, fcmToken string, namaBSU string, totalJenisBarang int, namaBSI string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Distribusi Barang Berhasil 🛒",
		Pesan:      fmt.Sprintf("%s baru saja menerima %d barang dari %s.", namaBSU, totalJenisBarang, namaBSI),
		RefID:      ptr(refID),
		RefType:    ptr("distribusi_barang"),
	})
}

func (s *notifikasiService) NotifPengajuanPenarikan(ctx context.Context, userID, fcmToken string, namaNasabah string, pesanNilai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Pengajuan Penarikan 📥",
		Pesan:      fmt.Sprintf("%s mengajukan penarikan saldo %s.", namaNasabah, pesanNilai),
		RefID:      ptr(refID),
		RefType:    ptr("penarikan"),
	})
}

func (s *notifikasiService) NotifReminderPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Jadwal Pengangkutan Sampah Hari Ini 🚛",
		Pesan:      fmt.Sprintf("Hari ini ada jadwal pengangkutan sampah dari %s ke %s pada pukul %s - %s.", namaBSU, namaBSI, utils.FormatJamIndo(jamMulai), utils.FormatJamIndo(jamSelesai)),
		RefID:      ptr(refID),
		RefType:    ptr("jadwal_pengangkutan"),
	})
}

func (s *notifikasiService) NotifReminderPenimbanganH1(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Reminder Jadwal Penimbangan ⏰",
		Pesan:      fmt.Sprintf("Besok ada jadwal penimbangan sampah di %s mulai pukul %s - %s. Siapkan sampah Anda!", namaBank, utils.FormatJamIndo(jamMulai), utils.FormatJamIndo(jamSelesai)),
		RefID:      ptr(refID),
		RefType:    ptr("jadwal_penimbangan"),
	})
}

func (s *notifikasiService) NotifReminderPenimbanganAdminH1(ctx context.Context, userID, fcmToken string, namaBank string, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Reminder Jadwal Penimbangan ⏰",
		Pesan:      fmt.Sprintf("Besok ada jadwal penimbangan sampah di %s mulai pukul %s - %s. Siapkan sesi penimbangannya!", namaBank, utils.FormatJamIndo(jamMulai), utils.FormatJamIndo(jamSelesai)),
		RefID:      ptr(refID),
		RefType:    ptr("jadwal_penimbangan"),
	})
}

func (s *notifikasiService) NotifReminderPengangkutanH1(ctx context.Context, userID, fcmToken string, namaBSU string, namaBSI string, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Reminder Jadwal Pengangkutan ⏰",
		Pesan:      fmt.Sprintf("Besok ada jadwal pengangkutan sampah dari %s ke %s pada pukul %s - %s.", namaBSU, namaBSI, utils.FormatJamIndo(jamMulai), utils.FormatJamIndo(jamSelesai)),
		RefID:      ptr(refID),
		RefType:    ptr("jadwal_pengangkutan"),
	})
}

func (s *notifikasiService) NotifPembatalanPenimbangan(ctx context.Context, userID, fcmToken string, tanggalSesi time.Time, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Pembatalan Sesi Penimbangan",
		Pesan: fmt.Sprintf("Sesi penimbangan pada %s pukul %s - %s WIB dibatalkan petugas. Lihat selengkapnya",
			utils.FormatTanggalSajaIndo(tanggalSesi), utils.FormatJamIndo(jamMulai), utils.FormatJamIndo(jamSelesai)),
		RefID:   ptr(refID),
		RefType: ptr("jadwal_penimbangan"),
	})
}

func (s *notifikasiService) NotifPembatalanPenimbanganAdmin(ctx context.Context, userID, fcmToken string, tanggalSesi time.Time, jamMulai string, jamSelesai string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Pembatalan Sesi Penimbangan",
		Pesan: fmt.Sprintf("Sesi penimbangan pada %s pukul %s - %s WIB dibatalkan petugas. Lihat selengkapnya",
			utils.FormatTanggalSajaIndo(tanggalSesi), utils.FormatJamIndo(jamMulai), utils.FormatJamIndo(jamSelesai)),
		RefID:   ptr(refID),
		RefType: ptr("jadwal_penimbangan"),
	})
}

func (s *notifikasiService) NotifSesiPenimbanganDibuka(ctx context.Context, userID, fcmToken string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Sesi penimbangan dibuka",
		Pesan:      "Sesi penimbangan hari ini sudah dibuka petugas. Ayo setor sampahmu ke bank sampah",
		RefID:      ptr(refID),
		RefType:    ptr("penimbangan"),
	})
}

func (s *notifikasiService) NotifSesiPenimbanganDitutup(ctx context.Context, userID, fcmToken string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "nasabah",
		Judul:      "Sesi penimbangan ditutup",
		Pesan:      "Petugas sudah menutup sesi penimbangan hari ini. Setor sampahnya di jadwal penimbangan selanjutnya ya",
		RefID:      ptr(refID),
		RefType:    ptr("penimbangan"),
	})
}

func (s *notifikasiService) NotifRequestPengangkutan(ctx context.Context, userID, fcmToken string, namaBSU string, refID string) error {
	return s.Kirim(ctx, KirimNotifRequest{
		UserID:     userID,
		FCMToken:   fcmToken,
		RoleTarget: "admin",
		Judul:      "Pengajuan Pengangkutan Sampah",
		Pesan:      fmt.Sprintf("%s mengajukan pengangkutan sampah di luar jadwal.", namaBSU),
		RefID:      ptr(refID),
		RefType:    ptr("pengajuan_pengangkutan"),
	})
}
