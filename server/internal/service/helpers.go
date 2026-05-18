package service

import "time"

func formatRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}
