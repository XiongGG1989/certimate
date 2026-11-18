package dtos

type DeploymentUCloudUCDNListDomainsReq struct {
	AccessId                string   `json:"accessId"`
	Endpoint                string   `json:"endpoint,omitempty"`
	FilterByCertificate     bool     `json:"filterByCertificate,omitempty"`
	CertificateId           string   `json:"certificateId,omitempty"`
	CertificateWorkflowId   string   `json:"certificateWorkflowId,omitempty"`
	CertificateOutputNodeId string   `json:"certificateOutputNodeId,omitempty"`
	SubjectAltNames         []string `json:"subjectAltNames,omitempty"`
}

type DeploymentUCloudUCDNListDomainsResp struct {
	Domains []string `json:"domains"`
}
