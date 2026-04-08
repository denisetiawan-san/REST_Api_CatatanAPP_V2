package apperror

// package apperror → package khusus untuk definisi semua error di project ini
// → diimport oleh repository, service, middleware, dan handler
// → kalau dihapus: semua layer tidak punya standar error yang sama

import "errors"

// import "errors" → butuh fungsi errors.New() untuk membuat sentinel error
// → kalau dihapus: kompilasi error karena errors.New() tidak dikenali

// var ( ... ) → grouping deklarasi semua sentinel error dalam satu blok
// → sentinel error = variabel error yang nilainya tetap dan bisa dikenali via errors.Is()
// → kenapa tidak pakai string biasa: karena string comparison rapuh dan mudah typo
// → dengan errors.Is() perbandingan berdasarkan identitas variabel, bukan isi string
var (
	// ErrNotFound → direturn oleh repository ketika rows.Scan() dapat sql.ErrNoRows
	// → service teruskan ke handler tanpa diubah
	// → handler mapping ke HTTP 404 Not Found
	// → kalau dihapus: repository tidak tahu cara beritahu handler bahwa data tidak ada
	ErrNotFound = errors.New("catatan tidak ditemukan")

	// ErrInvalidID → direturn oleh service ketika id yang diterima <= 0
	// → validasi ini ada di service karena ini validasi bisnis, bukan validasi format
	// → handler mapping ke HTTP 400 Bad Request
	// → kalau dihapus: request dengan id negatif atau nol bisa sampai ke repository dan query database
	ErrInvalidID = errors.New("id tidak valid")

	// ErrBadRequest → direturn oleh service ketika request tidak memenuhi syarat bisnis
	// → contoh: judul kosong setelah TrimSpace
	// → handler mapping ke HTTP 400 Bad Request
	// → kalau dihapus: tidak ada cara standar untuk beritahu handler bahwa input tidak valid secara bisnis
	ErrBadRequest = errors.New("request tidak valid")

	// ErrEmailSudahDipakai → direturn oleh auth_service ketika email sudah terdaftar di database
	// → dicek sebelum INSERT user baru
	// → handler mapping ke HTTP 409 Conflict
	// → kalau dihapus: tidak ada cara bedakan error duplikasi email dari error lainnya
	ErrEmailSudahDipakai = errors.New("email sudah dipakai")

	// ErrEmailAtauPasswordSalah → direturn oleh auth_service ketika login gagal
	// → sengaja pesannya digabung "email atau password salah" — tidak boleh beritahu
	//   client mana yang salah, karena itu informasi bagi attacker untuk enumerate email
	// → handler mapping ke HTTP 401 Unauthorized
	// → kalau dihapus: tidak ada cara standar untuk beritahu handler bahwa credentials salah
	ErrEmailAtauPasswordSalah = errors.New("email atau password salah")

	// ErrUnauthorized → direturn oleh middleware auth ketika token tidak ada, expired, atau invalid
	// → handler mapping ke HTTP 401 Unauthorized
	// → kalau dihapus: middleware tidak punya error standar untuk beritahu handler bahwa akses ditolak
	ErrUnauthorized = errors.New("unauthorized")
)

// Pattern file sentinel error:
// 1. Satu file khusus — semua error didefinisikan di satu tempat
// 2. Pakai errors.New() — bukan string biasa, agar bisa dikenali via errors.Is()
// 3. Nama error deskriptif — langsung tahu konteksnya dari namanya
// 4. Tidak ada logic — hanya definisi variabel error
// 5. Dipakai searah: repository → service → handler
//    → repository return ErrNotFound
//    → service teruskan atau return ErrInvalidID / ErrBadRequest
//    → handler pakai errors.Is() untuk mapping ke HTTP status
//
// Kenapa pesan ErrEmailAtauPasswordSalah digabung:
// → security by design — attacker tidak bisa tahu apakah email terdaftar atau tidak
// → kalau dipisah jadi ErrEmailTidakDitemukan dan ErrPasswordSalah
//   → attacker bisa enumerate: coba email sampai dapat "password salah" = email valid
