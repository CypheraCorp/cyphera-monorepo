package helpers

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// StringToNullableText converts string to nullable pgtype.Text
func StringToNullableText(s string) pgtype.Text {
	if s == "" {
		return pgtype.Text{Valid: false}
	}
	return pgtype.Text{String: s, Valid: true}
}

// TimeToNullableTimestamptz converts time to nullable pgtype.Timestamptz
func TimeToNullableTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

// Int32ToNullableInt4 converts int32 to nullable pgtype.Int4
func Int32ToNullableInt4(i int32) pgtype.Int4 {
	return pgtype.Int4{Int32: i, Valid: true}
}

// Int64ToNullableInt8 converts int64 to nullable pgtype.Int8
func Int64ToNullableInt8(i int64) pgtype.Int8 {
	return pgtype.Int8{Int64: i, Valid: true}
}
