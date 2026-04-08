package dto

// package dto → mapper ada di package yang sama dengan DTO lainnya
// → konsisten dengan catatan_mapper.go yang juga ada di package dto

import "catatan_app/internal/modul"

// import modul → butuh tipe modul.User sebagai input konversi
// → mapper adalah jembatan antara domain layer (modul) dan DTO layer (dto)
// → kalau dihapus: kompilasi error karena modul.User tidak dikenali

// ToUserResponse → konversi satu domain object user ke response DTO
// → menerima pointer *modul.User karena repository selalu return pointer
// → return UserResponse sebagai nilai, bukan pointer — response DTO tidak perlu pointer
// → dipakai auth_handler Register: writeJSON(w, http.StatusCreated, dto.ToUserResponse(user))
// → konsisten dengan ToCatatanResponse di catatan_mapper.go
// → kalau dihapus: handler harus konversi manual — tidak konsisten dengan pattern project
func ToUserResponse(user *modul.User) UserResponse {
	return UserResponse{
		ID: user.ID,
		// → mapping field ID dari domain object ke DTO

		Nama: user.Nama,
		// → mapping field Nama

		Email: user.Email,
		// → mapping field Email

		CreatedAt: user.CreatedAt,
		// → mapping field CreatedAt

		// → Password sengaja tidak di-mapping
		// → modul.User punya field Password tapi UserResponse tidak
		// → ini yang memastikan password hash tidak pernah bocor ke response
	}
}

// Kenapa mapper dipisah file tersendiri dari response DTO:
// → single responsibility — auth_response.go definisi struct, auth_mapper.go konversi
// → konsisten dengan catatan_respons.go dan catatan_mapper.go yang juga dipisah
// → kalau butuh ubah format response: cukup ubah di auth_response.go
// → kalau butuh ubah cara konversi: cukup ubah di auth_mapper.go

// Pattern file mapper — konsisten di semua resource:
// 1. Function single → konversi *modul.X ke XResponse — untuk operasi satu data
// 2. Tidak ada logic apapun selain mapping field
// 3. Password dan field sensitif tidak pernah di-mapping ke response DTO
