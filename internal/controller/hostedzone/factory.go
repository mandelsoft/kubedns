package hostedzone

import (
	"context"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/owner"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/types"
	"github.com/mandelsoft/kubedns/pkg/objutils"
	"github.com/mandelsoft/logging"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func Controller() controller.Definition {
	return controller.Define[corednsv1alpha1.HostedZone](common.ControllerHostedzone, "dataplane", &ReconcilerFactory{}).
		UseCluster("runtime").
		AddIndex(common.IndexKeyZoneParent, parentIndexer).
		AddTrigger(
			controller.OwnerTrigger[appsv1.Deployment]().OnCluster("runtime"),
			controller.OwnerTrigger[corev1.Secret]().OnCluster("runtime"),
			controller.ResourceTriggerByFactory[corev1.Secret](secretTriggerFactory),
		)
}

func parentIndexer(o *corednsv1alpha1.HostedZone) []string {
	if o.Spec.ParentRef == "" {
		return nil
	}
	return []string{o.Spec.ParentRef}
}

func secretTriggerFactory(c types.Controller, target types.Cluster, proto client.Object, log logging.Logger) (handler.TypedMapFunc[*corev1.Secret, reconcile.Request], error) {
	r := c.GetReconciler().(*HostedZoneReconciler)

	return func(ctx context.Context, obj *corev1.Secret) []reconcile.Request {
		var trigger []reconcile.Request
		key := client.ObjectKeyFromObject(obj)
		users := r.index.UsersFor(INDEX_SASECFRET, key)
		if len(users) > 0 {
			log.Info("change of service account secret {{secret}} triggers {{amount}} zones",
				"secret", key,
				"amount", len(users))
		}
		for user := range users {
			zone := apitypes.NamespacedName{
				Name:      user.Name,
				Namespace: user.Namespace,
			}
			trigger = append(trigger,
				reconcile.Request{
					NamespacedName: zone,
				},
			)
		}
		return trigger
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

type ReconcilerFactory struct {
	Options
}

func (f *ReconcilerFactory) CreateReconciler(ctx context.Context, controller controller.Controller[corednsv1alpha1.HostedZone, *corednsv1alpha1.HostedZone], b *builder.Builder) (reconcile.Reconciler, error) {
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
		recorder:   controller.GetRecoder(),
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

	r.Info("using dataplane cluster", "apiserver", r.DataPlane.GetConfig().Host)
	if !r.Runtime.IsSameAs(r.DataPlane) {
		r.Info("using separated runtime cluster", "apiserver", r.Runtime.GetConfig().Host)
	} else {
		r.Info("using same cluster as runtime")
	}

	if r.IsSeparateRuntime() {
		r.Info("setting up secret watch for serviceaccount secrets for separated runtime access")
	}
	return r, nil

}
