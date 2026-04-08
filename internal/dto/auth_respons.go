package dto

// package dto → sama dengan file dto lainnya, satu package untuk semua DTO
// → file ini khusus untuk response DTO yang berkaitan dengan auth/user

import "time"

// import time → butuh time.Time untuk field CreatedAt
// → time.Time di-encode JSON menjadi format RFC3339: "2024-01-01T10:00:00Z"

// UserResponse → DTO untuk response data user ke client
// → dipakai oleh dto.ToUserResponse di auth_mapper.go
// → dipakai handler Register untuk return data user yang baru dibuat
// → konsisten dengan CatatanResponse — semua resource punya response DTO sendiri
// → kalau dihapus: auth_handler tidak bisa pakai mapper, harus tulis response manual
type UserResponse struct {
	ID int `json:"id"`
	// → id user yang di-generate database AUTO_INCREMENT
	// → dikirim ke client sebagai referensi identitas user

	Nama string `json:"nama"`
	// → nama user yang didaftarkan
	// → json:"nama" → key di JSON response adalah "nama", bukan "Nama"

	Email string `json:"email"`
	// → email user yang didaftarkan
	// → json:"email" → key di JSON response adalah "email"

	CreatedAt time.Time `json:"created_at"`
	// → waktu user dibuat — diisi database otomatis via DEFAULT CURRENT_TIMESTAMP
	// → json:"created_at" → key di JSON response adalah "created_at"
}

// Kenapa Password tidak ada di struct ini:
// → modul.User punya field Password yang berisi bcrypt hash
// → hash ini hanya dibutuhkan saat verifikasi login di service
// → dengan tidak menyertakan Password di UserResponse:
//   tidak mungkin password bocor ke response secara tidak sengaja
//   meski developer lupa filter password, struct ini sudah proteksi dari awal

// Perbandingan UserResponse vs CatatanResponse:
// → keduanya tidak expose field sensitif (Password, dll)
// → keduanya pakai json tag lowercase dengan underscore
// → keduanya adalah pure data struct tanpa method apapun
