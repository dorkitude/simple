package cmd

import (
	"context"
	"net/http"
	"testing"
)

func TestEnableVanity(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":[{"id":1,"name":"ns1.example.com","ipv4":"127.0.0.1","ipv6":"::1"}]}`)
	got, err := enableVanity(context.Background(), c, "1010", "example.com")
	if err != nil || len(got) != 1 || got[0].Name != "ns1.example.com" {
		t.Fatalf("got %+v, %v", got, err)
	}
	expectOne(t, seen, http.MethodPut, "/v2/1010/vanity/example.com")
}

func TestDisableVanity(t *testing.T) {
	c, seen := recordingClient(t, 204, "")
	if err := disableVanity(context.Background(), c, "1010", "example.com"); err != nil {
		t.Fatal(err)
	}
	expectOne(t, seen, http.MethodDelete, "/v2/1010/vanity/example.com")
}
