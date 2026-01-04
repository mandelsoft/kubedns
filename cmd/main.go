package main

import (
	"os"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/internal/controller/hostedzone"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/manageropts"
	"github.com/mandelsoft/kubedns/pkg/setup"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

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

	options := flagutils.DefaultOptionSet{}

	options.Add(
		cluster.NewDefinitions().WithScheme(scheme).
			Add(cluster.NewDefinition("dataplane", "user facing configuration dataplane")).
			Add(cluster.NewDefinition("runtime", "runtime cluster").WithFallback("dataplane")),

		// zapopts.New(&zap.Options{
		//	Development: true,
		// }),
		manageropts.New("dataplane", "coredns.mandelsoft.org"),
		// metrics.New(),
		// webhookopts.New(),
		hostedzone.NewOptions(),
	)

	setup.Log.Info("condiguring options...")
	setup.Setup(options, os.Args[1:]...)

	setup.ExitIfErr(hostedzone.TestRenderManifests(), "problems with included mainfests")

	if err := hostedzone.Create(options); err != nil {
		setup.Log.Error(err, "unable to create controller", "controller", "HostedZone")
		os.Exit(1)
	}
	if err := entry.Create(options); err != nil {
		setup.Log.Error(err, "unable to create controller", "controller", "CoreDNSEntry")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	mgr := manageropts.From(options).GetManager()
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setup.Log.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setup.Log.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setup.Log.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setup.Log.Error(err, "problem running manager")
		os.Exit(1)
	}
}
