package routes

import (
	"enviroo-be/internal/controllers"
	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/internal/repositories"
	"enviroo-be/internal/services"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB, cfStorage *storage.CloudflareStorage, mailer *utils.Mailer, fcmClient *utils.FCMClient) {
	notifRepo := repositories.NewNotifikasiRepository(db)
	notifSvc := services.NewNotifikasiService(notifRepo, fcmClient)

	api := r.Group("")
	{
		api.GET("/ping", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "pong",
			})
		})
	}

	// ─── Auth (public) ───────────────────────────────────────────────────────────
	auth := r.Group("/auth")
	{
		authController := controllers.NewAuthController(db, mailer)

		// Rate limiter ketat untuk endpoint auth publik (anti brute-force &
		// tebak OTP): ~10 request/menit per IP, burst 5. Satu instance dibagi
		// ke semua endpoint sensitif agar kuota tidak bisa dipecah antar-endpoint.
		authLimiter := middleware.NewIPRateLimiter(rate.Every(6*time.Second), 5)
		authRL := authLimiter.Middleware()

		auth.POST("/login", authRL, authController.Login)
		auth.POST("/refresh", authController.RefreshToken)
		auth.POST("/logout", authController.Logout)
		auth.POST("/cek-user-mobile", authRL, authController.CekUserMobile)

		auth.POST("/aktivasi-akun", authRL, authController.AktivasiAkun)
		auth.POST("/deactivate-akun", authController.DeactivateAkun)
		auth.POST("/generate-reactivate-akun", authController.GenerateReactivateAkun)
		auth.POST("/reactivate-akun", authRL, authController.ReactivateAkun)

		auth.POST("/forget-password/send-email", authRL, authController.SendEmailForgetPassword)
		auth.POST("/forget-password/verifikasi-otp", authRL, authController.VerifikasiOTP)
		auth.POST("/forget-password/reset-password", authRL, authController.ResetPassword)

		authProtected := auth.Group("", middleware.RequireAuth(db))
		authProtected.GET("/me", authController.Me)
		authProtected.POST("/change-password", authController.ChangePassword)
		authProtected.POST("/switch-role", authController.SwitchRole)
	}

	// ─── Middleware shortcuts ────────────────────────────────────────────────────
	requireAuth        := middleware.RequireAuth(db)
	superadminRole     := middleware.RequireRole(models.SuperAdmin)
	nasabahRole        := middleware.RequireRole(models.RoleNasabah)
	adminBSI           := middleware.RequireRole(models.AdminBSI)
	petugasBSI         := middleware.RequireRole(models.PetugasBSI)
	petugasBSU         := middleware.RequireRole(models.PetugasBSU)
	allAdmin           := middleware.RequireRole(models.AdminBSI, models.AdminBSU, models.AdminBSM)
	allStaff           := middleware.RequireRole(models.AdminBSI, models.PetugasBSI, models.AdminBSU, models.PetugasBSU, models.AdminBSM, models.PetugasBSM)
	superadminAndAdmin := middleware.RequireRole(models.SuperAdmin, models.AdminBSI, models.AdminBSU, models.AdminBSM)
	nonNasabah         := middleware.RequireRole(models.SuperAdmin, models.AdminBSI, models.PetugasBSI, models.AdminBSU, models.PetugasBSU, models.AdminBSM, models.PetugasBSM)
	staffBSU           := middleware.RequireRole(models.AdminBSU, models.PetugasBSU)
	staffBSIBSM        := middleware.RequireRole(models.AdminBSI, models.PetugasBSI, models.AdminBSM, models.PetugasBSM)
	staffBSIBSU        := middleware.RequireRole(models.AdminBSI, models.PetugasBSI, models.AdminBSU, models.PetugasBSU)
	adminBSIBSM        := middleware.RequireRole(models.AdminBSI, models.AdminBSM)
	petugasBSIBSM      := middleware.RequireRole(models.PetugasBSI, models.PetugasBSM)
	petugasAll         := middleware.RequireRole(models.PetugasBSI, models.PetugasBSU, models.PetugasBSM)
	bankParam          := middleware.RequireSameBankParam("bank_id")
	bsiParam           := middleware.RequireSameBankParam("bsi_id")
	bsuParam           := middleware.RequireSameBankParam("bsu_id")
	// BSI staff boleh akses data bank lain (BSU di bawahnya) tanpa ownership check
	bankParamOrBSI     := middleware.RequireSameBankParam("bank_id", models.AdminBSI, models.PetugasBSI)
	// Nasabah hanya boleh mengakses data nasabah miliknya sendiri (role lain di-skip)
	sameNasabah        := middleware.RequireSameNasabah(db, "nasabah_id")

	// ─── Bank ───────────────────────────────────────────────────────────────────
	bank := r.Group("/bank", requireAuth)
	{
		bankController := controllers.NewBankController(db, cfStorage, mailer)
		bank.PATCH("/aktivasi/:bank_id", superadminAndAdmin, bankParamOrBSI, bankController.AktivasiBank)
		bank.GET("/get-nasabah/:bank_id", bankController.GetNasabahByBankID)
		bank.GET("/get-all", bankController.GetAllBankSampah)
		bank.PATCH("/edit-profil/:bank_id", allAdmin, bankParam, bankController.EditProfilBankSampah)
	}

	bsi := r.Group("/bsi", requireAuth)
	{
		bsiController := controllers.NewBSIController(db, cfStorage, mailer)
		bsi.POST("/add-bsi", superadminRole, bsiController.AddNewBSI)
		bsi.GET("/get-bsi", middleware.RequireRole(models.SuperAdmin, models.AdminBSI), bsiController.GetBSI)
		bsi.GET("/get-unit/:bank_id", middleware.RequireRole(models.SuperAdmin, models.AdminBSI, models.PetugasBSI), bankParam, bsiController.GetUnitBSI)
		bsi.POST("/add-unit/:bank_id", middleware.RequireRole(models.SuperAdmin, models.AdminBSI), bankParam, bsiController.AddNewUnit)
	}
	bsm := r.Group("/bsm", requireAuth, superadminRole)
	{
		bsmController := controllers.NewBSMController(db, cfStorage, mailer)
		bsm.POST("/add-bsm", bsmController.AddNewBSM)
		bsm.GET("/get-bsm", bsmController.GetBSM)
	}
	bsu := r.Group("/bsu", requireAuth, superadminRole)
	{
		bsuController := controllers.NewBSUController(db, cfStorage, mailer)
		bsu.POST("/add-bsu", bsuController.AddNewBSU)
		bsu.GET("/get-bsu", bsuController.GetBSU)
		bsu.GET("/get-bsu/:bank_id", bsuController.GetBSUbyBankID)
	}

	// ─── Nasabah ────────────────────────────────────────────────────────────────
	nasabah := r.Group("/nasabah", requireAuth)
	{
		nasabahController := controllers.NewNasabahController(db, cfStorage, mailer)
		nasabah.POST("/add-nasabah", allAdmin, nasabahController.AddNewNasabah)
		nasabah.POST("/add-nasabah-from-old-user", allAdmin, nasabahController.AddNewNasabahOldUser)
		nasabah.GET("/get-afiliasi", adminBSI, nasabahController.GetAfiliasi)
		nasabah.GET("/get-nasabah", superadminAndAdmin, nasabahController.GetNasabah)
		nasabah.GET("/:bank_id", nonNasabah, bankParamOrBSI, nasabahController.NasabahBankSampah)
		nasabah.DELETE("/:nasabah_id", superadminAndAdmin, nasabahController.DeleteNasabah)
	}

	// ─── Users ──────────────────────────────────────────────────────────────────
	superadminMgmt := r.Group("/superadmin", requireAuth, superadminRole)
	{
		userController := controllers.NewUserController(db, mailer, cfStorage)
		superadminMgmt.GET("/list", userController.GetListSuperadmin)
		superadminMgmt.POST("/add", userController.AddSuperadmin)
		superadminMgmt.PATCH("/nonaktif/:admin_id", userController.NonaktifkanSuperadmin)
	}

	user := r.Group("/users", requireAuth)
	{
		userController := controllers.NewUserController(db, mailer, cfStorage)
		user.POST("/add-user", superadminAndAdmin, userController.AddUser)
		user.POST("/update-profil/:user_id", userController.UpdateProfilUser)
		user.GET("/get-nonadmin-user", superadminAndAdmin, userController.GetNonAdminUser)
		user.GET("/get-nonnasabah-user", superadminAndAdmin, userController.GetNonNasabahUser)
		user.GET("/active-admin/:admin_id", allAdmin, userController.ActiveAdmin)
		user.GET("/active-petugas/:admin_id", petugasAll, userController.ActivePetugas)
		user.GET("/active-user/:user_id", userController.ActiveUser)
		user.PATCH("/update-fcm-token", userController.UpdateFCMToken)
		user.GET("/log/:user_id", userController.LogAkun)
		user.GET("/get-all", superadminRole, userController.GetAllUsers)
		user.GET("/detail-user/:user_id", superadminRole, userController.GetDetailUser)
		user.DELETE("/delete-user/:user_id", superadminRole, userController.DeleteUser)
	}

	// ─── Master Data (superadmin) ────────────────────────────────────────────────
	masterData := r.Group("/master", requireAuth, superadminRole)
	{
		masterDataController := controllers.NewMasterDataController(db)
		masterData.GET("/sampah", masterDataController.GetAllMasterSampah)
		masterData.POST("/sampah", masterDataController.CreateMasterSampah)
		masterData.GET("/sampah/statistik", masterDataController.GetStatistikSampah)
		masterData.GET("/sampah/favorit", masterDataController.GetFavoritSampah)
		masterData.GET("/sampah/per-kategori", masterDataController.GetSampahPerKategori)
		masterData.PATCH("/sampah/:sarok_id", masterDataController.UpdateMasterSampah)
		masterData.DELETE("/sampah/:sarok_id", masterDataController.DeleteMasterSampah)

		masterData.GET("/sembako", masterDataController.GetAllMasterSembako)
		masterData.POST("/sembako", masterDataController.CreateMasterSembako)
		masterData.GET("/sembako/statistik", masterDataController.GetStatistikSembako)
		masterData.GET("/sembako/favorit", masterDataController.GetFavoritSembako)
		masterData.PATCH("/sembako/:barang_id", masterDataController.UpdateMasterSembako)
		masterData.DELETE("/sembako/:barang_id", masterDataController.DeleteMasterSembako)
	}

	// ─── Statistik ──────────────────────────────────────────────────────────────
	statistik := r.Group("/statistik", requireAuth, superadminRole)
	{
		statistikController := controllers.NewStatistikController(db)
		statistik.GET("/bank-sampah", statistikController.GetBankSampahStatistik)
		statistik.GET("/superadmin/ringkasan", statistikController.GetRingkasanSuperadmin)
		statistik.GET("/superadmin/tren-penjualan", statistikController.GetTrenPenjualan)
		statistik.GET("/superadmin/ranking-bank", statistikController.GetRankingBank)
		statistik.GET("/superadmin/volume-sampah", statistikController.GetVolumeSampahNasabah)
	}
	statistikOpen := r.Group("/statistik", requireAuth)
	{
		statistikController := controllers.NewStatistikController(db)
		statistikOpen.GET("/setoran-sampah/:bank_id", statistikController.GetStatistikSetoranSampahBank)
		statistikOpen.GET("/kontribusi-nasabah/:bank_id", statistikController.GetKontribusiNasabah)
		statistikOpen.GET("/penjualan-sampah/:bank_id", statistikController.GetStatistikPenjualanSampahBank)
		statistikOpen.GET("/masuk-sampah/:bank_id", statistikController.GetStatistikMasukSampahBSU)
	}

	// ─── Profil ─────────────────────────────────────────────────────────────────
	profil := r.Group("/profil", requireAuth)
	{
		profilController := controllers.NewProfilController(db, cfStorage)
		profil.GET("/bank-sampah/:bank_id", nonNasabah, bankParamOrBSI, profilController.GetProfilBankSampah)
		profil.GET("/bank-sampah/:bank_id/history", allAdmin, bankParamOrBSI, profilController.GetHistoryAkunBank)
		profil.GET("/nasabah/:nasabah_id", sameNasabah, profilController.GetProfilNasabah)
		profil.PATCH("/nasabah/aktivasi/:nasabah_id", allAdmin, profilController.AktivasiNasabah)
		profil.DELETE("/bank-sampah/:bank_id", superadminRole, profilController.DeleteBankSampah)
		profil.GET("/:user_id", profilController.GetProfilUser)
		profil.GET("/detail-nasabah/:nasabah_id", sameNasabah, profilController.GetDetailNasabah)
		profil.GET("/detail-petugas/:petugas_id", nonNasabah, profilController.GetDetailPetugas)
		profil.GET("/detail-bank/:bank_id", middleware.RequireRole(models.AdminBSI, models.PetugasBSI, models.AdminBSU, models.PetugasBSU, models.AdminBSM, models.PetugasBSM, models.RoleNasabah), bankParamOrBSI, profilController.DetailBankSampah)
	}

	// ─── Admin ──────────────────────────────────────────────────────────────────
	admin := r.Group("/admin", requireAuth)
	{
		adminController := controllers.NewAdminController(db, mailer)
		admin.GET("/get-admin/:bank_id", nonNasabah, bankParamOrBSI, adminController.GetAdminBankSampah)
		admin.POST("/add-admin-bank-sampah", allAdmin, adminController.AddAdminBankSampah)
		admin.DELETE("/delete-staff/:admin_id", allAdmin, adminController.DeleteStaffBankSampah)
	}

	// ─── Lokasi ─────────────────────────────────────────────────────────────────
	lokasiController := controllers.NewLokasiController(db)

	lokasiOpen := r.Group("/lokasi", requireAuth)
	{
		lokasiOpen.GET("/bank-sampah", lokasiController.GetLokasiBankSampah)
		lokasiOpen.GET("/statistik-kecamatan", lokasiController.StatistikBankSampahPerKecamatan)

		kecamatan := lokasiOpen.Group("/kecamatan")
		kecamatan.GET("", lokasiController.GetAllKecamatan)
		kecamatan.GET("/:id", lokasiController.GetKecamatanByID)
		kecamatan.GET("/:id/kelurahan", lokasiController.GetKelurahanByKecamatan)

		kelurahan := lokasiOpen.Group("/kelurahan")
		kelurahan.GET("", lokasiController.GetAllKelurahan)
		kelurahan.GET("/:id", lokasiController.GetKelurahanByID)
	}

	lokasiAdmin := r.Group("/lokasi", requireAuth, superadminRole)
	{
		kecamatan := lokasiAdmin.Group("/kecamatan")
		kecamatan.POST("", lokasiController.CreateKecamatan)
		kecamatan.PATCH("/:id", lokasiController.UpdateKecamatan)
		kecamatan.DELETE("/:id", lokasiController.DeleteKecamatan)

		kelurahan := lokasiAdmin.Group("/kelurahan")
		kelurahan.POST("", lokasiController.CreateKelurahan)
		kelurahan.PATCH("/:id", lokasiController.UpdateKelurahan)
		kelurahan.DELETE("/:id", lokasiController.DeleteKelurahan)
	}

	// ─── Katalog ────────────────────────────────────────────────────────────────
	katalog := r.Group("/katalog", requireAuth)
	{
		katalogController := controllers.NewKatalogController(db, cfStorage)
		katalog.GET("/master-sampah", superadminAndAdmin, katalogController.GetMasterSampah)
		katalog.POST("/add-sampah/:bank_id", allAdmin, bankParam, katalogController.AddKatalog)
		katalog.PATCH("/edit-sampah/:sampah_id", allAdmin, katalogController.EditKatalog)
		katalog.DELETE("/delete-sampah/:sampah_id", allAdmin, katalogController.DeleteKatalogSampah)
		katalog.GET("/get-sampah/:bank_id", middleware.RequireRole(models.AdminBSI, models.PetugasBSI, models.AdminBSU, models.PetugasBSU, models.AdminBSM, models.PetugasBSM, models.RoleNasabah), bankParamOrBSI, katalogController.GetKatalogSampahBank)
		katalog.GET("/get-detail/:sampah_id", middleware.RequireRole(models.AdminBSI, models.PetugasBSI, models.AdminBSU, models.PetugasBSU, models.AdminBSM, models.PetugasBSM, models.RoleNasabah), katalogController.GetDetailSampah)
		katalog.POST("/add-kategori", superadminRole, katalogController.AddNewKategori)
		katalog.GET("/get-kategori", katalogController.GetKategori)
		katalog.PATCH("/update-kategori/:kategori_id", superadminRole, katalogController.UpdateKategori)
		katalog.DELETE("/delete-kategori/:kategori_id", superadminRole, katalogController.DeleteKategori)
	}

	// ─── Sembako ────────────────────────────────────────────────────────────────
	sembako := r.Group("/sembako", requireAuth)
	{
		sembakoController := controllers.NewSembakoController(db, cfStorage, notifSvc)
		sembako.GET("/get-master", superadminAndAdmin, sembakoController.GetMasterSembako)
		sembako.POST("/add-sembako/:bank_id", allAdmin, bankParam, sembakoController.AddNewSembako)
		sembako.GET("/get-sembako/:bank_id", bankParamOrBSI, sembakoController.GetSembakoBank)
		sembako.DELETE("/delete-sembako/:sembako_id", allAdmin, sembakoController.DeleteSembako)
		sembako.PATCH("/edit-sembako/:sembako_id", allAdmin, sembakoController.EditSembako)
		sembako.GET("/detail-sembako/:sembako_id", sembakoController.GetDetailSembako)
		sembako.GET("/list-distribusi/:bank_id", staffBSIBSU, bankParamOrBSI, sembakoController.ListDistribusiSembako)
		sembako.GET("/detail-distribusi/:distribusi_id", staffBSIBSU, sembakoController.GetDetailDistribusiSembako)
		sembako.POST("/qr-distribusi", petugasBSI, sembakoController.QRDistribusiSembako)
		sembako.POST("/add-distribusi-bsu", middleware.RequireRole(models.PetugasBSI, models.PetugasBSU), sembakoController.AddNewDistribusiSembakoBSU)
		sembako.POST("/preview-distribusi-bsu/:bsi_id/:bsu_id", petugasBSI, bsiParam, sembakoController.PreviewDistribusiSembakoBSU)
	}

	// ─── Konten ─────────────────────────────────────────────────────────────────
	{
		kontenController := controllers.NewKontenController(db, cfStorage)

		kontenRead := r.Group("/konten", requireAuth)
		kontenRead.GET("/all-konten", kontenController.GetAllKontenSuperadmin)
		kontenRead.GET("/all-konten/:bank_id", kontenController.GetAllKonten)
		kontenRead.GET("/get-konten/:konten_id", kontenController.GetKontenByID)

		kontenAdmin := r.Group("/konten", requireAuth, superadminAndAdmin)
		kontenAdmin.POST("/add-konten/:admin_id", kontenController.AddNewKonten)
		kontenAdmin.DELETE("/delete-konten/:konten_id", kontenController.DeleteKonten)
		kontenAdmin.PATCH("/edit-konten/:konten_id", kontenController.EditKonten)
	}

	// ─── Jadwal ─────────────────────────────────────────────────────────────────
	jadwal := r.Group("/jadwal", requireAuth)
	{
		jadwalController := controllers.NewJadwalController(db)
		jadwal.GET("/get-all", nonNasabah, jadwalController.GetAllJadwal)
		jadwal.GET("/get-jadwal/:bank_id", allStaff, bankParamOrBSI, jadwalController.GetJadwalBank)
		jadwal.POST("/add-jadwal/:bank_id", allAdmin, bankParam, jadwalController.AddNewJadwal)
		jadwal.POST("/add-jadwal-batch/:bank_id", allAdmin, bankParam, jadwalController.AddJadwalBatch)
		jadwal.DELETE("/delete-jadwal/:jadwal_id", allAdmin, jadwalController.DeleteJadwal)
		jadwal.PATCH("/update-jadwal/:jadwal_id", allAdmin, jadwalController.UpdateJadwal)
	}

	// ─── Penimbangan ────────────────────────────────────────────────────────────
	penimbangan := r.Group("/penimbangan", requireAuth)
	{
		penimbanganController := controllers.NewPenimbanganController(db)
		penimbangan.GET("/check/:bank_id", petugasAll, bankParam, penimbanganController.CheckJadwalHariIni)
		penimbangan.GET("/check-active/:bank_id", middleware.RequireRole(models.PetugasBSI, models.PetugasBSU, models.PetugasBSM, models.RoleNasabah), penimbanganController.CheckJadwalActive)
		penimbangan.GET("/get-sesi-aktif/:penimbangan_id", petugasAll, penimbanganController.GetPenimbanganSesiAktif)
		penimbangan.POST("/add/:bank_id/:admin_id", petugasAll, bankParam, penimbanganController.AddNewPenimbangan)
		penimbangan.PATCH("/update/:penimbangan_id/:admin_id", petugasAll, penimbanganController.UpdatePenimbangan)
		penimbangan.GET("/get/:bank_id", allStaff, bankParamOrBSI, penimbanganController.GetPenimbangan)
		penimbangan.GET("/list-setoran/:penimbangan_id", allStaff, penimbanganController.ListSetoranPenimbangan)
	}

	// ─── Dashboard ──────────────────────────────────────────────────────────────
	dashboard := r.Group("/dashboard", requireAuth)
	{
		dashboardController := controllers.NewDashboardController(db)
		dashboard.GET("/petugas/:bank_id", petugasAll, bankParam, dashboardController.GetDashboardPetugas)
		dashboard.GET("/saldo-bank/:bank_id", allStaff, bankParamOrBSI, dashboardController.GetSaldoBank)
		dashboard.GET("/saldo-nasabah/:nasabah_id", middleware.RequireRole(models.RoleNasabah, models.AdminBSI, models.AdminBSU, models.AdminBSM), sameNasabah, dashboardController.GetSaldoNasabah)
		dashboard.GET("/mutasi-nasabah/:nasabah_id", nasabahRole, sameNasabah, dashboardController.MutasiSaldoNasabah)
		dashboard.GET("/mutasi-bank/:bank_id", allAdmin, bankParamOrBSI, dashboardController.MutasiSaldoBank)
		dashboard.POST("/catat-manual/:bank_id", allAdmin, bankParam, dashboardController.CatatManualMutasiBank)
		dashboard.GET("/total-saldo-all-nasabah/:bank_id", allAdmin, bankParamOrBSI, dashboardController.TotalSaldoAllNasabah)
		dashboard.GET("/daftar-saldo-all-nasabah/:bank_id", allAdmin, bankParamOrBSI, dashboardController.ListSaldoAllNasabah)
	}

	// ─── Setoran ────────────────────────────────────────────────────────────────
	setoran := r.Group("/setoran", requireAuth)
	{
		setoranRepo := repositories.NewSetoranRepo(db)
		setoranSvc := services.NewSetoranService(db, setoranRepo, notifSvc)
		setoranController := controllers.NewSetoranController(setoranSvc, cfStorage)
		setoran.POST("/verifikasi", petugasAll, setoranController.VerifikasiSetoranNasabah)
		setoran.POST("/preview/:penimbangan_id/:nasabah_id", petugasAll, setoranController.PreviewSetoranNasabah)
		setoran.POST("/input/:penimbangan_id/:nasabah_id/:admin_id", petugasAll, setoranController.InputSetoranNasabah)
		setoran.GET("/detail/:setoran_id", setoranController.DetailSetoranNasabah)
		setoran.GET("/riwayat/:nasabah_id", sameNasabah, setoranController.ListRiwayatSetoranNasabah)
	}

	// ─── Pengangkutan ───────────────────────────────────────────────────────────
	pengangkutan := r.Group("/pengangkutan", requireAuth)
	{
		pengangkutanRepo := repositories.NewPengangkutanRepo(db)
		pengangkutanSvc := services.NewPengangkutanService(db, pengangkutanRepo, notifSvc)
		pengangkutanController := controllers.NewPengangkutanController(pengangkutanSvc, cfStorage)
		pengangkutan.GET("/check/:bsi_id", petugasBSI, bsiParam, pengangkutanController.CheckJadwalPengangkutan)
		pengangkutan.GET("/check-sesi-active/:bsu_id", petugasAll, pengangkutanController.CheckSesiActivePengangkutan)
		pengangkutan.GET("/detail-sesi-active/:pengangkutan_id", staffBSIBSU, pengangkutanController.DetailSesiActivePengangkutan)
		pengangkutan.GET("/get-all-active/:bsi_id/:admin_id", petugasBSI, bsiParam, pengangkutanController.GetAllActivePengangkutan)
		pengangkutan.POST("/start", petugasBSI, pengangkutanController.StartSesiPengangkutan)
		pengangkutan.GET("/get-all/:bank_id", staffBSIBSU, bankParamOrBSI, pengangkutanController.GetAllPengangkutan)
		pengangkutan.PATCH("/update/:pengangkutan_id/:admin_bsi_id", petugasBSI, pengangkutanController.UpdatePengangkutanByBSI)
		pengangkutan.POST("/request/:bsu_id/:admin_bsu_id", petugasBSU, bsuParam, pengangkutanController.RequestPengangkutanByBSU)
		pengangkutan.GET("/list-sampah/:bsi_id/:bsu_id", middleware.RequireRole(models.PetugasBSI, models.PetugasBSU), bsiParam, pengangkutanController.ListSampahPengangkutan)
		pengangkutan.POST("/preview/:pengangkutan_id", petugasBSI, pengangkutanController.PreviewPengangkutanSampah)
		pengangkutan.POST("/input", petugasBSI, pengangkutanController.InputSampahPengangkutan)
		pengangkutan.GET("/detail-sampah/:pengangkutan_id", staffBSIBSU, pengangkutanController.DetailSampahPengangkutan)
	}

	// ─── Reward ─────────────────────────────────────────────────────────────────
	reward := r.Group("/reward", requireAuth)
	{
		rewardController := controllers.NewRewardController(db, cfStorage, mailer)
		reward.GET("/get-all", rewardController.GetRewards)

		rewardAdmin := reward.Group("", superadminRole)
		rewardAdmin.POST("/add", rewardController.AddReward)
		rewardAdmin.PATCH("/update/:reward_id", rewardController.UpdateReward)
		rewardAdmin.DELETE("/delete/:reward_id", rewardController.DeleteReward)
	}

	nilaiReward := r.Group("/nilai-reward", requireAuth)
	{
		rewardController := controllers.NewRewardController(db, cfStorage, mailer)
		nilaiReward.GET("/get/:bank_id", rewardController.GetNilaiRewardBank)
		nilaiReward.GET("/detail/:nilai_reward_id", middleware.RequireRole(models.AdminBSI, models.AdminBSM, models.AdminBSU, models.PetugasBSU), rewardController.GetDetailNilaiRewardBank)
		nilaiReward.POST("/add/:bank_id", adminBSIBSM, bankParam, rewardController.AddNewNilaiRewardBank)
		nilaiReward.PATCH("/edit/:bank_id/:reward_id", adminBSIBSM, bankParam, rewardController.UpdateNilaiRewardBank)
		nilaiReward.GET("/history/:nilai_reward_id", adminBSIBSM, rewardController.GetHistoryNilaiRewardBank)
		nilaiReward.DELETE("/delete/:nilai_reward_id", adminBSIBSM, rewardController.DeleteNilaiRewardBank)
	}

	// ─── Penjualan ──────────────────────────────────────────────────────────────
	penjualan := r.Group("/penjualan", requireAuth)
	{
		penjualanController := controllers.NewPenjualanController(db, cfStorage)
		penjualan.POST("/preview/:bank_id", petugasBSIBSM, bankParam, penjualanController.PreviewPenjualanEksternal)
		penjualan.POST("/add-eksternal/:bank_id/:admin_id", petugasBSIBSM, bankParam, penjualanController.AddNewPenjualanEksternal)
		penjualan.GET("/riwayat-eksternal/:bank_id", staffBSIBSM, bankParam, penjualanController.GetRiwayatPenjualanEksternal)
		penjualan.GET("/detail-eksternal/:penjualan_id", staffBSIBSM, penjualanController.DetailPenjualanEksternal)
		penjualan.GET("/mitra/:bank_id", staffBSIBSM, bankParam, penjualanController.GetListMitraPenjualan)
	}

	// ─── Bagi Hasil ─────────────────────────────────────────────────────────────
	bagihasil := r.Group("/bagi-hasil", requireAuth)
	{
		bagihasilController := controllers.NewBagiHasilController(db, notifSvc)
		bagihasil.POST("/preview/:penjualan_id/:bank_id", petugasBSIBSM, bankParam, bagihasilController.PreviewHitungBagiHasil)
		bagihasil.POST("/submit/:penjualan_id/:bank_id", petugasBSIBSM, bankParam, bagihasilController.SubmitBagiHasil)
		bagihasil.GET("/detail/:penjualan_id", staffBSIBSM, bagihasilController.GetDetailBagiHasil)
		bagihasil.GET("/list-bh-nasabah/:nasabah_id", sameNasabah, bagihasilController.GetListBagiHasilPerNasabah)
		bagihasil.GET("/detail-bh-nasabah/:penerima_id", bagihasilController.GetDetailBagiHasilNasabah)
		bagihasil.GET("/list-bh-bsu/:bsu_id", staffBSIBSU, bagihasilController.GetListBagiHasilPerBsu)
		bagihasil.GET("/detail-bh-bsu/:penerima_id", staffBSIBSU, bagihasilController.GetDetailBagiHasilBSU)
		bagihasil.GET("/list-bh-bank/:bank_id", staffBSIBSM, bankParam, bagihasilController.GetListBagiHasilBankPusat)
		bagihasil.GET("/detail-bh-bank/:bagi_hasil_id", staffBSIBSM, bagihasilController.GetDetailBagiHasilBankPusat)
	}

	// ─── Distribusi Sisa ────────────────────────────────────────────────────────
	distribusiSisa := r.Group("/distribusi-sisa", requireAuth)
	{
		distribusiSisaController := controllers.NewDistribusiSisa(db, notifSvc)
		distribusiSisa.GET("/konfigurasi/:bank_id", adminBSI, bankParam, distribusiSisaController.GetKonfigurasi)
		distribusiSisa.POST("/konfigurasi/:bank_id", adminBSI, bankParam, distribusiSisaController.AddKonfigurasi)
		distribusiSisa.PATCH("/konfigurasi/:bank_id", adminBSI, bankParam, distribusiSisaController.UpdateKonfigurasi)
		distribusiSisa.GET("/preview/:bagi_hasil_id", petugasBSI, distribusiSisaController.PreviewDistribusiSisa)
		distribusiSisa.POST("/submit/:bagi_hasil_id", petugasBSI, distribusiSisaController.SubmitDistribusiSisa)
		distribusiSisa.GET("/detail/:distribusi_id", staffBSIBSU, distribusiSisaController.GetDetailDistribusiSisa)
		distribusiSisa.GET("/list-bh-bank/:bank_id", staffBSIBSU, bankParamOrBSI, distribusiSisaController.ListBagiHasilBank)
		distribusiSisa.GET("/detail-bh-bank/:penerima_sisa_id", staffBSIBSU, distribusiSisaController.DetailBagiHasilBank)
	}

	// ─── Tabungan Sampah ────────────────────────────────────────────────────────
	tabunganSampah := r.Group("/tabungan-sampah", requireAuth)
	{
		tabunganSampahController := controllers.NewTabunganSampahController(db)
		tabunganSampah.GET("/buku-tabungan/:nasabah_id", nasabahRole, sameNasabah, tabunganSampahController.GetBukuTabunganSampahNasabah)
		tabunganSampah.GET("/buku-tabungan-bsu/:bsu_id", staffBSU, bsuParam, tabunganSampahController.GetBukuTabunganSampahBSU)
	}

	// ─── Penarikan ──────────────────────────────────────────────────────────────
	penarikan := r.Group("/penarikan", requireAuth)
	{
		penarikanRepo := repositories.NewPenarikanRepo(db)
		penarikanSvc := services.NewPenarikanService(db, penarikanRepo, notifSvc)
		penarikanController := controllers.NewPenarikanController(penarikanSvc, cfStorage)
		penarikan.POST("/ajukan/:nasabah_id", nasabahRole, sameNasabah, penarikanController.AjukanPenarikan)
		penarikan.POST("/preview/:nasabah_id", nasabahRole, sameNasabah, penarikanController.PreviewAjukanPenarikan)
		penarikan.POST("/konfirmasi/:penarikan_id", petugasAll, penarikanController.KonfirmasiPenarikanNasabah)
		penarikan.GET("/list-bank/:bank_id", allStaff, bankParamOrBSI, penarikanController.ListPenarikanNasabahByBank)
		penarikan.GET("/list/:nasabah_id", sameNasabah, penarikanController.ListPenarikanNasabah)
		penarikan.GET("/detail/:penarikan_id", penarikanController.DetailPenarikanNasabah)
		penarikan.PATCH("/batal/:penarikan_id", nasabahRole, penarikanController.BatalPenarikan)
	}

	// ─── Notifikasi ─────────────────────────────────────────────────────────────
	notifikasi := r.Group("/notifikasi", requireAuth)
	{
		notifikasiController := controllers.NewNotifikasiController(notifSvc)
		notifikasi.GET("/list", notifikasiController.GetList)
		notifikasi.PATCH("/read/:notifikasi_id", notifikasiController.MarkAsRead)
		notifikasi.PATCH("/read-all", notifikasiController.MarkAllAsRead)
		notifikasi.GET("/unread-count", notifikasiController.UnreadCount)
	}

	// ─── Info Mobile ────────────────────────────────────────────────────────────
	infoMobile := r.Group("/info-mobile", requireAuth, nasabahRole, sameNasabah)
	{
		infoMobileController := controllers.NewInfoMobileController(db)
		infoMobile.GET("/jadwal-penimbangan/:nasabah_id", infoMobileController.JadwalPenimbanganForNasabah)
		infoMobile.GET("/reward-overview/:nasabah_id", infoMobileController.RewardOverviewNasabah)
	}

	// ─── Laporan ────────────────────────────────────────────────────────────────
	laporan := r.Group("/laporan", requireAuth)
	{
		laporanController := controllers.NewLaporanController(db)
		laporan.GET("/penimbangan/:penimbangan_id", allAdmin, laporanController.DownloadLaporanPenimbangan)
		laporan.GET("/penimbangan/rekap/:bank_id", allAdmin, bankParam, laporanController.DownloadLaporanRekapPenimbangan)
		laporan.GET("/penjualan/:penjualan_id", adminBSIBSM, laporanController.DownloadLaporanPenjualan)
		laporan.GET("/penjualan/rekap/:bank_id", adminBSIBSM, bankParam, laporanController.DownloadLaporanRekapPenjualan)
		laporan.GET("/nasabah/:bank_id", allAdmin, bankParamOrBSI, laporanController.DownloadLaporanNasabah)
		laporan.GET("/pengangkutan/:pengangkutan_id", middleware.RequireRole(models.AdminBSI, models.AdminBSU), laporanController.DownloadLaporanPengangkutan)
		laporan.GET("/bagi-hasil/:bagi_hasil_id", adminBSIBSM, laporanController.DownloadLaporanBagiHasil)
		laporan.GET("/bagi-hasil/rekap/:bank_id", middleware.RequireRole(models.AdminBSM), bankParam, laporanController.DownloadLaporanRekapBagiHasil)
		laporan.GET("/penarikan/rekap/:bank_id", allAdmin, bankParam, laporanController.DownloadLaporanRekapPenarikan)
		laporan.GET("/bank-sampah", superadminRole, laporanController.DownloadLaporanBankSampah)
		laporan.GET("/katalog-sampah/:bank_id", allAdmin, bankParam, laporanController.DownloadLaporanKatalogSampah)
		laporan.GET("/katalog-sembako/:bank_id", allAdmin, bankParam, laporanController.DownloadLaporanKatalogSembako)
	}
}
