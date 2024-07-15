package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"html"
	"html/template"
	"image"
	"image/color"
	"image/png"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"time"
)

func usage() {
	fmt.Fprintf(os.Stderr, "usage: helloserver [options]\n")
	flag.PrintDefaults()
	os.Exit(2)
}

var (
	// greeting = flag.String("g", "Hello", "Greet with `greeting`")
	addr     = flag.String("addr", "localhost:7860", "address to serve")
)

func main() {
	// Parse flags.
	flag.Usage = usage
	flag.Parse()

	// Parse and validate arguments (none).
	args := flag.Args()
	if len(args) != 0 {
		usage()
	}

	// Register handlers.
	http.HandleFunc("/", greet)
	http.HandleFunc("/version", version)
	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/extract", extractHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Printf("serving http://%s\n", *addr)
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func version(w http.ResponseWriter, r *http.Request) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		http.Error(w, "no build information available", 500)
		return
	}

	fmt.Fprintf(w, "<!DOCTYPE html>\n<pre>\n")
	fmt.Fprintf(w, "%s\n", html.EscapeString(info.String()))
}

func greet(w http.ResponseWriter, r *http.Request) {
	t, _ := template.ParseFiles("templates/index.html")
	t.Execute(w, nil)
}

func pesanKeBinary(pesan string) string {
	binaryPesan := ""
	for _, c := range pesan {
		binaryPesan += fmt.Sprintf("%08b", c)
	}
	return binaryPesan
}

func intToBinaryString(angka int) string {
	return fmt.Sprintf("%08b", angka)
}

func embedLSB(img image.Image, message string) image.Image {
	bounds := img.Bounds()
	binaryPesan := message + "1111110011011011101000" // Delimiter untuk menandai akhir pesan
	panjangPesan := len(binaryPesan)
	dataIndex := 0

	newImg := image.NewRGBA(bounds)
	for i := 0; i < bounds.Dx(); i++ {
		for j := 0; j < bounds.Dy(); j++ {
			r, g, b, a := img.At(i, j).RGBA()
			pixel := [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}

			for k := 0; k < 3; k++ { // Loop untuk setiap channel warna (RGB)
				if dataIndex < panjangPesan {
					isiPixel := intToBinaryString(int(pixel[k]))
					isiPixel = isiPixel[:7] + string(binaryPesan[dataIndex])
					pixelUint64, _ := strconv.ParseUint(isiPixel, 2, 8)
					pixel[k] = uint8(pixelUint64)
					dataIndex++
				}
			}

			newImg.Set(i, j, color.RGBA{pixel[0], pixel[1], pixel[2], uint8(a >> 8)})
		}
	}
	return newImg
}

func extractLSB(img image.Image) string {
	bounds := img.Bounds()
	binaryMessage := ""
	delimiter := "1111110011011011101000"
	dataIndex := 0

	for i := 0; i < bounds.Dx(); i++ {
		for j := 0; j < bounds.Dy(); j++ {
			r, g, b, _ := img.At(i, j).RGBA()
			pixel := [3]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)}

			for k := 0; k < 3; k++ { // Loop untuk setiap channel warna (RGB)
				binaryPixel := intToBinaryString(int(pixel[k]))
				binaryMessage += string(binaryPixel[7])
				dataIndex++

				// Check if we have reached the delimiter
				if len(binaryMessage) >= len(delimiter) && binaryMessage[len(binaryMessage)-len(delimiter):] == delimiter {
					binaryMessage = binaryMessage[:len(binaryMessage)-len(delimiter)]
					return binaryToMessage(binaryMessage)
				}
			}
		}
	}
	return binaryToMessage(binaryMessage)
}

func binaryToMessage(binary string) string {
	message := ""
	for i := 0; i+8 <= len(binary); i += 8 {
		parsedChar, _ := strconv.ParseUint(binary[i:i+8], 2, 8)
		message += string(parsedChar)
	}
	return message
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		start := time.Now()

		file, _, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		message := r.FormValue("message")
		binaryPesan := pesanKeBinary(message)

		newImg := embedLSB(img, binaryPesan)
		embedDuration := time.Since(start).Seconds()

		var buf bytes.Buffer
		err = png.Encode(&buf, newImg)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		imgBase64Str := base64.StdEncoding.EncodeToString(buf.Bytes())
		data := map[string]interface{}{
			"Image":         imgBase64Str,
			"EmbedDuration": fmt.Sprintf("%.2f detik", embedDuration),
		}
		t, _ := template.ParseFiles("templates/index.html")
		t.Execute(w, data)
	} else {
		http.ServeFile(w, r, "templates/index.html")
	}
}

func extractHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		start := time.Now()

		file, _, err := r.FormFile("image")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		img, _, err := image.Decode(file)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		message := extractLSB(img)
		extractDuration := time.Since(start).Seconds()

		data := map[string]interface{}{
			"Message":         message,
			"ExtractDuration": fmt.Sprintf("%.2f detik", extractDuration),
		}
		t, _ := template.ParseFiles("templates/index.html")
		t.Execute(w, data)
	} else {
		http.ServeFile(w, r, "templates/index.html")
	}
}
