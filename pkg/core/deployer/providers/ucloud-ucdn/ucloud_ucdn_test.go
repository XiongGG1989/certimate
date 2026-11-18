package uclouducdn

import (
	"context"
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"

	"github.com/certimate-go/certimate/pkg/core"
	ucloudsdk "github.com/certimate-go/certimate/pkg/sdk3rd/ucloud/ucdn"
)

func TestNormalizeDomainIds(t *testing.T) {
	t.Parallel()

	got := normalizeDomainIds([]string{" domain-1,domain-2\ndomain-3 ", "domain-1；domain-4", "  "})
	want := []string{"domain-1", "domain-2", "domain-3", "domain-4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeDomainIds() = %v, want %v", got, want)
	}
}

func TestNormalizeDomains(t *testing.T) {
	t.Parallel()

	got := normalizeDomains([]string{" CDN-A.EXAMPLE.COM;cdn-b.example.com ", "cdn-a.example.com，cdn-c.example.com"})
	want := []string{"cdn-a.example.com", "cdn-b.example.com", "cdn-c.example.com"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeDomains() = %v, want %v", got, want)
	}
}

func TestOrderDomainConfigs(t *testing.T) {
	t.Parallel()

	domainConfigs := []ucloudsdk.DomainConfigInfo{
		{DomainId: "domain-2", AreaCode: "abroad"},
		{DomainId: "domain-1", AreaCode: "cn"},
	}
	got, err := orderDomainConfigs([]string{"domain-1", "domain-2"}, domainConfigs, false)
	if err != nil {
		t.Fatalf("orderDomainConfigs() error = %v", err)
	}
	if got[0].DomainId != "domain-1" || got[1].DomainId != "domain-2" {
		t.Fatalf("orderDomainConfigs() = %v, want input order", got)
	}
}

func TestOrderDomainConfigsReportsAllMissingDomains(t *testing.T) {
	t.Parallel()

	_, err := orderDomainConfigs(
		[]string{"domain-1", "domain-2", "domain-3"},
		[]ucloudsdk.DomainConfigInfo{{DomainId: "domain-2"}},
		false,
	)
	if err == nil {
		t.Fatal("orderDomainConfigs() error = nil, want missing domain error")
	}
	if !strings.Contains(err.Error(), "domain-1") || !strings.Contains(err.Error(), "domain-3") {
		t.Fatalf("orderDomainConfigs() error = %v, want all missing domain IDs", err)
	}
}

func TestOrderDomainConfigsByDomain(t *testing.T) {
	t.Parallel()

	domainConfigs := []ucloudsdk.DomainConfigInfo{
		{Domain: "cdn-b.example.com", DomainId: "domain-2"},
		{Domain: "CDN-A.EXAMPLE.COM", DomainId: "domain-1"},
	}
	got, err := orderDomainConfigs([]string{"cdn-a.example.com", "cdn-b.example.com"}, domainConfigs, true)
	if err != nil {
		t.Fatalf("orderDomainConfigs() error = %v", err)
	}
	if got[0].DomainId != "domain-1" || got[1].DomainId != "domain-2" {
		t.Fatalf("orderDomainConfigs() = %v, want input domain order", got)
	}
}

func TestDeployResolvesDomainsToDomainIds(t *testing.T) {
	t.Parallel()

	client := &fakeUCDNClient{
		domainConfigs: []ucloudsdk.DomainConfigInfo{
			{Domain: "cdn-b.example.com", DomainId: "domain-2", AreaCode: "abroad", HttpsStatusAbroad: "enable"},
			{Domain: "cdn-a.example.com", DomainId: "domain-1", AreaCode: "cn", HttpsStatusCn: "enable"},
		},
	}
	certmgr := &fakeCertmgr{uploadResult: &core.CertmgrUploadResult{CertId: "123", CertName: "cert-name"}}
	deployer := &Deployer{
		config: &DeployerConfig{
			Domains:   []string{" CDN-A.EXAMPLE.COM, cdn-b.example.com\ncdn-a.example.com "},
			DomainIds: []string{"legacy-domain-id"},
		},
		logger:     slog.New(slog.DiscardHandler),
		sdkClient:  client,
		sdkCertmgr: certmgr,
	}

	if _, err := deployer.Deploy(context.Background(), "certificate", "private-key"); err != nil {
		t.Fatalf("Deploy() error = %v", err)
	}
	if !reflect.DeepEqual(client.queriedDomains, []string{"cdn-a.example.com", "cdn-b.example.com"}) {
		t.Fatalf("GetUcdnDomainConfig() domains = %v", client.queriedDomains)
	}
	if len(client.queriedDomainIds) != 0 {
		t.Fatalf("GetUcdnDomainConfig() domain IDs = %v, want none", client.queriedDomainIds)
	}
	if got := requestDomainIds(client.updateRequests); !reflect.DeepEqual(got, []string{"domain-1", "domain-2"}) {
		t.Fatalf("UpdateUcdnDomainHttpsConfigV2() domain IDs = %v", got)
	}
	if certmgr.uploadCalls != 1 {
		t.Fatalf("Upload() calls = %d, want 1", certmgr.uploadCalls)
	}
}

func TestDeployUpdatesAllDomainsWithOneUploadedCertificate(t *testing.T) {
	t.Parallel()

	client := &fakeUCDNClient{
		domainConfigs: []ucloudsdk.DomainConfigInfo{
			{DomainId: "domain-2", AreaCode: "abroad", HttpsStatusAbroad: "enable"},
			{DomainId: "domain-1", AreaCode: "cn", HttpsStatusCn: "enable"},
			{DomainId: "domain-3", AreaCode: "all", HttpsStatusCn: "enable", HttpsStatusAbroad: "disable"},
		},
		updateErrors: map[string]error{
			"domain-1": errors.New("first failure"),
			"domain-3": errors.New("third failure"),
		},
	}
	certmgr := &fakeCertmgr{uploadResult: &core.CertmgrUploadResult{CertId: "123", CertName: "cert-name"}}
	deployer := &Deployer{
		config:     &DeployerConfig{DomainIds: []string{" domain-1 ", "domain-2", "domain-1", "domain-3"}},
		logger:     slog.New(slog.DiscardHandler),
		sdkClient:  client,
		sdkCertmgr: certmgr,
	}

	_, err := deployer.Deploy(context.Background(), "certificate", "private-key")
	if err == nil {
		t.Fatal("Deploy() error = nil, want aggregated update errors")
	}
	if !strings.Contains(err.Error(), "domain-1") || !strings.Contains(err.Error(), "domain-3") {
		t.Fatalf("Deploy() error = %v, want every failed domain ID", err)
	}
	if certmgr.uploadCalls != 1 {
		t.Fatalf("Upload() calls = %d, want 1", certmgr.uploadCalls)
	}
	if !reflect.DeepEqual(client.queriedDomainIds, []string{"domain-1", "domain-2", "domain-3"}) {
		t.Fatalf("GetUcdnDomainConfig() domain IDs = %v", client.queriedDomainIds)
	}
	if got := requestDomainIds(client.updateRequests); !reflect.DeepEqual(got, []string{"domain-1", "domain-2", "domain-3"}) {
		t.Fatalf("UpdateUcdnDomainHttpsConfigV2() domain IDs = %v", got)
	}

	first := client.updateRequests[0]
	if first.HttpsStatusCn == nil || *first.HttpsStatusCn != "enable" || first.HttpsStatusAbroad != nil {
		t.Fatalf("domestic HTTPS status request = %#v", first)
	}
	second := client.updateRequests[1]
	if second.HttpsStatusCn != nil || second.HttpsStatusAbroad == nil || *second.HttpsStatusAbroad != "enable" {
		t.Fatalf("overseas HTTPS status request = %#v", second)
	}
	third := client.updateRequests[2]
	if third.HttpsStatusCn == nil || third.HttpsStatusAbroad == nil {
		t.Fatalf("global HTTPS status request = %#v", third)
	}
}

func TestDeployDoesNotUploadWhenDomainPreflightFails(t *testing.T) {
	t.Parallel()

	client := &fakeUCDNClient{domainConfigs: []ucloudsdk.DomainConfigInfo{{DomainId: "domain-1"}}}
	certmgr := &fakeCertmgr{uploadResult: &core.CertmgrUploadResult{CertId: "123", CertName: "cert-name"}}
	deployer := &Deployer{
		config:     &DeployerConfig{DomainIds: []string{"domain-1", "missing-domain"}},
		logger:     slog.New(slog.DiscardHandler),
		sdkClient:  client,
		sdkCertmgr: certmgr,
	}

	_, err := deployer.Deploy(context.Background(), "certificate", "private-key")
	if err == nil || !strings.Contains(err.Error(), "missing-domain") {
		t.Fatalf("Deploy() error = %v, want missing domain error", err)
	}
	if certmgr.uploadCalls != 0 {
		t.Fatalf("Upload() calls = %d, want 0", certmgr.uploadCalls)
	}
	if len(client.updateRequests) != 0 {
		t.Fatalf("update calls = %d, want 0", len(client.updateRequests))
	}
}

func requestDomainIds(requests []*ucloudsdk.UpdateUcdnDomainHttpsConfigV2Request) []string {
	result := make([]string, 0, len(requests))
	for _, request := range requests {
		if request.DomainId != nil {
			result = append(result, *request.DomainId)
		}
	}
	return result
}

type fakeUCDNClient struct {
	domainConfigs    []ucloudsdk.DomainConfigInfo
	queryError       error
	updateErrors     map[string]error
	queriedDomains   []string
	queriedDomainIds []string
	updateRequests   []*ucloudsdk.UpdateUcdnDomainHttpsConfigV2Request
}

func (c *fakeUCDNClient) NewGetUcdnDomainConfigRequest() *ucloudsdk.GetUcdnDomainConfigRequest {
	return &ucloudsdk.GetUcdnDomainConfigRequest{}
}

func (c *fakeUCDNClient) GetUcdnDomainConfig(request *ucloudsdk.GetUcdnDomainConfigRequest) (*ucloudsdk.GetUcdnDomainConfigResponse, error) {
	c.queriedDomains = append([]string(nil), request.Domain...)
	c.queriedDomainIds = append([]string(nil), request.DomainId...)
	return &ucloudsdk.GetUcdnDomainConfigResponse{DomainList: c.domainConfigs}, c.queryError
}

func (c *fakeUCDNClient) NewUpdateUcdnDomainHttpsConfigV2Request() *ucloudsdk.UpdateUcdnDomainHttpsConfigV2Request {
	return &ucloudsdk.UpdateUcdnDomainHttpsConfigV2Request{}
}

func (c *fakeUCDNClient) UpdateUcdnDomainHttpsConfigV2(request *ucloudsdk.UpdateUcdnDomainHttpsConfigV2Request) (*ucloudsdk.UpdateUcdnDomainHttpsConfigV2Response, error) {
	c.updateRequests = append(c.updateRequests, request)
	if request.DomainId != nil {
		return &ucloudsdk.UpdateUcdnDomainHttpsConfigV2Response{}, c.updateErrors[*request.DomainId]
	}
	return &ucloudsdk.UpdateUcdnDomainHttpsConfigV2Response{}, nil
}

type fakeCertmgr struct {
	uploadResult *core.CertmgrUploadResult
	uploadError  error
	uploadCalls  int
}

func (m *fakeCertmgr) SetLogger(_ *slog.Logger) {}

func (m *fakeCertmgr) Upload(_ context.Context, _, _ string) (*core.CertmgrUploadResult, error) {
	m.uploadCalls++
	return m.uploadResult, m.uploadError
}

func (m *fakeCertmgr) Replace(_ context.Context, _, _, _ string) (*core.CertmgrReplaceResult, error) {
	return nil, errors.New("not implemented")
}
