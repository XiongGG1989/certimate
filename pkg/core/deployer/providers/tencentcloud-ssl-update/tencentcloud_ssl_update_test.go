package tencentcloudsslupdate

import (
	"reflect"
	"testing"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tcsslmodel "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"

	tcssl "github.com/certimate-go/certimate/pkg/sdk3rd-trimmed/github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"
)

func TestValidateResourceProducts(t *testing.T) {
	tests := []struct {
		name             string
		resourceProducts []string
		resourceRegions  []string
		isReplaced       bool
		wantErr          bool
	}{
		{name: "global resource", resourceProducts: []string{"cdn"}},
		{name: "regional resource", resourceProducts: []string{"clb"}, resourceRegions: []string{"ap-guangzhou"}},
		{name: "new regional resource types", resourceProducts: []string{"mqtt", "scf", "tdmq"}, resourceRegions: []string{"ap-guangzhou"}},
		{name: "regional resource without region", resourceProducts: []string{"scf"}, wantErr: true},
		{name: "unsupported resource", resourceProducts: []string{"unknown"}, wantErr: true},
		{name: "replace certificate for clb", resourceProducts: []string{"clb"}, resourceRegions: []string{"ap-guangzhou"}, isReplaced: true},
		{name: "replace certificate for unsupported resource", resourceProducts: []string{"cdn"}, isReplaced: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResourceProducts(tt.resourceProducts, tt.resourceRegions, tt.isReplaced)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateResourceProducts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestWrapResourceProductRegions(t *testing.T) {
	got := wrapResourceProductRegions(
		[]string{"cdn", "clb", "mqtt", "scf", "tdmq"},
		[]string{"ap-guangzhou", "ap-shanghai"},
	)
	want := []*tcssl.ResourceTypeRegions{
		{ResourceType: common.StringPtr("clb"), Regions: common.StringPtrs([]string{"ap-guangzhou", "ap-shanghai"})},
		{ResourceType: common.StringPtr("mqtt"), Regions: common.StringPtrs([]string{"ap-guangzhou", "ap-shanghai"})},
		{ResourceType: common.StringPtr("scf"), Regions: common.StringPtrs([]string{"ap-guangzhou", "ap-shanghai"})},
		{ResourceType: common.StringPtr("tdmq"), Regions: common.StringPtrs([]string{"ap-guangzhou", "ap-shanghai"})},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("wrapResourceProductRegions() = %#v, want %#v", got, want)
	}
}

func TestResolveDeployRecordId(t *testing.T) {
	statusCreated := int64(1)
	statusExisting := int64(0)
	pendingRecordId := uint64(0)
	createdRecordId := uint64(123)

	tests := []struct {
		name           string
		deployStatus   *int64
		deployRecordId *uint64
		wantRecordId   uint64
		wantReady      bool
		wantErr        bool
	}{
		{name: "pending", deployStatus: &statusCreated, deployRecordId: &pendingRecordId},
		{name: "created", deployStatus: &statusCreated, deployRecordId: &createdRecordId, wantRecordId: 123, wantReady: true},
		{name: "existing task", deployStatus: &statusExisting, deployRecordId: &createdRecordId, wantRecordId: 123, wantReady: true},
		{name: "missing status", deployRecordId: &createdRecordId, wantErr: true},
		{name: "missing record id", deployStatus: &statusCreated, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recordId, ready, err := resolveDeployRecordId(tt.deployStatus, tt.deployRecordId)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveDeployRecordId() error = %v, wantErr %v", err, tt.wantErr)
			}
			if recordId != tt.wantRecordId || ready != tt.wantReady {
				t.Fatalf("resolveDeployRecordId() = (%d, %t), want (%d, %t)", recordId, ready, tt.wantRecordId, tt.wantReady)
			}
		})
	}
}

func TestResolveNewCertificateId(t *testing.T) {
	recordDetails := []*tcsslmodel.UpdateRecordDetails{
		nil,
		{List: []*tcsslmodel.UpdateRecordDetail{
			nil,
			{CertId: common.StringPtr("")},
			{CertId: common.StringPtr("new-cert-id")},
		}},
	}

	if got := resolveNewCertificateId(recordDetails); got != "new-cert-id" {
		t.Fatalf("resolveNewCertificateId() = %q, want %q", got, "new-cert-id")
	}
	if got := resolveNewCertificateId(nil); got != "" {
		t.Fatalf("resolveNewCertificateId(nil) = %q, want empty", got)
	}
}

func TestNewUpdateCertificateInstanceRequest(t *testing.T) {
	t.Run("certificate content", func(t *testing.T) {
		deployer := &Deployer{config: &DeployerConfig{
			CertificateId:         "old-cert-id",
			ProjectId:             123,
			ResourceProducts:      []string{"clb"},
			ResourceRegions:       []string{"ap-beijing"},
			UseCertificateContent: true,
		}}

		request := deployer.newUpdateCertificateInstanceRequest("certificate", "private-key", "")
		if request.CertificateId != nil {
			t.Fatal("CertificateId should be omitted when certificate content is used")
		}
		if request.CertificatePublicKey == nil || *request.CertificatePublicKey != "certificate" {
			t.Fatal("CertificatePublicKey was not set")
		}
		if request.CertificatePrivateKey == nil || *request.CertificatePrivateKey != "private-key" {
			t.Fatal("CertificatePrivateKey was not set")
		}
		if request.Repeatable == nil || !*request.Repeatable {
			t.Fatal("Repeatable should be enabled")
		}
		if request.ProjectId == nil || *request.ProjectId != 123 {
			t.Fatal("ProjectId was not set")
		}
	})

	t.Run("certificate id", func(t *testing.T) {
		deployer := &Deployer{config: &DeployerConfig{
			CertificateId:    "old-cert-id",
			ResourceProducts: []string{"cdn"},
		}}

		request := deployer.newUpdateCertificateInstanceRequest("certificate", "private-key", "new-cert-id")
		if request.CertificateId == nil || *request.CertificateId != "new-cert-id" {
			t.Fatal("CertificateId was not set")
		}
		if request.CertificatePublicKey != nil || request.CertificatePrivateKey != nil {
			t.Fatal("certificate content should be omitted when CertificateId is used")
		}
	})
}
