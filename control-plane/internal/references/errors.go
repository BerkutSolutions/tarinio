package references

import "fmt"

type ErrorCode string

const (
	CodeMissing                 ErrorCode = "missing"
	CodeCrossSiteIncompatible   ErrorCode = "cross_site_incompatible"
	CodeStale                   ErrorCode = "stale"
	CodeDuplicateHost           ErrorCode = "duplicate_host"
	CodeCertificateHostMismatch ErrorCode = "certificate_host_mismatch"
)

type ResolutionError struct {
	Code  ErrorCode
	Field string
	ID    string
}

func (e *ResolutionError) Error() string {
	if e.Field == "" {
		return string(e.Code)
	}
	if e.ID == "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Field)
	}
	return fmt.Sprintf("%s: %s %s", e.Code, e.Field, e.ID)
}

func NewError(code ErrorCode, field, id string) error {
	return &ResolutionError{Code: code, Field: field, ID: id}
}
