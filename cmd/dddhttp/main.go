package main

import (
	"encoding/json"
	"io"
	"time"

	"log"
	"net/http"

	"github.com/kyburz-switzerland-ag/tachoparser/pkg/decoder"
)

func uploadHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("Received request")

	card := false
	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	//r.ParseForm()
	format := r.FormValue("format")

	if format == "card" {
		card = true
	} else if format == "vu" {
		card = false
	} else {
		log.Printf("Invalid format: %s", format)
		http.Error(w, "Invalid format", http.StatusBadRequest)
		return
	}

	log.Printf("Format: %s", format)

	log.Printf("try to get file")
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	log.Printf("try to read file")
	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading the file", http.StatusInternalServerError)
		return
	}

	log.Printf("Received file with %d bytes", len(data))

	var jsonData []byte

	if card {
		log.Printf("Try to use the card decoder")
		var err error
		var c decoder.Card
		_, err = decoder.UnmarshalTLV(data, &c)
		if err != nil {
			log.Fatalf("error: could not parse card: %v", err)
		}
		jsonData, err = json.Marshal(c)
		if err != nil {
			log.Fatalf("error: could not marshal card: %v", err)
		}
	} else {
		log.Printf("Try to use the vu decoder")
		var err error
		var v decoder.Vu
		_, err = decoder.UnmarshalTV(data, &v)
		if err != nil {
			log.Fatalf("error: could not parse vu data: %v", err)
		}
		jsonData, err = json.Marshal(v)
		if err != nil {
			log.Fatalf("error: could not marshal vu data: %v", err)
		}
	}

	log.Println("Sending response")

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func main() {

	// http.HandleFunc("/upload", uploadHandler)
	log.Println("Starting server on :8080")
	// if err := http.ListenAndServe(":8080", nil); err != nil {
	// 	log.Fatalf("Could not start server: %s\n", err.Error())
	// }

	srv := &http.Server{
		Addr:    ":8080",
		Handler: http.HandlerFunc(uploadHandler),

		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Fatal(srv.ListenAndServe())
}
