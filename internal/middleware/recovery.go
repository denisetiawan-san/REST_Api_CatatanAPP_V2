package middleware

// package middleware → package untuk semua middleware HTTP
// → diimport oleh main.go untuk dibungkus ke mux
// → kalau dihapus: tidak ada yang tangkap panic, server bisa mati karena satu request

import (
	"net/http"
	// import net/http → butuh http.Handler, http.HandlerFunc, http.ResponseWriter, *http.Request

	"runtime/debug"
	// import runtime/debug → butuh debug.Stack() untuk ambil stack trace saat panic
	// → stack trace menunjukkan baris code mana yang menyebabkan panic
	// → kalau dihapus: log panic tidak ada stack trace — susah debug

	"github.com/rs/zerolog/log"
	// import zerolog/log → butuh log.Error() untuk catat panic ke log
	// → pakai package-level logger yang sudah di-init oleh InitLogger() di logger.go
)

// Recovery → middleware yang menangkap panic yang terjadi di handler manapun
// → menerima http.Handler (handler berikutnya) dan return http.Handler baru yang membungkusnya
// → dipasang di main.go sebagai bagian dari middleware chain
// → kalau dihapus: satu panic di handler akan membunuh seluruh server
//
//	semua request lain yang sedang berjalan ikut mati
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.HandlerFunc → konversi anonymous function ke tipe http.Handler
		// → supaya bisa di-return sebagai http.Handler
		// → anonymous function ini yang dieksekusi setiap ada request masuk

		defer func() {
			// defer → eksekusi function ini setelah function selesai, apapun yang terjadi
			// → termasuk kalau ada panic di tengah jalan
			// → ini kunci dari recovery middleware — defer dipanggil bahkan saat panic

			if err := recover(); err != nil {
				// recover() → tangkap panic yang terjadi di goroutine ini
				// → return nil kalau tidak ada panic
				// → return nilai panic kalau ada panic (bisa tipe apapun)
				// → recover() hanya bekerja di dalam defer function

				log.Error().
					Interface("panic", err).
					// Interface("panic", err) → log nilai panic — bisa string, error, atau tipe lain
					Bytes("stack", debug.Stack()).
					// debug.Stack() → ambil stack trace lengkap saat panic terjadi
					// → menunjukkan urutan function call yang menyebabkan panic
					Str("method", r.Method).
					// → log HTTP method untuk tahu request apa yang menyebabkan panic
					Str("path", r.URL.Path).
					// → log URL path untuk tahu endpoint mana yang panic
					Msg("panic recovered")
				// → tulis semua field ke log dengan level Error

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error":"terjadi kesalahan pada server"}`))
				// → kirim 500 ke client dengan format JSON
				// → tidak pakai writeJSON dari handler karena beda package
				// → tidak expose detail panic ke client — detail ada di server log
				// → client cukup tahu "terjadi kesalahan", bukan detail panic-nya
			}
		}()
		// → defer dipasang dulu sebelum next.ServeHTTP dipanggil
		// → urutan ini penting — kalau defer dipasang setelah ServeHTTP, tidak akan berguna

		next.ServeHTTP(w, r)
		// → lanjutkan ke handler berikutnya dalam chain
		// → kalau handler ini panic → defer di atas akan tangkap
		// → kalau tidak panic → request jalan normal
	})
}

// Cara kerja defer + recover() untuk tangkap panic:
// 1. Request masuk → defer func() didaftarkan tapi belum dieksekusi
// 2. next.ServeHTTP(w, r) dipanggil → jalankan handler berikutnya
// 3. Kalau handler panic → Go unwind stack → eksekusi semua defer yang terdaftar
// 4. defer func() dieksekusi → recover() tangkap panic → log + return 500
// 5. Server tetap hidup — panic sudah di-handle dengan bersih
//
// Kalau tidak ada Recovery middleware:
// 1. Handler panic → Go unwind stack → tidak ada recover()
// 2. Goroutine yang handle request ini mati
// 3. Seluruh server mati karena panic tidak tertangkap

// Pattern middleware di Go:
// 1. Terima http.Handler sebagai parameter → handler berikutnya dalam chain
// 2. Return http.Handler baru → membungkus handler lama dengan logic tambahan
// 3. Di dalam: jalankan logic sebelum next.ServeHTTP (pre-processing)
//              panggil next.ServeHTTP(w, r) → teruskan ke handler berikutnya
//              jalankan logic setelah next.ServeHTTP (post-processing)
// 4. defer dipakai untuk logic yang harus jalan apapun yang terjadi (recovery, logging durasi)
