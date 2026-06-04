package utils

import (
	"regexp"
	"strings"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]`)

// NormalizeString menjadikan string lowercase, membuang spasi, tanda baca,
// dan karakter non-alfanumerik lainnya — dipakai untuk cek duplikat nama.
func NormalizeString(s string) string {
	s = strings.ToLower(s)
	s = nonAlphanumeric.ReplaceAllString(s, "")
	return s
}
