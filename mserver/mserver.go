package mserver

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"net/http"
	"os/exec"
	"os/user"
	"strings"
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

type server struct {
	cert  *x509.Certificate
	key   crypto.PrivateKey
	cache map[string]*tls.Certificate
}

func (s *server) LoadKey(path string) error {
	PEMBlock, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	DERBlock, _ := pem.Decode(PEMBlock)
	if DERBlock == nil || DERBlock.Type != "PRIVATE KEY" {
		return fmt.Errorf("failed to read the CA key: unexpected content")
	}
	s.key, err = x509.ParsePKCS8PrivateKey(DERBlock.Bytes)
	return err
}

func (s *server) LoadCert(path string) error {
	PEMBlock, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	DERBlock, _ := pem.Decode(PEMBlock)
	if DERBlock == nil || DERBlock.Type != "CERTIFICATE" {
		return fmt.Errorf("failed to read the CA certificate: unexpected content")
	}
	s.cert, err = x509.ParseCertificate(DERBlock.Bytes)
	return err
}

func ListenAndServeTLS(address, certPath, keyPath string, mux http.Handler) error {
	s := &server{}

	s.LoadCert(certPath)
	s.LoadKey(keyPath)
	s.cache = make(map[string]*tls.Certificate)

	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	defer l.Close()

	l = tls.NewListener(l, &tls.Config{
		GetCertificate: s.GetCertificate,
	})
	defer l.Close()
	logrus.Infof("listening on %s", address)
	return http.Serve(l, mux)
}

func (s *server) NewCertificate(serverName string) (*tls.Certificate, error) {
	logrus.Infof("Making cert for %s", serverName)

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
			Organization:       []string{"mkcert development certificate"},
			OrganizationalUnit: []string{userAndHostname},
		},

		NotAfter:  time.Now().AddDate(10, 0, 0),
		NotBefore: time.Now(),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{serverName},
	}
	pub := priv.PublicKey
	cert, err := x509.CreateCertificate(rand.Reader, tpl, s.cert, &pub, s.key)

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

func (s *server) GetCertificate(h *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if c, ok := s.cache[h.ServerName]; ok {
		return c, nil
	}

	c, err := s.NewCertificate(h.ServerName)
	if err != nil {
		return nil, err
	}

	s.cache[h.ServerName] = c

	return c, nil
}
