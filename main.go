package main

import (
	"context"
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
	"time"

	"github.com/redis/go-redis/v9"
)

var webhookSecret []byte
var redisClient *redis.Client
var redisChannel string

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

	// Publish to Redis if client is configured
	if redisClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err = redisClient.Publish(ctx, redisChannel, body).Err()
		if err != nil {
			log.Printf("Error publishing to Redis: %v\n", err)
			// Don't fail the request if Redis publish fails
		} else {
			log.Printf("Published webhook to Redis channel: %s\n", redisChannel)
		}
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

	// Configure Redis connection
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisChannel = os.Getenv("REDIS_CHANNEL")

	// Set defaults
	if redisHost == "" {
		redisHost = "localhost"
	}
	if redisPort == "" {
		redisPort = "6379"
	}
	if redisChannel == "" {
		redisChannel = "github-webhook"
	}

	// Initialize Redis client
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)
	redisClient = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	// Test Redis connection
	ctx := context.Background()
	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Could not connect to Redis at %s: %v\n", redisAddr, err)
		log.Println("Redis publishing will be disabled. Webhook will continue to work without Redis.")
		redisClient = nil
	} else {
		log.Printf("Connected to Redis at %s\n", redisAddr)
		log.Printf("Will publish webhooks to channel: %s\n", redisChannel)
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
