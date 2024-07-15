# Gunakan gambar dasar yang ringan
FROM golang:1.20-alpine AS builder

# Set environment variables
ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# Buat direktori kerja untuk aplikasi
WORKDIR /app

# Copy semua file ke dalam direktori kerja
COPY . .

# Install dependencies dan build aplikasi
RUN go mod tidy
RUN go build -o main main.go

# Gambar akhir untuk menjalankan aplikasi
FROM alpine:latest

# Install ca-certificates agar kita bisa membuat koneksi https
RUN apk --no-cache add ca-certificates

# Set direktori kerja di dalam kontainer
WORKDIR /root/

# Copy file yang sudah di build dari gambar builder
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

# Expose port yang akan digunakan oleh aplikasi
EXPOSE 7860

# Command untuk menjalankan aplikasi
CMD ["./main", "-addr", "0.0.0.0:7860"]
