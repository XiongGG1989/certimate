package dispatcher

import (
	"context"
	"maps"

	"github.com/certimate-go/certimate/internal/domain"
)

const providerConfigOutputPrefix = "providerConfig."

type providerConfigUpdate struct {
	Provider string
	Values   map[string]any
}

func (wd *workflowDispatcher) syncProviderConfigUpdates(ctx context.Context, workflowId string, updates map[string]providerConfigUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	workflow, err := wd.workflowRepo.GetById(ctx, workflowId)
	if err != nil {
		return err
	}

	changed := applyProviderConfigUpdates(workflow.GraphContent, updates)
	changed = applyProviderConfigUpdates(workflow.GraphDraft, updates) || changed
	if !changed {
		return nil
	}

	_, err = wd.workflowRepo.Save(ctx, workflow)
	return err
}

func applyProviderConfigUpdates(graph *domain.WorkflowGraph, updates map[string]providerConfigUpdate) bool {
	if graph == nil || len(updates) == 0 {
		return false
	}

	changed := false
	for nodeId, update := range updates {
		node, ok := graph.GetNodeById(nodeId)
		if !ok || node.Type != domain.WorkflowNodeTypeBizDeploy {
			continue
		}

		nodeConfig := node.Data.Config.AsBizDeploy()
		if nodeConfig.Provider != update.Provider {
			continue
		}

		providerConfig := maps.Clone(nodeConfig.ProviderConfig)
		if providerConfig == nil {
			providerConfig = make(map[string]any)
		}
		for key, value := range update.Values {
			if providerConfig[key] != value {
				providerConfig[key] = value
				changed = true
			}
		}
		node.Data.Config["providerConfig"] = providerConfig
	}

	return changed
}
