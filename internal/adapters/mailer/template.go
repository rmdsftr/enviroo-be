package mailer

import (
	"fmt"
	"time"
)

func BuildAktivasiAdminEmail(nama, roleTitle, namaBank, labelBank, otp string, expiredAt time.Time) string {
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
			<hr style="border: 0; height: 1px; background: #ddd; margin: 25px 0;">
			<p style="font-size: 12px; color: #999; text-align: center; margin: 0;">&copy; Enviroo APP</p>
		</div>
	`, nama, roleTitle, namaBank, labelBank, otp, expiredAt.Format("02 Jan 2006 15:04 WIB"))
}

func BuildOTPEmail(nama, paragraf, otp, durasi, catatan string, expiredAt time.Time) string {
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
			<hr style="border: 0; height: 1px; background: #ddd; margin: 25px 0;">
			<p style="font-size: 12px; color: #999; text-align: center; margin: 0;">&copy; Enviroo APP</p>
		</div>
	`, nama, paragraf, otp, durasi, expiredAt.Format("02 Jan 2006 15:04 WIB"), catatan)
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
