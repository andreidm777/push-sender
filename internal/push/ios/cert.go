package ios

import (
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strings"
)

func makeCert(decryptedCert string, privateKeyPassword string) (cert *tls.Certificate, err error) {
	if decryptedCert == "" {
		return nil, errors.New("empty certificate data")
	}

	cert = &tls.Certificate{}
	data := []byte(decryptedCert)

	var block *pem.Block
	var key crypto.PrivateKey
	var leaf *x509.Certificate

	for {
		if block, data = pem.Decode(data); block == nil {
			break
		}

		if block.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, block.Bytes)
		}

		if block.Type == "PRIVATE KEY" || strings.HasSuffix(block.Type, "PRIVATE KEY") {
			if key, err = decryptPrivateKey(block, privateKeyPassword); err != nil {
				return
			}

			cert.PrivateKey = key
		}
	}

	if len(cert.Certificate) == 0 {
		return nil, errors.New("no certificate")
	}

	if cert.PrivateKey == nil {
		return nil, errors.New("no private key")
	}

	if leaf, err = x509.ParseCertificate(cert.Certificate[0]); err != nil {
		return
	}

	cert.Leaf = leaf

	return
}

func decryptPrivateKey(block *pem.Block, password string) (key crypto.PrivateKey, err error) {
	if x509.IsEncryptedPEMBlock(block) {
		var data []byte

		if data, err = x509.DecryptPEMBlock(block, []byte(password)); err != nil {
			return nil, errors.New("failed to decrypt private key")
		}

		return parsePrivateKey(data)
	}

	return parsePrivateKey(block.Bytes)
}

func parsePrivateKey(bytes []byte) (key crypto.PrivateKey, err error) {
	if key, err = x509.ParsePKCS1PrivateKey(bytes); err == nil {
		return
	}

	if key, err = x509.ParsePKCS8PrivateKey(bytes); err == nil {
		return
	}

	return nil, errors.New("failed to parse private key")
}
