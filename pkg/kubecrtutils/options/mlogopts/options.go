package mlogopts

import (
	"context"
	"fmt"
	"strings"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/logging"
	"github.com/spf13/pflag"
)

type Options struct {
	auto     bool
	Level    string
	Settings map[string]string
}

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

func New(auto ...bool) *Options {
	return &Options{auto: general.Optional(auto...)}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&o.Level, "log-level", "", "info", "logging level")
	fs.StringToStringVarP(&o.Settings, "log-rule", "", nil, "logging rules")
}

func (o *Options) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	_, err := logging.ParseLevel(o.Level)
	if err != nil {
		return err
	}
	for n, s := range o.Settings {
		_, err := logging.ParseLevel(s)
		if err != nil {
			return fmt.Errorf("%s: %w", n, err)
		}
	}
	if o.auto {
		return o.Configure(logging.DefaultContext())
	}
	return nil
}

func (o *Options) Configure(ctx logging.Context) error {
	l, err := logging.ParseLevel(o.Level)
	if err != nil {
		return err
	}
	ctx.SetDefaultLevel(l)
	for n, s := range o.Settings {
		l, err := logging.ParseLevel(s)
		if err != nil {
			return fmt.Errorf("%s: %w", n, err)
		}
		var c logging.Condition
		if strings.HasSuffix(n, "/*") {
			c = logging.NewRealmPrefix(n[:len(n)-2])
		} else {
			c = logging.NewRealm(n)
		}
		ctx.AddRule(logging.NewConditionRule(l, c))
	}
	return nil
}
