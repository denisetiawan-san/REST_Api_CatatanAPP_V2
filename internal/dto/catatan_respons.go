package dto

// package dto → sama dengan file dto lainnya, satu package untuk semua DTO

import "time"

// import "time" → butuh tipe time.Time untuk field CreatedAt
// → kalau dihapus: kompilasi error karena time.Time tidak dikenali

// CatatanResponse → struct yang mendefinisikan bentuk data yang keluar ke client
// → ini yang client terima setiap kali request GET, POST, PUT, PATCH
// → dipisah dari domain model (modul.Catatan) karena:
//   1. client tidak perlu tahu struktur internal database
//   2. kita punya kontrol penuh atas apa yang boleh keluar ke client
//   3. kalau domain model berubah, response tidak harus ikut berubah
type CatatanResponse struct {
	ID int `json:"id"`
	// json:"id" → client terima key "id" di JSON response
	// → kalau json tag dihapus: client terima key "ID" (kapital) — tidak konsisten dengan konvensi JSON

	Judul string `json:"judul"`
	// json:"judul" → client terima key "judul" di JSON response

	Isi string `json:"isi"`
	// json:"isi" → client terima key "isi" di JSON response

	Arsip bool `json:"arsip"`
	// json:"arsip" → client terima key "arsip" dengan nilai true atau false
	// → bool di Go di-encode ke JSON sebagai true/false, bukan 1/0

	CreatedAt time.Time `json:"created_at"`
	// json:"created_at" → client terima key "created_at" di JSON response
	// → time.Time di-encode ke JSON sebagai string format RFC3339
	//   contoh: "2024-01-15T10:30:00Z"
	// → kalau dihapus: client tidak tahu kapan catatan dibuat
}

// Kenapa tidak ada field Password di sini:
// → CatatanResponse untuk resource catatan, bukan user — memang tidak relevan
// → tapi prinsipnya sama: field sensitif tidak boleh masuk response DTO apapun
// → di UserResponse (kalau ada) pun Password tidak akan pernah ada

// Perbandingan CatatanResponse vs modul.Catatan:
// modul.Catatan    → tidak ada JSON tag, untuk komunikasi internal antar layer
// CatatanResponse  → ada JSON tag, untuk komunikasi keluar ke client
// → fieldnya sama persis di project ini, tapi tujuannya berbeda
// → di project yang lebih besar, response bisa sangat berbeda dari domain model
//   contoh: domain model punya foreign key user_id (int)
//           tapi response tampilkan nama user (string) hasil JOIN

// Pattern file response DTO:
// 1. Struct terpisah dari domain model — jangan return modul.Catatan langsung ke client
// 2. Hanya field yang boleh dilihat client — field sensitif tidak masuk sini
// 3. JSON tag wajib ada — konvensi JSON pakai snake_case, Go pakai PascalCase
// 4. Tipe data disesuaikan dengan yang mudah dikonsumsi client
//    → time.Time akan di-encode otomatis ke string RFC3339 oleh encoding/json
