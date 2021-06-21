package cert

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"
)

func CreateCertificate(pubkey crypto.PublicKey, privkey crypto.PrivateKey) (der []byte, err error) {
	keyUsage := x509.KeyUsageDigitalSignature
	notBefore := time.Now()
	notAfter := notBefore.Add(24 * time.Hour * 7)
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)

	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		BasicConstraintsValid: true,
		DNSNames:              []string{"tete"},
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:                  true,
		KeyUsage:              keyUsage,
		NotAfter:              notAfter,
		NotBefore:             notBefore,
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"tete"}},
	}

	return x509.CreateCertificate(
		rand.Reader,
		&template,
		&template,
		pubkey,
		privkey,
	)
}

func GenerateKey() (crypto.PublicKey, crypto.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}
