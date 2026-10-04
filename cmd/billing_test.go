package cmd

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
)

func TestListCharges(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":[{"invoiced_at":"2026-01-01T00:00:00Z","total_amount":"14.50","balance_amount":"0.00","reference":"1-2","state":"collected","items":[{"description":"Register example.com","amount":"14.50"}]}]}`)
	charges, err := listCharges(context.Background(), c, "1010", dnsimple.ListChargesOptions{StartDate: "2026-01-01", EndDate: "2026-06-30", Sort: "invoiced:desc"})
	if err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodGet, "/v2/1010/billing/charges")
	q, _ := url.ParseQuery(got.Query)
	if q.Get("start_date") != "2026-01-01" || q.Get("end_date") != "2026-06-30" || q.Get("sort") != "invoiced:desc" {
		t.Errorf("unexpected query %q", got.Query)
	}
	if len(charges) != 1 || charges[0].TotalAmount.StringFixed(2) != "14.50" || len(charges[0].Items) != 1 {
		t.Errorf("unexpected charges %+v", charges)
	}
}
