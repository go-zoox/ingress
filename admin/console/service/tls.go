package service

import (
	"crypto/tls"
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

// TLSCertCheckItem is one step in a certificate inspection report.
type TLSCertCheckItem struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Level   string `json:"level"` // ok, warn, fail
	Message string `json:"message"`
}

// TLSCertCheckResult is the outcome of inspecting one configured certificate.
type TLSCertCheckResult struct {
	Domain         string             `json:"domain"`
	Certificate    string             `json:"certificate"`
	CertificateKey string             `json:"certificate_key"`
	OK             bool               `json:"ok"`
	Status         string             `json:"status"`
	Issuer         string             `json:"issuer"`
	Subject        string             `json:"subject"`
	ExpiresAt      string             `json:"expires_at"`
	DaysRemaining  int                `json:"days_remaining"`
	DNSNames       []string           `json:"dns_names"`
	Checks         []TLSCertCheckItem `json:"checks"`
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

func (t *TLS) Inspect(domain string) (*TLSCertCheckResult, error) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return nil, fmt.Errorf("domain is required")
	}
	icfg, err := t.ingress.LoadConfig()
	if err != nil {
		return nil, err
	}
	var sslDomain, certPath, keyPath string
	for _, ssl := range icfg.HTTPS.SSL {
		if ssl.Domain == domain {
			sslDomain = ssl.Domain
			certPath = ssl.Cert.Certificate
			keyPath = ssl.Cert.CertificateKey
			break
		}
	}
	if sslDomain == "" {
		return nil, fmt.Errorf("certificate for domain %q not found", domain)
	}

	result := &TLSCertCheckResult{
		Domain:         sslDomain,
		Certificate:    certPath,
		CertificateKey: keyPath,
		Status:         "missing",
		OK:             true,
		Checks:         []TLSCertCheckItem{},
	}

	addCheck := func(id, label, level, message string) {
		result.Checks = append(result.Checks, TLSCertCheckItem{
			ID:      id,
			Label:   label,
			Level:   level,
			Message: message,
		})
		if level == "fail" {
			result.OK = false
		}
	}

	if strings.TrimSpace(certPath) == "" {
		addCheck("cert_path", "证书路径", "fail", "ingress.yaml 未配置 certificate")
		return result, nil
	}
	addCheck("cert_path", "证书路径", "ok", certPath)

	if _, err := os.Stat(certPath); err != nil {
		addCheck("cert_file", "证书文件", "fail", fmt.Sprintf("无法读取: %v", err))
		return result, nil
	}
	addCheck("cert_file", "证书文件", "ok", "文件存在且可读")

	if strings.TrimSpace(keyPath) == "" {
		addCheck("key_path", "私钥路径", "fail", "ingress.yaml 未配置 certificate_key")
		return result, nil
	}
	addCheck("key_path", "私钥路径", "ok", keyPath)

	if _, err := os.Stat(keyPath); err != nil {
		addCheck("key_file", "私钥文件", "fail", fmt.Sprintf("无法读取: %v", err))
		return result, nil
	}
	addCheck("key_file", "私钥文件", "ok", "文件存在且可读")

	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		addCheck("cert_parse", "证书解析", "fail", err.Error())
		return result, nil
	}
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		addCheck("key_parse", "私钥解析", "fail", err.Error())
		return result, nil
	}

	cert, err := parseX509CertPEM(certPEM)
	if err != nil {
		addCheck("cert_parse", "证书解析", "fail", err.Error())
		return result, nil
	}
	addCheck("cert_parse", "证书解析", "ok", "PEM / X.509 格式有效")

	if _, err := tls.X509KeyPair(certPEM, keyPEM); err != nil {
		addCheck("key_pair", "证书与私钥匹配", "fail", err.Error())
		return result, nil
	}
	addCheck("key_pair", "证书与私钥匹配", "ok", "公钥与私钥配对正确")

	result.Subject = cert.Subject.String()
	if cn := cert.Subject.CommonName; cn != "" {
		result.Subject = cn
	}
	result.Issuer = cert.Issuer.String()
	if cn := cert.Issuer.CommonName; cn != "" {
		result.Issuer = cn
	}
	result.ExpiresAt = cert.NotAfter.Format("2006-01-02")
	result.DaysRemaining = int(time.Until(cert.NotAfter).Hours() / 24)
	result.DNSNames = cert.DNSNames

	if certMatchesDomain(cert, sslDomain) {
		addCheck("domain_match", "域名匹配", "ok", fmt.Sprintf("证书覆盖配置域名 %s", sslDomain))
	} else {
		addCheck("domain_match", "域名匹配", "warn", fmt.Sprintf("证书 SAN/CN 未覆盖 %s", sslDomain))
	}

	switch {
	case result.DaysRemaining < 0:
		addCheck("expiry", "有效期", "fail", fmt.Sprintf("已于 %s 过期", result.ExpiresAt))
		result.Status = "expired"
		result.OK = false
	case result.DaysRemaining <= 30:
		addCheck("expiry", "有效期", "warn", fmt.Sprintf("剩余 %d 天（%s 到期）", result.DaysRemaining, result.ExpiresAt))
		if result.OK {
			result.Status = "warn"
		}
	default:
		addCheck("expiry", "有效期", "ok", fmt.Sprintf("剩余 %d 天（%s 到期）", result.DaysRemaining, result.ExpiresAt))
		if result.OK {
			result.Status = "ok"
		}
	}

	if !result.OK && result.Status != "expired" {
		result.Status = "missing"
	}

	return result, nil
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
	return parseX509CertPEM(data)
}

func parseX509CertPEM(data []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("no pem block")
	}
	return x509.ParseCertificate(block.Bytes)
}

func certMatchesDomain(cert *x509.Certificate, domain string) bool {
	if cert == nil || strings.TrimSpace(domain) == "" {
		return false
	}
	domain = strings.ToLower(strings.TrimSpace(domain))
	if cn := strings.ToLower(cert.Subject.CommonName); cn != "" && (cn == domain || wildcardMatch(cn, domain)) {
		return true
	}
	for _, name := range cert.DNSNames {
		n := strings.ToLower(strings.TrimSpace(name))
		if n == domain || wildcardMatch(n, domain) {
			return true
		}
	}
	return false
}

func wildcardMatch(pattern, host string) bool {
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	host = strings.ToLower(strings.TrimSpace(host))
	if !strings.HasPrefix(pattern, "*.") {
		return pattern == host
	}
	suffix := pattern[1:]
	return strings.HasSuffix(host, suffix) && len(host) > len(suffix)
}
