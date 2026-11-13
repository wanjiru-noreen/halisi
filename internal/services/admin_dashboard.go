package services

import (
	"context"
	"errors"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/wanjiru-noreen/halisi/internal/database/sqlc"
)

var ErrDashboardPeriod = errors.New("provide start and end as YYYY-MM-DD, with end on or after start and a range of at most 366 days")
var dashboardZone = time.FixedZone("Africa/Nairobi", 3*60*60)

type DashboardPeriod struct {
	Start    string    `json:"start"`
	End      string    `json:"end"`
	Timezone string    `json:"timezone"`
	StartAt  time.Time `json:"-"`
	EndAt    time.Time `json:"-"`
}

func ParseDashboardPeriod(values url.Values, now time.Time) (DashboardPeriod, error) {
	start, end := values.Get("start"), values.Get("end")
	if len(values["start"]) > 1 || len(values["end"]) > 1 {
		return DashboardPeriod{}, ErrDashboardPeriod
	}
	if start == "" && end == "" {
		local := now.In(dashboardZone)
		last := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, dashboardZone)
		start, end = last.AddDate(0, 0, -29).Format(time.DateOnly), last.Format(time.DateOnly)
	}
	first, err := time.ParseInLocation(time.DateOnly, start, dashboardZone)
	if err != nil {
		return DashboardPeriod{}, ErrDashboardPeriod
	}
	last, err := time.ParseInLocation(time.DateOnly, end, dashboardZone)
	if err != nil || last.Before(first) || last.Sub(first) > 365*24*time.Hour {
		return DashboardPeriod{}, ErrDashboardPeriod
	}
	return DashboardPeriod{Start: start, End: end, Timezone: "Africa/Nairobi", StartAt: first, EndAt: last.AddDate(0, 0, 1)}, nil
}

type DashboardAccount struct {
	Total    int64 `json:"total"`
	Added    int64 `json:"added"`
	Pending  int64 `json:"pending"`
	Approved int64 `json:"approved"`
}

type Dashboard struct {
	Period      DashboardPeriod `json:"period"`
	GeneratedAt time.Time       `json:"generated_at"`
	Accounts    struct {
		Customers DashboardAccount `json:"customers"`
		Vendors   DashboardAccount `json:"vendors"`
		Riders    DashboardAccount `json:"riders"`
	} `json:"accounts"`
	Orders struct {
		Total    int64                            `json:"total"`
		ByStatus []dbsqlc.AdminDashboardOrdersRow `json:"by_status"`
	} `json:"orders"`
	Financial struct {
		Currencies        []dbsqlc.AdminDashboardFinancesRow `json:"currencies"`
		Commissions       *string                            `json:"commissions"`
		Settlements       *string                            `json:"settlements"`
		Basis             string                             `json:"basis"`
		UnavailableReason string                             `json:"unavailable_reason"`
	} `json:"financial"`
	Activity []dbsqlc.AdminDashboardActivityRow `json:"activity"`
}

type AdminDashboard struct{ Pool *pgxpool.Pool }

func (s *AdminDashboard) Load(ctx context.Context, period DashboardPeriod) (Dashboard, error) {
	var result Dashboard
	tx, err := s.Pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	q := dbsqlc.New(tx)
	start, end := pgtype.Timestamptz{Time: period.StartAt, Valid: true}, pgtype.Timestamptz{Time: period.EndAt, Valid: true}
	accounts, err := q.AdminDashboardAccounts(ctx, dbsqlc.AdminDashboardAccountsParams{StartAt: start, EndAt: end})
	if err != nil {
		return result, err
	}
	result.Period = period
	result.GeneratedAt = time.Now().UTC()
	result.Accounts.Customers = DashboardAccount{Total: accounts.CustomersTotal, Added: accounts.CustomersAdded}
	result.Accounts.Vendors = DashboardAccount{Total: accounts.VendorsTotal, Added: accounts.VendorsAdded, Pending: accounts.VendorsPending, Approved: accounts.VendorsActive}
	result.Accounts.Riders = DashboardAccount{Total: accounts.RidersTotal, Added: accounts.RidersAdded, Pending: accounts.RidersPending, Approved: accounts.RidersApproved}
	result.Orders.ByStatus, err = q.AdminDashboardOrders(ctx, dbsqlc.AdminDashboardOrdersParams{StartAt: start, EndAt: end})
	if err != nil {
		return result, err
	}
	for _, status := range result.Orders.ByStatus {
		result.Orders.Total += status.Count
	}
	result.Financial.Currencies, err = q.AdminDashboardFinances(ctx, dbsqlc.AdminDashboardFinancesParams{StartAt: start, EndAt: end})
	if err != nil {
		return result, err
	}
	result.Financial.Basis = "Payments created in the selected period, using their current status. Gross sales include completed and fully refunded payments; net collections exclude full refunds. Currencies are reported separately."
	result.Financial.UnavailableReason = "Commission and settlement transactions are not recorded by the platform yet. Net collections are not platform profit or vendor settlements."
	result.Activity, err = q.AdminDashboardActivity(ctx, dbsqlc.AdminDashboardActivityParams{StartAt: start, EndAt: end})
	if err != nil {
		return result, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Dashboard{}, err
	}
	return result, nil
}
