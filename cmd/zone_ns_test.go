package cmd

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestUpdateZoneNSRecords(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":[{"id":1,"type":"NS","name":"","content":"ns1.dnsimple.com","ttl":3600}]}`)
	got, err := updateZoneNSRecords(context.Background(), c, "1010", "example.com", zoneNSUpdate{
		NSNames: []string{"ns1.dnsimple.com", "ns2.dnsimple-edge.net"}, NSSetIDs: []int64{1, 2},
	})
	if err != nil || len(got) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
	r := expectOne(t, seen, http.MethodPut, "/v2/1010/zones/example.com/ns_records")
	if strings.TrimSpace(r.Raw) != `{"ns_names":["ns1.dnsimple.com","ns2.dnsimple-edge.net"],"ns_set_ids":[1,2]}` {
		t.Errorf("body %s", r.Raw)
	}
}

func TestUpdateZoneNSRecordsNamesOnly(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":[]}`)
	if _, err := updateZoneNSRecords(context.Background(), c, "1010", "example.com", zoneNSUpdate{NSNames: []string{"ns1.example.net"}}); err != nil {
		t.Fatal(err)
	}
	r := expectOne(t, seen, http.MethodPut, "/v2/1010/zones/example.com/ns_records")
	if strings.TrimSpace(r.Raw) != `{"ns_names":["ns1.example.net"]}` {
		t.Errorf("body %s", r.Raw)
	}
}

func TestUpdateZoneNSRecordsEmpty(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":[]}`)
	if _, err := updateZoneNSRecords(context.Background(), c, "1010", "example.com", zoneNSUpdate{}); err == nil {
		t.Fatal("expected error")
	}
	if len(*seen) != 0 {
		t.Fatal("must not call the API")
	}
}
