package uclouducdn

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/ucloud/ucloud-sdk-go/ucloud"
	"github.com/ucloud/ucloud-sdk-go/ucloud/auth"

	"github.com/certimate-go/certimate/pkg/core"
	cmgrimpl "github.com/certimate-go/certimate/pkg/core/certmgr/providers/ucloud-ussl"
	ucloudsdk "github.com/certimate-go/certimate/pkg/sdk3rd/ucloud/ucdn"
	xloop "github.com/certimate-go/certimate/pkg/utils/loop"
)

type (
	Provider     = core.Deployer
	DeployResult = core.DeployerDeployResult
)

type DeployerConfig struct {
	// 优刻得 API 私钥。
	PrivateKey string `json:"privateKey"`
	// 优刻得 API 公钥。
	PublicKey string `json:"publicKey"`
	// 优刻得项目 ID。
	ProjectId string `json:"projectId,omitempty"`
	// 优刻得接口端点。
	Endpoint string `json:"endpoint,omitempty"`
	// 加速域名。
	Domains []string `json:"domains"`
	// 加速域名 ID。仅用于兼容旧配置。
	DomainIds []string `json:"domainIds,omitempty"`
}

type ucdnClient interface {
	NewGetUcdnDomainConfigRequest() *ucloudsdk.GetUcdnDomainConfigRequest
	GetUcdnDomainConfig(req *ucloudsdk.GetUcdnDomainConfigRequest) (*ucloudsdk.GetUcdnDomainConfigResponse, error)
	NewUpdateUcdnDomainHttpsConfigV2Request() *ucloudsdk.UpdateUcdnDomainHttpsConfigV2Request
	UpdateUcdnDomainHttpsConfigV2(req *ucloudsdk.UpdateUcdnDomainHttpsConfigV2Request) (*ucloudsdk.UpdateUcdnDomainHttpsConfigV2Response, error)
}

type Deployer struct {
	config     *DeployerConfig
	logger     *slog.Logger
	sdkClient  ucdnClient
	sdkCertmgr core.Certmgr
}

var _ Provider = (*Deployer)(nil)

func NewDeployer(config *DeployerConfig) (*Deployer, error) {
	if config == nil {
		return nil, fmt.Errorf("the configuration of the deployer provider is nil")
	}

	client, err := createSDKClient(config.PrivateKey, config.PublicKey, config.ProjectId, config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	pcertmgr, err := cmgrimpl.NewCertmgr(&cmgrimpl.CertmgrConfig{
		PrivateKey: config.PrivateKey,
		PublicKey:  config.PublicKey,
		ProjectId:  config.ProjectId,
		Endpoint:   config.Endpoint,
	})
	if err != nil {
		return nil, fmt.Errorf("could not create certmgr: %w", err)
	}

	return &Deployer{
		config:     config,
		logger:     slog.Default(),
		sdkClient:  client,
		sdkCertmgr: pcertmgr,
	}, nil
}

func (d *Deployer) SetLogger(logger *slog.Logger) {
	if logger == nil {
		d.logger = slog.New(slog.DiscardHandler)
	} else {
		d.logger = logger
	}

	d.sdkCertmgr.SetLogger(logger)
}

func (d *Deployer) Deploy(ctx context.Context, certPEM, privkeyPEM string) (*DeployResult, error) {
	domains := normalizeDomains(d.config.Domains)
	domainIds := normalizeDomainIds(d.config.DomainIds)
	if len(domains) == 0 && len(domainIds) == 0 {
		return nil, fmt.Errorf("config `domains` is required")
	}

	// 优先按加速域名查询；仅当新字段为空时回退到旧 DomainId 配置。
	domainConfigs, err := d.getDomainConfigs(domains, domainIds)
	if err != nil {
		return nil, err
	}

	// 所有域名共用同一份已上传证书。
	upres, err := d.sdkCertmgr.Upload(ctx, certPEM, privkeyPEM)
	if err != nil {
		return nil, fmt.Errorf("failed to upload certificate file: %w", err)
	}
	d.logger.Info("ssl certificate uploaded", slog.Any("result", upres))

	certId, err := strconv.Atoi(upres.CertId)
	if err != nil {
		return nil, fmt.Errorf("invalid uploaded certificate id '%s': %w", upres.CertId, err)
	}

	// UCloud 更新接口一次只接受一个 DomainId，因此逐域名更新并汇总失败。
	if err := xloop.ForRangeAllWithContext(ctx, domainConfigs, func(ctx context.Context, domainConfig ucloudsdk.DomainConfigInfo, _ int) error {
		if err := d.updateDomainCertificate(domainConfig, certId, upres.CertName); err != nil {
			target := domainConfig.Domain
			if target == "" {
				target = domainConfig.DomainId
			}
			return fmt.Errorf("failed to update domain '%s': %w", target, err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &DeployResult{}, nil
}

func (d *Deployer) getDomainConfigs(domains, domainIds []string) ([]ucloudsdk.DomainConfigInfo, error) {
	// REF: https://docs.ucloud.cn/api/ucdn-api/get_ucdn_domain_config
	request := d.sdkClient.NewGetUcdnDomainConfigRequest()
	lookupByDomain := len(domains) > 0
	targets := domainIds
	if lookupByDomain {
		request.Domain = domains
		targets = domains
	} else {
		request.DomainId = domainIds
	}

	response, err := d.sdkClient.GetUcdnDomainConfig(request)
	d.logger.Debug("sdk request 'ucdn.GetUcdnDomainConfig'", slog.Any("request", request), slog.Any("response", response))
	if err != nil {
		return nil, fmt.Errorf("failed to execute sdk request 'ucdn.GetUcdnDomainConfig': %w", err)
	}
	if response == nil {
		return nil, fmt.Errorf("sdk request 'ucdn.GetUcdnDomainConfig' returned an empty response")
	}

	return orderDomainConfigs(targets, response.DomainList, lookupByDomain)
}

func (d *Deployer) updateDomainCertificate(domainConfig ucloudsdk.DomainConfigInfo, certId int, certName string) error {
	// REF: https://docs.ucloud.cn/api/ucdn-api/update_ucdn_domain_https_config_v2
	request := d.sdkClient.NewUpdateUcdnDomainHttpsConfigV2Request()
	request.DomainId = ucloud.String(domainConfig.DomainId)
	request.HttpsStatusCn = lo.
		IfF(domainConfig.AreaCode == "all" || domainConfig.AreaCode == "cn", func() *string { return ucloud.String(domainConfig.HttpsStatusCn) }).
		Else(nil)
	request.HttpsStatusAbroad = lo.
		IfF(domainConfig.AreaCode == "all" || domainConfig.AreaCode == "abroad", func() *string { return ucloud.String(domainConfig.HttpsStatusAbroad) }).
		Else(nil)
	request.CertId = ucloud.Int(certId)
	request.CertName = ucloud.String(certName)
	request.CertType = ucloud.String("ussl")
	response, err := d.sdkClient.UpdateUcdnDomainHttpsConfigV2(request)
	d.logger.Debug("sdk request 'ucdn.UpdateUcdnDomainHttpsConfigV2'", slog.Any("request", request), slog.Any("response", response))
	if err != nil {
		return fmt.Errorf("failed to execute sdk request 'ucdn.UpdateUcdnDomainHttpsConfigV2': %w", err)
	}

	return nil
}

func normalizeDomains(domains []string) []string {
	return normalizeTargetValues(domains, true)
}

func normalizeDomainIds(domainIds []string) []string {
	return normalizeTargetValues(domainIds, false)
}

func normalizeTargetValues(values []string, lowercase bool) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		parts := strings.FieldsFunc(value, func(r rune) bool {
			return r == ';' || r == '；' || r == ',' || r == '，' || r == '\n' || r == '\r'
		})
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if lowercase {
				part = strings.ToLower(part)
			}
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}

			seen[part] = struct{}{}
			result = append(result, part)
		}
	}
	return result
}

func orderDomainConfigs(targets []string, domainConfigs []ucloudsdk.DomainConfigInfo, lookupByDomain bool) ([]ucloudsdk.DomainConfigInfo, error) {
	configsByTarget := make(map[string]ucloudsdk.DomainConfigInfo, len(domainConfigs))
	for _, domainConfig := range domainConfigs {
		target := domainConfig.DomainId
		if lookupByDomain {
			target = strings.ToLower(strings.TrimSpace(domainConfig.Domain))
		}
		configsByTarget[target] = domainConfig
	}

	result := make([]ucloudsdk.DomainConfigInfo, 0, len(targets))
	missingTargets := make([]string, 0)
	for _, target := range targets {
		domainConfig, ok := configsByTarget[target]
		if !ok {
			missingTargets = append(missingTargets, target)
			continue
		}
		result = append(result, domainConfig)
	}
	if len(missingTargets) > 0 {
		return nil, fmt.Errorf("could not find domains: %s", strings.Join(missingTargets, ", "))
	}

	return result, nil
}

func createSDKClient(privateKey, publicKey, projectId, endpoint string) (*ucloudsdk.UCDNClient, error) {
	if privateKey == "" {
		return nil, fmt.Errorf("ucloud: invalid private key")
	}
	if publicKey == "" {
		return nil, fmt.Errorf("ucloud: invalid public key")
	}

	cfg := ucloud.NewConfig()
	if projectId != "" {
		cfg.ProjectId = projectId
	}
	if endpoint != "" {
		if strings.Contains(endpoint, "://") {
			cfg.BaseUrl = endpoint
		} else {
			cfg.BaseUrl = "https://" + endpoint
		}
	}

	credential := auth.NewCredential()
	credential.PrivateKey = privateKey
	credential.PublicKey = publicKey

	client := ucloudsdk.NewClient(&cfg, &credential)
	return client, nil
}
