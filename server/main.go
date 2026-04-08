package main

// package main → entry point aplikasi Go
// → hanya ada satu package main dalam satu program
// → function main() di package ini yang dieksekusi pertama kali saat program dijalankan

import (
	conectdb "catatan_app/internal/conect_db"
	// import conectdb → butuh conectdb.New() untuk buat koneksi MySQL
	// → alias "conectdb" karena nama package aslinya "conect_db" dengan underscore

	"catatan_app/internal/handler"
	// import handler → butuh handler.NewCatatanHandler() dan handler.NewAuthHandler()

	"catatan_app/internal/middleware"
	// import middleware → butuh middleware.InitLogger(), middleware.NewCORS(),
	//                     middleware.Logger(), middleware.Recovery()

	"catatan_app/internal/repository"
	// import repository → butuh repository.NewCatatanRepository() dan repository.NewUserRepository()

	"catatan_app/internal/router"
	// import router → butuh router.Register() untuk daftarkan semua route ke mux

	"catatan_app/internal/service"
	// import service → butuh service.NewCatatanService() dan service.NewAuthService()

	"context"
	// import context → butuh context.WithTimeout() untuk graceful shutdown dengan batas waktu

	"net/http"
	// import net/http → butuh http.NewServeMux() dan http.Server

	"os"
	// import os → butuh os.Getenv() untuk baca APP_PORT dan os.Signal untuk tangkap sinyal OS

	"os/signal"
	// import os/signal → butuh signal.Notify() untuk subscribe ke sinyal OS (Ctrl+C, SIGTERM)

	"syscall"
	// import syscall → butuh syscall.SIGTERM — sinyal yang dikirim Docker saat container dihentikan

	"time"
	// import time → butuh time.Second untuk timeout konfigurasi server dan graceful shutdown

	"github.com/joho/godotenv"
	// import godotenv → butuh godotenv.Load() untuk baca file .env

	"github.com/rs/zerolog/log"
	// import zerolog/log → butuh log.Info(), log.Warn(), log.Fatal(), log.Error()
	// → untuk log setiap tahap startup dan shutdown
)

func main() {
	// → function main() — titik masuk program, dieksekusi pertama kali
	// → semua perakitan komponen terjadi di sini — Dependency Injection assembly point

	// ─────────────────────────────────────────
	// STEP 1 — LOAD ENVIRONMENT VARIABLE
	// ─────────────────────────────────────────
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("env tidak ditemukan, pakai system env")
		// godotenv.Load() → baca file .env di direktori yang sama dengan binary
		// → kalau .env tidak ada: tidak fatal — bisa pakai environment variable dari sistem
		// → di Docker: env_file di docker-compose.yml sudah inject variabel ke sistem
		// → log.Warn bukan log.Fatal karena ini bukan error fatal
	}

	// ─────────────────────────────────────────
	// STEP 2 — INIT LOGGER
	// ─────────────────────────────────────────
	middleware.InitLogger()
	// → setup zerolog sesuai APP_ENV sebelum apapun dijalankan
	// → dipanggil SETELAH godotenv.Load() agar APP_ENV sudah terbaca dari .env
	// → kalau dipanggil sebelum Load(): APP_ENV belum ada → selalu pakai default development

	// ─────────────────────────────────────────
	// STEP 3 — KONEKSI DATABASE
	// ─────────────────────────────────────────
	db, err := conectdb.New()
	// → buat koneksi MySQL + connection pool
	// → membaca DB_DSN dari environment variable yang sudah di-load
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect database")
		// log.Fatal() → log error lalu panggil os.Exit(1)
		// → kalau database tidak bisa connect: tidak ada gunanya lanjut
		// → server tidak bisa jalan tanpa database
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error().Err(err).Msg("error closing database")
		}
	}()
	// defer db.Close() → tutup semua koneksi database saat main() selesai
	// → dipanggil setelah graceful shutdown selesai
	// → dibungkus anonymous function agar bisa handle error dari db.Close()
	log.Info().Msg("database terhubung")

	// ─────────────────────────────────────────
	// STEP 4 — DEPENDENCY INJECTION ASSEMBLY
	// ─────────────────────────────────────────

	// Buat Repository — inject koneksi database
	catatanRepo := repository.NewCatatanRepository(db)
	// → CatatanRepository memegang *sql.DB untuk query catatan
	userRepo := repository.NewUserRepository(db)
	// → UserRepository memegang *sql.DB untuk query user

	// Buat Service — inject repository via interface
	catatanSvc := service.NewCatatanService(catatanRepo)
	// → CatatanService memegang CatatanRepo (interface) — tidak tahu implementasinya MySQL
	authSvc := service.NewAuthService(userRepo)
	// → AuthService memegang UserRepo (interface)

	// Buat Handler — inject service via interface
	catatanHandler := handler.NewCatatanHandler(catatanSvc)
	// → CatatanHandler memegang CatatanSvc (interface) — tidak tahu implementasinya
	authHandler := handler.NewAuthHandler(authSvc)
	// → AuthHandler memegang AuthSvc (interface)

	// → urutan assembly wajib dari dalam ke luar:
	//   db → repo → service → handler
	//   tidak bisa dibalik karena setiap layer butuh layer di bawahnya

	// ─────────────────────────────────────────
	// STEP 5 — ROUTING
	// ─────────────────────────────────────────
	mux := http.NewServeMux()
	// http.NewServeMux() → buat router baru yang kosong
	// → mux = multiplexer — routing request ke handler yang sesuai berdasarkan path

	router.Register(mux, catatanHandler, authHandler)
	// → daftarkan semua endpoint ke mux
	// → setelah ini mux sudah tahu: POST /api/v1/catatan → catatanHandler.Create, dst

	// ─────────────────────────────────────────
	// STEP 6 — MIDDLEWARE CHAIN
	// ─────────────────────────────────────────
	corsMiddleware := middleware.NewCORS()
	// → buat instance CORS dengan konfigurasi dari .env

	chain := middleware.Logger(
		middleware.Recovery(
			mux,
		),
	)
	// → bungkus mux dengan middleware chain dari dalam ke luar
	// → urutan pembungkusan: Recovery dulu baru Logger
	// → tapi urutan eksekusi saat request masuk: Logger → Recovery → mux
	//   karena middleware terluar (Logger) dieksekusi duluan
	// → kenapa Logger di luar Recovery:
	//   Logger perlu catat durasi dan status — harus bungkus semuanya termasuk recovery
	//   Recovery perlu tangkap panic dari handler — harus lebih dekat ke mux

	finalHandler := corsMiddleware.Handler(chain)
	// → bungkus chain dengan CORS middleware
	// → CORS paling luar karena harus diproses sebelum apapun
	//   preflight OPTIONS request dari browser harus dijawab sebelum masuk ke Auth atau handler
	// → urutan eksekusi lengkap saat request masuk:
	//   CORS → Logger → Recovery → Auth (per route) → Handler

	// ─────────────────────────────────────────
	// STEP 7 — KONFIGURASI HTTP SERVER
	// ─────────────────────────────────────────
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
		// → fallback ke 8080 kalau APP_PORT tidak di-set di .env
	}

	server := &http.Server{
		Addr: ":" + port,
		// → listen di semua interface pada port yang ditentukan
		// → ":" + "8080" = ":8080" = 0.0.0.0:8080

		Handler: finalHandler,
		// → semua request diproses oleh finalHandler (CORS → Logger → Recovery → mux)

		ReadTimeout: 10 * time.Second,
		// → maksimal 10 detik untuk baca seluruh request termasuk body
		// → proteksi dari slowloris attack — attacker kirim request sangat lambat

		WriteTimeout: 10 * time.Second,
		// → maksimal 10 detik untuk tulis response ke client
		// → proteksi dari client yang lambat terima response

		IdleTimeout: 60 * time.Second,
		// → maksimal 60 detik koneksi keep-alive idle sebelum ditutup
		// → koneksi HTTP/1.1 bisa di-reuse — IdleTimeout batasi berapa lama menunggu
	}

	// ─────────────────────────────────────────
	// STEP 8 — JALANKAN SERVER
	// ─────────────────────────────────────────
	go func() {
		// → jalankan server di goroutine terpisah (non-blocking)
		// → kenapa goroutine: ListenAndServe() adalah blocking call — tidak pernah return
		//   kalau tidak di goroutine: code graceful shutdown di bawah tidak pernah dieksekusi
		log.Info().Str("port", port).Msg("server berjalan")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// http.ErrServerClosed → error normal yang di-return ListenAndServe
			//   setelah server.Shutdown() dipanggil — bukan error sebenarnya
			// → kalau error selain ErrServerClosed: berarti server gagal start (port sudah dipakai, dll)
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// ─────────────────────────────────────────
	// STEP 9 — GRACEFUL SHUTDOWN
	// ─────────────────────────────────────────
	stop := make(chan os.Signal, 1)
	// make(chan os.Signal, 1) → buat buffered channel untuk terima sinyal OS
	// → buffer 1 agar signal.Notify tidak blocking kalau channel belum dibaca

	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	// signal.Notify → subscribe ke sinyal OS
	// os.Interrupt → sinyal dari Ctrl+C di terminal
	// syscall.SIGTERM → sinyal yang dikirim Docker saat docker compose down
	// → kalau salah satu sinyal ini diterima → kirim ke channel stop

	<-stop
	// → blokir di sini sampai ada sinyal masuk ke channel stop
	// → ini yang membuat main() tidak langsung selesai setelah server dijalankan
	// → saat Ctrl+C atau docker compose down → <-stop tidak blocking lagi → lanjut ke bawah
	log.Warn().Msg("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// context.WithTimeout → buat context dengan batas waktu 5 detik
	// → kalau shutdown tidak selesai dalam 5 detik → context cancel otomatis
	defer cancel()
	// defer cancel() → pastikan context di-cancel untuk bebaskan resource
	// → wajib dipanggil untuk mencegah context leak

	if err := server.Shutdown(ctx); err != nil {
		// server.Shutdown(ctx) → mulai graceful shutdown:
		// 1. berhenti terima request baru
		// 2. tunggu semua request yang sedang diproses selesai
		// 3. kalau lebih dari 5 detik → paksa shutdown
		log.Error().Err(err).Msg("server gagal mati")
	}

	log.Info().Msg("server keluar dengan selamat")
	// → setelah Shutdown selesai → defer db.Close() dieksekusi → koneksi database ditutup
	// → program selesai dengan bersih
}

// Kenapa urutan middleware CORS → Logger → Recovery (dari luar ke dalam):
// → CORS terluar: preflight OPTIONS harus dijawab sebelum masuk ke manapun
// → Logger tengah: perlu catat durasi seluruh proses termasuk recovery
// → Recovery terdalam: harus sedekat mungkin dengan handler untuk tangkap panic

// Kenapa server dijalankan di goroutine:
// → ListenAndServe() blocking — tidak pernah return sampai server shutdown
// → kalau tidak di goroutine: graceful shutdown code tidak pernah dieksekusi
// → goroutine memungkinkan main() lanjut ke <-stop untuk menunggu sinyal

// Pattern main.go — urutan assembly selalu:
// 1.  Load .env              → godotenv.Load()
// 2.  Init logger            → middleware.InitLogger()
// 3.  Connect database       → conectdb.New()
// 4.  Buat repository        → inject db
// 5.  Buat service           → inject repository (via interface)
// 6.  Buat handler           → inject service (via interface)
// 7.  Buat mux               → http.NewServeMux()
// 8.  Register route         → router.Register()
// 9.  Bungkus middleware      → Logger(Recovery(mux))
// 10. Bungkus CORS           → corsMiddleware.Handler(chain)
// 11. Konfigurasi server     → &http.Server{}
// 12. Jalankan server        → go server.ListenAndServe()
// 13. Tunggu sinyal shutdown → <-stop
// 14. Graceful shutdown      → server.Shutdown(ctx)
