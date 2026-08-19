package media

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/status-im/status-go/internal/httpserver"
	"github.com/status-im/status-go/internal/logutils"
)

var globalMediaCertificate *tls.Certificate = nil
var globalMediaPem string

func generateMediaTLSCert() (*tls.Certificate, string, error) {
	if globalMediaCertificate != nil {
		return globalMediaCertificate, globalMediaPem, nil
	}

	now := time.Now()
	notBefore := now.Add(-365 * 24 * time.Hour * 100)
	notAfter := now.Add(365 * 24 * time.Hour * 100)
	logutils.ZapLogger().Debug("generate media cert",
		logutils.UnixTimeMs("system time", time.Now()),
		logutils.UnixTimeMs("cert notBefore", notBefore),
		logutils.UnixTimeMs("cert notAfter", notAfter),
	)
	finalCert, certPem, err := httpserver.GenerateTLSCert(notBefore, notAfter, []net.IP{}, []string{httpserver.Localhost})
	if err != nil {
		return nil, "", err
	}

	globalMediaCertificate = finalCert
	globalMediaPem = string(certPem)
	return finalCert, globalMediaPem, nil
}

func PublicMediaTLSCert() (string, error) {
	_, pem, err := generateMediaTLSCert()
	if err != nil {
		return "", err
	}

	return pem, nil
}
