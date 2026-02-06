package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const centsPerDollar = int64(100) // 1 dollar per hour

type User struct {
	Balance    int64 // balance in cents
	Recordings []*Recording
	mu         sync.Mutex // to protect concurrent access to Balance and Recordings
}

type Recording struct {
	ID         string
	Start      int64 // start time in unix timestamp
	End        int64 // end time in unix timestamp
	IsFraud    bool  // whether the recording is fraudulent
	PaidAmount int64 // amount paid for this recording in cents
}

type Store struct {
	users map[string]*User
	mu    sync.RWMutex // to protect concurrent access to Users
}

func NewStore() *Store {
	return &Store{
		users: make(map[string]*User)}
}

func (s *Store) GetUser(userID string) *User {
	s.mu.RLock()
	u := s.users[userID]
	s.mu.RUnlock()
	if u != nil {
		return u
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u = s.users[userID]
	if u == nil {
		u = &User{Recordings: make([]*Recording, 0)}
		s.users[userID] = u
	}
	return u
}

type creditResult struct {
	fraudUsers   []string
	creditsCents map[string]int64
}

// $1/hour (pro-rated to the minute, round down to nearest minute):
// 100 cents (int) floor()
func CalculateEarnings(start, end time.Time) (int64, error) {
	if !end.After(start) {
		return 0, errors.New("invalid time range")
	}
	minutes := int64(end.Sub(start) / time.Minute)
	return minutes * centsPerDollar / 60, nil
}

func (s *Store) creditRecording(recordingID string, start, end time.Time, participant []string) (creditResult, error) {
	if recordingID == "" {
		return creditResult{}, errors.New("invalid input")
	}
	if !end.After(start) {
		return creditResult{}, errors.New("end must be after start")
	}

	res := creditResult{creditsCents: make(map[string]int64)}
	startunix := start.UnixNano()
	endunix := end.UnixNano()
	baseEarned, err := CalculateEarnings(start, end)
	if err != nil {
		return creditResult{}, err
	}

	seen := make(map[string]bool)
	uniqueParticipants := []string{}
	for _, p := range participant {
		if p != "" && !seen[p] {
			seen[p] = true
			uniqueParticipants = append(uniqueParticipants, p)
		}
	}
	if len(uniqueParticipants) == 0 {
		return res, errors.New("no valid participants")
	}
	sort.Strings(uniqueParticipants)
	users := make([]*User, 0, len(uniqueParticipants))
	for _, uid := range uniqueParticipants {
		u := s.GetUser(uid)
		users = append(users, u)
		u.mu.Lock()
	}
	defer func() {
		for i := len(users) - 1; i >= 0; i-- {
			users[i].mu.Unlock()
		}
	}()
	for i, uid := range uniqueParticipants {
		u := users[i]
		idx := sort.Search(len(u.Recordings), func(i int) bool {
			return u.Recordings[i].Start >= startunix
		})
		overlap := []*Recording{}
		for i := idx - 1; i >= 0; i-- {
			r := u.Recordings[i]
			if r.End <= startunix {
				break
			}
			overlap = append(overlap, r)
		}
		for i := idx; i < len(u.Recordings); i++ {
			r := u.Recordings[i]
			if r.Start >= endunix {
				break
			}
			overlap = append(overlap, r)
		}
		if len(overlap) == 0 {
			newRecording := &Recording{
				ID: recordingID, Start: startunix, End: endunix, PaidAmount: baseEarned, IsFraud: false}
			u.Recordings = append(u.Recordings, nil)
			copy(u.Recordings[idx+1:], u.Recordings[idx:])
			u.Recordings[idx] = newRecording
			u.Balance += baseEarned
			res.creditsCents[uid] = baseEarned
			continue
		}
		res.fraudUsers = append(res.fraudUsers, uid)
		res.creditsCents[uid] = 0
		for _, oldRec := range overlap {
			if !oldRec.IsFraud {
				oldRec.IsFraud = true
				u.Balance -= oldRec.PaidAmount
				oldRec.PaidAmount = 0
			}
		}
		if u.Balance < 0 {
			u.Balance = 0
		}
		newRecording := &Recording{ID: recordingID, Start: startunix, End: endunix, PaidAmount: 0, IsFraud: true}
		u.Recordings = append(u.Recordings, nil)
		copy(u.Recordings[idx+1:], u.Recordings[idx:])
		u.Recordings[idx] = newRecording
	}
	return res, nil
}

func (s *Store) Balance(userID string) int64 {
	u := s.GetUser(userID)
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.Balance
}

func (s *Store) Withdraw(userID string, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, errors.New("invalid amount")
	}
	u := s.GetUser(userID)
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.Balance < amount {
		return u.Balance, errors.New("insufficient balance")
	}
	u.Balance -= amount
	return u.Balance, nil
}

type server struct {
	store *Store
}

type endRecordingRequest struct {
	RecordingID  string   `json:"recordingId"`
	Start        string   `json:"start"`
	End          string   `json:"end"`
	Participants []string `json:"participants"`
}

type withdrawRequest struct {
	UserID string `json:"userId"`
	Amount string `json:"amount"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)

}

func (s *server) handleEndRecording(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req endRecordingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	start, err1 := time.Parse(time.RFC3339, req.Start)
	if err1 != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid start time"})
		return
	}
	end, err := time.Parse(time.RFC3339, req.End)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid end time"})
		return
	}
	res, err := s.store.creditRecording(req.RecordingID, start, end, req.Participants)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"credited": res.creditsCents, "fraudUsers": res.fraudUsers})
}

func (s *server) handleBalance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	userID := strings.TrimPrefix(r.URL.Path, "/balance/")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing userId"})
		return
	}
	balance := s.store.Balance(userID)
	writeJSON(w, http.StatusOK, map[string]any{"userid": userID, "balance": fmt.Sprintf("%.2f", float64(balance)/100.0)})
}

func (s *server) handleWithdraw(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req withdrawRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}
	if req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing userId"})
		return
	}
	amountCents, err := parsemoneytocents(req.Amount)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid amount"})
		return
	}
	balance, err := s.store.Withdraw(req.UserID, amountCents)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error(), "balance": fmt.Sprintf("%.2f", float64(balance)/100.0)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"userid": req.UserID, "balance": fmt.Sprintf("%.2f", float64(balance)/100.0)})
}

func parsemoneytocents(s string) (int64, error) {
	if s == "" {
		return 0, errors.New("empty amount")
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, errors.New("invalid amount format")
	}
	dollarsPart := parts[0]
	if dollarsPart == "" {
		dollarsPart = "0"
	}
	dollars, err := strconv.ParseInt(dollarsPart, 10, 64)
	if err != nil {
		return 0, errors.New("invalid dollars")
	}
	cents := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if fraction == "" {
			return dollars * centsPerDollar, nil
		}
		if len(fraction) > 2 {
			fraction = fraction[:2]
		}
		if len(fraction) == 1 {
			fraction += "0"
		}
		c, err := strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, errors.New("invalid cents")
		}
		cents = c
	}
	return dollars*centsPerDollar + cents, nil
}

func main() {
	store := NewStore()
	srv := &server{store: store}
	mux := http.NewServeMux()
	mux.HandleFunc("/recording/end", srv.handleEndRecording)
	mux.HandleFunc("/balance/", srv.handleBalance)
	mux.HandleFunc("/withdraw", srv.handleWithdraw)
	fmt.Println("Server running on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		fmt.Println("Server error:", err)
	}
}
