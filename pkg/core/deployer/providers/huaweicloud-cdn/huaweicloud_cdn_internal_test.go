package huaweicloudcdn

import (
	"strings"
	"testing"

	hwcdnmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/cdn/v2/model"
)

func TestSplitDomains(t *testing.T) {
	t.Parallel()

	got := splitDomains(" a.example.com; b.example.com,\na.example.com，\r\n c.example.com ")
	want := []string{"a.example.com", "b.example.com", "c.example.com"}
	if len(got) != len(want) {
		t.Fatalf("splitDomains() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("splitDomains()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestValidateUpdateDomainsCertificateResponse(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		status := "success"
		domain := "a.example.com"
		if err := validateUpdateDomainsCertificateResponse(&hwcdnmodel.UpdateDomainMultiCertificatesResponse{
			Status: &status,
			Result: &[]hwcdnmodel.UpdateDomainMultiCertificatesResponseBodyResult{{DomainName: &domain, Status: &status}},
		}); err != nil {
			t.Fatalf("validateUpdateDomainsCertificateResponse() error = %v", err)
		}
	})

	t.Run("partial failure", func(t *testing.T) {
		status := "fail"
		domain := "b.example.com"
		reason := "domain is not found"
		err := validateUpdateDomainsCertificateResponse(&hwcdnmodel.UpdateDomainMultiCertificatesResponse{
			Status: &status,
			Result: &[]hwcdnmodel.UpdateDomainMultiCertificatesResponseBodyResult{{DomainName: &domain, Status: &status, Reason: &reason}},
		})
		if err == nil || !strings.Contains(err.Error(), domain) || !strings.Contains(err.Error(), reason) {
			t.Fatalf("validateUpdateDomainsCertificateResponse() error = %v, want domain and reason", err)
		}
	})

	t.Run("empty response", func(t *testing.T) {
		if err := validateUpdateDomainsCertificateResponse(nil); err == nil {
			t.Fatal("validateUpdateDomainsCertificateResponse() error = nil, want error")
		}
	})
}
