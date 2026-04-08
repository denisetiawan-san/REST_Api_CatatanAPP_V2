# ─────────────────────────────────────────
# STAGE 1: BUILD
# ─────────────────────────────────────────
FROM golang:1.25 AS builder
# FROM golang:1.25 → pakai official Go image versi 1.25 sebagai base image stage 1
# AS builder → beri nama stage ini "builder" supaya bisa direferensikan di stage 2
# → image ini besar ~800MB tapi hanya dipakai untuk compile — tidak masuk ke image final

WORKDIR /app
# WORKDIR → set working directory di dalam container
# → semua perintah berikutnya (COPY, RUN) relatif dari /app
# → kalau /app belum ada → Docker buat otomatis

COPY go.mod go.sum ./
# → copy go.mod dan go.sum dulu, sebelum source code
# → kenapa dipisah dari COPY . . :
#   Docker build pakai layer cache — setiap instruksi adalah layer
#   kalau go.mod dan go.sum tidak berubah → layer ini di-cache → tidak perlu download ulang
#   kalau langsung COPY . . → setiap perubahan code apapun invalidate cache dependency
RUN go mod download
# → download semua dependency yang tercantum di go.mod
# → hasilnya di-cache di layer ini — hanya re-run kalau go.mod atau go.sum berubah

COPY . .
# → copy semua source code ke /app di dalam container
# → dilakukan SETELAH go mod download supaya cache dependency tidak invalidated setiap code berubah

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./server/main.go
# CGO_ENABLED=0 → matikan CGO (C Go) → hasilkan static binary
#   static binary = semua dependency sudah di-embed di dalam binary
#   tidak butuh C library (libc) di container tujuan
#   tanpa ini: binary tidak bisa jalan di alpine yang tidak punya libc lengkap
# GOOS=linux → compile untuk target OS Linux
#   penting kalau build dilakukan di Windows/Mac — binary harus untuk Linux container
# go build -o main → compile dan beri nama output binary "main"
# ./server/main.go → file entry point yang di-compile

# ─────────────────────────────────────────
# STAGE 2: RUN
# ─────────────────────────────────────────
FROM alpine:latest
# → stage baru dengan base image Alpine Linux — sangat minimal ~5MB
# → tidak ada Go toolchain, tidak ada source code, tidak ada dependency download
# → hanya berisi OS minimal dan binary yang kita copy dari stage builder
# → image final jauh lebih kecil: ~10MB vs ~800MB kalau pakai golang image

WORKDIR /app
# → set working directory di container final
# → sama dengan stage builder tapi ini container yang berbeda

COPY --from=builder /app/main .
# COPY --from=builder → copy file dari stage builder, bukan dari local machine
# /app/main → path binary di dalam stage builder
# . → copy ke WORKDIR saat ini (/app) di stage ini
# → ini inti dari multi-stage build: ambil hanya hasil compile, tinggalkan semua yang lain

# COPY .env .env
# → copy file .env dari local machine ke container
# → dibutuhkan karena aplikasi baca konfigurasi dari .env via godotenv.Load()
# → catatan: di production sebaiknya pakai environment variable langsung,
#   bukan file .env yang ter-embed di image

EXPOSE 8080
# → dokumentasi bahwa container ini listen di port 8080
# → EXPOSE tidak membuka port ke luar — hanya metadata
# → port mapping ke luar dilakukan di docker-compose.yml

CMD ["./main"]
# → perintah yang dijalankan saat container start
# → jalankan binary "main" yang ada di WORKDIR /app
# → pakai array form ["./main"] bukan string form "./main"
#   array form → tidak pakai shell, langsung jalankan binary — lebih efisien dan aman

# Kenapa multi-stage build:
# → stage 1 (builder): butuh Go toolchain yang besar untuk compile
# → stage 2 (runner) : hanya butuh binary hasil compile — tidak butuh Go toolchain
# → tanpa multi-stage: image final ~800MB karena bawa seluruh Go toolchain
# → dengan multi-stage: image final ~10MB karena hanya bawa binary
# → image lebih kecil = pull lebih cepat, deploy lebih cepat, attack surface lebih kecil


# ## Penjelasan Multi-Stage Build

# Stage 1 (builder) → image besar ~800MB, untuk compile Go
# Stage 2 (runner)  → image kecil ~10MB, hanya berisi binary

# Hasil akhir → container kamu hanya ~10MB

# # Jalankan semua container (Go app + MySQL + Nginx)
# docker compose up

# # Jalankan di background (tidak blocking terminal)
# docker compose up -d

# # Matikan semua container
# docker compose down

# # Lihat log semua container
# docker compose logs -f

# # Cek container yang sedang jalan
# docker ps