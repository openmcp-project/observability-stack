package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"sigs.k8s.io/e2e-framework/klient/wait"
)

// assertLogsAvailable waits until Victoria Logs contains at least one log entry,
// failing the test if no logs appear after the timeout.
//
// It queries the LogsQL HTTP API at /select/logsql/query, which returns NDJSON.
// A non-empty response body indicates that logs have been ingested.
func assertLogsAvailable(ctx context.Context, t *testing.T, localPort int) {
	t.Helper()
	if err := wait.For(func(ctx context.Context) (bool, error) {
		url := fmt.Sprintf("http://localhost:%d/select/logsql/query?query=*&limit=1&start=now-30m", localPort)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return false, nil
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return false, nil
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false, nil
		}

		return len(bytes.TrimSpace(body)) > 0, nil
	}, wait.WithTimeout(10*time.Minute), wait.WithContext(ctx)); err != nil {
		t.Errorf("no logs found in Victoria Logs after timeout: %v", err)
	}
}
