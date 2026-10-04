package cmd

import (
	"context"
	"net/http"
	"testing"
)

func TestGetDnssec(t *testing.T) {
	c, seen := dnsRecordingClient(t, 200, `{"data":{"enabled":true}}`)
	on, err := getDnssec(context.Background(), c, "1010", "example.com")
	if err != nil || !on {
		t.Fatalf("getDnssec = %v, %v", on, err)
	}
	expectOne(t, seen, http.MethodGet, "/v2/1010/domains/example.com/dnssec")
}

func TestSetDnssec(t *testing.T) {
	for _, tc := range []struct {
		enable bool
		method string
		status int
		body   string
	}{
		{true, http.MethodPost, 201, `{"data":{"enabled":true}}`},
		{false, http.MethodDelete, 204, ""},
	} {
		c, seen := dnsRecordingClient(t, tc.status, tc.body)
		if err := setDnssec(context.Background(), c, "1010", "example.com", tc.enable); err != nil {
			t.Fatalf("enable=%v: %v", tc.enable, err)
		}
		expectOne(t, seen, tc.method, "/v2/1010/domains/example.com/dnssec")
	}
}

func TestSetDnssecError(t *testing.T) {
	c, _ := dnsRecordingClient(t, 400, `{"message":"DNSSEC cannot be enabled"}`)
	if err := setDnssec(context.Background(), c, "1010", "example.com", true); err == nil {
		t.Fatal("expected error")
	}
}
