package devtls

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os/exec"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

var userAndHostname string

func init() {
	u, _ := user.Current()
	if u != nil {
		userAndHostname = u.Username + "@"
	}
	out, _ := exec.Command("hostname").Output()
	userAndHostname += strings.TrimSpace(string(out))
}

func LoadCert(PEMBlock []byte) (*x509.Certificate, error) {
	DERBlock, _ := pem.Decode(PEMBlock)
	if DERBlock == nil || DERBlock.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("failed to read the CA certificate: unexpected content")
	}
	return x509.ParseCertificate(DERBlock.Bytes)
}

func LoadKey(PEMBlock []byte) (crypto.PrivateKey, error) {
	DERBlock, _ := pem.Decode(PEMBlock)
	if DERBlock == nil || DERBlock.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("failed to read the CA Private Key: unexpected content")
	}
	return x509.ParsePKCS8PrivateKey(DERBlock.Bytes)
}

// Config devtls config
type Config struct {
	Cert *x509.Certificate
	Key  crypto.PrivateKey
}

// CertificateGenerator generates new certs from a config and a ClientHelloInfo
type CertificateGenerator struct {
	sync.Mutex
	config *Config
	cache  map[string]*tls.Certificate
}

// NewCertificateGenerator new CertificateGenerator with config
func NewCertificateGenerator(config *Config) *CertificateGenerator {
	return &CertificateGenerator{
		config: config,
		cache:  make(map[string]*tls.Certificate),
	}
}

// GetCertificate satifies the tls.GetCertificate func
func (s *CertificateGenerator) GetCertificate(h *tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.Lock()
	defer s.Unlock()
	if c, ok := s.cache[h.ServerName]; ok {
		return c, nil
	}

	c, err := s.NewCertificate(h)
	if err != nil {
		return nil, err
	}

	s.cache[h.ServerName] = c

	return c, nil
}

// NewCertificate generates a new certificate for a tls.ClientHelloInfo
func (s *CertificateGenerator) NewCertificate(h *tls.ClientHelloInfo) (*tls.Certificate, error) {
	logrus.Infof("Making cert for %s", h.ServerName)

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate certificate key: %v", err)
	}
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}
	tpl := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"mkcert development certificate"},
		},

		NotAfter:  time.Now().AddDate(10, 0, 0),
		NotBefore: time.Now(),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{h.ServerName},
	}
	pub := priv.PublicKey
	cert, err := x509.CreateCertificate(rand.Reader, tpl, s.config.Cert, &pub, s.config.Key)

	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("failed to encode certificate key: %v", err)
	}
	keyPem := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})
	certPem := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert})

	c, err := tls.X509KeyPair(certPem, keyPem)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func NewListener(inner net.Listener, config *Config) net.Listener {
	s := NewCertificateGenerator(config)
	return tls.NewListener(inner, &tls.Config{
		GetCertificate: s.GetCertificate,
	})
}

func ListenAndServeTLS(address string, certBytes, keyBytes []byte, mux http.Handler) error {

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer l.Close()

	cert, err := LoadCert(certBytes)
	if err != nil {
		return err
	}
	key, err := LoadKey(keyBytes)
	if err != nil {
		return err
	}

	l = NewListener(l, &Config{
		Cert: cert,
		Key:  key,
	})

	defer l.Close()
	logrus.Infof("listening on %s", address)
	return http.Serve(l, mux)
}
