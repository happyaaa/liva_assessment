package main

import (
	"testing"
	"time"
)

func mustparseTime(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("failed to parse time: %v", err)
	}
	return v
}

func TestFraudOverlap(t *testing.T) {
	store := NewStore()
	start1 := mustparseTime(t, "2023-01-01T10:00:00Z")
	end1 := mustparseTime(t, "2023-02-04T11:01:00Z")
	resA, err := store.creditRecording("rec1", start1, end1, []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resA.creditsCents["user1"] == 0 || resA.creditsCents["user2"] == 0 {
		t.Fatalf("expected non-zero credits for both users")
	}

	start2 := mustparseTime(t, "2023-01-01T10:30:00Z")
	end2 := mustparseTime(t, "2023-02-04T11:30:00Z")
	resB, err := store.creditRecording("rec2", start2, end2, []string{"user1", "user3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resB.creditsCents["user1"] != 0 {
		t.Fatalf("expected zero credits for user1 due to fraud")
	}
	if resB.creditsCents["user3"] == 0 {
		t.Fatalf("expected non-zero credits for user3")
	}

	balance1 := store.Balance("user1")
	if balance1 != 0 {
		t.Fatalf("expected balance to be 0 for user1, got %d", balance1)
	}
	balance2 := store.Balance("user2")
	if balance2 == 0 {
		t.Fatalf("expected non-zero balance for user2")
	}
	balance3 := store.Balance("user3")
	if balance3 == 0 {
		t.Fatalf("expected non-zero balance for user3")
	}
}

func TestFrauddoesnotaffectotherusers(t *testing.T) {
	store := NewStore()
	start1 := mustparseTime(t, "2023-01-01T10:00:00Z")
	end1 := mustparseTime(t, "2023-02-04T11:01:00Z")
	_, err := store.creditRecording("rec1", start1, end1, []string{"user1", "user2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	start2 := mustparseTime(t, "2023-01-01T10:30:00Z")
	end2 := mustparseTime(t, "2023-02-04T11:30:00Z")
	_, err = store.creditRecording("rec2", start2, end2, []string{"user3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.Balance("user2") == 0 {
		t.Fatalf("expected user2 to keep earnings unaffected by fraud")
	}
	if store.Balance("user3") == 0 {
		t.Fatalf("expected user3 to have earnings unaffected by fraud")
	}
}

func TestWithdrawInsufficientBalance(t *testing.T) {
	store := NewStore()
	start := mustparseTime(t, "2023-01-01T10:00:00Z")
	end := mustparseTime(t, "2023-01-01T10:30:00Z")

	_, err := store.creditRecording("rec1", start, end, []string{"user1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	balance := store.Balance("user1")
	_, err = store.Withdraw("user1", balance+centsPerDollar)
	if err == nil {
		t.Fatalf("expected error for insufficient balance")
	}
	if store.Balance("user1") != balance {
		t.Fatalf("expected balance to remain unchanged after failed withdrawal")

	}
}

func TestSubMinuteRecordingCreditsZero(t *testing.T) {
	store := NewStore()
	start := mustparseTime(t, "2023-01-01T10:00:00Z")
	end := mustparseTime(t, "2023-01-01T10:00:30Z")

	res, err := store.creditRecording("rec-short", start, end, []string{"user1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.creditsCents["user1"] != 0 {
		t.Fatalf("expected zero credits for sub-minute recording")
	}
	if store.Balance("user1") != 0 {
		t.Fatalf("expected balance to be 0 for sub-minute recording")
	}
}

func TestParseMoneyToCents(t *testing.T) {
	cases := map[string]int64{
		"1.":   100,
		".50":  50,
		"2.5":  250,
		"3.25": 325,
		"0.01": 1,
	}
	for input, expected := range cases {
		got, err := parsemoneytocents(input)
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", input, err)
		}
		if got != expected {
			t.Fatalf("parse %q: expected %d, got %d", input, expected, got)
		}
	}
}
