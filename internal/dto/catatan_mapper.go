package dto

// package dto → mapper ada di package yang sama dengan request dan response DTO
// → karena mapper adalah bagian dari lapisan DTO — tugasnya konversi antar bentuk data

import "catatan_app/internal/modul"

// import modul → butuh tipe modul.Catatan sebagai input konversi
// → mapper adalah jembatan antara domain layer (modul) dan DTO layer
// → kalau dihapus: kompilasi error karena modul.Catatan tidak dikenali

// ToCatatanResponse → konversi satu domain object ke response DTO
// → menerima pointer *modul.Catatan karena repository selalu return pointer
// → return CatatanResponse sebagai nilai, bukan pointer — response DTO tidak perlu pointer
// → dipakai handler untuk operasi yang return satu data: Create, GetByID, Update, Arsip, Unarsip
// → kalau dihapus: handler harus konversi manual di setiap function — duplikasi kode
func ToCatatanResponse(catatan *modul.Catatan) CatatanResponse {
	return CatatanResponse{
		// → mapping satu per satu dari domain object ke response struct
		// → kenapa tidak langsung return catatan: karena tipenya berbeda
		//   modul.Catatan ≠ CatatanResponse meski fieldnya sama
		// → kalau field baru ditambah di CatatanResponse tapi lupa di-mapping di sini
		//   field tersebut akan selalu zero value (0, "", false) di response
		ID:        catatan.ID,
		Judul:     catatan.Judul,
		Isi:       catatan.Isi,
		Arsip:     catatan.Arsip,
		CreatedAt: catatan.CreatedAt,
	}
}

// ToCatatanResponses → konversi slice domain object ke slice response DTO
// → menerima []modul.Catatan bukan []*modul.Catatan karena repository return slice nilai
// → return []CatatanResponse — slice response untuk dikirim ke client
// → dipakai handler hanya untuk operasi List yang return banyak data
// → kalau dihapus: handler harus loop dan konversi manual di function List
func ToCatatanResponses(notes []modul.Catatan) []CatatanResponse {
	responses := make([]CatatanResponse, len(notes))
	// make([]CatatanResponse, len(notes)) → alokasi slice dengan kapasitas sama dengan input
	// → lebih efisien dari append karena ukuran sudah diketahui di awal
	// → kalau pakai var responses []CatatanResponse + append: alokasi memory berkali-kali
	//   setiap append bisa trigger resize slice — tidak efisien untuk data besar

	for i, catatan := range notes {
		// range notes → iterasi setiap item di slice
		// i → index posisi di slice
		// catatan → copy nilai modul.Catatan di index i
		responses[i] = ToCatatanResponse(&catatan)
		// &catatan → ambil alamat memory copy catatan untuk dijadikan pointer
		// → ToCatatanResponse butuh *modul.Catatan, bukan modul.Catatan
		// → kenapa tidak langsung &notes[i]: &catatan dan &notes[i] sama hasilnya di sini
		//   tapi &catatan lebih idiomatis dalam Go
	}
	return responses
}

// Kenapa mapper dipisah file tersendiri:
// → single responsibility — mapper hanya punya satu tugas: konversi domain ke DTO
// → kalau konversi ditulis di handler: handler jadi terlalu panjang dan susah dibaca
// → kalau butuh ubah format response: cukup ubah di mapper, tidak perlu ubah handler

// Pattern file mapper:
// 1. Function single  → konversi *modul.X ke XResponse — dipakai untuk operasi satu data
// 2. Function slice   → konversi []modul.X ke []XResponse — dipakai untuk operasi list
// 3. Function slice selalu panggil function single di dalamnya — tidak duplikasi logika konversi
// 4. Tidak ada logic apapun selain mapping field — validasi dan transformasi bukan tugas mapper
