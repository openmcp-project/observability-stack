package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	apimachinerywait "k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/e2e-framework/klient/wait"
)

// waitFor wraps wait.For with periodic progress logging so that a stuck wait
// is immediately visible in CI logs rather than a silent 45-minute timeout.
//
// Every logInterval (default 30s) it logs "[still waiting] <label> (elapsed Xs)".
// On failure it logs the final error with elapsed time so you know exactly which
// condition timed out and for how long.
func waitFor(t *testing.T, label string, conditionFunc apimachinerywait.ConditionWithContextFunc, opts ...wait.Option) error {
	t.Helper()

	const logInterval = 30 * time.Second

	start := time.Now()
	ticker := time.NewTicker(logInterval)
	done := make(chan struct{})
	defer func() {
		close(done)
		ticker.Stop()
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			case tick := <-ticker.C:
				t.Logf("[still waiting] %s (elapsed %s)", label, tick.Sub(start).Round(time.Second))
			}
		}
	}()

	err := wait.For(func(ctx context.Context) (bool, error) {
		ok, condErr := conditionFunc(ctx)
		if condErr != nil {
			t.Logf("[waiting] %s: transient error: %v", label, condErr)
		}
		return ok, condErr
	}, opts...)

	elapsed := time.Since(start).Round(time.Second)
	if err != nil {
		return fmt.Errorf("timed out waiting for %s after %s: %w", label, elapsed, err)
	}
	t.Logf("[done] %s (took %s)", label, elapsed)
	return nil
}
