package queried

import "database/sql"

type QueriedDB struct {
	*Queries
	db *sql.DB
}

func NewQueriedDB(db *sql.DB) *QueriedDB {
	return &QueriedDB{
		Queries: New(db),
		db:      db,
	}
}

func (q *QueriedDB) DB() *sql.DB {
	return q.db
}
