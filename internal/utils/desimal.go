package utils

import "math"

// Round4 menyamakan nilai float ke skala kolom DECIMAL(20,4).
// Pembulatannya half-away-from-zero, sama dengan pembulatan numeric di Postgres.
// Dipakai sebelum membandingkan hasil hitungan, supaya noise float
// (mis. 2.78e-17) tidak ikut menentukan keputusan.
// Presisi 4 desimal terjaga sampai sekitar ±900 miliar; di atas itu float64
// sendiri memang sudah tidak sanggup menyimpan 4 angka di belakang koma.
func Round4(x float64) float64 {
	return math.Round(x*1e4) / 1e4
}
