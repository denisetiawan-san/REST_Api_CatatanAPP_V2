package handler

// package handler → sama dengan catatan_handler.go, satu package untuk semua handler
// → auth_handler.go bisa langsung pakai writeJSON, writeError, handleError dari catatan_handler.go
// → karena satu package, tidak perlu import

import (
	"catatan_app/internal/apperror"
	// import apperror → butuh ErrEmailSudahDipakai, ErrEmailAtauPasswordSalah, ErrUnauthorized
	// → untuk mapping di handleAuthError

	"catatan_app/internal/dto"
	// import dto → butuh dto.RegisterRequest, dto.LoginRequest untuk decode body
	// → dto.LoginResponse untuk encode response token
	// → dto.ToUserResponse untuk konversi *modul.User ke UserResponse ✅
	// → kalau dihapus: kompilasi error karena semua tipe dto tidak dikenali

	"catatan_app/internal/service"
	// import service → butuh service.AuthSvc (interface) sebagai dependency
	// → depend ke interface, bukan ke struct AuthService yang konkret

	"encoding/json"
	// import encoding/json → butuh json.NewDecoder untuk decode JSON body

	"errors"
	// import errors → butuh errors.As() untuk extract ValidationErrors dari validator
	// → dan errors.Is() untuk mapping jenis error di handleAuthError

	"net/http"
	// import net/http → butuh http.ResponseWriter, *http.Request, http.Status*

	"github.com/go-playground/validator/v10"
	// import validator → butuh validator.New() untuk instance validator
	// → dan validator.ValidationErrors untuk extract pesan error validasi
)

// validate → instance validator yang dipakai oleh semua handler di package ini
// → dibuat sekali di package level, bukan di dalam function
// → kenapa package level: validator.New() ada proses inisialisasi yang tidak murah
//
//	kalau dibuat di dalam function → setiap request buat instance baru → pemborosan
//
// → dipakai oleh Register dan Login di auth_handler.go
// → bisa juga dipakai handler lain di package ini kalau butuh validasi
var validate = validator.New()

// AuthHandler → struct yang memegang service sebagai dependency via interface
// → sama polanya dengan CatatanHandler — depend ke interface, bukan struct konkret
// → tidak ada business logic di sini — semua diproses di service
type AuthHandler struct {
	service service.AuthSvc
	// → interface AuthSvc, bukan *AuthService
	// → di unit test bisa di-inject MockAuthSvc tanpa business logic nyata
}

// NewAuthHandler → constructor untuk membuat instance AuthHandler
// → menerima service.AuthSvc dari main.go dan inject ke struct
func NewAuthHandler(s service.AuthSvc) *AuthHandler {
	return &AuthHandler{service: s}
}

// Register → handler untuk POST /api/v1/auth/register
// → pintu masuk request REGISTER dari client
// → public route — tidak ada middleware Auth di depannya
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	// → deklarasi variabel req bertipe RegisterRequest
	// → akan diisi dari JSON body request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// json.NewDecoder(r.Body) → buat decoder yang baca dari request body
		// .Decode(&req) → parse JSON dan isi ke struct req
		// → kalau JSON rusak atau format salah → return error
		writeError(w, http.StatusBadRequest, "request tidak valid")
		return
	}

	if err := validate.Struct(req); err != nil {
		// validate.Struct(req) → jalankan validasi berdasarkan validate tag di RegisterRequest
		// → cek required, min, max, email format
		// → return error kalau ada field yang tidak memenuhi syarat
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			// errors.As → cek apakah err bertipe validator.ValidationErrors
			// → kalau ya, isi variabel ve dengan detail error validasi
			// → ve adalah slice — satu entry per field yang gagal validasi
			writeError(w, http.StatusBadRequest, ve[0].Translate(nil))
			// ve[0] → ambil error validasi pertama yang gagal
			// .Translate(nil) → nil artinya tidak ada translator yang dipasang
			// → pesan yang keluar adalah pesan default validator dalam format teknis
			// → contoh: "Key: 'RegisterRequest.Nama' Error:Field validation for 'Nama' failed on the 'required' tag"
			// → kalau ingin pesan lebih ramah: perlu register custom translator dengan ut.RegisterTranslation()
		} else {
			writeError(w, http.StatusBadRequest, "request tidak valid")
			// → error validasi yang tidak bisa di-extract → pesan generic
		}
		return
	}
	// → sampai sini berarti JSON valid dan semua field lolos validasi

	user, err := h.service.Register(r.Context(), req)
	// → serahkan req ke service untuk diproses
	// → service yang cek duplikasi email, hash password, simpan ke database
	// → return *modul.User kalau berhasil
	if err != nil {
		handleAuthError(w, err)
		// → pakai handleAuthError bukan handleError
		// → karena error auth punya HTTP status berbeda (409, 401)
		return
	}

	writeJSON(w, http.StatusCreated, dto.ToUserResponse(user))
	// dto.ToUserResponse(user) → konversi *modul.User ke UserResponse DTO
	// → Password tidak akan masuk response karena UserResponse tidak punya field Password
	// → konsisten dengan semua endpoint lain yang pakai mapper — tidak ada map[string]any manual
	// → writeJSON encode UserResponse ke JSON dan tulis ke response
	// → 201 Created untuk operasi yang berhasil buat resource baru
}

// Login → handler untuk POST /api/v1/auth/login
// → pintu masuk request LOGIN dari client
// → public route — tidak ada middleware Auth di depannya
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	// → deklarasi variabel req bertipe LoginRequest
	// → akan diisi dari JSON body request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// → parse JSON body ke struct LoginRequest
		// → kalau JSON rusak atau format salah → return error
		writeError(w, http.StatusBadRequest, "request tidak valid")
		return
	}

	if err := validate.Struct(req); err != nil {
		// → jalankan validasi berdasarkan validate tag di LoginRequest
		// → cek required dan email format
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			// → extract detail error validasi
			writeError(w, http.StatusBadRequest, ve[0].Translate(nil))
			// → ambil error validasi pertama dan terjemahkan ke pesan yang bisa dibaca
		} else {
			writeError(w, http.StatusBadRequest, "request tidak valid")
		}
		return
	}

	token, err := h.service.Login(r.Context(), req)
	// → serahkan req ke service untuk diproses
	// → service yang cari user by email, verifikasi password, generate JWT token
	// → return string token kalau berhasil
	if err != nil {
		handleAuthError(w, err)
		// → ErrEmailAtauPasswordSalah → 401 Unauthorized
		return
	}

	writeJSON(w, http.StatusOK, dto.LoginResponse{Token: token})
	// → 200 OK untuk operasi login yang berhasil
	// → dto.LoginResponse{Token: token} → bungkus token dalam struct {"token": "eyJ..."}
	// → client simpan token ini dan kirim di header setiap request protected:
	//   Authorization: Bearer eyJ...
}

// handleAuthError → helper untuk mapping error auth ke HTTP status yang tepat
// → dipisah dari handleError di catatan_handler.go karena error auth punya status berbeda
// → handleError tidak tahu 409 Conflict dan 401 Unauthorized untuk credentials
// → kalau digabung: logic mapping jadi campur aduk antara catatan error dan auth error
func handleAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperror.ErrEmailSudahDipakai):
		writeError(w, http.StatusConflict, err.Error())
		// → ErrEmailSudahDipakai → 409 Conflict
		// → 409 khusus untuk kasus resource sudah ada / konflik data

	case errors.Is(err, apperror.ErrEmailAtauPasswordSalah):
		writeError(w, http.StatusUnauthorized, err.Error())
		// → ErrEmailAtauPasswordSalah → 401 Unauthorized
		// → pesan error sengaja tidak spesifik: "email atau password salah"
		// → attacker tidak bisa tahu mana yang salah — email tidak ada atau password salah

	case errors.Is(err, apperror.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
		// → ErrUnauthorized → 401 Unauthorized
		// → dipakai middleware Auth — tapi handleAuthError juga handle untuk konsistensi

	default:
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
		// → error tidak dikenal → 500 Internal Server Error
		// → tidak expose detail error ke client — detail ada di server log
	}
}

// Perbedaan auth_handler.go vs catatan_handler.go:
// → auth_handler.go pakai validate.Struct() — karena auth butuh validasi format ketat
//   (email format, password minimum length)
// → catatan_handler.go tidak pakai validate.Struct() — validasi diserahkan sepenuhnya ke service
// → auth_handler.go pakai handleAuthError — karena error auth punya HTTP status berbeda
// → catatan_handler.go pakai handleError — untuk error CRUD standar

// Pattern setiap function di auth_handler.go:
// 1. Parse input    → Decode JSON body ke request DTO
// 2. Validasi input → validate.Struct() untuk cek validate tag
// 3. Call service   → serahkan ke service, handler tidak proses apapun sendiri
// 4. Handle error   → handleAuthError() untuk mapping ke HTTP status yang tepat
// 5. Build response → writeJSON dengan dto mapper — tidak ada map[string]any manual
