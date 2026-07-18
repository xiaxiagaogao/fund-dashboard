package store

import (
	"context"
	"testing"
)

// A newly created friend is active by default.
func TestCreateFriend_DefaultsActive(t *testing.T) {
	ctx := context.Background()
	db, err := Open(t.TempDir() + "/fund.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id, err := CreateFriend(ctx, db, "Alice", "alice", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	f, err := GetFriendByID(ctx, db, id)
	if err != nil {
		t.Fatal(err)
	}
	if !f.Active {
		t.Errorf("new friend should default to active, got Active=false")
	}
}

// Reopening an existing DB re-runs migrate(); the active-column add must be a
// no-op the second time (idempotent) and preserve the stored flag.
func TestMigrate_ActiveColumnIdempotent(t *testing.T) {
	ctx := context.Background()
	path := t.TempDir() + "/fund.db"

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	id, err := CreateFriend(ctx, db, "Carol", "carol", "hash", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := SetFriendActive(ctx, db, id, false); err != nil {
		t.Fatal(err)
	}
	db.Close()

	db2, err := Open(path) // migrate() runs again over the existing table
	if err != nil {
		t.Fatalf("reopen (idempotent migrate): %v", err)
	}
	defer db2.Close()
	f, err := GetFriendByID(ctx, db2, id)
	if err != nil {
		t.Fatal(err)
	}
	if f.Active {
		t.Errorf("active flag not preserved across reopen: got Active=true, want false")
	}
}

// SetFriendActive flips the flag and it survives round-trips through both
// GetFriendByID and ListFriends.
func TestSetFriendActive_RoundTrips(t *testing.T) {
	ctx := context.Background()
	db, err := Open(t.TempDir() + "/fund.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	id, err := CreateFriend(ctx, db, "Bob", "bob", "hash", false)
	if err != nil {
		t.Fatal(err)
	}

	if err := SetFriendActive(ctx, db, id, false); err != nil {
		t.Fatalf("deactivate: %v", err)
	}
	f, _ := GetFriendByID(ctx, db, id)
	if f.Active {
		t.Errorf("after deactivate: Active=true, want false")
	}
	fs, _ := ListFriends(ctx, db)
	if len(fs) != 1 || fs[0].Active {
		t.Errorf("ListFriends should report deactivated friend as inactive")
	}

	if err := SetFriendActive(ctx, db, id, true); err != nil {
		t.Fatalf("reactivate: %v", err)
	}
	f, _ = GetFriendByID(ctx, db, id)
	if !f.Active {
		t.Errorf("after reactivate: Active=false, want true")
	}
}
