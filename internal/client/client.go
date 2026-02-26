package client

import (
	"context"
	"fmt"
	"strconv"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/dorkitude/simple/internal/config"
)

// App holds the authenticated DNSimple client and resolved account ID.
type App struct {
	Client    *dnsimple.Client
	AccountID string
}

// New creates an App from the stored token and config.
func New(ctx context.Context) (*App, error) {
	token, err := config.LoadToken()
	if err != nil {
		return nil, err
	}
	return newFromToken(ctx, token, "", false)
}

// NewFromFlags creates an App with optional overrides from CLI flags.
func NewFromFlags(ctx context.Context, accountOverride string, sandbox bool) (*App, error) {
	token, err := config.LoadToken()
	if err != nil {
		return nil, err
	}
	return newFromToken(ctx, token, accountOverride, sandbox)
}

func newFromToken(ctx context.Context, token, accountOverride string, sandbox bool) (*App, error) {
	tc := dnsimple.StaticTokenHTTPClient(ctx, token)
	c := dnsimple.NewClient(tc)
	c.SetUserAgent("dnsimplectl")

	if sandbox {
		c.BaseURL = "https://api.sandbox.dnsimple.com"
	}

	// Resolve account ID
	accountID := accountOverride
	if accountID == "" {
		cfg, _ := config.Load()
		if cfg != nil && cfg.AccountID != "" {
			accountID = cfg.AccountID
		}
	}

	if accountID == "" {
		// Look it up via Whoami
		whoami, err := c.Identity.Whoami(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to identify account: %w", err)
		}
		if whoami.Data.Account != nil {
			accountID = strconv.FormatInt(whoami.Data.Account.ID, 10)
		} else if whoami.Data.User != nil {
			// User token — need to list accounts and pick one
			accts, err := c.Accounts.ListAccounts(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to list accounts: %w", err)
			}
			if len(accts.Data) == 0 {
				return nil, fmt.Errorf("no accounts found for this user")
			}
			if len(accts.Data) == 1 {
				accountID = strconv.FormatInt(accts.Data[0].ID, 10)
			} else {
				// For now, use the first account. TUI can let user pick later.
				accountID = strconv.FormatInt(accts.Data[0].ID, 10)
			}
		} else {
			return nil, fmt.Errorf("whoami returned neither account nor user")
		}

		// Cache it
		cfg, _ := config.Load()
		if cfg == nil {
			cfg = &config.Config{}
		}
		cfg.AccountID = accountID
		_ = config.Save(cfg)
	}

	return &App{Client: c, AccountID: accountID}, nil
}

// ValidateToken checks if a token is valid by calling Whoami.
func ValidateToken(ctx context.Context, token string, sandbox bool) (*dnsimple.WhoamiData, error) {
	tc := dnsimple.StaticTokenHTTPClient(ctx, token)
	c := dnsimple.NewClient(tc)
	if sandbox {
		c.BaseURL = "https://api.sandbox.dnsimple.com"
	}

	resp, err := c.Identity.Whoami(ctx)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	return resp.Data, nil
}
