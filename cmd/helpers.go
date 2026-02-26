package cmd

import (
	"context"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/dorkitude/simple/internal/client"
	"github.com/dorkitude/simple/internal/output"
)

// getApp returns an authenticated App from stored credentials and flags.
func getApp(ctx context.Context) (*client.App, error) {
	return client.NewFromFlags(ctx, accountFlag, sandboxFlag)
}

// printJSON outputs v as JSON if --json flag is set, returns true if it did.
func printJSON(v interface{}) bool {
	if jsonOutput {
		output.JSON(v)
		return true
	}
	return false
}

// dnsimpleInt returns a pointer to an int (for SDK optional fields).
func dnsimpleInt(v int) *int {
	return dnsimple.Int(v)
}

// dnsimpleString returns a pointer to a string (for SDK optional fields).
func dnsimpleString(v string) *string {
	return dnsimple.String(v)
}

// truncate shortens a string to maxLen, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
