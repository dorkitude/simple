package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

const batchFile = `{
  "creates": [{"type":"MX","name":"","content":"mx.example.com","priority":0,"ttl":300}],
  "updates": [{"id":123,"content":"5.6.7.8"}],
  "deletes": [456, {"id":789}, "1000"]
}`

func TestParseBatchFile(t *testing.T) {
	b, err := parseBatchFile(strings.NewReader(batchFile))
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Creates) != 1 || *b.Creates[0].Priority != 0 || len(b.Updates) != 1 || b.Updates[0].ID != 123 {
		t.Fatalf("parsed %+v", b)
	}
	if len(b.Deletes) != 3 || b.Deletes[0].ID != 456 || b.Deletes[1].ID != 789 || b.Deletes[2].ID != 1000 {
		t.Fatalf("deletes %+v", b.Deletes)
	}
}

func TestParseBatchFileRejectsTypos(t *testing.T) {
	if _, err := parseBatchFile(strings.NewReader(`{"create":[{"type":"A"}]}`)); err == nil {
		t.Error("expected error for unknown key 'create'")
	}
	if _, err := parseBatchFile(strings.NewReader(`{"creates":[{"typ":"A"}]}`)); err == nil {
		t.Error("expected error for unknown record field 'typ'")
	}
}

func TestParseBatchDeletes(t *testing.T) {
	for in, want := range map[string][]int64{
		"1,2, 3":         {1, 2, 3},
		"[4,{\"id\":5}]": {4, 5},
		"{\"id\":6}":     {6},
		"":               nil,
	} {
		got, err := parseBatchDeletes(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if len(got) != len(want) {
			t.Fatalf("%q: got %+v", in, got)
		}
		for i := range want {
			if got[i].ID != want[i] {
				t.Errorf("%q: got %+v", in, got)
			}
		}
	}
	if _, err := parseBatchDeletes("1,x"); err == nil {
		t.Error("expected error")
	}
}

func TestValidateBatch(t *testing.T) {
	bad := []string{
		`{}`,
		`{"creates":[{"type":"A"}]}`,
		`{"creates":[{"id":1,"type":"A","content":"1.2.3.4"}]}`,
		`{"updates":[{"content":"x"}]}`,
		`{"deletes":[0]}`,
	}
	for _, s := range bad {
		b, err := parseBatchFile(strings.NewReader(s))
		if err != nil {
			t.Fatalf("%s: %v", s, err)
		}
		if validateBatch(b) == nil {
			t.Errorf("%s: expected validation error", s)
		}
	}
}

func TestBatchFromFlags(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "changes.json")
	if err := os.WriteFile(path, []byte(`{"deletes":[1]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	c := &cobra.Command{Use: "x"}
	addBatchFlags(c)
	err := c.ParseFlags([]string{
		"--file", path,
		"--create", `{"type":"A","name":"www","content":"1.2.3.4"}`,
		"--create", `[{"type":"TXT","name":"","content":"v=spf1 -all"}]`,
		"--update", `{"id":9,"ttl":0}`,
		"--delete", "2,3",
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := batchFromFlags(c, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Creates) != 2 || len(b.Updates) != 1 || len(b.Deletes) != 3 {
		t.Fatalf("got %+v", b)
	}
	if b.Updates[0].TTL == nil || *b.Updates[0].TTL != 0 {
		t.Errorf("ttl 0 dropped: %+v", b.Updates[0])
	}
}

func TestBatchFromStdin(t *testing.T) {
	c := &cobra.Command{Use: "x"}
	addBatchFlags(c)
	if err := c.ParseFlags([]string{"--file", "-"}); err != nil {
		t.Fatal(err)
	}
	b, err := batchFromFlags(c, strings.NewReader(batchFile))
	if err != nil || len(b.Deletes) != 3 {
		t.Fatalf("got %+v, %v", b, err)
	}
}

func TestBatchChangeRecords(t *testing.T) {
	c, seen := recordingClient(t, 200, `{"data":{"creates":[{"id":1,"type":"MX","content":"mx.example.com"}],"updates":[{"id":123,"type":"A","content":"5.6.7.8"}],"deletes":[{"id":456}]}}`)
	b, err := parseBatchFile(strings.NewReader(batchFile))
	if err != nil {
		t.Fatal(err)
	}
	res, err := batchChangeRecords(context.Background(), c, "1010", "example.com", b)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Creates) != 1 || len(res.Updates) != 1 || len(res.Deletes) != 1 {
		t.Fatalf("result %+v", res)
	}
	r := expectOne(t, seen, http.MethodPost, "/v2/1010/zones/example.com/batch")
	var body struct {
		Creates []map[string]any `json:"creates"`
		Updates []map[string]any `json:"updates"`
		Deletes []map[string]any `json:"deletes"`
	}
	if err := json.Unmarshal([]byte(r.Raw), &body); err != nil {
		t.Fatal(err)
	}
	if body.Creates[0]["priority"] != float64(0) {
		t.Errorf("priority 0 not sent: %s", r.Raw)
	}
	if _, ok := body.Creates[0]["id"]; ok {
		t.Errorf("create must not carry an id: %s", r.Raw)
	}
	if body.Updates[0]["id"] != float64(123) || len(body.Deletes) != 3 || body.Deletes[2]["id"] != float64(1000) {
		t.Errorf("body %s", r.Raw)
	}
}

func TestBatchChangeRecordsInvalidNotSent(t *testing.T) {
	c, seen := recordingClient(t, 200, `{}`)
	if _, err := batchChangeRecords(context.Background(), c, "1010", "example.com", BatchChange{}); err == nil {
		t.Fatal("expected error")
	}
	if len(*seen) != 0 {
		t.Fatal("empty batch must not reach the API")
	}
}
