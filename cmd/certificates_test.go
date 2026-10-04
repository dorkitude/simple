package cmd

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/spf13/cobra"
)

const certJSON = `{"data":{"id":101,"domain_id":2,"common_name":"www.example.com","alternate_names":["api.example.com"],"state":"issued","auto_renew":true,"expires_at":"2027-01-01T00:00:00Z"}}`

func TestListCertificates(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":[{"id":101,"common_name":"www.example.com","state":"issued"}],"pagination":{"current_page":1,"per_page":30,"total_entries":1,"total_pages":1}}`)
	resp, err := listCertificates(context.Background(), c, "1010", "example.com", pageListOptions(2, 0, "expiration:asc"))
	if err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodGet, "/v2/1010/domains/example.com/certificates")
	if !strings.Contains(got.Query, "page=2") || !strings.Contains(got.Query, "sort=expiration%3Aasc") {
		t.Errorf("unexpected query %q", got.Query)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != 101 {
		t.Errorf("unexpected data %+v", resp.Data)
	}
}

func TestGetCertificate(t *testing.T) {
	c, calls := recordingClient(t, 200, certJSON)
	cert, err := getCertificate(context.Background(), c, "1010", "example.com", 101)
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/1010/domains/example.com/certificates/101")
	if cert.CommonName != "www.example.com" || !cert.AutoRenew {
		t.Errorf("unexpected cert %+v", cert)
	}
}

func TestDownloadCertificate(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":{"server":"SERVER","root":"ROOT","chain":["INT1\n","INT2"]}}`)
	dir := filepath.Join(t.TempDir(), "tls")
	files, err := downloadCertificate(context.Background(), c, "1010", "example.com", 101, dir)
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/1010/domains/example.com/certificates/101/download")
	for path, want := range map[string]string{files.Server: "SERVER\n", files.Root: "ROOT\n", files.Chain: "INT1\nINT2\n"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != want {
			t.Errorf("%s = %q, want %q", path, data, want)
		}
	}
}

func TestDownloadCertificatePending(t *testing.T) {
	c, _ := recordingClient(t, 200, `{"data":{"server":null,"root":null,"chain":[]}}`)
	_, err := downloadCertificate(context.Background(), c, "1010", "example.com", 101, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "pending") {
		t.Fatalf("expected pending error, got %v", err)
	}
}

func TestPrivateKeyWrittenWith0600(t *testing.T) {
	c, calls := recordingClient(t, 200, `{"data":{"private_key":"-----BEGIN RSA PRIVATE KEY-----\nFAKE\n-----END RSA PRIVATE KEY-----"}}`)
	key, err := fetchCertificatePrivateKey(context.Background(), c, "1010", "example.com", 101)
	if err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodGet, "/v2/1010/domains/example.com/certificates/101/private_key")

	path := filepath.Join(t.TempDir(), "k.key")
	// Pre-create with loose permissions to ensure they get tightened.
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writePrivateKeyFile(path, key); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("key mode = %o, want 600", st.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "-----BEGIN") || !strings.HasSuffix(string(data), "\n") {
		t.Errorf("unexpected key file contents")
	}
}

func TestPurchaseLetsencrypt(t *testing.T) {
	c, calls := recordingClient(t, 201, `{"data":{"id":200,"certificate_id":101,"state":"new","auto_renew":true}}`)
	p, err := purchaseLetsencrypt(context.Background(), c, "1010", "example.com", dnsimple.LetsencryptCertificateAttributes{
		Name: "www", AutoRenew: true, AlternateNames: []string{"api.example.com"}, SignatureAlgorithm: "RSA",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodPost, "/v2/1010/domains/example.com/certificates/letsencrypt")
	if got.Body["name"] != "www" || got.Body["auto_renew"] != true || got.Body["signature_algorithm"] != "RSA" {
		t.Errorf("unexpected body %+v", got.Body)
	}
	if alts, _ := got.Body["alternate_names"].([]interface{}); len(alts) != 1 || alts[0] != "api.example.com" {
		t.Errorf("unexpected alternate_names %+v", got.Body["alternate_names"])
	}
	if p.ID != 200 || p.CertificateID != 101 {
		t.Errorf("unexpected purchase %+v", p)
	}
}

func TestIssueLetsencrypt(t *testing.T) {
	c, calls := recordingClient(t, 202, certJSON)
	if _, err := issueLetsencrypt(context.Background(), c, "1010", "example.com", 200); err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodPost, "/v2/1010/domains/example.com/certificates/letsencrypt/200/issue")
}

func TestRenewLetsencrypt(t *testing.T) {
	c, calls := recordingClient(t, 201, `{"data":{"id":300,"old_certificate_id":101,"state":"new"}}`)
	r, err := renewLetsencrypt(context.Background(), c, "1010", "example.com", 101, dnsimple.LetsencryptCertificateAttributes{AutoRenew: true})
	if err != nil {
		t.Fatal(err)
	}
	got := expectOneCall(t, calls, http.MethodPost, "/v2/1010/domains/example.com/certificates/letsencrypt/101/renewals")
	if got.Body["auto_renew"] != true {
		t.Errorf("unexpected body %+v", got.Body)
	}
	if r.ID != 300 {
		t.Errorf("unexpected renewal %+v", r)
	}
}

func TestIssueLetsencryptRenewal(t *testing.T) {
	c, calls := recordingClient(t, 202, certJSON)
	if _, err := issueLetsencryptRenewal(context.Background(), c, "1010", "example.com", 101, 300); err != nil {
		t.Fatal(err)
	}
	expectOneCall(t, calls, http.MethodPost, "/v2/1010/domains/example.com/certificates/letsencrypt/101/renewals/300/issue")
}

func TestLetsencryptPurchaseRequiresYes(t *testing.T) {
	for _, c := range []*cobra.Command{certificatesLEPurchaseCmd, certificatesLERenewCmd} {
		if c.Flags().Lookup("yes") == nil {
			t.Errorf("%s has no --yes flag", c.Name())
		}
	}
	// Without --yes, RunE must fail before touching the API (getApp is never reached).
	err := certificatesLEPurchaseCmd.RunE(certificatesLEPurchaseCmd, []string{"example.com"})
	if _, ok := err.(errNeedsYes); !ok {
		t.Fatalf("expected errNeedsYes, got %v", err)
	}
	err = certificatesLERenewCmd.RunE(certificatesLERenewCmd, []string{"example.com", "101"})
	if _, ok := err.(errNeedsYes); !ok {
		t.Fatalf("expected errNeedsYes, got %v", err)
	}
}
