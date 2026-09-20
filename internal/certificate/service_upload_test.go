package certificate

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	"github.com/pocketbase/dbx"

	"github.com/certimate-go/certimate/internal/domain"
	"github.com/certimate-go/certimate/internal/domain/dtos"
)

type uploadTestCertificateRepository struct {
	record *domain.Certificate
}

func (r *uploadTestCertificateRepository) GetById(_ context.Context, id string) (*domain.Certificate, error) {
	if r.record == nil || r.record.Id != id {
		return nil, domain.ErrRecordNotFound
	}
	return r.record, nil
}

func (r *uploadTestCertificateRepository) Save(_ context.Context, certificate *domain.Certificate) (*domain.Certificate, error) {
	if certificate.Id == "" {
		certificate.Id = "manual-cert-id"
		certificate.CreatedAt = time.Now()
	}
	certificate.UpdatedAt = time.Now()
	r.record = certificate
	return certificate, nil
}

func (r *uploadTestCertificateRepository) DeleteWithExprs(_ context.Context, _ ...dbx.Expression) (int, error) {
	return 0, nil
}

func TestUploadCertificateKeepsIdWhenUpdating(t *testing.T) {
	certPEM, keyPEM := newUploadTestCertificatePair(t, "example.com")
	repo := &uploadTestCertificateRepository{}
	service := NewCertificateService(nil, repo)

	created, err := service.UploadCertificate(context.Background(), &dtos.CertificateUploadReq{
		Certificate: certPEM,
		PrivateKey:  keyPEM,
	})
	if err != nil {
		t.Fatalf("UploadCertificate() create error = %v", err)
	}

	updatedCertPEM, updatedKeyPEM := newUploadTestCertificatePair(t, "updated.example.com")
	updated, err := service.UploadCertificate(context.Background(), &dtos.CertificateUploadReq{
		CertificateId: created.CertificateId,
		Certificate:   updatedCertPEM,
		PrivateKey:    updatedKeyPEM,
	})
	if err != nil {
		t.Fatalf("UploadCertificate() update error = %v", err)
	}
	if updated.CertificateId != created.CertificateId {
		t.Fatalf("updated id = %q, want %q", updated.CertificateId, created.CertificateId)
	}
	if repo.record.SubjectAltNames != "updated.example.com" {
		t.Fatalf("updated subjectAltNames = %q", repo.record.SubjectAltNames)
	}
}

func TestUploadCertificateRejectsMismatchedPrivateKey(t *testing.T) {
	certPEM, _ := newUploadTestCertificatePair(t, "example.com")
	_, otherKeyPEM := newUploadTestCertificatePair(t, "other.example.com")
	service := NewCertificateService(nil, &uploadTestCertificateRepository{})

	if _, err := service.UploadCertificate(context.Background(), &dtos.CertificateUploadReq{
		Certificate: certPEM,
		PrivateKey:  otherKeyPEM,
	}); err == nil {
		t.Fatal("UploadCertificate() error = nil, want mismatched private key error")
	}
}

func TestUploadCertificateRejectsWorkflowCertificateUpdate(t *testing.T) {
	certPEM, keyPEM := newUploadTestCertificatePair(t, "example.com")
	repo := &uploadTestCertificateRepository{record: &domain.Certificate{
		Meta:       domain.Meta{Id: "workflow-cert-id"},
		Source:     domain.CertificateSourceTypeUpload,
		WorkflowId: "workflow-id",
	}}
	service := NewCertificateService(nil, repo)

	if _, err := service.UploadCertificate(context.Background(), &dtos.CertificateUploadReq{
		CertificateId: repo.record.Id,
		Certificate:   certPEM,
		PrivateKey:    keyPEM,
	}); err == nil {
		t.Fatal("UploadCertificate() error = nil, want workflow certificate update error")
	}
}

func newUploadTestCertificatePair(t *testing.T, dnsName string) (string, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: big.NewInt(now.UnixNano()),
		Subject:      pkix.Name{CommonName: dnsName},
		DNSNames:     []string{dnsName},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	return string(certPEM), string(keyPEM)
}
