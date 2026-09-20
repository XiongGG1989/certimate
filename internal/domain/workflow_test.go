package domain

import "testing"

func TestWorkflowNodeConfigAsBizUploadLibrary(t *testing.T) {
	config := WorkflowNodeConfig{
		"source":        "library",
		"certificateId": "library-cert-id",
	}

	got := config.AsBizUpload()
	if got.Source != "library" || got.CertificateId != "library-cert-id" {
		t.Fatalf("AsBizUpload() = %#v", got)
	}
}
