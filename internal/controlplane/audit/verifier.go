// Package audit provides audit log chain integrity verification using cryptographic hashing.
package audit

import (
	"context"
	"log"
	"time"
)

// BackgroundVerifier runs periodic chain integrity checks in the background.
type BackgroundVerifier struct {
	chainManager *ChainManager
	interval     time.Duration
	stopCh       chan struct{}
	lastResult   *ChainVerificationResult
	running      bool
}

// NewBackgroundVerifier creates a new background verifier.
// The interval parameter specifies how often to run verification checks.
func NewBackgroundVerifier(chainManager *ChainManager, interval time.Duration) *BackgroundVerifier {
	if interval <= 0 {
		interval = 5 * time.Minute // Default interval
	}

	return &BackgroundVerifier{
		chainManager: chainManager,
		interval:     interval,
		stopCh:       make(chan struct{}),
		running:      false,
	}
}

// Start begins the background verification process.
// This method should be called in a goroutine as it blocks until Stop is called.
func (bv *BackgroundVerifier) Start(ctx context.Context) {
	if bv.running {
		return
	}

	bv.running = true
	ticker := time.NewTicker(bv.interval)
	defer ticker.Stop()

	log.Println("[audit] Background verifier started")

	// Run initial verification
	bv.runVerification(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("[audit] Background verifier stopping due to context cancellation")
			bv.running = false
			return
		case <-bv.stopCh:
			log.Println("[audit] Background verifier stopped")
			bv.running = false
			return
		case <-ticker.C:
			bv.runVerification(ctx)
		}
	}
}

// Stop stops the background verifier.
func (bv *BackgroundVerifier) Stop() {
	if !bv.running {
		return
	}
	close(bv.stopCh)
}

// IsRunning returns true if the verifier is currently running.
func (bv *BackgroundVerifier) IsRunning() bool {
	return bv.running
}

// GetLastResult returns the result of the last verification run.
func (bv *BackgroundVerifier) GetLastResult() *ChainVerificationResult {
	return bv.lastResult
}

// runVerification performs a single verification run.
func (bv *BackgroundVerifier) runVerification(ctx context.Context) {
	start := time.Now()
	result, err := bv.chainManager.VerifyChain(ctx)
	if err != nil {
		log.Printf("[audit] Chain verification failed: %v", err)
		return
	}

	bv.lastResult = result
	duration := time.Since(start)

	if !result.Valid {
		log.Printf("[audit] WARNING: Chain integrity check FAILED - %d/%d events invalid (first valid: %d, took %v)",
			result.Invalid, result.Total, result.FirstValid, duration)
	} else {
		log.Printf("[audit] Chain integrity check PASSED - %d events verified (took %v)",
			result.Total, duration)
	}
}

// VerifyOnDemand performs an on-demand verification and returns the result.
// This can be called independently of the background verification.
func (bv *BackgroundVerifier) VerifyOnDemand(ctx context.Context) (*ChainVerificationResult, error) {
	return bv.chainManager.VerifyChain(ctx)
}

// VerifyEventOnDemand performs on-demand verification of a single event.
func (bv *BackgroundVerifier) VerifyEventOnDemand(ctx context.Context, id int64) (bool, error) {
	return bv.chainManager.VerifyEvent(ctx, id)
}
