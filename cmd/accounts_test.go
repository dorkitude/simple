package cmd

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/dorkitude/simple/internal/config"
)

const accountsJSON = `{"data":[{"id":1010,"email":"a@example.com","plan_identifier":"teams-v1-monthly"},{"id":2020,"email":"b@example.com"}]}`

func TestListAccounts(t *testing.T) {
	c, calls := recordingClient(t, 200, accountsJSON)
	accts, err := listAccounts(context.Background(), c)
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/accounts")
	if len(accts) != 2 || accts[1].ID != 2020 {
		t.Errorf("unexpected accounts %+v", accts)
	}
}

func TestFindAccount(t *testing.T) {
	c, _ := recordingClient(t, 200, accountsJSON)
	a, err := findAccount(context.Background(), c, 2020)
	if err != nil || a.Email != "b@example.com" {
		t.Fatalf("findAccount(2020) = %+v, %v", a, err)
	}
	if _, err := findAccount(context.Background(), c, 9999); err == nil || !strings.Contains(err.Error(), "not accessible") {
		t.Fatalf("expected not-accessible error, got %v", err)
	}
}

func TestSaveActiveAccountPreservesConfig(t *testing.T) {
	t.Setenv("DNSIMPLE_CONFIG_DIR", t.TempDir())
	if err := config.Save(&config.Config{AccountID: "1010", Sandbox: true}); err != nil {
		t.Fatal(err)
	}
	if err := saveActiveAccount(2020); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.AccountID != "2020" || !cfg.Sandbox {
		t.Errorf("unexpected config %+v", cfg)
	}
}
