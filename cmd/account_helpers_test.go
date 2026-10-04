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

// apiCall records one request seen by the fake API server.
type apiCall struct {
	Method string
	Path   string
	Query  string
	Body   map[string]interface{}
}

// recordingClient returns a client whose fake server records each request
// and replies with the given status and JSON body.
func recordingClient(t *testing.T, status int, body string) (*dnsimple.Client, *[]apiCall) {
	t.Helper()
	calls := &[]apiCall{}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		call := apiCall{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery}
		if raw, _ := io.ReadAll(r.Body); len(raw) > 0 {
			if err := json.Unmarshal(raw, &call.Body); err != nil {
				t.Errorf("request body is not JSON: %s", raw)
			}
		}
		*calls = append(*calls, call)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	})
	return c, calls
}

// expectOneCall asserts exactly one request with the given method and path.
func expectOneCall(t *testing.T, calls *[]apiCall, method, path string) apiCall {
	t.Helper()
	if len(*calls) != 1 {
		t.Fatalf("expected 1 request, got %d: %+v", len(*calls), *calls)
	}
	got := (*calls)[0]
	if got.Method != method || got.Path != path {
		t.Fatalf("got %s %s, want %s %s", got.Method, got.Path, method, path)
	}
	return got
}

func TestConfirmWithYes(t *testing.T) {
	c := &cobra.Command{Use: "x"}
	c.Flags().Bool("yes", false, "")

	err := confirmWithYes(c, "do a thing", []string{"preview"})
	var needs errNeedsYes
	if !errors.As(err, &needs) {
		t.Fatalf("expected errNeedsYes without --yes, got %v", err)
	}

	_ = c.Flags().Set("yes", "true")
	if err := confirmWithYes(c, "do a thing", nil); err != nil {
		t.Fatalf("expected nil with --yes, got %v", err)
	}
}

func TestParseIDArg(t *testing.T) {
	if id, err := parseIDArg("x", "42"); err != nil || id != 42 {
		t.Errorf("parseIDArg(42) = %d, %v", id, err)
	}
	for _, bad := range []string{"", "abc", "0", "-3"} {
		if _, err := parseIDArg("x", bad); err == nil {
			t.Errorf("parseIDArg(%q) expected error", bad)
		}
	}
}

func TestPageListOptions(t *testing.T) {
	o := pageListOptions(0, 0, "")
	if o.Page != nil || o.PerPage != nil || o.Sort != nil {
		t.Errorf("expected empty options, got %+v", o)
	}
	o = pageListOptions(2, 50, "id:desc")
	if *o.Page != 2 || *o.PerPage != 50 || *o.Sort != "id:desc" {
		t.Errorf("unexpected options %+v", o)
	}
}
