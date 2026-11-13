package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
	"github.com/wanjiru-noreen/halisi/internal/storage"
)

var ErrVendorForbidden = errors.New("vendor application or action is not permitted")
var ErrVendorIncomplete = errors.New("complete the business profile and upload business registration and LPG licensing documents before submission or approval")
var ErrVendorReviewStale = errors.New("application changed; reopen it and review the latest information and documents before deciding")

type Hours struct {
	Open   string `json:"open"`
	Close  string `json:"close"`
	Closed bool   `json:"closed"`
}
type VendorInput struct {
	Name           string           `json:"name"`
	ContactEmail   string           `json:"contact_email"`
	ContactPhone   string           `json:"contact_phone"`
	Address        string           `json:"address"`
	City           string           `json:"city"`
	OperatingHours map[string]Hours `json:"operating_hours"`
}

func (input *VendorInput) Validate() error {
	input.Name = strings.TrimSpace(input.Name)
	input.ContactEmail = NormalizeEmail(input.ContactEmail)
	input.ContactPhone = strings.TrimSpace(input.ContactPhone)
	input.Address = strings.TrimSpace(input.Address)
	input.City = strings.TrimSpace(input.City)
	for _, s := range []struct {
		v   string
		max int
	}{{input.Name, 150}, {input.Address, 500}, {input.City, 100}} {
		if !utf8.ValidString(s.v) || utf8.RuneCountInString(s.v) < 2 || len(s.v) > s.max || strings.ContainsFunc(s.v, unicode.IsControl) {
			return &ValidationError{"Provide a business name, street location and city without control characters."}
		}
	}
	email, err := mail.ParseAddress(input.ContactEmail)
	if err != nil || email.Address != input.ContactEmail || len(input.ContactEmail) > 254 || !phonePattern.MatchString(input.ContactPhone) {
		return &ValidationError{"Provide a valid contact email and international phone number."}
	}
	if len(input.OperatingHours) != 7 {
		return &ValidationError{"Provide operating hours for all seven days."}
	}
	open := false
	for _, day := range []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"} {
		h, ok := input.OperatingHours[day]
		if !ok {
			return &ValidationError{"Use mon, tue, wed, thu, fri, sat and sun for operating hours."}
		}
		if h.Closed {
			if h.Open != "" || h.Close != "" {
				return &ValidationError{"Closed days must not contain opening times."}
			}
			continue
		}
		a, err := time.Parse("15:04", h.Open)
		b, err2 := time.Parse("15:04", h.Close)
		if err != nil || err2 != nil || len(h.Open) != 5 || len(h.Close) != 5 || !a.Before(b) {
			return &ValidationError{"Hours must use HH:MM with closing after opening (Africa/Nairobi)."}
		}
		open = true
	}
	if !open {
		return &ValidationError{"At least one operating day must be open."}
	}
	return nil
}

type VendorOnboarding struct {
	Pool  *pgxpool.Pool
	Files *storage.Documents
}

func (s *VendorOnboarding) Save(ctx context.Context, userID int64, input VendorInput) (dbsqlc.Vendor, error) {
	if err := input.Validate(); err != nil {
		return dbsqlc.Vendor{}, err
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return dbsqlc.Vendor{}, err
	}
	defer tx.Rollback(ctx)
	q := dbsqlc.New(tx)
	u, err := q.LockAuthUser(ctx, userID)
	if err != nil {
		return dbsqlc.Vendor{}, err
	}
	if !u.EmailVerifiedAt.Valid || (u.Role != "customer" && u.Role != "vendor") {
		return dbsqlc.Vendor{}, ErrVendorForbidden
	}
	hours, _ := json.Marshal(input.OperatingHours)
	v, err := q.SaveApplication(ctx, dbsqlc.SaveApplicationParams{UserID: userID, Name: input.Name, Address: input.Address, City: input.City, ContactEmail: input.ContactEmail, ContactPhone: input.ContactPhone, OperatingHours: hours})
	if errors.Is(err, pgx.ErrNoRows) {
		return dbsqlc.Vendor{}, ErrVendorForbidden
	}
	if err != nil {
		return dbsqlc.Vendor{}, err
	}
	if _, err := tx.Exec(ctx, "UPDATE vendor_documents SET status='pending' WHERE vendor_id=$1", v.ID); err != nil {
		return dbsqlc.Vendor{}, err
	}
	if _, err := tx.Exec(ctx, "UPDATE users SET role='vendor' WHERE id=$1", userID); err != nil {
		return dbsqlc.Vendor{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return dbsqlc.Vendor{}, err
	}
	return v, nil
}
func (s *VendorOnboarding) Upload(ctx context.Context, userID int64, kind string, file storage.File) (dbsqlc.VendorDocument, error) {
	if kind != "business_registration" && kind != "lpg_license" {
		return dbsqlc.VendorDocument{}, &ValidationError{"Unsupported document category."}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return dbsqlc.VendorDocument{}, err
	}
	defer tx.Rollback(ctx)
	q := dbsqlc.New(tx)
	v, err := q.ApplicationByOwner(ctx, userID)
	if err != nil {
		return dbsqlc.VendorDocument{}, ErrVendorForbidden
	}
	v, err = q.LockVendor(ctx, v.ID)
	if err != nil {
		return dbsqlc.VendorDocument{}, err
	}
	if v.UserID != userID || v.Status == "suspended" {
		return dbsqlc.VendorDocument{}, ErrVendorForbidden
	}
	var old string
	_ = tx.QueryRow(ctx, "SELECT storage_key FROM vendor_documents WHERE vendor_id=$1 AND document_type=$2", v.ID, kind).Scan(&old)
	doc, err := q.SaveUploadedDocument(ctx, dbsqlc.SaveUploadedDocumentParams{VendorID: v.ID, DocumentType: kind, StorageKey: file.Key, OriginalName: file.Name, ContentType: file.ContentType, SizeBytes: file.Size, Sha256: file.SHA256})
	if err != nil {
		return doc, err
	}
	if err := q.ResetApplicationReview(ctx, v.ID); err != nil {
		return doc, err
	}
	if _, err := tx.Exec(ctx, "UPDATE vendor_documents SET status='pending' WHERE vendor_id=$1", v.ID); err != nil {
		return doc, err
	}
	if err := tx.Commit(ctx); err != nil {
		return doc, err
	}
	if old != "" && old != file.Key {
		_ = s.Files.Remove(old)
	}
	return doc, nil
}
func (s *VendorOnboarding) complete(ctx context.Context, q *dbsqlc.Queries, v dbsqlc.Vendor) error {
	owner, err := q.GetUser(ctx, v.UserID)
	if err != nil {
		return err
	}
	if !owner.EmailVerifiedAt.Valid {
		return ErrVendorIncomplete
	}
	var hours map[string]Hours
	if err := json.Unmarshal(v.OperatingHours, &hours); err != nil {
		return ErrVendorIncomplete
	}
	input := VendorInput{Name: v.Name, ContactEmail: v.ContactEmail, ContactPhone: v.ContactPhone, Address: v.Address, City: v.City, OperatingHours: hours}
	if input.Validate() != nil {
		return ErrVendorIncomplete
	}
	docs, err := q.ApplicationDocuments(ctx, v.ID)
	if err != nil {
		return err
	}
	found := map[string]bool{}
	for _, doc := range docs {
		if doc.DocumentType != "business_registration" && doc.DocumentType != "lpg_license" {
			continue
		}
		stored, err := q.StoredVendorDocument(ctx, doc.ID)
		if err != nil {
			return err
		}
		f, err := s.Files.Open(stored.StorageKey)
		if err != nil {
			return ErrVendorIncomplete
		}
		info, err := f.Stat()
		_ = f.Close()
		if err != nil || info.Size() != doc.SizeBytes || doc.SizeBytes == 0 || len(doc.Sha256) != 64 {
			return ErrVendorIncomplete
		}
		found[doc.DocumentType] = true
	}
	if !found["business_registration"] || !found["lpg_license"] {
		return ErrVendorIncomplete
	}
	return nil
}
func (s *VendorOnboarding) Submit(ctx context.Context, userID int64) (dbsqlc.Vendor, error) {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return dbsqlc.Vendor{}, err
	}
	defer tx.Rollback(ctx)
	q := dbsqlc.New(tx)
	v, err := q.ApplicationByOwner(ctx, userID)
	if err != nil {
		return v, ErrVendorForbidden
	}
	v, err = q.LockVendor(ctx, v.ID)
	if err != nil {
		return v, err
	}
	if v.UserID != userID || v.Status == "suspended" {
		return v, ErrVendorForbidden
	}
	if err := s.complete(ctx, q, v); err != nil {
		return v, err
	}
	v, err = q.SubmitApplication(ctx, v.ID)
	if err != nil {
		return v, err
	}
	return v, tx.Commit(ctx)
}
func (s *VendorOnboarding) Review(ctx context.Context, adminID, vendorID int64, status, reason string, expectedVersion int64) (dbsqlc.Vendor, error) {
	reason = strings.TrimSpace(reason)
	if expectedVersion <= 0 {
		return dbsqlc.Vendor{}, &ValidationError{Message: "Provide the review_version returned when opening the application."}
	}
	if (status != "approved" && status != "rejected" && status != "suspended" && status != "pending") || len(reason) < 3 || len(reason) > 1000 {
		return dbsqlc.Vendor{}, &ValidationError{"Provide an allowed status and a review reason (3–1000 bytes)."}
	}
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return dbsqlc.Vendor{}, err
	}
	defer tx.Rollback(ctx)
	q := dbsqlc.New(tx)
	admin, err := q.GetUser(ctx, adminID)
	if err != nil {
		return dbsqlc.Vendor{}, err
	}
	if admin.Role != "admin" {
		return dbsqlc.Vendor{}, ErrVendorForbidden
	}
	v, err := q.LockVendor(ctx, vendorID)
	if err != nil {
		return v, ErrVendorForbidden
	}
	if v.UserID == adminID {
		return v, ErrVendorForbidden
	}
	if v.ReviewVersion != expectedVersion {
		return v, ErrVendorReviewStale
	}
	if status == "approved" || status == "rejected" {
		if v.Status != "pending" || !v.SubmittedAt.Valid {
			return v, ErrVendorForbidden
		}
		if status == "approved" {
			if err := s.complete(ctx, q, v); err != nil {
				return v, err
			}
		}
	}
	if status == "suspended" && v.Status != "approved" {
		return v, ErrVendorForbidden
	}
	if status == "pending" && v.Status != "suspended" {
		return v, ErrVendorForbidden
	}
	v, err = q.ReviewApplication(ctx, dbsqlc.ReviewApplicationParams{ID: v.ID, Status: status, ReviewReason: reason, ReviewedBy: pgtype.Int8{Int64: adminID, Valid: true}})
	if err != nil {
		return v, err
	}
	if status == "pending" {
		if _, err := tx.Exec(ctx, "UPDATE vendors SET submitted_at=NULL WHERE id=$1", v.ID); err != nil {
			return v, err
		}
		v.SubmittedAt = pgtype.Timestamptz{}
		v, err = q.GetVendor(ctx, v.ID)
		if err != nil {
			return v, err
		}
	}
	docStatus := "pending"
	if status == "approved" {
		docStatus = "approved"
	}
	if status == "rejected" {
		docStatus = "rejected"
	}
	if err := q.ReviewDocuments(ctx, dbsqlc.ReviewDocumentsParams{VendorID: v.ID, Status: docStatus}); err != nil {
		return v, err
	}
	if err := q.RecordVendorReview(ctx, dbsqlc.RecordVendorReviewParams{VendorID: v.ID, AdministratorID: adminID, Status: status, Reason: reason, ReviewVersion: pgtype.Int8{Int64: expectedVersion, Valid: true}}); err != nil {
		return v, err
	}
	return v, tx.Commit(ctx)
}
