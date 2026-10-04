package cmd

import (
	"context"
	"net/http"
	"testing"
)

const tplRecordJSON = `{"data":{"id":301,"template_id":1,"name":"","content":"1.2.3.4","ttl":600,"type":"A","priority":0}}`

func TestListTemplateRecords(t *testing.T) {
	got, c := tplServer(t, 200, `{"data":[{"id":301,"template_id":1,"name":"www","content":"example.com","ttl":3600,"type":"CNAME"}],"pagination":{}}`)
	recs, err := listTemplateRecords(context.Background(), c, "1010", "web", nil)
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/v2/1010/templates/web/records")
	if len(recs) != 1 || recs[0].Type != "CNAME" {
		t.Errorf("unexpected records %+v", recs)
	}
}

func TestGetTemplateRecord(t *testing.T) {
	got, c := tplServer(t, 200, tplRecordJSON)
	r, err := getTemplateRecord(context.Background(), c, "1010", "web", 301)
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodGet, "/v2/1010/templates/web/records/301")
	if r.ID != 301 || r.Content != "1.2.3.4" {
		t.Errorf("unexpected record %+v", r)
	}
}

func TestCreateTemplateRecordSendsExplicitZero(t *testing.T) {
	got, c := tplServer(t, 201, tplRecordJSON)
	zero, ttl := 0, 600
	_, err := createTemplateRecord(context.Background(), c, "1010", "web", templateRecordInput{
		Type: "MX", Name: "", Content: "mx.example.com", TTL: &ttl, Priority: &zero,
	})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodPost, "/v2/1010/templates/web/records")
	if got.Body["type"] != "MX" || got.Body["content"] != "mx.example.com" {
		t.Errorf("body = %v", got.Body)
	}
	if v, ok := got.Body["name"]; !ok || v != "" {
		t.Errorf("apex name must be sent as empty string, body = %v", got.Body)
	}
	if v, ok := got.Body["priority"]; !ok || v != float64(0) {
		t.Errorf("explicit priority 0 must be sent, body = %v", got.Body)
	}
	if got.Body["ttl"] != float64(600) {
		t.Errorf("ttl = %v", got.Body["ttl"])
	}
}

func TestCreateTemplateRecordOmitsUnsetOptionals(t *testing.T) {
	got, c := tplServer(t, 201, tplRecordJSON)
	if _, err := createTemplateRecord(context.Background(), c, "1010", "web", templateRecordInput{Type: "A", Content: "1.2.3.4"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Body["ttl"]; ok {
		t.Errorf("unset ttl must be omitted, body = %v", got.Body)
	}
	if _, ok := got.Body["priority"]; ok {
		t.Errorf("unset priority must be omitted, body = %v", got.Body)
	}
}

func TestDeleteTemplateRecord(t *testing.T) {
	got, c := tplServer(t, 204, "")
	if err := deleteTemplateRecord(context.Background(), c, "1010", "web", 301); err != nil {
		t.Fatal(err)
	}
	got.expect(t, http.MethodDelete, "/v2/1010/templates/web/records/301")
}

func TestParseTemplateRecordID(t *testing.T) {
	if id, err := parseTemplateRecordID("42"); err != nil || id != 42 {
		t.Errorf("got %d, %v", id, err)
	}
	if _, err := parseTemplateRecordID("abc"); err == nil {
		t.Error("expected error for non-numeric ID")
	}
}
