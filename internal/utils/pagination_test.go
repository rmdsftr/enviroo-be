package utils_test

import (
	"fmt"
	"testing"

	"enviroo-be/internal/utils"

	"github.com/stretchr/testify/assert"
)

func TestParsePage_KosongBerartiTanpaPaginasi(t *testing.T) {
	assert.Equal(t, 0, utils.ParsePage(""), "page tidak dikirim = ambil semua data tanpa paginasi")
}

func TestParsePage_Angka(t *testing.T) {
	kasus := map[string]int{"1": 1, "3": 3, "25": 25}
	for masukan, harapan := range kasus {
		assert.Equal(t, harapan, utils.ParsePage(masukan), masukan)
	}
}

func TestParsePage_TidakValidJadiHalamanPertama(t *testing.T) {
	for _, masukan := range []string{"0", "-5", "abc", "2.5", " 2"} {
		assert.Equal(t, 1, utils.ParsePage(masukan), "%q", masukan)
	}
}

func TestOffset(t *testing.T) {
	kasus := []struct {
		page, limit, harapan int
	}{
		{1, 12, 0},
		{2, 12, 12},
		{3, 10, 20},
		{0, 12, 0},
		{-1, 12, 0},
	}
	for _, k := range kasus {
		assert.Equal(t, k.harapan, utils.Offset(k.page, k.limit), "page=%d limit=%d", k.page, k.limit)
	}
}

func TestNewPageMeta(t *testing.T) {
	kasus := []struct {
		nama               string
		page, limit, total int
		harapan            utils.PageMeta
	}{
		{"tanpa data", 1, 12, 0, utils.PageMeta{Page: 1, Limit: 12, Total: 0, TotalPages: 0}},
		{"pas satu halaman", 1, 12, 12, utils.PageMeta{Page: 1, Limit: 12, Total: 12, TotalPages: 1}},
		{"sisa dibulatkan ke atas", 1, 12, 13, utils.PageMeta{Page: 1, Limit: 12, Total: 13, TotalPages: 2}},
		{"halaman tengah", 2, 10, 25, utils.PageMeta{Page: 2, Limit: 10, Total: 25, TotalPages: 3}},
		{"halaman nol jadi satu", 0, 10, 25, utils.PageMeta{Page: 1, Limit: 10, Total: 25, TotalPages: 3}},
		{"limit nol tanpa bagi nol", 1, 0, 25, utils.PageMeta{Page: 1, Limit: 0, Total: 25, TotalPages: 0}},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			meta := utils.NewPageMeta(k.page, k.limit, k.total)

			if assert.NotNil(t, meta) {
				assert.Equal(t, k.harapan, *meta, fmt.Sprintf("page=%d limit=%d total=%d", k.page, k.limit, k.total))
			}
		})
	}
}
