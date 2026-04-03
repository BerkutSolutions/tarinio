package compiler

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

type tlsRefsData struct {
	CertificatePath string
	PrivateKeyPath  string
}

// RenderTLSArtifacts produces deterministic TLS reference artifacts for the MVP
// TLSConfig and Certificate compiler mapping.
func RenderTLSArtifacts(sites []SiteInput, tlsConfigs []TLSConfigInput, certificates []CertificateInput) ([]ArtifactOutput, error) {
	sortedSites := append([]SiteInput(nil), sites...)
	sort.Slice(sortedSites, func(i, j int) bool {
		return sortedSites[i].ID < sortedSites[j].ID
	})

	tlsBySite := make(map[string]TLSConfigInput, len(tlsConfigs))
	for _, tlsConfig := range tlsConfigs {
		if tlsConfig.ID == "" {
			return nil, errors.New("tls config id is required")
		}
		if tlsConfig.SiteID == "" {
			return nil, fmt.Errorf("tls config %s site id is required", tlsConfig.ID)
		}
		if tlsConfig.CertificateID == "" {
			return nil, fmt.Errorf("tls config %s certificate id is required", tlsConfig.ID)
		}
		tlsBySite[tlsConfig.SiteID] = tlsConfig
	}

	certByID := make(map[string]CertificateInput, len(certificates))
	for _, cert := range certificates {
		if cert.ID == "" {
			return nil, errors.New("certificate id is required")
		}
		if cert.SiteID == "" {
			return nil, fmt.Errorf("certificate %s site id is required", cert.ID)
		}
		if strings.TrimSpace(cert.StorageRef) == "" || strings.TrimSpace(cert.PrivateKeyRef) == "" {
			return nil, fmt.Errorf("certificate %s must define certificate and private key refs", cert.ID)
		}
		certByID[cert.ID] = cert
	}

	var artifacts []ArtifactOutput
	for _, site := range sortedSites {
		if !site.Enabled || !site.ListenHTTPS {
			continue
		}

		tlsConfig, ok := tlsBySite[site.ID]
		if !ok {
			return nil, fmt.Errorf("site %s requires tls config for HTTPS listener", site.ID)
		}

		cert, ok := certByID[tlsConfig.CertificateID]
		if !ok {
			return nil, fmt.Errorf("site %s certificate %s not found", site.ID, tlsConfig.CertificateID)
		}
		if cert.SiteID != site.ID {
			return nil, fmt.Errorf("site %s certificate %s belongs to site %s", site.ID, cert.ID, cert.SiteID)
		}

		content, err := renderTemplate(filepath.Join(filepath.Dir(templatesRoot()), "tls", "refs.conf.tmpl"), tlsRefsData{
			CertificatePath: cert.StorageRef,
			PrivateKeyPath:  cert.PrivateKeyRef,
		})
		if err != nil {
			return nil, fmt.Errorf("render tls refs template for %s: %w", site.ID, err)
		}

		artifacts = append(artifacts, newArtifact(
			fmt.Sprintf("tls/%s.conf", site.ID),
			ArtifactKindTLSRef,
			content,
		))
	}

	return artifacts, nil
}
