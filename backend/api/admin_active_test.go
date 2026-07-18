package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/xiagao/fund-dashboard/backend/middleware"
	"github.com/xiagao/fund-dashboard/backend/store"
)

func postSetActive(t *testing.T, s *Server, callerID, targetID int64, active bool) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"active": active})
	idStr := strconv.FormatInt(targetID, 10)
	req := httptest.NewRequest("POST", "/api/admin/friends/"+idStr+"/active", bytes.NewReader(body))
	req.SetPathValue("id", idStr)
	req = req.WithContext(middleware.WithClaims(req.Context(),
		&middleware.Claims{FriendID: callerID, Username: "admin", IsAdmin: true}))
	rr := httptest.NewRecorder()
	s.handleSetFriendActive(rr, req)
	return rr
}

// Admin can deactivate and then reactivate another friend.
func TestSetFriendActive_AdminToggles(t *testing.T) {
	s, adminID, _ := newTestServer(t)
	bobID, err := store.CreateFriend(context.Background(), s.DB, "Bob", "bob", "hash", false)
	if err != nil {
		t.Fatal(err)
	}

	if rr := postSetActive(t, s, adminID, bobID, false); rr.Code != http.StatusOK {
		t.Fatalf("deactivate: got %d want 200 (%s)", rr.Code, rr.Body.String())
	}
	if f, _ := store.GetFriendByID(context.Background(), s.DB, bobID); f.Active {
		t.Errorf("bob should be inactive after deactivate")
	}

	if rr := postSetActive(t, s, adminID, bobID, true); rr.Code != http.StatusOK {
		t.Fatalf("reactivate: got %d want 200 (%s)", rr.Code, rr.Body.String())
	}
	if f, _ := store.GetFriendByID(context.Background(), s.DB, bobID); !f.Active {
		t.Errorf("bob should be active after reactivate")
	}
}

// An admin cannot deactivate themselves (lockout guard).
func TestSetFriendActive_SelfDeactivationBlocked(t *testing.T) {
	s, adminID, _ := newTestServer(t)

	rr := postSetActive(t, s, adminID, adminID, false)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("self-deactivation: got %d want 400 (%s)", rr.Code, rr.Body.String())
	}
	if f, _ := store.GetFriendByID(context.Background(), s.DB, adminID); !f.Active {
		t.Errorf("admin must remain active after blocked self-deactivation")
	}
}

// Toggling a non-existent friend is a 404.
func TestSetFriendActive_NotFound(t *testing.T) {
	s, adminID, _ := newTestServer(t)
	if rr := postSetActive(t, s, adminID, 9999, false); rr.Code != http.StatusNotFound {
		t.Errorf("unknown friend: got %d want 404", rr.Code)
	}
}

// The friends list exposes the active flag so the admin UI can render status.
func TestListFriends_IncludesActive(t *testing.T) {
	s, _, _ := newTestServer(t)
	bobID, _ := store.CreateFriend(context.Background(), s.DB, "Bob", "bob", "hash", false)
	if err := store.SetFriendActive(context.Background(), s.DB, bobID, false); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest("GET", "/api/admin/friends", nil)
	rr := httptest.NewRecorder()
	s.handleListFriends(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list: got %d", rr.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	var sawBob bool
	for _, row := range out {
		if row["username"] == "bob" {
			sawBob = true
			active, ok := row["active"].(bool)
			if !ok {
				t.Fatalf("friends row missing bool 'active' field: %v", row)
			}
			if active {
				t.Errorf("bob should be listed as inactive")
			}
		}
	}
	if !sawBob {
		t.Errorf("bob not found in friends list")
	}
}
