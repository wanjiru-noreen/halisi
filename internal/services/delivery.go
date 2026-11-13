package services

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
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
