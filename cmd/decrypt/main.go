package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/main/ingest/internal/crypto"
)

func main() {
	keyPath := flag.String("key", "keys/private.pem", "path to private key PEM")
	flag.Parse()

	keyPEM, err := os.ReadFile(*keyPath)
	if err != nil {
		log.Fatalf("read key: %v", err)
	}

	privateKey, err := crypto.LoadPrivateKeyPEM(keyPEM)
	if err != nil {
		log.Fatalf("load key: %v", err)
	}

	var envelope json.RawMessage
	if err := json.NewDecoder(os.Stdin).Decode(&envelope); err != nil {
		log.Fatalf("read envelope JSON from stdin: %v", err)
	}

	plaintext, err := crypto.Decrypt(privateKey, envelope)
	if err != nil {
		log.Fatalf("decrypt: %v", err)
	}

	if _, err := os.Stdout.Write(plaintext); err != nil {
		log.Fatalf("write stdout: %v", err)
	}
	if len(plaintext) == 0 || plaintext[len(plaintext)-1] != '\n' {
		fmt.Println()
	}
}
