package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dnsimple/dnsimple-go/v9/dnsimple"
	"github.com/dorkitude/simple/internal/ui"
	"github.com/spf13/cobra"
)

var certificatesCmd = &cobra.Command{
	Use:     "certificates",
	Aliases: []string{"certs", "cert", "certificate"},
	Short:   "Manage SSL/TLS certificates",
	Long: `List, inspect, download, and order SSL/TLS certificates for a domain.

Examples:
  simple certs list example.com
  simple certs get example.com 101
  simple certs download example.com 101 --out ./tls
  simple certs private-key example.com 101 --out example.key
  simple certs letsencrypt purchase example.com --name www --yes`,
}

// ---------- logic (testable) ----------

func listCertificates(ctx context.Context, c *dnsimple.Client, accountID, domain string, opts *dnsimple.ListOptions) (*dnsimple.CertificatesResponse, error) {
	resp, err := c.Certificates.ListCertificates(ctx, accountID, domain, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list certificates: %w", err)
	}
	return resp, nil
}

func getCertificate(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64) (*dnsimple.Certificate, error) {
	resp, err := c.Certificates.GetCertificate(ctx, accountID, domain, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get certificate: %w", err)
	}
	return resp.Data, nil
}

// CertFiles lists the PEM files written by a certificate download.
type CertFiles struct {
	Server string `json:"server,omitempty"`
	Root   string `json:"root,omitempty"`
	Chain  string `json:"chain,omitempty"`
}

// downloadCertificate fetches the certificate bundle and writes server/root/chain PEMs into dir.
func downloadCertificate(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64, dir string) (*CertFiles, error) {
	resp, err := c.Certificates.DownloadCertificate(ctx, accountID, domain, id)
	if err != nil {
		return nil, fmt.Errorf("failed to download certificate: %w", err)
	}
	b := resp.Data
	if b == nil || b.ServerCertificate == "" {
		return nil, fmt.Errorf("certificate %d has no issued certificate to download (is it still pending?)", id)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	files := &CertFiles{}
	write := func(name, content string) (string, error) {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(ensureTrailingNewline(content)), 0o644); err != nil {
			return "", fmt.Errorf("failed to write %s: %w", path, err)
		}
		return path, nil
	}
	if files.Server, err = write("server.pem", b.ServerCertificate); err != nil {
		return nil, err
	}
	if b.RootCertificate != "" {
		if files.Root, err = write("root.pem", b.RootCertificate); err != nil {
			return nil, err
		}
	}
	if len(b.IntermediateCertificates) > 0 {
		var chain strings.Builder
		for _, pem := range b.IntermediateCertificates {
			chain.WriteString(ensureTrailingNewline(pem))
		}
		if files.Chain, err = write("chain.pem", chain.String()); err != nil {
			return nil, err
		}
	}
	return files, nil
}

func ensureTrailingNewline(s string) string {
	if s == "" || strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

// fetchCertificatePrivateKey returns the certificate's private key PEM.
func fetchCertificatePrivateKey(ctx context.Context, c *dnsimple.Client, accountID, domain string, id int64) (string, error) {
	resp, err := c.Certificates.GetCertificatePrivateKey(ctx, accountID, domain, id)
	if err != nil {
		return "", fmt.Errorf("failed to get private key: %w", err)
	}
	if resp.Data == nil || resp.Data.PrivateKey == "" {
		return "", fmt.Errorf("certificate %d has no private key stored at DNSimple", id)
	}
	return resp.Data.PrivateKey, nil
}

// writePrivateKeyFile writes key material to path with 0600 permissions,
// tightening permissions on an existing file too.
func writePrivateKeyFile(path, key string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.WriteString(ensureTrailingNewline(key)); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

func purchaseLetsencrypt(ctx context.Context, c *dnsimple.Client, accountID, domain string, attrs dnsimple.LetsencryptCertificateAttributes) (*dnsimple.CertificatePurchase, error) {
	resp, err := c.Certificates.PurchaseLetsencryptCertificate(ctx, accountID, domain, attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to purchase certificate: %w", err)
	}
	return resp.Data, nil
}

func issueLetsencrypt(ctx context.Context, c *dnsimple.Client, accountID, domain string, purchaseID int64) (*dnsimple.Certificate, error) {
	resp, err := c.Certificates.IssueLetsencryptCertificate(ctx, accountID, domain, purchaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to issue certificate: %w", err)
	}
	return resp.Data, nil
}

func renewLetsencrypt(ctx context.Context, c *dnsimple.Client, accountID, domain string, certID int64, attrs dnsimple.LetsencryptCertificateAttributes) (*dnsimple.CertificateRenewal, error) {
	resp, err := c.Certificates.PurchaseLetsencryptCertificateRenewal(ctx, accountID, domain, certID, attrs)
	if err != nil {
		return nil, fmt.Errorf("failed to purchase renewal: %w", err)
	}
	return resp.Data, nil
}

func issueLetsencryptRenewal(ctx context.Context, c *dnsimple.Client, accountID, domain string, certID, renewalID int64) (*dnsimple.Certificate, error) {
	resp, err := c.Certificates.IssueLetsencryptCertificateRenewal(ctx, accountID, domain, certID, renewalID)
	if err != nil {
		return nil, fmt.Errorf("failed to issue renewal: %w", err)
	}
	return resp.Data, nil
}

// ---------- output ----------

func certStateStyle(state string) string {
	switch state {
	case "issued":
		return ui.SuccessStyle.Render(state)
	case "cancelled", "rejected", "refunded", "expired":
		return ui.ErrorStyle.Render(state)
	}
	return ui.WarningStyle.Render(state)
}

func printCertificate(cert *dnsimple.Certificate) {
	fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🔒 %s", cert.CommonName)))
	fmt.Println()
	fmt.Printf("  %-14s %d\n", "ID:", cert.ID)
	fmt.Printf("  %-14s %s\n", "Common Name:", cert.CommonName)
	if len(cert.AlternateNames) > 0 {
		fmt.Printf("  %-14s %s\n", "Alt Names:", strings.Join(cert.AlternateNames, ", "))
	}
	fmt.Printf("  %-14s %s\n", "State:", certStateStyle(cert.State))
	if cert.AuthorityIdentifier != "" {
		fmt.Printf("  %-14s %s\n", "Authority:", cert.AuthorityIdentifier)
	}
	if cert.Years > 0 {
		fmt.Printf("  %-14s %d\n", "Years:", cert.Years)
	}
	fmt.Printf("  %-14s %v\n", "Auto-Renew:", cert.AutoRenew)
	if cert.ExpiresAt != "" {
		fmt.Printf("  %-14s %s\n", "Expires:", cert.ExpiresAt)
	}
	fmt.Printf("  %-14s %s\n", "Created:", cert.CreatedAt)
	fmt.Printf("  %-14s %s\n", "Updated:", cert.UpdatedAt)
}

// ---------- commands ----------

var certificatesListCmd = &cobra.Command{
	Use:   "list [domain]",
	Short: "List certificates for a domain",
	Long: `List the SSL/TLS certificates for a domain.

Examples:
  simple certs list example.com
  simple certs list example.com --sort expiration:asc
  simple certs list example.com --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		resp, err := listCertificates(ctx, app.Client, app.AccountID, args[0], listOptionsFromFlags(cmd))
		if err != nil {
			return err
		}
		if printJSON(resp.Data) {
			return nil
		}
		if len(resp.Data) == 0 {
			fmt.Println(ui.Warn("No certificates found for " + args[0]))
			return nil
		}
		fmt.Println(ui.TitleStyle.Render(fmt.Sprintf("🔒 %d certificates for %s", len(resp.Data), args[0])))
		fmt.Println()
		for _, cert := range resp.Data {
			fmt.Printf("  %-10d %-30s %s", cert.ID, ui.AccentStyle.Render(cert.CommonName), certStateStyle(cert.State))
			if cert.ExpiresAt != "" {
				fmt.Printf("  %s", ui.SubtleStyle.Render("expires: "+shortDate(cert.ExpiresAt)))
			}
			if cert.AutoRenew {
				fmt.Printf("  %s", ui.SubtleStyle.Render("↻"))
			}
			fmt.Println()
		}
		printPagination(resp.Pagination)
		return nil
	},
}

var certificatesGetCmd = &cobra.Command{
	Use:   "get [domain] [certificate-id]",
	Short: "Get certificate details",
	Long: `Display details for a certificate.

Examples:
  simple certs get example.com 101
  simple certs get example.com 101 --json`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("certificate ID", args[1])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		cert, err := getCertificate(ctx, app.Client, app.AccountID, args[0], id)
		if err != nil {
			return err
		}
		if printJSON(cert) {
			return nil
		}
		printCertificate(cert)
		return nil
	},
}

var certificatesDownloadCmd = &cobra.Command{
	Use:   "download [domain] [certificate-id]",
	Short: "Download certificate PEM files",
	Long: `Download an issued certificate and write server.pem, root.pem, and
chain.pem (intermediates) into a directory. The private key is NOT included;
use 'simple certs private-key' for that.

Examples:
  simple certs download example.com 101
  simple certs download example.com 101 --out ./tls`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("certificate ID", args[1])
		if err != nil {
			return err
		}
		out, _ := cmd.Flags().GetString("out")
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		files, err := downloadCertificate(ctx, app.Client, app.AccountID, args[0], id, out)
		if err != nil {
			return err
		}
		if printJSON(files) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Certificate %d downloaded", id)))
		fmt.Printf("  %-14s %s\n", "Server:", files.Server)
		if files.Root != "" {
			fmt.Printf("  %-14s %s\n", "Root:", files.Root)
		}
		if files.Chain != "" {
			fmt.Printf("  %-14s %s\n", "Chain:", files.Chain)
		}
		return nil
	},
}

var certificatesPrivateKeyCmd = &cobra.Command{
	Use:     "private-key [domain] [certificate-id]",
	Aliases: []string{"key"},
	Short:   "Save a certificate's private key",
	Long: `Fetch a certificate's private key and save it to a file with 0600
permissions. The key is never printed unless --stdout is given.

Examples:
  simple certs private-key example.com 101
  simple certs private-key example.com 101 --out /etc/ssl/private/example.key
  simple certs private-key example.com 101 --stdout > example.key`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		id, err := parseIDArg("certificate ID", args[1])
		if err != nil {
			return err
		}
		out, _ := cmd.Flags().GetString("out")
		toStdout, _ := cmd.Flags().GetBool("stdout")
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		key, err := fetchCertificatePrivateKey(ctx, app.Client, app.AccountID, domain, id)
		if err != nil {
			return err
		}
		if toStdout {
			fmt.Print(ensureTrailingNewline(key))
			return nil
		}
		if out == "" {
			out = fmt.Sprintf("%s-%d.key", domain, id)
		}
		if err := writePrivateKeyFile(out, key); err != nil {
			return fmt.Errorf("failed to write private key: %w", err)
		}
		if printJSON(map[string]string{"path": out}) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Private key written to %s (mode 0600)", out)))
		return nil
	},
}

var certificatesLetsencryptCmd = &cobra.Command{
	Use:     "letsencrypt",
	Aliases: []string{"le"},
	Short:   "Order and renew Let's Encrypt certificates",
	Long: `Purchase, issue, and renew Let's Encrypt certificates.

Ordering is two steps: 'purchase' creates the order, then 'issue' requests
issuance. Renewals work the same way with 'renew' then 'issue-renewal'.

Examples:
  simple certs le purchase example.com --name www --yes
  simple certs le issue example.com 200
  simple certs le renew example.com 101 --yes
  simple certs le issue-renewal example.com 101 300`,
}

func letsencryptAttrsFromFlags(cmd *cobra.Command) dnsimple.LetsencryptCertificateAttributes {
	var attrs dnsimple.LetsencryptCertificateAttributes
	if cmd.Flags().Lookup("name") != nil {
		attrs.Name, _ = cmd.Flags().GetString("name")
	}
	if cmd.Flags().Lookup("alternate-names") != nil {
		attrs.AlternateNames, _ = cmd.Flags().GetStringSlice("alternate-names")
	}
	attrs.AutoRenew, _ = cmd.Flags().GetBool("auto-renew")
	attrs.SignatureAlgorithm, _ = cmd.Flags().GetString("signature-algorithm")
	return attrs
}

var certificatesLEPurchaseCmd = &cobra.Command{
	Use:   "purchase [domain]",
	Short: "Purchase a Let's Encrypt certificate (requires --yes)",
	Long: `Order a Let's Encrypt certificate for a domain. Requires --yes.
After purchasing, run 'simple certs le issue <domain> <purchase-id>'.

Examples:
  simple certs le purchase example.com --yes
  simple certs le purchase example.com --name www --auto-renew --yes
  simple certs le purchase example.com --name www --alternate-names api.example.com,docs.example.com --yes
  simple certs le purchase example.com --signature-algorithm RSA --yes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		attrs := letsencryptAttrsFromFlags(cmd)
		host := domain
		if attrs.Name != "" {
			host = attrs.Name + "." + domain
		}
		preview := []string{fmt.Sprintf("%-14s %s", "Domain:", domain), fmt.Sprintf("%-14s %s", "Common Name:", host)}
		if len(attrs.AlternateNames) > 0 {
			preview = append(preview, fmt.Sprintf("%-14s %s", "Alt Names:", strings.Join(attrs.AlternateNames, ", ")))
		}
		preview = append(preview, fmt.Sprintf("%-14s %v", "Auto-Renew:", attrs.AutoRenew))
		if attrs.SignatureAlgorithm != "" {
			preview = append(preview, fmt.Sprintf("%-14s %s", "Signature:", attrs.SignatureAlgorithm))
		}
		if err := confirmWithYes(cmd, "purchase a Let's Encrypt certificate", preview); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		p, err := purchaseLetsencrypt(ctx, app.Client, app.AccountID, domain, attrs)
		if err != nil {
			return err
		}
		if printJSON(p) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Certificate purchased for %s (purchase ID: %d, certificate ID: %d, state: %s)", host, p.ID, p.CertificateID, p.State)))
		fmt.Println(ui.Info(fmt.Sprintf("Next: simple certs le issue %s %d", domain, p.ID)))
		return nil
	},
}

var certificatesLEIssueCmd = &cobra.Command{
	Use:   "issue [domain] [purchase-id]",
	Short: "Issue a purchased Let's Encrypt certificate",
	Long: `Request issuance of a purchased Let's Encrypt certificate.

Examples:
  simple certs le issue example.com 200`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseIDArg("purchase ID", args[1])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		cert, err := issueLetsencrypt(ctx, app.Client, app.AccountID, args[0], id)
		if err != nil {
			return err
		}
		if printJSON(cert) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Issuance requested for %s (certificate ID: %d, state: %s)", cert.CommonName, cert.ID, cert.State)))
		return nil
	},
}

var certificatesLERenewCmd = &cobra.Command{
	Use:   "renew [domain] [certificate-id]",
	Short: "Purchase a Let's Encrypt renewal (requires --yes)",
	Long: `Purchase a renewal for an existing Let's Encrypt certificate. Requires --yes.
After purchasing, run 'simple certs le issue-renewal <domain> <cert-id> <renewal-id>'.

Examples:
  simple certs le renew example.com 101 --yes
  simple certs le renew example.com 101 --auto-renew --yes`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		id, err := parseIDArg("certificate ID", args[1])
		if err != nil {
			return err
		}
		attrs := letsencryptAttrsFromFlags(cmd)
		preview := []string{
			fmt.Sprintf("%-14s %s", "Domain:", domain),
			fmt.Sprintf("%-14s %d", "Certificate:", id),
			fmt.Sprintf("%-14s %v", "Auto-Renew:", attrs.AutoRenew),
		}
		if attrs.SignatureAlgorithm != "" {
			preview = append(preview, fmt.Sprintf("%-14s %s", "Signature:", attrs.SignatureAlgorithm))
		}
		if err := confirmWithYes(cmd, "purchase a Let's Encrypt renewal", preview); err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		r, err := renewLetsencrypt(ctx, app.Client, app.AccountID, domain, id, attrs)
		if err != nil {
			return err
		}
		if printJSON(r) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Renewal purchased (renewal ID: %d, state: %s)", r.ID, r.State)))
		fmt.Println(ui.Info(fmt.Sprintf("Next: simple certs le issue-renewal %s %d %d", domain, id, r.ID)))
		return nil
	},
}

var certificatesLEIssueRenewalCmd = &cobra.Command{
	Use:   "issue-renewal [domain] [certificate-id] [renewal-id]",
	Short: "Issue a purchased Let's Encrypt renewal",
	Long: `Request issuance of a purchased Let's Encrypt certificate renewal.

Examples:
  simple certs le issue-renewal example.com 101 300`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		certID, err := parseIDArg("certificate ID", args[1])
		if err != nil {
			return err
		}
		renewalID, err := parseIDArg("renewal ID", args[2])
		if err != nil {
			return err
		}
		ctx := context.Background()
		app, err := getApp(ctx)
		if err != nil {
			return err
		}
		cert, err := issueLetsencryptRenewal(ctx, app.Client, app.AccountID, args[0], certID, renewalID)
		if err != nil {
			return err
		}
		if printJSON(cert) {
			return nil
		}
		fmt.Println(ui.Success(fmt.Sprintf("Renewal issuance requested for %s (new certificate ID: %d, state: %s)", cert.CommonName, cert.ID, cert.State)))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(certificatesCmd)

	certificatesCmd.AddCommand(certificatesListCmd)
	addPageFlags(certificatesListCmd)
	certificatesListCmd.Flags().String("sort", "", "Sort order (e.g. id:asc, common_name:desc, expiration:asc)")

	certificatesCmd.AddCommand(certificatesGetCmd)

	certificatesCmd.AddCommand(certificatesDownloadCmd)
	certificatesDownloadCmd.Flags().StringP("out", "o", ".", "Directory to write PEM files into")

	certificatesCmd.AddCommand(certificatesPrivateKeyCmd)
	certificatesPrivateKeyCmd.Flags().StringP("out", "o", "", "File to write the key to (default <domain>-<id>.key)")
	certificatesPrivateKeyCmd.Flags().Bool("stdout", false, "Print the private key to stdout instead of writing a file")

	certificatesCmd.AddCommand(certificatesLetsencryptCmd)

	certificatesLetsencryptCmd.AddCommand(certificatesLEPurchaseCmd)
	certificatesLEPurchaseCmd.Flags().String("name", "", "Certificate name (subdomain, e.g. www); empty for the apex")
	certificatesLEPurchaseCmd.Flags().StringSlice("alternate-names", nil, "Comma-separated alternate names (SANs)")
	certificatesLEPurchaseCmd.Flags().Bool("auto-renew", false, "Automatically renew the certificate")
	certificatesLEPurchaseCmd.Flags().String("signature-algorithm", "", "Signature algorithm (ECDSA or RSA)")
	certificatesLEPurchaseCmd.Flags().Bool("yes", false, "Confirm the purchase")

	certificatesLetsencryptCmd.AddCommand(certificatesLEIssueCmd)

	certificatesLetsencryptCmd.AddCommand(certificatesLERenewCmd)
	certificatesLERenewCmd.Flags().Bool("auto-renew", false, "Automatically renew the renewed certificate")
	certificatesLERenewCmd.Flags().String("signature-algorithm", "", "Signature algorithm (ECDSA or RSA)")
	certificatesLERenewCmd.Flags().Bool("yes", false, "Confirm the renewal purchase")

	certificatesLetsencryptCmd.AddCommand(certificatesLEIssueRenewalCmd)
}
