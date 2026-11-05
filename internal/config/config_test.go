package config

import (
	"reflect"
	"testing"
)

func TestParseDaftar(t *testing.T) {
	kasus := []struct {
		nama  string
		input string
		mau   []string
	}{
		{"satu origin", "https://admin.enviroo.tech", []string{"https://admin.enviroo.tech"}},
		{"spasi dan item kosong dibuang", "a, b ,,c", []string{"a", "b", "c"}},
		{"string kosong", "", []string{}},
		{"cuma koma dan spasi", " , ", []string{}},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			if hasil := parseDaftar(k.input); !reflect.DeepEqual(hasil, k.mau) {
				t.Errorf("parseDaftar(%q) = %v, mau %v", k.input, hasil, k.mau)
			}
		})
	}
}

func TestLoadConfig_CORSOriginsDefaultLocalhost(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "")
	cfg := LoadConfig()
	if !reflect.DeepEqual(cfg.CORSOrigins, []string{"http://localhost:5173"}) {
		t.Errorf("default CORSOrigins = %v", cfg.CORSOrigins)
	}
}

func TestLoadConfig_CORSOriginsDariEnv(t *testing.T) {
	t.Setenv("CORS_ORIGINS", "https://admin.enviroo.tech, http://localhost:5173")
	cfg := LoadConfig()
	mau := []string{"https://admin.enviroo.tech", "http://localhost:5173"}
	if !reflect.DeepEqual(cfg.CORSOrigins, mau) {
		t.Errorf("CORSOrigins = %v, mau %v", cfg.CORSOrigins, mau)
	}
}
