package tencentcloudsslupdate

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	tcsslmodel "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"

	tcssl "github.com/certimate-go/certimate/pkg/sdk3rd-trimmed/github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/ssl/v20191205"

	"github.com/certimate-go/certimate/pkg/core"
	cmgrimpl "github.com/certimate-go/certimate/pkg/core/certmgr/providers/tencentcloud-ssl"
	xwait "github.com/certimate-go/certimate/pkg/utils/wait"
)

type (
	Provider     = core.Deployer
	DeployResult = core.DeployerDeployResult
)

var (
	// UpdateCertificateInstance 支持的资源类型。
	// REF: https://cloud.tencent.com/document/product/400/91649
	updateCertificateResourceProducts = []string{
		"apigateway", "cdn", "clb", "cos", "ddos", "gaap", "live", "mqtt",
		"scf", "tcb", "tdmq", "teo", "tke", "tse", "vod", "waf",
	}
	// UploadUpdateCertificateInstance 当前仅支持 CLB。
	// REF: https://cloud.tencent.com/document/product/400/119791
	uploadUpdateCertificateResourceProducts = []string{"clb"}
	// 这些资源类型调用更新接口时必须同时指定地域。
	resourceProductsRequireRegion = []string{
		"apigateway", "clb", "cos", "mqtt", "scf", "tcb", "tdmq", "tke", "tse", "waf",
	}
)

type DeployerConfig struct {
	// 腾讯云 SecretId。
	SecretId string `json:"secretId"`
	// 腾讯云 SecretKey。
	SecretKey string `json:"secretKey"`
	// 腾讯云项目 ID。
	ProjectId int64 `json:"projectId,omitempty"`
	// 腾讯云接口端点。
	Endpoint string `json:"endpoint,omitempty"`
	// 原证书 ID。
	CertificateId string `json:"certificateId"`
	// 云产品地域数组。
	ResourceRegions []string `json:"resourceRegions,omitempty"`
	// 云产品类型数组。
	ResourceProducts []string `json:"resourceProducts"`
	// 是否替换原有证书（即保持原证书 ID 不变）。
	IsReplaced bool `json:"isReplaced,omitempty"`
	// 是否直接通过一键更新接口上传新证书内容。
	UseCertificateContent bool `json:"useCertificateContent,omitempty"`
}

type Deployer struct {
	config     *DeployerConfig
	logger     *slog.Logger
	sdkClient  *tcssl.Client
	sdkCertmgr core.Certmgr
}

var _ Provider = (*Deployer)(nil)

func NewDeployer(config *DeployerConfig) (*Deployer, error) {
	if config == nil {
		return nil, fmt.Errorf("the configuration of the deployer provider is nil")
	}

	client, err := createSDKClient(config.SecretId, config.SecretKey, config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("could not create client: %w", err)
	}

	pcertmgr, err := cmgrimpl.NewCertmgr(&cmgrimpl.CertmgrConfig{
		SecretId:  config.SecretId,
		SecretKey: config.SecretKey,
		ProjectId: config.ProjectId,
		Endpoint:  config.Endpoint,
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
	if d.config.CertificateId == "" {
		return nil, fmt.Errorf("config `certificateId` is required")
	}
	if len(d.config.ResourceProducts) == 0 {
		return nil, fmt.Errorf("config `resourceProducts` is required")
	}
	if err := validateResourceProducts(d.config.ResourceProducts, d.config.ResourceRegions, d.config.IsReplaced); err != nil {
		return nil, err
	}

	if d.config.IsReplaced {
		if err := d.executeUploadUpdateCertificateInstance(ctx, certPEM, privkeyPEM); err != nil {
			return nil, err
		}
	} else {
		newCertificateId, err := d.executeUpdateCertificateInstance(ctx, certPEM, privkeyPEM)
		if err != nil {
			return nil, err
		}
		if d.config.UseCertificateContent {
			return &DeployResult{ExtendedData: map[string]any{
				"providerConfig.certificateId": newCertificateId,
			}}, nil
		}
	}

	return &DeployResult{}, nil
}

func (d *Deployer) executeUpdateCertificateInstance(ctx context.Context, certPEM, privkeyPEM string) (string, error) {
	var certificateId string
	if !d.config.UseCertificateContent {
		// 兼容原有部署选项：先上传证书，再使用新证书 ID 更新旧证书资源。
		upres, err := d.sdkCertmgr.Upload(ctx, certPEM, privkeyPEM)
		if err != nil {
			return "", fmt.Errorf("failed to upload certificate file: %w", err)
		} else {
			d.logger.Info("ssl certificate uploaded", slog.Any("result", upres))
			certificateId = upres.CertId
		}
	}

	// 一键更新新旧证书资源
	// REF: https://cloud.tencent.com/document/product/400/91649
	var deployRecordId string
	if _, err := xwait.UntilWithContext(ctx, func(_ context.Context, _ int) (bool, error) {
		updateCertificateInstanceReq := d.newUpdateCertificateInstanceRequest(certPEM, privkeyPEM, certificateId)
		updateCertificateInstanceResp, err := d.sdkClient.UpdateCertificateInstanceWithContext(ctx, updateCertificateInstanceReq)
		d.logger.Debug("sdk request 'ssl.UpdateCertificateInstance'", slog.String("oldCertificateId", d.config.CertificateId), slog.Any("resourceProducts", d.config.ResourceProducts), slog.Any("response", updateCertificateInstanceResp))
		if err != nil {
			return false, fmt.Errorf("failed to execute sdk request 'ssl.UpdateCertificateInstance': %w", err)
		}

		if updateCertificateInstanceResp == nil || updateCertificateInstanceResp.Response == nil {
			return false, fmt.Errorf("unexpected deployment job status")
		}

		recordId, ready, err := resolveDeployRecordId(updateCertificateInstanceResp.Response.DeployStatus, updateCertificateInstanceResp.Response.DeployRecordId)
		if err != nil {
			return false, err
		} else if ready {
			deployRecordId = fmt.Sprintf("%d", recordId)
			return true, nil
		}

		return false, nil
	}, 10*time.Second); err != nil {
		return "", err
	}

	// 查询证书云资源更新记录详情，等待任务状态变更
	// REF: https://cloud.tencent.com/document/api/400/91652
	var newCertificateId string
	if _, err := xwait.UntilWithContext(ctx, func(_ context.Context, _ int) (bool, error) {
		describeHostUpdateRecordDetailReq := tcssl.NewDescribeHostUpdateRecordDetailRequest()
		describeHostUpdateRecordDetailReq.DeployRecordId = common.StringPtr(deployRecordId)
		describeHostUpdateRecordDetailReq.Limit = common.StringPtr("200")
		describeHostUpdateRecordDetailResp, err := d.sdkClient.DescribeHostUpdateRecordDetailWithContext(ctx, describeHostUpdateRecordDetailReq)
		d.logger.Debug("sdk request 'ssl.DescribeHostUpdateRecordDetail'", slog.Any("request", describeHostUpdateRecordDetailReq), slog.Any("response", describeHostUpdateRecordDetailResp))
		if err != nil {
			return false, fmt.Errorf("failed to execute sdk request 'ssl.DescribeHostUpdateRecordDetail': %w", err)
		}

		var pendingCount, runningCount, succeededCount, failedCount, totalCount int64
		if describeHostUpdateRecordDetailResp == nil || describeHostUpdateRecordDetailResp.Response == nil || describeHostUpdateRecordDetailResp.Response.TotalCount == nil {
			return false, fmt.Errorf("unexpected deployment job status")
		} else {
			newCertificateId = resolveNewCertificateId(describeHostUpdateRecordDetailResp.Response.RecordDetailList)
			pendingCount = lo.FromPtr(describeHostUpdateRecordDetailResp.Response.PendingTotalCount)
			runningCount = lo.FromPtr(describeHostUpdateRecordDetailResp.Response.RunningTotalCount)
			succeededCount = lo.FromPtr(describeHostUpdateRecordDetailResp.Response.SuccessTotalCount)
			failedCount = lo.FromPtr(describeHostUpdateRecordDetailResp.Response.FailedTotalCount)
			totalCount = lo.FromPtr(describeHostUpdateRecordDetailResp.Response.TotalCount)

			if succeededCount+failedCount == totalCount {
				if failedCount > 0 {
					return false, fmt.Errorf("unexpected deployment deployment job status (succeeded: %d, failed: %d, total: %d)", succeededCount, failedCount, totalCount)
				}
				if d.config.UseCertificateContent && newCertificateId == "" {
					return false, fmt.Errorf("deployment completed but the new certificate ID was not returned")
				}
				return true, nil
			}
		}

		d.logger.Info(fmt.Sprintf("waiting for deployment job completion (pending: %d, running: %d, succeeded: %d, failed: %d, total: %d) ...", pendingCount, runningCount, succeededCount, failedCount, totalCount))
		return false, nil
	}, 10*time.Second); err != nil {
		return "", err
	}

	if newCertificateId != "" {
		return newCertificateId, nil
	}
	return certificateId, nil
}

func (d *Deployer) newUpdateCertificateInstanceRequest(certPEM, privkeyPEM, certificateId string) *tcssl.UpdateCertificateInstanceRequest {
	request := tcssl.NewUpdateCertificateInstanceRequest()
	request.OldCertificateId = common.StringPtr(d.config.CertificateId)
	request.ResourceTypes = common.StringPtrs(d.config.ResourceProducts)
	request.ResourceTypesRegions = wrapResourceProductRegions(d.config.ResourceProducts, d.config.ResourceRegions)

	if d.config.UseCertificateContent {
		request.CertificatePublicKey = common.StringPtr(certPEM)
		request.CertificatePrivateKey = common.StringPtr(privkeyPEM)
		request.Repeatable = common.BoolPtr(true)
		request.AllowDownload = common.BoolPtr(true)
		request.ProjectId = lo.EmptyableToPtr(uint64(d.config.ProjectId))
	} else {
		request.CertificateId = common.StringPtr(certificateId)
	}

	return request
}

func (d *Deployer) executeUploadUpdateCertificateInstance(ctx context.Context, certPEM, privkeyPEM string) error {
	// 更新证书内容并更新关联的云资源
	// REF: https://cloud.tencent.com/document/product/400/119791
	var deployRecordId int64
	if _, err := xwait.UntilWithContext(ctx, func(_ context.Context, _ int) (bool, error) {
		uploadUpdateCertificateInstanceReq := tcssl.NewUploadUpdateCertificateInstanceRequest()
		uploadUpdateCertificateInstanceReq.OldCertificateId = common.StringPtr(d.config.CertificateId)
		uploadUpdateCertificateInstanceReq.CertificatePublicKey = common.StringPtr(certPEM)
		uploadUpdateCertificateInstanceReq.CertificatePrivateKey = common.StringPtr(privkeyPEM)
		uploadUpdateCertificateInstanceReq.ResourceTypes = common.StringPtrs(d.config.ResourceProducts)
		uploadUpdateCertificateInstanceReq.ResourceTypesRegions = wrapResourceProductRegions(d.config.ResourceProducts, d.config.ResourceRegions)
		uploadUpdateCertificateInstanceResp, err := d.sdkClient.UploadUpdateCertificateInstanceWithContext(ctx, uploadUpdateCertificateInstanceReq)
		d.logger.Debug("sdk request 'ssl.UploadUpdateCertificateInstance'", slog.String("oldCertificateId", d.config.CertificateId), slog.Any("resourceProducts", d.config.ResourceProducts), slog.Any("response", uploadUpdateCertificateInstanceResp))
		if err != nil {
			return false, fmt.Errorf("failed to execute sdk request 'ssl.UploadUpdateCertificateInstance': %w", err)
		}

		if uploadUpdateCertificateInstanceResp == nil || uploadUpdateCertificateInstanceResp.Response == nil {
			return false, fmt.Errorf("unexpected deployment job status")
		}

		recordId, ready, err := resolveDeployRecordId(uploadUpdateCertificateInstanceResp.Response.DeployStatus, uploadUpdateCertificateInstanceResp.Response.DeployRecordId)
		if err != nil {
			return false, err
		} else if ready {
			deployRecordId = int64(recordId)
			return true, nil
		}

		return false, nil
	}, 10*time.Second); err != nil {
		return err
	}

	// 查询证书云资源更新记录详情，等待任务状态变更
	// REF: https://cloud.tencent.com/document/product/400/120056
	if _, err := xwait.UntilWithContext(ctx, func(_ context.Context, _ int) (bool, error) {
		describeHostUploadUpdateRecordDetailReq := tcssl.NewDescribeHostUploadUpdateRecordDetailRequest()
		describeHostUploadUpdateRecordDetailReq.DeployRecordId = common.Int64Ptr(deployRecordId)
		describeHostUploadUpdateRecordDetailReq.Limit = common.Int64Ptr(200)
		describeHostUploadUpdateRecordDetailResp, err := d.sdkClient.DescribeHostUploadUpdateRecordDetailWithContext(ctx, describeHostUploadUpdateRecordDetailReq)
		d.logger.Debug("sdk request 'ssl.DescribeHostUploadUpdateRecordDetail'", slog.Any("request", describeHostUploadUpdateRecordDetailReq), slog.Any("response", describeHostUploadUpdateRecordDetailResp))
		if err != nil {
			return false, fmt.Errorf("failed to execute sdk request 'ssl.DescribeHostUploadUpdateRecordDetail': %w", err)
		}

		var runningCount, succeededCount, failedCount, totalCount int64
		if describeHostUploadUpdateRecordDetailResp == nil || describeHostUploadUpdateRecordDetailResp.Response == nil || describeHostUploadUpdateRecordDetailResp.Response.DeployRecordDetail == nil {
			return false, fmt.Errorf("unexpected tencentcloud deployment job status")
		} else {
			for _, record := range describeHostUploadUpdateRecordDetailResp.Response.DeployRecordDetail {
				runningCount += lo.FromPtr(record.RunningTotalCount)
				succeededCount += lo.FromPtr(record.SuccessTotalCount)
				failedCount += lo.FromPtr(record.FailedTotalCount)
				totalCount += lo.FromPtr(record.TotalCount)
			}

			if succeededCount+failedCount == totalCount {
				if failedCount > 0 {
					return false, fmt.Errorf("unexpected deployment deployment job status (succeeded: %d, failed: %d, total: %d)", succeededCount, failedCount, totalCount)
				}
				return true, nil
			}
		}

		d.logger.Info(fmt.Sprintf("waiting for deployment job completion (running: %d, succeeded: %d, failed: %d, total: %d) ...", runningCount, succeededCount, failedCount, totalCount))
		return false, nil
	}, 10*time.Second); err != nil {
		return err
	}

	return nil
}

func createSDKClient(secretId, secretKey, endpoint string) (*tcssl.Client, error) {
	credential := common.NewCredential(secretId, secretKey)

	cpf := profile.NewClientProfile()
	if endpoint != "" {
		cpf.HttpProfile.Endpoint = endpoint
	}

	client, err := tcssl.NewClient(credential, "", cpf)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func wrapResourceProductRegions(resourceProducts, resourceRegions []string) []*tcssl.ResourceTypeRegions {
	if len(resourceProducts) == 0 || len(resourceRegions) == 0 {
		return nil
	}

	temp := make([]*tcssl.ResourceTypeRegions, 0)
	for _, resourceProduct := range resourceProducts {
		if slices.Contains(resourceProductsRequireRegion, resourceProduct) {
			temp = append(temp, &tcssl.ResourceTypeRegions{
				ResourceType: common.StringPtr(resourceProduct),
				Regions:      common.StringPtrs(resourceRegions),
			})
		}
	}

	return temp
}

func validateResourceProducts(resourceProducts, resourceRegions []string, isReplaced bool) error {
	supportedProducts := updateCertificateResourceProducts
	if isReplaced {
		supportedProducts = uploadUpdateCertificateResourceProducts
	}

	for _, resourceProduct := range resourceProducts {
		if !slices.Contains(supportedProducts, resourceProduct) {
			return fmt.Errorf("config `resourceProducts` contains unsupported resource product %q", resourceProduct)
		}
		if slices.Contains(resourceProductsRequireRegion, resourceProduct) && len(resourceRegions) == 0 {
			return fmt.Errorf("config `resourceRegions` is required for resource product %q", resourceProduct)
		}
	}

	return nil
}

func resolveDeployRecordId(deployStatus *int64, deployRecordId *uint64) (uint64, bool, error) {
	if deployStatus == nil || deployRecordId == nil {
		return 0, false, fmt.Errorf("unexpected deployment job status")
	}

	return *deployRecordId, *deployRecordId > 0, nil
}

func resolveNewCertificateId(recordDetails []*tcsslmodel.UpdateRecordDetails) string {
	for _, recordDetailsByType := range recordDetails {
		if recordDetailsByType == nil {
			continue
		}
		for _, recordDetail := range recordDetailsByType.List {
			if recordDetail != nil && lo.FromPtr(recordDetail.CertId) != "" {
				return lo.FromPtr(recordDetail.CertId)
			}
		}
	}

	return ""
}
