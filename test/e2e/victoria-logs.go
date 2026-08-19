package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	urls "net/url"
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
	assertLogsAvailable_helper(ctx, t, localPort, false)
}

// assertWorkloadLogsAvailable works like assertLogsAvailable, but it explicitly queries for logs from the workload cluster.
func assertWorkloadLogsAvailable(ctx context.Context, t *testing.T, localPort int) {
	assertLogsAvailable_helper(ctx, t, localPort, true)
}

func assertLogsAvailable_helper(ctx context.Context, t *testing.T, localPort int, workload bool) {
	t.Helper()
	label := "logs in Victoria Logs (platform)"
	if workload {
		label = "workload cluster logs in Victoria Logs"
	}
	if err := waitFor(t, label, func(ctx context.Context) (bool, error) {
		var url string
		if workload {
			url = fmt.Sprintf("http://localhost:%d/select/logsql/query?query=%s&limit=1&start=now-30m", localPort, urls.QueryEscape(`_stream:{k8s_cluster="openmcp-system/workload"}`))
		} else {
			url = fmt.Sprintf("http://localhost:%d/select/logsql/query?query=*&limit=1&start=now-30m", localPort)
		}
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
		t.Errorf("%v", err)
	}
}
