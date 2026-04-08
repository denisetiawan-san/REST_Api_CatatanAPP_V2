package router

// package router → package untuk definisi semua route API
// → diimport oleh main.go untuk daftarkan semua endpoint ke mux
// → kalau dihapus: tidak ada endpoint yang terdaftar, server tidak bisa terima request apapun

import (
	"catatan_app/internal/handler"
	// import handler → butuh *handler.CatatanHandler dan *handler.AuthHandler
	// → sebagai tujuan dari setiap route yang didaftarkan

	"catatan_app/internal/middleware"
	// import middleware → butuh middleware.Auth untuk bungkus protected route

	"net/http"
	// import net/http → butuh *http.ServeMux, http.HandlerFunc, http.MethodPost, dll
)

// methodNotAllowed → helper untuk tulis response 405 dalam format JSON
// → dipanggil di setiap route kalau client kirim method yang tidak didukung
// → dipisah agar format response 405 konsisten di semua route
// → tidak bisa pakai writeError dari package handler karena beda package
func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusMethodNotAllowed)
	w.Write([]byte(`{"error":"method not allowed"}`))
	// → tulis manual karena tidak bisa akses writeError dari package handler
}

// Register → daftarkan semua route ke mux dengan prefix /api/v1
// → dipanggil di main.go satu kali saat startup
// → menerima mux, CatatanHandler, dan AuthHandler sebagai parameter
// → kenapa tidak dibuat per resource (RegisterCatatan, RegisterAuth):
//
//	karena project ini masih kecil — satu function cukup
//	kalau resource bertambah banyak bisa dipisah per function
func Register(mux *http.ServeMux, h *handler.CatatanHandler, ah *handler.AuthHandler) {

	// ─────────────────────────────────────────
	// PUBLIC ROUTE — tidak butuh JWT token
	// ─────────────────────────────────────────

	mux.HandleFunc("/api/v1/auth/register", func(w http.ResponseWriter, r *http.Request) {
		// mux.HandleFunc → daftarkan route dengan function handler langsung
		// → dipakai untuk public route karena tidak perlu dibungkus middleware Auth
		// → http.HandlerFunc dibuat implisit oleh HandleFunc
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			// → method selain POST ke /register → 405 Method Not Allowed
			// → register hanya boleh POST — tidak ada GET /register
			return
		}
		ah.Register(w, r)
		// → teruskan ke AuthHandler.Register untuk diproses
	})

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			// → method selain POST ke /login → 405 Method Not Allowed
			return
		}
		ah.Login(w, r)
		// → teruskan ke AuthHandler.Login untuk diproses
	})

	// ─────────────────────────────────────────
	// PROTECTED ROUTE — butuh JWT token valid
	// ─────────────────────────────────────────

	mux.Handle("/api/v1/catatan", middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// mux.Handle → daftarkan route dengan http.Handler
		// → dipakai untuk protected route karena bisa dibungkus middleware
		// → middleware.Auth(...) → bungkus handler dengan Auth middleware
		// → http.HandlerFunc(...) → konversi anonymous function ke http.Handler
		// → kalau Auth gagal (token invalid): middleware return 401, handler tidak dipanggil
		switch r.Method {
		case http.MethodPost:
			h.Create(w, r)
			// → POST /api/v1/catatan → buat catatan baru

		case http.MethodGet:
			h.List(w, r)
			// → GET /api/v1/catatan → ambil semua catatan dengan filter dan pagination

		default:
			methodNotAllowed(w)
			// → method lain (PUT, DELETE, PATCH) ke /catatan → 405
			// → PUT dan DELETE hanya valid ke /catatan/{id}, bukan ke /catatan
		}
	})))

	mux.Handle("/api/v1/catatan/{id}", middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// {id} → wildcard path parameter — fitur Go 1.22+
		// → cocok dengan /api/v1/catatan/1, /api/v1/catatan/99, dll
		// → nilai id diambil di handler dengan r.PathValue("id")
		switch r.Method {
		case http.MethodGet:
			h.GetByID(w, r)
			// → GET /api/v1/catatan/{id} → ambil satu catatan

		case http.MethodPut:
			h.Update(w, r)
			// → PUT /api/v1/catatan/{id} → update catatan

		case http.MethodDelete:
			h.Delete(w, r)
			// → DELETE /api/v1/catatan/{id} → hapus catatan

		default:
			methodNotAllowed(w)
			// → method lain (POST, PATCH) ke /catatan/{id} → 405
		}
	})))

	mux.Handle("/api/v1/catatan/{id}/arsip", middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// → path spesifik untuk operasi arsip
		// → {id} sama dengan route sebelumnya — diambil di handler dengan r.PathValue("id")
		if r.Method != http.MethodPatch {
			methodNotAllowed(w)
			// → hanya PATCH yang valid ke /arsip — ini partial update, bukan full update
			// → PATCH = ubah sebagian resource (hanya kolom arsip)
			// → PUT = ubah seluruh resource (judul + isi)
			return
		}
		h.Arsip(w, r)
		// → PATCH /api/v1/catatan/{id}/arsip → arsipkan catatan
	})))

	mux.Handle("/api/v1/catatan/{id}/unarsip", middleware.Auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			methodNotAllowed(w)
			// → hanya PATCH yang valid ke /unarsip
			return
		}
		h.Unarsip(w, r)
		// → PATCH /api/v1/catatan/{id}/unarsip → kembalikan catatan dari arsip
	})))
}

// Perbedaan mux.HandleFunc vs mux.Handle:
// mux.HandleFunc → terima func(w, r) langsung — cocok untuk public route tanpa middleware
// mux.Handle     → terima http.Handler — cocok untuk protected route yang perlu dibungkus middleware
//                  middleware.Auth() return http.Handler, jadi harus pakai mux.Handle

// Kenapa middleware.Auth dipasang per route, bukan di global chain di main.go:
// → /auth/register dan /auth/login adalah public route — tidak butuh token
// → kalau Auth dipasang global: user tidak bisa login karena butuh token untuk login
// → dengan pasang per route: kontrol eksplisit — hanya route catatan yang diproteksi

// Kenapa switch method, bukan satu endpoint per method:
// → Go standard library tidak support routing per method seperti Gin atau Echo
// → solusinya: satu endpoint untuk satu path, switch method di dalamnya
// → framework seperti Gin punya r.GET(), r.POST() — di standard library tidak ada
// → ini salah satu trade-off pakai standard library vs framework

// Pattern setiap route di router:
// 1. Public route  → mux.HandleFunc → cek method → panggil handler
// 2. Protected route → mux.Handle → bungkus middleware.Auth → switch method → panggil handler
// 3. Setiap route selalu punya default case methodNotAllowed
// 4. Semua path pakai prefix /api/v1/ untuk API versioning
