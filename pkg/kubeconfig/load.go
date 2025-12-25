package kubeconfig

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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
