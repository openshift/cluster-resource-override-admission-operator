package tlsprofile

import (
	"context"
	"crypto/tls"
	"fmt"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	tlspkg "github.com/openshift/controller-runtime-common/pkg/tls"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var APIServerGVR = schema.GroupVersionResource{
	Group:    "config.openshift.io",
	Version:  "v1",
	Resource: "apiservers",
}

type Args struct {
	MinVersion   string
	CipherSuites string
}

func (a Args) Hash() string {
	if a.MinVersion == "" && a.CipherSuites == "" {
		return ""
	}
	return a.MinVersion + "|" + a.CipherSuites
}

func Fetch(ctx context.Context, dynClient dynamic.Interface) (Args, error) {
	obj, err := dynClient.Resource(APIServerGVR).Get(ctx, "cluster", metav1.GetOptions{})
	if err != nil {
		return Args{}, fmt.Errorf("fetching cluster APIServer config: %w", err)
	}

	apiServer := &configv1.APIServer{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, apiServer); err != nil {
		return Args{}, fmt.Errorf("converting APIServer config: %w", err)
	}

	profile, err := tlspkg.GetTLSProfileSpec(apiServer.Spec.TLSSecurityProfile)
	if err != nil {
		return Args{}, fmt.Errorf("extracting TLS profile: %w", err)
	}

	tlsConfigFn, _ := tlspkg.NewTLSConfigFromProfile(profile)
	cfg := &tls.Config{}
	tlsConfigFn(cfg)

	return ArgsFromTLSConfig(cfg), nil
}

func ArgsFromTLSConfig(cfg *tls.Config) Args {
	var args Args
	if cfg.MinVersion != 0 {
		args.MinVersion = versionName(cfg.MinVersion)
	}

	if len(cfg.CipherSuites) > 0 {
		names := make([]string, 0, len(cfg.CipherSuites))
		for _, id := range cfg.CipherSuites {
			names = append(names, tls.CipherSuiteName(id))
		}
		args.CipherSuites = strings.Join(names, ",")
	}

	return args
}

func versionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "VersionTLS10"
	case tls.VersionTLS11:
		return "VersionTLS11"
	case tls.VersionTLS12:
		return "VersionTLS12"
	case tls.VersionTLS13:
		return "VersionTLS13"
	default:
		return fmt.Sprintf("0x%04X", version)
	}
}
