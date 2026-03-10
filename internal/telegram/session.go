package telegram

import (
	"sync"

	"github.com/google/uuid"
)

type State int

const (
	StateIdle State = iota

	// Registration wizard
	StateRegisterName
	StateRegisterPhone
	StateRegisterPassword
	StateRegisterRole

	// Listing creation wizard
	StateListingTitle
	StateListingCategory
	StateListingLocation
	StateListingPrice
	StateListingMinDays
	StateListingYear
	StateListingHours
	StateListingLastServiced
	StateListingDescription
	StateListingPhotos

	// Booking deposit
	StateBookingDeposit

	// Block dates wizard
	StateBlockDatesStart
	StateBlockDatesEnd
)

// Session holds per-chat state — browse results and multi-step wizard state.
type Session struct {
	LastListings []uuid.UUID       // index 0 = listing "1" in bot messages
	State        State             // current wizard step (StateIdle = no active wizard)
	Data         map[string]string // accumulates wizard input across steps
}

type SessionStore struct {
	mu   sync.RWMutex
	data map[int64]*Session
}

func newSessionStore() *SessionStore {
	return &SessionStore{data: make(map[int64]*Session)}
}

func (s *SessionStore) get(chatID int64) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if sess, ok := s.data[chatID]; ok {
		return sess
	}
	return &Session{}
}

func (s *SessionStore) setState(chatID int64, state State) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[chatID]
	if sess == nil {
		sess = &Session{}
		s.data[chatID] = sess
	}
	sess.State = state
	if state != StateIdle && sess.Data == nil {
		sess.Data = make(map[string]string)
	}
}

func (s *SessionStore) setData(chatID int64, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[chatID]
	if sess == nil {
		sess = &Session{Data: make(map[string]string)}
		s.data[chatID] = sess
	}
	if sess.Data == nil {
		sess.Data = make(map[string]string)
	}
	sess.Data[key] = value
}

func (s *SessionStore) setListings(chatID int64, ids []uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := s.data[chatID]
	if sess == nil {
		sess = &Session{}
		s.data[chatID] = sess
	}
	sess.LastListings = ids
}

func (s *SessionStore) reset(chatID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sess, ok := s.data[chatID]; ok {
		sess.State = StateIdle
		sess.Data = nil
	}
}
