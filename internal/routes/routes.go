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

	"github.com/gin-gonic/gin"
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
		auth.POST("/login", authController.Login)
		auth.POST("/refresh", authController.RefreshToken)
		auth.POST("/logout", authController.Logout)
		auth.POST("/cek-user-mobile", authController.CekUserMobile)

		auth.POST("/aktivasi-akun", authController.AktivasiAkun)
		auth.POST("/deactivate-akun", authController.DeactivateAkun)
		auth.POST("/generate-reactivate-akun", authController.GenerateReactivateAkun)
		auth.POST("/reactivate-akun", authController.ReactivateAkun)

		auth.POST("/forget-password/send-email", authController.SendEmailForgetPassword)
		auth.POST("/forget-password/verifikasi-otp", authController.VerifikasiOTP)
		auth.POST("/forget-password/reset-password", authController.ResetPassword)

		authProtected := auth.Group("", middleware.RequireAuth(db))
		authProtected.GET("/me", authController.Me)
		authProtected.POST("/change-password", authController.ChangePassword)
		authProtected.POST("/switch-role", authController.SwitchRole)
	}

	// ─── Middleware shortcuts ────────────────────────────────────────────────────
	requireAuth := middleware.RequireAuth(db)
	superadminOnly := middleware.RequireAuth(db)
	superadminRole := middleware.RequireRole(models.SuperAdmin)

	// ─── Bank ───────────────────────────────────────────────────────────────────
	bank := r.Group("/bank", requireAuth)
	{
		bankController := controllers.NewBankController(db, cfStorage, mailer)
		bank.PATCH("/aktivasi/:bank_id", bankController.AktivasiBank)
		bank.GET("/get-nasabah/:bank_id", bankController.GetNasabahByBankID)
		bank.GET("/get-all", bankController.GetAllBankSampah)
		bank.PATCH("/edit-profil/:bank_id", bankController.EditProfilBankSampah)
	}

	bsi := r.Group("/bsi", requireAuth)
	{
		bsiController := controllers.NewBSIController(db, cfStorage, mailer)
		bsi.POST("/add-bsi", bsiController.AddNewBSI)
		bsi.GET("/get-bsi", bsiController.GetBSI)
		bsi.GET("/get-unit/:bank_id", bsiController.GetUnitBSI)
		bsi.POST("/add-unit/:bank_id", bsiController.AddNewUnit)
	}
	bsm := r.Group("/bsm", superadminOnly, superadminRole)
	{
		bsmController := controllers.NewBSMController(db, cfStorage, mailer)
		bsm.POST("/add-bsm", bsmController.AddNewBSM)
		bsm.GET("/get-bsm", bsmController.GetBSM)
	}
	bsu := r.Group("/bsu", superadminOnly, superadminRole)
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
		nasabah.POST("/add-nasabah", nasabahController.AddNewNasabah)
		nasabah.POST("/add-nasabah-from-old-user", nasabahController.AddNewNasabahOldUser)
		nasabah.GET("/get-afiliasi", nasabahController.GetAfiliasi)
		nasabah.GET("/get-nasabah", nasabahController.GetNasabah)
		nasabah.GET("/:bank_id", nasabahController.NasabahBankSampah)
	}

	// ─── Users ──────────────────────────────────────────────────────────────────
	superadminMgmt := r.Group("/superadmin", superadminOnly, superadminRole)
	{
		userController := controllers.NewUserController(db, mailer, cfStorage)
		superadminMgmt.GET("/list", userController.GetListSuperadmin)
		superadminMgmt.POST("/add", userController.AddSuperadmin)
		superadminMgmt.PATCH("/nonaktif/:admin_id", userController.NonaktifkanSuperadmin)
	}

	user := r.Group("/users", requireAuth)
	{
		userController := controllers.NewUserController(db, mailer, cfStorage)
		user.POST("/add-user", userController.AddUser)
		user.POST("/update-profil/:user_id", userController.UpdateProfilUser)
		user.GET("/get-nonadmin-user", userController.GetNonAdminUser)
		user.GET("/get-nonnasabah-user", userController.GetNonNasabahUser)
		user.GET("/active-admin/:admin_id", userController.ActiveAdmin)
		user.GET("/active-petugas/:admin_id", userController.ActivePetugas)
		user.GET("/active-user/:user_id", userController.ActiveUser)
		user.PATCH("/update-fcm-token", userController.UpdateFCMToken)
		user.GET("/log/:user_id", userController.LogAkun)
		user.GET("/get-all", userController.GetAllUsers)
		user.GET("/detail-user/:user_id", userController.GetDetailUser)
		user.DELETE("/delete-user/:user_id", userController.DeleteUser)
	}

	// ─── Statistik ──────────────────────────────────────────────────────────────
	statistik := r.Group("/statistik", superadminOnly, superadminRole)
	{
		statistikController := controllers.NewStatistikController(db)
		statistik.GET("/bank-sampah", statistikController.GetBankSampahStatistik)
		statistik.GET("/superadmin/ringkasan", statistikController.GetRingkasanSuperadmin)
		statistik.GET("/superadmin/tren-penjualan", statistikController.GetTrenPenjualan)
		statistik.GET("/superadmin/ranking-bank", statistikController.GetRankingBank)
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
		profil.GET("/bank-sampah/:bank_id", profilController.GetProfilBankSampah)
		profil.GET("/bank-sampah/:bank_id/history", profilController.GetHistoryAkunBank)
		profil.GET("/nasabah/:nasabah_id", profilController.GetProfilNasabah)
		profil.PATCH("/nasabah/aktivasi/:nasabah_id", profilController.AktivasiNasabah)
		profil.DELETE("/bank-sampah/:bank_id", profilController.DeleteBankSampah)
		profil.DELETE("/nasabah/:nasabah_id", profilController.DeleteNasabah)
		profil.GET("/:user_id", profilController.GetProfilUser)
		profil.GET("/detail-nasabah/:nasabah_id", profilController.GetDetailNasabah)
		profil.GET("/detail-petugas/:petugas_id", profilController.GetDetailPetugas)
		profil.GET("/detail-bank/:bank_id", profilController.DetailBankSampah)
	}

	// ─── Admin ──────────────────────────────────────────────────────────────────
	admin := r.Group("/admin", requireAuth)
	{
		adminController := controllers.NewAdminController(db, mailer)
		admin.GET("/get-admin/:bank_id", adminController.GetAdminBankSampah)
		admin.POST("/add-admin-bank-sampah", adminController.AddAdminBankSampah)
		admin.DELETE("/delete-staff/:admin_id", adminController.DeleteStaffBankSampah)
	}

	// ─── Lokasi ─────────────────────────────────────────────────────────────────
	lokasiController := controllers.NewLokasiController(db)

	// Read-only: semua user yang sudah login
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

	// Mutasi: superadmin saja
	lokasiAdmin := r.Group("/lokasi", superadminOnly, superadminRole)
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
		katalog.POST("/add-sampah/:bank_id", katalogController.AddKatalog)
		katalog.PATCH("/edit-sampah/:sampah_id", katalogController.EditKatalog)
		katalog.DELETE("/delete-sampah/:sampah_id", katalogController.DeleteKatalogSampah)
		katalog.GET("/get-sampah/:bank_id", katalogController.GetKatalogSampahBank)
		katalog.GET("/get-detail/:sampah_id", katalogController.GetDetailSampah)
		katalog.POST("/add-kategori", katalogController.AddNewKategori)
		katalog.GET("/get-kategori", katalogController.GetKategori)
		katalog.PATCH("/update-kategori/:kategori_id", katalogController.UpdateKategori)
		katalog.DELETE("/delete-kategori/:kategori_id", katalogController.DeleteKategori)
	}

	// ─── Sembako ────────────────────────────────────────────────────────────────
	sembako := r.Group("/sembako", requireAuth)
	{
		sembakoController := controllers.NewSembakoController(db, cfStorage, notifSvc)
		sembako.POST("/add-sembako/:bank_id", sembakoController.AddNewSembako)
		sembako.GET("/get-sembako/:bank_id", sembakoController.GetSembakoBank)
		sembako.DELETE("/delete-sembako/:sembako_id", sembakoController.DeleteSembako)
		sembako.PATCH("/edit-sembako/:sembako_id", sembakoController.EditSembako)
		sembako.GET("/detail-sembako-bsu/:sembako_id", sembakoController.GetDetailSembakoBSU)
		sembako.POST("/add-distribusi-bsu/:bsi_id/:bsu_id", sembakoController.AddNewDistribusiSembakoBSU)
		sembako.POST("/preview-distribusi-bsu/:bsi_id/:bsu_id", sembakoController.PreviewDistribusiSembakoBSU)
	}

	// ─── Konten ─────────────────────────────────────────────────────────────────
	konten := r.Group("/konten", requireAuth)
	{
		kontenController := controllers.NewKontenController(db, cfStorage)
		konten.POST("/add-konten/:admin_id", kontenController.AddNewKonten)
		konten.GET("/all-konten", kontenController.GetAllKontenSuperadmin)
		konten.GET("/all-konten/:bank_id", kontenController.GetAllKonten)
		konten.GET("/get-konten/:konten_id", kontenController.GetKontenByID)
		konten.DELETE("/delete-konten/:konten_id", kontenController.DeleteKonten)
		konten.PATCH("/edit-konten/:konten_id", kontenController.EditKonten)
	}

	// ─── Jadwal ─────────────────────────────────────────────────────────────────
	jadwal := r.Group("/jadwal", requireAuth)
	{
		jadwalController := controllers.NewJadwalController(db)
		jadwal.GET("/get-all", jadwalController.GetAllJadwal)
		jadwal.GET("/get-jadwal/:bank_id", jadwalController.GetJadwalBank)
		jadwal.POST("/add-jadwal/:bank_id", jadwalController.AddNewJadwal)
		jadwal.DELETE("/delete-jadwal/:jadwal_id", jadwalController.DeleteJadwal)
		jadwal.PATCH("/update-jadwal/:jadwal_id", jadwalController.UpdateJadwal)
	}

	// ─── Penimbangan ────────────────────────────────────────────────────────────
	penimbangan := r.Group("/penimbangan", requireAuth)
	{
		penimbanganController := controllers.NewPenimbanganController(db)
		penimbangan.GET("/check/:bank_id", penimbanganController.CheckJadwalHariIni)
		penimbangan.GET("/check-active/:bank_id", penimbanganController.CheckJadwalActive)
		penimbangan.GET("/get-sesi-aktif/:penimbangan_id", penimbanganController.GetPenimbanganSesiAktif)
		penimbangan.POST("/add/:bank_id/:admin_id", penimbanganController.AddNewPenimbangan)
		penimbangan.PATCH("/update/:penimbangan_id/:admin_id", penimbanganController.UpdatePenimbangan)
		penimbangan.GET("/get/:bank_id", penimbanganController.GetPenimbangan)
		penimbangan.GET("/list-setoran/:penimbangan_id", penimbanganController.ListSetoranPenimbangan)
	}

	// ─── Dashboard ──────────────────────────────────────────────────────────────
	dashboard := r.Group("/dashboard", requireAuth)
	{
		dashboardController := controllers.NewDashboardController(db)
		dashboard.GET("/petugas/:bank_id", dashboardController.GetDashboardPetugas)
		dashboard.GET("/saldo-bank/:bank_id", dashboardController.GetSaldoBank)
		dashboard.GET("/saldo-nasabah/:nasabah_id", dashboardController.GetSaldoNasabah)
		dashboard.GET("/mutasi-nasabah/:nasabah_id", dashboardController.MutasiSaldoNasabah)
		dashboard.GET("/mutasi-bank/:bank_id", dashboardController.MutasiSaldoBank)
		dashboard.POST("/catat-manual/:bank_id", dashboardController.CatatManualMutasiBank)
	}

	// ─── Setoran ────────────────────────────────────────────────────────────────
	setoran := r.Group("/setoran", requireAuth)
	{
		setoranController := controllers.NewSetoranController(db, cfStorage, notifSvc)
		setoran.GET("/verifikasi/:penimbangan_id/:nasabah_id/:admin_id", setoranController.VerifikasiSetoranNasabah)
		setoran.POST("/preview/:penimbangan_id/:nasabah_id", setoranController.PreviewSetoranNasabah)
		setoran.POST("/input/:penimbangan_id/:nasabah_id/:admin_id", setoranController.InputSetoranNasabah)
		setoran.GET("/detail-setoran-nasabah/:setoran_id", setoranController.DetailSetoranNasabah)
		setoran.GET("/list-setoran-nasabah/:nasabah_id", setoranController.ListRiwayatSetoranNasabah)
	}

	// ─── Pengangkutan ───────────────────────────────────────────────────────────
	pengangkutan := r.Group("/pengangkutan", requireAuth)
	{
		pengangkutanController := controllers.NewPengangkutanController(db, cfStorage, notifSvc)
		pengangkutan.GET("/check/:bsi_id/:bsu_id", pengangkutanController.CheckJadwalPengangkutan)
		pengangkutan.GET("/check-sesi-active/:bsu_id", pengangkutanController.CheckSesiActivePengangkutan)
		pengangkutan.GET("/detail-sesi-active/:pengangkutan_id", pengangkutanController.DetailSesiActivePengangkutan)
		pengangkutan.GET("/get-all-active/:bsi_id/:admin_id", pengangkutanController.GetAllActivePengangkutan)
		pengangkutan.POST("/start", pengangkutanController.StartSesiPengangkutan)
		pengangkutan.GET("/get-all/:bank_id", pengangkutanController.GetAllPengangkutan)
		pengangkutan.PATCH("/update/:pengangkutan_id/:admin_bsi_id", pengangkutanController.UpdatePengangkutanByBSI)
		pengangkutan.POST("/request/:bsu_id/:admin_bsu_id", pengangkutanController.RequestPengangkutanByBSU)
		pengangkutan.GET("/list-sampah/:bsi_id/:bsu_id", pengangkutanController.ListSampahPengangkutan)
		pengangkutan.POST("/preview/:pengangkutan_id", pengangkutanController.PreviewPengangkutanSampah)
		pengangkutan.POST("/input/:pengangkutan_id/:admin_bsi_id/:admin_bsu_id", pengangkutanController.InputSampahPengangkutan)
		pengangkutan.GET("/detail-sampah/:pengangkutan_id", pengangkutanController.DetailSampahPengangkutan)
	}

	// ─── Reward ─────────────────────────────────────────────────────────────────
	reward := r.Group("/reward", requireAuth)
	{
		rewardController := controllers.NewRewardController(db, cfStorage, mailer)
		reward.GET("/get-all", rewardController.GetRewards)

		rewardAdmin := reward.Group("", superadminOnly, superadminRole)
		rewardAdmin.POST("/add", rewardController.AddReward)
		rewardAdmin.PATCH("/update/:reward_id", rewardController.UpdateReward)
		rewardAdmin.DELETE("/delete/:reward_id", rewardController.DeleteReward)
	}

	nilaiReward := r.Group("/nilai-reward", requireAuth)
	{
		rewardController := controllers.NewRewardController(db, cfStorage, mailer)
		nilaiReward.GET("/get/:bank_id", rewardController.GetNilaiRewardBank)
		nilaiReward.GET("/detail/:nilai_reward_id", rewardController.GetDetailNilaiRewardBank)
		nilaiReward.POST("/add/:bank_id", rewardController.AddNewNilaiRewardBank)
		nilaiReward.PATCH("/edit/:bank_id/:reward_id", rewardController.UpdateNilaiRewardBank)
		nilaiReward.GET("/history/:nilai_reward_id", rewardController.GetHistoryNilaiRewardBank)
		nilaiReward.DELETE("/delete/:nilai_reward_id", rewardController.DeleteNilaiRewardBank)
	}

	// ─── Penjualan ──────────────────────────────────────────────────────────────
	penjualan := r.Group("/penjualan", requireAuth)
	{
		penjualanController := controllers.NewPenjualanController(db, cfStorage)
		penjualan.POST("/preview/:bank_id", penjualanController.PreviewPenjualanEksternal)
		penjualan.POST("/add-eksternal/:bank_id/:admin_id", penjualanController.AddNewPenjualanEksternal)
		penjualan.GET("/riwayat-eksternal/:bank_id", penjualanController.GetRiwayatPenjualanEksternal)
		penjualan.GET("/detail-eksternal/:penjualan_id", penjualanController.DetailPenjualanEksternal)
		penjualan.GET("/mitra/:bank_id", penjualanController.GetListMitraPenjualan)
	}

	// ─── Bagi Hasil ─────────────────────────────────────────────────────────────
	bagihasil := r.Group("/bagi-hasil", requireAuth)
	{
		bagihasilController := controllers.NewBagiHasilController(db, notifSvc)
		bagihasil.POST("/preview/:penjualan_id/:bank_id", bagihasilController.PreviewHitungBagiHasil)
		bagihasil.POST("/submit/:penjualan_id/:bank_id", bagihasilController.SubmitBagiHasil)
		bagihasil.GET("/detail/:penjualan_id", bagihasilController.GetDetailBagiHasil)
		bagihasil.GET("/list-bh-nasabah/:nasabah_id", bagihasilController.GetListBagiHasilPerNasabah)
		bagihasil.GET("/detail-bh-nasabah/:penerima_id", bagihasilController.GetDetailBagiHasilNasabah)
		bagihasil.GET("/list-bh-bsu/:bsu_id", bagihasilController.GetListBagiHasilPerBsu)
		bagihasil.GET("/detail-bh-bsu/:penerima_id", bagihasilController.GetDetailBagiHasilBSU)
		bagihasil.GET("/list-bh-bank/:bank_id", bagihasilController.GetListBagiHasilBankPusat)
		bagihasil.GET("/detail-bh-bank/:bagi_hasil_id", bagihasilController.GetDetailBagiHasilBankPusat)
	}

	// ─── Distribusi Sisa ────────────────────────────────────────────────────────
	distribusiSisa := r.Group("/distribusi-sisa", requireAuth)
	{
		distribusiSisaController := controllers.NewDistribusiSisa(db, notifSvc)
		distribusiSisa.GET("/konfigurasi/:bank_id", distribusiSisaController.GetKonfigurasi)
		distribusiSisa.POST("/konfigurasi/:bank_id", distribusiSisaController.AddKonfigurasi)
		distribusiSisa.PATCH("/konfigurasi/:bank_id", distribusiSisaController.UpdateKonfigurasi)
		distribusiSisa.GET("/preview/:bagi_hasil_id", distribusiSisaController.PreviewDistribusiSisa)
		distribusiSisa.POST("/submit/:bagi_hasil_id", distribusiSisaController.SubmitDistribusiSisa)
		distribusiSisa.GET("/detail/:distribusi_id", distribusiSisaController.GetDetailDistribusiSisa)
		distribusiSisa.GET("/list-bh-bank/:bank_id", distribusiSisaController.ListBagiHasilBank)
		distribusiSisa.GET("/detail-bh-bank/:penerima_sisa_id", distribusiSisaController.DetailBagiHasilBank)
	}

	// ─── Tabungan Sampah ────────────────────────────────────────────────────────
	tabunganSampah := r.Group("/tabungan-sampah", requireAuth)
	{
		tabunganSampahController := controllers.NewTabunganSampahController(db)
		tabunganSampah.GET("/buku-tabungan/:nasabah_id", tabunganSampahController.GetBukuTabunganSampahNasabah)
		tabunganSampah.GET("/buku-tabungan-bsu/:bsu_id", tabunganSampahController.GetBukuTabunganSampahBSU)
	}

	// ─── Penarikan ──────────────────────────────────────────────────────────────
	penarikan := r.Group("/penarikan", requireAuth)
	{
		penarikanController := controllers.NewPenarikanController(db, cfStorage, notifSvc)
		penarikan.POST("/ajukan/:nasabah_id", penarikanController.AjukanPenarikan)
		penarikan.POST("/preview/:nasabah_id", penarikanController.PreviewAjukanPenarikan)
		penarikan.POST("/konfirmasi/:penarikan_id", penarikanController.KonfirmasiPenarikanNasabah)
		penarikan.GET("/list-bank/:bank_id", penarikanController.ListPenarikanNasabahByBank)
		penarikan.GET("/list/:nasabah_id", penarikanController.ListPenarikanNasabah)
		penarikan.GET("/detail/:penarikan_id", penarikanController.DetailPenarikanNasabah)
		penarikan.PATCH("/batal/:penarikan_id", penarikanController.BatalPenarikan)
	}

	// ─── Notifikasi ─────────────────────────────────────────────────────────────
	notifikasi := r.Group("/notifikasi", requireAuth)
	{
		notifikasiController := controllers.NewNotifikasiController(notifSvc)
		notifikasi.GET("/list/:user_id", notifikasiController.GetList)
		notifikasi.PATCH("/read/:notifikasi_id", notifikasiController.MarkAsRead)
		notifikasi.PATCH("/read-all/:user_id", notifikasiController.MarkAllAsRead)
		notifikasi.GET("/unread-count/:user_id", notifikasiController.UnreadCount)
	}

	// ─── Info Mobile ────────────────────────────────────────────────────────────
	infoMobile := r.Group("/info-mobile", requireAuth)
	{
		infoMobileController := controllers.NewInfoMobileController(db)
		infoMobile.GET("/jadwal-penimbangan/:nasabah_id", infoMobileController.JadwalPenimbanganForNasabah)
		infoMobile.GET("/reward-overview/:nasabah_id", infoMobileController.RewardOverviewNasabah)
	}

	// ─── Laporan ────────────────────────────────────────────────────────────────
	laporan := r.Group("/laporan", requireAuth)
	{
		laporanController := controllers.NewLaporanController(db)
		laporan.GET("/penimbangan/:penimbangan_id", laporanController.DownloadLaporanPenimbangan)
		laporan.GET("/penjualan/:penjualan_id", laporanController.DownloadLaporanPenjualan)
		laporan.GET("/nasabah/:bank_id", laporanController.DownloadLaporanNasabah)
		laporan.GET("/pengangkutan/:pengangkutan_id", laporanController.DownloadLaporanPengangkutan)
		laporan.GET("/bagi-hasil/:bagi_hasil_id", laporanController.DownloadLaporanBagiHasil)
		laporan.GET("/bank-sampah", laporanController.DownloadLaporanBankSampah)
	}
}
