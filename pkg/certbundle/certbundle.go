package certbundle

import (
	"bytes"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/pavlo-v-chernykh/keystore-go"
	log "github.com/sirupsen/logrus"
)

type Bundle struct {
	certs     []*x509.Certificate
	password  string
	changedAt time.Time
}

func New(password string) *Bundle {
	return &Bundle{
		certs:    make([]*x509.Certificate, 0),
		password: password,
	}
}

func decode(data []byte) ([]byte, *x509.Certificate, error) {
	block, rest := pem.Decode(data)
	if block != nil {
		cert, err := x509.ParseCertificate(block.Bytes)
		return rest, cert, err
	}

	if len(rest) > 0 {
		cert, err := x509.ParseCertificate(rest)
		return nil, cert, err
	}

	return nil, nil, nil
}

// Read PEM blocks or DER certificate from a reader until there are none left. Consumes all the data from the reader.
func (bundle *Bundle) ReadAll(r io.Reader) error {
	buf := &bytes.Buffer{}
	_, err := io.Copy(buf, r)
	if err != nil {
		return err
	}

	data := buf.Bytes()
	certs := make([]*x509.Certificate, 0)
	for {
		var cert *x509.Certificate
		data, cert, err = decode(data)
		if err != nil {
			return err
		}
		if cert == nil {
			break
		}
		log.Debugf("Importing %s", cert.Subject.String())
		certs = append(certs, cert)
	}

	bundle.certs = append(bundle.certs, certs...)
	bundle.changedAt = time.Now()

	return nil
}

func (bundle *Bundle) KeyStore() keystore.KeyStore {
	gentime := time.Now()
	ks := keystore.KeyStore{}
	for i, cert := range bundle.certs {
		name := fmt.Sprintf("%04d_%s", i, certificateAlias(cert))
		entry := &keystore.TrustedCertificateEntry{
			Entry: keystore.Entry{
				CreationDate: gentime,
			},
			Certificate: keystore.Certificate{
				Type:    "X509",
				Content: cert.Raw,
			},
		}
		ks[name] = entry
	}
	return ks
}

func (bundle *Bundle) WriteJKS(w io.Writer) error {
	return keystore.Encode(w, bundle.KeyStore(), []byte(bundle.password))
}

func (bundle *Bundle) WritePEM(w io.Writer) error {
	for _, cert := range bundle.certs {
		err := pem.Encode(w, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.Raw,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (bundle *Bundle) Certificates() []*x509.Certificate {
	result := make([]*x509.Certificate, len(bundle.certs))
	for i, ptr := range bundle.certs {
		cert := *ptr
		result[i] = &cert
	}
	return result
}

func (bundle *Bundle) Len() int {
	return len(bundle.certs)
}

func (bundle *Bundle) ChangedAt() time.Time {
	return bundle.changedAt
}

// Generate a keytool compatible alias for a certificate.
// Converts to lowercase and strips away non-alphanumeric characters.
// In case no CN is defined, this function generates a name based on the signature data.
func certificateAlias(cert *x509.Certificate) string {
	replace := func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}
	name := strings.ToLower(cert.Subject.CommonName)
	if len(name) == 0 {
		name = "anon_" + hex.EncodeToString(cert.Signature)[:32]
	}
	name = strings.Map(replace, name)
	return name
}
