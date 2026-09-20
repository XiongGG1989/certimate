package certmgmt

import (
	"context"
	"log/slog"
	"testing"

	"github.com/certimate-go/certimate/internal/certmgmt/deployers"
	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/pkg/core"
)

type deployResultTestProvider struct{}

func (*deployResultTestProvider) SetLogger(*slog.Logger) {}

func (*deployResultTestProvider) Deploy(context.Context, string, string) (*core.DeployerDeployResult, error) {
	return &core.DeployerDeployResult{ExtendedData: map[string]any{
		"providerConfig.certificateId": "new-cert-id",
	}}, nil
}

func TestDeployCertificatePropagatesExtendedData(t *testing.T) {
	providerType := domain.DeploymentProviderType("test-deploy-result-propagation")
	if err := deployers.Registries.Register(providerType, func(*deployers.ProviderFactoryOptions) (core.Deployer, error) {
		return &deployResultTestProvider{}, nil
	}); err != nil {
		t.Fatal(err)
	}

	response, err := NewClient().DeployCertificate(context.Background(), &DeployCertificateRequest{Provider: providerType})
	if err != nil {
		t.Fatal(err)
	}
	if got := response.ExtendedData["providerConfig.certificateId"]; got != "new-cert-id" {
		t.Fatalf("certificate ID = %v, want %q", got, "new-cert-id")
	}
}
