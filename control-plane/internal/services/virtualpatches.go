package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"waf/control-plane/internal/virtualpatches"
)

// VirtualPatchStore is the storage interface used by VirtualPatchService.
type VirtualPatchStore interface {
	Create(patch virtualpatches.VirtualPatch) (virtualpatches.VirtualPatch, error)
	List(siteID string) ([]virtualpatches.VirtualPatch, error)
	ListActive(siteID string) ([]virtualpatches.VirtualPatch, error)
	Delete(id string) error
}

// VirtualPatchService manages virtual patches for sites.
type VirtualPatchService struct {
	store VirtualPatchStore
}

// NewVirtualPatchService creates a new VirtualPatchService.
func NewVirtualPatchService(store VirtualPatchStore) *VirtualPatchService {
	return &VirtualPatchService{store: store}
}

// Create adds a new virtual patch for a site.
func (s *VirtualPatchService) Create(ctx context.Context, patch virtualpatches.VirtualPatch) (virtualpatches.VirtualPatch, error) {
	patch = virtualpatches.Normalize(patch)
	if strings.TrimSpace(patch.ID) == "" {
		patch.ID = generateVirtualPatchID()
	}
	if err := virtualpatches.Validate(patch); err != nil {
		return virtualpatches.VirtualPatch{}, err
	}
	created, err := s.store.Create(patch)
	if err != nil {
		return virtualpatches.VirtualPatch{}, err
	}
	if err := runAutoApply(ctx); err != nil {
		return virtualpatches.VirtualPatch{}, err
	}
	return created, nil
}

// List returns all patches for a site (both active and expired).
func (s *VirtualPatchService) List(_ context.Context, siteID string) ([]virtualpatches.VirtualPatch, error) {
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return nil, fmt.Errorf("site id is required")
	}
	return s.store.List(siteID)
}

// ListActive returns only non-expired patches for a site.
func (s *VirtualPatchService) ListActive(_ context.Context, siteID string) ([]virtualpatches.VirtualPatch, error) {
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return nil, fmt.Errorf("site id is required")
	}
	return s.store.ListActive(siteID)
}

// Delete removes a patch by ID.
func (s *VirtualPatchService) Delete(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("virtual patch id is required")
	}
	if err := s.store.Delete(id); err != nil {
		return err
	}
	return runAutoApply(ctx)
}

// generateVirtualPatchID creates a time-based unique ID for a virtual patch.
func generateVirtualPatchID() string {
	return fmt.Sprintf("vp-%d", time.Now().UnixNano())
}
