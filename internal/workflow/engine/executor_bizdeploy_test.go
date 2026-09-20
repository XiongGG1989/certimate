package engine

import (
	"reflect"
	"testing"

	"github.com/certimate-go/certimate/internal/domain"
)

func TestApplyPersistedProviderConfig(t *testing.T) {
	tests := []struct {
		name       string
		config     map[string]any
		lastOutput *domain.WorkflowOutput
		want       map[string]any
	}{
		{
			name:   "last successful output overrides form value",
			config: map[string]any{"certificateId": "form-cert-id", "resourceProducts": "cdn"},
			lastOutput: &domain.WorkflowOutput{Succeeded: true, Outputs: []*domain.WorkflowOutputEntry{
				{Name: "providerConfig.certificateId", Value: "latest-cert-id", ValueType: stateValTypeString},
			}},
			want: map[string]any{"certificateId": "latest-cert-id", "resourceProducts": "cdn"},
		},
		{
			name:   "form value is retained without output",
			config: map[string]any{"certificateId": "form-cert-id"},
			want:   map[string]any{"certificateId": "form-cert-id"},
		},
		{
			name:   "failed output is ignored",
			config: map[string]any{"certificateId": "form-cert-id"},
			lastOutput: &domain.WorkflowOutput{Succeeded: false, Outputs: []*domain.WorkflowOutputEntry{
				{Name: "providerConfig.certificateId", Value: "failed-cert-id", ValueType: stateValTypeString},
			}},
			want: map[string]any{"certificateId": "form-cert-id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			applyPersistedProviderConfig(tt.config, tt.lastOutput)
			if !reflect.DeepEqual(tt.config, tt.want) {
				t.Fatalf("applyPersistedProviderConfig() = %#v, want %#v", tt.config, tt.want)
			}
		})
	}
}
