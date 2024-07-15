# Menggunakan image Golang versi 1.22.4
FROM golang:1.22.4

# Set working directory di dalam container
WORKDIR /app

# Menyalin isi dari direktori proyek ke dalam container di /app
COPY . .

# Install dependencies jika ada
# RUN go mod download

# Build aplikasi Go
RUN go build -o main .

# Menjalankan aplikasi Go pada port 7860
CMD ["./main", "-port", "7860"]