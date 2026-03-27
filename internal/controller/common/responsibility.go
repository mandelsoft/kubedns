package common

import (
	"context"
	"fmt"
	"slices"

	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Responsibility struct {
	Root       *corednsv1alpha1.HostedZone
	Parent     *corednsv1alpha1.HostedZone
	RuntimeSet bool
	Runtime    string
	Class      string
}

func (r *Responsibility) String() string {
	if r == nil {
		return "no info"
	}
	return fmt.Sprintf("root %q, parent %q, runtime: %q, class: %q", r.Root.Name, r.Parent.Name, r.Runtime, r.Class)
}

func GetRootInfoForEntry(ctx context.Context, c cluster.Cluster, logger logging.Logger, n client.ObjectKey) (*Responsibility, bool, reconcile.Problem) {
	var resp Responsibility

	hist := []string{}
	path := ""
	zone := n.Name
	for zone != "" {
		var parent corednsv1alpha1.HostedZone
		path = path + "/" + zone
		logger.Info("handle parent", "parent", path)
		if slices.Contains(hist, zone) {
			return nil, true, reconcile.Failedf("reference cyle %s", path)
		}
		err := c.Get(ctx, client.ObjectKey{Namespace: n.Namespace, Name: zone}, &parent)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, true, reconcile.Failedf("parent %s not found", zone)
			}
			return nil, false, reconcile.TemporaryProblem(err)
		}
		if !parent.GetDeletionTimestamp().IsZero() {
			return nil, true, reconcile.Failedf("parent %s deleted", zone)
		}
		if resp.Parent == nil {
			resp.Parent = &parent
		}
		resp.Root = &parent
		hist = append(hist, zone)
		zone = parent.Spec.ParentRef
	}
	resp.Runtime = String(resp.Root.Spec.Runtime, "")
	resp.RuntimeSet = resp.Root.Spec.Runtime != nil
	resp.Class = String(resp.Root.Spec.Class, "")
	return &resp, true, nil
}

func GetRootInfo(ctx context.Context, c cluster.Cluster, logger logging.Logger, obj *corednsv1alpha1.HostedZone) (*Responsibility, bool, reconcile.Problem) {
	var path string
	var directParent *corednsv1alpha1.HostedZone

	hist := []string{obj.GetName()}
	for obj.Spec.ParentRef != "" {
		var parent corednsv1alpha1.HostedZone
		path = path + "/" + obj.Spec.ParentRef
		logger.Info("handle parent", "parent", path)
		if slices.Contains(hist, obj.Spec.ParentRef) {
			return nil, true, reconcile.Failedf("reference cyle %s", path)
		}
		err := c.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.Spec.ParentRef}, &parent)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, true, reconcile.Failedf("parent %s not found", obj.Spec.ParentRef)
			}
			return nil, false, reconcile.TemporaryProblem(err)
		}
		if !parent.GetDeletionTimestamp().IsZero() {
			return nil, true, reconcile.Failedf("parent %s deleted", obj.Spec.ParentRef)
		}
		if directParent == nil {
			directParent = &parent
		}
		hist = append(hist, obj.Spec.ParentRef)
		obj = &parent
	}
	return &Responsibility{Root: obj, Parent: directParent, Runtime: String(obj.Spec.Runtime, ""), RuntimeSet: obj.Spec.Runtime != nil, Class: String(obj.Spec.Class, "")}, true, nil
}
