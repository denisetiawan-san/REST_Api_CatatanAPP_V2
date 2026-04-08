package dto

// package dto → sama dengan catatan_request.go, satu package untuk semua DTO
// → auth request dan catatan request dipisah file agar setiap resource punya DTO sendiri

// RegisterRequest → struct yang mendefinisikan bentuk data yang masuk dari client saat REGISTER
// → hanya field yang dibutuhkan untuk buat akun baru
// → id dan created_at tidak ada — di-generate otomatis oleh MySQL
// → password ada di sini karena client perlu kirim password saat register
//   tapi setelah masuk service, password langsung di-hash bcrypt sebelum disimpan ke DB
type RegisterRequest struct {
	Nama string `json:"nama"     validate:"required,min=2,max=100"`
	// json:"nama"         → mapping key "nama" dari JSON body
	// validate:"required" → wajib ada dan tidak boleh kosong
	// validate:"min=2"    → minimal 2 karakter — nama 1 huruf tidak masuk akal
	// validate:"max=100"  → sinkron dengan VARCHAR(100) di tabel users

	Email string `json:"email"    validate:"required,email"`
	// json:"email"        → mapping key "email" dari JSON body
	// validate:"required" → wajib ada
	// validate:"email"    → format harus valid email (ada @, ada domain)
	//                     → go-playground/validator cek format, bukan cek email benar-benar ada
	// → uniqueness email tidak dicek di sini — itu validasi bisnis, dicek di auth_service.go

	Password string `json:"password" validate:"required,min=8"`
	// json:"password"     → mapping key "password" dari JSON body
	// validate:"required" → wajib ada
	// validate:"min=8"    → minimal 8 karakter — standar keamanan password minimum
	// → tidak ada max karena bcrypt sendiri punya batas 72 karakter, tapi tidak perlu expose ke client
	// → password plain text di struct ini hanya hidup sampai service — langsung di-hash bcrypt
}

// LoginRequest → struct yang mendefinisikan bentuk data yang masuk dari client saat LOGIN
// → lebih simpel dari RegisterRequest — hanya butuh email dan password untuk verifikasi
// → Nama tidak ada karena tidak dibutuhkan untuk login
type LoginRequest struct {
	Email string `json:"email"    validate:"required,email"`
	// → sama dengan RegisterRequest, format email harus valid

	Password string `json:"password" validate:"required"`
	// validate:"required" → wajib ada
	// → tidak ada min=8 di sini karena ini verifikasi, bukan pembuatan password
	//   kalau user kirim password pendek, bcrypt.CompareHashAndPassword() akan return salah
	//   tidak perlu validasi panjang di sini
}

// LoginResponse → struct yang mendefinisikan bentuk data yang keluar ke client setelah LOGIN berhasil
// → ini satu-satunya tempat token JWT dikirim ke client
// → kenapa dipisah dari LoginRequest: karena request adalah data masuk, response adalah data keluar
//   keduanya punya tanggung jawab berbeda meski untuk operasi yang sama
// → tidak ada field user (id, nama, email) di sini — client cukup simpan token
//   semua informasi user sudah ada di dalam JWT claims
type LoginResponse struct {
	Token string `json:"token"`
	// json:"token" → client terima {"token": "eyJhbGci..."}
	// → token ini yang akan dikirim client di header setiap request protected
	//   Authorization: Bearer eyJhbGci...
}

// Perbedaan auth_request.go vs catatan_request.go:
// → catatan_request.go  → tidak ada LoginResponse karena catatan tidak return token
// → auth_request.go     → ada LoginResponse karena login perlu return token ke client
// → keduanya dipisah file agar setiap resource punya DTO sendiri yang tidak saling ganggu

// Pattern file auth DTO:
// 1. RegisterRequest  → semua field yang dibutuhkan untuk buat akun baru + validate tag ketat
// 2. LoginRequest     → hanya email dan password, validate lebih longgar dari Register
// 3. LoginResponse    → hanya token, tidak return data user — informasi user ada di dalam token
// 4. Password selalu ada di request DTO — tapi hanya hidup sampai service sebelum di-hash
// 5. Password tidak pernah ada di response DTO — client tidak boleh terima password dalam bentuk apapun
