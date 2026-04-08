package middleware

// package middleware → sama dengan recovery.go, satu package untuk semua middleware

import (
	"net/http"
	// import net/http → butuh http.Handler, http.HandlerFunc, http.ResponseWriter, *http.Request

	"os"
	// import os → butuh os.Getenv() untuk baca APP_ENV dari .env

	"time"
	// import time → butuh time.Now() untuk catat waktu mulai request
	// → dan time.Since() untuk hitung durasi response

	"github.com/rs/zerolog"
	// import zerolog → butuh zerolog.SetGlobalLevel() dan zerolog.ConsoleWriter
	// → untuk konfigurasi format dan level log

	"github.com/rs/zerolog/log"
	// import zerolog/log → butuh log.Logger dan log.Info() untuk tulis log
	// → log adalah package-level logger dari zerolog
)

// InitLogger → setup zerolog satu kali saat aplikasi start
// → dipanggil di main.go sebelum server dijalankan, bukan di setiap request
// → menentukan format dan level log berdasarkan APP_ENV di .env
// → kalau dihapus: zerolog pakai default config — format JSON tanpa level filter
func InitLogger() {
	appEnv := os.Getenv("APP_ENV")
	// → baca APP_ENV dari environment variable
	// → "production" → format JSON untuk mesin
	// → apapun selain "production" → format console untuk manusia

	if appEnv == "production" {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		// → production: hanya log level Info ke atas (Info, Warn, Error, Fatal)
		// → level Debug tidak di-log di production — terlalu verbose, bisa expose info sensitif
		// → format tetap JSON default zerolog — mudah dibaca Datadog, Grafana, ELK stack
		// → contoh output: {"level":"info","method":"GET","path":"/api/v1/catatan","status":200}
	} else {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		// zerolog.ConsoleWriter → format log dengan warna dan human-readable di terminal
		// Out: os.Stderr → tulis ke stderr bukan stdout
		//   → stdout untuk output program, stderr untuk log dan error — konvensi Unix
		// → contoh output: 10:30:45 INF request masuk method=GET path=/api/v1/catatan status=200

		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		// → development: log semua level termasuk Debug
		// → Debug berguna saat development untuk trace detail alur program
	}
}

// Logger → middleware yang mencatat setiap request yang masuk ke server
// → dieksekusi setiap request — berbeda dengan InitLogger yang hanya sekali
// → mencatat method, path, status code, dan durasi response
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		start := time.Now()
		// → catat waktu tepat saat request masuk
		// → dipakai di akhir untuk hitung durasi: time.Since(start)

		wrapped := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		// → bungkus http.ResponseWriter asli dengan struct responseWriter kita sendiri
		// → kenapa perlu dibungkus: http.ResponseWriter default tidak expose status code
		//   setelah WriteHeader dipanggil — kita tidak tahu handler tulis status berapa
		// → dengan wrapper ini, kita bisa capture status code yang ditulis handler
		// → status: http.StatusOK → default 200 kalau handler tidak panggil WriteHeader
		//   (kalau handler hanya Write() tanpa WriteHeader(), status default adalah 200)

		next.ServeHTTP(wrapped, r)
		// → teruskan request ke handler berikutnya
		// → pakai wrapped bukan w — supaya WriteHeader handler ter-capture
		// → setelah baris ini selesai, handler sudah tulis response ke client

		log.Info().
			Str("method", r.Method).
			// → catat HTTP method: GET, POST, PUT, PATCH, DELETE
			Str("path", r.URL.Path).
			// → catat URL path: /api/v1/catatan, /api/v1/auth/login
			Int("status", wrapped.status).
			// → catat status code yang ditulis handler: 200, 201, 400, 404, 500
			// → diambil dari wrapped.status yang ter-capture saat handler panggil WriteHeader
			Dur("durasi", time.Since(start)).
			// time.Since(start) → hitung waktu dari request masuk sampai response ditulis
			// → berguna untuk monitor performa: endpoint mana yang lambat
			Msg("request masuk")
		// → tulis semua field ke log dengan level Info dan pesan "request masuk"
	})
}

// responseWriter → struct wrapper untuk capture status code dari handler
// → http.ResponseWriter adalah interface — kita buat struct yang implement interface yang sama
// → tapi dengan tambahan field status untuk capture status code
type responseWriter struct {
	http.ResponseWriter
	// → embed http.ResponseWriter asli
	// → embedding berarti semua method ResponseWriter (Header, Write, WriteHeader)
	//   tersedia di struct ini secara otomatis
	// → kalau handler panggil Write() → diteruskan ke ResponseWriter asli

	status int
	// → simpan status code yang ditulis handler
	// → diisi oleh override WriteHeader di bawah
}

// WriteHeader → override method WriteHeader dari http.ResponseWriter
// → dipanggil handler saat tulis status code: w.WriteHeader(http.StatusNotFound)
// → kalau tidak di-override: status code ditulis ke client tapi kita tidak tahu nilainya
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	// → simpan status code ke field kita sendiri dulu
	// → ini yang dipakai Logger untuk log status code

	rw.ResponseWriter.WriteHeader(status)
	// → teruskan ke ResponseWriter asli supaya status code benar-benar ditulis ke response
	// → kalau tidak diteruskan: client tidak terima status code yang benar
}

// Kenapa log ditulis SETELAH next.ServeHTTP, bukan sebelum:
// → karena status code dan durasi baru tersedia setelah handler selesai
// → kalau log sebelum: status selalu 0 dan durasi selalu 0ms
// → pattern ini disebut "post-processing" — logic dijalankan setelah handler selesai

// Kenapa InitLogger dipisah dari Logger middleware:
// → InitLogger → dipanggil sekali saat startup, setup global config zerolog
// → Logger     → dipanggil setiap request, catat info request
// → dua tanggung jawab berbeda — dipisah dengan benar

// Pattern logger middleware:
// 1. Catat waktu mulai request (pre-processing)
// 2. Bungkus ResponseWriter untuk capture status code
// 3. Panggil next.ServeHTTP → jalankan handler
// 4. Setelah handler selesai → log method, path, status, durasi (post-processing)
