package common

import (
	"context"
	"fmt"
	"slices"

	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/objutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/logging"
	"github.com/miekg/dns"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Responsibility struct {
	Root    *corednsv1alpha1.HostedZone
	Parent  *corednsv1alpha1.HostedZone
	Runtime *string
	Class   *string
}

func (r *Responsibility) String() string {
	if r == nil {
		return "no info"
	}
	return fmt.Sprintf("root %q, parent %q, runtime: %q, class: %q", r.Root.Name, r.Parent.Name, String(r.Runtime, "<none>"), r.Class)
}

func (r *Responsibility) Check(class *string, runtime *string) bool {
	if r.Class == nil {
		return class == nil
	}
	if class == nil {
		return false
	}
	if *r.Class != *class {
		return false
	}
	if r.Runtime != nil {
		if runtime == nil || *r.Runtime != *runtime {
			return false
		}
	}
	return false
}

func GetRootInfoForEntry(ctx context.Context, c cluster.Cluster, logger logging.Logger, n client.ObjectKey) (*Responsibility, bool, reconcile.Problem) {
	var resp Responsibility

	var hist []string
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
	resp.Runtime = resp.Root.Spec.Runtime
	resp.Class = resp.Root.Spec.Class
	return &resp, true, nil
}

// GetRootInfo determine information about root zone.
// If dnsnames is given it aggregates as set covered final fqdn.
//
//	ok ->  err -> config problem
//         !err -> info
//	!ok && err -> temp problem

func GetRootInfo(ctx context.Context, c cluster.Cluster, logger logging.Logger, obj *corednsv1alpha1.HostedZone, dnsnames *[]string) (*Responsibility, bool, reconcile.Problem) {
	var path string
	var directParent *corednsv1alpha1.HostedZone

	var hist []string
	for {
		AggregateNames(obj, dnsnames)
		if obj.Spec.ParentRef == "" {
			return &Responsibility{Root: obj, Parent: directParent, Runtime: obj.Spec.Runtime, Class: obj.Spec.Class}, true, nil
		}
		hist = append(hist, obj.Name)

		var parent corednsv1alpha1.HostedZone
		path = path + "/" + obj.Spec.ParentRef
		logger.Info("handle parent", "parent", path)
		if slices.Contains(hist, obj.Spec.ParentRef) {
			return nil, true, reconcile.Failedf("reference cyle %s", path)
		}
		err := c.Get(ctx, objutils.RefObjectKeyFor(obj, obj.Spec.ParentRef), &parent)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, true, reconcile.Failedf("parent %q not found", path)
			}
			return nil, false, reconcile.TemporaryProblem(err)
		}
		if !parent.GetDeletionTimestamp().IsZero() {
			return nil, true, reconcile.Failedf("parent %s deleted", obj.Spec.ParentRef)
		}
		if directParent == nil {
			directParent = &parent
		}
		obj = &parent
	}
}

func AggregateNames(z *corednsv1alpha1.HostedZone, names *[]string) {
	if names == nil {
		return
	}
	var result []string

	for _, n := range *names {
		for _, suf := range z.Spec.DomainNames {
			result = append(result, dns.Fqdn(n)+dns.Fqdn(suf))
		}
	}
	*names = result
}
