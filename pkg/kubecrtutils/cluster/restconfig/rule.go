package restconfig

import (
	"slices"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

type Personalizable interface {
	PersonalizedWith(name string) Rule
}

type Rule interface {
	GetConfig() (*rest.Config, error)
}

type Rules interface {
	flagutils.Options
	flagutils.OptionSetProvider

	Add(r ...Rule) Rules
	GetConfig() (*rest.Config, error)
	PersonalizedWith(name string) Rules
	Rules(yield func(r Rule) bool)
}

////////////////////////////////////////////////////////////////////////////////

type rules struct {
	options flagutils.DefaultOptionSet
	rules   []Rule
	orig    *rules
}

var _ Rule = Rules(nil)

func NewRules(r ...Rule) Rules {
	return &rules{rules: slices.Clone(r)}
}

func (r *rules) Add(e ...Rule) Rules {
	flagutils.AddOptionally(&r.options, e...)
	r.rules = append(r.rules, e...)
	return r
}

func (r *rules) GetConfig() (*rest.Config, error) {
	for _, rule := range r.rules {
		cfg, err := rule.GetConfig()
		if err != nil || cfg != nil {
			return cfg, err
		}
	}
	return nil, nil
}

func (r *rules) Rules(yield func(r Rule) bool) {
	for _, rule := range r.rules {
		if !yield(rule) {
			return
		}
	}
}

func (r *rules) PersonalizedWith(name string) Rules {
	n := &rules{
		orig: r.orig,
	}
	for _, e := range general.Optional(r.orig, r).rules {
		n.Add(PersonalizeRule(e, name))
	}
	return n
}

////////////////////////////////////////////////////////////////////////////////

func (r *rules) AsOptionSet() flagutils.OptionSet {
	return r.options
}

func (r *rules) AddFlags(fs *pflag.FlagSet) {
	for _, e := range r.rules {
		if o, ok := e.(flagutils.Options); ok {
			o.AddFlags(fs)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func PersonalizeRule(r Rule, name string) Rule {
	if name == "" {
		return r
	}
	if p, ok := r.(Personalizable); ok {
		return p.PersonalizedWith(name)
	} else {
		return r
	}
}

func DefaultRules() Rules {
	return NewRules(
		NewKubeconfigOption("", "").WithInClusterMode(),
		NewEnvironmentVariable(),
		NewHomeDirectory(),
		NewInClusterConfig(),
	)
}

func DedicatedConfigRules(name, desc string) Rules {
	return NewRules(
		NewKubeconfigOption("", "").WithInClusterMode().PersonalizedWith(name + "@" + desc),
	)
}
