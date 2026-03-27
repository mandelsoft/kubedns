package main

import (
	"os"

	kcpapisv1alpha1 "github.com/kcp-dev/sdk/apis/apis/v1alpha1"
	kcpcorev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	kcptenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/cluster/fleet/kcp"
	"github.com/mandelsoft/kubecrtutils/ctrlmgmt"
	"github.com/mandelsoft/kubecrtutils/options/activationopts"
	"github.com/mandelsoft/kubecrtutils/options/metricsopts"
	"github.com/mandelsoft/kubecrtutils/options/mlogopts"
	"github.com/mandelsoft/kubecrtutils/setup"
	entrydown "github.com/mandelsoft/kubedns/internal/controller/replicate/entry/down"
	entryup "github.com/mandelsoft/kubedns/internal/controller/replicate/entry/up"
	zonedown "github.com/mandelsoft/kubedns/internal/controller/replicate/hostedzone/down"
	zoneup "github.com/mandelsoft/kubedns/internal/controller/replicate/hostedzone/up"
	"github.com/spf13/pflag"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/entry"
	"github.com/mandelsoft/kubedns/internal/controller/hostedzone"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(kcpcorev1alpha1.AddToScheme(scheme))
	utilruntime.Must(kcptenancyv1alpha1.AddToScheme(scheme))
	utilruntime.Must(kcpapisv1alpha1.AddToScheme(scheme))

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(corednsv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// nolint:gocyclo
func main() {

	setup.ExitIfErr(hostedzone.TestRenderManifests(setup.Log), "problems with included manifests")
	setup.ExitIfErr(hostedzone.TestRenderKubeDNSManifests(setup.Log), "problems with included dns manifests")

	def := ctrlmgmt.Define(corednsv1alpha1.GroupVersion.Group, "source").
		WithScheme(scheme).
		AddCluster(
			cluster.Define("runtime", "runtime cluster").WithFallback("dataplane"),
			cluster.DefineFleet("dataplane", "api cluster for functional controllers", kcp.Type()).WithFallback("target"),
			cluster.DefineFleet("source", "user api cluster", kcp.Type()).WithFallback("target"),
			cluster.Define("target", "replication target").WithFallback(cluster.DEFAULT),
		).
		AddController(
			hostedzone.Controller(),
			entry.Controller(),

			entryup.Controller(),
			entrydown.Controller(),

			zoneup.Controller(),
			zonedown.Controller(),
		)

	options := &flagutils.DefaultOptionSet{}

	options.Add(
		metricsopts.New(),    // options to control the manager metrics service
		mlogopts.New(true),   // options to control mandelsoft/logging
		activationopts.New(), // enable controller selection
		// other options
	)

	err := ctrlmgmt.Setup("dnsmanager", options, def, os.Args[1:]...)
	if err == pflag.ErrHelp {
		os.Exit(0)
	}
	setup.ExitIfErr(err, "setup controller manager")
}
