package dto

// package dto → sama dengan file dto lainnya, satu package untuk semua DTO
// → pagination dipisah file sendiri karena bisa dipakai oleh semua resource, bukan hanya catatan

// PaginationQuery → struct untuk menampung parameter pagination dari query string URL
// → dipakai handler untuk parse ?page=1&limit=10 dari URL
// → dipakai service sebagai parameter input fungsi List
// → tidak ada JSON tag karena ini bukan dari JSON body — dari query string URL
// → tidak ada validate tag karena validasi dan default value dihandle manual di handler
type PaginationQuery struct {
	Page int
	// → halaman ke berapa yang diminta client
	// → default 1 kalau client tidak kirim ?page=
	// → kalau client kirim ?page=0 atau negatif → handler paksa jadi 1

	Limit int
	// → berapa item per halaman yang diminta client
	// → default 10 kalau client tidak kirim ?limit=
	// → kalau client kirim ?limit=0 atau negatif → handler paksa jadi 10
	// → kalau client kirim ?limit=200 → handler paksa jadi 100 (maksimal)
	// → batas maksimal 100 untuk proteksi — cegah client minta semua data sekaligus
}

// PaginatedResponse → struct pembungkus response untuk semua operasi List
// → semua response List dibungkus struct ini — standarisasi format response
// → client selalu terima format yang sama: {data: [...], meta: {...}}
// → kalau tidak pakai wrapper ini: response hanya array [] tanpa informasi pagination
//   client tidak tahu total data, tidak tahu ada berapa halaman
type PaginatedResponse struct {
	Data interface{} `json:"data"`
	// interface{} → tipe kosong yang bisa menampung nilai apapun
	// → dipakai supaya PaginatedResponse bisa dipakai untuk semua resource
	//   sekarang: []CatatanResponse
	//   nanti kalau ada resource lain: []UserResponse, []ProductResponse, dll
	// → kalau pakai []CatatanResponse: struct ini hanya bisa dipakai untuk catatan
	// → json:"data" → client terima key "data" berisi array

	Meta MetaData `json:"meta"`
	// → informasi pagination dikirim bersama data
	// → json:"meta" → client terima key "meta" berisi object pagination
	// → client butuh ini untuk tahu: total data berapa, sekarang di halaman berapa,
	//   masih ada halaman berikutnya atau tidak
}

// MetaData → struct yang berisi informasi pagination di dalam response
// → diisi oleh handler setelah dapat total dari service
// → client pakai informasi ini untuk navigasi halaman
type MetaData struct {
	Page int `json:"page"`
	// → halaman yang sedang ditampilkan sekarang
	// → client pakai ini untuk tahu posisi sekarang di halaman berapa

	Limit int `json:"limit"`
	// → jumlah item per halaman yang digunakan
	// → client pakai ini untuk konfirmasi limit yang dipakai server

	Total int `json:"total"`
	// → total semua item di database sesuai filter (arsip/tidak arsip)
	// → client pakai ini untuk hitung total halaman: math.Ceil(total / limit)
	// → contoh: total=25, limit=10 → ada 3 halaman
}

// Contoh response yang dihasilkan:
// {
//   "data": [
//     {"id": 1, "judul": "catatan pertama", ...},
//     {"id": 2, "judul": "catatan kedua", ...}
//   ],
//   "meta": {
//     "page": 1,
//     "limit": 10,
//     "total": 25
//   }
// }

// Pattern file pagination:
// 1. PaginationQuery   → input dari client via query param — tidak ada JSON/validate tag
// 2. PaginatedResponse → output ke client — Data pakai interface{} agar reusable
// 3. MetaData          → informasi pagination — page, limit, total
// 4. Handler yang isi PaginationQuery dari URL, validasi default, dan build PaginatedResponse
// 5. Service yang return ([]data, total, error) — total dipakai handler untuk isi MetaData
