package cmd

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

const primaryJSON = `{"id":4,"account_id":1010,"name":"PrimaryProduction","ip":"1.2.3.4","port":53,"linked_secondary_zones":["example.com"],"created_at":"2016-08-11T10:16:46Z","updated_at":"2016-08-11T10:16:46Z"}`

func TestListPrimaryServers(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":[`+primaryJSON+`],"pagination":{"current_page":1}}`)
	got, err := listPrimaryServers(context.Background(), c, "1010", 2, 10, "name:asc")
	if err != nil || len(got) != 1 || got[0].Port != 53 || got[0].LinkedSecondaryZones[0] != "example.com" {
		t.Fatalf("got %+v, %v", got, err)
	}
	r := expectOne(t, seen, http.MethodGet, "/v2/1010/secondary_dns/primaries")
	if r.Query != "page=2&per_page=10&sort=name%3Aasc" {
		t.Errorf("query = %q", r.Query)
	}
}

func TestGetPrimaryServer(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":`+primaryJSON+`}`)
	p, err := getPrimaryServer(context.Background(), c, "1010", 4)
	if err != nil || p.Name != "PrimaryProduction" {
		t.Fatalf("got %+v, %v", p, err)
	}
	expectOne(t, seen, http.MethodGet, "/v2/1010/secondary_dns/primaries/4")
}

func TestCreatePrimaryServer(t *testing.T) {
	c, seen := recordingClient(t, 201, `{"data":`+primaryJSON+`}`)
	if _, err := createPrimaryServer(context.Background(), c, "1010", PrimaryServerAttributes{Name: "PrimaryProduction", IP: "1.2.3.4", Port: "5353"}); err != nil {
		t.Fatal(err)
	}
	r := expectOne(t, seen, http.MethodPost, "/v2/1010/secondary_dns/primaries")
	if strings.TrimSpace(r.Raw) != `{"name":"PrimaryProduction","ip":"1.2.3.4","port":"5353"}` {
		t.Errorf("body %s", r.Raw)
	}
	if _, err := createPrimaryServer(context.Background(), c, "1010", PrimaryServerAttributes{Name: "x"}); err == nil {
		t.Error("expected error without ip")
	}
}

func TestDeletePrimaryServer(t *testing.T) {
	c, seen := recordingClient(t, 204, "")
	if err := deletePrimaryServer(context.Background(), c, "1010", 4); err != nil {
		t.Fatal(err)
	}
	expectOne(t, seen, http.MethodDelete, "/v2/1010/secondary_dns/primaries/4")
}

func TestLinkUnlinkPrimaryServer(t *testing.T) {
	for _, link := range []bool{true, false} {
		c, seen := recordingClient(t, 200, `{"data":`+primaryJSON+`}`)
		if _, err := linkPrimaryServer(context.Background(), c, "1010", 4, "example.com", link); err != nil {
			t.Fatal(err)
		}
		path := "/v2/1010/secondary_dns/primaries/4/unlink"
		if link {
			path = "/v2/1010/secondary_dns/primaries/4/link"
		}
		r := expectOne(t, seen, http.MethodPut, path)
		if r.Body["zone"] != "example.com" {
			t.Errorf("body %s", r.Raw)
		}
	}
}

func TestCreateSecondaryZone(t *testing.T) {
	c, seen := recordingClient(t, 201, `{"data":{"id":1,"account_id":1010,"name":"example.com","secondary":true,"active":true}}`)
	z, err := createSecondaryZone(context.Background(), c, "1010", "example.com")
	if err != nil || !z.Secondary {
		t.Fatalf("got %+v, %v", z, err)
	}
	r := expectOne(t, seen, http.MethodPost, "/v2/1010/secondary_dns/zones")
	if r.Body["name"] != "example.com" {
		t.Errorf("body %s", r.Raw)
	}
}

func TestSecondaryAPIError(t *testing.T) {
	c, _ := recordingClient(t, 404, `{"message":"Primary server not found"}`)
	_, err := getPrimaryServer(context.Background(), c, "1010", 99)
	if err == nil || !strings.Contains(err.Error(), "Primary server not found") {
		t.Fatalf("expected API message in error, got %v", err)
	}
}
