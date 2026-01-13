package monitor

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/paw-chain/paw/faucet/pkg/config"
	"github.com/paw-chain/paw/faucet/pkg/faucet"
)

// BalanceMonitor monitors faucet balance and triggers refills
type BalanceMonitor struct {
	config       *config.Config
	faucetSvc    *faucet.Service
	threshold    int64
	refillAmount int64
	alertFunc    func(string)
}

// NewBalanceMonitor creates a new balance monitor
func NewBalanceMonitor(cfg *config.Config, svc *faucet.Service) *BalanceMonitor {
	return &BalanceMonitor{
		config:       cfg,
		faucetSvc:    svc,
		threshold:    cfg.LowBalanceThreshold,
		refillAmount: cfg.AutoRefillAmount,
		alertFunc:    defaultAlertFunc,
	}
}

// Start begins balance monitoring
func (bm *BalanceMonitor) Start(ctx context.Context) {
	log.Println("Starting balance monitor")

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	// Check immediately on start
	bm.checkBalance()

	for {
		select {
		case <-ctx.Done():
			log.Println("Balance monitor stopped")
			return
		case <-ticker.C:
			bm.checkBalance()
		}
	}
}

// checkBalance checks the current balance and triggers actions if needed
func (bm *BalanceMonitor) checkBalance() {
	balance, err := bm.faucetSvc.GetBalance()
	if err != nil {
		log.Printf("Failed to check balance: %v", err)
		return
	}

	log.Printf("Current faucet balance: %d %s", balance, bm.config.Denom)

	// Check if balance is below threshold
	if balance < bm.threshold {
		log.Printf("WARNING: Balance (%d) is below threshold (%d)", balance, bm.threshold)
		bm.handleLowBalance(balance)
	}

	// Check if balance is critically low
	criticalThreshold := bm.threshold / 2
	if balance < criticalThreshold {
		log.Printf("CRITICAL: Balance (%d) is critically low!", balance)
		bm.alertFunc(fmt.Sprintf("CRITICAL: Faucet balance is critically low: %d %s", balance, bm.config.Denom))
	}
}

// handleLowBalance handles low balance situation
func (bm *BalanceMonitor) handleLowBalance(currentBalance int64) {
	// Send alert
	bm.alertFunc(fmt.Sprintf("Faucet balance is low: %d %s (threshold: %d)",
		currentBalance, bm.config.Denom, bm.threshold))

	// Trigger automatic refill if enabled
	if bm.config.EnableAutoRefill {
		log.Println("Attempting automatic refill...")
		if err := bm.triggerAutoRefill(); err != nil {
			log.Printf("Auto-refill failed: %v", err)
			bm.alertFunc(fmt.Sprintf("Auto-refill failed: %v", err))
		} else {
			log.Println("Auto-refill completed successfully")
		}
	}
}

// triggerAutoRefill triggers an automatic balance refill
func (bm *BalanceMonitor) triggerAutoRefill() error {
	// In production, this would:
	// 1. Call a governance proposal to mint tokens
	// 2. Transfer from a treasury account
	// 3. Request from a multi-sig wallet
	// 4. Or use another configured refill mechanism

	log.Printf("Requesting refill of %d %s from treasury", bm.refillAmount, bm.config.Denom)

	// Simulated refill for now
	// In production, implement actual refill logic based on your chain's governance

	return nil
}

// SetAlertFunc sets the alert callback function
func (bm *BalanceMonitor) SetAlertFunc(fn func(string)) {
	bm.alertFunc = fn
}

// GetBalance returns the current balance
func (bm *BalanceMonitor) GetBalance() (int64, error) {
	return bm.faucetSvc.GetBalance()
}

// GetStatus returns the monitor status
func (bm *BalanceMonitor) GetStatus() map[string]interface{} {
	balance, err := bm.faucetSvc.GetBalance()
	status := "unknown"
	if err == nil {
		if balance >= bm.threshold {
			status = "healthy"
		} else if balance >= bm.threshold/2 {
			status = "low"
		} else {
			status = "critical"
		}
	}

	return map[string]interface{}{
		"current_balance":     balance,
		"threshold":           bm.threshold,
		"status":              status,
		"auto_refill_enabled": bm.config.EnableAutoRefill,
		"refill_amount":       bm.refillAmount,
		"error":               err,
	}
}

// defaultAlertFunc is the default alert function
func defaultAlertFunc(message string) {
	log.Printf("ALERT: %s", message)
}
