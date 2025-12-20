package hostedzone

import (
	"context"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/pkg/enqueue"
	"github.com/mandelsoft/kubedns/pkg/setup"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const IndexKeyParent = "hostedzone.parent"

// SetupWithManager sets up the controller with the Manager.
func (r *HostedZoneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Mux = enqueue.NewMux(mgr.GetScheme())

	src, err := r.Mux.Source(&corednsv1alpha1.HostedZone{})
	if err != nil {
		return err
	}
	_ = src

	r.FieldManager = "hostedzone-controller"
	if r.Options.Class != "" {
		r.FieldManager += "-" + r.Options.Class
	}
	if r.Options.Runtime != "" {
		r.FieldManager += "--" + r.Options.Runtime
	}

	r.Finalizer = r.FieldManager

	setup.Log.Info("for Class", "class", r.Options.Class)
	setup.Log.Info("for Runtime", "runtime", r.Options.Runtime)
	setup.Log.Info("using FieldManger", "fieldmanager", r.FieldManager)
	setup.Log.Info("using Finalizer", "finalizer", r.Finalizer)

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
		r.Runtime = r.DataPlane
	} else {
		r.Runtime, err = client.New(r.Options.RuntimeConfig.GetRestConfig(), client.Options{Scheme: mgr.GetScheme()})
		if err != nil {
			return err
		}
	}

	if r.Runtime == r.DataPlane && r.Options.RuntimeNamespace == "" {
		setup.Log.Info("using Local mode")
		r.Mode = NewLocalMode(r)
	} else {
		setup.Log.Info("using Runtime mode", "runtime-namespace", r.Options.RuntimeNamespace)
		r.Mode = NewRuntimeMode(r)
	}

	setup.Log.Info("using dataplane cluster", "apiserver", mgr.GetConfig().Host)
	if r.Runtime != r.DataPlane {
		setup.Log.Info("using separated runtime cluster", "apiserver", r.Options.RuntimeConfig.GetRestConfig().Host)
	} else {
		setup.Log.Info("using same cluster as runtime")
	}

	if r.IsSeparateRuntime() {
		setup.Log.Info("using separated runtime namespace", "namespace", r.Options.RuntimeNamespace)
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

	return ctrl.NewControllerManagedBy(mgr).
		For(&corednsv1alpha1.HostedZone{}).
		Named("coredns-hostedzone"). // WatchesRawSource(src).
		Complete(r)
}

func (r *HostedZoneReconciler) IsSeparateRuntime() bool {
	return r.Options.RuntimeNamespace != "" || r.DataPlane != r.Runtime
}

func (r *HostedZoneReconciler) GetChildren(ctx context.Context, ns string, n string) []corednsv1alpha1.HostedZone {
	var list corednsv1alpha1.HostedZoneList

	err := r.DataPlane.List(ctx, &list, client.InNamespace(ns), client.MatchingFields{IndexKeyParent: n})
	if err != nil {
		return nil
	}

	return list.Items
}
