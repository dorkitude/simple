package cmd

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/spf13/cobra"
)

// seenRequest is one request captured by the fake API server.
type seenRequest struct {
	Method string
	Path   string
	Query  string
	Body   map[string]any
	Raw    string
}

// recordingClient returns a client whose fake API answers every request with
// status and body, and records the requests it saw.
func dnsRecordingClient(t *testing.T, status int, body string) (*dnsimple.Client, *[]seenRequest) {
	t.Helper()
	var seen []seenRequest
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		req := seenRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Raw: string(raw)}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &req.Body)
		}
		seen = append(seen, req)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	})
	return c, &seen
}

// expectOne asserts exactly one request with the given method and path was made.
func expectOne(t *testing.T, seen *[]seenRequest, method, path string) seenRequest {
	t.Helper()
	if len(*seen) != 1 {
		t.Fatalf("expected 1 request, got %d: %+v", len(*seen), *seen)
	}
	r := (*seen)[0]
	if r.Method != method || r.Path != path {
		t.Fatalf("got %s %s, want %s %s", r.Method, r.Path, method, path)
	}
	return r
}

func TestRequireYes(t *testing.T) {
	c := &cobra.Command{Use: "x"}
	addYesFlag(c, "test")
	if err := requireYes(c, "do a thing"); !errors.Is(err, errRefusedWithoutYes) {
		t.Fatalf("without --yes: got %v, want errRefusedWithoutYes", err)
	}
	if err := c.Flags().Set("yes", "true"); err != nil {
		t.Fatal(err)
	}
	if err := requireYes(c, "do a thing"); err != nil {
		t.Fatalf("with --yes: %v", err)
	}
}

func TestParseID(t *testing.T) {
	if id, err := parseID("x", "42"); err != nil || id != 42 {
		t.Fatalf("parseID(42) = %d, %v", id, err)
	}
	for _, bad := range []string{"", "0", "-1", "abc"} {
		if _, err := parseID("x", bad); err == nil {
			t.Errorf("parseID(%q) expected error", bad)
		}
	}
}
