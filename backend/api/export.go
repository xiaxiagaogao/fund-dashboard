package api

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/store"
)

// GET /api/me/export.csv → friend's full statement.
//
// Columns: date_iso, event_type, amount_usdt, nav_at_event, shares_delta,
//          shares_after, value_at_event_usdt, source, note
func (s *Server) handleMyExportCSV(w http.ResponseWriter, r *http.Request) {
	c := middleware.FromContext(r.Context())
	events, err := store.ListCashEventsByFriend(r.Context(), s.DB, c.FriendID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "events query failed")
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="%s_statement_%s.csv"`,
			c.Username, time.Now().Format("2006-01-02")))

	cw := csv.NewWriter(w)
	defer cw.Flush()

	cw.Write([]string{
		"date_iso", "event_type", "amount_usdt", "nav_at_event",
		"shares_delta", "shares_after", "value_at_event_usdt", "source", "note",
	})

	running := 0.0
	for _, e := range events {
		running += e.SharesDelta
		valueAtEvent := running * e.NAVAtEvent
		row := []string{
			time.UnixMilli(e.OccurredAt).UTC().Format(time.RFC3339),
			e.Type,
			strconv.FormatFloat(e.AmountUSDT, 'f', 4, 64),
			strconv.FormatFloat(e.NAVAtEvent, 'f', 8, 64),
			strconv.FormatFloat(e.SharesDelta, 'f', 8, 64),
			strconv.FormatFloat(running, 'f', 8, 64),
			strconv.FormatFloat(valueAtEvent, 'f', 4, 64),
			e.Source,
			nullStrOrEmpty(e.Note),
		}
		cw.Write(row)
	}
}

func nullStrOrEmpty(ns sql.NullString) string {
	if !ns.Valid {
		return ""
	}
	return ns.String
}
