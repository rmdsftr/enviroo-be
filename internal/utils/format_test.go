package utils_test

import (
	"testing"

	"enviroo-be/internal/models"
	"enviroo-be/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/xuri/excelize/v2"
)

const (
	formatExcelPoin   = `#,##0.00" poin"`
	formatExcelRupiah = `"Rp"#,##0.00`
)

func TestFormatFloat(t *testing.T) {
	kasus := []struct {
		masukan float64
		harapan string
	}{
		{2, "2"},
		{100, "100"},
		{2.5, "2.5"},
		{0.125, "0.125"},
		{1.23456, "1.2346"},
		{0, "0"},
		{-1.5, "-1.5"},
	}
	for _, k := range kasus {
		assert.Equal(t, k.harapan, utils.FormatFloat(k.masukan), "%v", k.masukan)
	}
}

func formatBagiHasil(t *testing.T, nilai float64, satuan models.SatuanRewardEnum) string {
	t.Helper()
	hasil, err := utils.FormatBagiHasilNilai(nilai, satuan)
	assert.NoError(t, err, "%v %s", nilai, satuan)
	return hasil
}

func TestFormatBagiHasilNilai_Rupiah(t *testing.T) {
	kasus := []struct {
		nilai   float64
		harapan string
	}{
		{0, "Rp0"},
		{999, "Rp999"},
		{1000, "Rp1.000"},
		{100000, "Rp100.000"},
		{1234567, "Rp1.234.567"},
	}
	for _, k := range kasus {
		assert.Equal(t, k.harapan, formatBagiHasil(t, k.nilai, models.SatuanRewardEnumRp), "%v", k.nilai)
	}
}

func TestFormatBagiHasilNilai_Poin(t *testing.T) {
	assert.Equal(t, "150 poin", formatBagiHasil(t, 150, models.SatuanRewardEnumPoin))
	assert.Equal(t, "0 poin", formatBagiHasil(t, 0, models.SatuanRewardEnumPoin))
}

func TestFormatBagiHasilNilai_Dibulatkan(t *testing.T) {
	kasus := []struct {
		nilai   float64
		satuan  models.SatuanRewardEnum
		harapan string
	}{
		{1234.99, models.SatuanRewardEnumRp, "Rp1.235"},
		{1234.4, models.SatuanRewardEnumRp, "Rp1.234"},
		{1234.5, models.SatuanRewardEnumRp, "Rp1.235"},
		{999.9, models.SatuanRewardEnumRp, "Rp1.000"},
		{12.4, models.SatuanRewardEnumPoin, "12 poin"},
		{12.5, models.SatuanRewardEnumPoin, "13 poin"},
		{13.5, models.SatuanRewardEnumPoin, "14 poin"},
	}
	for _, k := range kasus {
		assert.Equal(t, k.harapan, formatBagiHasil(t, k.nilai, k.satuan), "%v %s", k.nilai, k.satuan)
	}
}

func TestFormatBagiHasilNilai_NegatifError(t *testing.T) {
	for _, satuan := range []models.SatuanRewardEnum{models.SatuanRewardEnumRp, models.SatuanRewardEnumPoin} {
		for _, nilai := range []float64{-1, -0.4, -123456} {
			hasil, err := utils.FormatBagiHasilNilai(nilai, satuan)

			assert.Empty(t, hasil, "%v %s", nilai, satuan)
			assert.EqualError(t, err, "nilai bagi hasil tidak boleh negatif", "%v %s", nilai, satuan)
		}
	}
}

func TestFormatReward_KodeFormat(t *testing.T) {
	for _, satuan := range []string{"poin", "POIN", "Poin"} {
		assert.Equal(t, formatExcelPoin, utils.FormatReward(satuan), "perbandingan satuan abai huruf besar-kecil: %q", satuan)
	}
	for _, satuan := range []string{"Rp", "", "kg"} {
		assert.Equal(t, formatExcelRupiah, utils.FormatReward(satuan), "selain poin jatuh ke rupiah: %q", satuan)
	}
}

func tampilanExcel(t *testing.T, satuan string, nilai float64) string {
	t.Helper()

	f := excelize.NewFile()
	defer f.Close()

	numFmt := utils.FormatReward(satuan)
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

func TestFormatReward_TampilDiExcel(t *testing.T) {
	assert.Equal(t, "Rp100,000.00", tampilanExcel(t, "Rp", 100000), "rupiah jadi prefix")
	assert.Equal(t, "100.00 poin", tampilanExcel(t, "poin", 100), "poin jadi suffix")
}
