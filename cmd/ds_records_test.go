package cmd

import (
	"context"
	"net/http"
	"testing"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
)

const dsJSON = `{"id":24,"domain_id":1010,"algorithm":"8","digest":"C1F6","digest_type":"2","keytag":"44620","public_key":null}`

func TestListDSRecords(t *testing.T) {
	c, seen := dnsRecordingClient(t, 200, `{"data":[`+dsJSON+`],"pagination":{}}`)
	got, err := listDSRecords(context.Background(), c, "1010", "example.com", &dnsimple.ListOptions{Page: dnsimple.Int(2)})
	if err != nil || len(got) != 1 || got[0].Keytag != "44620" {
		t.Fatalf("got %+v, %v", got, err)
	}
	r := expectOne(t, seen, http.MethodGet, "/v2/1010/domains/example.com/ds_records")
	if r.Query != "page=2" {
		t.Errorf("query = %q", r.Query)
	}
}

func TestGetDSRecord(t *testing.T) {
	c, seen := dnsRecordingClient(t, 200, `{"data":`+dsJSON+`}`)
	got, err := getDSRecord(context.Background(), c, "1010", "example.com", 24)
	if err != nil || got.ID != 24 {
		t.Fatalf("got %+v, %v", got, err)
	}
	expectOne(t, seen, http.MethodGet, "/v2/1010/domains/example.com/ds_records/24")
}

func TestCreateDSRecord(t *testing.T) {
	c, seen := dnsRecordingClient(t, 201, `{"data":`+dsJSON+`}`)
	_, err := createDSRecord(context.Background(), c, "1010", "example.com", dnsimple.DelegationSignerRecord{
		Algorithm: "13", Digest: "ABCD", DigestType: "2", Keytag: "2371",
	})
	if err != nil {
		t.Fatal(err)
	}
	r := expectOne(t, seen, http.MethodPost, "/v2/1010/domains/example.com/ds_records")
	want := map[string]string{"algorithm": "13", "digest": "ABCD", "digest_type": "2", "keytag": "2371"}
	for k, v := range want {
		if r.Body[k] != v {
			t.Errorf("body[%s] = %v, want %s (body %s)", k, r.Body[k], v, r.Raw)
		}
	}
	if _, ok := r.Body["public_key"]; ok {
		t.Errorf("public_key should be omitted: %s", r.Raw)
	}
}

func TestCreateDSRecordValidation(t *testing.T) {
	c, seen := dnsRecordingClient(t, 201, `{}`)
	for _, attrs := range []dnsimple.DelegationSignerRecord{
		{Digest: "A", DigestType: "2", Keytag: "1"},
		{Algorithm: "13", Digest: "A"},
	} {
		if _, err := createDSRecord(context.Background(), c, "1010", "example.com", attrs); err == nil {
			t.Errorf("expected validation error for %+v", attrs)
		}
	}
	if len(*seen) != 0 {
		t.Fatalf("invalid input must not reach the API: %+v", *seen)
	}
	if _, err := createDSRecord(context.Background(), c, "1010", "example.de", dnsimple.DelegationSignerRecord{Algorithm: "13", PublicKey: "KEY"}); err != nil {
		t.Fatalf("public-key form should pass: %v", err)
	}
}

func TestDeleteDSRecord(t *testing.T) {
	c, seen := dnsRecordingClient(t, 204, "")
	if err := deleteDSRecord(context.Background(), c, "1010", "example.com", 24); err != nil {
		t.Fatal(err)
	}
	expectOne(t, seen, http.MethodDelete, "/v2/1010/domains/example.com/ds_records/24")
}
