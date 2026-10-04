package cmd

import (
	"context"
	"net/http"
	"testing"
)

func TestParseServiceSettings(t *testing.T) {
	got, err := parseServiceSettings([]string{"app=my-app", "token=a=b", "empty="})
	if err != nil {
		t.Fatal(err)
	}
	if got["app"] != "my-app" || got["token"] != "a=b" || got["empty"] != "" || len(got) != 3 {
		t.Errorf("unexpected settings %v", got)
	}
	for _, bad := range []string{"novalue", "=x"} {
		if _, err := parseServiceSettings([]string{bad}); err == nil {
			t.Errorf("parseServiceSettings(%q) expected error", bad)
		}
	}
}

func TestListServices(t *testing.T) {
	got, c := tplServer(t, 200, `{"data":[{"id":1,"sid":"heroku","name":"Heroku","requires_setup":true}],"pagination":{}}`)
	svcs, err := listServices(context.Background(), c, nil)
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/v2/services")
	if len(svcs) != 1 || svcs[0].SID != "heroku" || !svcs[0].RequiresSetup {
		t.Errorf("unexpected services %+v", svcs)
	}
}

func TestGetService(t *testing.T) {
	got, c := tplServer(t, 200, `{"data":{"id":1,"sid":"heroku","name":"Heroku","settings":[{"name":"app","label":"App"}]}}`)
	s, err := getService(context.Background(), c, "heroku")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/v2/services/heroku")
	if len(s.Settings) != 1 || s.Settings[0].Name != "app" {
		t.Errorf("unexpected service %+v", s)
	}
}

func TestListAppliedServices(t *testing.T) {
	got, c := tplServer(t, 200, `{"data":[],"pagination":{}}`)
	if _, err := listAppliedServices(context.Background(), c, "1010", "example.com", nil); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/v2/1010/domains/example.com/services")
}

func TestApplyServiceWithSettings(t *testing.T) {
	got, c := tplServer(t, 204, "")
	if err := applyService(context.Background(), c, "1010", "heroku", "example.com", map[string]string{"app": "my-app"}); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/v2/1010/domains/example.com/services/heroku")
	settings, ok := got.Body["settings"].(map[string]interface{})
	if !ok || settings["app"] != "my-app" {
		t.Errorf("body must be {\"settings\":{...}} (lowercase key), got %s", got.RawBody)
	}
}

func TestApplyServiceWithoutSettings(t *testing.T) {
	got, c := tplServer(t, 204, "")
	if err := applyService(context.Background(), c, "1010", "google-workspace", "example.com", nil); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/v2/1010/domains/example.com/services/google-workspace")
	if got.Body != nil {
		t.Errorf("expected empty body, got %s", got.RawBody)
	}
}

func TestUnapplyService(t *testing.T) {
	got, c := tplServer(t, 204, "")
	if err := unapplyService(context.Background(), c, "1010", "heroku", "example.com"); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodDelete, "/v2/1010/domains/example.com/services/heroku")
}
