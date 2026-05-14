package services

import "time"

type AuthUser struct {
	ID                     string   `json:"id"`
	Username               string   `json:"username"`
	Email                  string   `json:"email"`
	FullName               string   `json:"full_name,omitempty"`
	Department             string   `json:"department,omitempty"`
	Position               string   `json:"position,omitempty"`
	Language               string   `json:"language,omitempty"`
	SessionCreatedAt       string   `json:"session_created_at,omitempty"`
	SessionExpiresAt       string   `json:"session_expires_at,omitempty"`
	LastLoginIP            string   `json:"last_login_ip,omitempty"`
	FrequentLoginIP        string   `json:"frequent_login_ip,omitempty"`
	PasswordChangedAt      string   `json:"password_changed_at,omitempty"`
	RoleIDs                []string `json:"role_ids"`
	Permissions            []string `json:"permissions"`
	TOTPEnabled            bool     `json:"totp_enabled"`
	RecoveryCodesRemaining int      `json:"recovery_codes_remaining"`
}

type AuthUserPreferences struct {
	Language string `json:"language,omitempty"`
}

type SessionResult struct {
	ID   string   `json:"id"`
	User AuthUser `json:"user"`
}

type LoginResult struct {
	RequiresTwoFactor bool          `json:"requires_2fa"`
	ChallengeID       string        `json:"challenge_id,omitempty"`
	Methods           []string      `json:"methods,omitempty"`
	Session           SessionResult `json:"session,omitempty"`
}

type TOTPSetupResult struct {
	ChallengeID     string `json:"challenge_id"`
	Secret          string `json:"secret,omitempty"`
	ProvisioningURI string `json:"provisioning_uri,omitempty"`
	QRPNGBase64     string `json:"qr_png_base64,omitempty"`
	ManualSecret    string `json:"manual_secret,omitempty"`
}

type TOTPEnableResult struct {
	User          AuthUser  `json:"user"`
	RecoveryCodes []string  `json:"recovery_codes"`
	OK            bool      `json:"ok"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
}

type PasskeyListResult struct {
	Items []PasskeyItem `json:"items"`
}

type PasskeyItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	CreatedAt  string `json:"created_at"`
	LastUsedAt string `json:"last_used_at,omitempty"`
}

type PasskeyBeginResult struct {
	ChallengeID string         `json:"challenge_id,omitempty"`
	Options     map[string]any `json:"options"`
}

type Passkey2FABeginResult struct {
	WebAuthnChallengeID string         `json:"webauthn_challenge_id"`
	Options             map[string]any `json:"options"`
}

type AuthSecurityConfig struct {
	Pepper   string
	WebAuthn WebAuthnConfig
}

type WebAuthnConfig struct {
	Enabled bool
	RPID    string
	RPName  string
	Origins []string
}
