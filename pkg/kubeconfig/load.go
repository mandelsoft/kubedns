package kubeconfig

import (
	"fmt"
	"os"

	"github.com/mandelsoft/goutils/general"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

// GetConfig provides a rest config based on an optional path
// (for a kubeconfig file) and an optional context.
// If path is in-cluster the usage of an in-cluster kubeconfig
// is enforced.
// If the path is empty the usual suspects as defined in
// clientcmd.NewDefaultClientConfigLoadingRules are used
// falling back to the in-cluster kubeconfig.
// If a path is given, it must provide a kubeconfig.
func GetConfig(path string, context string) (*rest.Config, error) {
	if path == "in-cluster" {
		return rest.InClusterConfig()
	}

	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = path
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		rules,
		&clientcmd.ConfigOverrides{CurrentContext: context}).ClientConfig()
}

func GetRestConfigFromKubeconfig(apiConfig *api.Config, ctx ...string) (*rest.Config, error) {
	// 1. Create a ClientConfig object from the api.Config
	// We use NewDefaultClientConfig to respect the "current-context" in the struct.
	clientConfig := clientcmd.NewDefaultClientConfig(*apiConfig, &clientcmd.ConfigOverrides{CurrentContext: general.Optional(ctx...)})

	// 2. Convert it to a rest.Config
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	return restConfig, nil
}

func TryKubeconfigFile(path string, ctx ...string) (*rest.Config, error) {
	config, err := clientcmd.LoadFromFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("kubeconfig file %s: %w", path, err)
	}
	return GetRestConfigFromKubeconfig(config)
}
