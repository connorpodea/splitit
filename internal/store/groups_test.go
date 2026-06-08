package store_test

import (
	"testing"

	"github.com/connorpodea/splitit/internal/models"
)

// TestGetGroupMemberCounts_SingleGroup verifies that a single group with two members
// returns the correct count of 2.
func TestGetGroupMemberCounts_SingleGroup(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)

	g := &models.Group{Name: "Test Group", CreatorID: "alice"}
	if err := s.CreateGroup(g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if err := s.SendGroupInvitation(g.ID, "alice", "bob"); err != nil {
		t.Fatalf("SendGroupInvitation: %v", err)
	}
	invitations, err := s.ListIncomingGroupInvitations("bob")
	if err != nil || len(invitations) == 0 {
		t.Fatalf("ListIncomingGroupInvitations: %v (len=%d)", err, len(invitations))
	}
	if err := s.AcceptGroupInvitation(invitations[0].ID, "bob"); err != nil {
		t.Fatalf("AcceptGroupInvitation: %v", err)
	}

	counts, err := s.GetGroupMemberCounts("alice")
	if err != nil {
		t.Fatalf("GetGroupMemberCounts: %v", err)
	}
	if counts[g.ID] != 2 {
		t.Errorf("want 2 members, got %d", counts[g.ID])
	}
}

// TestGetGroupMemberCounts_MultipleGroups verifies counts across two distinct groups.
func TestGetGroupMemberCounts_MultipleGroups(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)
	seedUser(t, s, "bob", 0)
	seedUser(t, s, "carol", 0)

	g1 := &models.Group{Name: "Group One", CreatorID: "alice"}
	if err := s.CreateGroup(g1); err != nil {
		t.Fatalf("CreateGroup g1: %v", err)
	}
	g2 := &models.Group{Name: "Group Two", CreatorID: "alice"}
	if err := s.CreateGroup(g2); err != nil {
		t.Fatalf("CreateGroup g2: %v", err)
	}

	// Add bob to g1
	if err := s.SendGroupInvitation(g1.ID, "alice", "bob"); err != nil {
		t.Fatalf("SendGroupInvitation g1 bob: %v", err)
	}
	invs, _ := s.ListIncomingGroupInvitations("bob")
	if len(invs) > 0 {
		if err := s.AcceptGroupInvitation(invs[0].ID, "bob"); err != nil {
			t.Fatalf("AcceptGroupInvitation bob g1: %v", err)
		}
	}

	// Add carol to g2
	if err := s.SendGroupInvitation(g2.ID, "alice", "carol"); err != nil {
		t.Fatalf("SendGroupInvitation g2 carol: %v", err)
	}
	invs2, _ := s.ListIncomingGroupInvitations("carol")
	if len(invs2) > 0 {
		if err := s.AcceptGroupInvitation(invs2[0].ID, "carol"); err != nil {
			t.Fatalf("AcceptGroupInvitation carol g2: %v", err)
		}
	}

	counts, err := s.GetGroupMemberCounts("alice")
	if err != nil {
		t.Fatalf("GetGroupMemberCounts: %v", err)
	}
	if counts[g1.ID] != 2 {
		t.Errorf("g1: want 2 members, got %d", counts[g1.ID])
	}
	if counts[g2.ID] != 2 {
		t.Errorf("g2: want 2 members, got %d", counts[g2.ID])
	}
}

// TestGetGroupMemberCounts_Empty verifies that a user with no groups returns an empty map.
func TestGetGroupMemberCounts_Empty(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "solo", 0)

	counts, err := s.GetGroupMemberCounts("solo")
	if err != nil {
		t.Fatalf("GetGroupMemberCounts: %v", err)
	}
	if len(counts) != 0 {
		t.Errorf("expected empty map, got %v", counts)
	}
}

// TestGetGroupMemberCounts_CreatorOnlyGroup verifies that a newly created group
// with only the creator returns a count of 1.
func TestGetGroupMemberCounts_CreatorOnlyGroup(t *testing.T) {
	s := newTestStore(t)
	seedUser(t, s, "alice", 0)

	g := &models.Group{Name: "Solo Group", CreatorID: "alice"}
	if err := s.CreateGroup(g); err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}

	counts, err := s.GetGroupMemberCounts("alice")
	if err != nil {
		t.Fatalf("GetGroupMemberCounts: %v", err)
	}
	if counts[g.ID] != 1 {
		t.Errorf("want 1 member (creator only), got %d", counts[g.ID])
	}
}
