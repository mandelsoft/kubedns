package hostedzone

import (
	"context"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/clusterutils"
	"github.com/mandelsoft/kubedns/pkg/enqueue"
	"github.com/mandelsoft/kubedns/pkg/index"
	"github.com/mandelsoft/kubedns/pkg/objutils"
	"github.com/mandelsoft/kubedns/pkg/owner"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/cluster"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const IndexKeyParent = "hostedzone.parent"

// SetupWithManager sets up the controller with the Manager.
func (r *HostedZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Logger = Log
	r.index = index.NewUntyped()
	r.DataPlane = clusterutils.NewClusterForCluster("dataplane", mgr)

	r.Mux = enqueue.NewMux(mgr.GetScheme())

	trigger, err := r.Mux.Source(&corednsv1alpha1.HostedZone{})
	if err != nil {
		return err
	}
	_ = trigger

	r.FieldManager = "coredns.mandelsoft.org/hostedzone"
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

	u, _, err := rest.DefaultServerUrlFor(mgr.GetConfig())
	if err != nil {
		return err
	}
	r.DataPlaneURL = u.String()
	m, err := GetManifests()
	if err != nil {
		return err
	}
	r.Manifests = m
	if mgr.GetConfig() == r.Options.RuntimeConfig.GetRestConfig() {
		r.Runtime = clusterutils.NewAlias("runtime", r.DataPlane)
	} else {
		r.Runtime, err = clusterutils.NewCluster("runtime", r.Options.RuntimeConfig.GetRestConfig(), func(opts *cluster.Options) {
			opts.Scheme = mgr.GetScheme()
		})
		mgr.Add(r.Runtime)
		if err != nil {
			return err
		}
	}

	if r.IsSeparateRuntime() {
		r.Info("using separated runtime namespace", "namespace", r.Options.RuntimeNamespace)
		r.Info("using Runtime mode")
		r.Mode = NewRuntimeMode(r)
		r.runtimeOwner = owner.RemoteOwner("coredns.mandelsoft.org/owner-id", r.Options.Class)
	} else {
		r.Info("using Local mode")
		r.Mode = NewLocalMode(r)
		r.runtimeOwner = owner.LocalOwner(r.DataPlane.GetScheme())
	}

	r.runtimeOwner = owner.Conditional(r.runtimeOwner, objutils.Or(
		objutils.GroupKindFilter("core", "Service"),
		objutils.GroupKindFilter("apps", "Deployment"),
	))

	r.Info("using dataplane cluster", "apiserver", mgr.GetConfig().Host)
	if r.Runtime != r.DataPlane {
		r.Info("using separated runtime cluster", "apiserver", r.Options.RuntimeConfig.GetRestConfig().Host)
	} else {
		r.Info("using same cluster as runtime")
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corednsv1alpha1.HostedZone{}, IndexKeyParent, func(rawObj client.Object) []string {
		res := rawObj.(*corednsv1alpha1.HostedZone)
		if res.Spec.ParentRef == "" {
			return nil
		}
		return []string{res.Spec.ParentRef}
	}); err != nil {
		return err
	}

	dlog := LoggingFor("deploymentwatch")
	slog := LoggingFor("servicewatch")
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&corednsv1alpha1.HostedZone{}).
		Named("coredns-hostedzone").
		WatchesRawSource(trigger).
		WatchesRawSource(
			owner.WatchSourceForSlave[*appsv1.Deployment, *corednsv1alpha1.HostedZone](r.Runtime, r.runtimeOwner, r.DataPlane, dlog),
		).
		WatchesRawSource(
			owner.WatchSourceForSlave[*corev1.Service, *corednsv1alpha1.HostedZone](r.Runtime, r.runtimeOwner, r.DataPlane, slog),
		)

	if r.IsSeparateRuntime() {
		log := LoggingFor("sa-secretwatch")
		r.Info("setting up secret watch for serviceaccount secrets for separated runtime access")
		builder.Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
				var trigger []reconcile.Request
				key := client.ObjectKeyFromObject(obj)
				users := r.index.UsersFor(INDEX_SASECFRET, key)
				if len(users) > 0 {
					log.Info("change of service account secret {{secret}} triggers {{amount}} zones",
						"secret", key,
						"amount", len(users))
				}
				for user := range users {
					zone := types.NamespacedName{
						Name:      user.Name,
						Namespace: user.Namespace,
					}
					log.Info("triggering hostedzone {{zone}}",
						"secret", key, "zone", zone)
					trigger = append(trigger,
						reconcile.Request{
							NamespacedName: zone,
						},
					)
				}
				return trigger
			}),
		)
	}
	return builder.Complete(r)
}

func (r *HostedZoneReconciler) IsSeparateRuntime() bool {
	return r.Options.RuntimeNamespace != "" || r.DataPlane.GetClient() != r.Runtime.GetClient()
}

func (r *HostedZoneReconciler) GetChildren(ctx context.Context, ns string, n string) []corednsv1alpha1.HostedZone {
	var list corednsv1alpha1.HostedZoneList

	err := r.DataPlane.List(ctx, &list, client.InNamespace(ns), client.MatchingFields{IndexKeyParent: n})
	if err != nil {
		return nil
	}

	return list.Items
}
