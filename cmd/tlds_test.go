package cmd

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestListTLDs(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":[{"tld":"com","minimum_registration":1,"registration_enabled":true}],"pagination":{"current_page":1,"per_page":100,"total_entries":500,"total_pages":5}}`)
	resp, err := listTLDs(context.Background(), c, pageListOptions(1, 100, ""))
	if err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodGet, "/v2/tlds")
	if !strings.Contains(got.Query, "per_page=100") {
		t.Errorf("unexpected query %q", got.Query)
	}
	if len(resp.Data) != 1 || resp.Data[0].Tld != "com" {
		t.Errorf("unexpected data %+v", resp.Data)
	}
}

func TestGetTLDNormalizes(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":{"tld":"io","whois_privacy":true}}`)
	tld, err := getTLD(context.Background(), c, ".IO")
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/tlds/io")
	if tld.Tld != "io" || !tld.WhoisPrivacy {
		t.Errorf("unexpected tld %+v", tld)
	}
}

func TestGetTLDAttributes(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":[{"name":"uk_legal_type","description":"Legal type","required":true,"options":[{"title":"Individual","value":"IND","description":"UK individual"}]}]}`)
	attrs, err := getTLDAttributes(context.Background(), c, "uk")
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/tlds/uk/extended_attributes")
	if len(attrs) != 1 || !attrs[0].Required || attrs[0].Options[0].Value != "IND" {
		t.Errorf("unexpected attrs %+v", attrs)
	}
}
