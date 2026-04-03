package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
)

var ErrTOTPSecretDecryptFailed = errors.New("totp secret decrypt failed")

func EncryptTOTPSecret(secretBase32 string, pepper string) (string, error) {
	secret := strings.TrimSpace(secretBase32)
	if secret == "" {
		return "", nil
	}
	gcm, err := totpGCM(pepper)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(secret), nil)
	blob := make([]byte, 0, len(nonce)+len(ciphertext))
	blob = append(blob, nonce...)
	blob = append(blob, ciphertext...)
	return base64.RawStdEncoding.EncodeToString(blob), nil
}

func DecryptTOTPSecret(secretEnc string, pepper string) (string, error) {
	secretEnc = strings.TrimSpace(secretEnc)
	if secretEnc == "" {
		return "", nil
	}
	blob, err := base64.RawStdEncoding.DecodeString(secretEnc)
	if err != nil {
		return "", ErrTOTPSecretDecryptFailed
	}
	gcm, err := totpGCM(pepper)
	if err != nil {
		return "", err
	}
	ns := gcm.NonceSize()
	if len(blob) <= ns {
		return "", ErrTOTPSecretDecryptFailed
	}
	nonce := blob[:ns]
	ciphertext := blob[ns:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrTOTPSecretDecryptFailed
	}
	return strings.TrimSpace(string(plain)), nil
}

func totpGCM(pepper string) (cipher.AEAD, error) {
	pepper = strings.TrimSpace(pepper)
	if pepper == "" {
		return nil, errors.New("empty pepper")
	}
	sum := sha256.Sum256([]byte("waf:totp:" + pepper))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
