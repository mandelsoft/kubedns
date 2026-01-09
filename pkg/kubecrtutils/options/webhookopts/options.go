package webhookopts

import (
	"context"
	"crypto/tls"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/tlsopts"
	"github.com/mandelsoft/kubedns/pkg/setup"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type Options struct {
	webhookCertPath, webhookCertName, webhookCertKey string

	tlsOpts       *tlsopts.Options
	webhookServer webhook.Server
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

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.webhookCertPath, "webhook-cert-path", "", "The directory that contains the webhook certificate.")
	fs.StringVar(&o.webhookCertName, "webhook-cert-name", "tls.crt", "The name of the webhook certificate file.")
	fs.StringVar(&o.webhookCertKey, "webhook-cert-key", "tls.key", "The name of the webhook key file.")
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	tls, err := flagutils.ValidatedOptions[*tlsopts.Options](ctx, opts, v)
	if err == nil {
		o.tlsOpts = tls
	}
	return err
}

func (o *Options) GetServer() webhook.Server {
	if o == nil {
		return nil
	}
	if o.webhookServer == nil {
		var webhookTLSOpts []func(*tls.Config)

		webhookTLSOpts = append(webhookTLSOpts, o.tlsOpts.TlsOpts()...)

		// Initial webhook TLS options
		webhookServerOptions := webhook.Options{
			TLSOpts: webhookTLSOpts,
		}

		if len(o.webhookCertPath) > 0 {
			setup.Log.Info("Initializing webhook certificate watcher using provided certificates",
				"webhook-cert-path", o.webhookCertPath, "webhook-cert-name", o.webhookCertName, "webhook-cert-key", o.webhookCertKey)

			webhookServerOptions.CertDir = o.webhookCertPath
			webhookServerOptions.CertName = o.webhookCertName
			webhookServerOptions.KeyName = o.webhookCertKey
		}

		o.webhookServer = webhook.NewServer(webhookServerOptions)
	}

	return o.webhookServer
}
