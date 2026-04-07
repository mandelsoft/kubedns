package entry

import (
	"fmt"
	"slices"

	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/objutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/internal/controller/server"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TODO: plain mode support

type Request struct {
	*reconciler.BaseRequest[*corednsv1alpha1.CoreDNSEntry]
	*server.Options
	master bool
}

func (r *Request) Reconcile() reconcile.Problem {

	var names []string
	var zone corednsv1alpha1.HostedZone
	e := &r.Object.Spec

	if e.ZoneRef == "" {
		return r.updateStatus(false, "", nil, fmt.Errorf("no zone set"))
	}

	zn := objutils.RefObjectKeyFor(r.Object, r.Object.Spec.ZoneRef)

	err := r.Get(r, zn, &zone)
	if err != nil {
		if errors.IsNotFound(err) {
			return r.updateStatus(false, "", nil, fmt.Errorf("no root zone found"))
		}
		return reconcile.TemporaryProblem(err)
	}

	names = slices.Clone(e.DNSNames)
	info, ok, prob := common.GetRootInfo(r, r, r, &zone, &names)
	if prob != nil {
		if !ok {
			// temp problem
			return prob
		}
		// config problem
		return r.updateStatus(false, "", nil, fmt.Errorf("%s", prob.Message()))
	}

	if !info.Check(r.Class, nil) {
		// not responsible
		if r.Object.Status.RootZone != "" && r.Object.Status.RootZone != info.Root.Name {
			// potentially lost responsibility
			r.Object.Status.RootZone = ""
			meta.RemoveStatusCondition(&r.Object.Status.Conditions, corednsv1alpha1.ServerConditionType)
			r.Object.Status.State = ""
			r.Object.Status.Message = ""
		}
		return nil
	}

	err = common.ValidateEntryData(r.Object)
	if err != nil {
		return r.updateStatus(true, "", nil, err)
	}

	r.Info("domain names: {{domainnames}}", "domainnames", names)

	if zone.Status.State != "Ready" && zone.Status.State != "Ok" {
		r.Info("zone {{zone}} state is {{state}}", "zone", zone.Name, "state", zone.Status.State)
		return r.updateStatus(true, zone.Name, names, fmt.Errorf("zone failure: %s", zone.Status.Message))
	}
	return r.updateStatus(true, info.Root.Name, names, nil)
}

func (r *Request) ReconcileDeleting() reconcile.Problem {
	return nil
}

func (r *Request) ReconcileDeleted() reconcile.Problem {
	return nil
}

func (r *Request) updateStatus(force bool, zn string, names []string, err error) reconcile.Problem {
	o := r.Object
	if o.Status.RootZone != zn {
		o.Status.RootZone = zn
	}
	if !slices.Equal(o.Status.EffectiveDomainNames, names) {
		o.Status.EffectiveDomainNames = names
	}
	if err != nil {
		r.Info("entry has problems: {{problem}}", "problem", err)
	}
	if force && r.master {
		if len(o.Status.Conditions) > 0 {
			o.Status.Conditions = nil
		}
	}
	if err != nil {
		if r.master {
			if force || o.Status.State != "Invalid" {
				if o.Status.Message != err.Error() || o.Status.State != "Invalid" {
					o.Status.Message = err.Error()
					o.Status.State = "Invalid"
				}
			}
		} else {
			meta.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:               corednsv1alpha1.ServerConditionType,
				Status:             metav1.ConditionFalse,
				ObservedGeneration: o.ObjectMeta.Generation,
				Reason:             corednsv1alpha1.ReasonServerValidationFailure,
				Message:            err.Error(),
			})
		}
		return reconcile.Failed(err)
	} else {
		if r.master {
			if o.Status.Message != "" || o.Status.State != "Ok" {
				o.Status.Message = ""
				o.Status.State = "Ok"
				r.Info("set entry to Ok")
			}
		} else {
			meta.SetStatusCondition(&o.Status.Conditions, metav1.Condition{
				Type:               corednsv1alpha1.ServerConditionType,
				Status:             metav1.ConditionTrue,
				ObservedGeneration: o.ObjectMeta.Generation,
				Reason:             corednsv1alpha1.ReasonServerActive,
				Message:            "entry served",
			})
		}
		return nil
	}
}
