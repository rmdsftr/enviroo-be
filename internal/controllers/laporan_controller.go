package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type LaporanController struct {
	DB *gorm.DB
}

func NewLaporanController(db *gorm.DB) *LaporanController {
	return &LaporanController{DB: db}
}

// GET /laporan/penimbangan/:penimbangan_id
func (lc *LaporanController) DownloadLaporanPenimbangan(c *gin.Context) {
	penimbanganID := c.Param("penimbangan_id")

	// ── 1. Header penimbangan ─────────────────────────────────────────────────
	type headerData struct {
		PenimbanganID      string     `gorm:"column:penimbangan_id"`
		NamaBank           string     `gorm:"column:nama_bank"`
		StartedAt          *time.Time `gorm:"column:started_at"`
		EndedAt            *time.Time `gorm:"column:ended_at"`
		StatusPenimbangan  string     `gorm:"column:status_penimbangan"`
		NamaPetugasMulai   string     `gorm:"column:nama_petugas_mulai"`
		NamaPetugasSelesai string     `gorm:"column:nama_petugas_selesai"`
	}

	var hdr headerData
	if err := lc.DB.Raw(`
		SELECT
			p.penimbangan_id,
			bs.nama_bank,
			p.started_at,
			p.ended_at,
			p.status_penimbangan,
			COALESCE(u_start.nama, '-') AS nama_petugas_mulai,
			COALESCE(u_end.nama,   '-') AS nama_petugas_selesai
		FROM penimbangan p
		JOIN bank_sampah bs     ON bs.bank_id      = p.bank_id
		LEFT JOIN users u_start ON u_start.user_id  = p.started_by
		LEFT JOIN users u_end   ON u_end.user_id    = p.ended_by
		WHERE p.penimbangan_id = ?
	`, penimbanganID).Scan(&hdr).Error; err != nil || hdr.PenimbanganID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Penimbangan tidak ditemukan"})
		return
	}

	// ── 2. Detail setoran per item sampah ─────────────────────────────────────
	type detailRow struct {
		SetoranID     string    `gorm:"column:setoran_id"`
		NasabahID     string    `gorm:"column:nasabah_id"`
		NamaNasabah   string    `gorm:"column:nama_nasabah"`
		NomorRekening string    `gorm:"column:nomor_rekening"`
		WaktuSetoran  time.Time `gorm:"column:waktu_setoran"`
		NamaSampah    string    `gorm:"column:nama_sampah"`
		Satuan        string    `gorm:"column:satuan"`
		Qty           float64   `gorm:"column:qty"`
	}

	var details []detailRow
	if err := lc.DB.Raw(`
		SELECT
			sn.setoran_id,
			n.nasabah_id,
			u_n.nama          AS nama_nasabah,
			n.nomor_rekening,
			sn.created_at     AS waktu_setoran,
			s.nama_sampah,
			s.satuan,
			dsn.qty
		FROM setoran_nasabah sn
		JOIN nasabah n                   ON n.nasabah_id   = sn.nasabah_id
		JOIN users u_n                   ON u_n.user_id    = n.user_id
		JOIN detail_setoran_nasabah dsn  ON dsn.setoran_id = sn.setoran_id
		JOIN katalog_sampah ks           ON ks.sampah_id   = dsn.sampah_id
		JOIN sampah s                    ON s.sarok_id     = ks.sarok_id
		WHERE sn.penimbangan_id = ?
		ORDER BY sn.created_at ASC, s.nama_sampah ASC
	`, penimbanganID).Scan(&details).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data setoran: " + err.Error()})
		return
	}

	// ── 3. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Laporan Penimbangan"
	f.SetSheetName("Sheet1", sheet)

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2E7D32"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Row 1: Title ──────────────────────────────────────────────────────────
	f.MergeCell(sheet, "A1", "H1")
	f.SetCellValue(sheet, "A1", "LAPORAN PENIMBANGAN SAMPAH")
	f.SetCellStyle(sheet, "A1", "H1", styleTitle)
	f.SetRowHeight(sheet, 1, 32)

	// ── Rows 3+: Info header ──────────────────────────────────────────────────
	// Layout: A:B = label (bold), C:H = value
	tanggal, waktuMulai, waktuSelesai := "-", "-", "-"
	if hdr.StartedAt != nil {
		tanggal = hdr.StartedAt.Format("02 January 2006")
		waktuMulai = hdr.StartedAt.Format("15:04:05")
	}
	if hdr.EndedAt != nil {
		waktuSelesai = hdr.EndedAt.Format("15:04:05")
	}

	infoRows := [][2]string{
		{"Nama Bank Sampah", hdr.NamaBank},
		{"ID Penimbangan", hdr.PenimbanganID},
		{"Tanggal", tanggal},
		{"Waktu Mulai", waktuMulai},
		{"Waktu Selesai", waktuSelesai},
		{"Petugas Mulai", hdr.NamaPetugasMulai},
		{"Petugas Selesai", hdr.NamaPetugasSelesai},
		{"Status", hdr.StatusPenimbangan},
	}

	const infoStartRow = 3
	for i, row := range infoRows {
		r := infoStartRow + i
		labelCell := fmt.Sprintf("A%d", r)
		labelEnd := fmt.Sprintf("B%d", r)
		valueCell := fmt.Sprintf("C%d", r)
		valueEnd := fmt.Sprintf("H%d", r)

		f.MergeCell(sheet, labelCell, labelEnd)
		f.SetCellValue(sheet, labelCell, row[0])
		f.SetCellStyle(sheet, labelCell, labelEnd, styleBold)

		f.MergeCell(sheet, valueCell, valueEnd)
		f.SetCellValue(sheet, valueCell, row[1])
	}

	// ── Table header ──────────────────────────────────────────────────────────
	// Blank row between info and table = infoStartRow + len(infoRows)
	tableHeaderRow := infoStartRow + len(infoRows) + 1
	headers := []string{"No", "Nasabah ID", "Nama Nasabah", "No. Rekening", "Waktu Setoran", "Nama Sampah", "Satuan", "Qty"}
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], tableHeaderRow)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheet, tableHeaderRow, 25)

	// ── Data rows ─────────────────────────────────────────────────────────────
	rowIdx := tableHeaderRow + 1
	nasabahNo := 0
	lastSetoran := ""

	for _, d := range details {
		isFirst := d.SetoranID != lastSetoran
		if isFirst {
			nasabahNo++
			lastSetoran = d.SetoranID
		}

		// Alternating style per grup nasabah
		bSt := styleBorder
		bcSt := styleBorderCenter
		if nasabahNo%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
		}

		if isFirst {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), nasabahNo)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), d.NasabahID)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), d.NamaNasabah)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), d.NomorRekening)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), d.WaktuSetoran.Format("02/01/2006 15:04"))
		}
		// Style tetap diterapkan di semua baris (termasuk non-first) agar border konsisten
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bSt)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), d.NamaSampah)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), d.Qty)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), bcSt)

		rowIdx++
	}

	// ── Column widths ─────────────────────────────────────────────────────────
	// A+B bersama jadi label info (gabungan ~28 char), di tabel A=No, B=NasabahID
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 24)
	f.SetColWidth(sheet, "C", "C", 26)
	f.SetColWidth(sheet, "D", "D", 18)
	f.SetColWidth(sheet, "E", "E", 20)
	f.SetColWidth(sheet, "F", "F", 26)
	f.SetColWidth(sheet, "G", "G", 10)
	f.SetColWidth(sheet, "H", "H", 10)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-penimbangan-%s.xlsx", penimbanganID)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/penimbangan/rekap/:bank_id?start=YYYY-MM-DD&end=YYYY-MM-DD
func (lc *LaporanController) DownloadLaporanRekapPenimbangan(c *gin.Context) {
	bankID := c.Param("bank_id")

	startStr := c.Query("start")
	endStr := c.Query("end")
	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter start dan end wajib diisi (format YYYY-MM-DD)"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format start tidak valid, gunakan YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format end tidak valid, gunakan YYYY-MM-DD"})
		return
	}
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tanggal end tidak boleh sebelum start"})
		return
	}
	endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// ── 1. Info bank sampah ───────────────────────────────────────────────────
	type bankData struct {
		BankID   string `gorm:"column:bank_id"`
		NamaBank string `gorm:"column:nama_bank"`
	}
	var bank bankData
	if err := lc.DB.Raw(`SELECT bank_id, nama_bank FROM bank_sampah WHERE bank_id = ?`, bankID).Scan(&bank).Error; err != nil || bank.BankID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	// ── 2. Rekap per sampah ───────────────────────────────────────────────────
	type sampahRow struct {
		NamaSampah string  `gorm:"column:nama_sampah"`
		Kategori   string  `gorm:"column:kategori"`
		Qty        float64 `gorm:"column:qty"`
		Satuan     string  `gorm:"column:satuan"`
	}
	var sampahRows []sampahRow
	if err := lc.DB.Raw(`
		SELECT
			s.nama_sampah,
			kat.kategori,
			SUM(dsn.qty) AS qty,
			s.satuan
		FROM setoran_nasabah sn
		JOIN penimbangan p              ON p.penimbangan_id = sn.penimbangan_id
		JOIN detail_setoran_nasabah dsn ON dsn.setoran_id    = sn.setoran_id
		JOIN katalog_sampah ks          ON ks.sampah_id      = dsn.sampah_id
		JOIN sampah s                   ON s.sarok_id        = ks.sarok_id
		JOIN kategori_sampah kat        ON kat.kategori_id   = ks.kategori_id
		WHERE p.bank_id = ? AND sn.created_at BETWEEN ? AND ?
		GROUP BY s.sarok_id, s.nama_sampah, kat.kategori, s.satuan
		ORDER BY s.nama_sampah ASC
	`, bankID, startDate, endDate).Scan(&sampahRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil rekap sampah: " + err.Error()})
		return
	}

	// ── 3. Rekap per nasabah ──────────────────────────────────────────────────
	type nasabahDetailRow struct {
		NasabahID   string  `gorm:"column:nasabah_id"`
		NamaNasabah string  `gorm:"column:nama_nasabah"`
		NamaSampah  string  `gorm:"column:nama_sampah"`
		Qty         float64 `gorm:"column:qty"`
		Satuan      string  `gorm:"column:satuan"`
	}
	var nasabahRows []nasabahDetailRow
	if err := lc.DB.Raw(`
		SELECT
			n.nasabah_id,
			u.nama            AS nama_nasabah,
			s.nama_sampah,
			SUM(dsn.qty)      AS qty,
			s.satuan
		FROM setoran_nasabah sn
		JOIN penimbangan p              ON p.penimbangan_id = sn.penimbangan_id
		JOIN nasabah n                  ON n.nasabah_id      = sn.nasabah_id
		JOIN users u                    ON u.user_id         = n.user_id
		JOIN detail_setoran_nasabah dsn ON dsn.setoran_id     = sn.setoran_id
		JOIN katalog_sampah ks          ON ks.sampah_id       = dsn.sampah_id
		JOIN sampah s                   ON s.sarok_id         = ks.sarok_id
		WHERE p.bank_id = ? AND sn.created_at BETWEEN ? AND ?
		GROUP BY n.nasabah_id, u.nama, s.sarok_id, s.nama_sampah, s.satuan
		ORDER BY u.nama ASC, s.nama_sampah ASC
	`, bankID, startDate, endDate).Scan(&nasabahRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil rekap nasabah: " + err.Error()})
		return
	}

	// ── 4. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	periodeLabel := fmt.Sprintf("%s - %s", startDate.Format("02 January 2006"), endDate.Format("02 January 2006"))

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2E7D32"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Sheet 1: Rekap Sampah ─────────────────────────────────────────────────
	sheetSampah := "Rekap Sampah"
	f.SetSheetName("Sheet1", sheetSampah)

	f.MergeCell(sheetSampah, "A1", "D1")
	f.SetCellValue(sheetSampah, "A1", "REKAP PENIMBANGAN SAMPAH")
	f.SetCellStyle(sheetSampah, "A1", "D1", styleTitle)
	f.SetRowHeight(sheetSampah, 1, 32)

	sampahInfoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Periode", periodeLabel},
	}
	const sampahInfoStartRow = 3
	for i, row := range sampahInfoRows {
		r := sampahInfoStartRow + i
		f.MergeCell(sheetSampah, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r))
		f.SetCellValue(sheetSampah, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheetSampah, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), styleBold)
		f.MergeCell(sheetSampah, fmt.Sprintf("B%d", r), fmt.Sprintf("D%d", r))
		f.SetCellValue(sheetSampah, fmt.Sprintf("B%d", r), row[1])
	}

	sampahTableHeaderRow := sampahInfoStartRow + len(sampahInfoRows) + 1
	sampahHeaders := []string{"Nama Sampah", "Kategori", "Qty", "Satuan"}
	sampahCols := []string{"A", "B", "C", "D"}
	for i, h := range sampahHeaders {
		cell := fmt.Sprintf("%s%d", sampahCols[i], sampahTableHeaderRow)
		f.SetCellValue(sheetSampah, cell, h)
		f.SetCellStyle(sheetSampah, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheetSampah, sampahTableHeaderRow, 25)

	rowIdx := sampahTableHeaderRow + 1
	for i, d := range sampahRows {
		bSt := styleBorder
		bcSt := styleBorderCenter
		if (i+1)%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
		}

		f.SetCellValue(sheetSampah, fmt.Sprintf("A%d", rowIdx), d.NamaSampah)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("B%d", rowIdx), d.Kategori)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("C%d", rowIdx), d.Qty)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bcSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("D%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)

		rowIdx++
	}

	f.SetColWidth(sheetSampah, "A", "A", 28)
	f.SetColWidth(sheetSampah, "B", "B", 22)
	f.SetColWidth(sheetSampah, "C", "C", 14)
	f.SetColWidth(sheetSampah, "D", "D", 12)

	// ── Sheet 2: Rekap Nasabah ────────────────────────────────────────────────
	sheetNasabah := "Rekap Nasabah"
	f.NewSheet(sheetNasabah)

	f.MergeCell(sheetNasabah, "A1", "E1")
	f.SetCellValue(sheetNasabah, "A1", "REKAP PENIMBANGAN PER NASABAH")
	f.SetCellStyle(sheetNasabah, "A1", "E1", styleTitle)
	f.SetRowHeight(sheetNasabah, 1, 32)

	nasabahInfoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Periode", periodeLabel},
	}
	const nasabahInfoStartRow = 3
	for i, row := range nasabahInfoRows {
		r := nasabahInfoStartRow + i
		f.MergeCell(sheetNasabah, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r))
		f.SetCellValue(sheetNasabah, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), styleBold)
		f.MergeCell(sheetNasabah, fmt.Sprintf("B%d", r), fmt.Sprintf("E%d", r))
		f.SetCellValue(sheetNasabah, fmt.Sprintf("B%d", r), row[1])
	}

	nasabahTableHeaderRow := nasabahInfoStartRow + len(nasabahInfoRows) + 1
	nasabahHeaders := []string{"No", "Nasabah ID", "Nama Nasabah", "Nama Sampah", "Qty", "Satuan"}
	nasabahCols := []string{"A", "B", "C", "D", "E", "F"}
	for i, h := range nasabahHeaders {
		cell := fmt.Sprintf("%s%d", nasabahCols[i], nasabahTableHeaderRow)
		f.SetCellValue(sheetNasabah, cell, h)
		f.SetCellStyle(sheetNasabah, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheetNasabah, nasabahTableHeaderRow, 25)

	rowIdx = nasabahTableHeaderRow + 1
	nasabahNo := 0
	lastNasabahID := ""
	for _, d := range nasabahRows {
		isFirst := d.NasabahID != lastNasabahID
		if isFirst {
			nasabahNo++
			lastNasabahID = d.NasabahID
		}

		bSt := styleBorder
		bcSt := styleBorderCenter
		if nasabahNo%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
		}

		if isFirst {
			f.SetCellValue(sheetNasabah, fmt.Sprintf("A%d", rowIdx), nasabahNo)
			f.SetCellValue(sheetNasabah, fmt.Sprintf("B%d", rowIdx), d.NasabahID)
			f.SetCellValue(sheetNasabah, fmt.Sprintf("C%d", rowIdx), d.NamaNasabah)
		}
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("D%d", rowIdx), d.NamaSampah)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("E%d", rowIdx), d.Qty)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bcSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("F%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bcSt)

		rowIdx++
	}

	f.SetColWidth(sheetNasabah, "A", "A", 6)
	f.SetColWidth(sheetNasabah, "B", "B", 24)
	f.SetColWidth(sheetNasabah, "C", "C", 26)
	f.SetColWidth(sheetNasabah, "D", "D", 26)
	f.SetColWidth(sheetNasabah, "E", "E", 12)
	f.SetColWidth(sheetNasabah, "F", "F", 12)

	f.SetActiveSheet(0)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-rekap-penimbangan-%s-%s-%s.xlsx", bankID, startStr, endStr)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/nasabah/:bank_id
func (lc *LaporanController) DownloadLaporanNasabah(c *gin.Context) {
	bankID := c.Param("bank_id")

	// ── 1. Info bank sampah ───────────────────────────────────────────────────
	type bankData struct {
		BankID   string `gorm:"column:bank_id"`
		NamaBank string `gorm:"column:nama_bank"`
	}
	var bank bankData
	if err := lc.DB.Raw(`SELECT bank_id, nama_bank FROM bank_sampah WHERE bank_id = ?`, bankID).Scan(&bank).Error; err != nil || bank.BankID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	// ── 2. Daftar nasabah ─────────────────────────────────────────────────────
	type nasabahRow struct {
		NasabahID     string    `gorm:"column:nasabah_id"`
		NIK           string    `gorm:"column:nik"`
		Nama          string    `gorm:"column:nama"`
		Email         string    `gorm:"column:email"`
		NomorRekening string    `gorm:"column:nomor_rekening"`
		JoinedAt      time.Time `gorm:"column:joined_at"`
		StatusNasabah string    `gorm:"column:status_nasabah"`
	}
	var rows []nasabahRow
	if err := lc.DB.Raw(`
		SELECT
			n.nasabah_id,
			n.user_id      AS nik,
			u.nama,
			u.email,
			n.nomor_rekening,
			n.joined_at,
			n.status_nasabah
		FROM nasabah n
		JOIN users u ON u.user_id = n.user_id
		WHERE n.bank_id = ?
		ORDER BY n.joined_at ASC
	`, bankID).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data nasabah: " + err.Error()})
		return
	}

	// ── 3. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Daftar Nasabah"
	f.SetSheetName("Sheet1", sheet)

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"2E7D32"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleStatusAktif, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "1B5E20"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8F5E9"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleStatusPending, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "E65100"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFF3E0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleStatusNonaktif, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "B71C1C"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFEBEE"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Row 1: Title ──────────────────────────────────────────────────────────
	f.MergeCell(sheet, "A1", "H1")
	f.SetCellValue(sheet, "A1", "DAFTAR NASABAH BANK SAMPAH")
	f.SetCellStyle(sheet, "A1", "H1", styleTitle)
	f.SetRowHeight(sheet, 1, 32)

	// ── Rows 3+: Info header ──────────────────────────────────────────────────
	infoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Bank ID", bank.BankID},
		{"Tanggal Cetak", time.Now().Format("02 January 2006, 15:04:05")},
		{"Total Nasabah", fmt.Sprintf("%d nasabah", len(rows))},
	}

	const infoStartRow = 3
	for i, row := range infoRows {
		r := infoStartRow + i
		f.MergeCell(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r), styleBold)
		f.MergeCell(sheet, fmt.Sprintf("C%d", r), fmt.Sprintf("H%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row[1])
	}

	// ── Table header ──────────────────────────────────────────────────────────
	tableHeaderRow := infoStartRow + len(infoRows) + 1
	headers := []string{"No", "Nasabah ID", "NIK", "Nama Nasabah", "Email", "No. Rekening", "Tanggal Bergabung", "Status"}
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], tableHeaderRow)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheet, tableHeaderRow, 25)

	// ── Data rows ─────────────────────────────────────────────────────────────
	rowIdx := tableHeaderRow + 1
	for i, d := range rows {
		no := i + 1
		bSt := styleBorder
		bcSt := styleBorderCenter
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), no)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), d.NasabahID)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), d.NIK)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), d.Nama)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), d.Email)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), d.NomorRekening)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), d.JoinedAt.Format("02 January 2006"))
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), bcSt)

		statusStyle := bcSt
		switch d.StatusNasabah {
		case "aktif":
			statusStyle = styleStatusAktif
		case "pending":
			statusStyle = styleStatusPending
		case "nonaktif":
			statusStyle = styleStatusNonaktif
		}
		f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), d.StatusNasabah)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), statusStyle)

		rowIdx++
	}

	// ── Column widths ─────────────────────────────────────────────────────────
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 28)
	f.SetColWidth(sheet, "C", "C", 20)
	f.SetColWidth(sheet, "D", "D", 26)
	f.SetColWidth(sheet, "E", "E", 30)
	f.SetColWidth(sheet, "F", "F", 18)
	f.SetColWidth(sheet, "G", "G", 22)
	f.SetColWidth(sheet, "H", "H", 12)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-nasabah-%s.xlsx", bankID)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/pengangkutan/:pengangkutan_id
func (lc *LaporanController) DownloadLaporanPengangkutan(c *gin.Context) {
	pengangkutanID := c.Param("pengangkutan_id")

	// ── 1. Header pengangkutan ────────────────────────────────────────────────
	type headerData struct {
		PengangkutanID string    `gorm:"column:pengangkutan_id"`
		NamaBSI        string    `gorm:"column:nama_bsi"`
		NamaBSU        string    `gorm:"column:nama_bsu"`
		NamaAdminBSI   string    `gorm:"column:nama_admin_bsi"`
		NamaAdminBSU   string    `gorm:"column:nama_admin_bsu"`
		StatusTerakhir string    `gorm:"column:status_terakhir"`
		TanggalMulai   time.Time `gorm:"column:tanggal_mulai"`
		TanggalSelesai time.Time `gorm:"column:tanggal_selesai"`
	}

	var hdr headerData
	if err := lc.DB.Raw(`
		SELECT
			ps.pengangkutan_id,
			bsi.nama_bank AS nama_bsi,
			bsu.nama_bank AS nama_bsu,
			COALESCE(u_bsi.nama, '-') AS nama_admin_bsi,
			COALESCE(u_bsu.nama, '-') AS nama_admin_bsu,
			rp_last.status_pengangkutan AS status_terakhir,
			rp_first.changed_at AS tanggal_mulai,
			rp_last.changed_at AS tanggal_selesai
		FROM pengangkutan_sampah ps
		LEFT JOIN bank_sampah bsi ON bsi.bank_id = ps.bsi_id
		LEFT JOIN bank_sampah bsu ON bsu.bank_id = ps.bsu_id
		LEFT JOIN admin a_bsi ON a_bsi.admin_id = ps.admin_bsi_id
		LEFT JOIN users u_bsi ON u_bsi.user_id = a_bsi.user_id
		LEFT JOIN admin a_bsu ON a_bsu.admin_id = ps.admin_bsu_id
		LEFT JOIN users u_bsu ON u_bsu.user_id = a_bsu.user_id
		LEFT JOIN riwayat_pengangkutan rp_last ON rp_last.riwayat_pengangkutan_id = (
			SELECT MAX(riwayat_pengangkutan_id) FROM riwayat_pengangkutan WHERE pengangkutan_id = ps.pengangkutan_id
		)
		LEFT JOIN riwayat_pengangkutan rp_first ON rp_first.riwayat_pengangkutan_id = (
			SELECT MIN(riwayat_pengangkutan_id) FROM riwayat_pengangkutan WHERE pengangkutan_id = ps.pengangkutan_id
		)
		WHERE ps.pengangkutan_id = ?
	`, pengangkutanID).Scan(&hdr).Error; err != nil || hdr.PengangkutanID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pengangkutan tidak ditemukan"})
		return
	}

	// ── 2. Guard: pengangkutan harus sudah selesai ────────────────────────────
	if hdr.StatusTerakhir != "completed" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Laporan hanya tersedia untuk pengangkutan yang sudah selesai"})
		return
	}

	// ── 3. Riwayat status (ASC) ───────────────────────────────────────────────
	type riwayatRow struct {
		Status      string    `gorm:"column:status"`
		ChangedAt   time.Time `gorm:"column:changed_at"`
		NamaPetugas string    `gorm:"column:nama_petugas"`
		Notes       string    `gorm:"column:notes"`
	}
	var riwayat []riwayatRow
	if err := lc.DB.Raw(`
		SELECT
			rp.status_pengangkutan AS status,
			rp.changed_at,
			COALESCE(u.nama, rp.changed_by) AS nama_petugas,
			rp.notes
		FROM riwayat_pengangkutan rp
		LEFT JOIN admin a ON a.admin_id = rp.changed_by
		LEFT JOIN users u ON u.user_id = a.user_id
		WHERE rp.pengangkutan_id = ?
		ORDER BY rp.changed_at ASC
	`, pengangkutanID).Scan(&riwayat).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil riwayat: " + err.Error()})
		return
	}

	// ── 4. Detail sampah ──────────────────────────────────────────────────────
	type detailRow struct {
		NamaSampah string  `gorm:"column:nama_sampah"`
		Satuan     string  `gorm:"column:satuan"`
		Qty        float64 `gorm:"column:qty"`
	}
	var details []detailRow
	if err := lc.DB.Raw(`
		SELECT s.nama_sampah, s.satuan, dp.qty
		FROM detail_pengangkutan dp
		JOIN katalog_sampah ks ON ks.sampah_id = dp.sampah_id
		JOIN sampah s ON s.sarok_id = ks.sarok_id
		WHERE dp.pengangkutan_id = ?
		ORDER BY s.nama_sampah ASC
	`, pengangkutanID).Scan(&details).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail sampah: " + err.Error()})
		return
	}

	// ── 5. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Laporan Pengangkutan"
	f.SetSheetName("Sheet1", sheet)

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"BF360C"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleSectionHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "BF360C"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FBE9E7"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "BF360C", Style: 2},
			{Type: "bottom", Color: "BF360C", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleTotalLabel, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FBE9E7"}},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 2},
			{Type: "bottom", Color: "CCCCCC", Style: 2},
		},
	})
	styleTotalNum, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FBE9E7"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 2},
			{Type: "bottom", Color: "CCCCCC", Style: 2},
		},
	})

	// ── Row 1: Title ──────────────────────────────────────────────────────────
	f.MergeCell(sheet, "A1", "E1")
	f.SetCellValue(sheet, "A1", "LAPORAN PENGANGKUTAN SAMPAH")
	f.SetCellStyle(sheet, "A1", "E1", styleTitle)
	f.SetRowHeight(sheet, 1, 32)

	// ── Rows 3+: Info header ──────────────────────────────────────────────────
	infoRows := [][2]string{
		{"ID Pengangkutan", hdr.PengangkutanID},
		{"Bank Sampah Induk (BSI)", hdr.NamaBSI},
		{"Bank Sampah Unit (BSU)", hdr.NamaBSU},
		{"Admin BSI", hdr.NamaAdminBSI},
		{"Admin BSU", hdr.NamaAdminBSU},
		{"Status Terakhir", hdr.StatusTerakhir},
		{"Tanggal Mulai", hdr.TanggalMulai.Format("02 January 2006, 15:04:05")},
		{"Tanggal Selesai", hdr.TanggalSelesai.Format("02 January 2006, 15:04:05")},
	}

	const infoStartRow = 3
	for i, row := range infoRows {
		r := infoStartRow + i
		f.MergeCell(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r), styleBold)
		f.MergeCell(sheet, fmt.Sprintf("C%d", r), fmt.Sprintf("E%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row[1])
	}

	// ── Section 1: Riwayat Status ─────────────────────────────────────────────
	sectionRiwayatRow := infoStartRow + len(infoRows) + 1
	f.MergeCell(sheet, fmt.Sprintf("A%d", sectionRiwayatRow), fmt.Sprintf("E%d", sectionRiwayatRow))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", sectionRiwayatRow), "  RIWAYAT STATUS PENGANGKUTAN")
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", sectionRiwayatRow), fmt.Sprintf("E%d", sectionRiwayatRow), styleSectionHeader)
	f.SetRowHeight(sheet, sectionRiwayatRow, 22)

	riwayatHeaderRow := sectionRiwayatRow + 1
	riwayatHeaders := []string{"No", "Status", "Waktu", "Diubah Oleh", "Catatan"}
	riwayatCols := []string{"A", "B", "C", "D", "E"}
	for i, h := range riwayatHeaders {
		cell := fmt.Sprintf("%s%d", riwayatCols[i], riwayatHeaderRow)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheet, riwayatHeaderRow, 25)

	rowIdx := riwayatHeaderRow + 1
	for i, r := range riwayat {
		no := i + 1
		bSt := styleBorder
		bcSt := styleBorderCenter
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), no)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), r.Status)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), r.ChangedAt.Format("02/01/2006 15:04:05"))
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), r.NamaPetugas)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bSt)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), r.Notes)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bSt)
		rowIdx++
	}

	// ── Section 2: Detail Sampah ──────────────────────────────────────────────
	sectionDetailRow := rowIdx + 1
	f.MergeCell(sheet, fmt.Sprintf("A%d", sectionDetailRow), fmt.Sprintf("E%d", sectionDetailRow))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", sectionDetailRow), "  DETAIL SAMPAH YANG DIANGKUT")
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", sectionDetailRow), fmt.Sprintf("E%d", sectionDetailRow), styleSectionHeader)
	f.SetRowHeight(sheet, sectionDetailRow, 22)

	detailHeaderRow := sectionDetailRow + 1
	detailHeaders := []string{"No", "Nama Sampah", "Satuan", "Qty", ""}
	for i, h := range detailHeaders {
		cell := fmt.Sprintf("%s%d", riwayatCols[i], detailHeaderRow)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheet, detailHeaderRow, 25)

	rowIdx = detailHeaderRow + 1
	for i, d := range details {
		no := i + 1
		bSt := styleBorder
		bcSt := styleBorderCenter
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), no)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), d.NamaSampah)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), d.Qty)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bSt)
		rowIdx++
	}

	// ── Total row ─────────────────────────────────────────────────────────────
	totalRow := rowIdx + 1
	f.MergeCell(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("C%d", totalRow))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", totalRow), "TOTAL JENIS SAMPAH")
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("C%d", totalRow), styleTotalLabel)
	f.SetCellValue(sheet, fmt.Sprintf("D%d", totalRow), len(details))
	f.SetCellStyle(sheet, fmt.Sprintf("D%d", totalRow), fmt.Sprintf("D%d", totalRow), styleTotalNum)
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", totalRow), fmt.Sprintf("E%d", totalRow), styleTotalLabel)
	f.SetRowHeight(sheet, totalRow, 22)

	// ── Column widths ─────────────────────────────────────────────────────────
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 26)
	f.SetColWidth(sheet, "C", "C", 24)
	f.SetColWidth(sheet, "D", "D", 22)
	f.SetColWidth(sheet, "E", "E", 36)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-pengangkutan-%s.xlsx", pengangkutanID)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/bagi-hasil/:bagi_hasil_id
func (lc *LaporanController) DownloadLaporanBagiHasil(c *gin.Context) {
	bagiHasilID := c.Param("bagi_hasil_id")

	// ── 1. Header bagi hasil ──────────────────────────────────────────────────
	type headerData struct {
		BagiHasilID            string    `gorm:"column:bagi_hasil_id"`
		NamaBank               string    `gorm:"column:nama_bank"`
		PenjualanID            string    `gorm:"column:penjualan_id"`
		CreatedAt              time.Time `gorm:"column:created_at"`
		NamaPetugas            string    `gorm:"column:nama_petugas"`
		NamaReward             string    `gorm:"column:nama_reward"`
		GrossBank              float64   `gorm:"column:gross_bank"`
		TotalDistribusiNasabah float64   `gorm:"column:total_distribusi_nasabah"`
		SisaBagiHasil          float64   `gorm:"column:sisa_bagi_hasil"`
		SatuanBagiHasil        string    `gorm:"column:satuan_bagi_hasil"`
	}

	var hdr headerData
	if err := lc.DB.Raw(`
		SELECT
			bh.bagi_hasil_id,
			bs.nama_bank,
			bh.penjualan_id,
			bh.created_at,
			COALESCE(u.nama, '-') AS nama_petugas,
			r.nama_reward,
			bh.gross_bank,
			bh.total_distribusi_nasabah,
			bh.sisa_bagi_hasil,
			bh.satuan_bagi_hasil
		FROM bagi_hasil bh
		LEFT JOIN bank_sampah bs  ON bs.bank_id    = bh.bank_id
		LEFT JOIN penjualan p     ON p.penjualan_id = bh.penjualan_id
		LEFT JOIN reward r        ON r.reward_id   = p.reward_id
		LEFT JOIN admin a         ON a.admin_id    = bh.created_by
		LEFT JOIN users u         ON u.user_id     = a.user_id
		WHERE bh.bagi_hasil_id = ?
	`, bagiHasilID).Scan(&hdr).Error; err != nil || hdr.BagiHasilID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Data bagi hasil tidak ditemukan"})
		return
	}

	// ── 2. Nasabah penerima (ringkasan per nasabah) ───────────────────────────
	type nasabahRow struct {
		NasabahID     string  `gorm:"column:nasabah_id"`
		NamaNasabah   string  `gorm:"column:nama_nasabah"`
		NamaBank      string  `gorm:"column:nama_bank"`
		TotalDiterima float64 `gorm:"column:total_diterima"`
		Satuan        string  `gorm:"column:satuan"`
	}

	var nasabahList []nasabahRow
	if err := lc.DB.Raw(`
		SELECT
			n.nasabah_id,
			u.nama      AS nama_nasabah,
			bs.nama_bank AS nama_bank,
			pbh.total_diterima,
			pbh.satuan_diterima AS satuan
		FROM penerima_bagi_hasil pbh
		LEFT JOIN nasabah n      ON n.nasabah_id  = pbh.nasabah_id
		LEFT JOIN users u        ON u.user_id     = n.user_id
		LEFT JOIN bank_sampah bs ON bs.bank_id    = n.bank_id
		WHERE pbh.bagi_hasil_id = ?
		ORDER BY bs.nama_bank ASC, u.nama ASC
	`, bagiHasilID).Scan(&nasabahList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penerima: " + err.Error()})
		return
	}

	// ── 3. Distribusi sisa (opsional) ─────────────────────────────────────────
	type distribusiHeader struct {
		DistribusiID      string    `gorm:"column:distribusi_id"`
		TanggalDistribusi time.Time `gorm:"column:tanggal_distribusi"`
		PetugasDistribusi string    `gorm:"column:petugas_distribusi"`
		TotalSisa         float64   `gorm:"column:total_sisa"`
		SatuanTotal       string    `gorm:"column:satuan_total"`
	}

	var ds distribusiHeader
	hasDistribusiSisa := false
	lc.DB.Raw(`
		SELECT
			ds.distribusi_id,
			ds.created_at  AS tanggal_distribusi,
			COALESCE(u.nama, ds.created_by) AS petugas_distribusi,
			ds.total_sisa,
			ds.satuan_total
		FROM distribusi_sisa ds
		LEFT JOIN admin a ON a.admin_id = ds.created_by
		LEFT JOIN users u ON u.user_id  = a.user_id
		WHERE ds.bagi_hasil_id = ?
		LIMIT 1
	`, bagiHasilID).Scan(&ds)
	if ds.DistribusiID != "" {
		hasDistribusiSisa = true
	}

	type penerimaSisaRow struct {
		NamaBank        string  `gorm:"column:nama_bank"`
		JenisPenerimaan string  `gorm:"column:jenis_penerimaan"`
		Porsi           float64 `gorm:"column:porsi"`
		Transportasi    float64 `gorm:"column:transportasi"`
		NominalDiterima float64 `gorm:"column:nominal_diterima"`
		Satuan          string  `gorm:"column:satuan"`
	}

	var penerimaSisaList []penerimaSisaRow
	if hasDistribusiSisa {
		if err := lc.DB.Raw(`
			SELECT
				bs.nama_bank,
				pds.jenis_penerimaan,
				pds.porsi,
				pds.transportasi,
				pds.nominal_diterima,
				pds.satuan_nominal AS satuan
			FROM penerima_distribusi_sisa pds
			LEFT JOIN bank_sampah bs ON bs.bank_id = pds.bank_id
			WHERE pds.distribusi_id = ?
			ORDER BY pds.jenis_penerimaan ASC
		`, ds.DistribusiID).Scan(&penerimaSisaList).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data penerima sisa: " + err.Error()})
			return
		}
	}

	// ── 4. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Laporan Bagi Hasil"
	f.SetSheetName("Sheet1", sheet)

	numFmt := "#,##0.00"

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4527A0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleSectionHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "4527A0"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EDE7F6"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "4527A0", Style: 2},
			{Type: "bottom", Color: "4527A0", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNum, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNumAlt, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleTotalLabel, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EDE7F6"}},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 2},
			{Type: "bottom", Color: "CCCCCC", Style: 2},
		},
	})
	styleTotalNum, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		CustomNumFmt: &numFmt,
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EDE7F6"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 2},
			{Type: "bottom", Color: "CCCCCC", Style: 2},
		},
	})

	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	maxCol := "H"

	cell := func(col string, row int) string {
		return fmt.Sprintf("%s%d", col, row)
	}
	mergeRow := func(from, to string, row int) {
		f.MergeCell(sheet, cell(from, row), cell(to, row))
	}
	styleRange := func(from, to string, row int, styleID int) {
		f.SetCellStyle(sheet, cell(from, row), cell(to, row), styleID)
	}

	// ── Row 1: Title ──────────────────────────────────────────────────────────
	mergeRow("A", maxCol, 1)
	f.SetCellValue(sheet, cell("A", 1), "LAPORAN BAGI HASIL SAMPAH")
	styleRange("A", maxCol, 1, styleTitle)
	f.SetRowHeight(sheet, 1, 32)

	// ── Rows 3+: Info header ──────────────────────────────────────────────────
	infoRows := [][2]string{
		{"ID Bagi Hasil", hdr.BagiHasilID},
		{"Bank Pelaksana", hdr.NamaBank},
		{"ID Penjualan", hdr.PenjualanID},
		{"Tanggal Bagi Hasil", hdr.CreatedAt.Format("02 January 2006, 15:04:05")},
		{"Petugas", hdr.NamaPetugas},
		{"Jenis Reward", hdr.NamaReward},
		{"Gross Bank", fmt.Sprintf("%.2f %s", hdr.GrossBank, hdr.SatuanBagiHasil)},
		{"Total Distribusi Nasabah", fmt.Sprintf("%.2f %s", hdr.TotalDistribusiNasabah, hdr.SatuanBagiHasil)},
		{"Sisa Bagi Hasil", fmt.Sprintf("%.2f %s", hdr.SisaBagiHasil, hdr.SatuanBagiHasil)},
	}

	const infoStartRow = 3
	for i, row := range infoRows {
		r := infoStartRow + i
		mergeRow("A", "B", r)
		f.SetCellValue(sheet, cell("A", r), row[0])
		styleRange("A", "B", r, styleBold)
		mergeRow("C", maxCol, r)
		f.SetCellValue(sheet, cell("C", r), row[1])
	}

	// ── Section 1: Distribusi Nasabah ─────────────────────────────────────────
	secNasabahRow := infoStartRow + len(infoRows) + 1
	mergeRow("A", maxCol, secNasabahRow)
	f.SetCellValue(sheet, cell("A", secNasabahRow), "  DISTRIBUSI KE NASABAH")
	styleRange("A", maxCol, secNasabahRow, styleSectionHeader)
	f.SetRowHeight(sheet, secNasabahRow, 22)

	nasabahHeaderRow := secNasabahRow + 1
	nasabahHeaders := []string{"No", "Nasabah ID", "Nama Nasabah", "Bank", "Total Diterima", "Satuan", "", ""}
	for i, h := range nasabahHeaders {
		f.SetCellValue(sheet, cell(cols[i], nasabahHeaderRow), h)
		f.SetCellStyle(sheet, cell(cols[i], nasabahHeaderRow), cell(cols[i], nasabahHeaderRow), styleTableHeader)
	}
	f.SetRowHeight(sheet, nasabahHeaderRow, 25)

	rowIdx := nasabahHeaderRow + 1
	var totalNasabah float64
	lastSatuan := ""
	for i, n := range nasabahList {
		no := i + 1
		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}
		f.SetCellValue(sheet, cell("A", rowIdx), no)
		f.SetCellStyle(sheet, cell("A", rowIdx), cell("A", rowIdx), bcSt)
		f.SetCellValue(sheet, cell("B", rowIdx), n.NasabahID)
		f.SetCellStyle(sheet, cell("B", rowIdx), cell("B", rowIdx), bSt)
		f.SetCellValue(sheet, cell("C", rowIdx), n.NamaNasabah)
		f.SetCellStyle(sheet, cell("C", rowIdx), cell("C", rowIdx), bSt)
		f.SetCellValue(sheet, cell("D", rowIdx), n.NamaBank)
		f.SetCellStyle(sheet, cell("D", rowIdx), cell("D", rowIdx), bSt)
		f.SetCellValue(sheet, cell("E", rowIdx), n.TotalDiterima)
		f.SetCellStyle(sheet, cell("E", rowIdx), cell("E", rowIdx), bnSt)
		f.SetCellValue(sheet, cell("F", rowIdx), n.Satuan)
		f.SetCellStyle(sheet, cell("F", rowIdx), cell("F", rowIdx), bcSt)
		f.SetCellStyle(sheet, cell("G", rowIdx), cell("G", rowIdx), bSt)
		f.SetCellStyle(sheet, cell("H", rowIdx), cell("H", rowIdx), bSt)
		totalNasabah += n.TotalDiterima
		lastSatuan = n.Satuan
		rowIdx++
	}

	totalNasabahRow := rowIdx + 1
	mergeRow("A", "D", totalNasabahRow)
	f.SetCellValue(sheet, cell("A", totalNasabahRow), fmt.Sprintf("TOTAL (%d nasabah)", len(nasabahList)))
	styleRange("A", "D", totalNasabahRow, styleTotalLabel)
	f.SetCellValue(sheet, cell("E", totalNasabahRow), totalNasabah)
	f.SetCellStyle(sheet, cell("E", totalNasabahRow), cell("E", totalNasabahRow), styleTotalNum)
	f.SetCellValue(sheet, cell("F", totalNasabahRow), lastSatuan)
	styleRange("F", maxCol, totalNasabahRow, styleTotalLabel)
	f.SetRowHeight(sheet, totalNasabahRow, 22)

	// ── Section 2: Distribusi Sisa (opsional) ─────────────────────────────────
	if hasDistribusiSisa {
		secSisaRow := totalNasabahRow + 2
		mergeRow("A", maxCol, secSisaRow)
		f.SetCellValue(sheet, cell("A", secSisaRow), "  DISTRIBUSI SISA BAGI HASIL")
		styleRange("A", maxCol, secSisaRow, styleSectionHeader)
		f.SetRowHeight(sheet, secSisaRow, 22)

		// Info distribusi sisa
		sisaInfoRows := [][2]string{
			{"ID Distribusi", ds.DistribusiID},
			{"Tanggal Distribusi", ds.TanggalDistribusi.Format("02 January 2006, 15:04:05")},
			{"Petugas Distribusi", ds.PetugasDistribusi},
			{"Total Sisa", fmt.Sprintf("%.2f %s", ds.TotalSisa, ds.SatuanTotal)},
		}
		sisaInfoStart := secSisaRow + 1
		for i, row := range sisaInfoRows {
			r := sisaInfoStart + i
			mergeRow("A", "B", r)
			f.SetCellValue(sheet, cell("A", r), row[0])
			styleRange("A", "B", r, styleBold)
			mergeRow("C", maxCol, r)
			f.SetCellValue(sheet, cell("C", r), row[1])
		}

		sisaTableHeaderRow := sisaInfoStart + len(sisaInfoRows) + 1
		sisaHeaders := []string{"No", "Bank Penerima", "Jenis Penerimaan", "Porsi (%)", "Transportasi", "Nominal Diterima", "Satuan"}
		for i, h := range sisaHeaders {
			f.SetCellValue(sheet, cell(cols[i], sisaTableHeaderRow), h)
			f.SetCellStyle(sheet, cell(cols[i], sisaTableHeaderRow), cell(cols[i], sisaTableHeaderRow), styleTableHeader)
		}
		f.SetRowHeight(sheet, sisaTableHeaderRow, 25)

		rowIdx = sisaTableHeaderRow + 1
		for i, p := range penerimaSisaList {
			no := i + 1
			bSt := styleBorder
			bcSt := styleBorderCenter
			bnSt := styleBorderNum
			if no%2 == 0 {
				bSt = styleBorderAlt
				bcSt = styleBorderCenterAlt
				bnSt = styleBorderNumAlt
			}
			f.SetCellValue(sheet, cell("A", rowIdx), no)
			f.SetCellStyle(sheet, cell("A", rowIdx), cell("A", rowIdx), bcSt)
			f.SetCellValue(sheet, cell("B", rowIdx), p.NamaBank)
			f.SetCellStyle(sheet, cell("B", rowIdx), cell("B", rowIdx), bSt)
			f.SetCellValue(sheet, cell("C", rowIdx), p.JenisPenerimaan)
			f.SetCellStyle(sheet, cell("C", rowIdx), cell("C", rowIdx), bcSt)
			f.SetCellValue(sheet, cell("D", rowIdx), p.Porsi)
			f.SetCellStyle(sheet, cell("D", rowIdx), cell("D", rowIdx), bnSt)
			f.SetCellValue(sheet, cell("E", rowIdx), p.Transportasi)
			f.SetCellStyle(sheet, cell("E", rowIdx), cell("E", rowIdx), bnSt)
			f.SetCellValue(sheet, cell("F", rowIdx), p.NominalDiterima)
			f.SetCellStyle(sheet, cell("F", rowIdx), cell("F", rowIdx), bnSt)
			f.SetCellValue(sheet, cell("G", rowIdx), p.Satuan)
			f.SetCellStyle(sheet, cell("G", rowIdx), cell("G", rowIdx), bcSt)
			f.SetCellStyle(sheet, cell("H", rowIdx), cell("H", rowIdx), bSt)
			rowIdx++
		}
	}

	// ── Column widths ─────────────────────────────────────────────────────────
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 24)
	f.SetColWidth(sheet, "C", "C", 26)
	f.SetColWidth(sheet, "D", "D", 22)
	f.SetColWidth(sheet, "E", "E", 18)
	f.SetColWidth(sheet, "F", "F", 12)
	f.SetColWidth(sheet, "G", "G", 14)
	f.SetColWidth(sheet, "H", "H", 16)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-bagi-hasil-%s.xlsx", bagiHasilID)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/penjualan/:penjualan_id
func (lc *LaporanController) DownloadLaporanPenjualan(c *gin.Context) {
	penjualanID := c.Param("penjualan_id")

	// ── 1. Header penjualan ───────────────────────────────────────────────────
	type headerData struct {
		PenjualanID      string    `gorm:"column:penjualan_id"`
		NamaBank         string    `gorm:"column:nama_bank"`
		CreatedAt        time.Time `gorm:"column:created_at"`
		IdentitasPembeli string    `gorm:"column:identitas_pembeli"`
		NamaAdmin        string    `gorm:"column:nama_admin"`
		NamaReward       string    `gorm:"column:nama_reward"`
		TotalItem        int       `gorm:"column:total_item"`
		TotalPenjualan   float64   `gorm:"column:total_penjualan"`
		SatuanReward     string    `gorm:"column:satuan_reward"`
		StatusBagiHasil  string    `gorm:"column:status_bagi_hasil"`
	}

	var hdr headerData
	if err := lc.DB.Raw(`
		SELECT
			p.penjualan_id,
			bs.nama_bank,
			p.created_at,
			p.identitas_pembeli,
			COALESCE(u.nama, '-')  AS nama_admin,
			r.nama_reward,
			p.total_item,
			p.total_penjualan,
			p.satuan_reward,
			p.status_bagi_hasil
		FROM penjualan p
		JOIN bank_sampah bs  ON bs.bank_id   = p.bank_id
		LEFT JOIN admin adm  ON adm.admin_id  = p.sold_by
		LEFT JOIN users u    ON u.user_id     = adm.user_id
		JOIN reward r        ON r.reward_id   = p.reward_id
		WHERE p.penjualan_id = ?
	`, penjualanID).Scan(&hdr).Error; err != nil || hdr.PenjualanID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Penjualan tidak ditemukan"})
		return
	}

	// ── 2. Detail item sampah ─────────────────────────────────────────────────
	type detailRow struct {
		NamaSampah           string  `gorm:"column:nama_sampah"`
		Satuan               string  `gorm:"column:satuan"`
		Qty                  float64 `gorm:"column:qty"`
		HargaJual            float64 `gorm:"column:harga_jual"`
		SubtotalPenjualan    float64 `gorm:"column:subtotal_penjualan"`
		HargaNasabahSnapshot float64 `gorm:"column:harga_nasabah_snapshot"`
	}

	var details []detailRow
	if err := lc.DB.Raw(`
		SELECT
			s.nama_sampah,
			s.satuan,
			dp.qty,
			dp.harga_jual,
			dp.subtotal_penjualan,
			dp.harga_nasabah_snapshot
		FROM detail_penjualan dp
		JOIN katalog_sampah ks ON ks.sampah_id = dp.sampah_id
		JOIN sampah s ON s.sarok_id = ks.sarok_id
		WHERE dp.penjualan_id = ?
		ORDER BY s.nama_sampah ASC
	`, penjualanID).Scan(&details).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail penjualan: " + err.Error()})
		return
	}

	// ── 3. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Laporan Penjualan"
	f.SetSheetName("Sheet1", sheet)

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1565C0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})

	numFmt := "#,##0.00"
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNum, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNumAlt, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleTotalLabel, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E3F2FD"}},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 2},
			{Type: "bottom", Color: "CCCCCC", Style: 2},
		},
	})
	styleTotalNum, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		CustomNumFmt: &numFmt,
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E3F2FD"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 2},
			{Type: "bottom", Color: "CCCCCC", Style: 2},
		},
	})

	// ── Row 1: Title ──────────────────────────────────────────────────────────
	f.MergeCell(sheet, "A1", "G1")
	f.SetCellValue(sheet, "A1", "LAPORAN PENJUALAN SAMPAH")
	f.SetCellStyle(sheet, "A1", "G1", styleTitle)
	f.SetRowHeight(sheet, 1, 32)

	// ── Rows 3+: Info header ──────────────────────────────────────────────────
	// Layout: A:B = label (bold), C:G = value
	infoRows := [][2]string{
		{"Nama Bank Sampah", hdr.NamaBank},
		{"ID Penjualan", hdr.PenjualanID},
		{"Tanggal Penjualan", hdr.CreatedAt.Format("02 January 2006, 15:04:05")},
		{"Admin Penjual", hdr.NamaAdmin},
		{"Pembeli", hdr.IdentitasPembeli},
		{"Jenis Reward", hdr.NamaReward},
		{"Total Item", fmt.Sprintf("%d jenis sampah", hdr.TotalItem)},
		{"Total Penjualan", fmt.Sprintf("%.2f %s", hdr.TotalPenjualan, hdr.SatuanReward)},
		{"Status Bagi Hasil", hdr.StatusBagiHasil},
	}

	const infoStartRow = 3
	for i, row := range infoRows {
		r := infoStartRow + i
		f.MergeCell(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r), styleBold)
		f.MergeCell(sheet, fmt.Sprintf("C%d", r), fmt.Sprintf("G%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row[1])
	}

	// ── Table header ──────────────────────────────────────────────────────────
	tableHeaderRow := infoStartRow + len(infoRows) + 1
	headers := []string{"No", "Nama Sampah", "Satuan", "Qty", "Harga Jual/Satuan", "Subtotal Penjualan", "Harga Snapshot Nasabah/Satuan"}
	cols := []string{"A", "B", "C", "D", "E", "F", "G"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], tableHeaderRow)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheet, tableHeaderRow, 30)

	// ── Data rows ─────────────────────────────────────────────────────────────
	rowIdx := tableHeaderRow + 1
	for i, d := range details {
		no := i + 1

		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), no)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), d.NamaSampah)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), d.Qty)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), d.HargaJual)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bnSt)

		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), d.SubtotalPenjualan)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bnSt)

		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), d.HargaNasabahSnapshot)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), bnSt)

		rowIdx++
	}

	// ── Total row ─────────────────────────────────────────────────────────────
	totalRow := rowIdx + 1
	f.MergeCell(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("E%d", totalRow))
	f.SetCellValue(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("TOTAL PENJUALAN (%s)", hdr.SatuanReward))
	f.SetCellStyle(sheet, fmt.Sprintf("A%d", totalRow), fmt.Sprintf("E%d", totalRow), styleTotalLabel)
	f.SetCellValue(sheet, fmt.Sprintf("F%d", totalRow), hdr.TotalPenjualan)
	f.SetCellStyle(sheet, fmt.Sprintf("F%d", totalRow), fmt.Sprintf("F%d", totalRow), styleTotalNum)
	f.SetCellStyle(sheet, fmt.Sprintf("G%d", totalRow), fmt.Sprintf("G%d", totalRow), styleTotalLabel)
	f.SetRowHeight(sheet, totalRow, 22)

	// ── Column widths ─────────────────────────────────────────────────────────
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 28)
	f.SetColWidth(sheet, "C", "C", 10)
	f.SetColWidth(sheet, "D", "D", 10)
	f.SetColWidth(sheet, "E", "E", 22)
	f.SetColWidth(sheet, "F", "F", 22)
	f.SetColWidth(sheet, "G", "G", 30)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-penjualan-%s.xlsx", penjualanID)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/penjualan/rekap/:bank_id?start=YYYY-MM-DD&end=YYYY-MM-DD
func (lc *LaporanController) DownloadLaporanRekapPenjualan(c *gin.Context) {
	bankID := c.Param("bank_id")

	startStr := c.Query("start")
	endStr := c.Query("end")
	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter start dan end wajib diisi (format YYYY-MM-DD)"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format start tidak valid, gunakan YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format end tidak valid, gunakan YYYY-MM-DD"})
		return
	}
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tanggal end tidak boleh sebelum start"})
		return
	}
	endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// ── 1. Info bank sampah ───────────────────────────────────────────────────
	type bankData struct {
		BankID   string `gorm:"column:bank_id"`
		NamaBank string `gorm:"column:nama_bank"`
	}
	var bank bankData
	if err := lc.DB.Raw(`SELECT bank_id, nama_bank FROM bank_sampah WHERE bank_id = ?`, bankID).Scan(&bank).Error; err != nil || bank.BankID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	// ── 2. Rekap per sampah (di-group termasuk harga_jual, karena harga bisa
	//      berbeda antar transaksi penjualan dalam periode yang sama) ──────────
	// NB: satuan_reward (Rp/poin) ikut di-group karena harga_jual & subtotal_penjualan
	// didenominasi dalam satuan itu — transaksi Rp dan poin gak boleh ketuker/kegabung.
	type sampahRow struct {
		NamaSampah   string  `gorm:"column:nama_sampah"`
		Kategori     string  `gorm:"column:kategori"`
		Qty          float64 `gorm:"column:qty"`
		Satuan       string  `gorm:"column:satuan"`
		SatuanReward string  `gorm:"column:satuan_reward"`
		HargaJual    float64 `gorm:"column:harga_jual"`
		Subtotal     float64 `gorm:"column:subtotal"`
	}
	var sampahRows []sampahRow
	if err := lc.DB.Raw(`
		SELECT
			s.nama_sampah,
			kat.kategori,
			SUM(dp.qty)                 AS qty,
			s.satuan,
			p.satuan_reward,
			dp.harga_jual,
			SUM(dp.subtotal_penjualan)  AS subtotal
		FROM detail_penjualan dp
		JOIN penjualan p         ON p.penjualan_id  = dp.penjualan_id
		JOIN katalog_sampah ks   ON ks.sampah_id     = dp.sampah_id
		JOIN sampah s            ON s.sarok_id       = ks.sarok_id
		JOIN kategori_sampah kat ON kat.kategori_id  = ks.kategori_id
		WHERE p.bank_id = ? AND p.created_at BETWEEN ? AND ?
		GROUP BY s.sarok_id, s.nama_sampah, kat.kategori, s.satuan, p.satuan_reward, dp.harga_jual
		ORDER BY p.satuan_reward ASC, s.nama_sampah ASC, dp.harga_jual ASC
	`, bankID, startDate, endDate).Scan(&sampahRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil rekap sampah: " + err.Error()})
		return
	}

	// ── 3. Detail per transaksi penjualan ─────────────────────────────────────
	type transaksiRow struct {
		PenjualanID      string    `gorm:"column:penjualan_id"`
		CreatedAt        time.Time `gorm:"column:created_at"`
		IdentitasPembeli string    `gorm:"column:identitas_pembeli"`
		SatuanReward     string    `gorm:"column:satuan_reward"`
		NamaSampah       string    `gorm:"column:nama_sampah"`
		Qty              float64   `gorm:"column:qty"`
		Satuan           string    `gorm:"column:satuan"`
		HargaJual        float64   `gorm:"column:harga_jual"`
		Subtotal         float64   `gorm:"column:subtotal_penjualan"`
	}
	var transaksiRows []transaksiRow
	if err := lc.DB.Raw(`
		SELECT
			p.penjualan_id,
			p.created_at,
			p.identitas_pembeli,
			p.satuan_reward,
			s.nama_sampah,
			dp.qty,
			s.satuan,
			dp.harga_jual,
			dp.subtotal_penjualan
		FROM detail_penjualan dp
		JOIN penjualan p       ON p.penjualan_id = dp.penjualan_id
		JOIN katalog_sampah ks ON ks.sampah_id    = dp.sampah_id
		JOIN sampah s          ON s.sarok_id      = ks.sarok_id
		WHERE p.bank_id = ? AND p.created_at BETWEEN ? AND ?
		ORDER BY p.created_at ASC, s.nama_sampah ASC
	`, bankID, startDate, endDate).Scan(&transaksiRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail transaksi: " + err.Error()})
		return
	}

	// ── 4. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	periodeLabel := fmt.Sprintf("%s - %s", startDate.Format("02 January 2006"), endDate.Format("02 January 2006"))
	numFmt := "#,##0.00"

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1565C0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNum, _ := f.NewStyle(&excelize.Style{
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNumAlt, _ := f.NewStyle(&excelize.Style{
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Sheet 1: Rekap Sampah ─────────────────────────────────────────────────
	sheetSampah := "Rekap Sampah"
	f.SetSheetName("Sheet1", sheetSampah)

	f.MergeCell(sheetSampah, "A1", "G1")
	f.SetCellValue(sheetSampah, "A1", "REKAP PENJUALAN SAMPAH")
	f.SetCellStyle(sheetSampah, "A1", "G1", styleTitle)
	f.SetRowHeight(sheetSampah, 1, 32)

	sampahInfoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Periode", periodeLabel},
	}
	const sampahInfoStartRow = 3
	for i, row := range sampahInfoRows {
		r := sampahInfoStartRow + i
		f.SetCellValue(sheetSampah, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheetSampah, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), styleBold)
		f.MergeCell(sheetSampah, fmt.Sprintf("B%d", r), fmt.Sprintf("G%d", r))
		f.SetCellValue(sheetSampah, fmt.Sprintf("B%d", r), row[1])
	}

	sampahTableHeaderRow := sampahInfoStartRow + len(sampahInfoRows) + 1
	sampahHeaders := []string{"Nama Sampah", "Kategori", "Qty", "Satuan", "Satuan Reward", "Harga Jual / Unit", "Subtotal"}
	sampahCols := []string{"A", "B", "C", "D", "E", "F", "G"}
	for i, h := range sampahHeaders {
		cell := fmt.Sprintf("%s%d", sampahCols[i], sampahTableHeaderRow)
		f.SetCellValue(sheetSampah, cell, h)
		f.SetCellStyle(sheetSampah, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheetSampah, sampahTableHeaderRow, 25)

	styleTotalLabel, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E3F2FD"}},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleTotalNum, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E3F2FD"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// Data sudah terurut per satuan_reward, jadi tiap kelompok kontigu — begitu
	// satuan_reward ganti (atau data habis), tutup kelompok dgn baris TOTAL sendiri
	// supaya Rp dan poin gak pernah kejumlah jadi satu angka.
	writeSampahTotalRow := func(rowIdx int, satuanReward string, subtotal float64) {
		f.MergeCell(sheetSampah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("F%d", rowIdx))
		f.SetCellValue(sheetSampah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("TOTAL (%s)", satuanReward))
		f.SetCellStyle(sheetSampah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("F%d", rowIdx), styleTotalLabel)
		f.SetCellValue(sheetSampah, fmt.Sprintf("G%d", rowIdx), subtotal)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), styleTotalNum)
	}

	rowIdx := sampahTableHeaderRow + 1
	var groupSubtotal float64
	lastSatuanReward := ""
	for i, d := range sampahRows {
		if i > 0 && d.SatuanReward != lastSatuanReward {
			writeSampahTotalRow(rowIdx, lastSatuanReward, groupSubtotal)
			rowIdx++
			groupSubtotal = 0
		}
		lastSatuanReward = d.SatuanReward

		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if (i+1)%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		f.SetCellValue(sheetSampah, fmt.Sprintf("A%d", rowIdx), d.NamaSampah)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("B%d", rowIdx), d.Kategori)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("C%d", rowIdx), d.Qty)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bcSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("D%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("E%d", rowIdx), d.SatuanReward)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bcSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("F%d", rowIdx), d.HargaJual)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bnSt)

		f.SetCellValue(sheetSampah, fmt.Sprintf("G%d", rowIdx), d.Subtotal)
		f.SetCellStyle(sheetSampah, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), bnSt)

		groupSubtotal += d.Subtotal
		rowIdx++
	}
	if len(sampahRows) > 0 {
		writeSampahTotalRow(rowIdx, lastSatuanReward, groupSubtotal)
		rowIdx++
	}

	f.SetColWidth(sheetSampah, "A", "A", 28)
	f.SetColWidth(sheetSampah, "B", "B", 22)
	f.SetColWidth(sheetSampah, "C", "C", 12)
	f.SetColWidth(sheetSampah, "D", "D", 12)
	f.SetColWidth(sheetSampah, "E", "E", 15)
	f.SetColWidth(sheetSampah, "F", "F", 18)
	f.SetColWidth(sheetSampah, "G", "G", 18)

	// ── Sheet 2: Detail Transaksi ─────────────────────────────────────────────
	sheetTransaksi := "Detail Transaksi"
	f.NewSheet(sheetTransaksi)

	f.MergeCell(sheetTransaksi, "A1", "I1")
	f.SetCellValue(sheetTransaksi, "A1", "DETAIL TRANSAKSI PENJUALAN")
	f.SetCellStyle(sheetTransaksi, "A1", "I1", styleTitle)
	f.SetRowHeight(sheetTransaksi, 1, 32)

	transaksiInfoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Periode", periodeLabel},
	}
	const transaksiInfoStartRow = 3
	for i, row := range transaksiInfoRows {
		r := transaksiInfoStartRow + i
		f.SetCellValue(sheetTransaksi, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), styleBold)
		f.MergeCell(sheetTransaksi, fmt.Sprintf("B%d", r), fmt.Sprintf("I%d", r))
		f.SetCellValue(sheetTransaksi, fmt.Sprintf("B%d", r), row[1])
	}

	transaksiTableHeaderRow := transaksiInfoStartRow + len(transaksiInfoRows) + 1
	transaksiHeaders := []string{"No", "Tanggal", "Pembeli", "Satuan Reward", "Nama Sampah", "Qty", "Satuan", "Harga Jual / Unit", "Subtotal"}
	transaksiCols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	for i, h := range transaksiHeaders {
		cell := fmt.Sprintf("%s%d", transaksiCols[i], transaksiTableHeaderRow)
		f.SetCellValue(sheetTransaksi, cell, h)
		f.SetCellStyle(sheetTransaksi, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheetTransaksi, transaksiTableHeaderRow, 25)

	rowIdx = transaksiTableHeaderRow + 1
	transaksiNo := 0
	lastPenjualanID := ""
	for _, d := range transaksiRows {
		isFirst := d.PenjualanID != lastPenjualanID
		if isFirst {
			transaksiNo++
			lastPenjualanID = d.PenjualanID
		}

		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if transaksiNo%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		if isFirst {
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("A%d", rowIdx), transaksiNo)
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("B%d", rowIdx), d.CreatedAt.Format("02/01/2006 15:04"))
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("C%d", rowIdx), d.IdentitasPembeli)
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("D%d", rowIdx), d.SatuanReward)
		}
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("E%d", rowIdx), d.NamaSampah)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("F%d", rowIdx), d.Qty)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bcSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("G%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), bcSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("H%d", rowIdx), d.HargaJual)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), bnSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("I%d", rowIdx), d.Subtotal)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), bnSt)

		rowIdx++
	}

	f.SetColWidth(sheetTransaksi, "A", "A", 6)
	f.SetColWidth(sheetTransaksi, "B", "B", 18)
	f.SetColWidth(sheetTransaksi, "C", "C", 24)
	f.SetColWidth(sheetTransaksi, "D", "D", 14)
	f.SetColWidth(sheetTransaksi, "E", "E", 26)
	f.SetColWidth(sheetTransaksi, "F", "F", 10)
	f.SetColWidth(sheetTransaksi, "G", "G", 10)
	f.SetColWidth(sheetTransaksi, "H", "H", 18)
	f.SetColWidth(sheetTransaksi, "I", "I", 18)

	f.SetActiveSheet(0)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-rekap-penjualan-%s-%s-%s.xlsx", bankID, startStr, endStr)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/bagi-hasil/rekap/:bank_id?start=YYYY-MM-DD&end=YYYY-MM-DD
func (lc *LaporanController) DownloadLaporanRekapBagiHasil(c *gin.Context) {
	bankID := c.Param("bank_id")

	startStr := c.Query("start")
	endStr := c.Query("end")
	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter start dan end wajib diisi (format YYYY-MM-DD)"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format start tidak valid, gunakan YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format end tidak valid, gunakan YYYY-MM-DD"})
		return
	}
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tanggal end tidak boleh sebelum start"})
		return
	}
	endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// ── 1. Info bank sampah ───────────────────────────────────────────────────
	type bankData struct {
		BankID   string `gorm:"column:bank_id"`
		NamaBank string `gorm:"column:nama_bank"`
	}
	var bank bankData
	if err := lc.DB.Raw(`SELECT bank_id, nama_bank FROM bank_sampah WHERE bank_id = ?`, bankID).Scan(&bank).Error; err != nil || bank.BankID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	// ── 2. Rekap per nasabah (di-group termasuk satuan, karena bisa Rp atau
	//      poin tergantung reward penjualan yang dipakai di tiap event bagi hasil) ─
	type nasabahRow struct {
		NasabahID     string  `gorm:"column:nasabah_id"`
		NamaNasabah   string  `gorm:"column:nama_nasabah"`
		Satuan        string  `gorm:"column:satuan"`
		TotalDiterima float64 `gorm:"column:total_diterima"`
	}
	var nasabahRows []nasabahRow
	if err := lc.DB.Raw(`
		SELECT
			n.nasabah_id,
			u.nama                    AS nama_nasabah,
			pbh.satuan_diterima       AS satuan,
			SUM(pbh.total_diterima)   AS total_diterima
		FROM penerima_bagi_hasil pbh
		JOIN bagi_hasil bh ON bh.bagi_hasil_id = pbh.bagi_hasil_id
		JOIN nasabah n     ON n.nasabah_id      = pbh.nasabah_id
		JOIN users u       ON u.user_id         = n.user_id
		WHERE bh.bank_id = ? AND bh.created_at BETWEEN ? AND ?
		GROUP BY n.nasabah_id, u.nama, pbh.satuan_diterima
		ORDER BY pbh.satuan_diterima ASC, u.nama ASC
	`, bankID, startDate, endDate).Scan(&nasabahRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil rekap nasabah: " + err.Error()})
		return
	}

	// ── 3. Detail per transaksi (event) bagi hasil ────────────────────────────
	type transaksiRow struct {
		BagiHasilID     string    `gorm:"column:bagi_hasil_id"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		PenjualanID     string    `gorm:"column:penjualan_id"`
		GrossBank       float64   `gorm:"column:gross_bank"`
		SisaBagiHasil   float64   `gorm:"column:sisa_bagi_hasil"`
		Satuan          string    `gorm:"column:satuan"`
		NasabahID       string    `gorm:"column:nasabah_id"`
		NamaNasabah     string    `gorm:"column:nama_nasabah"`
		NominalDiterima float64   `gorm:"column:nominal_diterima"`
	}
	var transaksiRows []transaksiRow
	if err := lc.DB.Raw(`
		SELECT
			bh.bagi_hasil_id,
			bh.created_at,
			COALESCE(bh.penjualan_id, '-') AS penjualan_id,
			bh.gross_bank,
			bh.sisa_bagi_hasil,
			bh.satuan_bagi_hasil           AS satuan,
			pbh.nasabah_id,
			u.nama                         AS nama_nasabah,
			pbh.total_diterima             AS nominal_diterima
		FROM penerima_bagi_hasil pbh
		JOIN bagi_hasil bh ON bh.bagi_hasil_id = pbh.bagi_hasil_id
		JOIN nasabah n     ON n.nasabah_id      = pbh.nasabah_id
		JOIN users u       ON u.user_id         = n.user_id
		WHERE bh.bank_id = ? AND bh.created_at BETWEEN ? AND ?
		ORDER BY bh.created_at ASC, u.nama ASC
	`, bankID, startDate, endDate).Scan(&transaksiRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail transaksi: " + err.Error()})
		return
	}

	// ── 4. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	periodeLabel := fmt.Sprintf("%s - %s", startDate.Format("02 January 2006"), endDate.Format("02 January 2006"))
	numFmt := "#,##0.00"

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"4527A0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNum, _ := f.NewStyle(&excelize.Style{
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNumAlt, _ := f.NewStyle(&excelize.Style{
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleTotalLabel, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EDE7F6"}},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleTotalNum, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EDE7F6"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Sheet 1: Rekap Nasabah ────────────────────────────────────────────────
	sheetNasabah := "Rekap Nasabah"
	f.SetSheetName("Sheet1", sheetNasabah)

	f.MergeCell(sheetNasabah, "A1", "E1")
	f.SetCellValue(sheetNasabah, "A1", "REKAP BAGI HASIL NASABAH")
	f.SetCellStyle(sheetNasabah, "A1", "E1", styleTitle)
	f.SetRowHeight(sheetNasabah, 1, 32)

	nasabahInfoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Periode", periodeLabel},
	}
	const nasabahInfoStartRow = 3
	for i, row := range nasabahInfoRows {
		r := nasabahInfoStartRow + i
		f.SetCellValue(sheetNasabah, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), styleBold)
		f.MergeCell(sheetNasabah, fmt.Sprintf("B%d", r), fmt.Sprintf("E%d", r))
		f.SetCellValue(sheetNasabah, fmt.Sprintf("B%d", r), row[1])
	}

	nasabahTableHeaderRow := nasabahInfoStartRow + len(nasabahInfoRows) + 1
	nasabahHeaders := []string{"No", "Nasabah ID", "Nama Nasabah", "Satuan", "Total Diterima"}
	nasabahCols := []string{"A", "B", "C", "D", "E"}
	for i, h := range nasabahHeaders {
		cell := fmt.Sprintf("%s%d", nasabahCols[i], nasabahTableHeaderRow)
		f.SetCellValue(sheetNasabah, cell, h)
		f.SetCellStyle(sheetNasabah, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheetNasabah, nasabahTableHeaderRow, 25)

	// Data sudah terurut per satuan, jadi tiap kelompok kontigu — begitu satuan
	// ganti (atau data habis), tutup kelompok dgn baris TOTAL sendiri supaya Rp
	// dan poin gak pernah kejumlah jadi satu angka.
	writeNasabahTotalRow := func(rowIdx int, satuan string, total float64) {
		f.MergeCell(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("D%d", rowIdx))
		f.SetCellValue(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("TOTAL (%s)", satuan))
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("D%d", rowIdx), styleTotalLabel)
		f.SetCellValue(sheetNasabah, fmt.Sprintf("E%d", rowIdx), total)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), styleTotalNum)
	}

	rowIdx := nasabahTableHeaderRow + 1
	var groupTotal float64
	lastSatuan := ""
	for i, d := range nasabahRows {
		if i > 0 && d.Satuan != lastSatuan {
			writeNasabahTotalRow(rowIdx, lastSatuan, groupTotal)
			rowIdx++
			groupTotal = 0
		}
		lastSatuan = d.Satuan

		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if (i+1)%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		f.SetCellValue(sheetNasabah, fmt.Sprintf("A%d", rowIdx), i+1)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("B%d", rowIdx), d.NasabahID)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("C%d", rowIdx), d.NamaNasabah)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("D%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("E%d", rowIdx), d.TotalDiterima)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bnSt)

		groupTotal += d.TotalDiterima
		rowIdx++
	}
	if len(nasabahRows) > 0 {
		writeNasabahTotalRow(rowIdx, lastSatuan, groupTotal)
		rowIdx++
	}

	f.SetColWidth(sheetNasabah, "A", "A", 6)
	f.SetColWidth(sheetNasabah, "B", "B", 24)
	f.SetColWidth(sheetNasabah, "C", "C", 26)
	f.SetColWidth(sheetNasabah, "D", "D", 12)
	f.SetColWidth(sheetNasabah, "E", "E", 18)

	// ── Sheet 2: Detail Transaksi Bagi Hasil ──────────────────────────────────
	sheetTransaksi := "Detail Transaksi"
	f.NewSheet(sheetTransaksi)

	f.MergeCell(sheetTransaksi, "A1", "I1")
	f.SetCellValue(sheetTransaksi, "A1", "DETAIL TRANSAKSI BAGI HASIL")
	f.SetCellStyle(sheetTransaksi, "A1", "I1", styleTitle)
	f.SetRowHeight(sheetTransaksi, 1, 32)

	transaksiInfoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Periode", periodeLabel},
	}
	const transaksiInfoStartRow = 3
	for i, row := range transaksiInfoRows {
		r := transaksiInfoStartRow + i
		f.SetCellValue(sheetTransaksi, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), styleBold)
		f.MergeCell(sheetTransaksi, fmt.Sprintf("B%d", r), fmt.Sprintf("I%d", r))
		f.SetCellValue(sheetTransaksi, fmt.Sprintf("B%d", r), row[1])
	}

	transaksiTableHeaderRow := transaksiInfoStartRow + len(transaksiInfoRows) + 1
	transaksiHeaders := []string{"No", "Tanggal", "Penjualan Terkait", "Gross Bank", "Sisa Bank", "Satuan", "Nasabah ID", "Nama Nasabah", "Nominal Diterima"}
	transaksiCols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	for i, h := range transaksiHeaders {
		cell := fmt.Sprintf("%s%d", transaksiCols[i], transaksiTableHeaderRow)
		f.SetCellValue(sheetTransaksi, cell, h)
		f.SetCellStyle(sheetTransaksi, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheetTransaksi, transaksiTableHeaderRow, 25)

	rowIdx = transaksiTableHeaderRow + 1
	transaksiNo := 0
	lastBagiHasilID := ""
	for _, d := range transaksiRows {
		isFirst := d.BagiHasilID != lastBagiHasilID
		if isFirst {
			transaksiNo++
			lastBagiHasilID = d.BagiHasilID
		}

		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if transaksiNo%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		if isFirst {
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("A%d", rowIdx), transaksiNo)
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("B%d", rowIdx), d.CreatedAt.Format("02/01/2006 15:04"))
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("C%d", rowIdx), d.PenjualanID)
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("D%d", rowIdx), d.GrossBank)
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("E%d", rowIdx), d.SisaBagiHasil)
			f.SetCellValue(sheetTransaksi, fmt.Sprintf("F%d", rowIdx), d.Satuan)
		}
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bnSt)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bnSt)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bcSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("G%d", rowIdx), d.NasabahID)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), bSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("H%d", rowIdx), d.NamaNasabah)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), bSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("I%d", rowIdx), d.NominalDiterima)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), bnSt)

		rowIdx++
	}

	f.SetColWidth(sheetTransaksi, "A", "A", 6)
	f.SetColWidth(sheetTransaksi, "B", "B", 18)
	f.SetColWidth(sheetTransaksi, "C", "C", 20)
	f.SetColWidth(sheetTransaksi, "D", "D", 16)
	f.SetColWidth(sheetTransaksi, "E", "E", 16)
	f.SetColWidth(sheetTransaksi, "F", "F", 12)
	f.SetColWidth(sheetTransaksi, "G", "G", 24)
	f.SetColWidth(sheetTransaksi, "H", "H", 26)
	f.SetColWidth(sheetTransaksi, "I", "I", 18)

	f.SetActiveSheet(0)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-rekap-bagi-hasil-%s-%s-%s.xlsx", bankID, startStr, endStr)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/penarikan/rekap/:bank_id?start=YYYY-MM-DD&end=YYYY-MM-DD
func (lc *LaporanController) DownloadLaporanRekapPenarikan(c *gin.Context) {
	bankID := c.Param("bank_id")

	startStr := c.Query("start")
	endStr := c.Query("end")
	if startStr == "" || endStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Parameter start dan end wajib diisi (format YYYY-MM-DD)"})
		return
	}

	startDate, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format start tidak valid, gunakan YYYY-MM-DD"})
		return
	}
	endDate, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Format end tidak valid, gunakan YYYY-MM-DD"})
		return
	}
	if endDate.Before(startDate) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tanggal end tidak boleh sebelum start"})
		return
	}
	endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)

	// ── 1. Info bank sampah ───────────────────────────────────────────────────
	type bankData struct {
		BankID   string `gorm:"column:bank_id"`
		NamaBank string `gorm:"column:nama_bank"`
	}
	var bank bankData
	if err := lc.DB.Raw(`SELECT bank_id, nama_bank FROM bank_sampah WHERE bank_id = ?`, bankID).Scan(&bank).Error; err != nil || bank.BankID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	// ── 2. Rekap per nasabah — cuma status 'berhasil' yang dihitung, karena cuma
	//      itu yang beneran ngurangin saldo nasabah. Di-group termasuk satuan
	//      (Rp/poin) biar gak ketuker/kegabung. ─────────────────────────────────
	type nasabahRow struct {
		NasabahID    string  `gorm:"column:nasabah_id"`
		NamaNasabah  string  `gorm:"column:nama_nasabah"`
		Satuan       string  `gorm:"column:satuan"`
		TotalDitarik float64 `gorm:"column:total_ditarik"`
	}
	var nasabahRows []nasabahRow
	if err := lc.DB.Raw(`
		SELECT
			n.nasabah_id,
			u.nama                       AS nama_nasabah,
			p.satuan_penarikan           AS satuan,
			SUM(p.nominal_penarikan)     AS total_ditarik
		FROM penarikan p
		JOIN nasabah n ON n.nasabah_id = p.nasabah_id
		JOIN users u   ON u.user_id    = n.user_id
		WHERE p.bank_id = ? AND p.status_penarikan = 'berhasil' AND p.created_at BETWEEN ? AND ?
		GROUP BY n.nasabah_id, u.nama, p.satuan_penarikan
		ORDER BY p.satuan_penarikan ASC, u.nama ASC
	`, bankID, startDate, endDate).Scan(&nasabahRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil rekap nasabah: " + err.Error()})
		return
	}

	// ── 3. Detail per transaksi penarikan — semua status, buat audit trail ────
	type transaksiRow struct {
		CreatedAt   time.Time `gorm:"column:created_at"`
		NasabahID   string    `gorm:"column:nasabah_id"`
		NamaNasabah string    `gorm:"column:nama_nasabah"`
		Nominal     float64   `gorm:"column:nominal_penarikan"`
		Satuan      string    `gorm:"column:satuan"`
		Status      string    `gorm:"column:status"`
	}
	var transaksiRows []transaksiRow
	if err := lc.DB.Raw(`
		SELECT
			p.created_at,
			n.nasabah_id,
			u.nama                 AS nama_nasabah,
			p.nominal_penarikan,
			p.satuan_penarikan     AS satuan,
			p.status_penarikan     AS status
		FROM penarikan p
		JOIN nasabah n ON n.nasabah_id = p.nasabah_id
		JOIN users u   ON u.user_id    = n.user_id
		WHERE p.bank_id = ? AND p.created_at BETWEEN ? AND ?
		ORDER BY p.created_at ASC
	`, bankID, startDate, endDate).Scan(&transaksiRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil detail transaksi: " + err.Error()})
		return
	}

	// ── 4. Build Excel ────────────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()

	periodeLabel := fmt.Sprintf("%s - %s", startDate.Format("02 January 2006"), endDate.Format("02 January 2006"))
	numFmt := "#,##0.00"

	// ── Styles ────────────────────────────────────────────────────────────────
	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"00695C"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNum, _ := f.NewStyle(&excelize.Style{
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNumAlt, _ := f.NewStyle(&excelize.Style{
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleTotalLabel, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E0F2F1"}},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleTotalNum, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true},
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E0F2F1"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &numFmt,
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleStatusBerhasil, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "1B5E20"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8F5E9"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleStatusPending, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "E65100"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFF3E0"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleStatusGagal, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "B71C1C"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFEBEE"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Sheet 1: Rekap Nasabah ────────────────────────────────────────────────
	sheetNasabah := "Rekap Nasabah"
	f.SetSheetName("Sheet1", sheetNasabah)

	f.MergeCell(sheetNasabah, "A1", "E1")
	f.SetCellValue(sheetNasabah, "A1", "REKAP PENARIKAN NASABAH")
	f.SetCellStyle(sheetNasabah, "A1", "E1", styleTitle)
	f.SetRowHeight(sheetNasabah, 1, 32)

	nasabahInfoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Periode", periodeLabel},
		{"Status Dihitung", "Berhasil"},
	}
	const nasabahInfoStartRow = 3
	for i, row := range nasabahInfoRows {
		r := nasabahInfoStartRow + i
		f.SetCellValue(sheetNasabah, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), styleBold)
		f.MergeCell(sheetNasabah, fmt.Sprintf("B%d", r), fmt.Sprintf("E%d", r))
		f.SetCellValue(sheetNasabah, fmt.Sprintf("B%d", r), row[1])
	}

	nasabahTableHeaderRow := nasabahInfoStartRow + len(nasabahInfoRows) + 1
	nasabahHeaders := []string{"No", "Nasabah ID", "Nama Nasabah", "Satuan", "Total Ditarik"}
	nasabahCols := []string{"A", "B", "C", "D", "E"}
	for i, h := range nasabahHeaders {
		cell := fmt.Sprintf("%s%d", nasabahCols[i], nasabahTableHeaderRow)
		f.SetCellValue(sheetNasabah, cell, h)
		f.SetCellStyle(sheetNasabah, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheetNasabah, nasabahTableHeaderRow, 25)

	// Data sudah terurut per satuan, jadi tiap kelompok kontigu — begitu satuan
	// ganti (atau data habis), tutup kelompok dgn baris TOTAL sendiri supaya Rp
	// dan poin gak pernah kejumlah jadi satu angka.
	writeNasabahTotalRow := func(rowIdx int, satuan string, total float64) {
		f.MergeCell(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("D%d", rowIdx))
		f.SetCellValue(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("TOTAL (%s)", satuan))
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("D%d", rowIdx), styleTotalLabel)
		f.SetCellValue(sheetNasabah, fmt.Sprintf("E%d", rowIdx), total)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), styleTotalNum)
	}

	rowIdx := nasabahTableHeaderRow + 1
	var groupTotal float64
	lastSatuan := ""
	for i, d := range nasabahRows {
		if i > 0 && d.Satuan != lastSatuan {
			writeNasabahTotalRow(rowIdx, lastSatuan, groupTotal)
			rowIdx++
			groupTotal = 0
		}
		lastSatuan = d.Satuan

		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if (i+1)%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		f.SetCellValue(sheetNasabah, fmt.Sprintf("A%d", rowIdx), i+1)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("B%d", rowIdx), d.NasabahID)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("C%d", rowIdx), d.NamaNasabah)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("D%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)

		f.SetCellValue(sheetNasabah, fmt.Sprintf("E%d", rowIdx), d.TotalDitarik)
		f.SetCellStyle(sheetNasabah, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bnSt)

		groupTotal += d.TotalDitarik
		rowIdx++
	}
	if len(nasabahRows) > 0 {
		writeNasabahTotalRow(rowIdx, lastSatuan, groupTotal)
		rowIdx++
	}

	f.SetColWidth(sheetNasabah, "A", "A", 6)
	f.SetColWidth(sheetNasabah, "B", "B", 24)
	f.SetColWidth(sheetNasabah, "C", "C", 26)
	f.SetColWidth(sheetNasabah, "D", "D", 12)
	f.SetColWidth(sheetNasabah, "E", "E", 18)

	// ── Sheet 2: Detail Transaksi ─────────────────────────────────────────────
	sheetTransaksi := "Detail Transaksi"
	f.NewSheet(sheetTransaksi)

	f.MergeCell(sheetTransaksi, "A1", "G1")
	f.SetCellValue(sheetTransaksi, "A1", "DETAIL TRANSAKSI PENARIKAN")
	f.SetCellStyle(sheetTransaksi, "A1", "G1", styleTitle)
	f.SetRowHeight(sheetTransaksi, 1, 32)

	transaksiInfoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Periode", periodeLabel},
	}
	const transaksiInfoStartRow = 3
	for i, row := range transaksiInfoRows {
		r := transaksiInfoStartRow + i
		f.SetCellValue(sheetTransaksi, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("A%d", r), fmt.Sprintf("A%d", r), styleBold)
		f.MergeCell(sheetTransaksi, fmt.Sprintf("B%d", r), fmt.Sprintf("G%d", r))
		f.SetCellValue(sheetTransaksi, fmt.Sprintf("B%d", r), row[1])
	}

	transaksiTableHeaderRow := transaksiInfoStartRow + len(transaksiInfoRows) + 1
	// Status ditaruh di kolom terakhir (G) supaya bisa disorot warnanya
	transaksiHeaders := []string{"No", "Tanggal", "Nasabah ID", "Nama Nasabah", "Nominal", "Satuan", "Status"}
	transaksiCols := []string{"A", "B", "C", "D", "E", "F", "G"}
	for i, h := range transaksiHeaders {
		cell := fmt.Sprintf("%s%d", transaksiCols[i], transaksiTableHeaderRow)
		f.SetCellValue(sheetTransaksi, cell, h)
		f.SetCellStyle(sheetTransaksi, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheetTransaksi, transaksiTableHeaderRow, 25)

	rowIdx = transaksiTableHeaderRow + 1
	for i, d := range transaksiRows {
		no := i + 1
		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("A%d", rowIdx), no)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("B%d", rowIdx), d.CreatedAt.Format("02/01/2006 15:04"))
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("C%d", rowIdx), d.NasabahID)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("D%d", rowIdx), d.NamaNasabah)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("E%d", rowIdx), d.Nominal)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bnSt)

		f.SetCellValue(sheetTransaksi, fmt.Sprintf("F%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bcSt)

		statusStyle := bcSt
		switch d.Status {
		case "berhasil":
			statusStyle = styleStatusBerhasil
		case "pending":
			statusStyle = styleStatusPending
		case "kadaluarsa", "dibatalkan":
			statusStyle = styleStatusGagal
		}
		f.SetCellValue(sheetTransaksi, fmt.Sprintf("G%d", rowIdx), d.Status)
		f.SetCellStyle(sheetTransaksi, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), statusStyle)

		rowIdx++
	}

	f.SetColWidth(sheetTransaksi, "A", "A", 6)
	f.SetColWidth(sheetTransaksi, "B", "B", 18)
	f.SetColWidth(sheetTransaksi, "C", "C", 24)
	f.SetColWidth(sheetTransaksi, "D", "D", 26)
	f.SetColWidth(sheetTransaksi, "E", "E", 16)
	f.SetColWidth(sheetTransaksi, "F", "F", 12)
	f.SetColWidth(sheetTransaksi, "G", "G", 14)

	f.SetActiveSheet(0)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-rekap-penarikan-%s-%s-%s.xlsx", bankID, startStr, endStr)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/bank-sampah
func (lc *LaporanController) DownloadLaporanBankSampah(c *gin.Context) {
	type bankRow struct {
		BankID        string    `gorm:"column:bank_id"`
		NamaBank      string    `gorm:"column:nama_bank"`
		JenisBank     string    `gorm:"column:jenis_bank"`
		Alamat        string    `gorm:"column:alamat"`
		Kecamatan     string    `gorm:"column:kecamatan"`
		Kelurahan     string    `gorm:"column:kelurahan"`
		IsActive      bool      `gorm:"column:is_active"`
		CreatedAt     time.Time `gorm:"column:created_at"`
		JumlahNasabah int       `gorm:"column:jumlah_nasabah"`
	}

	var rows []bankRow
	if err := lc.DB.Raw(`
		SELECT
			bs.bank_id,
			bs.nama_bank,
			bs.jenis_bank,
			bs.alamat,
			COALESCE(k.kecamatan, '-')  AS kecamatan,
			COALESCE(kl.kelurahan, '-') AS kelurahan,
			bs.is_active,
			bs.created_at,
			COUNT(n.nasabah_id) FILTER (WHERE n.status_nasabah = 'aktif') AS jumlah_nasabah
		FROM bank_sampah bs
		LEFT JOIN kecamatan k   ON k.id_kecamatan  = bs.id_kecamatan
		LEFT JOIN kelurahan kl  ON kl.id_kelurahan  = bs.id_kelurahan
		LEFT JOIN nasabah n     ON n.bank_id         = bs.bank_id
		GROUP BY bs.bank_id, bs.nama_bank, bs.jenis_bank, bs.alamat, k.kecamatan, kl.kelurahan, bs.is_active, bs.created_at
		ORDER BY bs.jenis_bank ASC, bs.nama_bank ASC
	`).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data bank sampah: " + err.Error()})
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Daftar Bank Sampah"
	f.SetSheetName("Sheet1", sheet)

	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"00695C"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleAktif, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "1B5E20"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E8F5E9"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleNonaktif, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "B71C1C"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"FFEBEE"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Row 1: Title ──────────────────────────────────────────────────────────
	f.MergeCell(sheet, "A1", "I1")
	f.SetCellValue(sheet, "A1", "DAFTAR BANK SAMPAH")
	f.SetCellStyle(sheet, "A1", "I1", styleTitle)
	f.SetRowHeight(sheet, 1, 32)

	// ── Rows 3+: Info header ──────────────────────────────────────────────────
	infoRows := [][2]string{
		{"Tanggal Cetak", time.Now().Format("02 January 2006, 15:04:05")},
		{"Total Bank", fmt.Sprintf("%d bank sampah", len(rows))},
	}
	const infoStartRow = 3
	for i, row := range infoRows {
		r := infoStartRow + i
		f.MergeCell(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r), styleBold)
		f.MergeCell(sheet, fmt.Sprintf("C%d", r), fmt.Sprintf("I%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row[1])
	}

	// ── Table header ──────────────────────────────────────────────────────────
	tableHeaderRow := infoStartRow + len(infoRows) + 1
	headers := []string{"No", "Bank ID", "Nama Bank", "Jenis Bank", "Kecamatan", "Kelurahan", "Alamat", "Nasabah Aktif", "Status"}
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], tableHeaderRow)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheet, tableHeaderRow, 25)

	// ── Data rows ─────────────────────────────────────────────────────────────
	rowIdx := tableHeaderRow + 1
	for i, d := range rows {
		no := i + 1
		bSt := styleBorder
		bcSt := styleBorderCenter
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), no)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), d.BankID)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), d.NamaBank)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), d.JenisBank)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)

		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), d.Kecamatan)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), d.Kelurahan)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), d.Alamat)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), bSt)

		f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), d.JumlahNasabah)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), bcSt)

		statusLabel := "Nonaktif"
		statusStyle := styleNonaktif
		if d.IsActive {
			statusLabel = "Aktif"
			statusStyle = styleAktif
		}
		f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), statusLabel)
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), statusStyle)

		rowIdx++
	}

	// ── Column widths ─────────────────────────────────────────────────────────
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 16)
	f.SetColWidth(sheet, "C", "C", 28)
	f.SetColWidth(sheet, "D", "D", 12)
	f.SetColWidth(sheet, "E", "E", 20)
	f.SetColWidth(sheet, "F", "F", 20)
	f.SetColWidth(sheet, "G", "G", 36)
	f.SetColWidth(sheet, "H", "H", 14)
	f.SetColWidth(sheet, "I", "I", 10)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("laporan-bank-sampah-%s.xlsx", time.Now().Format("20060102"))
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/katalog-sampah/:bank_id
func (lc *LaporanController) DownloadLaporanKatalogSampah(c *gin.Context) {
	bankID := c.Param("bank_id")

	type bankData struct {
		BankID       string  `gorm:"column:bank_id"`
		NamaBank     string  `gorm:"column:nama_bank"`
		JenisBank    string  `gorm:"column:jenis_bank"`
		ParentBankID *string `gorm:"column:parent_bank_id"`
	}
	var bank bankData
	if err := lc.DB.Raw(`SELECT bank_id, nama_bank, jenis_bank, parent_bank_id FROM bank_sampah WHERE bank_id = ?`, bankID).Scan(&bank).Error; err != nil || bank.BankID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	// BSU menggunakan katalog milik parent BSI, difilter berdasarkan stok yang dimiliki BSU
	catalogBankID := bankID
	if bank.JenisBank == "bsu" && bank.ParentBankID != nil {
		catalogBankID = *bank.ParentBankID
	}

	type katalogRow struct {
		NamaSampah      string    `gorm:"column:nama_sampah"`
		Kategori        string    `gorm:"column:kategori"`
		Satuan          string    `gorm:"column:satuan"`
		JenisReward     string    `gorm:"column:jenis_reward"`
		HargaNasabah    float64   `gorm:"column:harga_nasabah"`
		HargaEksternal  float64   `gorm:"column:harga_eksternal"`
		SatuanReward    string    `gorm:"column:satuan_reward"`
		SyaratPemilahan string    `gorm:"column:syarat_pemilahan"`
		CreatedAt       time.Time `gorm:"column:created_at"`
	}

	var rows []katalogRow
	var queryArgs []interface{}
	stokFilter := ""
	if bank.JenisBank == "bsu" {
		stokFilter = "AND EXISTS (SELECT 1 FROM stok_sampah ss WHERE ss.sampah_id = ks.sampah_id AND ss.bank_id = ?)"
		queryArgs = append(queryArgs, catalogBankID, bankID)
	} else {
		queryArgs = append(queryArgs, catalogBankID)
	}
	if err := lc.DB.Raw(fmt.Sprintf(`
		SELECT
			s.nama_sampah,
			k.kategori,
			s.satuan,
			r.nama_reward                                                                        AS jenis_reward,
			COALESCE(MAX(sh.harga) FILTER (WHERE sh.level_user = 'nasabah'),   0)               AS harga_nasabah,
			COALESCE(MAX(sh.harga) FILTER (WHERE sh.level_user = 'eksternal'), 0)               AS harga_eksternal,
			COALESCE(MAX(sh.satuan_reward::text) FILTER (WHERE sh.level_user = 'nasabah'), '-') AS satuan_reward,
			ks.syarat_pemilahan,
			ks.created_at
		FROM katalog_sampah ks
		JOIN sampah s           ON s.sarok_id    = ks.sarok_id
		JOIN kategori_sampah k  ON k.kategori_id = ks.kategori_id
		JOIN reward r           ON r.reward_id   = ks.reward_id
		LEFT JOIN schema_harga_sampah sh ON sh.sampah_id = ks.sampah_id
		WHERE ks.bank_id = ? %s
		GROUP BY s.nama_sampah, k.kategori, s.satuan, r.nama_reward, ks.syarat_pemilahan, ks.created_at
		ORDER BY k.kategori ASC, s.nama_sampah ASC
	`, stokFilter), queryArgs...).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data katalog sampah: " + err.Error()})
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Katalog Sampah"
	f.SetSheetName("Sheet1", sheet)

	numFmt := "#,##0.00"

	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"33691E"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNum, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNumAlt, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Row 1: Title ──────────────────────────────────────────────────────────
	f.MergeCell(sheet, "A1", "J1")
	f.SetCellValue(sheet, "A1", "KATALOG SAMPAH BANK SAMPAH")
	f.SetCellStyle(sheet, "A1", "J1", styleTitle)
	f.SetRowHeight(sheet, 1, 32)

	// ── Rows 3+: Info header ──────────────────────────────────────────────────
	infoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Bank ID", bank.BankID},
		{"Tanggal Cetak", time.Now().Format("02 January 2006, 15:04:05")},
		{"Total Item", fmt.Sprintf("%d jenis sampah", len(rows))},
	}
	const infoStartRow = 3
	for i, row := range infoRows {
		r := infoStartRow + i
		f.MergeCell(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r), styleBold)
		f.MergeCell(sheet, fmt.Sprintf("C%d", r), fmt.Sprintf("J%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row[1])
	}

	// ── Table header ──────────────────────────────────────────────────────────
	tableHeaderRow := infoStartRow + len(infoRows) + 1
	headers := []string{"No", "Nama Sampah", "Kategori", "Satuan", "Jenis Reward", "Harga Nasabah/Satuan", "Harga Eksternal/Satuan", "Satuan Reward", "Syarat Pemilahan", "Tanggal Ditambahkan"}
	cols := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], tableHeaderRow)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheet, tableHeaderRow, 30)

	// ── Data rows ─────────────────────────────────────────────────────────────
	rowIdx := tableHeaderRow + 1
	for i, d := range rows {
		no := i + 1
		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), no)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), d.NamaSampah)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), d.Kategori)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bSt)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), d.Satuan)
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", rowIdx), d.JenisReward)
		f.SetCellStyle(sheet, fmt.Sprintf("E%d", rowIdx), fmt.Sprintf("E%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", rowIdx), d.HargaNasabah)
		f.SetCellStyle(sheet, fmt.Sprintf("F%d", rowIdx), fmt.Sprintf("F%d", rowIdx), bnSt)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", rowIdx), d.HargaEksternal)
		f.SetCellStyle(sheet, fmt.Sprintf("G%d", rowIdx), fmt.Sprintf("G%d", rowIdx), bnSt)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", rowIdx), d.SatuanReward)
		f.SetCellStyle(sheet, fmt.Sprintf("H%d", rowIdx), fmt.Sprintf("H%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", rowIdx), d.SyaratPemilahan)
		f.SetCellStyle(sheet, fmt.Sprintf("I%d", rowIdx), fmt.Sprintf("I%d", rowIdx), bSt)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", rowIdx), d.CreatedAt.Format("02 January 2006"))
		f.SetCellStyle(sheet, fmt.Sprintf("J%d", rowIdx), fmt.Sprintf("J%d", rowIdx), bcSt)

		rowIdx++
	}

	// ── Column widths ─────────────────────────────────────────────────────────
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 26)
	f.SetColWidth(sheet, "C", "C", 20)
	f.SetColWidth(sheet, "D", "D", 10)
	f.SetColWidth(sheet, "E", "E", 14)
	f.SetColWidth(sheet, "F", "F", 24)
	f.SetColWidth(sheet, "G", "G", 24)
	f.SetColWidth(sheet, "H", "H", 14)
	f.SetColWidth(sheet, "I", "I", 36)
	f.SetColWidth(sheet, "J", "J", 22)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("katalog-sampah-%s.xlsx", bankID)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}

// GET /laporan/katalog-sembako/:bank_id
func (lc *LaporanController) DownloadLaporanKatalogSembako(c *gin.Context) {
	bankID := c.Param("bank_id")

	type bankData struct {
		BankID       string  `gorm:"column:bank_id"`
		NamaBank     string  `gorm:"column:nama_bank"`
		JenisBank    string  `gorm:"column:jenis_bank"`
		ParentBankID *string `gorm:"column:parent_bank_id"`
	}
	var bank bankData
	if err := lc.DB.Raw(`SELECT bank_id, nama_bank, jenis_bank, parent_bank_id FROM bank_sampah WHERE bank_id = ?`, bankID).Scan(&bank).Error; err != nil || bank.BankID == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bank sampah tidak ditemukan"})
		return
	}

	// BSU menggunakan katalog milik parent BSI, difilter berdasarkan stok yang dimiliki BSU
	catalogBankID := bankID
	if bank.JenisBank == "bsu" && bank.ParentBankID != nil {
		catalogBankID = *bank.ParentBankID
	}

	type katalogRow struct {
		NamaBarang string    `gorm:"column:nama_barang"`
		NilaiPoin  float64   `gorm:"column:nilai_poin"`
		CreatedAt  time.Time `gorm:"column:created_at"`
	}

	var rows []katalogRow
	var queryArgs []interface{}
	stokFilter := ""
	if bank.JenisBank == "bsu" {
		stokFilter = "AND EXISTS (SELECT 1 FROM stok_sembako st WHERE st.sembako_id = ks.sembako_id AND st.bank_id = ?)"
		queryArgs = append(queryArgs, catalogBankID, bankID)
	} else {
		queryArgs = append(queryArgs, catalogBankID)
	}
	if err := lc.DB.Raw(fmt.Sprintf(`
		SELECT
			s.nama_barang,
			ks.nilai_poin,
			ks.created_at
		FROM katalog_sembako ks
		JOIN sembako s ON s.barang_id = ks.barang_id
		WHERE ks.bank_id = ? %s
		ORDER BY s.nama_barang ASC
	`, stokFilter), queryArgs...).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengambil data katalog sembako: " + err.Error()})
		return
	}

	f := excelize.NewFile()
	defer f.Close()

	sheet := "Katalog Sembako"
	f.SetSheetName("Sheet1", sheet)

	numFmt := "#,##0.00"

	styleTitle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 14},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	styleBold, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
	})
	styleTableHeader, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E65100"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "FFFFFF", Style: 1},
			{Type: "right", Color: "FFFFFF", Style: 1},
			{Type: "top", Color: "FFFFFF", Style: 1},
			{Type: "bottom", Color: "FFFFFF", Style: 1},
		},
	})
	styleBorder, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenter, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderCenterAlt, _ := f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNum, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})
	styleBorderNumAlt, _ := f.NewStyle(&excelize.Style{
		CustomNumFmt: &numFmt,
		Fill:         excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F5F5F5"}},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "CCCCCC", Style: 1},
			{Type: "right", Color: "CCCCCC", Style: 1},
			{Type: "top", Color: "CCCCCC", Style: 1},
			{Type: "bottom", Color: "CCCCCC", Style: 1},
		},
	})

	// ── Row 1: Title ──────────────────────────────────────────────────────────
	f.MergeCell(sheet, "A1", "D1")
	f.SetCellValue(sheet, "A1", "KATALOG SEMBAKO BANK SAMPAH")
	f.SetCellStyle(sheet, "A1", "D1", styleTitle)
	f.SetRowHeight(sheet, 1, 32)

	// ── Rows 3+: Info header ──────────────────────────────────────────────────
	infoRows := [][2]string{
		{"Nama Bank Sampah", bank.NamaBank},
		{"Bank ID", bank.BankID},
		{"Tanggal Cetak", time.Now().Format("02 January 2006, 15:04:05")},
		{"Total Item", fmt.Sprintf("%d jenis barang", len(rows))},
	}
	const infoStartRow = 3
	for i, row := range infoRows {
		r := infoStartRow + i
		f.MergeCell(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("A%d", r), row[0])
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", r), fmt.Sprintf("B%d", r), styleBold)
		f.MergeCell(sheet, fmt.Sprintf("C%d", r), fmt.Sprintf("D%d", r))
		f.SetCellValue(sheet, fmt.Sprintf("C%d", r), row[1])
	}

	// ── Table header ──────────────────────────────────────────────────────────
	tableHeaderRow := infoStartRow + len(infoRows) + 1
	headers := []string{"No", "Nama Barang", "Nilai Poin", "Tanggal Ditambahkan"}
	cols := []string{"A", "B", "C", "D"}
	for i, h := range headers {
		cell := fmt.Sprintf("%s%d", cols[i], tableHeaderRow)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, styleTableHeader)
	}
	f.SetRowHeight(sheet, tableHeaderRow, 25)

	// ── Data rows ─────────────────────────────────────────────────────────────
	rowIdx := tableHeaderRow + 1
	for i, d := range rows {
		no := i + 1
		bSt := styleBorder
		bcSt := styleBorderCenter
		bnSt := styleBorderNum
		if no%2 == 0 {
			bSt = styleBorderAlt
			bcSt = styleBorderCenterAlt
			bnSt = styleBorderNumAlt
		}

		f.SetCellValue(sheet, fmt.Sprintf("A%d", rowIdx), no)
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", rowIdx), fmt.Sprintf("A%d", rowIdx), bcSt)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", rowIdx), d.NamaBarang)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", rowIdx), fmt.Sprintf("B%d", rowIdx), bSt)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", rowIdx), d.NilaiPoin)
		f.SetCellStyle(sheet, fmt.Sprintf("C%d", rowIdx), fmt.Sprintf("C%d", rowIdx), bnSt)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", rowIdx), d.CreatedAt.Format("02 January 2006"))
		f.SetCellStyle(sheet, fmt.Sprintf("D%d", rowIdx), fmt.Sprintf("D%d", rowIdx), bcSt)

		rowIdx++
	}

	// ── Column widths ─────────────────────────────────────────────────────────
	f.SetColWidth(sheet, "A", "A", 6)
	f.SetColWidth(sheet, "B", "B", 32)
	f.SetColWidth(sheet, "C", "C", 16)
	f.SetColWidth(sheet, "D", "D", 22)

	// ── Send response ─────────────────────────────────────────────────────────
	filename := fmt.Sprintf("katalog-sembako-%s.xlsx", bankID)
	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Header("Cache-Control", "no-cache")

	if err := f.Write(c.Writer); err != nil {
		fmt.Printf("[Laporan] Gagal menulis Excel: %v\n", err)
	}
}
