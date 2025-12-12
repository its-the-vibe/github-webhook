package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

func webhookHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}

	var payload interface{}
	err = json.Unmarshal(body, &payload)
	if err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	jsonOutput, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Printf("Error formatting JSON: %v\n", err)
		fmt.Println(string(body))
	} else {
		fmt.Println(string(jsonOutput))
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("Webhook received")); err != nil {
		log.Printf("Error writing response: %v\n", err)
	}
}

func main() {
	http.HandleFunc("/webhook", webhookHandler)

	port := ":8080"
	log.Printf("Starting webhook server on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
