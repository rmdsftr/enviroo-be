package routes

import (
	"enviroo-be/internal/controllers"
	"enviroo-be/internal/middleware"
	"enviroo-be/internal/models"
	"enviroo-be/pkg/storage"
	"enviroo-be/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB, cfStorage *storage.CloudflareStorage, mailer *utils.Mailer) {
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
		auth.POST("/add-superadmin", authController.AddSuperadmin)
		auth.POST("/login", authController.Login)
		auth.POST("/refresh", authController.RefreshToken)
		auth.POST("/logout", authController.Logout)
		auth.POST("/cek-user-mobile", authController.CekUserMobile)

		//aktivasi akun
		auth.POST("/aktivasi-akun", authController.AktivasiAkun)
		auth.POST("/deactivate-akun", authController.DeactivateAkun)
		auth.POST("/generate-reactivate-akun", authController.GenerateReactivateAkun)
		auth.POST("/reactivate-akun", authController.ReactivateAkun)

		// Protected: hanya user yang sudah login
		authProtected := auth.Group("", middleware.RequireAuth())
		authProtected.GET("/me", authController.Me)
	}

	// ─── Protected routes ────────────────────────────────────────────────────────
	superadminOnly := middleware.RequireAuth()
	superadminRole := middleware.RequireRole(models.SuperAdmin)

	bank := r.Group("/bank")
	{
		bankController := controllers.NewBankController(db, cfStorage, mailer)
		bank.PATCH("/aktivasi/:bank_id", bankController.AktivasiBank)
		bank.GET("/get-nasabah/:bank_id", bankController.GetNasabahByBankID)
	}

	bsi := r.Group("/bsi")
	{
		bsiController := controllers.NewBSIController(db, cfStorage, mailer)
		bsi.POST("/add-bsi", bsiController.AddNewBSI)
		bsi.GET("/get-bsi", bsiController.GetBSI)
		bsi.GET("/get-unit/:bank_id", bsiController.GetUnitBSI)
		bsi.POST("/add-unit/:bank_id", bsiController.AddNewUnit)
		bsi.GET("/get-nasabah/:bank_id", bsiController.GetNasabahBSI)
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
	nasabah := r.Group("/nasabah")
	{
		nasabahController := controllers.NewNasabahController(db, cfStorage, mailer)
		nasabah.POST("/add-nasabah", nasabahController.AddNewNasabah)
		nasabah.POST("/add-nasabah-from-old-user", nasabahController.AddNewNasabahOldUser)
		nasabah.GET("/get-afiliasi", nasabahController.GetAfiliasi)
		nasabah.GET("/get-nasabah", nasabahController.GetNasabah)
		nasabah.GET("/:bank_id", nasabahController.NasabahBankSampah)
	}
	user := r.Group("/users")
	{
		userController := controllers.NewUserController(db, mailer)
		user.POST("/add-user", userController.AddUser)
		user.GET("/get-nonadmin-user", userController.GetNonAdminUser)
		user.GET("/get-nonnasabah-user", userController.GetNonNasabahUser)
		user.GET("/get-nonadmin-nonnasabah/:bank_id", userController.GetNonAdminBankSampah)
		user.GET("/get-nonnasabah-nonadmin-bsi/:bank_id", userController.GetNonNasabahNonAdminBSI)
		user.GET("/active-admin/:admin_id", userController.ActiveAdmin)
		user.GET("/active-petugas/:admin_id", userController.ActivePetugas)
	}
	statistik := r.Group("/statistik", superadminOnly, superadminRole)
	{
		statistikController := controllers.NewStatistikController(db)
		statistik.GET("/bank-sampah", statistikController.GetBankSampahStatistik)
	}
	profil := r.Group("/profil")
	{
		profilController := controllers.NewProfilController(db, cfStorage)
		profil.GET("/bank-sampah/:bank_id", profilController.GetProfilBankSampah)
		profil.GET("/bank-sampah/:bank_id/history", profilController.GetHistoryAkunBank)
		profil.GET("/nasabah/:nasabah_id", profilController.GetProfilNasabah)
		profil.PATCH("/bank-sampah/aktivasi/:bank_id", profilController.AktivasiBankSampah)
		profil.PATCH("/nasabah/aktivasi/:nasabah_id", profilController.AktivasiNasabah)
		profil.DELETE("/bank-sampah/:bank_id", profilController.DeleteBankSampah)
		profil.DELETE("/nasabah/:nasabah_id", profilController.DeleteNasabah)
		profil.GET("/:user_id", profilController.GetProfilUser)
	}
	admin := r.Group("/admin")
	{
		adminController := controllers.NewAdminController(db, mailer)
		admin.GET("/get-admin/:bank_id", adminController.GetAdminBankSampah)
		admin.POST("/add-admin-bank-sampah", adminController.AddAdminBankSampah)
		admin.DELETE("/delete-staff/:admin_id", adminController.DeleteStaffBankSampah)
	}
	lokasi := r.Group("/lokasi", superadminOnly, superadminRole)
	{
		lokasiController := controllers.NewLokasiController(db)
		lokasi.GET("/bank-sampah", lokasiController.GetLokasiBankSampah)
	}

	katalog := r.Group("/katalog")
	{
		katalogController := controllers.NewKatalogController(db, cfStorage)

		// ─── Shared ───────────────────────────────────────────────────────
		katalog.POST("/add-kategori", katalogController.AddNewKategori)
		katalog.GET("/get-kategori", katalogController.GetKategori)
		katalog.GET("/get-sampah/:bank_id", katalogController.GetKatalogSampahBank)
		katalog.GET("/get-schema-harga/:sampah_id", katalogController.GetSchemaHarga)
		katalog.GET("/get-history/:sampah_id", katalogController.GetHistoryKatalogSampah)
		katalog.DELETE("/delete-sampah/:sampah_id", katalogController.DeleteKatalogSampah)
		katalog.PATCH("/update-harga/:sampah_id", katalogController.UpdateHargaSchema)

		// ─── BSI ──────────────────────────────────────────────────────────
		katalogBSI := katalog.Group("/bsi")
		katalogBSI.POST("/add-sampah/:bank_id", katalogController.AddKatalogBSI)
		katalogBSI.PATCH("/edit-sampah/:sampah_id", katalogController.EditKatalogBSI)

		// ─── BSM ──────────────────────────────────────────────────────────
		katalogBSM := katalog.Group("/bsm")
		katalogBSM.POST("/add-sampah/:bank_id", katalogController.AddKatalogBSM)
		katalogBSM.PATCH("/edit-sampah/:sampah_id", katalogController.EditKatalogBSM)
	}

	sembako := r.Group("/sembako")
	{
		sembakoController := controllers.NewSembakoController(db, cfStorage)
		sembako.POST("/add-sembako/:bank_id", sembakoController.AddNewSembako)
		sembako.GET("/get-sembako/:bank_id", sembakoController.GetSembakoBank)
		sembako.DELETE("/delete-sembako/:sembako_id", sembakoController.DeleteSembako)
		sembako.PATCH("/update-harga-sembako/:sembako_id", sembakoController.UpdateHargaSembako)
		sembako.PATCH("/edit-sembako/:sembako_id", sembakoController.EditSembako)
		sembako.GET("/get-history/:sembako_id", sembakoController.GetHistorySembako)
	}
	konten := r.Group("/konten")
	{
		kontenController := controllers.NewKontenController(db, cfStorage)
		konten.POST("/add-konten/:bank_id/:admin_id", kontenController.AddNewKonten)
		konten.GET("/all-konten/:bank_id", kontenController.GetAllKonten)
		konten.GET("/get-konten/:konten_id", kontenController.GetKontenByID)
		konten.DELETE("/delete-konten/:konten_id", kontenController.DeleteKonten)
		konten.PATCH("/edit-konten/:konten_id", kontenController.EditKonten)
	}

	jadwal := r.Group("/jadwal")
	{
		jadwalController := controllers.NewJadwalController(db)
		jadwal.GET("/get-jadwal/:bank_id", jadwalController.GetJadwalBank)
		jadwal.POST("/add-jadwal/:bank_id", jadwalController.AddNewJadwal)
		jadwal.DELETE("/delete-jadwal/:jadwal_id", jadwalController.DeleteJadwal)
		jadwal.PATCH("/update-jadwal/:jadwal_id", jadwalController.UpdateJadwal)
	}

	penimbangan := r.Group("/penimbangan")
	{
		penimbanganController := controllers.NewPenimbanganController(db)
		penimbangan.GET("/check/:bank_id", penimbanganController.CheckJadwalHariIni)
		penimbangan.POST("/add/:bank_id/:admin_id", penimbanganController.AddNewPenimbangan)
		penimbangan.PATCH("/update/:penimbangan_id/:admin_id", penimbanganController.UpdatePenimbangan)
		penimbangan.GET("/get/:bank_id", penimbanganController.GetPenimbangan)
		penimbangan.GET("/list-setoran/:penimbangan_id", penimbanganController.ListSetoranPenimbangan)
	}

	dashboard := r.Group("/dashboard")
	{
		dashboardController := controllers.NewDashboardController(db)
		dashboard.GET("/petugas/:bank_id", dashboardController.GetDashboardPetugas)
		dashboard.GET("/saldo-bank/:bank_id", dashboardController.GetSaldoBank)
	}

	setoran := r.Group("/setoran")
	{
		setoranController := controllers.NewSetoranController(db, cfStorage)
		setoran.GET("/verifikasi/:penimbangan_id/:nasabah_id/:admin_id", setoranController.VerifikasiSetoranNasabah)
		setoran.POST("/input/:penimbangan_id/:nasabah_id/:admin_id", setoranController.InputSetoranNasabah)
		setoran.GET("/detail-setoran-nasabah/:setoran_id", setoranController.DetailSetoranNasabah)
		setoran.GET("/list-setoran-nasabah/:nasabah_id", setoranController.ListRiwayatSetoranNasabah)
	}

	pengangkutan := r.Group("/pengangkutan")
	{
		pengangkutanController := controllers.NewPengangkutanController(db, cfStorage)
		pengangkutan.GET("/check/:bsi_id/:bsu_id", pengangkutanController.CheckJadwalPengangkutan)
		pengangkutan.POST("/start", pengangkutanController.StartSesiPengangkutan)
		pengangkutan.GET("/get-all/:bank_id", pengangkutanController.GetAllPengangkutan)
		pengangkutan.PATCH("/update/:pengangkutan_id/:admin_bsi_id", pengangkutanController.UpdatePengangkutanByBSI)
		pengangkutan.POST("/request/:bsu_id/:admin_bsu_id", pengangkutanController.RequestPengangkutanByBSU)
		pengangkutan.GET("/list-sampah/:bsi_id/:bsu_id", pengangkutanController.ListSampahPengangkutan)
		pengangkutan.POST("/input/:pengangkutan_id/:admin_bsi_id/:admin_bsu_id", pengangkutanController.InputSampahPengangkutan)
		pengangkutan.GET("/detail-sampah/:pengangkutan_id", pengangkutanController.DetailSampahPengangkutan)
	} 

	reward := r.Group("/reward")
	{
		rewardController := controllers.NewRewardController(db, cfStorage, mailer)

		// Read: dibutuhkan oleh semua admin bank sampah (BSI/BSM/BSU)
		// agar bisa memilih jenis reward saat menetapkan nilai konversi.
		reward.GET("/get-all", middleware.RequireAuth(), rewardController.GetRewards)

		// Write: hanya superadmin yang boleh mengelola master data reward.
		rewardAdmin := reward.Group("", superadminOnly, superadminRole)
		rewardAdmin.POST("/add", rewardController.AddReward)
		rewardAdmin.PATCH("/update/:reward_id", rewardController.UpdateReward)
		rewardAdmin.DELETE("/delete/:reward_id", rewardController.DeleteReward)
	}

	nilaiReward := r.Group("/nilai-reward", middleware.RequireAuth())
	{
		rewardController := controllers.NewRewardController(db, cfStorage, mailer)
		nilaiReward.GET("/get/:bank_id", rewardController.GetNilaiRewardBank)
		nilaiReward.POST("/add/:bank_id", rewardController.AddNewNilaiRewardBank)
		nilaiReward.PATCH("/edit/:nilai_reward_id", rewardController.UpdateNilaiRewardBank)
		nilaiReward.GET("/history/:nilai_reward_id", rewardController.GetHistoryNilaiRewardBank)
		nilaiReward.DELETE("/delete/:nilai_reward_id", rewardController.DeleteNilaiRewardBank)
	}

	penjualan := r.Group("/penjualan")
	{
		penjualanController := controllers.NewPenjualanController(db, cfStorage) 
		penjualan.POST("/add-eksternal/:bank_id/:admin_id", penjualanController.AddNewPenjualanEksternal)
		penjualan.GET("/riwayat-eksternal/:bank_id", penjualanController.GetRiwayatPenjualanEksternal)
		penjualan.GET("/detail-eksternal/:penjualan_id", penjualanController.DetailPenjualanEksternal)
	}

	redeemBSU := r.Group("/redeem-bsu")
	{
		redeemBSUController := controllers.NewRedeemBSUController(db)
		redeemBSU.POST("/request/:bsu_id/:admin_bsu_id", redeemBSUController.RequestRedeemBSU)
		redeemBSU.GET("/verifikasi-manual/:transaksi_id/:admin_bsi_id", redeemBSUController.VerifikasiManualTransaksiReward)
		redeemBSU.POST("/confirm/:transaksi_id/:admin_bsi_id", redeemBSUController.ConfirmRedeemBSU)
		redeemBSU.PATCH("/cancel/:transaksi_id/:admin_bsu_id", redeemBSUController.CancelRequestRedeemBSU)
		redeemBSU.GET("/list-redeem/:bank_id", redeemBSUController.GetListRedeemBSU)
		redeemBSU.GET("/detail/:redeem_id", redeemBSUController.GetDetailRedeemBSU)
	}
}
