package handlers

import (
	"context"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"

	"github.com/certimate-go/certimate/internal/domain/dtos"
	"github.com/certimate-go/certimate/internal/rest/resp"
)

type deploymentResourceService interface {
	ListUCloudUCDNDomains(ctx context.Context, req *dtos.DeploymentUCloudUCDNListDomainsReq) (*dtos.DeploymentUCloudUCDNListDomainsResp, error)
}

type DeploymentResourcesHandler struct {
	service deploymentResourceService
}

func NewDeploymentResourcesHandler(router *router.RouterGroup[*core.RequestEvent], service deploymentResourceService) {
	handler := &DeploymentResourcesHandler{service: service}

	group := router.Group("/deployment/providers")
	group.POST("/ucloud-ucdn/domains", handler.listUCloudUCDNDomains)
}

func (handler *DeploymentResourcesHandler) listUCloudUCDNDomains(e *core.RequestEvent) error {
	req := &dtos.DeploymentUCloudUCDNListDomainsReq{}
	if err := e.BindBody(req); err != nil {
		return resp.Err(e, err)
	}

	res, err := handler.service.ListUCloudUCDNDomains(e.Request.Context(), req)
	if err != nil {
		return resp.Err(e, err)
	}

	return resp.Ok(e, res)
}
