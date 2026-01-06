package config

import (
	"slices"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/spf13/pflag"
)

type Rules interface {
	flagutils.Options
	flagutils.OptionSetProvider

	// RequireIdentity calls this method on all rules
	// to enable cluster identities.
	RequireIdentity()

	Add(r ...Rule) Rules
	GetConfig(*ConfigOptions) (*Config, error)
	PersonalizedWith(o *Personalization) Rules
	Rules(yield func(r Rule) bool)
}

////////////////////////////////////////////////////////////////////////////////

type _rules struct {
	options flagutils.DefaultOptionSet
	rules   []Rule
	orig    *_rules
}

var _ Rule = Rules(nil)

func NewRules(r ...Rule) Rules {
	return &_rules{rules: slices.Clone(r)}
}

func (r *_rules) RequireIdentity() {
	for _, e := range r.rules {
		if i, ok := e.(IdentityProvider); ok {
			i.RequireIdentity()
		}
	}
}

func (r *_rules) Add(e ...Rule) Rules {
	flagutils.AddOptionally(&r.options, e...)
	r.rules = append(r.rules, e...)
	return r
}

func (r *_rules) GetConfig(opts *ConfigOptions) (*Config, error) {
	if opts == nil {
		opts = &ConfigOptions{}
	}
	for _, rule := range r.rules {
		cfg, err := rule.GetConfig(opts)
		if err != nil || cfg != nil {
			return cfg, err
		}
	}
	return nil, nil
}

func (r *_rules) Rules(yield func(r Rule) bool) {
	for _, rule := range r.rules {
		if !yield(rule) {
			return
		}
	}
}

func (r *_rules) PersonalizedWith(o *Personalization) Rules {
	n := &_rules{
		orig: r.orig,
	}
	for _, e := range general.Optional(r.orig, r).rules {
		n.Add(PersonalizeRule(e, o))
	}
	return n
}

////////////////////////////////////////////////////////////////////////////////

func (r *_rules) AsOptionSet() flagutils.OptionSet {
	return r.options
}

func (r *_rules) AddFlags(fs *pflag.FlagSet) {
	for _, e := range r.rules {
		if o, ok := e.(flagutils.Options); ok {
			o.AddFlags(fs)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func PersonalizeRule(r Rule, o *Personalization) Rule {
	if o.Name == "" {
		return r
	}
	if p, ok := r.(Personalizable); ok {
		return p.PersonalizedWith(o)
	} else {
		return r
	}
}

func DefaultRules() Rules {
	return NewRules(
		NewIdOption(""),
		NewContextOption(""),
		NewKubeconfigOption("", "").WithInClusterMode(),
		NewEnvironmentVariable(),
		NewHomeDirectory(),
		NewInClusterConfig(),
	)
}

func DedicatedConfigRules(name, desc string) Rules {
	return NewRules(
		NewIdOption(""),
		NewContextOption(""),
		NewKubeconfigOption("", "").WithInClusterMode(),
	).PersonalizedWith(&Personalization{
		Name:        name,
		Description: desc,
	})
}
