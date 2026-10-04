package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/spf13/cobra"
)

// regReq is what the fake registrar API received.
type regReq struct {
	Method string
	Path   string
	Query  string
	Body   map[string]interface{}
	Raw    string
	Calls  int
}

// regFake returns a client whose fake API answers every request with status and body.
func regFake(t *testing.T, status int, body string) (*dnsimple.Client, *regReq) {
	t.Helper()
	got := &regReq{}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got.Calls++
		got.Method, got.Path, got.Query = r.Method, r.URL.Path, r.URL.RawQuery
		b, _ := io.ReadAll(r.Body)
		got.Raw = string(b)
		got.Body = nil
		if len(b) > 0 {
			_ = json.Unmarshal(b, &got.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if body != "" {
			_, _ = w.Write([]byte(body))
		}
	})
	return c, got
}

func (g *regReq) expect(t *testing.T, method, path string) {
	t.Helper()
	if g.Method != method || g.Path != path {
		t.Errorf("got %s %s, want %s %s", g.Method, g.Path, method, path)
	}
}

var ctxBG = context.Background()

func TestParseExtendedAttributes(t *testing.T) {
	m, err := parseExtendedAttributes([]string{"us_nexus=C11", "x= a=b "})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"us_nexus": "C11", "x": " a=b "}
	if !reflect.DeepEqual(m, want) {
		t.Errorf("got %v, want %v", m, want)
	}
	if m, err := parseExtendedAttributes(nil); err != nil || m != nil {
		t.Errorf("empty input: got %v, %v", m, err)
	}
	for _, bad := range []string{"novalue", "=v"} {
		if _, err := parseExtendedAttributes([]string{bad}); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestParseRegID(t *testing.T) {
	if id, err := parseRegID("transfer", "361"); err != nil || id != 361 {
		t.Errorf("got %d, %v", id, err)
	}
	for _, bad := range []string{"0", "-1", "abc", ""} {
		if _, err := parseRegID("transfer", bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

func TestRegErrorIncludesAttributeErrors(t *testing.T) {
	c, _ := regFake(t, http.StatusBadRequest, `{"message":"Validation failed","errors":{"registrant_id":["is invalid"]}}`)
	_, err := regCheckDomain(ctxBG, c, "1010", "example.com")
	if err == nil || !strings.Contains(err.Error(), "Validation failed (registrant_id: is invalid)") {
		t.Fatalf("got %v", err)
	}
}

func TestRegCheckDomain(t *testing.T) {
	c, got := regFake(t, 200, `{"data":{"domain":"ruby.codes","available":true,"premium":true}}`)
	d, err := regCheckDomain(ctxBG, c, "1010", "ruby.codes")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/domains/ruby.codes/check")
	if !d.Available || !d.Premium || d.Domain != "ruby.codes" {
		t.Errorf("unexpected %+v", d)
	}
	var buf bytes.Buffer
	printDomainCheck(&buf, d)
	for _, s := range []string{"ruby.codes", "Available:", "yes", "Premium:"} {
		if !strings.Contains(buf.String(), s) {
			t.Errorf("output missing %q:\n%s", s, buf.String())
		}
	}
}

func TestRegGetPrices(t *testing.T) {
	c, got := regFake(t, 200, `{"data":{"domain":"bingo.pizza","premium":true,"registration_price":20.0,"renewal_price":21.5,"restore_price":22.0,"transfer_price":23.0}}`)
	p, err := regGetPrices(ctxBG, c, "1010", "bingo.pizza")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/domains/bingo.pizza/prices")
	var buf bytes.Buffer
	printDomainPrices(&buf, p)
	for _, s := range []string{"$20.00", "$21.50", "$22.00", "$23.00"} {
		if !strings.Contains(buf.String(), s) {
			t.Errorf("output missing %q:\n%s", s, buf.String())
		}
	}
	if strings.Contains(buf.String(), "Trustee") {
		t.Error("trustee price shown though absent")
	}
}

func TestRegResearch(t *testing.T) {
	c, got := regFake(t, 200, `{"data":{"request_id":"abc","domain":"taken.com","availability":"unavailable","errors":["oops"]}}`)
	r, err := regResearch(ctxBG, c, "1010", "taken.com")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/domains/research/status")
	if got.Query != "domain=taken.com" {
		t.Errorf("query = %q", got.Query)
	}
	var buf bytes.Buffer
	printResearch(&buf, r)
	if !strings.Contains(buf.String(), "unavailable") || !strings.Contains(buf.String(), "oops") {
		t.Errorf("unexpected output:\n%s", buf.String())
	}
}

func TestRegRegister(t *testing.T) {
	c, got := regFake(t, 201, `{"data":{"id":1,"domain_id":999,"registrant_id":2,"period":1,"state":"new","auto_renew":true,"whois_privacy":false}}`)
	tr := false
	f := &regAcquireFlags{RegistrantID: 2, AutoRenew: true, Trustee: &tr, PremiumPrice: "109.00",
		ExtendedAttributes: map[string]string{"us_nexus": "C11"}}
	r, err := regRegister(ctxBG, c, "1010", "example.com", f)
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "POST", "/v2/1010/registrar/domains/example.com/registrations")
	want := map[string]interface{}{
		"registrant_id": float64(2), "auto_renew": true, "trustee": false,
		"premium_price": "109.00", "extended_attributes": map[string]interface{}{"us_nexus": "C11"},
	}
	if !reflect.DeepEqual(got.Body, want) {
		t.Errorf("body = %v, want %v", got.Body, want)
	}
	if r.ID != 1 || r.State != "new" {
		t.Errorf("unexpected %+v", r)
	}
}

func TestRegGetRegistration(t *testing.T) {
	c, got := regFake(t, 200, `{"data":{"id":361,"domain_id":104040,"registrant_id":2715,"period":1,"state":"registering","auto_renew":false,"whois_privacy":true,"created_at":"2023-01-27T17:44:32Z"}}`)
	r, err := regGetRegistration(ctxBG, c, "1010", "example.com", "361")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/domains/example.com/registrations/361")
	var buf bytes.Buffer
	printRegistration(&buf, "example.com", r)
	for _, s := range []string{"361", "registering", "2715", "1 year(s)", "2023-01-27T17:44:32Z"} {
		if !strings.Contains(buf.String(), s) {
			t.Errorf("output missing %q:\n%s", s, buf.String())
		}
	}
}

func TestRegTransfer(t *testing.T) {
	c, got := regFake(t, 201, `{"data":{"id":361,"domain_id":182245,"registrant_id":2715,"state":"transferring"}}`)
	f := &regAcquireFlags{RegistrantID: 2715, WhoisPrivacy: true}
	tr, err := regTransfer(ctxBG, c, "1010", "example.com", "SECRET", f)
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "POST", "/v2/1010/registrar/domains/example.com/transfers")
	want := map[string]interface{}{"registrant_id": float64(2715), "auth_code": "SECRET", "whois_privacy": true}
	if !reflect.DeepEqual(got.Body, want) {
		t.Errorf("body = %v, want %v", got.Body, want)
	}
	if tr.State != "transferring" {
		t.Errorf("unexpected %+v", tr)
	}
}

func TestRegGetAndCancelTransfer(t *testing.T) {
	body := `{"data":{"id":361,"domain_id":182245,"registrant_id":2715,"state":"cancelled","status_description":"Canceled by customer"}}`
	c, got := regFake(t, 200, body)
	tr, err := regGetTransfer(ctxBG, c, "1010", "example.com", 361)
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/domains/example.com/transfers/361")
	var buf bytes.Buffer
	printTransfer(&buf, "example.com", tr)
	if !strings.Contains(buf.String(), "Canceled by customer") {
		t.Errorf("output missing status:\n%s", buf.String())
	}

	c, got = regFake(t, 202, body)
	if _, err := regCancelTransfer(ctxBG, c, "1010", "example.com", 361); err != nil {
		t.Fatal(err)
	}
	got.expect(t, "DELETE", "/v2/1010/registrar/domains/example.com/transfers/361")
}

func TestRegTransferOut(t *testing.T) {
	c, got := regFake(t, 204, "")
	if err := regTransferOut(ctxBG, c, "1010", "example.com"); err != nil {
		t.Fatal(err)
	}
	got.expect(t, "POST", "/v2/1010/registrar/domains/example.com/authorize_transfer_out")
}

func TestRegRenewAndRenewal(t *testing.T) {
	c, got := regFake(t, 201, `{"data":{"id":1,"domain_id":999,"period":3,"state":"new"}}`)
	r, err := regRenew(ctxBG, c, "1010", "example.com", 3, "")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "POST", "/v2/1010/registrar/domains/example.com/renewals")
	if !reflect.DeepEqual(got.Body, map[string]interface{}{"period": float64(3)}) {
		t.Errorf("body = %v", got.Body)
	}
	if r.Period != 3 {
		t.Errorf("unexpected %+v", r)
	}

	c, got = regFake(t, 200, `{"data":{"id":1,"domain_id":999,"period":1,"state":"renewed"}}`)
	r, err = regGetRenewal(ctxBG, c, "1010", "example.com", "1")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/domains/example.com/renewals/1")
	var buf bytes.Buffer
	printRenewal(&buf, "example.com", r)
	if !strings.Contains(buf.String(), "renewed") {
		t.Errorf("output:\n%s", buf.String())
	}
}

func TestRegRestoreAndStatus(t *testing.T) {
	c, got := regFake(t, 201, `{"data":{"id":43,"domain_id":214,"state":"new"}}`)
	r, err := regRestore(ctxBG, c, "1010", "example.com", "50.00")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "POST", "/v2/1010/registrar/domains/example.com/restores")
	if got.Body["premium_price"] != "50.00" {
		t.Errorf("body = %v", got.Body)
	}
	if r.ID != 43 {
		t.Errorf("unexpected %+v", r)
	}

	c, got = regFake(t, 200, `{"data":{"id":43,"domain_id":214,"state":"restored"}}`)
	rs, err := regGetRestore(ctxBG, c, "1010", "example.com", "43")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/domains/example.com/restores/43")
	var buf bytes.Buffer
	printRestore(&buf, "example.com", rs)
	if !strings.Contains(buf.String(), "restored") {
		t.Errorf("output:\n%s", buf.String())
	}
}

func TestRegDelegation(t *testing.T) {
	c, got := regFake(t, 200, `{"data":["ns1.dnsimple.com","ns2.dnsimple.com"]}`)
	ns, err := regGetDelegation(ctxBG, c, "1010", "example.com")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/domains/example.com/delegation")
	if !reflect.DeepEqual(ns, []string{"ns1.dnsimple.com", "ns2.dnsimple.com"}) {
		t.Errorf("got %v", ns)
	}
	var buf bytes.Buffer
	printDelegation(&buf, &RegDelegation{Domain: "example.com", NameServers: ns})
	if !strings.Contains(buf.String(), "ns2.dnsimple.com") {
		t.Errorf("output:\n%s", buf.String())
	}

	c, got = regFake(t, 200, `{"data":["a.example.net","b.example.net"]}`)
	ns, err = regSetDelegation(ctxBG, c, "1010", "example.com", []string{"a.example.net", "b.example.net"})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "PUT", "/v2/1010/registrar/domains/example.com/delegation")
	if strings.TrimSpace(got.Raw) != `["a.example.net","b.example.net"]` || len(ns) != 2 {
		t.Errorf("body = %s, result %v", got.Raw, ns)
	}
}

func TestRegDelegationVanity(t *testing.T) {
	c, got := regFake(t, 200, `{"data":[{"id":1,"name":"ns1.example.com","ipv4":"127.0.0.1","ipv6":"::1"}]}`)
	v, err := regDelegationToVanity(ctxBG, c, "1010", "example.com", []string{"ns1.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "PUT", "/v2/1010/registrar/domains/example.com/delegation/vanity")
	if strings.TrimSpace(got.Raw) != `["ns1.example.com"]` {
		t.Errorf("body = %s", got.Raw)
	}
	var buf bytes.Buffer
	printDelegation(&buf, &RegDelegation{Domain: "example.com", Vanity: v})
	if !strings.Contains(buf.String(), "127.0.0.1 ::1") {
		t.Errorf("output:\n%s", buf.String())
	}

	c, got = regFake(t, 204, "")
	if err := regDelegationFromVanity(ctxBG, c, "1010", "example.com"); err != nil {
		t.Fatal(err)
	}
	got.expect(t, "DELETE", "/v2/1010/registrar/domains/example.com/delegation/vanity")
}

func TestRegTransferLock(t *testing.T) {
	c, got := regFake(t, 200, `{"data":{"enabled":true}}`)
	on, err := regGetTransferLock(ctxBG, c, "1010", "example.com")
	if err != nil || !on {
		t.Fatalf("got %v, %v", on, err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/domains/example.com/transfer_lock")

	for _, tc := range []struct {
		enable bool
		method string
		body   string
	}{
		{true, "POST", `{"data":{"enabled":true}}`},
		{false, "DELETE", `{"data":{"enabled":false}}`},
	} {
		c, got := regFake(t, 201, tc.body)
		on, err := regSetTransferLock(ctxBG, c, "1010", "example.com", tc.enable)
		if err != nil || on != tc.enable {
			t.Fatalf("enable=%v: got %v, %v", tc.enable, on, err)
		}
		got.expect(t, tc.method, "/v2/1010/registrar/domains/example.com/transfer_lock")
	}
}

func TestRegWhoisPrivacy(t *testing.T) {
	for _, tc := range []struct {
		enable bool
		method string
	}{
		{true, "PUT"},
		{false, "DELETE"},
	} {
		c, got := regFake(t, 200, `{"data":{"id":1,"domain_id":2,"enabled":`+onOffJSON(tc.enable)+`,"expires_on":"2027-02-13"}}`)
		wp, err := regSetWhoisPrivacy(ctxBG, c, "1010", "example.com", tc.enable)
		if err != nil {
			t.Fatal(err)
		}
		got.expect(t, tc.method, "/v2/1010/registrar/domains/example.com/whois_privacy")
		if wp.Enabled != tc.enable || wp.ExpiresOn != "2027-02-13" {
			t.Errorf("unexpected %+v", wp)
		}
	}

	c, got := regFake(t, 200, `{"data":{"id":1,"name":"example.com","private_whois":true}}`)
	on, err := regGetWhoisPrivacy(ctxBG, c, "1010", "example.com")
	if err != nil || !on {
		t.Fatalf("got %v, %v", on, err)
	}
	got.expect(t, "GET", "/v2/1010/domains/example.com")
}

func onOffJSON(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

func TestRegRegistrantChanges(t *testing.T) {
	rcJSON := `{"id":101,"account_id":1010,"domain_id":101,"contact_id":101,"state":"new","extended_attributes":{"a":"b"},"registry_owner_change":true,"irt_lock_lifted_by":null,"created_at":"2017-02-03T17:43:22Z"}`

	c, got := regFake(t, 200, `{"data":[`+rcJSON+`],"pagination":{"current_page":1,"per_page":30,"total_entries":1,"total_pages":1}}`)
	resp, err := regListRegistrantChanges(ctxBG, c, "1010", &dnsimple.RegistrantChangeListOptions{State: dnsimple.String("new"), DomainId: dnsimple.String("example.com")})
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/registrant_changes")
	if !strings.Contains(got.Query, "state=new") || !strings.Contains(got.Query, "domain_id=example.com") {
		t.Errorf("query = %q", got.Query)
	}
	var buf bytes.Buffer
	printRegistrantChanges(&buf, resp.Data)
	if !strings.Contains(buf.String(), "1 registrant changes") || !strings.Contains(buf.String(), "contact 101") {
		t.Errorf("output:\n%s", buf.String())
	}

	c, got = regFake(t, 200, `{"data":`+rcJSON+`}`)
	rc, err := regGetRegistrantChange(ctxBG, c, "1010", 101)
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "GET", "/v2/1010/registrar/registrant_changes/101")
	buf.Reset()
	printRegistrantChange(&buf, rc)
	for _, s := range []string{"Attr a:", "Owner change:", "yes"} {
		if !strings.Contains(buf.String(), s) {
			t.Errorf("output missing %q:\n%s", s, buf.String())
		}
	}

	c, got = regFake(t, 200, `{"data":{"domain_id":101,"contact_id":101,"extended_attributes":[{"name":"x-fr","description":"Birth","required":true,"options":[{"title":"A","value":"a"}]}],"registry_owner_change":true}}`)
	chk, err := regCheckRegistrantChange(ctxBG, c, "1010", "example.fr", "101")
	if err != nil {
		t.Fatal(err)
	}
	got.expect(t, "POST", "/v2/1010/registrar/registrant_changes/check")
	if !reflect.DeepEqual(got.Body, map[string]interface{}{"domain_id": "example.fr", "contact_id": "101"}) {
		t.Errorf("body = %v", got.Body)
	}
	buf.Reset()
	printRegistrantChangeCheck(&buf, "example.fr", chk)
	for _, s := range []string{"x-fr", "(required)", "options: a"} {
		if !strings.Contains(buf.String(), s) {
			t.Errorf("output missing %q:\n%s", s, buf.String())
		}
	}

	c, got = regFake(t, 202, `{"data":`+rcJSON+`}`)
	if _, err := regCreateRegistrantChange(ctxBG, c, "1010", "example.com", "101", nil); err != nil {
		t.Fatal(err)
	}
	got.expect(t, "POST", "/v2/1010/registrar/registrant_changes")
	want := map[string]interface{}{"domain_id": "example.com", "contact_id": "101", "extended_attributes": map[string]interface{}{}}
	if !reflect.DeepEqual(got.Body, want) {
		t.Errorf("body = %v, want %v", got.Body, want)
	}

	c, got = regFake(t, 204, "")
	if err := regDeleteRegistrantChange(ctxBG, c, "1010", 101); err != nil {
		t.Fatal(err)
	}
	got.expect(t, "DELETE", "/v2/1010/registrar/registrant_changes/101")
}

func TestRegAcquireFlags(t *testing.T) {
	newCmd := func() *cobra.Command {
		c := &cobra.Command{Use: "x"}
		addRegAcquireFlags(c)
		return c
	}

	c := newCmd()
	if _, err := readRegAcquireFlags(c); err == nil {
		t.Error("expected error without --registrant-id")
	}

	c = newCmd()
	_ = c.ParseFlags([]string{"--registrant-id", "7", "--auto-renew", "--extended-attribute", "k=v"})
	f, err := readRegAcquireFlags(c)
	if err != nil {
		t.Fatal(err)
	}
	if f.RegistrantID != 7 || !f.AutoRenew || f.WhoisPrivacy || f.Trustee != nil || f.ExtendedAttributes["k"] != "v" {
		t.Errorf("unexpected %+v", f)
	}

	c = newCmd()
	_ = c.ParseFlags([]string{"--registrant-id", "7", "--trustee=false"})
	f, _ = readRegAcquireFlags(c)
	if f.Trustee == nil || *f.Trustee {
		t.Errorf("explicit --trustee=false should be sent, got %v", f.Trustee)
	}
}

func TestRegGateRequiresYes(t *testing.T) {
	c := &cobra.Command{Use: "x"}
	regAddYesFlag(c)
	if err := regGate(c, regPlan{Action: "do it"}); !errors.Is(err, errRegNotConfirmed) {
		t.Errorf("without --yes: got %v", err)
	}
	_ = c.ParseFlags([]string{"--yes"})
	if err := regGate(c, regPlan{Action: "do it"}); err != nil {
		t.Errorf("with --yes: got %v", err)
	}

	var buf bytes.Buffer
	printRegPlan(&buf, regPlan{Action: "register x.com", Details: []regField{{"Price", "$1.00"}}, Warnings: []string{"premium"}})
	for _, s := range []string{"Dry run: register x.com", "Price:", "$1.00", "premium", "--yes"} {
		if !strings.Contains(buf.String(), s) {
			t.Errorf("plan missing %q:\n%s", s, buf.String())
		}
	}
}

// Gated commands that need no lookup must refuse before touching the API.
func TestGatedCommandsRefuseWithoutYes(t *testing.T) {
	for _, args := range [][]string{
		{"registrar", "transfer-out", "example.com"},
		{"registrar", "registrant-changes", "create", "example.com", "--contact-id", "5"},
	} {
		rootCmd.SetArgs(args)
		rootCmd.SetOut(io.Discard)
		rootCmd.SetErr(io.Discard)
		err := rootCmd.Execute()
		if !errors.Is(err, errRegNotConfirmed) {
			t.Errorf("%v: got %v, want not-confirmed", args, err)
		}
	}
	rootCmd.SetArgs(nil)
}

func TestNameServersFlagValidation(t *testing.T) {
	for _, args := range [][]string{
		{"registrar", "nameservers", "example.com", "--to-vanity", "--from-vanity"},
		{"registrar", "nameservers", "example.com", "--to-vanity"},
		{"registrar", "nameservers", "example.com", "ns1.x.com", "--from-vanity"},
	} {
		rootCmd.SetArgs(args)
		rootCmd.SetOut(io.Discard)
		rootCmd.SetErr(io.Discard)
		if err := rootCmd.Execute(); err == nil {
			t.Errorf("%v: expected error", args)
		}
		// reset sticky flag state on the shared command
		_ = registrarNameServersCmd.Flags().Set("to-vanity", "false")
		_ = registrarNameServersCmd.Flags().Set("from-vanity", "false")
	}
	rootCmd.SetArgs(nil)
}
