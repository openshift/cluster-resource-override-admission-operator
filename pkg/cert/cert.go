package cert

import (
	"time"

	corev1 "k8s.io/api/core/v1"
)

// GenerateWithLocalhostServing generates self-signed 'localhost' serving cert(s).
func GenerateWithLocalhostServing(notAfter time.Time, organization string) (bundle *Bundle, err error) {
	ca, err := GenerateCA(notAfter, organization)
	if err != nil {
		return
	}

	// Create signed serving cert
	hosts := []string{
		"localhost",
	}

	servingPair, err := CreateSignedServingPair(notAfter, organization, ca, hosts)
	if err != nil {
		return
	}

	serviceCert, serviceKey, err := servingPair.ToPEM()
	if err != nil {
		return
	}

	servingCertCA, _, err := ca.ToPEM()
	if err != nil {
		return
	}

	bundle = &Bundle{
		Serving: Serving{
			ServiceKey:  serviceKey,
			ServiceCert: serviceCert,
		},
		ServingCertCA: servingCertCA,
	}

	return
}

// IsPopulated returns true if the given Secret object contains the serving key and cert.
func IsPopulated(secret *corev1.Secret) bool {
	if secret == nil {
		return false
	}

	if len(secret.Data) == 0 {
		return false
	}

	if len(secret.Data["tls.key"]) == 0 ||
		len(secret.Data["tls.crt"]) == 0 {
		return false
	}

	return true
}
