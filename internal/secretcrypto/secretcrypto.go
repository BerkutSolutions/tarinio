package secretcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

var ErrDecryptFailed = errors.New("secret decrypt failed")

func Encrypt(label string, value string, pepper string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	gcm, err := newGCM(label, pepper)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(trimmed), nil)
	blob := make([]byte, 0, len(nonce)+len(ciphertext))
	blob = append(blob, nonce...)
	blob = append(blob, ciphertext...)
	return base64.RawStdEncoding.EncodeToString(blob), nil
}

func Decrypt(label string, valueEnc string, pepper string) (string, error) {
	valueEnc = strings.TrimSpace(valueEnc)
	if valueEnc == "" {
		return "", nil
	}
	blob, err := base64.RawStdEncoding.DecodeString(valueEnc)
	if err != nil {
		return "", ErrDecryptFailed
	}
	gcm, err := newGCM(label, pepper)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(blob) <= ns {
		return "", ErrDecryptFailed
	}
	nonce := blob[:ns]
	ciphertext := blob[ns:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrDecryptFailed
	}
	return strings.TrimSpace(string(plain)), nil
}

func newGCM(label string, pepper string) (cipher.AEAD, error) {
	label = strings.TrimSpace(label)
	pepper = strings.TrimSpace(pepper)
	if label == "" || pepper == "" {
		return nil, errors.New("label and pepper are required")
	}
	sum := sha256.Sum256([]byte(label + ":" + pepper))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
