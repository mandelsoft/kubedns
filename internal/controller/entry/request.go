package entry

import (
	errors2 "errors"
	"fmt"
	"net"
	"slices"

	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconciler"
	"github.com/mandelsoft/kubecrtutils/objutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ReconcileRequest struct {
	reconciler.DefaultReconcileRequest[*corednsv1alpha1.CoreDNSEntry, *CoreDNSEntryReconciler]
}

func (r *ReconcileRequest) Reconcile() reconcile.Problem {
	baseerr := r.Validate()
	e := r.Object

	var root *corednsv1alpha1.HostedZone
	var zone *corednsv1alpha1.HostedZone

	if e.Spec.ZoneRef != "" {
		resp, prob := r.responsibleForEntry()
		r.Info("responsible", "responsible", resp, "error", prob)
		if prob != nil {
			return prob
		}
		r.Info("responsible {{responsible}}", "responsible", resp)

		if resp.Error != "" {
			baseerr = errors2.Join(fmt.Errorf("zone error: %s", resp.Error), baseerr)
		} else {
			root = resp.Root
			zone = resp.Zone
		}
	} else {
		baseerr = errors2.Join(fmt.Errorf("zone reference required"), baseerr)
	}

	if baseerr != nil {
		r.Info("found problem {{error}}", "error", baseerr)
	}
	if root != nil {
		if common.String(root.Spec.Runtime, "") != r.Reconciler.Options.Runtime || common.String(root.Spec.Class, "") != r.Reconciler.Options.Class {
			r.Info("runtime or class mismatch -> ignore", "rumtime", root.Spec.Runtime, "class", root.Spec.Class)
			return nil
		}
	}
	if baseerr != nil {
		r.SetStatusCondition(metav1.Condition{
			Type:    corednsv1alpha1.ValidationConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  corednsv1alpha1.ReasonConfigurarationInvalid,
			Message: baseerr.Error(),
		})
	} else {
		r.SetStatusCondition(metav1.Condition{
			Type:    corednsv1alpha1.ValidationConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  corednsv1alpha1.ReasonConfigurarationValid,
			Message: "configuration valid",
		})
	}

	// create summary
	state := corednsv1alpha1.STATE_READY
	msg := ""
	if baseerr != nil {
		state = "Invalid"
		msg = baseerr.Error()
	}

	c := meta.FindStatusCondition(e.Status.Conditions, corednsv1alpha1.ServerConditionType)
	if c != nil {
		if c.Status == metav1.ConditionFalse {
			if baseerr == nil {
				state = "Problem"
				msg = c.Message
			}
		}
	}

	if state == corednsv1alpha1.STATE_READY {
		if zone.Status.State != corednsv1alpha1.STATE_READY {
			state = zone.Status.State
			msg = zone.Status.Message
		} else {
			if e.Status.RootZone != root.Name || len(e.Status.EffectiveDomainNames) == 0 {
				state = "Pending"
				msg = "waiting for server"
			}
		}
	}
	e.Status.Message = msg
	e.Status.State = state
	return nil
}

func (r *ReconcileRequest) UpdateStatus() reconcile.Problem {
	r.Info("update status '{{state}}' '{{message}}'", "state", r.Object.Status.State, "message", r.Object.Status.Message)
	return r.DefaultReconcileRequest.UpdateStatus()
}

func (r *ReconcileRequest) responsibleForEntry() (*Responsibility, reconcile.Problem) {
	var resp Responsibility

	hist := []string{}
	path := ""
	n := objutils.RefObjectKeyFor(r.Object, r.Object.Spec.ZoneRef)
	for {
		var zone corednsv1alpha1.HostedZone
		path = path + "/" + n.Name
		err := r.Get(r, n, &zone)
		if err != nil {
			if errors.IsNotFound(err) {
				resp.Error = fmt.Sprintf("zone %q not found", n.Name)
				return &resp, nil
			}
			return nil, reconcile.Failed(err)
		}
		if resp.Zone == nil {
			resp.Zone = &zone
		}
		hist = append(hist, n.Name)

		if zone.Spec.ParentRef == "" {
			resp.Root = &zone
			return &resp, nil
		}

		if slices.Contains(hist, zone.Spec.ParentRef) {
			resp.Error = fmt.Sprintf("reference cycle %v", path+"/"+zone.Spec.ParentRef)
			return &resp, nil
		}
		n = objutils.RefObjectKeyFor(&zone, zone.Spec.ParentRef)
	}
}

func (r *ReconcileRequest) Validate() error {
	var err error

	e := r.Object
	if len(e.Spec.DNSNames) == 0 {
		err = fmt.Errorf("no DNS names specified")
	}
	for _, n := range e.Spec.DNSNames {
		_ = n
		//  TODO: validate DNS names
	}

	for _, ips := range e.Spec.A {
		ip := net.ParseIP(ips)
		if ip == nil || ip.To4() == nil {
			err = errors2.Join(err, fmt.Errorf("invalid ipv4 address %q", ips))
		}
	}

	for _, ips := range e.Spec.AAAA {
		ip := net.ParseIP(ips)
		if ip == nil || ip.To4() != nil {
			err = errors2.Join(err, fmt.Errorf("invalid ipv6 address %q", ips))
		}
	}

	if len(e.Spec.CNAME) > 0 {
		// TODO: validate cname
	}

	if len(e.Spec.A) == 0 && len(e.Spec.AAAA) == 0 && len(e.Spec.CNAME) == 0 && len(e.Spec.TXT) == 0 && len(e.Spec.NS) == 0 && (e.Spec.SRV == nil || len(e.Spec.SRV.Records) == 0) {
		err = errors2.Join(err, fmt.Errorf("no record defined"))
	}
	if e.Spec.SRV != nil {
		if len(e.Spec.SRV.Records) != 0 && len(e.Spec.SRV.Service) == 0 {
			err = errors2.Join(err, fmt.Errorf("service name required for SRV record"))
		}
		for i, r := range e.Spec.SRV.Records {
			if r.Protocol != "TCP" && r.Protocol != "UDP" {
				err = errors2.Join(err, fmt.Errorf("invalid protocol %q for SRV record %d", r.Protocol, i))
			}
			if r.Port <= 0 {
				err = errors2.Join(err, fmt.Errorf("invalid port for SRV record %d", i))
			}
			if len(r.Host) == 0 {
				err = errors2.Join(err, fmt.Errorf("host missing for SRV record %d", i))
			}
		}
	}
	return err
}

func (r *ReconcileRequest) SetStatusCondition(condition metav1.Condition) bool {
	if condition.ObservedGeneration == 0 {
		condition.ObservedGeneration = r.Object.GetGeneration()
	}
	return meta.SetStatusCondition(&r.Object.Status.Conditions, condition)
}

/////////////////////////////////////////////////////////////////////////////

type Responsibility struct {
	Zone  *corednsv1alpha1.HostedZone
	Root  *corednsv1alpha1.HostedZone
	Error string
}

func (r *Responsibility) String() string {
	s := ""
	if r.Zone != nil {
		s += " zone=" + r.Zone.Name + ", "
	}
	if r.Root != nil {
		s += " root " + r.Root.Name + ", "
	}
	if r.Error == "" {
		return s + "Ok"
	}
	return s + "Error:" + r.Error
}
