package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func routerUji(rl gin.HandlerFunc) (*gin.Engine, *[]string) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	terbaca := []string{}

	r.POST("/coba", rl, func(c *gin.Context) {
		var isi struct {
			Email string `json:"email"`
		}
		if err := c.ShouldBindJSON(&isi); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		terbaca = append(terbaca, isi.Email)
		c.JSON(http.StatusOK, gin.H{"email": isi.Email})
	})

	return r, &terbaca
}

func postJSON(r *gin.Engine, body string, ip string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/coba", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if ip != "" {
		req.Header.Set("X-Forwarded-For", ip)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDimBody_BodyTetapTerbacaHandler(t *testing.T) {
	lim := NewIPRateLimiter("uji", rate.Every(time.Millisecond), 1000)
	r, terbaca := routerUji(lim.Middleware(DimBody()))

	w := postJSON(r, `{"email":"budi@enviroo.test"}`, "")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"budi@enviroo.test"}, *terbaca)
	assert.JSONEq(t, `{"email":"budi@enviroo.test"}`, w.Body.String())
}

func TestDimBody_BodyTetapUtuhDiBerbagaiKondisi(t *testing.T) {
	kasus := []struct {
		nama        string
		body        string
		contentType string
		metode      string
		mauStatus   int
	}{
		{"json sah", `{"email":"a@b.test"}`, "application/json", http.MethodPost, http.StatusOK},
		{"json rusak", `{`, "application/json", http.MethodPost, http.StatusBadRequest},
		{"body kosong", ``, "application/json", http.MethodPost, http.StatusBadRequest},
		{"bukan json", `email=a@b.test`, "text/plain", http.MethodPost, http.StatusBadRequest},
		{"json dengan charset", `{"email":"a@b.test"}`, "application/json; charset=utf-8", http.MethodPost, http.StatusOK},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			lim := NewIPRateLimiter("uji", rate.Every(time.Millisecond), 1000)
			r, _ := routerUji(lim.Middleware(DimBody()))

			req := httptest.NewRequest(k.metode, "/coba", strings.NewReader(k.body))
			req.Header.Set("Content-Type", k.contentType)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, k.mauStatus, w.Code)
			assert.NotEqual(t, http.StatusTooManyRequests, w.Code,
				"limiter tidak boleh menolak gara-gara body tidak bisa diparse")
		})
	}
}

func TestDimBody_IdentitasDariBody(t *testing.T) {
	kasus := []struct {
		nama string
		body string
		mau  string
	}{
		{"email dipakai", `{"email":"Budi@Enviroo.Test"}`, "budi@enviroo.test"},
		{"spasi dibuang", `{"email":"  budi@enviroo.test  "}`, "budi@enviroo.test"},
		{"user_id dipakai kalau email kosong", `{"user_id":"USR0000001"}`, "USR0000001"},
		{"email menang atas user_id", `{"email":"a@b.test","user_id":"USR1"}`, "a@b.test"},
		{"dua-duanya kosong", `{"otp":"123456"}`, ""},
		{"json rusak", `{`, ""},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(k.body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req

			assert.Equal(t, k.mau, identitasDariBody(c))

			sisa, err := io.ReadAll(c.Request.Body)
			assert.NoError(t, err)
			assert.Equal(t, k.body, string(sisa), "body harus dipasang ulang utuh")
		})
	}
}

func TestDimBody_EmailBerbedaBucketBerbeda(t *testing.T) {
	lim := NewIPRateLimiterWithBlock("uji", rate.Every(time.Hour), 2, 10*time.Minute)
	r, _ := routerUji(lim.Middleware(DimBody()))

	assert.Equal(t, http.StatusOK, postJSON(r, `{"email":"a@b.test"}`, "").Code)
	assert.Equal(t, http.StatusOK, postJSON(r, `{"email":"a@b.test"}`, "").Code)
	assert.Equal(t, http.StatusTooManyRequests, postJSON(r, `{"email":"a@b.test"}`, "").Code)

	assert.Equal(t, http.StatusOK, postJSON(r, `{"email":"lain@b.test"}`, "").Code,
		"email lain harus punya bucket sendiri")
}

func TestDimBody_RotasiIPTidakMemberiKuotaBaru(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lim := NewIPRateLimiterWithBlock("uji", rate.Every(time.Hour), 2, 10*time.Minute)
	r := gin.New()
	assert.NoError(t, r.SetTrustedProxies([]string{"0.0.0.0/0"}))
	r.POST("/coba", lim.Middleware(DimBody()), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := `{"email":"korban@enviroo.test"}`
	assert.Equal(t, http.StatusOK, postJSON(r, body, "203.0.113.1").Code)
	assert.Equal(t, http.StatusOK, postJSON(r, body, "203.0.113.2").Code)

	for i := 3; i <= 8; i++ {
		w := postJSON(r, body, "203.0.113."+strconv.Itoa(i))
		assert.Equal(t, http.StatusTooManyRequests, w.Code,
			"ganti IP tidak boleh memberi kuota baru untuk email yang sama")
	}
}

func TestDimIP_EmailSamaIPBerbedaBucketBerbeda(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lim := NewIPRateLimiter("uji", rate.Every(time.Hour), 1)
	r := gin.New()
	assert.NoError(t, r.SetTrustedProxies([]string{"0.0.0.0/0"}))
	r.POST("/coba", lim.Middleware(DimIP()), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	body := `{"email":"sama@enviroo.test"}`
	assert.Equal(t, http.StatusOK, postJSON(r, body, "203.0.113.1").Code)
	assert.Equal(t, http.StatusTooManyRequests, postJSON(r, body, "203.0.113.1").Code)
	assert.Equal(t, http.StatusOK, postJSON(r, body, "203.0.113.9").Code,
		"IP lain harus punya bucket sendiri")
}

func TestDimUser_TanpaTokenDilewatiTanpa401(t *testing.T) {
	lim := NewIPRateLimiter("uji", rate.Every(time.Hour), 1)
	gin.SetMode(gin.TestMode)
	r := gin.New()

	var claimsAda bool
	r.GET("/coba", lim.Middleware(DimUser()), func(c *gin.Context) {
		_, claimsAda = c.Get(ClaimsKey)
		c.Status(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/coba", nil))
		assert.Equal(t, http.StatusOK, w.Code,
			"tanpa token dimensi user harus dilewati, bukan ditolak")
	}
	assert.False(t, claimsAda, "limiter tidak boleh mengisi ClaimsKey")
}

func TestDimUser_TokenRusakDilewati(t *testing.T) {
	lim := NewIPRateLimiter("uji", rate.Every(time.Hour), 1)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/coba", lim.Middleware(DimUser()), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/coba", nil)
		req.Header.Set("Authorization", "Bearer bukan.token.valid")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}

func TestMiddleware_TanpaDimensiDefaultKeIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	lim := NewIPRateLimiter("uji", rate.Every(time.Hour), 1)
	r := gin.New()
	assert.NoError(t, r.SetTrustedProxies([]string{"0.0.0.0/0"}))
	r.GET("/coba", lim.Middleware(), func(c *gin.Context) { c.Status(http.StatusOK) })

	kirim := func(ip string) int {
		req := httptest.NewRequest(http.MethodGet, "/coba", nil)
		req.Header.Set("X-Forwarded-For", ip)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w.Code
	}

	assert.Equal(t, http.StatusOK, kirim("203.0.113.1"))
	assert.Equal(t, http.StatusTooManyRequests, kirim("203.0.113.1"))
	assert.Equal(t, http.StatusOK, kirim("203.0.113.2"))
}

func TestMiddleware_PesanDanHeader429(t *testing.T) {
	t.Run("dengan block ada Retry-After dan pesan menit", func(t *testing.T) {
		lim := NewIPRateLimiterWithBlock("uji", rate.Every(time.Hour), 1, 10*time.Minute)
		r, _ := routerUji(lim.Middleware(DimBody()))

		postJSON(r, `{"email":"a@b.test"}`, "")
		w := postJSON(r, `{"email":"a@b.test"}`, "")

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Equal(t, "600", w.Header().Get("Retry-After"))

		var isi map[string]string
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &isi))
		assert.Equal(t, "Terlalu banyak percobaan. Silakan coba lagi dalam 10 menit.", isi["error"])
	})

	t.Run("tanpa block pesan generik tanpa Retry-After", func(t *testing.T) {
		lim := NewIPRateLimiter("uji", rate.Every(time.Hour), 1)
		r, _ := routerUji(lim.Middleware(DimBody()))

		postJSON(r, `{"email":"a@b.test"}`, "")
		w := postJSON(r, `{"email":"a@b.test"}`, "")

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		assert.Empty(t, w.Header().Get("Retry-After"))

		var isi map[string]string
		assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &isi))
		assert.Equal(t, "Terlalu banyak permintaan. Silakan coba lagi beberapa saat.", isi["error"])
	})
}

func TestMiddleware_SemuaDimensiDibebaniWalauSatuMenolak(t *testing.T) {
	gin.SetMode(gin.TestMode)
	limIP := NewIPRateLimiter("uji", rate.Every(time.Hour), 1)
	limBody := NewIPRateLimiter("uji", rate.Every(time.Hour), 5)

	r := gin.New()
	assert.NoError(t, r.SetTrustedProxies([]string{"0.0.0.0/0"}))
	r.POST("/coba",
		limIP.Middleware(DimIP()),
		limBody.Middleware(DimBody()),
		func(c *gin.Context) { c.Status(http.StatusOK) })

	assert.Equal(t, http.StatusOK, postJSON(r, `{"email":"a@b.test"}`, "203.0.113.1").Code)
	assert.Equal(t, http.StatusTooManyRequests, postJSON(r, `{"email":"a@b.test"}`, "203.0.113.1").Code)

	assert.Equal(t, http.StatusOK, postJSON(r, `{"email":"a@b.test"}`, "203.0.113.2").Code,
		"bucket email masih punya sisa kuota")
}

func TestIdentitasDariBody_BodyBesarDilewati(t *testing.T) {
	besar := bytes.Repeat([]byte("a"), maksIntipBody+1)
	body := `{"email":"` + string(besar) + `"}`

	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	assert.Empty(t, identitasDariBody(c), "body di atas plafon harus dilewati")

	sisa, err := io.ReadAll(c.Request.Body)
	assert.NoError(t, err)
	assert.Equal(t, body, string(sisa), "body harus tetap utuh walau dilewati")
}
