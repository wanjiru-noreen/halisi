package repository

import (
	"database/sql"
	"time"
)

func intFromNullInt64(value sql.NullInt64) int {
	if !value.Valid {
		return 0
	}
	return int(value.Int64)
}

func float64FromNullFloat64(value sql.NullFloat64) float64 {
	if !value.Valid {
		return 0
	}
	return value.Float64
}

func stringFromNullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func timestampFromNullTime(value sql.NullTime) string {
	if !value.Valid {
		return ""
	}
	return value.Time.Format(time.RFC3339)
}
