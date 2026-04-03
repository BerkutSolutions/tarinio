package handlers

import (
	"net/http"

	"waf/control-plane/internal/easysiteprofiles"
)

type EasySiteProfileCatalogHandler struct{}

func NewEasySiteProfileCatalogHandler() *EasySiteProfileCatalogHandler {
	return &EasySiteProfileCatalogHandler{}
}

func (h *EasySiteProfileCatalogHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/easy-site-profiles/catalog/countries" || r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, easysiteprofiles.DefaultCountryCatalog())
}
