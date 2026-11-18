package certmgmt

import (
	"context"
	"fmt"
	"strings"

	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/internal/domain/dtos"
	uclouducdn "github.com/certimate-go/certimate/pkg/core/deployer/providers/ucloud-ucdn"
	xcerthostname "github.com/certimate-go/certimate/pkg/utils/cert/hostname"
	xmaps "github.com/certimate-go/certimate/pkg/utils/maps"
)

type DeploymentResourceService struct {
	accessRepo              accessRepository
	certificateRepo         deploymentCertificateRepository
	listUCloudUCDNDomainsFn func(ctx context.Context, config *uclouducdn.DeployerConfig) ([]string, error)
}

func NewDeploymentResourceService(accessRepo accessRepository, certificateRepo deploymentCertificateRepository) *DeploymentResourceService {
	return &DeploymentResourceService{
		accessRepo:              accessRepo,
		certificateRepo:         certificateRepo,
		listUCloudUCDNDomainsFn: uclouducdn.ListDomains,
	}
}

func (s *DeploymentResourceService) ListUCloudUCDNDomains(ctx context.Context, req *dtos.DeploymentUCloudUCDNListDomainsReq) (*dtos.DeploymentUCloudUCDNListDomainsResp, error) {
	if req == nil || req.AccessId == "" {
		return nil, domain.ErrInvalidParams
	}

	access, err := s.accessRepo.GetById(ctx, req.AccessId)
	if err != nil {
		return nil, fmt.Errorf("failed to get access #%s record: %w", req.AccessId, err)
	}
	if access.Provider != domain.AccessProviderTypeUCloud.String() {
		return nil, fmt.Errorf("access #%s is not available for UCloud", req.AccessId)
	}

	credentials := domain.AccessConfigForUCloud{}
	if err := xmaps.Populate(access.Config, &credentials); err != nil {
		return nil, fmt.Errorf("failed to populate UCloud access config: %w", err)
	}

	subjectAltNames, err := s.resolveCertificateSubjectAltNames(ctx, req)
	if err != nil {
		return nil, err
	}
	if req.FilterByCertificate && len(subjectAltNames) == 0 {
		return &dtos.DeploymentUCloudUCDNListDomainsResp{Domains: []string{}}, nil
	}

	domains, err := s.listUCloudUCDNDomainsFn(ctx, &uclouducdn.DeployerConfig{
		PrivateKey: credentials.PrivateKey,
		PublicKey:  credentials.PublicKey,
		ProjectId:  credentials.ProjectId,
		Endpoint:   req.Endpoint,
	})
	if err != nil {
		return nil, err
	}

	if req.FilterByCertificate {
		matchedDomains := make([]string, 0, len(domains))
		for _, domainName := range domains {
			for _, subjectAltName := range subjectAltNames {
				if xcerthostname.IsMatch(subjectAltName, domainName) {
					matchedDomains = append(matchedDomains, domainName)
					break
				}
			}
		}
		domains = matchedDomains
	}

	return &dtos.DeploymentUCloudUCDNListDomainsResp{Domains: domains}, nil
}

func (s *DeploymentResourceService) resolveCertificateSubjectAltNames(ctx context.Context, req *dtos.DeploymentUCloudUCDNListDomainsReq) ([]string, error) {
	if !req.FilterByCertificate {
		return nil, nil
	}

	subjectAltNames := normalizeSubjectAltNames(req.SubjectAltNames)
	if len(subjectAltNames) > 0 {
		return subjectAltNames, nil
	}

	var certificate *domain.Certificate
	var err error
	if req.CertificateId != "" {
		certificate, err = s.certificateRepo.GetById(ctx, req.CertificateId)
	} else if req.CertificateWorkflowId != "" && req.CertificateOutputNodeId != "" {
		certificate, err = s.certificateRepo.GetByWorkflowIdAndNodeId(ctx, req.CertificateWorkflowId, req.CertificateOutputNodeId)
	} else {
		return nil, nil
	}
	if err != nil {
		if domain.IsRecordNotFoundError(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get certificate for UCDN domain filtering: %w", err)
	}

	if certificate == nil {
		return nil, nil
	}
	return normalizeSubjectAltNames([]string{certificate.SubjectAltNames}), nil
}

func normalizeSubjectAltNames(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		for _, item := range strings.FieldsFunc(value, func(r rune) bool {
			return r == ';' || r == '；' || r == ',' || r == '，' || r == '\n' || r == '\r'
		}) {
			item = strings.ToLower(strings.TrimSpace(item))
			if item == "" {
				continue
			}
			if _, ok := seen[item]; ok {
				continue
			}
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
