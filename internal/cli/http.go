package cli

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// debugTransport logs each HTTP exchange to w: method, URL, headers with
// credentials redacted, status, and duration. Response bodies are shown for
// non-2xx statuses (capped), where the API's error detail would otherwise be
// invisible. Request bodies are never shown — the login request contains the
// password.
type debugTransport struct {
	base http.RoundTripper
	w    io.Writer
}

func newDebugTransport(w io.Writer) *debugTransport {
	return &debugTransport{base: http.DefaultTransport, w: w}
}

func (t *debugTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	fmt.Fprintf(t.w, "* > %s %s\n", req.Method, req.URL)
	for key, values := range req.Header {
		if key == "X-Auth-Token" || key == "X-Auth-Login" {
			values = []string{"<redacted>"}
		}
		fmt.Fprintf(t.w, "* >   %s: %s\n", key, strings.Join(values, ", "))
	}

	start := time.Now()
	resp, err := t.base.RoundTrip(req)
	elapsed := time.Since(start).Round(time.Millisecond)
	if err != nil {
		fmt.Fprintf(t.w, "* ! %v (%s)\n", err, elapsed)
		return resp, err
	}

	fmt.Fprintf(t.w, "* < %s  Content-Type: %s  (%s)\n", resp.Status, resp.Header.Get("Content-Type"), elapsed)
	if resp.StatusCode >= 300 && resp.Body != nil {
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(body))
		if readErr == nil && len(body) > 0 {
			const cap = 4096
			shown := body
			if len(shown) > cap {
				shown = shown[:cap]
			}
			fmt.Fprintf(t.w, "* <   %s\n", strings.TrimSpace(string(shown)))
		}
	}
	return resp, err
}
