package cmd

import (
	"context"
	"net/http"
	"testing"
)

const fwdJSON = `{"id":17,"domain_id":1,"alias_email":"hello@example.com","destination_email":"me@gmail.com","active":true}`

func TestListEmailForwards(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":[`+fwdJSON+`],"pagination":{}}`)
	got, err := listEmailForwards(context.Background(), c, "1010", "example.com", nil)
	if err != nil || len(got) != 1 || got[0].AliasEmail != "hello@example.com" {
		t.Fatalf("got %+v, %v", got, err)
	}
	expectOne(t, seen, http.MethodGet, "/v2/1010/domains/example.com/email_forwards")
}

func TestGetEmailForward(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":`+fwdJSON+`}`)
	if _, err := getEmailForward(context.Background(), c, "1010", "example.com", 17); err != nil {
		t.Fatal(err)
	}
	expectOne(t, seen, http.MethodGet, "/v2/1010/domains/example.com/email_forwards/17")
}

func TestCreateEmailForward(t *testing.T) {
	for _, from := range []string{"hello", "hello@example.com", "hello@EXAMPLE.com"} {
		c, seen := recordingClient(t, 201, `{"data":`+fwdJSON+`}`)
		if _, err := createEmailForward(context.Background(), c, "1010", "example.com", from, "me@gmail.com"); err != nil {
			t.Fatalf("from %q: %v", from, err)
		}
		r := expectOne(t, seen, http.MethodPost, "/v2/1010/domains/example.com/email_forwards")
		if r.Body["alias_name"] != "hello" || r.Body["destination_email"] != "me@gmail.com" {
			t.Errorf("from %q: body %s", from, r.Raw)
		}
	}
}

func TestCreateEmailForwardValidation(t *testing.T) {
	c, seen := recordingClient(t, 201, `{}`)
	if _, err := createEmailForward(context.Background(), c, "1010", "example.com", "hello@other.com", "me@gmail.com"); err == nil {
		t.Error("expected error for alias on another domain")
	}
	if _, err := createEmailForward(context.Background(), c, "1010", "example.com", "hello", ""); err == nil {
		t.Error("expected error for missing destination")
	}
	if len(*seen) != 0 {
		t.Fatalf("invalid input must not reach the API: %+v", *seen)
	}
}

func TestDeleteEmailForward(t *testing.T) {
	c, seen := recordingClient(t, 204, "")
	if err := deleteEmailForward(context.Background(), c, "1010", "example.com", 17); err != nil {
		t.Fatal(err)
	}
	expectOne(t, seen, http.MethodDelete, "/v2/1010/domains/example.com/email_forwards/17")
}
