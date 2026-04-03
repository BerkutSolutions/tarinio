package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/hex"
	"strings"
)

type RecoveryCodeHash struct {
	Hash string
	Salt string
}

func GenerateRecoveryCodes(count int) ([]string, error) {
	if count <= 0 {
		count = 10
	}
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		raw := make([]byte, 8)
		if _, err := rand.Read(raw); err != nil {
			return nil, err
		}
		token := strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "=")
		token = strings.ToUpper(token)
		if len(token) < 16 {
			token += strings.Repeat("X", 16-len(token))
		}
		out = append(out, token[:8]+"-"+token[8:16])
	}
	return out, nil
}

func HashRecoveryCode(code, pepper string) (RecoveryCodeHash, error) {
	rawSalt := make([]byte, 16)
	if _, err := rand.Read(rawSalt); err != nil {
		return RecoveryCodeHash{}, err
	}
	norm := normalizeRecoveryCode(code)
	if norm == "" {
		return RecoveryCodeHash{}, nil
	}
	sum := deriveRecoveryHash(norm, strings.TrimSpace(pepper), rawSalt)
	return RecoveryCodeHash{
		Hash: hex.EncodeToString(sum),
		Salt: base64.RawStdEncoding.EncodeToString(rawSalt),
	}, nil
}

func VerifyRecoveryCode(code, pepper, hash, salt string) bool {
	norm := normalizeRecoveryCode(code)
	if norm == "" || strings.TrimSpace(hash) == "" || strings.TrimSpace(salt) == "" {
		return false
	}
	rawSalt, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(salt))
	if err != nil {
		return false
	}
	expected, err := hex.DecodeString(strings.TrimSpace(hash))
	if err != nil {
		return false
	}
	actual := deriveRecoveryHash(norm, strings.TrimSpace(pepper), rawSalt)
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func deriveRecoveryHash(code, pepper string, salt []byte) []byte {
	seed := append([]byte("waf:recovery:"), []byte(pepper)...)
	seed = append(seed, []byte(":")...)
	seed = append(seed, []byte(code)...)
	sum := sha256.Sum256(append(seed, salt...))
	for i := 0; i < 120000; i++ {
		buf := make([]byte, 0, len(sum)+len(salt)+len(seed))
		buf = append(buf, sum[:]...)
		buf = append(buf, salt...)
		buf = append(buf, seed...)
		sum = sha256.Sum256(buf)
	}
	return sum[:]
}

func normalizeRecoveryCode(code string) string {
	code = strings.TrimSpace(strings.ToUpper(code))
	code = strings.ReplaceAll(code, "-", "")
	code = strings.ReplaceAll(code, " ", "")
	return code
}
