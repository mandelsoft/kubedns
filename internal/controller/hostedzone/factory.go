package hostedzone

import (
	"context"
	"time"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/objutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/owner"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	crtreconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type ReconcilerFactory struct {
	Options
}

func (f *ReconcilerFactory) CreateReconciler(ctx context.Context, controller controller.Controller[corednsv1alpha1.HostedZone, *corednsv1alpha1.HostedZone], b *builder.Builder) (crtreconcile.Reconciler, error) {
	logger := controller.GetLogger()
	logger.Info("creating hostedzone reconciler...")
	base, err := common.NewReconciler(controller)
	if err != nil {
		return nil, err
	}
	clusters := controller.GetClusters()

	r := &HostedZoneReconciler{
		Reconciler: base,
		Runtime:    clusters.Get("runtime"),
		index:      index.NewUntyped(),
		Options:    &f.Options,
	}

	if r.Options == nil {
		r.Options = NewOptions()
	}

	if r.Options.Class != "" {
		r.FieldManager += "-" + r.Options.Class
	}
	if r.Options.Runtime != "" {
		r.FieldManager += "--" + r.Options.Runtime
	}
	r.Finalizer = r.FieldManager

	r.Info("using dataplane cluster", "apiserver", r.DataPlane.GetConfig().Host)
	if !r.Runtime.IsSameAs(r.DataPlane) {
		r.Info("using separated runtime cluster", "apiserver", r.Runtime.GetConfig().Host)
	} else {
		r.Info("using same cluster as runtime")
	}

	r.Info("for Class '{{class}}'", "class", r.Options.Class)
	r.Info("for Runtime '{{runtime}}'", "runtime", r.Options.Runtime)
	r.Info("for Platform mode '{{platform}}'", "platform", r.Options.Platform)
	r.Info("using FieldManger '{{fieldmanager}}'", "fieldmanager", r.FieldManager)
	r.Info("using Finalizer '{{finalizer}}'", "finalizer", r.Finalizer)
	r.Info("using Nameserver mode '{{mode}}'", "mode", r.Options.DNSMode)

	u, _, err := rest.DefaultServerUrlFor(r.DataPlane.GetConfig())
	if err != nil {
		return nil, err
	}
	r.DataPlaneURL = u.String()
	m, err := GetManifests()
	if err != nil {
		return nil, err
	}
	r.Manifests = m

	r.runtimeOwner = owner.Conditional(owner.For(r.DataPlane, r.Runtime), objutils.Or(
		objutils.GroupKindFilter("core", "Service"),
		objutils.GroupKindFilter("apps", "Deployment"),
	))

	if r.IsSeparateRuntime() {
		r.Info("using separated runtime namespace", "namespace", r.Options.RuntimeNamespace)
		r.Info("using Runtime mode")
		r.Mode = NewRuntimeMode(r)
	} else {
		r.Info("using Local mode")
		r.Mode = NewLocalMode(r)
	}

	if r.IsSeparateRuntime() {
		r.Info("setting up secret watch for serviceaccount secrets for separated runtime access")
	}
	return reconciler.CRTReconcilerFor(controller, r, 300*time.Second), nil
}
