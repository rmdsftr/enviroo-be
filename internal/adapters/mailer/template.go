package mailer

import (
	"fmt"
	"html"
	"time"
)

// InstallInfo berisi link APK terbaru untuk disisipkan di email aktivasi
// pengguna aplikasi mobile (nasabah & petugas).
type InstallInfo struct {
	URL         string
	VersionName string
}

func BuildAktivasiAdminEmail(nama, roleTitle, namaBank, labelBank, otp string, expiredAt time.Time, install *InstallInfo) string {
	return fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
			<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
			<p style="font-size: 14px; color: #333; line-height: 1.5;">
				Anda telah ditunjuk sebagai <b>%s</b> di <b>%s</b>%s.
				Untuk menyelesaikan proses aktivasi akun, gunakan kode OTP berikut:
			</p>
			<div style="background-color: #fff; border: 2px dashed #4ea771; padding: 15px; text-align: center; margin: 20px 0;">
				<h1 style="color: #4ea771; font-size: 32px; font-weight: bold; margin: 0; letter-spacing: 5px;">%s</h1>
			</div>
			<p style="font-size: 13px; color: #666;">
				Kode ini berlaku selama 24 jam hingga <b>%s</b>.
				Mohon untuk tidak membagikan kode OTP ini ke siapapun.
			</p>
			%s
			<hr style="border: 0; height: 1px; background: #ddd; margin: 25px 0;">
			<p style="font-size: 12px; color: #999; text-align: center; margin: 0;">&copy; Enviroo APP</p>
		</div>
	`, nama, roleTitle, namaBank, labelBank, otp, expiredAt.Format("02 Jan 2006 15:04 WIB"), blokInstallAplikasi(install))
}

func BuildOTPEmail(nama, paragraf, otp, durasi, catatan string, expiredAt time.Time, install *InstallInfo) string {
	return fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
			<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
			<p style="font-size: 14px; color: #333; line-height: 1.5;">
				%s
			</p>
			<div style="background-color: #fff; border: 2px dashed #4ea771; padding: 15px; text-align: center; margin: 20px 0;">
				<h1 style="color: #4ea771; font-size: 32px; font-weight: bold; margin: 0; letter-spacing: 5px;">%s</h1>
			</div>
			<p style="font-size: 13px; color: #666;">
				Kode ini berlaku selama %s hingga <b>%s</b>.
				%s
			</p>
			%s
			<hr style="border: 0; height: 1px; background: #ddd; margin: 25px 0;">
			<p style="font-size: 12px; color: #999; text-align: center; margin: 0;">&copy; Enviroo APP</p>
		</div>
	`, nama, paragraf, otp, durasi, expiredAt.Format("02 Jan 2006 15:04 WIB"), catatan, blokInstallAplikasi(install))
}

// blokInstallAplikasi menghasilkan langkah aktivasi + tombol unduh APK.
// Kosong bila belum ada rilis, sehingga email tampil seperti biasa.
func blokInstallAplikasi(info *InstallInfo) string {
	if info == nil || info.URL == "" {
		return ""
	}

	url := html.EscapeString(info.URL)
	versi := "Khusus Android"
	if info.VersionName != "" {
		versi = fmt.Sprintf("Versi %s &middot; Khusus Android", html.EscapeString(info.VersionName))
	}

	return fmt.Sprintf(`
			<div style="background-color: #fff; border-radius: 8px; padding: 20px; margin: 20px 0;">
				<p style="font-size: 14px; color: #333; font-weight: bold; margin: 0 0 10px;">Cara mengaktifkan akun</p>
				<ol style="font-size: 14px; color: #333; line-height: 1.6; padding-left: 20px; margin: 0 0 20px;">
					<li>Unduh dan pasang aplikasi Enviroo melalui tombol di bawah. Jika aplikasi Enviroo sudah terpasang di HP Anda, lewati langkah ini.</li>
					<li>Buka aplikasi, lalu ketuk <b>"Aktivasi di sini"</b> di halaman login.</li>
					<li>Masukkan <b>ID Pengguna</b> Anda dan <b>kode OTP</b> di atas.</li>
					<li>Jika diminta, buat password baru. Setelah itu, login menggunakan email dan password Anda.</li>
				</ol>
				<div style="text-align: center;">
					<a href="%s" style="display: inline-block; background-color: #4ea771; color: #fff; text-decoration: none; font-weight: bold; font-size: 14px; padding: 12px 28px; border-radius: 24px;">Unduh Aplikasi Enviroo</a>
					<p style="font-size: 12px; color: #999; margin: 8px 0 0;">%s</p>
				</div>
				<p style="font-size: 13px; color: #666; line-height: 1.5; margin: 16px 0 0;">
					Saat memasang, Android mungkin menampilkan peringatan atau meminta izin "Instal aplikasi tidak dikenal". Izinkan untuk melanjutkan pemasangan.
				</p>
				<p style="font-size: 12px; color: #999; line-height: 1.5; margin: 12px 0 0; word-break: break-all;">
					Tombol tidak berfungsi? Salin link berikut ke browser: <a href="%s" style="color: #4ea771;">%s</a>
				</p>
			</div>`, url, versi, url, url)
}

func BuildNotifEmail(nama, paragraf string) string {
	return fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; background-color: #f4fdf4; padding: 30px; border-radius: 10px;">
			<h2 style="color: #4ea771; margin-top: 0;">Halo, %s!</h2>
			<p style="font-size: 14px; color: #333; line-height: 1.5;">
				%s
			</p>
			<hr style="border: 0; height: 1px; background: #ddd; margin: 25px 0;">
			<p style="font-size: 12px; color: #999; text-align: center; margin: 0;">&copy; Enviroo APP</p>
		</div>
	`, nama, paragraf)
}
