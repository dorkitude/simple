package cmd

import (
	"context"
	"net/http"
	"testing"
)

const pushJSON = `{"id":42,"domain_id":100,"contact_id":null,"account_id":2020}`

func TestInitiateDomainPush(t *testing.T) {
	c, seen := recordingClient(t, 201, `{"data":`+pushJSON+`}`)
	p, err := initiateDomainPush(context.Background(), c, "1010", "example.com", "friend@example.net", "")
	if err != nil || p.ID != 42 {
		t.Fatalf("got %+v, %v", p, err)
	}
	r := expectOne(t, seen, http.MethodPost, "/v2/1010/domains/example.com/pushes")
	if r.Body["new_account_email"] != "friend@example.net" {
		t.Errorf("body %s", r.Raw)
	}
	if _, ok := r.Body["new_account_identifier"]; ok {
		t.Errorf("new_account_identifier should be omitted: %s", r.Raw)
	}
}

func TestInitiateDomainPushNeedsTarget(t *testing.T) {
	c, seen := recordingClient(t, 201, `{}`)
	if _, err := initiateDomainPush(context.Background(), c, "1010", "example.com", "", ""); err == nil {
		t.Fatal("expected error")
	}
	if len(*seen) != 0 {
		t.Fatal("must not call the API")
	}
}

func TestListDomainPushes(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":[`+pushJSON+`],"pagination":{}}`)
	got, err := listDomainPushes(context.Background(), c, "1010", nil)
	if err != nil || len(got) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
	expectOne(t, seen, http.MethodGet, "/v2/1010/pushes")
}

func TestAcceptDomainPush(t *testing.T) {
	c, seen := recordingClient(t, 204, "")
	if err := acceptDomainPush(context.Background(), c, "1010", 42, 7); err != nil {
		t.Fatal(err)
	}
	r := expectOne(t, seen, http.MethodPost, "/v2/1010/pushes/42")
	if r.Body["contact_id"] != float64(7) {
		t.Errorf("body %s", r.Raw)
	}
	if err := acceptDomainPush(context.Background(), c, "1010", 42, 0); err == nil {
		t.Error("expected error without contact id")
	}
}

func TestRejectDomainPush(t *testing.T) {
	c, seen := recordingClient(t, 204, "")
	if err := rejectDomainPush(context.Background(), c, "1010", 42); err != nil {
		t.Fatal(err)
	}
	expectOne(t, seen, http.MethodDelete, "/v2/1010/pushes/42")
}
