package service

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"time"
)

// TLSCertRow is one HTTPS certificate entry for the admin UI.
type TLSCertRow struct {
	Domain         string `json:"domain"`
	Certificate    string `json:"certificate"`
	CertificateKey string `json:"certificate_key"`
	Issuer         string `json:"issuer"`
	ExpiresAt      string `json:"expires_at"`
	DaysRemaining  int    `json:"days_remaining"`
	Status         string `json:"status"`
}

// TLS lists and inspects certificates from ingress config.
type TLS struct {
	ingress *Ingress
}

func NewTLS(ingress *Ingress) *TLS {
	return &TLS{ingress: ingress}
}

func (t *TLS) List() ([]TLSCertRow, error) {
	icfg, err := t.ingress.LoadConfig()
	if err != nil {
		return nil, err
	}
	rows := make([]TLSCertRow, 0, len(icfg.HTTPS.SSL))
	for _, ssl := range icfg.HTTPS.SSL {
		row := TLSCertRow{
			Domain:         ssl.Domain,
			Certificate:    ssl.Cert.Certificate,
			CertificateKey: ssl.Cert.CertificateKey,
		}
		fillCertMeta(&row)
		rows = append(rows, row)
	}
	return rows, nil
}

func fillCertMeta(row *TLSCertRow) {
	if row == nil {
		return
	}
	if strings.TrimSpace(row.Certificate) == "" {
		row.Status = "missing"
		return
	}
	if _, err := os.Stat(row.Certificate); err != nil {
		row.Status = "missing"
		return
	}
	cert, err := loadX509(row.Certificate)
	if err != nil {
		row.Status = "missing"
		row.Issuer = "unreadable"
		return
	}
	row.Issuer = cert.Issuer.String()
	if cn := cert.Issuer.CommonName; cn != "" {
		row.Issuer = cn
	}
	row.ExpiresAt = cert.NotAfter.Format("2006-01-02")
	days := int(time.Until(cert.NotAfter).Hours() / 24)
	row.DaysRemaining = days
	switch {
	case days < 0:
		row.Status = "expired"
	case days <= 30:
		row.Status = "warn"
	default:
		row.Status = "ok"
	}
}

func loadX509(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no pem block in %s", path)
	}
	return x509.ParseCertificate(block.Bytes)
}
