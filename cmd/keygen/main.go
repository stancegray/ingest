package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/main/ingest/internal/crypto"
)

func main() {
	outDir := flag.String("out", "keys", "output directory")
	bits := flag.Int("bits", 4096, "RSA key size")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o700); err != nil {
		log.Fatalf("mkdir: %v", err)
	}

	privateKey, err := crypto.GenerateKeyPair(*bits)
	if err != nil {
		log.Fatalf("generate key pair: %v", err)
	}

	privatePath := filepath.Join(*outDir, "private.pem")
	publicPath := filepath.Join(*outDir, "public.pem")

	if err := crypto.SavePrivateKey(privatePath, privateKey); err != nil {
		log.Fatalf("save private key: %v", err)
	}
	if err := crypto.SavePublicKey(publicPath, &privateKey.PublicKey); err != nil {
		log.Fatalf("save public key: %v", err)
	}

	fmt.Printf("Private key: %s  (keep local, never deploy)\n", privatePath)
	fmt.Printf("Public key:  %s  (set INGEST_PUBLIC_KEY_FILE=%s on the server)\n", publicPath, publicPath)
}
