package mailer

import (
	"strings"
	"testing"
	"time"
)

var infoContoh = &InstallInfo{
	URL:         "https://media.enviroo.tech/apk/enviroo-1.0.2-3-abcd1234.apk",
	VersionName: "1.0.2",
}

var teksBlokInstall = []string{"Unduh Aplikasi Enviroo", "Aktivasi di sini", "Cara mengaktifkan akun"}

func assertAdaBlokInstall(t *testing.T, body string) {
	t.Helper()
	for _, teks := range append(teksBlokInstall, infoContoh.URL, "Versi 1.0.2") {
		if !strings.Contains(body, teks) {
			t.Errorf("body harus berisi %q", teks)
		}
	}
}

func assertTanpaBlokInstall(t *testing.T, body string) {
	t.Helper()
	for _, teks := range teksBlokInstall {
		if strings.Contains(body, teks) {
			t.Errorf("body tidak boleh berisi %q", teks)
		}
	}
}

func TestBuildOTPEmail_DenganInstall(t *testing.T) {
	body := BuildOTPEmail("Budi", "paragraf", "123456", "24 jam", "catatan", time.Now(), infoContoh)
	assertAdaBlokInstall(t, body)
	if !strings.Contains(body, "123456") {
		t.Error("OTP harus tetap ada")
	}
}

func TestBuildOTPEmail_TanpaInstall(t *testing.T) {
	body := BuildOTPEmail("Budi", "paragraf", "123456", "24 jam", "catatan", time.Now(), nil)
	assertTanpaBlokInstall(t, body)
}

func TestBuildOTPEmail_URLKosongTanpaInstall(t *testing.T) {
	body := BuildOTPEmail("Budi", "paragraf", "123456", "24 jam", "catatan", time.Now(), &InstallInfo{})
	assertTanpaBlokInstall(t, body)
}

func TestBuildAktivasiAdminEmail_DenganInstall(t *testing.T) {
	body := BuildAktivasiAdminEmail("Sari", "Petugas", "Bank Sampah A", "", "654321", time.Now(), infoContoh)
	assertAdaBlokInstall(t, body)
	if !strings.Contains(body, "<b>Petugas</b>") {
		t.Error("peran harus tetap disebut")
	}
}

func TestBuildAktivasiAdminEmail_TanpaInstall(t *testing.T) {
	body := BuildAktivasiAdminEmail("Sari", "Admin", "Bank Sampah A", "", "654321", time.Now(), nil)
	assertTanpaBlokInstall(t, body)
}

func TestBlokInstall_EscapeHTML(t *testing.T) {
	body := blokInstallAplikasi(&InstallInfo{URL: `https://x.test/a"b.apk`, VersionName: "<1>"})
	if strings.Contains(body, `a"b`) || strings.Contains(body, "<1>") {
		t.Error("URL dan versi harus di-escape")
	}
}
