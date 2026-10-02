package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dnsimple/dnsimple-go/dnsimple"
)

func TestParseOnOff(t *testing.T) {
	cases := map[string]bool{
		"on": true, "ON": true, "enable": true, "true": true, "yes": true,
		"off": false, "Off": false, "disable": false, "false": false, "no": false,
	}
	for in, want := range cases {
		got, err := parseOnOff(in)
		if err != nil {
			t.Fatalf("parseOnOff(%q) error: %v", in, err)
		}
		if got != want {
			t.Errorf("parseOnOff(%q) = %v, want %v", in, got, want)
		}
	}
	if _, err := parseOnOff("maybe"); err == nil {
		t.Error("parseOnOff(\"maybe\") expected error")
	}
}

// newTestClient returns a dnsimple client pointed at a fake API server.
func newTestClient(t *testing.T, h http.HandlerFunc) *dnsimple.Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c := dnsimple.NewClient(srv.Client())
	c.BaseURL = srv.URL
	return c
}

func TestGetAutoRenewStatus(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v2/1010/domains/example.com" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"id":1,"name":"example.com","state":"registered","auto_renew":true,"expires_at":"2027-01-01T00:00:00Z"}}`))
	})

	s, err := getAutoRenewStatus(context.Background(), c, "1010", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !s.AutoRenew || s.State != "registered" || s.Domain != "example.com" {
		t.Errorf("unexpected status: %+v", s)
	}
}

func TestGetAutoRenewStatusNotFound(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Domain ` + "`nope.com`" + ` not found"}`))
	})

	_, err := getAutoRenewStatus(context.Background(), c, "1010", "nope.com")
	if err == nil || !strings.Contains(err.Error(), "not found in this DNSimple account") {
		t.Fatalf("expected friendly not-found error, got %v", err)
	}
}

func TestSetAutoRenew(t *testing.T) {
	for _, tc := range []struct {
		enable bool
		method string
	}{
		{true, http.MethodPut},
		{false, http.MethodDelete},
	} {
		var gotMethod, gotPath string
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			gotMethod, gotPath = r.Method, r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		})

		if err := setAutoRenew(context.Background(), c, "1010", "example.com", tc.enable); err != nil {
			t.Fatalf("enable=%v: %v", tc.enable, err)
		}
		if gotMethod != tc.method || gotPath != "/v2/1010/registrar/domains/example.com/auto_renewal" {
			t.Errorf("enable=%v: got %s %s", tc.enable, gotMethod, gotPath)
		}
	}
}

func TestSetAutoRenewNotRegistered(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"Domain ` + "`example.org`" + ` not found"}`))
	})

	err := setAutoRenew(context.Background(), c, "1010", "example.org", false)
	if err == nil || !strings.Contains(err.Error(), "not registered with DNSimple") {
		t.Fatalf("expected not-registered error, got %v", err)
	}
}
