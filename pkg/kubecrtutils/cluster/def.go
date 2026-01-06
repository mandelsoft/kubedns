package cluster

import (
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster/config"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/runtime"
)

const DEFAULT = "default"

type DefinitionProvider interface {
	GetDefinition() Definition
}

type Definition interface {
	DefinitionProvider
	flagutils.Options
	flagutils.OptionSetProvider

	RequireIdentity()

	GetConfig(*config.ConfigOptions) (*config.Config, error)

	GetName() string
	GetFallback() string
	GetDescription() string
	GetScheme() *runtime.Scheme

	WithFallback(fallback string) Definition
	WithScheme(scheme *runtime.Scheme) Definition
}

type definition struct {
	name     string
	fallback string
	rules    config.Rules
	desc     string
	scheme   *runtime.Scheme
}

var _ Definition = (*definition)(nil)

func Define(name string, desc string, rule ...config.Rule) Definition {
	if len(rule) == 0 {
		rule = []config.Rule{config.DedicatedConfigRules(name, desc)}
	}
	return &definition{name: name, desc: desc, fallback: DEFAULT, rules: config.NewRules(rule...)}
}

func (d *definition) WithFallback(fallback string) Definition {
	d.fallback = fallback
	return d
}

func (d *definition) WithScheme(scheme *runtime.Scheme) Definition {
	d.scheme = scheme
	return d
}

func (d *definition) GetDefinition() Definition {
	return d
}

func (d *definition) RequireIdentity() {
	d.rules.RequireIdentity()
}

func (d *definition) GetName() string {
	return d.name
}

func (d *definition) GetFallback() string {
	return d.fallback
}

func (d *definition) GetDescription() string {
	return d.desc
}

func (d *definition) GetScheme() *runtime.Scheme {
	return d.scheme
}

func (d *definition) GetConfig(*config.ConfigOptions) (*config.Config, error) {
	return d.rules.GetConfig(nil)
}

func (d *definition) AddFlags(fs *pflag.FlagSet) {
	d.rules.AddFlags(fs)
}

func (d *definition) AsOptionSet() flagutils.OptionSet {
	return d.rules.AsOptionSet()
}

////////////////////////////////////////////////////////////////////////////////
