package engine

import (
	"testing"

	"github.com/certimate-go/certimate/internal/domain"
)

func TestGetCertificateIdFromWorkflowOutput(t *testing.T) {
	output := &domain.WorkflowOutput{Outputs: []*domain.WorkflowOutputEntry{
		nil,
		{Name: "other", Value: "certificate#ignored"},
		{Name: "certificate", Value: "certificate#library-cert-id"},
	}}

	if got := getCertificateIdFromWorkflowOutput(output); got != "library-cert-id" {
		t.Fatalf("getCertificateIdFromWorkflowOutput() = %q, want %q", got, "library-cert-id")
	}
	if got := getCertificateIdFromWorkflowOutput(nil); got != "" {
		t.Fatalf("getCertificateIdFromWorkflowOutput(nil) = %q, want empty", got)
	}
}
