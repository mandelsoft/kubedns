package restconfig

import (
	"fmt"
	"os"

	"github.com/mandelsoft/goutils/general"
	"github.com/mandelsoft/kubedns/pkg/kubeconfig"
	"k8s.io/client-go/rest"
)

type EnvironmentVariable struct {
	Name string
}

func NewEnvironmentVariable(name ...string) *EnvironmentVariable {
	return &EnvironmentVariable{general.OptionalDefaulted("KUBECONFIG", name...)}
}

func (r *EnvironmentVariable) GetConfig() (*rest.Config, error) {
	v := os.Getenv(r.Name)
	if v == "" {
		return nil, nil
	}
	cfg, err := kubeconfig.TryKubeconfigFile(v)
	if cfg == nil && err == nil {
		return nil, fmt.Errorf("kubeconfig file %q from environment variable %s not found", v, r.Name)
	}
	return cfg, err
}
