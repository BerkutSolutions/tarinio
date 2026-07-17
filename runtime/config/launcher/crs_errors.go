package main

import (
	"errors"
	"net/http"
)

const (
	crsErrorReleaseUnavailable = "crs_release_unavailable"
	crsErrorReleaseInvalid     = "crs_release_invalid"
	crsErrorDigestInvalid      = "crs_release_digest_invalid"
	crsErrorArchiveDownload    = "crs_archive_download_failed"
	crsErrorArchiveDigest      = "crs_archive_digest_mismatch"
	crsErrorArchiveInvalid     = "crs_archive_invalid"
	crsErrorUpdateFailed       = "crs_update_failed"
)

type crsError struct {
	code string
	err  error
}

func (e *crsError) Error() string {
	if e == nil || e.err == nil {
		return "CRS update failed"
	}
	return e.err.Error()
}

func (e *crsError) Unwrap() error { return e.err }

func newCRSError(code string, err error) error {
	return &crsError{code: code, err: err}
}

func crsErrorCode(err error) string {
	var target *crsError
	if errors.As(err, &target) && target.code != "" {
		return target.code
	}
	return crsErrorUpdateFailed
}

func (m *crsManager) recordErrorLocked(err error) {
	m.lastError = err.Error()
	m.lastErrorCode = crsErrorCode(err)
}

func (m *crsManager) clearLastErrorLocked() {
	m.lastError = ""
	m.lastErrorCode = ""
}

func writeCRSRuntimeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadGateway, map[string]string{
		"code":  crsErrorCode(err),
		"error": "CRS update request failed",
	})
}
