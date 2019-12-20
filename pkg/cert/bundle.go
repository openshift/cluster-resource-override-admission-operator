package cert

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// Bundle encapsulates
// - PEM encoded serving private key and certificate
// - certificate of the self-signed CA that signed the serving cert.
type Bundle struct {
	Serving
	ServingCertCA []byte
}

type Serving struct {
	ServiceKey  []byte
	ServiceCert []byte
}

func (b *Bundle) Validate() error {
	if len(b.ServingCertCA) == 0 {
		return errors.New("serving service cert CA must be specified")
	}

	if len(b.Serving.ServiceCert) == 0 {
		return errors.New("serving service cert must be specified")
	}

	if len(b.Serving.ServiceKey) == 0 {
		return errors.New("serving service private key must be specified")
	}

	return nil
}

// Hash generates a sha256 hash of the given Bundle object
// The hash is generated from the hash of the serving key, serving cert, serving CA cert.
func (b *Bundle) Hash() string {
	writer := sha256.New()

	writer.Write(b.ServiceKey)
	h1 := writer.Sum(nil)

	writer.Reset()
	writer.Write(b.ServiceCert)
	h2 := writer.Sum(h1)

	writer.Reset()
	writer.Write(b.ServingCertCA)
	h := writer.Sum(h2)

	writer.Reset()
	writer.Write(h)

	return hex.EncodeToString(writer.Sum(nil))
}
