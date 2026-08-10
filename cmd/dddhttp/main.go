package main

import (
	"encoding/json"
	"io"
	"time"

	"log"
	"net/http"

	"github.com/kyburz-switzerland-ag/tachoparser/pkg/decoder"
	"github.com/kyburz-switzerland-ag/tachoparser/pkg/simple"
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

	jsonData, status, message := decodeFull(data, card)
	if status != http.StatusOK {
		http.Error(w, message, status)
		return
	}

	log.Println("Sending response")

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// decodeFull runs the full decoder and reports any failure as an HTTP status
// instead of terminating the process.
//
//   - the file cannot be parsed -> 422 Unprocessable Entity. The request was
//     well formed, the file is not. Retrying will not help.
//   - the result cannot be marshalled -> 500. That is a bug in this service,
//     not a bad file.
func decodeFull(data []byte, card bool) (jsonData []byte, status int, message string) {
	// A truncated buffer can make the parsers panic on a slice bound rather than
	// return an error. net/http would recover that per connection and reply
	// nothing at all, so the caller could not tell a corrupt file from a service
	// that is down. Turn it into the same 422 as a reported parse error.
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("error: panic while decoding (card=%t): %v", card, rec)
			jsonData, status, message = nil, http.StatusUnprocessableEntity, "Could not parse file"
		}
	}()

	if card {
		log.Printf("Try to use the card decoder")
		var c decoder.Card
		if _, err := decoder.UnmarshalTLV(data, &c); err != nil {
			log.Printf("error: could not parse card: %v", err)
			return nil, http.StatusUnprocessableEntity, "Could not parse card"
		}
		out, err := json.Marshal(c)
		if err != nil {
			log.Printf("error: could not marshal card: %v", err)
			return nil, http.StatusInternalServerError, "Could not marshal card"
		}
		return out, http.StatusOK, ""
	}

	log.Printf("Try to use the vu decoder")
	var v decoder.Vu
	if _, err := decoder.UnmarshalTV(data, &v); err != nil {
		log.Printf("error: could not parse vu data: %v", err)
		return nil, http.StatusUnprocessableEntity, "Could not parse vu data"
	}
	out, err := json.Marshal(v)
	if err != nil {
		log.Printf("error: could not marshal vu data: %v", err)
		return nil, http.StatusInternalServerError, "Could not marshal vu data"
	}
	return out, http.StatusOK, ""
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

	result, status, message := extractInfo(data, format)
	if status != http.StatusOK {
		http.Error(w, message, status)
		return
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

// extractInfo pulls the key identifiers out of a file, reporting any failure as
// an HTTP status.
//
// The panic guard matters here as much as in decodeFull: pkg/simple works by
// matching bytes, so a truncated file can panic on a slice bound. Without the
// guard net/http replies nothing, and a caller cannot tell "this file is
// unreadable" from "the decoder is down".
func extractInfo(data []byte, format string) (result map[string]string, status int, message string) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("error: panic while extracting %s info: %v", format, rec)
			result, status, message = nil, http.StatusUnprocessableEntity, "Could not extract info"
		}
	}()

	if format == "card" {
		cardNo, firstName, lastName, err := simple.CardExtractCardNumberAndDriverName(data)
		if err != nil {
			log.Printf("could not extract card info: %v", err)
			return nil, http.StatusUnprocessableEntity, "Could not extract card info"
		}
		return map[string]string{
			"cardNumber": cardNo,
			"firstName":  firstName,
			"lastName":   lastName,
		}, http.StatusOK, ""
	}

	idNo, regNo, err := simple.VuExtractIdentificationNumberAndRegistrationNumber(data)
	if err != nil {
		log.Printf("could not extract vehicle info: %v", err)
		return nil, http.StatusUnprocessableEntity, "Could not extract vehicle info"
	}
	return map[string]string{
		"registrationNumber":   regNo,
		"identificationNumber": idNo,
	}, http.StatusOK, ""
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
