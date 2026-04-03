package services

import (
	"context"
	"fmt"
	"strings"

	"waf/control-plane/internal/audits"
	"waf/control-plane/internal/certificatematerials"
	"waf/control-plane/internal/certificates"
)

type CertificateMaterialStore interface {
	Put(certificateID string, certificatePEM []byte, privateKeyPEM []byte) (certificatematerials.MaterialRecord, error)
	Get(certificateID string) (certificatematerials.MaterialRecord, error)
}

type CertificateUploadRequest struct {
	CertificateID string
	CommonName    string
	SANList       []string
	NotBefore     string
	NotAfter      string
	Status        string
}

type CertificateUploadResult struct {
	Certificate certificates.Certificate            `json:"certificate"`
	Material    certificatematerials.MaterialRecord `json:"material"`
}

type CertificateUploadService struct {
	certificates CertificateStore
	materials    CertificateMaterialStore
	audits       *AuditService
}

func NewCertificateUploadService(certificates CertificateStore, materials CertificateMaterialStore, audits *AuditService) *CertificateUploadService {
	return &CertificateUploadService{
		certificates: certificates,
		materials:    materials,
		audits:       audits,
	}
}

func (s *CertificateUploadService) Upload(ctx context.Context, req CertificateUploadRequest, certificatePEM []byte, privateKeyPEM []byte) (result CertificateUploadResult, err error) {
	req.CertificateID = strings.ToLower(strings.TrimSpace(req.CertificateID))
	req.CommonName = strings.TrimSpace(req.CommonName)
	req.NotBefore = strings.TrimSpace(req.NotBefore)
	req.NotAfter = strings.TrimSpace(req.NotAfter)
	req.Status = strings.ToLower(strings.TrimSpace(req.Status))
	defer func() {
		recordAudit(ctx, s.audits, audits.AuditEvent{
			Action:       "certificate.upload",
			ResourceType: "certificate",
			ResourceID:   req.CertificateID,
			Status:       auditStatus(err),
			Summary:      "manual certificate upload",
		})
	}()

	if req.CertificateID == "" {
		return CertificateUploadResult{}, fmt.Errorf("certificate id is required")
	}

	certificate, err := s.ensureCertificate(req)
	if err != nil {
		return CertificateUploadResult{}, err
	}

	material, err := s.materials.Put(req.CertificateID, certificatePEM, privateKeyPEM)
	if err != nil {
		return CertificateUploadResult{}, err
	}

	return CertificateUploadResult{
		Certificate: certificate,
		Material:    material,
	}, nil
}

func (s *CertificateUploadService) ensureCertificate(req CertificateUploadRequest) (certificates.Certificate, error) {
	items, err := s.certificates.List()
	if err != nil {
		return certificates.Certificate{}, err
	}
	for _, item := range items {
		if item.ID == req.CertificateID {
			return item, nil
		}
	}

	if req.CommonName == "" {
		return certificates.Certificate{}, fmt.Errorf("certificate %s not found", req.CertificateID)
	}

	status := req.Status
	if status == "" {
		status = "active"
	}

	return s.certificates.Create(certificates.Certificate{
		ID:         req.CertificateID,
		CommonName: req.CommonName,
		SANList:    req.SANList,
		NotBefore:  req.NotBefore,
		NotAfter:   req.NotAfter,
		Status:     status,
	})
}
