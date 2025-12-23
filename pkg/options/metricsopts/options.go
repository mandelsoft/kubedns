package metricsopts

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/options/tlsopts"
	"github.com/mandelsoft/kubedns/pkg/setup"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

type Options struct {
	MetricsAddr                                      string
	MetricsCertPath, MetricsCertName, MetricsCertKey string
	SecureMetrics                                    bool

	tlsOpts *tlsopts.Options
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

var (
	_ flagutils.Options     = (*Options)(nil)
	_ flagutils.Validatable = (*Options)(nil)
)

func New() *Options {
	return &Options{}
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	tls, err := flagutils.ValidatedOptions[*tlsopts.Options](ctx, opts, v)
	if err == nil {
		o.tlsOpts = tls
	}
	return err
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.MetricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	fs.BoolVar(&o.SecureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	fs.StringVar(&o.MetricsCertPath, "metrics-cert-path", "",
		"The directory that contains the metrics server certificate.")
	fs.StringVar(&o.MetricsCertName, "metrics-cert-name", "tls.crt", "The name of the metrics server certificate file.")
	fs.StringVar(&o.MetricsCertKey, "metrics-cert-key", "tls.key", "The name of the metrics server key file.")
}

func (o *Options) GetMetricsServerOpts() metricsserver.Options {
	if o == nil {
		return metricsserver.Options{BindAddress: "0"}
	}
	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   o.MetricsAddr,
		SecureServing: o.SecureMetrics,
		TLSOpts:       o.tlsOpts.TlsOpts(),
	}

	if o.SecureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	// If the certificate is not specified, controller-runtime will automatically
	// generate self-signed certificates for the metrics server. While convenient for development and testing,
	// this setup is not recommended for production.
	//
	// TODO(user): If you enable certManager, uncomment the following lines:
	// - [METRICS-WITH-CERTS] at config/default/kustomization.yaml to generate and use certificates
	// managed by cert-manager for the metrics server.
	// - [PROMETHEUS-WITH-CERTS] at config/prometheus/kustomization.yaml for TLS certification.
	if len(o.MetricsCertPath) > 0 {
		setup.Log.Info("Initializing metrics certificate watcher using provided certificates",
			"metrics-cert-path", o.MetricsCertPath, "metrics-cert-name", o.MetricsCertName, "metrics-cert-key", o.MetricsCertKey)

		metricsServerOptions.CertDir = o.MetricsCertPath
		metricsServerOptions.CertName = o.MetricsCertName
		metricsServerOptions.KeyName = o.MetricsCertKey
	}
	return metricsServerOptions
}
