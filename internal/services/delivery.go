package services

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
)

var (
	ErrDeliveryNotFound      = errors.New("delivery not found or access is not permitted")
	ErrRiderNotApproved      = errors.New("selected rider is not an approved delivery partner")
	ErrDeliveryNotAssignable = errors.New("cannot assign or reassign an active or completed delivery")
)

// DeliveryManagement exposes ownership-scoped operations. Confirmation and automatic
// delivery creation commit in the same PostgreSQL statement/transaction.
type DeliveryManagement struct{ queries *dbsqlc.Queries }

func NewDeliveryManagement(pool *pgxpool.Pool) *DeliveryManagement {
	return &DeliveryManagement{queries: dbsqlc.New(pool)}
}

func (s *DeliveryManagement) Confirm(ctx context.Context, orderID, vendorUserID int64) (dbsqlc.Order, error) {
	return s.queries.ConfirmOwnedOrder(ctx, dbsqlc.ConfirmOwnedOrderParams{ID: orderID, UserID: vendorUserID})
}

func (s *DeliveryManagement) Get(ctx context.Context, id int64, user User) (dbsqlc.Delivery, error) {
	return s.queries.DeliveryForUser(ctx, dbsqlc.DeliveryForUserParams{ID: id, UserID: user.ID, Role: user.Role})
}

func (s *DeliveryManagement) Assign(ctx context.Context, deliveryID, vendorUserID, riderID int64) (dbsqlc.Delivery, error) {
	if riderID <= 0 {
		return dbsqlc.Delivery{}, &ValidationError{Message: "Provide a valid rider ID."}
	}
	approved, err := s.queries.RiderIsApproved(ctx, riderID)
	if err != nil {
		return dbsqlc.Delivery{}, err
	}
	if !approved {
		return dbsqlc.Delivery{}, ErrRiderNotApproved
	}

	delivery, err := s.queries.AssignOwnedDelivery(ctx, dbsqlc.AssignOwnedDeliveryParams{
		ID:           deliveryID,
		RiderID:      pgtype.Int8{Int64: riderID, Valid: true},
		VendorUserID: vendorUserID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			d, getErr := s.queries.DeliveryForUser(ctx, dbsqlc.DeliveryForUserParams{
				ID:     deliveryID,
				UserID: vendorUserID,
				Role:   "vendor",
			})
			if getErr != nil {
				return dbsqlc.Delivery{}, ErrDeliveryNotFound
			}
			if d.Status != "pending" && d.Status != "assigned" {
				return dbsqlc.Delivery{}, ErrDeliveryNotAssignable
			}
			return dbsqlc.Delivery{}, ErrDeliveryNotFound
		}
		return dbsqlc.Delivery{}, err
	}
	return delivery, nil
}

func (s *DeliveryManagement) AdminAssign(ctx context.Context, deliveryID, riderID int64) (dbsqlc.Delivery, error) {
	if riderID <= 0 {
		return dbsqlc.Delivery{}, &ValidationError{Message: "Provide a valid rider ID."}
	}
	approved, err := s.queries.RiderIsApproved(ctx, riderID)
	if err != nil {
		return dbsqlc.Delivery{}, err
	}
	if !approved {
		return dbsqlc.Delivery{}, ErrRiderNotApproved
	}

	delivery, err := s.queries.AdminAssignDelivery(ctx, dbsqlc.AdminAssignDeliveryParams{
		ID:      deliveryID,
		RiderID: pgtype.Int8{Int64: riderID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			d, getErr := s.queries.DeliveryForUser(ctx, dbsqlc.DeliveryForUserParams{
				ID:     deliveryID,
				Role:   "admin",
				UserID: 0,
			})
			if getErr != nil {
				return dbsqlc.Delivery{}, ErrDeliveryNotFound
			}
			if d.Status != "pending" && d.Status != "assigned" {
				return dbsqlc.Delivery{}, ErrDeliveryNotAssignable
			}
			return dbsqlc.Delivery{}, ErrDeliveryNotFound
		}
		return dbsqlc.Delivery{}, err
	}
	return delivery, nil
}

func (s *DeliveryManagement) ListApprovedRiders(ctx context.Context) ([]dbsqlc.ListApprovedRidersRow, error) {
	return s.queries.ListApprovedRiders(ctx)
}

func (s *DeliveryManagement) DeliveryWithRider(ctx context.Context, deliveryID int64) (dbsqlc.DeliveryWithRiderRow, error) {
	return s.queries.DeliveryWithRider(ctx, deliveryID)
}

// DeliveryTransitionAllowed describes the lifecycle shared by delivery operations.
// PostgreSQL enforces the same edges for writes outside the Go application.
func DeliveryTransitionAllowed(from, to string) bool {
	switch from {
	case "pending":
		return to == "assigned" || to == "cancelled"
	case "assigned":
		return to == "picked_up" || to == "failed" || to == "cancelled"
	case "picked_up":
		return to == "delivered" || to == "failed"
	default:
		return false
	}
}
