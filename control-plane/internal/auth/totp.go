package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	totpDigits = 6
	totpStep   = 30 * time.Second
)

func NewTOTPSecret() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(raw), "="), nil
}

func VerifyTOTP(secret, code string, now time.Time) bool {
	secret = strings.TrimSpace(secret)
	code = strings.TrimSpace(code)
	if secret == "" || len(code) != totpDigits {
		return false
	}
	for _, delta := range []int64{-1, 0, 1} {
		if generateTOTP(secret, now.Add(time.Duration(delta)*totpStep)) == code {
			return true
		}
	}
	return false
}

func ProvisioningURI(issuer, username, secret string) string {
	label := url.PathEscape(strings.TrimSpace(issuer) + ":" + strings.TrimSpace(username))
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", strings.TrimSpace(issuer))
	query.Set("algorithm", "SHA1")
	query.Set("digits", strconv.Itoa(totpDigits))
	query.Set("period", strconv.Itoa(int(totpStep.Seconds())))
	return fmt.Sprintf("otpauth://totp/%s?%s", label, query.Encode())
}

func GenerateCodeForTest(secret string, when time.Time) string {
	return generateTOTP(secret, when)
}

func generateTOTP(secret string, when time.Time) string {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return ""
	}
	counter := uint64(when.UTC().Unix() / int64(totpStep.Seconds()))
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := (int(sum[offset])&0x7f)<<24 |
		(int(sum[offset+1])&0xff)<<16 |
		(int(sum[offset+2])&0xff)<<8 |
		(int(sum[offset+3]) & 0xff)
	value %= 1000000
	return fmt.Sprintf("%06d", value)
}
