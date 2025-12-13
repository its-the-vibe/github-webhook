package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

var webhookSecret []byte

func verifySignature(payload []byte, signature string) bool {
	if len(webhookSecret) == 0 {
		// No secret configured, skip verification
		return true
	}

	if signature == "" {
		return false
	}

	// GitHub sends signature as "sha256=<hash>"
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	signatureHash := strings.TrimPrefix(signature, "sha256=")
	
	mac := hmac.New(sha256.New, webhookSecret)
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)

	return hmac.Equal([]byte(signatureHash), []byte(expectedSignature))
}

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

	// Verify GitHub webhook signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if !verifySignature(body, signature) {
		log.Printf("Invalid webhook signature")
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
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
	// Load webhook secret from .secret file
	secretData, err := os.ReadFile(".secret")
	if err != nil {
		log.Println("Warning: .secret file not found. Webhook signature verification will be skipped.")
		log.Println("To enable verification, create a .secret file with your GitHub webhook secret.")
	} else {
		webhookSecret = []byte(strings.TrimSpace(string(secretData)))
		log.Println("Webhook secret loaded. Signature verification enabled.")
	}

	http.HandleFunc("/webhook", webhookHandler)

	// Get port from environment variable, default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	// Ensure port has colon prefix
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	log.Printf("Starting webhook server on port %s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
