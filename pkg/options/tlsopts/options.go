package tlsopts

import (
	"crypto/tls"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/setup"
	"github.com/spf13/pflag"
)

type Options struct {
	EnableHTTP2          bool
}

func From(set flagutils.OptionSet) *Options {
	return flagutils.GetFrom[*Options](set)
}

var _ flagutils.Options = (*Options)(nil)

func New() *Options {
	return &Options{}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.EnableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
}

func (o *Options) TlsOpts() []func(*tls.Config) {
	if o!=nil && o.EnableHTTP2 {
		return []func(*tls.Config){disableHTTP2}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// if the enable-http2 flag is false (the default), http/2 should be disabled
// due to its vulnerabilities. More specifically, disabling http/2 will
// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
// Rapid Reset CVEs. For more information see:
// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
// - https://github.com/advisories/GHSA-4374-p667-p6c8
var disableHTTP2 = func(c *tls.Config) {
	setup.SetupLog.Info("disabling http/2")
	c.NextProtos = []string{"http/1.1"}
}
