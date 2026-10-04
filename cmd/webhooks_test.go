package cmd

import (
	"context"
	"net/http"
	"testing"
)

func TestListWebhooks(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":[{"id":7,"url":"https://example.com/hook"}]}`)
	hooks, err := listWebhooks(context.Background(), c, "1010")
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/1010/webhooks")
	if len(hooks) != 1 || hooks[0].URL != "https://example.com/hook" {
		t.Errorf("unexpected hooks %+v", hooks)
	}
}

func TestGetWebhook(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":{"id":7,"url":"https://example.com/hook"}}`)
	h, err := getWebhook(context.Background(), c, "1010", 7)
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/1010/webhooks/7")
	if h.ID != 7 {
		t.Errorf("unexpected hook %+v", h)
	}
}

func TestCreateWebhook(t *testing.T) {
	c, calls := recordingClient(t, 201, `{"data":{"id":8,"url":"https://example.com/new"}}`)
	h, err := createWebhook(context.Background(), c, "1010", "https://example.com/new")
	if err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodPost, "/v2/1010/webhooks")
	if got.Body["url"] != "https://example.com/new" {
		t.Errorf("unexpected body %+v", got.Body)
	}
	if h.ID != 8 {
		t.Errorf("unexpected hook %+v", h)
	}
}

func TestCreateWebhookRejectsBadURL(t *testing.T) {
	c, calls := recordingClient(t, 201, `{}`)
	for _, bad := range []string{"example.com/hook", "ftp://example.com", "https://"} {
		if _, err := createWebhook(context.Background(), c, "1010", bad); err == nil {
			t.Errorf("createWebhook(%q) expected error", bad)
		}
	}
	if len(*calls) != 0 {
		t.Errorf("bad URLs must not reach the API, got %d calls", len(*calls))
	}
}

func TestDeleteWebhook(t *testing.T) {
	c, calls := recordingClient(t, 204, "")
	if err := deleteWebhook(context.Background(), c, "1010", 7); err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodDelete, "/v2/1010/webhooks/7")
}

func TestDeleteWebhookRequiresYes(t *testing.T) {
	err := webhooksDeleteCmd.RunE(webhooksDeleteCmd, []string{"7"})
	if _, ok := err.(errNeedsYes); !ok {
		t.Fatalf("expected errNeedsYes, got %v", err)
	}
}
