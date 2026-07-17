package certificateexportapprovals

import "errors"

var (
	ErrNotFound          = errors.New("certificate export approval not found")
	ErrAlreadyApproved   = errors.New("certificate export approval is already approved")
	ErrSelfApproval      = errors.New("certificate export approval requires a distinct approver")
	ErrAlreadyConsumed   = errors.New("certificate export approval is already consumed")
	ErrExpired           = errors.New("certificate export approval has expired")
	ErrExportMismatch    = errors.New("certificate export approval does not match requested material")
	ErrUnauthorisedActor = errors.New("certificate export approval actor does not match requester")
)

type Approval struct {
	ID             string   `json:"id"`
	CertificateIDs []string `json:"certificate_ids"`
	RequesterID    string   `json:"requester_id"`
	ApprovedByID   string   `json:"approved_by_id,omitempty"`
	CreatedAt      string   `json:"created_at"`
	ExpiresAt      string   `json:"expires_at"`
	ApprovedAt     string   `json:"approved_at,omitempty"`
	ConsumedAt     string   `json:"consumed_at,omitempty"`
}
