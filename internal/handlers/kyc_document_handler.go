package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	maxKYCFileSize = 5 << 20
	kycUploadDir   = "uploads/kyc"
)

// UploadKYCDocument handles KYC document uploads from shop owners.
func (h *Handler) UploadKYCDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(
			w,
			"Method Not Allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	session, err := h.Store.Get(
		r,
		"halisi-session",
	)
	if err != nil {
		http.Error(
			w,
			"Failed to get session",
			http.StatusInternalServerError,
		)
		return
	}

	ownerID, ok := session.Values["user_id"].(int)
	if !ok || ownerID <= 0 {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	shop, err := h.ShopService.GetShopByOwnerID(ownerID)
	if err != nil || shop.ID <= 0 {
		http.Error(
			w,
			"Shop not found",
			http.StatusNotFound,
		)
		return
	}

	documentType := strings.TrimSpace(
		r.FormValue("document_type"),
	)

	if documentType == "" {
		http.Error(
			w,
			"Document type is required",
			http.StatusBadRequest,
		)
		return
	}

	allowedDocumentTypes := map[string]bool{
		"business_registration": true,
		"kra_pin":               true,
		"license":               true,
	}

	if !allowedDocumentTypes[documentType] {
		http.Error(
			w,
			"Invalid document type",
			http.StatusBadRequest,
		)
		return
	}

	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		maxKYCFileSize,
	)

	file, header, err := r.FormFile("document")
	if err != nil {
		http.Error(
			w,
			"Document file is required and must be at most 5 MB",
			http.StatusBadRequest,
		)
		return
	}
	defer file.Close()

	extension := strings.ToLower(
		filepath.Ext(header.Filename),
	)

	allowedExtensions := map[string]bool{
		".pdf":  true,
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}

	if !allowedExtensions[extension] {
		http.Error(
			w,
			"Only PDF, JPG, JPEG, and PNG files are allowed",
			http.StatusBadRequest,
		)
		return
	}

	if err := os.MkdirAll(kycUploadDir, 0755); err != nil {
		log.Printf(
			"Failed to create KYC upload directory: %v",
			err,
		)

		http.Error(
			w,
			"Failed to save document",
			http.StatusInternalServerError,
		)
		return
	}

	filename := fmt.Sprintf(
		"shop_%d_%d%s",
		shop.ID,
		time.Now().UnixNano(),
		extension,
	)

	filePath := filepath.Join(
		kycUploadDir,
		filename,
	)

	destination, err := os.Create(filePath)
	if err != nil {
		log.Printf(
			"Failed to create KYC file: %v",
			err,
		)

		http.Error(
			w,
			"Failed to save document",
			http.StatusInternalServerError,
		)
		return
	}
	defer destination.Close()

	written, err := io.Copy(
		destination,
		file,
	)
	if err != nil {
		_ = os.Remove(filePath)

		http.Error(
			w,
			"Failed to save document",
			http.StatusInternalServerError,
		)
		return
	}

	if written == 0 {
		_ = os.Remove(filePath)

		http.Error(
			w,
			"Uploaded document is empty",
			http.StatusBadRequest,
		)
		return
	}

	_, err = h.KYCDocumentService.SubmitDocument(
		shop.ID,
		documentType,
		filePath,
	)
	if err != nil {
		_ = os.Remove(filePath)

		http.Error(
			w,
			err.Error(),
			http.StatusBadRequest,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/shop/manage",
		http.StatusSeeOther,
	)
}

// GetKYCDocuments returns all KYC documents submitted by the logged-in shop owner.
func (h *Handler) GetKYCDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	session, err := h.Store.Get(r, "halisi-session")
	if err != nil {
		http.Error(w, "Failed to get session", http.StatusInternalServerError)
		return
	}

	ownerID, ok := session.Values["user_id"].(int)
	if !ok || ownerID <= 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	shop, err := h.ShopService.GetShopByOwnerID(ownerID)
	if err != nil || shop.ID <= 0 {
		http.Error(w, "Shop not found", http.StatusNotFound)
		return
	}

	documents, err := h.KYCDocumentService.GetShopDocuments(shop.ID)
	if err != nil {
		http.Error(w, "Failed to get KYC documents", http.StatusInternalServerError)
		return
	}

	for _, document := range documents {
		fmt.Fprintf(
			w,
			"ID: %d | Type: %s | Status: %s | File: %s\n",
			document.ID,
			document.DocumentType,
			document.Status,
			document.FilePath,
		)
	}
}

// GetPendingKYCDocuments returns pending KYC documents for admins.
func (h *Handler) GetPendingKYCDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	documents, err := h.KYCDocumentService.GetPendingDocuments()
	if err != nil {
		http.Error(
			w,
			"Failed to get pending KYC documents",
			http.StatusInternalServerError,
		)
		return
	}

	for _, document := range documents {
		fmt.Fprintf(
			w,
			"ID: %d | Shop ID: %d | Type: %s | Status: %s | File: %s\n",
			document.ID,
			document.ShopID,
			document.DocumentType,
			document.Status,
			document.FilePath,
		)
	}
}

// ReviewKYCDocument allows an admin to approve or reject a KYC document.
func (h *Handler) ReviewKYCDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	documentID, err := strconv.Atoi(r.FormValue("document_id"))
	if err != nil || documentID <= 0 {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}

	status := strings.TrimSpace(r.FormValue("status"))
	rejectionReason := strings.TrimSpace(r.FormValue("rejection_reason"))

	err = h.KYCDocumentService.ReviewDocument(
		documentID,
		status,
		rejectionReason,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Redirect(
		w,
		r,
		"/admin/kyc",
		http.StatusSeeOther,
	)
}

// ViewKYCDocument allows an admin to view a submitted KYC document.
func (h *Handler) ViewKYCDocument(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	documentID, err := strconv.Atoi(r.URL.Query().Get("id"))
	if err != nil || documentID <= 0 {
		http.Error(w, "Invalid document ID", http.StatusBadRequest)
		return
	}

	document, err := h.KYCDocumentService.GetDocument(documentID)
	if err != nil {
		http.Error(w, "KYC document not found", http.StatusNotFound)
		return
	}

	file, err := os.Open(document.FilePath)
	if err != nil {
		http.Error(w, "KYC document file not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf(
			`inline; filename="kyc_document_%d%s"`,
			document.ID,
			filepath.Ext(document.FilePath),
		),
	)

	http.ServeContent(
		w,
		r,
		filepath.Base(document.FilePath),
		time.Time{},
		file,
	)
}
