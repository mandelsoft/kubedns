package restconfig

import (
	"github.com/mandelsoft/kubedns/pkg/kubeconfig"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type HomeDirectory struct {
}

var _ Rule = (*HomeDirectory)(nil)

func NewHomeDirectory() Rule {
	return HomeDirectory{}
}

func (h HomeDirectory) GetConfig() (*rest.Config, error) {
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	if err := rules.Migrate(); err != nil {
		return nil, err
	}

	return kubeconfig.TryKubeconfigFile(clientcmd.RecommendedHomeFile)
}
