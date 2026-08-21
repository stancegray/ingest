package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
)

const envelopeVersion = 1

type Envelope struct {
	V            int    `json:"v"`
	EncryptedKey string `json:"ek"`
	Nonce        string `json:"n"`
	Ciphertext   string `json:"c"`
}

type Encryptor struct {
	publicKey *rsa.PublicKey
}

func GenerateKeyPair(bits int) (*rsa.PrivateKey, error) {
	if bits == 0 {
		bits = 4096
	}
	return rsa.GenerateKey(rand.Reader, bits)
}

func SavePrivateKey(path string, key *rsa.PrivateKey) error {
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: der}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o600)
}

func SavePublicKey(path string, key *rsa.PublicKey) error {
	der, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		return fmt.Errorf("marshal public key: %w", err)
	}
	block := &pem.Block{Type: "PUBLIC KEY", Bytes: der}
	return os.WriteFile(path, pem.EncodeToMemory(block), 0o644)
}

func LoadPublicKeyPEM(pemBytes []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid public key PEM")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return rsaPub, nil
}

func LoadPrivateKeyPEM(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("invalid private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse private key: %w", err)
		}
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rsaKey, nil
}

func NewEncryptorFromPEM(pemBytes []byte) (*Encryptor, error) {
	pub, err := LoadPublicKeyPEM(pemBytes)
	if err != nil {
		return nil, err
	}
	return &Encryptor{publicKey: pub}, nil
}

func (e *Encryptor) Encrypt(plaintext []byte) (json.RawMessage, error) {
	aesKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, aesKey); err != nil {
		return nil, fmt.Errorf("generate aes key: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	encryptedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, e.publicKey, aesKey, nil)
	if err != nil {
		return nil, fmt.Errorf("rsa encrypt: %w", err)
	}

	envelope := Envelope{
		V:            envelopeVersion,
		EncryptedKey: encodeB64(encryptedKey),
		Nonce:        encodeB64(nonce),
		Ciphertext:   encodeB64(ciphertext),
	}

	raw, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("marshal envelope: %w", err)
	}
	return raw, nil
}

func Decrypt(privateKey *rsa.PrivateKey, envelopeJSON json.RawMessage) ([]byte, error) {
	var envelope Envelope
	if err := json.Unmarshal(envelopeJSON, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if envelope.V != envelopeVersion {
		return nil, fmt.Errorf("unsupported envelope version: %d", envelope.V)
	}

	aesKey, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, decodeB64(envelope.EncryptedKey), nil)
	if err != nil {
		return nil, fmt.Errorf("rsa decrypt: %w", err)
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("aes cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}

	nonce := decodeB64(envelope.Nonce)
	ciphertext := decodeB64(envelope.Ciphertext)
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("gcm decrypt: %w", err)
	}
	return plaintext, nil
}

func IsEnvelope(raw json.RawMessage) bool {
	var envelope Envelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return false
	}
	return envelope.V == envelopeVersion && envelope.EncryptedKey != "" && envelope.Ciphertext != ""
}
