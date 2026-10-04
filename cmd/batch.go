package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

// BatchChange is the body of POST /v2/{account}/zones/{zone}/batch.
//
// It mirrors dnsimple.BatchChangeZoneRecordsRequest but uses RecordFields, so a
// TTL or priority of 0 is sent instead of being dropped by omitempty. The call
// goes through Client.Request for the same reason.
type BatchChange struct {
	Creates []RecordFields `json:"creates,omitempty"`
	Updates []RecordFields `json:"updates,omitempty"`
	Deletes []BatchDelete  `json:"deletes,omitempty"`
}

// BatchDelete names a record to delete. In input it may be written as a bare
// ID (42) or an object ({"id": 42}).
type BatchDelete struct {
	ID int64 `json:"id"`
}

// UnmarshalJSON accepts 42, "42", or {"id": 42}.
func (d *BatchDelete) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) > 0 && b[0] == '{' {
		var obj struct {
			ID int64 `json:"id"`
		}
		if err := json.Unmarshal(b, &obj); err != nil {
			return err
		}
		d.ID = obj.ID
		return nil
	}
	s := strings.Trim(string(b), `"`)
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid delete entry %s (want an ID or {\"id\": ...})", string(b))
	}
	d.ID = id
	return nil
}

// decodeOneOrMany decodes a JSON object or an array of objects into out.
func decodeOneOrMany[T any](raw string) ([]T, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "[") {
		var many []T
		if err := strictUnmarshal([]byte(raw), &many); err != nil {
			return nil, err
		}
		return many, nil
	}
	var one T
	if err := strictUnmarshal([]byte(raw), &one); err != nil {
		return nil, err
	}
	return []T{one}, nil
}

func strictUnmarshal(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("unexpected data after JSON value")
	}
	return nil
}

// parseBatchDeletes accepts "1,2,3" or JSON (an ID, an object, or an array).
func parseBatchDeletes(raw string) ([]BatchDelete, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if strings.HasPrefix(raw, "[") || strings.HasPrefix(raw, "{") {
		return decodeOneOrMany[BatchDelete](raw)
	}
	var out []BatchDelete
	for _, part := range strings.Split(raw, ",") {
		id, err := parseID("record", strings.TrimSpace(part))
		if err != nil {
			return nil, err
		}
		out = append(out, BatchDelete{ID: id})
	}
	return out, nil
}

// parseBatchFile decodes a {"creates":[],"updates":[],"deletes":[]} document.
func parseBatchFile(r io.Reader) (BatchChange, error) {
	var b BatchChange
	data, err := io.ReadAll(r)
	if err != nil {
		return b, err
	}
	if err := strictUnmarshal(data, &b); err != nil {
		return b, fmt.Errorf("invalid batch file: %w", err)
	}
	return b, nil
}

// validateBatch checks each entry has what the API needs.
func validateBatch(b BatchChange) error {
	if len(b.Creates)+len(b.Updates)+len(b.Deletes) == 0 {
		return fmt.Errorf("batch is empty: give --file, --create, --update, or --delete")
	}
	for i, c := range b.Creates {
		if c.ID != 0 {
			return fmt.Errorf("creates[%d]: has an id; put it under updates instead", i)
		}
		if c.Type == "" || c.Content == "" {
			return fmt.Errorf("creates[%d]: type and content are required", i)
		}
	}
	for i, u := range b.Updates {
		if u.ID <= 0 {
			return fmt.Errorf("updates[%d]: id is required", i)
		}
	}
	for i, d := range b.Deletes {
		if d.ID <= 0 {
			return fmt.Errorf("deletes[%d]: id is required", i)
		}
	}
	return nil
}

// batchChangeRecords applies a batch of record changes to a zone atomically.
func batchChangeRecords(ctx context.Context, c *dnsimple.Client, accountID, zone string, b BatchChange) (*dnsimple.BatchChangeZoneRecordsData, error) {
	if err := validateBatch(b); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/v2/%s/zones/%s/batch", url.PathEscape(accountID), url.PathEscape(zone))
	var out dnsimple.BatchChangeZoneRecordsResponse
	if _, err := c.Request(ctx, http.MethodPost, path, b, &out, nil); err != nil {
		return nil, fmt.Errorf("batch change for '%s' failed (nothing was applied): %w", zone, err)
	}
	if out.Data == nil {
		return &dnsimple.BatchChangeZoneRecordsData{}, nil
	}
	return out.Data, nil
}

// batchFromFlags merges --file with --create/--update/--delete.
func batchFromFlags(cmd *cobra.Command, stdin io.Reader) (BatchChange, error) {
	var b BatchChange
	if file, _ := cmd.Flags().GetString("file"); file != "" {
		var r io.Reader = stdin
		if file != "-" {
			f, err := os.Open(file)
			if err != nil {
				return b, err
			}
			defer f.Close()
			r = f
		}
		parsed, err := parseBatchFile(r)
		if err != nil {
			return b, err
		}
		b = parsed
	}
	creates, _ := cmd.Flags().GetStringArray("create")
	for _, raw := range creates {
		items, err := decodeOneOrMany[RecordFields](raw)
		if err != nil {
			return b, fmt.Errorf("invalid --create %q: %w", raw, err)
		}
		b.Creates = append(b.Creates, items...)
	}
	updates, _ := cmd.Flags().GetStringArray("update")
	for _, raw := range updates {
		items, err := decodeOneOrMany[RecordFields](raw)
		if err != nil {
			return b, fmt.Errorf("invalid --update %q: %w", raw, err)
		}
		b.Updates = append(b.Updates, items...)
	}
	deletes, _ := cmd.Flags().GetStringArray("delete")
	for _, raw := range deletes {
		items, err := parseBatchDeletes(raw)
		if err != nil {
			return b, fmt.Errorf("invalid --delete %q: %w", raw, err)
		}
		b.Deletes = append(b.Deletes, items...)
	}
	return b, nil
}

func describeRecordFields(r RecordFields) string {
	name := "@"
	if r.Name != nil && *r.Name != "" {
		name = *r.Name
	}
	s := strings.TrimSpace(fmt.Sprintf("%s %s %s", r.Type, name, r.Content))
	if r.TTL != nil {
		s += fmt.Sprintf(" ttl=%d", *r.TTL)
	}
	if r.Priority != nil {
		s += fmt.Sprintf(" pri=%d", *r.Priority)
	}
	if len(r.Regions) > 0 {
		s += " regions=" + strings.Join(r.Regions, ",")
	}
	return s
}

func printBatchPlan(zone string, b BatchChange) {
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🧺 Batch for %s: %d create, %d update, %d delete",
		zone, len(b.Creates), len(b.Updates), len(b.Deletes))))
	for _, c := range b.Creates {
		fmt.Printf("  %s %s\n", ui.SuccessStyle.Render("+"), describeRecordFields(c))
	}
	for _, u := range b.Updates {
		fmt.Printf("  %s %d → %s\n", ui.AccentStyle.Render("~"), u.ID, describeRecordFields(u))
	}
	for _, d := range b.Deletes {
		fmt.Printf("  %s %d\n", ui.ErrorStyle.Render("-"), d.ID)
	}
}

var zonesBatchCmd = &cobra.Command{
	Use:   "batch [zone]",
	Short: "Create, update, and delete records in one atomic change",
	Long: `Apply many record changes to a zone in one request. DNSimple applies all of
them or none.

The file is JSON: {"creates": [...], "updates": [...], "deletes": [...]}.
Creates and updates take type, name, content, ttl, priority, regions; updates
and deletes need the record id. A delete may be written as a bare id.
--create and --update take a JSON object or array and may repeat; --delete
takes ids ("1,2,3") or JSON. They add to --file. Use --file - for stdin.

--dry-run prints the parsed batch and sends nothing. A batch with deletes
requires --yes.

Examples:
  simple zones batch example.com --file changes.json --dry-run
  simple zones batch example.com --file changes.json --yes
  simple zones batch example.com \
    --create '{"type":"A","name":"www","content":"1.2.3.4","ttl":300}' \
    --update '{"id":123,"content":"5.6.7.8"}' \
    --delete 456,789 --yes
  cat changes.json | simple zones batch example.com --file - --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		zone := args[0]
		b, err := batchFromFlags(cmd, cmd.InOrStdin())
		if err != nil {
			return err
		}
		if err := validateBatch(b); err != nil {
			return err
		}

		if dryRun, _ := cmd.Flags().GetBool("dry-run"); dryRun {
			if jsonOutput {
				printJSON(b)
				return nil
			}
			printBatchPlan(zone, b)
			fmt.Println()
			fmt.Println(ui.Info("Dry run: nothing was sent."))
			return nil
		}

		if len(b.Deletes) > 0 {
			ids := make([]string, len(b.Deletes))
			for i, d := range b.Deletes {
				ids[i] = fmt.Sprint(d.ID)
			}
			if yes, _ := cmd.Flags().GetBool("yes"); !yes {
				printBatchPlan(zone, b)
				fmt.Println()
			}
			if err := requireYes(cmd, fmt.Sprintf("permanently delete %d records from '%s' (%s)",
				len(b.Deletes), zone, strings.Join(ids, ", "))); err != nil {
				return err
			}
		}

		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		res, err := batchChangeRecords(ctx, app.Client, app.AccountID, zone, b)
		if err != nil {
			return err
		}
		if printJSON(res) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Batch applied to '%s': %d created, %d updated, %d deleted",
			zone, len(res.Creates), len(res.Updates), len(res.Deletes))))
		for _, r := range res.Creates {
			fmt.Printf("  %s %s %s %s\n", ui.SuccessStyle.Render("+"), ui.SubtleStyle.Render(fmt.Sprint(r.ID)), r.Type, r.Content)
		}
		for _, r := range res.Updates {
			fmt.Printf("  %s %s %s %s\n", ui.AccentStyle.Render("~"), ui.SubtleStyle.Render(fmt.Sprint(r.ID)), r.Type, r.Content)
		}
		for _, d := range res.Deletes {
			fmt.Printf("  %s %d\n", ui.ErrorStyle.Render("-"), d.ID)
		}
		return nil
	},
}

func init() {
	zonesCmd.AddCommand(zonesBatchCmd)
	addBatchFlags(zonesBatchCmd)
}

func addBatchFlags(c *cobra.Command) {
	c.Flags().StringP("file", "f", "", "JSON batch file (- for stdin)")
	c.Flags().StringArray("create", nil, "Record to create, as JSON (object or array; repeatable)")
	c.Flags().StringArray("update", nil, "Record update with id, as JSON (object or array; repeatable)")
	c.Flags().StringArray("delete", nil, "Record IDs to delete (1,2,3 or JSON; repeatable)")
	c.Flags().Bool("dry-run", false, "Print the parsed batch without sending it")
	addYesFlag(c, "required when the batch deletes records")
}
