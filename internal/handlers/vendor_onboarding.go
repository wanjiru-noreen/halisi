package handlers

import (
	"context"
	"errors"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/models"
	"github.com/wanjiru-noreen/halisi/internal/services"
	"github.com/wanjiru-noreen/halisi/internal/storage"
	webassets "github.com/wanjiru-noreen/halisi/web"
)

type VendorOnboarding struct {
	auth    *Auth
	service *services.VendorOnboarding
	queries *dbsqlc.Queries
	uploads chan struct{}
}

func NewVendorOnboarding(auth *Auth, service *services.VendorOnboarding) *VendorOnboarding {
	return &VendorOnboarding{auth: auth, service: service, queries: dbsqlc.New(service.Pool), uploads: make(chan struct{}, 2)}
}
func (h *VendorOnboarding) Routes(mux *http.ServeMux) {
	protect := func(next http.Handler) http.Handler {
		return middleware.RequireAuth(h.auth.service, h.auth.cookieName, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		}))
	}
	applicant := func(next http.Handler) http.Handler {
		return protect(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, _ := middleware.CurrentUser(r.Context())
			if u.Role != "customer" && u.Role != "vendor" {
				h.fail(w, services.ErrVendorForbidden)
				return
			}
			next.ServeHTTP(w, r)
		}))
	}
	admin := func(next http.Handler) http.Handler {
		return protect(middleware.RequirePermission(models.Administration, next))
	}
	mux.Handle("GET /vendor/register", applicant(webassets.Page("vendor-register")))
	mux.Handle("GET /vendor/verification", applicant(webassets.Page("vendor-status")))
	mux.Handle("GET /admin/vendor-verification", admin(webassets.Page("vendor-review")))
	mux.Handle("GET /api/vendor/application", applicant(http.HandlerFunc(h.application)))
	mux.Handle("PUT /api/vendor/application", applicant(http.HandlerFunc(h.save)))
	mux.Handle("POST /api/vendor/application/submit", applicant(http.HandlerFunc(h.submit)))
	mux.Handle("POST /api/vendor/documents/{type}", protect(middleware.RequirePermission(models.VendorResources, http.HandlerFunc(h.upload))))
	mux.Handle("GET /api/vendor/documents/{id}", protect(http.HandlerFunc(h.download)))
	mux.Handle("GET /api/admin/vendor-applications/{id}", admin(http.HandlerFunc(h.adminApplication)))
	mux.Handle("GET /api/admin/vendor-applications", admin(http.HandlerFunc(h.listApplications)))
	mux.Handle("POST /api/admin/vendor-applications/{id}/review", admin(http.HandlerFunc(h.review)))
	mux.Handle("GET /api/customer/vendors", protect(middleware.RequirePermission(models.CustomerResources, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		search := strings.TrimSpace(r.URL.Query().Get("q"))
		if len(search) > 100 {
			h.fail(w, &services.ValidationError{Message: "Search must not exceed 100 bytes."})
			return
		}
		vendors, err := h.queries.SearchApprovedVendors(r.Context(), search)
		if err != nil {
			h.fail(w, err)
			return
		}
		writeJSON(w, 200, map[string]any{"vendors": vendors})
	}))))
}
func (h *VendorOnboarding) fail(w http.ResponseWriter, err error) {
	var validation *services.ValidationError
	switch {
	case errors.Is(err, services.ErrVendorReviewStale):
		writeJSON(w, 409, map[string]string{"error": err.Error()})
	case errors.Is(err, services.ErrVendorForbidden), errors.Is(err, pgx.ErrNoRows):
		writeJSON(w, 403, map[string]string{"error": "Vendor application or action is not permitted."})
	case errors.Is(err, services.ErrVendorIncomplete):
		writeJSON(w, 422, map[string]string{"error": err.Error()})
	case errors.As(err, &validation):
		writeJSON(w, 422, map[string]string{"error": validation.Error()})
	case errors.Is(err, storage.ErrDocument):
		writeJSON(w, 400, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, 503, map[string]string{"error": "Vendor verification is temporarily unavailable."})
	}
}
func (h *VendorOnboarding) application(w http.ResponseWriter, r *http.Request) {
	u, _ := middleware.CurrentUser(r.Context())
	v, err := h.queries.ApplicationByOwner(r.Context(), u.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 200, map[string]any{"application": nil, "documents": []any{}})
		return
	}
	if err != nil {
		h.fail(w, err)
		return
	}
	docs, err := h.queries.ApplicationDocuments(r.Context(), v.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"application": v, "documents": docs})
}
func (h *VendorOnboarding) save(w http.ResponseWriter, r *http.Request) {
	var input services.VendorInput
	if !decodeJSON(w, r, &input) {
		return
	}
	u, _ := middleware.CurrentUser(r.Context())
	v, err := h.service.Save(r.Context(), u.ID, input)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"application": v, "redirect": "/vendor/verification"})
}
func (h *VendorOnboarding) submit(w http.ResponseWriter, r *http.Request) {
	var input struct{}
	if !decodeJSON(w, r, &input) {
		return
	}
	u, _ := middleware.CurrentUser(r.Context())
	v, err := h.service.Submit(r.Context(), u.ID)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"application": v})
}
func (h *VendorOnboarding) upload(w http.ResponseWriter, r *http.Request) {
	select {
	case h.uploads <- struct{}{}:
		defer func() { <-h.uploads }()
	default:
		writeJSON(w, 503, map[string]string{"error": "Uploads are busy. Please retry."})
		return
	}
	kind := r.PathValue("type")
	if kind != "business_registration" && kind != "lpg_license" {
		h.fail(w, &services.ValidationError{Message: "Unsupported document category."})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, storage.MaxDocumentSize+65536)
	err := r.ParseMultipartForm(256 * 1024)
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	if err != nil || r.MultipartForm == nil {
		h.fail(w, storage.ErrDocument)
		return
	}
	files := r.MultipartForm.File["document"]
	if len(r.MultipartForm.Value) != 0 || len(r.MultipartForm.File) != 1 || len(files) != 1 {
		h.fail(w, storage.ErrDocument)
		return
	}
	source, err := files[0].Open()
	if err != nil {
		h.fail(w, err)
		return
	}
	defer source.Close()
	file, err := h.service.Files.Put(files[0].Filename, files[0].Header.Get("Content-Type"), source)
	if err != nil {
		h.fail(w, err)
		return
	}
	u, _ := middleware.CurrentUser(r.Context())
	doc, err := h.service.Upload(r.Context(), u.ID, kind, file)
	if err != nil {
		_ = h.service.Files.Remove(file.Key)
		h.fail(w, err)
		return
	}
	// Storage keys are server-private and never returned to clients.
	writeJSON(w, 201, map[string]any{"document": map[string]any{"id": doc.ID, "document_type": doc.DocumentType, "original_name": doc.OriginalName, "size_bytes": doc.SizeBytes, "status": doc.Status}})
}
func (h *VendorOnboarding) download(w http.ResponseWriter, r *http.Request) {
	id, err := resourceID(r)
	if err != nil {
		h.fail(w, services.ErrVendorForbidden)
		return
	}
	u, _ := middleware.CurrentUser(r.Context())
	if u.Role != "vendor" && u.Role != "admin" {
		h.fail(w, services.ErrVendorForbidden)
		return
	}
	doc, err := h.queries.StoredVendorDocument(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	if u.Role != "admin" {
		v, err := h.queries.ApplicationByOwner(r.Context(), u.ID)
		if err != nil || v.ID != doc.VendorID {
			h.fail(w, services.ErrVendorForbidden)
			return
		}
	}
	file, err := h.service.Files.Open(doc.StorageKey)
	if err != nil {
		h.fail(w, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		h.fail(w, services.ErrVendorIncomplete)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "sandbox; default-src 'none'")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": doc.OriginalName}))
	http.ServeContent(w, r, doc.OriginalName, info.ModTime(), file)
}
func (h *VendorOnboarding) adminApplication(w http.ResponseWriter, r *http.Request) {
	id, err := resourceID(r)
	if err != nil {
		h.fail(w, services.ErrVendorForbidden)
		return
	}
	v, err := h.queries.GetVendor(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	docs, err := h.queries.ApplicationDocuments(r.Context(), id)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"application": v, "documents": docs})
}
func (h *VendorOnboarding) review(w http.ResponseWriter, r *http.Request) {
	id, err := resourceID(r)
	if err != nil {
		h.fail(w, services.ErrVendorForbidden)
		return
	}
	var input struct {
		Status        string `json:"status"`
		Reason        string `json:"reason"`
		ReviewVersion int64  `json:"review_version"`
	}
	if !decodeJSON(w, r, &input) {
		return
	}
	u, _ := middleware.CurrentUser(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	v, err := h.service.Review(ctx, u.ID, id, input.Status, strings.TrimSpace(input.Reason), input.ReviewVersion)
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"application": v})
}

func (h *VendorOnboarding) listApplications(w http.ResponseWriter, r *http.Request) {
	after := int64(0)
	if raw := r.URL.Query().Get("after_id"); raw != "" {
		var err error
		after, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || after < 0 {
			h.fail(w, &services.ValidationError{Message: "after_id must be a nonnegative integer."})
			return
		}
	}
	rows, err := h.queries.ListVendorApplications(r.Context(), after)
	if err != nil {
		h.fail(w, err)
		return
	}
	next := int64(0)
	if len(rows) > 50 {
		rows = rows[:50]
		next = rows[len(rows)-1].ID
	}
	writeJSON(w, 200, map[string]any{"applications": rows, "next_after_id": next})
}
