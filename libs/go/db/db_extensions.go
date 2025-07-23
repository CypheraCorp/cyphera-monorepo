package db

// GetDBTX returns the underlying database transaction or connection interface
// This is useful for starting transactions or accessing the raw database connection
func (q *Queries) GetDBTX() DBTX {
	return q.db
}
