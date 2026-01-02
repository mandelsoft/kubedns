package restconfig

import (
	"context"
	"strings"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/goutils/maputils"
	"github.com/mandelsoft/kubedns/pkg/kubeconfig"
	"github.com/spf13/pflag"
	"k8s.io/client-go/rest"
)

type KubeConfigOption struct {
	name    string
	desc    string
	special map[string]Rule

	path    string
	context string

	config *rest.Config
	err    error
}

var (
	_ flagutils.Validatable = (*KubeConfigOption)(nil)
	_ flagutils.Options     = (*KubeConfigOption)(nil)
	_ Rule                  = (*KubeConfigOption)(nil)
	_ Personalizable        = (*KubeConfigOption)(nil)
)

func NewKubeconfigOption(name, desc string) *KubeConfigOption {
	if name == "" {
		name = "kubeconfig"
	}
	if desc == "" {
		desc = "path to standard kubeconfig"
	}
	return &KubeConfigOption{
		name:    name,
		desc:    desc,
		special: make(map[string]Rule),
	}
}

// WithSpecialCase adds a configurable key mapped to the usage of a special rule.
// ATTENTION: This rule must not use options anymore.
func (r *KubeConfigOption) WithSpecialCase(name string, rule Rule) *KubeConfigOption {
	r.special[name] = rule
	return r
}

func (r *KubeConfigOption) PersonalizedWith(name string) Rule {
	desc := r.desc
	i := strings.Index(name, "@")
	pers := name
	if i >= 0 {
		desc = name[i+1:]
		pers = name[:i]
	}
	if len(pers) == 0 {
		name = r.name
	} else {
		name = pers + "-" + r.name
	}
	return &KubeConfigOption{
		name:    name,
		desc:    desc,
		special: maputils.TransformValues(r.special, func(r Rule) Rule { return PersonalizeRule(r, pers) }),
	}
}

func (r *KubeConfigOption) GetConfig() (*rest.Config, error) {
	return r.config, r.err
}

func (r *KubeConfigOption) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&r.context, r.name+"-context", "", "", "context used together with "+r.name)
	fs.StringVarP(&r.path, r.name, "", "", r.desc)
}

func (r *KubeConfigOption) Validate(ctx context.Context, opts flagutils.OptionSet, v flagutils.ValidationSet) error {
	if r.config == nil && r.path != "" {

		if s := r.special[r.path]; s != nil {
			r.config, r.err = s.GetConfig()
			return r.err
		}
		context := ""
		path := r.path
		i := strings.Index(r.path, "@")
		if i >= 0 {
			context = r.path[:i]
			path = r.path[i+1:]
		}
		r.config, r.err = kubeconfig.TryKubeconfigFile(path, context)
	}
	return r.err
}

func (r *KubeConfigOption) WithInClusterMode(name ...string) *KubeConfigOption {
	return r.WithSpecialCase(general.OptionalDefaulted("in-cluster", name...), NewInClusterConfig())
}
