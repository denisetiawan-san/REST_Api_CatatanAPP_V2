RESTful API dengan Layered Architecture (N-Tier Architecture) menggunakan Repository Pattern dan Dependency Injection, dilengkapi JWT Authentication, Docker containerization, dan Nginx sebagai reverse proxy dengan rate limiting.

Alurnya mengikuti pola layered architecture:
Request → Nginx → Middleware (CORS, Logger, Recovery) → Router → Middleware Auth → Handler → Service → Repository → Database
Response ←───────────────────────────────────────────────────────────────────────────────────────────────────────────────
Penjelasan simpelnya:
Nginx → rate limiting, reverse proxy dari port 80 ke port 8080
Middleware → CORS, logging setiap request, recovery dari panic
Router → penerima request, tentukan ke handler mana
Auth → validasi JWT token sebelum masuk ke handler
Handler → parse request, validasi input, panggil service
Service → business logic (validasi bisnis, transformasi data)
Repository → komunikasi langsung ke database
DTO → bentuk data yang masuk dan keluar dari API
Model → representasi tabel database di Go

LIST API LENGKAP
Public Route (tanpa token)
Method Endpoint Deskripsi Success
POST /api/v1/auth/register Daftar user baru 201 Created
POST /api/v1/auth/login Login, dapat JWT token 200 OK

Protected Route (butuh Authorization: Bearer <token>)
Method Endpoint Deskripsi Success
POST /api/v1/catatan Buat catatan baru 201 Created
GET /api/v1/catatan Ambil semua catatan aktif 200 OK
GET /api/v1/catatan?arsip=true Ambil catatan yang diarsip 200 OK
GET /api/v1/catatan?arsip=false Ambil catatan tidak diarsip 200 OK
GET /api/v1/catatan?page=1&limit=10 Ambil catatan dengan pagination 200 OK
GET /api/v1/catatan/{id} Ambil satu catatan by ID 200 OK
PUT /api/v1/catatan/{id} Update catatan by ID 200 OK
PATCH /api/v1/catatan/{id}/arsip Arsipkan catatan 200 OK
PATCH /api/v1/catatan/{id}/unarsip Kembalikan dari arsip 200 OK
DELETE /api/v1/catatan/{id} Hapus catatan by ID 204 No Content

Error Response
400 Bad Request → input tidak valid / format salah / validasi gagal
401 Unauthorized → token tidak ada / expired / invalid / login gagal
404 Not Found → data tidak ditemukan
405 Method Not Allowed → HTTP method tidak didukung di endpoint tersebut
409 Conflict → email sudah terdaftar saat register
429 Too Many Requests → melebihi rate limit Nginx (10 req/detik per IP)
500 Internal Server Error → kesalahan tidak terduga di server

Query Param yang Didukung di GET /api/v1/catatan
?arsip=true → tampilkan catatan yang diarsip
?arsip=false → tampilkan catatan yang tidak diarsip
?page=1 → halaman ke berapa (default 1)
?limit=10 → jumlah item per halaman (default 10, maksimal 100)

Kombinasi:
?arsip=true&page=1&limit=5 → catatan arsip, halaman 1, 5 item per halaman

==========================================================================================

Arsitektur & Pattern:
Layered Architecture → handler, service, repository terpisah jelas
Repository Pattern → abstraksi database via interface
Dependency Injection → semua dependency inject lewat constructor
Interface-based Design → depend ke interface, bukan concrete struct
Sentinel Error Pattern → satu sumber kebenaran untuk semua error
DTO Pattern → domain model terpisah dari API contract
Graceful Shutdown → server mati dengan bersih, tidak paksa
Connection Pooling → koneksi database dikelola efisien

Keamanan:
JWT Authentication → stateless auth dengan signed token
Bcrypt Password Hashing → password tidak pernah disimpan plain text
Request Validation → input divalidasi sebelum masuk ke logic
Middleware Auth → route diproteksi di level routing

Kualitas:
Structured Logging → setiap request tercatat dengan zerolog
Panic Recovery → server tidak mati karena satu request error
Unit & Integration Test → 21 test membuktikan code benar-benar bekerja
CORS → akses dikontrol per origin

Skalabilitas:
Pagination → siap handle data jutaan baris
API Versioning → breaking change bisa rilis tanpa ganggu client lama
Rate Limiting → terlindungi dari request berlebihan via Nginx

Infrastruktur:
Docker Containerization → environment konsisten di semua mesin
Nginx Reverse Proxy → layer pertama sebelum masuk ke aplikasi
Environment Config → tidak ada hardcode, semua dari .env

=============================================================================================

Catatan API

RESTful API untuk manajemen catatan dengan fitur authentication (JWT), CRUD, middleware, dan containerized environment menggunakan Docker.

Getting Started

Ikuti langkah berikut untuk menjalankan project di local environment.

1. Clone Repository

git clone https://github.com/username/REST_Api_CatatanAPP_V2.git
cd REST_Api_CatatanAPP_V2

2. Setup Environment Variable

Copy file .env.example menjadi .env:

cp .env.example .env

Atau jika di Windows (PowerShell):

copy .env.example .env

3. Jalankan Project dengan Docker

Build dan jalankan semua service:

docker compose up --build

Service yang akan berjalan:

- Go API (app)
- MySQL database
- Nginx (reverse proxy)

4. Verifikasi Aplikasi Berjalan

Akses API melalui:

http://localhost:8080

Atau jika menggunakan Nginx:

http://localhost

5. Test Endpoint (contoh)

Register

curl -X POST http://localhost:8080/register \
-H "Content-Type: application/json" \
-d '{"nama":"deni","email":"deni@mail.com","password":"123456"}'

Login

curl -X POST http://localhost:8080/login \
-H "Content-Type: application/json" \
-d '{"email":"deni@mail.com","password":"123456"}'

6. Menjalankan Unit Test

go test ./...

7. Stop Project

docker compose down

Tech Stack

- Golang
- MySQL
- Docker & Docker Compose
- Nginx
- JWT Authentication

project by : denisetiawan-san
