package restconfig

import (
	"slices"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type RuleOptions struct {
	clientcmd.ConfigOverrides
}

type Personalization struct {
	Name        string
	Description string
}

type Personalizable interface {
	PersonalizedWith(name *Personalization) Rule
}

type Rule interface {
	GetConfig(opts *RuleOptions) (*rest.Config, error)
}

type Rules interface {
	flagutils.Options
	flagutils.OptionSetProvider

	Add(r ...Rule) Rules
	GetConfig(*RuleOptions) (*rest.Config, error)
	PersonalizedWith(o *Personalization) Rules
	Rules(yield func(r Rule) bool)
}

////////////////////////////////////////////////////////////////////////////////

type rules struct {
	options  flagutils.DefaultOptionSet
	ruleopts *RuleOptions
	rules    []Rule
	orig     *rules
}

var _ Rule = Rules(nil)

func NewRules(r ...Rule) Rules {
	return &rules{rules: slices.Clone(r), ruleopts: &RuleOptions{}}
}

func (r *rules) WithOptions(o *RuleOptions) Rules {
	if o == nil {
		o = &RuleOptions{}
	}
	r.ruleopts = o
	return r
}

func (r *rules) Add(e ...Rule) Rules {
	flagutils.AddOptionally(&r.options, e...)
	r.rules = append(r.rules, e...)
	return r
}

func (r *rules) GetConfig(*RuleOptions) (*rest.Config, error) {
	for _, rule := range r.rules {
		cfg, err := rule.GetConfig(r.ruleopts)
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

func (r *rules) PersonalizedWith(o *Personalization) Rules {
	n := &rules{
		orig: r.orig,
	}
	for _, e := range general.Optional(r.orig, r).rules {
		n.Add(PersonalizeRule(e, o))
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
		NewContextOption(""),
		NewKubeconfigOption("", "").WithInClusterMode(),
		NewEnvironmentVariable(),
		NewHomeDirectory(),
		NewInClusterConfig(),
	)
}

func DedicatedConfigRules(name, desc string) Rules {
	return NewRules(
		NewContextOption(""),
		NewKubeconfigOption("", "").WithInClusterMode(),
	).PersonalizedWith(&Personalization{
		Name:        name,
		Description: desc,
	})
}
