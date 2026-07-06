package main

import (
	"encoding/json"
	"io"
	"time"

	"log"
	"net/http"

	"github.com/traconiq/tachoparser/pkg/decoder"
	"github.com/traconiq/tachoparser/pkg/simple"
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

// infoHandler returns only the key identifiers of an uploaded file: the driver
// card number (for a "card" file) or the vehicle registration (for a "vu"
// file). It uses the lightweight byte-matching parser in pkg/simple, so it does
// not need the ERCA certificates and does not verify signatures.
func infoHandler(w http.ResponseWriter, r *http.Request) {

	log.Println("Received info request")

	if r.Method != "POST" {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	format := r.FormValue("format")
	if format != "card" && format != "vu" {
		log.Printf("Invalid format: %s", format)
		http.Error(w, "Invalid format", http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "Error reading the file", http.StatusInternalServerError)
		return
	}

	log.Printf("Received %s file with %d bytes", format, len(data))

	var result map[string]string
	if format == "card" {
		cardNo, firstName, lastName, err := simple.CardExtractCardNumberAndDriverName(data)
		if err != nil {
			log.Printf("could not extract card info: %v", err)
			http.Error(w, "Could not extract card info", http.StatusUnprocessableEntity)
			return
		}
		result = map[string]string{
			"cardNumber": cardNo,
			"firstName":  firstName,
			"lastName":   lastName,
		}
	} else {
		idNo, regNo, err := simple.VuExtractIdentificationNumberAndRegistrationNumber(data)
		if err != nil {
			log.Printf("could not extract vehicle info: %v", err)
			http.Error(w, "Could not extract vehicle info", http.StatusUnprocessableEntity)
			return
		}
		result = map[string]string{
			"registrationNumber":   regNo,
			"identificationNumber": idNo,
		}
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		http.Error(w, "Could not marshal response", http.StatusInternalServerError)
		return
	}

	log.Println("Sending response")

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func main() {

	// http.HandleFunc("/upload", uploadHandler)
	mux := http.NewServeMux()
	mux.HandleFunc("/info", infoHandler) // returns card number / vehicle registration only
	mux.HandleFunc("/", uploadHandler)   // full decode (existing behaviour; matches any other path)

	log.Println("Starting server on :8080")
	// if err := http.ListenAndServe(":8080", nil); err != nil {
	// 	log.Fatalf("Could not start server: %s\n", err.Error())
	// }

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,

		ReadTimeout:       5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	log.Fatal(srv.ListenAndServe())
}
