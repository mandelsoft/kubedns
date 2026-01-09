package main

import (
	"os"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/internal/controller/hostedzone"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/ctrlmgmt"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/metricsopts"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/mlogopts"
	"github.com/mandelsoft/kubedns/pkg/setup"
	"github.com/spf13/pflag"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/entry"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corednsv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {

	setup.ExitIfErr(hostedzone.TestRenderManifests(setup.Log), "problems with included manifests")
	setup.ExitIfErr(hostedzone.TestRenderKubeDNSManifests(setup.Log), "problems with included dns manifests")

	def := ctrlmgmt.Define(corednsv1alpha1.GroupVersion.Group, "dataplane").
		WithScheme(scheme).
		AddCluster(
			cluster.Define("runtime", "runtime cluster").WithFallback("dataplane"),
			cluster.Define("dataplane", "user api cluster").WithFallback(cluster.DEFAULT),
		).
		AddController(
			hostedzone.Controller(),
			entry.Controller(),
		)

	options := flagutils.DefaultOptionSet{}

	options.Add(
		metricsopts.New(),  // options to control the manager metrics service
		mlogopts.New(true), // options to control mandelsoft/logging
		// other options
	)

	err := ctrlmgmt.Setup(options, def, os.Args[1:]...)
	if err == pflag.ErrHelp {
		os.Exit(0)
	}
	setup.ExitIfErr(err, "setup controller manager")
}
