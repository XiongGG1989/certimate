package certmgmt

import (
	"context"
	"reflect"
	"testing"

	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/internal/domain/dtos"
	uclouducdn "github.com/certimate-go/certimate/pkg/core/deployer/providers/ucloud-ucdn"
)

func TestDeploymentResourceServiceListUCloudUCDNDomains(t *testing.T) {
	t.Parallel()

	accessRepo := &fakeAccessRepository{
		access: &domain.Access{
			Meta:     domain.Meta{Id: "access-1"},
			Provider: domain.AccessProviderTypeUCloud.String(),
			Config: map[string]any{
				"privateKey": "private-key",
				"publicKey":  "public-key",
				"projectId":  "project-id",
			},
		},
	}
	service := NewDeploymentResourceService(accessRepo, &fakeDeploymentCertificateRepository{})
	var gotConfig *uclouducdn.DeployerConfig
	service.listUCloudUCDNDomainsFn = func(_ context.Context, config *uclouducdn.DeployerConfig) ([]string, error) {
		gotConfig = config
		return []string{"cdn-a.example.com", "cdn-b.example.com"}, nil
	}

	response, err := service.ListUCloudUCDNDomains(context.Background(), &dtos.DeploymentUCloudUCDNListDomainsReq{
		AccessId: "access-1",
		Endpoint: "api.ucloud-global.com",
	})
	if err != nil {
		t.Fatalf("ListUCloudUCDNDomains() error = %v", err)
	}
	if !reflect.DeepEqual(response.Domains, []string{"cdn-a.example.com", "cdn-b.example.com"}) {
		t.Fatalf("ListUCloudUCDNDomains() domains = %v", response.Domains)
	}
	if gotConfig == nil || gotConfig.PrivateKey != "private-key" || gotConfig.PublicKey != "public-key" || gotConfig.ProjectId != "project-id" || gotConfig.Endpoint != "api.ucloud-global.com" {
		t.Fatalf("ListUCloudUCDNDomains() config = %#v", gotConfig)
	}
}

func TestDeploymentResourceServiceRejectsNonUCloudAccess(t *testing.T) {
	t.Parallel()

	service := NewDeploymentResourceService(&fakeAccessRepository{
		access: &domain.Access{Provider: domain.AccessProviderTypeAliyun.String(), Config: map[string]any{}},
	}, &fakeDeploymentCertificateRepository{})
	called := false
	service.listUCloudUCDNDomainsFn = func(_ context.Context, _ *uclouducdn.DeployerConfig) ([]string, error) {
		called = true
		return nil, nil
	}

	_, err := service.ListUCloudUCDNDomains(context.Background(), &dtos.DeploymentUCloudUCDNListDomainsReq{AccessId: "access-1"})
	if err == nil {
		t.Fatal("ListUCloudUCDNDomains() error = nil, want provider validation error")
	}
	if called {
		t.Fatal("ListUCloudUCDNDomains() called UCloud API for non-UCloud access")
	}
}

func TestDeploymentResourceServiceFiltersUCloudUCDNDomainsBySubjectAltNames(t *testing.T) {
	t.Parallel()

	service := NewDeploymentResourceService(&fakeAccessRepository{
		access: &domain.Access{
			Provider: domain.AccessProviderTypeUCloud.String(),
			Config:   map[string]any{"privateKey": "private-key", "publicKey": "public-key"},
		},
	}, &fakeDeploymentCertificateRepository{})
	service.listUCloudUCDNDomainsFn = func(_ context.Context, _ *uclouducdn.DeployerConfig) ([]string, error) {
		return []string{"cdn.example.com", "edge.video.example.com", "nested.edge.video.example.com", "other.example.net"}, nil
	}

	response, err := service.ListUCloudUCDNDomains(context.Background(), &dtos.DeploymentUCloudUCDNListDomainsReq{
		AccessId:            "access-1",
		FilterByCertificate: true,
		SubjectAltNames:     []string{"cdn.example.com", "*.video.example.com"},
	})
	if err != nil {
		t.Fatalf("ListUCloudUCDNDomains() error = %v", err)
	}
	want := []string{"cdn.example.com", "edge.video.example.com"}
	if !reflect.DeepEqual(response.Domains, want) {
		t.Fatalf("ListUCloudUCDNDomains() domains = %v, want %v", response.Domains, want)
	}
}

func TestDeploymentResourceServiceFiltersUCloudUCDNDomainsByCertificateId(t *testing.T) {
	t.Parallel()

	certificateRepo := &fakeDeploymentCertificateRepository{
		certificate: &domain.Certificate{SubjectAltNames: "cdn.example.com"},
	}
	service := NewDeploymentResourceService(&fakeAccessRepository{
		access: &domain.Access{
			Provider: domain.AccessProviderTypeUCloud.String(),
			Config:   map[string]any{"privateKey": "private-key", "publicKey": "public-key"},
		},
	}, certificateRepo)
	service.listUCloudUCDNDomainsFn = func(_ context.Context, _ *uclouducdn.DeployerConfig) ([]string, error) {
		return []string{"cdn.example.com", "other.example.net"}, nil
	}

	response, err := service.ListUCloudUCDNDomains(context.Background(), &dtos.DeploymentUCloudUCDNListDomainsReq{
		AccessId:            "access-1",
		FilterByCertificate: true,
		CertificateId:       "certificate-1",
	})
	if err != nil {
		t.Fatalf("ListUCloudUCDNDomains() error = %v", err)
	}
	if !reflect.DeepEqual(response.Domains, []string{"cdn.example.com"}) {
		t.Fatalf("ListUCloudUCDNDomains() domains = %v", response.Domains)
	}
	if certificateRepo.gotCertificateId != "certificate-1" {
		t.Fatalf("certificate lookup ID = %q", certificateRepo.gotCertificateId)
	}
}

func TestDeploymentResourceServiceFiltersUCloudUCDNDomainsByLatestNodeCertificate(t *testing.T) {
	t.Parallel()

	certificateRepo := &fakeDeploymentCertificateRepository{
		certificate: &domain.Certificate{SubjectAltNames: "cdn.example.com;*.video.example.com"},
	}
	service := NewDeploymentResourceService(&fakeAccessRepository{
		access: &domain.Access{
			Provider: domain.AccessProviderTypeUCloud.String(),
			Config:   map[string]any{"privateKey": "private-key", "publicKey": "public-key"},
		},
	}, certificateRepo)
	service.listUCloudUCDNDomainsFn = func(_ context.Context, _ *uclouducdn.DeployerConfig) ([]string, error) {
		return []string{"cdn.example.com", "edge.video.example.com", "other.example.net"}, nil
	}

	response, err := service.ListUCloudUCDNDomains(context.Background(), &dtos.DeploymentUCloudUCDNListDomainsReq{
		AccessId:                "access-1",
		FilterByCertificate:     true,
		CertificateWorkflowId:   "workflow-1",
		CertificateOutputNodeId: "node-1",
	})
	if err != nil {
		t.Fatalf("ListUCloudUCDNDomains() error = %v", err)
	}
	want := []string{"cdn.example.com", "edge.video.example.com"}
	if !reflect.DeepEqual(response.Domains, want) {
		t.Fatalf("ListUCloudUCDNDomains() domains = %v, want %v", response.Domains, want)
	}
	if certificateRepo.gotWorkflowId != "workflow-1" || certificateRepo.gotWorkflowNodeId != "node-1" {
		t.Fatalf("certificate lookup = workflow %q, node %q", certificateRepo.gotWorkflowId, certificateRepo.gotWorkflowNodeId)
	}
}

type fakeAccessRepository struct {
	access *domain.Access
	err    error
}

func (r *fakeAccessRepository) GetById(_ context.Context, _ string) (*domain.Access, error) {
	return r.access, r.err
}

type fakeDeploymentCertificateRepository struct {
	certificate       *domain.Certificate
	err               error
	gotCertificateId  string
	gotWorkflowId     string
	gotWorkflowNodeId string
}

func (r *fakeDeploymentCertificateRepository) GetById(_ context.Context, id string) (*domain.Certificate, error) {
	r.gotCertificateId = id
	return r.certificate, r.err
}

func (r *fakeDeploymentCertificateRepository) GetByWorkflowIdAndNodeId(_ context.Context, workflowId string, workflowNodeId string) (*domain.Certificate, error) {
	r.gotWorkflowId = workflowId
	r.gotWorkflowNodeId = workflowNodeId
	return r.certificate, r.err
}
