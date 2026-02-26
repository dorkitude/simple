package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dnsimple/dnsimple-go/dnsimple"
	"github.com/dorkitude/simple/internal/client"
)

type Backend interface {
	IsDemo() bool
	Whoami(ctx context.Context) (*dnsimple.WhoamiData, error)
	ListDomains(ctx context.Context) ([]dnsimple.Domain, error)
	GetDomain(ctx context.Context, name string) (*dnsimple.Domain, error)
	DeleteDomain(ctx context.Context, name string) error
	ListZones(ctx context.Context) ([]dnsimple.Zone, error)
	GetZone(ctx context.Context, name string) (*dnsimple.Zone, error)
	GetZoneFile(ctx context.Context, name string) (string, error)
	CheckZoneDistribution(ctx context.Context, name string) (bool, error)
	ActivateZoneDNS(ctx context.Context, name string) error
	DeactivateZoneDNS(ctx context.Context, name string) error
	ListRecords(ctx context.Context, zone string) ([]dnsimple.ZoneRecord, error)
	GetRecord(ctx context.Context, zone string, recordID int64) (*dnsimple.ZoneRecord, error)
	CheckRecordDistribution(ctx context.Context, zone string, recordID int64) (bool, error)
	DeleteRecord(ctx context.Context, zone string, recordID int64) error
}

var (
	backendMu      sync.RWMutex
	currentBackend Backend = &realBackend{}
)

func setBackend(b Backend) {
	backendMu.Lock()
	defer backendMu.Unlock()
	currentBackend = b
}

func useRealBackend() {
	setBackend(&realBackend{})
}

func useDemoBackend() {
	setBackend(newDemoBackend())
}

func getBackend() Backend {
	backendMu.RLock()
	defer backendMu.RUnlock()
	return currentBackend
}

type realBackend struct{}

func (b *realBackend) IsDemo() bool { return false }

func (b *realBackend) Whoami(ctx context.Context) (*dnsimple.WhoamiData, error) {
	app, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := app.Client.Identity.Whoami(ctx)
	if err != nil {
		return nil, fmt.Errorf("whoami failed: %w", err)
	}
	return resp.Data, nil
}

func (b *realBackend) ListDomains(ctx context.Context) ([]dnsimple.Domain, error) {
	app, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := app.Client.Domains.ListDomains(ctx, app.AccountID, &dnsimple.DomainListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list domains: %w", err)
	}
	return resp.Data, nil
}

func (b *realBackend) GetDomain(ctx context.Context, name string) (*dnsimple.Domain, error) {
	app, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := app.Client.Domains.GetDomain(ctx, app.AccountID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get domain: %w", err)
	}
	return resp.Data, nil
}

func (b *realBackend) DeleteDomain(ctx context.Context, name string) error {
	app, err := client.New(ctx)
	if err != nil {
		return err
	}
	_, err = app.Client.Domains.DeleteDomain(ctx, app.AccountID, name)
	if err != nil {
		return fmt.Errorf("failed to delete domain: %w", err)
	}
	return nil
}

func (b *realBackend) ListZones(ctx context.Context) ([]dnsimple.Zone, error) {
	app, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := app.Client.Zones.ListZones(ctx, app.AccountID, &dnsimple.ZoneListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list zones: %w", err)
	}
	return resp.Data, nil
}

func (b *realBackend) GetZone(ctx context.Context, name string) (*dnsimple.Zone, error) {
	app, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := app.Client.Zones.GetZone(ctx, app.AccountID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone: %w", err)
	}
	return resp.Data, nil
}

func (b *realBackend) GetZoneFile(ctx context.Context, name string) (string, error) {
	app, err := client.New(ctx)
	if err != nil {
		return "", err
	}
	resp, err := app.Client.Zones.GetZoneFile(ctx, app.AccountID, name)
	if err != nil {
		return "", fmt.Errorf("failed to get zone file: %w", err)
	}
	return strings.TrimSpace(resp.Data.Zone), nil
}

func (b *realBackend) CheckZoneDistribution(ctx context.Context, name string) (bool, error) {
	app, err := client.New(ctx)
	if err != nil {
		return false, err
	}
	resp, err := app.Client.Zones.CheckZoneDistribution(ctx, app.AccountID, name)
	if err != nil {
		return false, fmt.Errorf("failed to check zone distribution: %w", err)
	}
	return resp.Data.Distributed, nil
}

func (b *realBackend) ActivateZoneDNS(ctx context.Context, name string) error {
	app, err := client.New(ctx)
	if err != nil {
		return err
	}
	_, err = app.Client.Zones.ActivateZoneDns(ctx, app.AccountID, name)
	if err != nil {
		return fmt.Errorf("failed to activate zone: %w", err)
	}
	return nil
}

func (b *realBackend) DeactivateZoneDNS(ctx context.Context, name string) error {
	app, err := client.New(ctx)
	if err != nil {
		return err
	}
	_, err = app.Client.Zones.DeactivateZoneDns(ctx, app.AccountID, name)
	if err != nil {
		return fmt.Errorf("failed to deactivate zone: %w", err)
	}
	return nil
}

func (b *realBackend) ListRecords(ctx context.Context, zone string) ([]dnsimple.ZoneRecord, error) {
	app, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := app.Client.Zones.ListRecords(ctx, app.AccountID, zone, &dnsimple.ZoneRecordListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list records for %s: %w", zone, err)
	}
	return resp.Data, nil
}

func (b *realBackend) GetRecord(ctx context.Context, zone string, recordID int64) (*dnsimple.ZoneRecord, error) {
	app, err := client.New(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := app.Client.Zones.GetRecord(ctx, app.AccountID, zone, recordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get record: %w", err)
	}
	return resp.Data, nil
}

func (b *realBackend) CheckRecordDistribution(ctx context.Context, zone string, recordID int64) (bool, error) {
	app, err := client.New(ctx)
	if err != nil {
		return false, err
	}
	resp, err := app.Client.Zones.CheckZoneRecordDistribution(ctx, app.AccountID, zone, recordID)
	if err != nil {
		return false, fmt.Errorf("failed to check record distribution: %w", err)
	}
	return resp.Data.Distributed, nil
}

func (b *realBackend) DeleteRecord(ctx context.Context, zone string, recordID int64) error {
	app, err := client.New(ctx)
	if err != nil {
		return err
	}
	_, err = app.Client.Zones.DeleteRecord(ctx, app.AccountID, zone, recordID)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}
	return nil
}

type demoBackend struct {
	mu      sync.RWMutex
	whoami  *dnsimple.WhoamiData
	domains map[string]dnsimple.Domain
	zones   map[string]dnsimple.Zone
	records map[string][]dnsimple.ZoneRecord
}

func newDemoBackend() *demoBackend {
	b := &demoBackend{
		domains: map[string]dnsimple.Domain{},
		zones:   map[string]dnsimple.Zone{},
		records: map[string][]dnsimple.ZoneRecord{},
	}
	b.seed()
	return b
}

func (b *demoBackend) IsDemo() bool { return true }

func (b *demoBackend) Whoami(ctx context.Context) (*dnsimple.WhoamiData, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	cp := *b.whoami
	if b.whoami.Account != nil {
		acc := *b.whoami.Account
		cp.Account = &acc
	}
	if b.whoami.User != nil {
		u := *b.whoami.User
		cp.User = &u
	}
	return &cp, nil
}

func (b *demoBackend) ListDomains(ctx context.Context) ([]dnsimple.Domain, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]dnsimple.Domain, 0, len(b.domains))
	for _, d := range b.domains {
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (b *demoBackend) GetDomain(ctx context.Context, name string) (*dnsimple.Domain, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	d, ok := b.domains[name]
	if !ok {
		return nil, fmt.Errorf("domain not found (demo): %s", name)
	}
	cp := d
	return &cp, nil
}

func (b *demoBackend) DeleteDomain(ctx context.Context, name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.domains[name]; !ok {
		return fmt.Errorf("domain not found (demo): %s", name)
	}
	delete(b.domains, name)
	delete(b.zones, name)
	delete(b.records, name)
	return nil
}

func (b *demoBackend) ListZones(ctx context.Context) ([]dnsimple.Zone, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]dnsimple.Zone, 0, len(b.zones))
	for _, z := range b.zones {
		out = append(out, z)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (b *demoBackend) GetZone(ctx context.Context, name string) (*dnsimple.Zone, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	z, ok := b.zones[name]
	if !ok {
		return nil, fmt.Errorf("zone not found (demo): %s", name)
	}
	cp := z
	return &cp, nil
}

func (b *demoBackend) GetZoneFile(ctx context.Context, name string) (string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if _, ok := b.zones[name]; !ok {
		return "", fmt.Errorf("zone not found (demo): %s", name)
	}
	recs := b.records[name]
	var lines []string
	lines = append(lines, "$ORIGIN "+name+".")
	lines = append(lines, "@ 3600 IN SOA ns1.dnsimple.com. admin.dnsimple.com. 1 7200 3600 1209600 3600")
	for _, r := range recs {
		n := r.Name
		if n == "" {
			n = "@"
		}
		line := fmt.Sprintf("%s %d IN %s %s", n, r.TTL, r.Type, r.Content)
		if r.Priority != 0 {
			line = fmt.Sprintf("%s %d IN %s %d %s", n, r.TTL, r.Type, r.Priority, r.Content)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n"), nil
}

func (b *demoBackend) CheckZoneDistribution(ctx context.Context, name string) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if z, ok := b.zones[name]; ok {
		return z.Active, nil
	}
	return false, fmt.Errorf("zone not found (demo): %s", name)
}

func (b *demoBackend) ActivateZoneDNS(ctx context.Context, name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	z, ok := b.zones[name]
	if !ok {
		return fmt.Errorf("zone not found (demo): %s", name)
	}
	z.Active = true
	b.zones[name] = z
	return nil
}

func (b *demoBackend) DeactivateZoneDNS(ctx context.Context, name string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	z, ok := b.zones[name]
	if !ok {
		return fmt.Errorf("zone not found (demo): %s", name)
	}
	z.Active = false
	b.zones[name] = z
	return nil
}

func (b *demoBackend) ListRecords(ctx context.Context, zone string) ([]dnsimple.ZoneRecord, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	recs, ok := b.records[zone]
	if !ok {
		return nil, fmt.Errorf("zone not found (demo): %s", zone)
	}
	out := append([]dnsimple.ZoneRecord(nil), recs...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (b *demoBackend) GetRecord(ctx context.Context, zone string, recordID int64) (*dnsimple.ZoneRecord, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for _, r := range b.records[zone] {
		if r.ID == recordID {
			cp := r
			return &cp, nil
		}
	}
	return nil, fmt.Errorf("record not found (demo): %d", recordID)
}

func (b *demoBackend) CheckRecordDistribution(ctx context.Context, zone string, recordID int64) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	recs, ok := b.records[zone]
	if !ok {
		return false, fmt.Errorf("zone not found (demo): %s", zone)
	}
	for _, r := range recs {
		if r.ID == recordID {
			return r.ID%2 == 0, nil
		}
	}
	return false, fmt.Errorf("record not found (demo): %d", recordID)
}

func (b *demoBackend) DeleteRecord(ctx context.Context, zone string, recordID int64) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	recs, ok := b.records[zone]
	if !ok {
		return fmt.Errorf("zone not found (demo): %s", zone)
	}
	for i, r := range recs {
		if r.ID == recordID {
			b.records[zone] = append(recs[:i], recs[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("record not found (demo): %d", recordID)
}

func (b *demoBackend) seed() {
	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	b.whoami = &dnsimple.WhoamiData{
		Account: &dnsimple.Account{
			ID:             424242,
			Email:          "demo@dnsimplectl.local",
			PlanIdentifier: "professional",
		},
	}

	demoNames := []string{
		"absurdophile.com", "acme.dev", "alpha-example.net", "beta-labs.io", "bluebird.ai",
		"canvasworks.co", "deltaops.com", "echovalley.org", "foxtrotapps.dev", "glaciermail.com",
		"harborstack.io", "ivorypixel.net", "jupiterhub.app", "kineticdata.dev", "lighthouse.tools",
		"mintorchard.com", "northfieldhq.com", "opalroute.io", "paperplane.dev", "quietforest.org",
		"rangergrid.com", "signalpath.io", "tideline.app", "umbraworks.dev", "vectorlane.net",
	}

	var nextDomainID int64 = 1028000
	var nextZoneID int64 = 972300
	var nextRecordID int64 = 8800000
	for i, name := range demoNames {
		d := dnsimple.Domain{
			ID:           nextDomainID + int64(i),
			Name:         name,
			UnicodeName:  name,
			State:        "hosted",
			AutoRenew:    i%3 == 0,
			PrivateWhois: i%2 == 0,
			ExpiresAt:    "2027-12-31T00:00:00Z",
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		b.domains[name] = d

		z := dnsimple.Zone{
			ID:        nextZoneID + int64(i),
			Name:      name,
			Active:    i%4 != 0,
			Reverse:   false,
			Secondary: i%9 == 0,
			CreatedAt: now,
			UpdatedAt: now,
		}
		b.zones[name] = z

		txt := "v=spf1 include:_spf.google.com include:mailgun.org include:amazonses.com ip4:192.0.2.42 ip4:198.51.100.17 ~all"
		if i%5 == 0 {
			txt = txt + " demo-segment=" + strings.Repeat("abcdef0123456789", 8)
		}

		recs := []dnsimple.ZoneRecord{
			{
				ID:           nextRecordID + int64(i*10) + 1,
				Type:         "A",
				Name:         "",
				Content:      "203.0.113." + fmt.Sprintf("%d", (i%200)+10),
				TTL:          3600,
				SystemRecord: false,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			{
				ID:           nextRecordID + int64(i*10) + 2,
				Type:         "CNAME",
				Name:         "www",
				Content:      name + ".",
				TTL:          3600,
				SystemRecord: false,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			{
				ID:           nextRecordID + int64(i*10) + 3,
				Type:         "MX",
				Name:         "",
				Content:      "mail." + name + ".",
				TTL:          3600,
				Priority:     10,
				SystemRecord: false,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			{
				ID:           nextRecordID + int64(i*10) + 4,
				Type:         "TXT",
				Name:         "",
				Content:      txt,
				TTL:          3600,
				SystemRecord: false,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
			{
				ID:           nextRecordID + int64(i*10) + 5,
				Type:         "TXT",
				Name:         "_acme-challenge",
				Content:      strings.Repeat("challenge-token-", 7) + fmt.Sprintf("%d", i),
				TTL:          600,
				SystemRecord: false,
				CreatedAt:    now,
				UpdatedAt:    now,
			},
		}
		b.records[name] = recs
	}
}
