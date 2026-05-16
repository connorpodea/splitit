package store

import (
	"errors"
	"sync"
	"time"

	"github.com/you/p2p-bnpl/internal/models"
)

// Store holds all in-memory state.
// TODO: replace with a real database (SQLite → Postgres).
type Store struct {
	mu           sync.RWMutex
	users        map[string]*models.User
	payments     map[string]*models.Payment
	plans        map[string]*models.BNPLPlan
	installments map[string][]*models.Installment // keyed by plan ID
}

func New() *Store {
	return &Store{
		users:        make(map[string]*models.User),
		payments:     make(map[string]*models.Payment),
		plans:        make(map[string]*models.BNPLPlan),
		installments: make(map[string][]*models.Installment),
	}
}

// --- Users ---

func (s *Store) CreateUser(u *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u.CreatedAt = time.Now()
	s.users[u.ID] = u
	return nil
}

func (s *Store) GetUser(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (s *Store) ListUsers() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.User, 0, len(s.users))
	for _, u := range s.users {
		out = append(out, u)
	}
	return out
}

// --- Payments ---

func (s *Store) CreatePayment(p *models.Payment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p.CreatedAt = time.Now()
	p.Status = models.StatusPending
	s.payments[p.ID] = p
	return nil
}

func (s *Store) GetPayment(id string) (*models.Payment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.payments[id]
	if !ok {
		return nil, errors.New("payment not found")
	}
	return p, nil
}

func (s *Store) ListPayments() []*models.Payment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.Payment, 0, len(s.payments))
	for _, p := range s.payments {
		out = append(out, p)
	}
	return out
}

func (s *Store) SettlePayment(id string, planID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.payments[id]
	if !ok {
		return errors.New("payment not found")
	}
	p.Status = models.StatusSettled
	if planID != "" {
		p.PlanID = planID
	}
	return nil
}

// --- BNPL Plans ---

func (s *Store) CreatePlan(plan *models.BNPLPlan) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	plan.CreatedAt = time.Now()
	plan.Status = models.PlanActive
	s.plans[plan.ID] = plan
	return nil
}

func (s *Store) GetPlan(id string) (*models.BNPLPlan, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plans[id]
	if !ok {
		return nil, errors.New("plan not found")
	}
	return p, nil
}

func (s *Store) ListPlans() []*models.BNPLPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*models.BNPLPlan, 0, len(s.plans))
	for _, p := range s.plans {
		out = append(out, p)
	}
	return out
}

func (s *Store) ListPlansByUser(userID string) []*models.BNPLPlan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []*models.BNPLPlan
	for _, p := range s.plans {
		if p.SenderID == userID {
			out = append(out, p)
		}
	}
	return out
}

// --- Installments ---

func (s *Store) SaveInstallments(planID string, items []*models.Installment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.installments[planID] = items
	return nil
}

func (s *Store) GetInstallments(planID string) ([]*models.Installment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, ok := s.installments[planID]
	if !ok {
		return []*models.Installment{}, nil
	}
	return items, nil
}

func (s *Store) MarkInstallmentPaid(planID string, number int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items, ok := s.installments[planID]
	if !ok {
		return errors.New("plan not found")
	}
	now := time.Now()
	for _, item := range items {
		if item.Number == number {
			if item.Status == models.InstallmentPaid {
				return errors.New("already paid")
			}
			item.Status = models.InstallmentPaid
			item.PaidAt = &now
			return nil
		}
	}
	return errors.New("installment not found")
}

func (s *Store) CheckPlanComplete(planID string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items, ok := s.installments[planID]
	if !ok {
		return false, errors.New("plan not found")
	}
	for _, item := range items {
		if item.Status != models.InstallmentPaid {
			return false, nil
		}
	}
	return true, nil
}

func (s *Store) CompletePlan(planID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.plans[planID]
	if !ok {
		return errors.New("plan not found")
	}
	p.Status = models.PlanCompleted
	return nil
}
