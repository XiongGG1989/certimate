package certmgmt

import (
	"context"

	"github.com/certimate-go/certimate/internal/domain"
)

type accessRepository interface {
	GetById(ctx context.Context, id string) (*domain.Access, error)
}

type deploymentCertificateRepository interface {
	GetById(ctx context.Context, id string) (*domain.Certificate, error)
	GetByWorkflowIdAndNodeId(ctx context.Context, workflowId string, workflowNodeId string) (*domain.Certificate, error)
}
