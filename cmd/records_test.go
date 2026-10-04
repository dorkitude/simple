package cmd

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func newRecordFlagsCmd() *cobra.Command {
	c := &cobra.Command{Use: "x"}
	c.Flags().Int("ttl", 0, "")
	c.Flags().Int("priority", 0, "")
	c.Flags().StringSlice("regions", nil, "")
	return c
}

func TestRecordFieldsFromFlagsKeepsZero(t *testing.T) {
	c := newRecordFlagsCmd()
	if err := c.ParseFlags([]string{"--priority", "0", "--regions", "SV1,IAD"}); err != nil {
		t.Fatal(err)
	}
	f := recordFieldsFromFlags(c)
	if f.Priority == nil || *f.Priority != 0 {
		t.Errorf("priority 0 was dropped: %+v", f)
	}
	if f.TTL != nil {
		t.Errorf("ttl was not given but is set: %v", *f.TTL)
	}
	if len(f.Regions) != 2 || f.Regions[0] != "SV1" || f.Regions[1] != "IAD" {
		t.Errorf("regions = %v", f.Regions)
	}
}

func TestRecordFieldsFromFlagsUnset(t *testing.T) {
	c := newRecordFlagsCmd()
	if err := c.ParseFlags(nil); err != nil {
		t.Fatal(err)
	}
	f := recordFieldsFromFlags(c)
	if f.TTL != nil || f.Priority != nil || f.Regions != nil {
		t.Errorf("expected nothing set, got %+v", f)
	}
}

func TestCreateZoneRecordSendsPriorityZero(t *testing.T) {
	c, seen := dnsRecordingClient(t, 201, `{"data":{"id":5,"type":"MX","name":"","content":"mx.example.com","ttl":3600,"priority":0}}`)
	zero, ttl := 0, 600
	resp, err := createZoneRecord(context.Background(), c, "1010", "example.com", RecordFields{
		Type: "MX", Name: dnsimpleString(""), Content: "mx.example.com", Priority: &zero, TTL: &ttl, Regions: []string{"global"},
	})
	if err != nil || resp.Data.ID != 5 {
		t.Fatalf("got %+v, %v", resp, err)
	}
	r := expectOne(t, seen, http.MethodPost, "/v2/1010/zones/example.com/records")
	want := `{"type":"MX","name":"","content":"mx.example.com","ttl":600,"priority":0,"regions":["global"]}`
	if strings.TrimSpace(r.Raw) != want {
		t.Errorf("body %s\nwant %s", r.Raw, want)
	}
}

func TestUpdateZoneRecordOnlyChangedFields(t *testing.T) {
	c, seen := dnsRecordingClient(t, 200, `{"data":{"id":5,"type":"MX","content":"mx.example.com","priority":0}}`)
	zero := 0
	if _, err := updateZoneRecord(context.Background(), c, "1010", "example.com", 5, RecordFields{Priority: &zero}); err != nil {
		t.Fatal(err)
	}
	r := expectOne(t, seen, http.MethodPatch, "/v2/1010/zones/example.com/records/5")
	if strings.TrimSpace(r.Raw) != `{"priority":0}` {
		t.Errorf("body %s", r.Raw)
	}
}

func TestCreateZoneRecordAPIError(t *testing.T) {
	c, _ := dnsRecordingClient(t, 400, `{"message":"Validation failed"}`)
	if _, err := createZoneRecord(context.Background(), c, "1010", "example.com", RecordFields{Type: "A", Content: "x"}); err == nil || !strings.Contains(err.Error(), "Validation failed") {
		t.Fatalf("got %v", err)
	}
}
