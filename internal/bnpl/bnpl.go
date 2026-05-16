package bnpl

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/you/p2p-bnpl/internal/models"
)

// FeeRate is what the platform charges the sender for the convenience of
// paying over time. e.g. 0.05 = 5% flat fee on the payment amount.
//
// TODO: make this dynamic based on:
//   - number of installments chosen
//   - sender's credit/repayment history
//   - payment amount (bigger amounts = lower rate?)
const FeeRate = 0.05

// MaxInstallments is the most we allow for now.
const MaxInstallments = 4

// CreatePlan builds a BNPL plan for a payment.
// The receiver already got their money (platform paid them).
// This plan tracks what the sender owes the platform.
func CreatePlan(payment *models.Payment, numInstallments int) (*models.BNPLPlan, []*models.Installment, error) {
	if numInstallments < 2 || numInstallments > MaxInstallments {
		return nil, nil, fmt.Errorf("installments must be between 2 and %d", MaxInstallments)
	}

	fee := round(payment.Amount * FeeRate)
	total := payment.Amount + fee

	plan := &models.BNPLPlan{
		ID:           newID(),
		PaymentID:    payment.ID,
		SenderID:     payment.SenderID,
		Principal:    payment.Amount,
		Fee:          fee,
		Total:        total,
		Installments: numInstallments,
	}

	schedule := buildSchedule(plan)
	return plan, schedule, nil
}

// buildSchedule splits the total evenly into N bi-weekly installments.
// First payment is due today (or on purchase — like Klarna's "Pay in 4").
//
// TODO: offer monthly option, or let user pick due dates.
// TODO: handle rounding so totals always add up exactly.
func buildSchedule(plan *models.BNPLPlan) []*models.Installment {
	perInstallment := round(plan.Total / float64(plan.Installments))
	schedule := make([]*models.Installment, plan.Installments)

	for i := 0; i < plan.Installments; i++ {
		schedule[i] = &models.Installment{
			ID:      newID(),
			PlanID:  plan.ID,
			Number:  i + 1,
			Amount:  perInstallment,
			DueDate: time.Now().Add(time.Duration(i*14) * 24 * time.Hour), // bi-weekly
			Status:  models.InstallmentPending,
		}
	}
	return schedule
}

func newID() string {
	return fmt.Sprintf("%08x", rand.Int63()&0xffffffff)
}

func round(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}
