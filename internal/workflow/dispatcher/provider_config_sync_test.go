package dispatcher

import (
	"testing"

	"github.com/certimate-go/certimate/internal/domain"
)

func TestApplyProviderConfigUpdates(t *testing.T) {
	graph := &domain.WorkflowGraph{Nodes: []*domain.WorkflowNode{
		{
			Id:   "deploy-node",
			Type: domain.WorkflowNodeTypeBizDeploy,
			Data: domain.WorkflowNodeData{Config: domain.WorkflowNodeConfig{
				"provider": "tencentcloud-ssloneclickupdate",
				"providerConfig": map[string]any{
					"certificateId":    "old-cert-id",
					"resourceProducts": "cdn",
				},
			}},
		},
	}}

	changed := applyProviderConfigUpdates(graph, map[string]providerConfigUpdate{
		"deploy-node": {
			Provider: "tencentcloud-ssloneclickupdate",
			Values:   map[string]any{"certificateId": "new-cert-id"},
		},
	})
	if !changed {
		t.Fatal("applyProviderConfigUpdates() did not report a change")
	}
	got := graph.Nodes[0].Data.Config.AsBizDeploy().ProviderConfig["certificateId"]
	if got != "new-cert-id" {
		t.Fatalf("certificateId = %v, want %q", got, "new-cert-id")
	}
}

func TestApplyProviderConfigUpdatesDoesNotChangeAnotherProvider(t *testing.T) {
	graph := &domain.WorkflowGraph{Nodes: []*domain.WorkflowNode{
		{
			Id:   "deploy-node",
			Type: domain.WorkflowNodeTypeBizDeploy,
			Data: domain.WorkflowNodeData{Config: domain.WorkflowNodeConfig{
				"provider":       "another-provider",
				"providerConfig": map[string]any{"certificateId": "manual-cert-id"},
			}},
		},
	}}

	changed := applyProviderConfigUpdates(graph, map[string]providerConfigUpdate{
		"deploy-node": {
			Provider: "tencentcloud-ssloneclickupdate",
			Values:   map[string]any{"certificateId": "new-cert-id"},
		},
	})
	if changed {
		t.Fatal("applyProviderConfigUpdates() changed a node using another provider")
	}
}
