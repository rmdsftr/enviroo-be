package controllers

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func renderAngka(t *testing.T, satuan string, nilai float64) string {
	t.Helper()

	f := excelize.NewFile()
	defer f.Close()

	numFmt := formatSatuanReward(satuan)
	style, err := f.NewStyle(&excelize.Style{CustomNumFmt: &numFmt})
	if err != nil {
		t.Fatalf("gagal membuat style untuk satuan %q: %v", satuan, err)
	}
	if err := f.SetCellValue("Sheet1", "A1", nilai); err != nil {
		t.Fatalf("gagal menulis nilai: %v", err)
	}
	if err := f.SetCellStyle("Sheet1", "A1", "A1", style); err != nil {
		t.Fatalf("gagal memasang style: %v", err)
	}

	hasil, err := f.GetCellValue("Sheet1", "A1")
	if err != nil {
		t.Fatalf("gagal membaca nilai: %v", err)
	}
	return hasil
}

func TestFormatSatuanReward_RupiahJadiPrefix(t *testing.T) {
	hasil := renderAngka(t, "Rp", 100000)
	if hasil != "Rp100,000.00" {
		t.Errorf("rupiah harus jadi prefix, dapat %q", hasil)
	}
}

func TestFormatSatuanReward_PoinJadiSuffix(t *testing.T) {
	hasil := renderAngka(t, "poin", 100)
	if hasil != "100.00 poin" {
		t.Errorf("poin harus jadi suffix, dapat %q", hasil)
	}
}

func TestFormatSatuanReward_DefaultRupiah(t *testing.T) {
	if got := formatSatuanReward(""); got != `"Rp"#,##0.00` {
		t.Errorf("satuan kosong harus jatuh ke rupiah, dapat %q", got)
	}
	if got := formatSatuanReward("POIN"); got != `#,##0.00" poin"` {
		t.Errorf("perbandingan satuan harus abai huruf besar-kecil, dapat %q", got)
	}
}
