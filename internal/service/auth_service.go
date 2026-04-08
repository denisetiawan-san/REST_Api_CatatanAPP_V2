package service

// package service → sama dengan catatan_service.go, satu package untuk semua service
// → implementasi konkret dari interface AuthSvc yang didefinisikan di interface.go

import (
	"catatan_app/internal/apperror"
	// import apperror → butuh ErrEmailSudahDipakai dan ErrEmailAtauPasswordSalah

	"catatan_app/internal/dto"
	// import dto → butuh dto.RegisterRequest dan dto.LoginRequest sebagai parameter input

	"catatan_app/internal/modul"
	// import modul → butuh modul.User untuk bangun domain object

	"catatan_app/internal/repository"
	// import repository → butuh repository.UserRepo (interface) sebagai dependency

	"context"
	"errors"

	// import errors → butuh errors.Is() untuk cek jenis error dari repository

	"os"
	// import os → butuh os.Getenv() untuk baca JWT_SECRET dan JWT_EXPIRED_HOURS dari .env

	"strconv"
	// import strconv → butuh strconv.Atoi() untuk konversi string JWT_EXPIRED_HOURS ke int

	"time"
	// import time → butuh time.Now() dan time.Duration untuk hitung waktu expired token

	"github.com/golang-jwt/jwt/v5"
	// import jwt → butuh jwt.MapClaims, jwt.NewWithClaims, jwt.SigningMethodHS256
	// → untuk generate JWT token yang ditandatangani dengan secret key

	"golang.org/x/crypto/bcrypt"
	// import bcrypt → butuh bcrypt.GenerateFromPassword dan bcrypt.CompareHashAndPassword
	// → untuk hash password saat register dan verifikasi password saat login
)

// AuthService → struct konkret yang mengimplementasikan interface AuthSvc
// → memegang repository.UserRepo sebagai dependency — interface, bukan struct konkret
type AuthService struct {
	repo repository.UserRepo
	// → interface UserRepo, bukan *UserRepository
	// → di unit test bisa di-inject MockUserRepo tanpa database nyata
}

// NewAuthService → constructor untuk membuat instance AuthService
// → menerima repository.UserRepo dari main.go dan inject ke struct
func NewAuthService(repo repository.UserRepo) *AuthService {
	return &AuthService{repo: repo}
}

// compile-time check → pastikan AuthService memenuhi semua method di interface AuthSvc
var _ AuthSvc = (*AuthService)(nil)

// Register → implementasi kontrak Register dari interface AuthSvc
// → pintu masuk logic bisnis untuk daftar user baru
// → flow: cek email duplikat → hash password → simpan user baru
func (s *AuthService) Register(ctx context.Context, req dto.RegisterRequest) (*modul.User, error) {
	existing, err := s.repo.GetByEmail(ctx, req.Email)
	// → cek apakah email sudah terdaftar di database
	// → GetByEmail return (*modul.User, error)
	// → kalau email ada: existing != nil, err = nil
	// → kalau email tidak ada: existing = nil, err = apperror.ErrNotFound

	if existing != nil {
		return nil, apperror.ErrEmailSudahDipakai
		// → email sudah ada di database → tolak registrasi
		// → handler mapping ErrEmailSudahDipakai ke HTTP 409 Conflict
		// → kenapa cek existing != nil, bukan errors.Is(err, nil):
		//   karena kalau email ada, err = nil dan existing berisi data user
		//   jadi cek existing != nil lebih tepat
	}
	if err != nil && !errors.Is(err, apperror.ErrNotFound) {
		return nil, err
		// → kalau err bukan ErrNotFound berarti error teknis database
		// → ErrNotFound adalah kondisi normal — email belum terdaftar → boleh lanjut register
		// → kenapa tidak langsung: if err != nil { return nil, err }
		//   karena ErrNotFound bukan error yang harus di-stop — itu kondisi yang diharapkan
	}
	// → sampai sini berarti email belum terdaftar → boleh lanjut

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	// bcrypt.GenerateFromPassword → hash password dengan algoritma bcrypt
	// []byte(req.Password) → konversi string password ke slice byte
	// bcrypt.DefaultCost → cost factor 10 — semakin tinggi semakin aman tapi semakin lambat
	// → hasil hash selalu berbeda meski input sama — bcrypt include random salt otomatis
	// → hash tidak bisa di-decrypt — hanya bisa di-verify dengan CompareHashAndPassword
	if err != nil {
		return nil, err
	}

	user := &modul.User{
		Nama:     req.Nama,
		Email:    req.Email,
		Password: string(hashedPassword),
		// → simpan hash, bukan plain text password
		// → string(hashedPassword) → konversi []byte hash ke string untuk disimpan di database
		// → ID dan CreatedAt tidak diisi — diisi otomatis oleh MySQL
	}

	return s.repo.Create(ctx, user)
	// → simpan user baru ke database
	// → return *modul.User lengkap setelah INSERT, termasuk id dan created_at
	// → handler yang akan putuskan field mana yang dikirim ke client
}

// Login → implementasi kontrak Login dari interface AuthSvc
// → pintu masuk logic bisnis untuk autentikasi user
// → flow: cari user by email → verifikasi password → generate JWT token
// → return string token, bukan *modul.User — tujuan login adalah dapat token
func (s *AuthService) Login(ctx context.Context, req dto.LoginRequest) (string, error) {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	// → cari user berdasarkan email yang dikirim client
	// → return *modul.User lengkap termasuk Password hash untuk diverifikasi
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return "", apperror.ErrEmailAtauPasswordSalah
			// → email tidak ditemukan → jangan bilang "email tidak ditemukan"
			// → selalu bilang "email atau password salah"
			// → security by design: attacker tidak bisa enumerate email valid
			//   kalau pesan berbeda antara "email salah" vs "password salah",
			//   attacker bisa coba-coba email sampai dapat pesan "password salah" = email valid
		}
		return "", err
		// → error teknis database — return as-is
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		// bcrypt.CompareHashAndPassword → bandingkan hash di database dengan plain text dari client
		// []byte(user.Password) → hash yang ada di database
		// []byte(req.Password)  → plain text password yang dikirim client
		// → bcrypt extract salt dari hash, hash ulang password input, bandingkan hasilnya
		// → kalau tidak sama → return error (bukan nil)
		return "", apperror.ErrEmailAtauPasswordSalah
		// → password salah → pesan error sama dengan email tidak ditemukan
		// → konsisten: apapun yang salah, client hanya tahu "email atau password salah"
	}
	// → sampai sini berarti email ada dan password benar → boleh generate token

	token, err := generateToken(user)
	// → buat JWT token berisi informasi user
	if err != nil {
		return "", err
	}

	return token, nil
	// → return token string ke handler
	// → handler akan bungkus dalam dto.LoginResponse{Token: token}
}

// generateToken → helper private untuk generate JWT token
// → private (huruf kecil) karena hanya dipakai di dalam package service
// → dipisah dari Login agar Login tidak terlalu panjang dan mudah dibaca
func generateToken(user *modul.User) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	// → baca secret key dari environment variable
	// → tidak hardcode di code karena secret key tidak boleh masuk ke git repository

	expiredHours, err := strconv.Atoi(os.Getenv("JWT_EXPIRED_HOURS"))
	// strconv.Atoi → konversi string "24" dari .env ke int 24
	if err != nil {
		expiredHours = 24
		// → kalau JWT_EXPIRED_HOURS tidak di-set atau bukan angka → default 24 jam
	}

	claims := jwt.MapClaims{
		"user_id": user.ID,
		// → simpan user_id di token — middleware auth akan extract ini
		// → dipakai handler untuk tahu request ini dari user mana

		"email": user.Email,
		// → simpan email di token — opsional, untuk informasi tambahan

		"exp": time.Now().Add(time.Duration(expiredHours) * time.Hour).Unix(),
		// time.Now() → waktu sekarang
		// .Add(time.Duration(expiredHours) * time.Hour) → tambah durasi expired
		// .Unix() → konversi ke Unix timestamp (int64) — format standar JWT
		// → setelah waktu ini token tidak valid lagi — middleware auth akan reject
	}
	// → jwt.MapClaims → map[string]interface{} untuk data yang disimpan di dalam token
	// → claims ini bisa dibaca siapapun yang punya token — jangan simpan data sensitif
	// → yang membuat token aman adalah signature, bukan enkripsi claims

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// jwt.SigningMethodHS256 → algoritma HMAC-SHA256 untuk sign token
	// → simetrik: sign dan verify pakai secret key yang sama
	// → middleware auth wajib cek algoritma ini — cegah algorithm confusion attack

	return token.SignedString([]byte(secret))
	// → sign token dengan secret key → hasilkan string JWT: header.payload.signature
	// → kalau secret berubah → semua token lama langsung invalid
}

// Perbedaan Register vs Login di level service:
// → Register → cek duplikasi email, hash password, simpan user, return *modul.User
// → Login    → cari user, verifikasi password, generate token, return string token
//
// Kenapa Login return string bukan *modul.User:
// → tujuan login adalah dapat token untuk akses protected route
// → client tidak butuh data user lengkap dari login — cukup token
// → kalau butuh data user, client bisa GET /api/v1/users/me (kalau endpoint itu ada)

// Pattern setiap function di auth_service.go:
// 1. Validasi bisnis    → cek duplikasi email (Register), cek credentials (Login)
// 2. Transformasi data  → hash password (Register), generate JWT token (Login)
// 3. Call repository    → Create untuk simpan user, GetByEmail untuk cari user
// 4. Return hasil       → *modul.User (Register) atau string token (Login)
