package dtos

import (
	"github.com/certimate-go/certimate/internal/domain"
)

type CertificateDownloadReq struct {
	CertificateId string                       `json:"-"`
	FileFormat    domain.CertificateFormatType `json:"fileFormat"`
	PfxPassword   string                       `json:"pfxPassword,omitempty"`
	PfxEncoder    string                       `json:"pfxEncoder,omitempty"`
	JksAlias      string                       `json:"jksAlias,omitempty"`
	JksKeypass    string                       `json:"jksKeypass,omitempty"`
	JksStorepass  string                       `json:"jksStorepass,omitempty"`
}

type CertificateDownloadResp struct {
	ZipBytes []byte `json:"zipBytes"`
}

type CertificateRevokeReq struct {
	CertificateId string `json:"-"`
}

type CertificateRevokeResp struct{}

type CertificateUploadReq struct {
	CertificateId string `json:"-"`
	Certificate   string `json:"certificate"`
	PrivateKey    string `json:"privateKey"`
}

type CertificateUploadResp struct {
	CertificateId string `json:"certificateId"`
}
