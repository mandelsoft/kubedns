package main

import (
	"os"

	kcpapisv1alpha1 "github.com/kcp-dev/sdk/apis/apis/v1alpha1"
	kcpcorev1alpha1 "github.com/kcp-dev/sdk/apis/core/v1alpha1"
	kcptenancyv1alpha1 "github.com/kcp-dev/sdk/apis/tenancy/v1alpha1"
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/cluster/fleet/kcp"
	"github.com/mandelsoft/kubecrtutils/component"
	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/ctrlmgmt"
	"github.com/mandelsoft/kubecrtutils/mapping"
	"github.com/mandelsoft/kubecrtutils/options/activationopts"
	"github.com/mandelsoft/kubecrtutils/options/healthzopts"
	"github.com/mandelsoft/kubecrtutils/options/metricsopts"
	"github.com/mandelsoft/kubecrtutils/options/mlogopts"
	"github.com/mandelsoft/kubecrtutils/options/workeropts"
	"github.com/mandelsoft/kubecrtutils/setup"
	"github.com/mandelsoft/kubedns/internal/controller/direct"
	"github.com/mandelsoft/kubedns/internal/controller/direct/entry"
	"github.com/mandelsoft/kubedns/internal/controller/direct/hostedzone"
	"github.com/mandelsoft/kubedns/internal/controller/replicate"
	repentry "github.com/mandelsoft/kubedns/internal/controller/replicate/entry"
	repzone "github.com/mandelsoft/kubedns/internal/controller/replicate/hostedzone"
	srventry "github.com/mandelsoft/kubedns/internal/controller/server/entry"
	srvzone "github.com/mandelsoft/kubedns/internal/controller/server/hostedzone"
	"github.com/mandelsoft/kubedns/internal/controller/server/servercomp"
	"github.com/spf13/pflag"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
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

const IndexKeyEntryZone = "entry2zone"
const IndexKeyZoneParent = "zone2parent"

// nolint:gocyclo
func main() {

	setup.ExitIfErr(hostedzone.TestRenderManifests(setup.Log), "problems with included manifests")
	setup.ExitIfErr(hostedzone.TestRenderKubeDNSManifests(setup.Log), "problems with included dns manifests")

	// index mappings for direct controllers
	dmap := mapping.NewConfigurableControllerMappings().
		MapIndex(direct.IndexKeyEntryZone, IndexKeyEntryZone).
		MapIndex(direct.IndexKeyZoneParent, IndexKeyZoneParent)

	def := ctrlmgmt.Define(corednsv1alpha1.GroupVersion.Group, cluster.DEFAULT, "runtime", "target", "dataplane", "source").
		WithScheme(scheme).
		AddCluster(
			cluster.Define("runtime", "runtime cluster").WithFallback("target"),
			cluster.DefineFleet("dataplane", "api cluster for functional controllers", kcp.Type()).WithFallback("runtime"),
			cluster.DefineFleet("source", "user api cluster", kcp.Type()).WithFallback("target"),
			cluster.Define("target", "replication target").WithFallback(cluster.DEFAULT),
			cluster.DefineFleet(cluster.DEFAULT, "default cluster as fallback for target", kcp.Type()),
		).
		AddController(
			controller.WithMappings(hostedzone.Controller()).
				UseMappings(dmap),
			controller.WithMappings(entry.Controller()).
				UseMappings(dmap),

			repentry.Controller(),
			controller.WithMappings(repzone.Controller()).
				MapIndex(replicate.IndexKeyEntryZone, IndexKeyEntryZone).
				MapIndex(replicate.IndexKeyZoneParent, IndexKeyZoneParent),

			srventry.Controller(),
			srvzone.Controller(),
		).
		AddComponent(
			component.WithMappings(servercomp.Server()).MapIndex(servercomp.INDEX_ZONENAMES, "dnsnames"),
		)

	options := &flagutils.DefaultOptionSet{}

	options.Add(
		metricsopts.New(),    // options to control the manager metrics service
		healthzopts.New(),    // options to configure readiness and livenaess probe
		mlogopts.New(true),   // options to control mandelsoft/logging
		activationopts.New(), // enable controller selection
		workeropts.New(),     // enable work queue configuration
		// other options
	)

	err := ctrlmgmt.Setup("dnsmanager", options, def, os.Args[1:]...)
	if err == pflag.ErrHelp {
		os.Exit(0)
	}
	setup.ExitIfErr(err, "setup controller manager")
}
