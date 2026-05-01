package hostedzone

import (
	"context"
	"time"

	"github.com/mandelsoft/kubecrtutils/controller"
	"github.com/mandelsoft/kubecrtutils/controller/builder"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/index"
	"github.com/mandelsoft/kubecrtutils/objutils/objfilter"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/direct/common"
	crtreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcilerFactory struct {
	Options
}

func (f *ReconcilerFactory) CreateReconciler(ctx context.Context, controller controller.TypedController[*corednsv1alpha1.HostedZone, corednsv1alpha1.HostedZone], b builder.Builder) (crtreconcile.Reconciler, error) {
	logger := controller.GetLogger()
	logger.Info("creating hostedzone reconciler...")
	base, err := common.NewReconciler(controller)
	if err != nil {
		return nil, err
	}
	clusters := controller.GetClusters()

	r := &HostedZoneReconciler{
		Reconciler: base,
		Runtime:    clusters.Get("runtime").AsCluster(),
		index:      index.NewUntyped(),
		Options:    &f.Options,
	}

	if r.Options == nil {
		r.Options = NewOptions()
	}

	if r.Options.Class != nil && *r.Options.Class != "" {
		r.FieldManager += "-" + *r.Options.Class
	}
	if r.Options.Runtime != nil && *r.Options.Runtime != "" {
		r.FieldManager += "--" + *r.Options.Runtime
	}
	r.Finalizer = r.FieldManager

	r.Info("using dataplane {{type}} {{info}}", "type", r.Dataplane.GetTypeInfo(), "info", r.Dataplane.GetInfo())
	if !r.Runtime.IsSameAs(r.Dataplane) {
		r.Info("using separated runtime cluster", "apiserver", r.Runtime.GetInfo())
	} else {
		r.Info("using same cluster as runtime")
	}

	r.Info("for Class '{{class}}'", "class", r.Options.Class)
	r.Info("for Runtime '{{runtime}}'", "runtime", r.Options.Runtime)
	r.Info("for Platform mode '{{platform}}'", "platform", r.Options.Platform)
	r.Info("using FieldManger '{{fieldmanager}}'", "fieldmanager", r.FieldManager)
	r.Info("using Finalizer '{{finalizer}}'", "finalizer", r.Finalizer)
	r.Info("using Nameserver mode '{{mode}}'", "mode", r.Options.DNSMode)

	m, err := GetManifests()
	if err != nil {
		return nil, err
	}
	r.Manifests = m

	r.ownerFilter = objfilter.Or(
		objfilter.GroupKind("core", "Service"),
		objfilter.GroupKind("apps", "Deployment"),
	)

	if r.IsSeparateRuntime() {
		r.Info("using separated runtime namespace", "namespace", r.Options.RuntimeNamespace)
		r.Info("using Runtime mode")
		r.Mode = NewRuntimeMode(r)
	} else {
		r.Info("using Local mode")
		r.Mode = NewLocalMode(r)
	}

	if err := r.Mode.Validate(); err != nil {
		return nil, err
	}

	r.ServerMode, err = NewDataplaneServer(ctx, r.Options)
	if err != nil {
		return nil, err
	}

	if r.IsSeparateRuntime() {
		r.Info("setting up secret watch for serviceaccount secrets for separated runtime access")
	}
	return reconciler.CRTReconcilerFor(controller, r, 300*time.Second), nil
}
