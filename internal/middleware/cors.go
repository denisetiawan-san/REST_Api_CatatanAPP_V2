package middleware

// package middleware → sama dengan file middleware lainnya, satu package untuk semua middleware

import (
	"os"
	// import os → butuh os.Getenv() untuk baca CORS_ALLOWED_ORIGINS dan APP_ENV dari .env

	"strings"
	// import strings → butuh strings.Split() untuk pisah string origins berdasarkan koma

	"github.com/rs/cors"
	// import cors → library rs/cors untuk handle CORS
	// → butuh cors.New() untuk buat instance dan cors.Options untuk konfigurasi
)

// NewCORS → buat dan return instance cors yang sudah dikonfigurasi
// → dipanggil di main.go satu kali saat startup, bukan setiap request
// → return *cors.Cors yang nanti di-chain ke mux di main.go
// → kalau dihapus: tidak ada CORS handling → semua request dari browser akan diblokir
func NewCORS() *cors.Cors {
	originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	// → baca nilai CORS_ALLOWED_ORIGINS dari .env
	// → contoh nilai di .env: CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:5173
	// → kalau tidak di-set: originsEnv = "" → allowedOrigins = [""] → tidak ada origin yang diizinkan

	allowedOrigins := strings.Split(originsEnv, ",")
	// strings.Split(originsEnv, ",") → pisah string berdasarkan koma jadi slice
	// → "http://localhost:3000,http://localhost:5173" → ["http://localhost:3000", "http://localhost:5173"]
	// → kenapa dari .env bukan hardcode: karena origin berbeda di setiap environment
	//   development → http://localhost:3000
	//   staging     → https://staging.frontend.com
	//   production  → https://frontend.com
	// → kalau hardcode: harus ubah code setiap ganti environment — itu salah

	return cors.New(cors.Options{
		AllowedOrigins: allowedOrigins,
		// → hanya origin di list ini yang boleh akses API
		// → request dari origin lain → browser blokir sebelum sampai ke server
		// → ini proteksi dari website jahat yang coba akses API kamu dari browser user

		AllowedMethods: []string{
			"GET",
			"POST",
			"PUT",
			"PATCH",
			"DELETE",
			"OPTIONS",
			// OPTIONS → wajib ada untuk handle preflight request dari browser
			// → browser selalu kirim OPTIONS dulu sebelum request asli (POST, PUT, DELETE)
			// → untuk cek apakah server izinkan cross-origin request
			// → kalau OPTIONS tidak ada: semua request non-GET dari browser akan gagal
		},

		AllowedHeaders: []string{
			"Content-Type",
			// → izinkan header Content-Type — dibutuhkan untuk kirim JSON body
			// → tanpa ini: request dengan body JSON akan diblokir browser

			"Authorization",
			// → izinkan header Authorization — dibutuhkan untuk kirim JWT token
			// → format: Authorization: Bearer eyJ...
			// → tanpa ini: semua protected request akan diblokir browser
		},

		ExposedHeaders: []string{
			"Content-Length",
			// → izinkan client baca header Content-Length dari response
			// → by default browser hanya expose beberapa header standar
			// → header lain harus di-whitelist di sini agar bisa dibaca JavaScript
		},

		AllowCredentials: true,
		// → izinkan credentials dikirim bersama request
		// → credentials: cookie, Authorization header, TLS client certificate
		// → dibutuhkan karena kita pakai Authorization header untuk JWT
		// → kalau false: browser tidak kirim Authorization header → semua protected request gagal

		Debug: os.Getenv("APP_ENV") != "production",
		// → aktifkan debug log CORS kalau bukan production
		// → debug log menampilkan detail setiap CORS decision — berguna saat development
		// → dimatikan di production karena terlalu verbose dan expose info konfigurasi
		// → os.Getenv("APP_ENV") != "production" → true di development, false di production
	})
}

// Apa itu CORS dan kenapa perlu:
// → CORS = Cross-Origin Resource Sharing
// → browser modern punya "same-origin policy" — JavaScript hanya boleh request ke origin yang sama
// → origin = protokol + domain + port: http://localhost:3000 ≠ http://localhost:8080
// → tanpa CORS header dari server: browser blokir response sebelum JavaScript bisa baca
// → CORS middleware menambahkan header ke response yang memberi tahu browser:
//   "origin ini boleh baca response dari server ini"

// Apa itu preflight request:
// → sebelum kirim request "tidak sederhana" (POST, PUT, DELETE, atau dengan custom header)
// → browser otomatis kirim request OPTIONS dulu ke endpoint yang sama
// → untuk tanya server: "boleh tidak saya kirim POST dari origin X dengan header Y?"
// → kalau server jawab boleh (return 200 + CORS headers) → browser kirim request aslinya
// → kalau server tidak handle OPTIONS → browser blokir request asli → error di frontend

// Pattern CORS middleware:
// 1. Baca allowed origins dari .env — bukan hardcode
// 2. Definisikan allowed methods — jangan lupa OPTIONS untuk preflight
// 3. Definisikan allowed headers — minimal Content-Type dan Authorization
// 4. AllowCredentials true — supaya Authorization header bisa dikirim
// 5. Debug hanya di non-production
