package middleware

// package middleware → sama dengan file middleware lainnya, satu package untuk semua middleware

import (
	"catatan_app/internal/apperror"
	// import apperror → butuh apperror.ErrUnauthorized untuk pesan error 401

	"context"
	// import context → butuh context.WithValue() untuk inject claims ke context request

	"encoding/json"
	// import encoding/json → butuh json.NewEncoder() untuk encode response error ke JSON
	// → dipakai di writeUnauthorized agar karakter khusus di-escape dengan benar
	// → lebih aman dari string concatenation manual

	"net/http"
	// import net/http → butuh http.Handler, http.HandlerFunc, http.ResponseWriter, *http.Request

	"os"
	// import os → butuh os.Getenv() untuk baca JWT_SECRET dari .env

	"strings"
	// import strings → butuh strings.SplitN() untuk pisah "Bearer <token>" jadi dua bagian

	"github.com/golang-jwt/jwt/v5"
	// import jwt → butuh jwt.Parse(), jwt.MapClaims, jwt.SigningMethodHMAC
	// → untuk parse dan verifikasi JWT token

	"github.com/rs/zerolog/log"
	// import zerolog/log → butuh log.Warn() untuk catat token tidak valid ke log
)

// contextKey → custom type untuk key di context
// → kenapa tidak pakai string biasa: kalau pakai string "claims", bisa bentrok
//
//	dengan key yang sama dari package lain yang juga pakai string "claims"
//
// → custom type memastikan key unik — hanya code yang import package ini
//
//	yang bisa akses nilai dengan key bertipe contextKey
type contextKey string

// ClaimsKey → konstanta key untuk simpan dan ambil claims dari context
// → di-export (huruf kapital) agar bisa diakses handler di package lain
// → handler yang butuh data user: claims := r.Context().Value(middleware.ClaimsKey)
const ClaimsKey contextKey = "claims"

// Auth → middleware yang memvalidasi JWT token dari header Authorization
// → dipasang di router hanya untuk protected route — bukan global middleware
// → public route (register, login) tidak melewati middleware ini
// → kalau dihapus: semua route catatan bisa diakses tanpa token
func Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		authHeader := r.Header.Get("Authorization")
		// r.Header.Get("Authorization") → ambil nilai header Authorization dari request
		// → return string kosong "" kalau header tidak ada
		if authHeader == "" {
			writeUnauthorized(w, apperror.ErrUnauthorized.Error())
			// → tidak ada header Authorization → tolak dengan 401
			// → client belum login atau lupa kirim token
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		// strings.SplitN(authHeader, " ", 2) → pisah string berdasarkan spasi, maksimal 2 bagian
		// → "Bearer eyJ..." → ["Bearer", "eyJ..."]
		// → N=2 → kalau ada lebih dari satu spasi, sisa string tetap di parts[1]
		if len(parts) != 2 || parts[0] != "Bearer" {
			writeUnauthorized(w, apperror.ErrUnauthorized.Error())
			// → format header salah → tolak dengan 401
			// → contoh format salah: "Token eyJ..." atau "eyJ..." tanpa prefix
			// → standar JWT di HTTP adalah "Bearer <token>" — tidak boleh format lain
			return
		}

		tokenString := parts[1]
		// → ambil token string dari parts[1] — bagian setelah "Bearer "
		// → ini yang akan di-parse dan diverifikasi

		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			// jwt.Parse → parse token string, verifikasi signature, cek expired
			// → parameter kedua adalah key function — dipanggil jwt library untuk ambil secret key
			// → t adalah token yang sedang di-parse, sebelum diverifikasi

			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, apperror.ErrUnauthorized
				// → cek algoritma yang dipakai token harus HMAC (HS256)
				// → kalau bukan HMAC → tolak
				// → ini proteksi dari algorithm confusion attack:
				//   attacker bisa kirim token dengan algoritma "none" untuk bypass verifikasi
				//   dengan cek eksplisit ini, token dengan algoritma selain HMAC ditolak
				// → *jwt.SigningMethodHMAC → type assertion untuk cek tipe method
			}

			return []byte(os.Getenv("JWT_SECRET")), nil
			// → return secret key sebagai []byte untuk verifikasi signature token
			// → jwt library pakai ini untuk hitung ulang signature dan bandingkan dengan token
			// → kalau secret berbeda: signature tidak cocok → token invalid
		})

		if err != nil || !token.Valid {
			log.Warn().Err(err).Str("path", r.URL.Path).Msg("token tidak valid")
			// → log dengan level Warn — bukan Error karena ini bukan bug, bisa saja attacker
			// → catat error dan path untuk monitoring keamanan
			writeUnauthorized(w, apperror.ErrUnauthorized.Error())
			// → token expired, signature tidak cocok, atau format token salah → 401
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		// token.Claims → data yang tersimpan di dalam token (user_id, email, exp)
		// .(jwt.MapClaims) → type assertion — konversi interface{} ke jwt.MapClaims
		// ok → false kalau type assertion gagal
		if !ok {
			writeUnauthorized(w, apperror.ErrUnauthorized.Error())
			// → claims tidak bisa di-extract → format token tidak sesuai → 401
			return
		}
		// → sampai sini berarti token valid dan claims berhasil di-extract

		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		// context.WithValue → buat context baru dengan tambahan key-value pair
		// r.Context() → context asli dari request (sudah berisi context dari middleware sebelumnya)
		// ClaimsKey → key untuk simpan dan ambil claims
		// claims → data user yang disimpan: user_id, email, exp
		// → kenapa di context: supaya handler bisa akses data user tanpa perlu query database lagi
		//   token sudah berisi semua info yang dibutuhkan handler

		next.ServeHTTP(w, r.WithContext(ctx))
		// r.WithContext(ctx) → buat request baru dengan context yang sudah berisi claims
		// → handler yang ada di depan bisa ambil claims dengan:
		//   claims := r.Context().Value(middleware.ClaimsKey).(jwt.MapClaims)
	})
}

// writeUnauthorized → helper untuk tulis response 401 dalam format JSON
// → dipisah dari writeError di handler karena beda package
// → middleware tidak bisa akses writeError dari package handler
// → kalau dihapus: harus tulis 3 baris response manual di setiap tempat yang butuh 401
func writeUnauthorized(w http.ResponseWriter, pesan string) {
	w.Header().Set("Content-Type", "application/json")
	// → set header Content-Type sebelum WriteHeader — urutan ini wajib
	// → kalau set header setelah WriteHeader: header diabaikan, sudah terlanjur dikirim

	w.WriteHeader(http.StatusUnauthorized)
	// → tulis status code 401 ke response

	json.NewEncoder(w).Encode(map[string]string{"error": pesan})
	// json.NewEncoder(w) → buat encoder yang langsung tulis ke http.ResponseWriter
	// .Encode(map[string]string{"error": pesan}) → encode map ke JSON dan tulis ke w
	// → hasil: {"error": "pesan error"}
	// → kenapa ganti dari string concatenation ke json.NewEncoder:
	//   versi lama: `{"error":"` + pesan + `"}` → tidak aman
	//   kalau pesan mengandung kutip: {"error":"dia bilang "halo""} → JSON invalid
	//   json.NewEncoder otomatis escape karakter khusus → selalu menghasilkan JSON valid
	// → konsisten dengan writeError di package handler yang juga pakai json.NewEncoder
}

// Alur lengkap middleware Auth setiap request masuk ke protected route:
// 1. Ambil header Authorization → tidak ada → 401
// 2. Split "Bearer <token>" → format salah → 401
// 3. Parse token → verifikasi signature dengan JWT_SECRET → gagal → 401
// 4. Cek algoritma HMAC → bukan HMAC → 401 (cegah algorithm confusion attack)
// 5. Cek token.Valid → expired atau invalid → 401
// 6. Extract claims → gagal → 401
// 7. Inject claims ke context → lanjut ke handler

// Kenapa Auth dipasang di router, bukan di global middleware chain di main.go:
// → public route (register, login) tidak butuh token — tidak perlu Auth
// → kalau Auth dipasang global: user tidak bisa login karena butuh token untuk login
// → dengan pasang di router per-route: kontrol eksplisit mana public mana protected

// Pattern middleware Auth:
// 1. Ambil dan validasi format header Authorization
// 2. Parse dan verifikasi JWT token — cek signature, expired, algoritma
// 3. Extract claims dari token yang sudah terverifikasi
// 4. Inject claims ke context untuk diakses handler
// 5. Lanjut ke handler kalau semua valid, tolak 401 kalau ada yang gagal
