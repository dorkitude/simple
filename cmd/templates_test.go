package cmd

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/spf13/cobra"
)

// tplRequest is what a fake API server saw.
type tplRequest struct {
	Method, Path, Query string
	Body                map[string]interface{}
	RawBody             string
}

// tplServer returns a fake API client that records the last request and
// replies with status and body.
func tplServer(t *testing.T, status int, body string) (*tplRequest, *dnsimple.Client) {
	t.Helper()
	got := &tplRequest{}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		got.Method, got.Path, got.Query, got.RawBody = r.Method, r.URL.Path, r.URL.RawQuery, string(raw)
		got.Body = nil
		if len(strings.TrimSpace(string(raw))) > 0 {
			_ = json.Unmarshal(raw, &got.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	})
	return got, c
}

func (g *tplRequest) expect(t *testing.T, method, path string) {
	t.Helper()
	if g.Method != method || g.Path != path {
		t.Errorf("got %s %s, want %s %s", g.Method, g.Path, method, path)
	}
}

const tplJSON = `{"data":{"id":1,"sid":"web","account_id":1010,"name":"Web","description":"A + www"}}`

func TestListTemplates(t *testing.T) {
	got, c := tplServer(t, 200, `{"data":[{"id":1,"sid":"web","name":"Web"}],"pagination":{}}`)
	tpls, err := listTemplates(context.Background(), c, "1010", tplPageOptions(2, 5))
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/v2/1010/templates")
	if !strings.Contains(got.Query, "page=2") || !strings.Contains(got.Query, "per_page=5") {
		t.Errorf("query = %q", got.Query)
	}
	if len(tpls) != 1 || tpls[0].SID != "web" {
		t.Errorf("unexpected templates %+v", tpls)
	}
}

func TestGetTemplate(t *testing.T) {
	got, c := tplServer(t, 200, tplJSON)
	tpl, err := getTemplate(context.Background(), c, "1010", "web")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/v2/1010/templates/web")
	if tpl.Name != "Web" || tpl.ID != 1 {
		t.Errorf("unexpected template %+v", tpl)
	}
}

func TestCreateTemplate(t *testing.T) {
	got, c := tplServer(t, 201, tplJSON)
	if _, err := createTemplate(context.Background(), c, "1010", "Web", "web", "A + www"); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/v2/1010/templates")
	if got.Body["name"] != "Web" || got.Body["sid"] != "web" || got.Body["description"] != "A + www" {
		t.Errorf("body = %v", got.Body)
	}
}

func TestUpdateTemplateSendsOnlyChangedFields(t *testing.T) {
	got, c := tplServer(t, 200, tplJSON)
	if _, err := updateTemplate(context.Background(), c, "1010", "web", map[string]string{"description": ""}); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPatch, "/v2/1010/templates/web")
	if len(got.Body) != 1 {
		t.Errorf("body should only carry description, got %v", got.Body)
	}
	if v, ok := got.Body["description"]; !ok || v != "" {
		t.Errorf("empty description must be sent to clear it, body = %v", got.Body)
	}
}

func TestUpdateTemplateRequiresAField(t *testing.T) {
	_, c := tplServer(t, 200, tplJSON)
	if _, err := updateTemplate(context.Background(), c, "1010", "web", nil); err == nil {
		t.Fatal("expected error with no fields")
	}
}

func TestDeleteTemplate(t *testing.T) {
	got, c := tplServer(t, 204, "")
	if err := deleteTemplate(context.Background(), c, "1010", "web"); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodDelete, "/v2/1010/templates/web")
}

func TestApplyTemplate(t *testing.T) {
	got, c := tplServer(t, 204, "")
	if err := applyTemplate(context.Background(), c, "1010", "web", "example.com"); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/v2/1010/domains/example.com/templates/web")
}

func TestApplyTemplateError(t *testing.T) {
	_, c := tplServer(t, 404, `{"message":"Template not found"}`)
	err := applyTemplate(context.Background(), c, "1010", "nope", "example.com")
	if err == nil || !strings.Contains(err.Error(), "Template not found") {
		t.Fatalf("expected API error, got %v", err)
	}
}

func newYesCmd(yes bool) *cobra.Command {
	c := &cobra.Command{Use: "x"}
	c.Flags().Bool("yes", false, "")
	if yes {
		_ = c.Flags().Set("yes", "true")
	}
	return c
}

func TestTplConfirm(t *testing.T) {
	err := tplConfirm(newYesCmd(false), "delete template 'web'")
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("without --yes expected refusal, got %v", err)
	}
	if err := tplConfirm(newYesCmd(true), "delete template 'web'"); err != nil {
		t.Fatalf("with --yes expected nil, got %v", err)
	}
}

// Destructive commands must refuse before touching the API when --yes is absent.
func TestDestructiveCommandsRequireYes(t *testing.T) {
	cases := []struct {
		cmd  *cobra.Command
		args []string
	}{
		{templatesDeleteCmd, []string{"web"}},
		{templateRecordsDeleteCmd, []string{"web", "5"}},
		{servicesApplyCmd, []string{"heroku", "example.com"}},
		{servicesUnapplyCmd, []string{"heroku", "example.com"}},
	}
	for _, tc := range cases {
		_ = tc.cmd.Flags().Set("yes", "false")
		err := tc.cmd.RunE(tc.cmd, tc.args)
		if err == nil || !strings.Contains(err.Error(), "--yes") {
			t.Errorf("%s: expected --yes refusal, got %v", tc.cmd.CommandPath(), err)
		}
	}
}
