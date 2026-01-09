package entry

import (
	"context"
	errors2 "errors"
	"fmt"
	"net"
	"slices"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/objutils"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ReconcilationRequest struct {
	common.ReconcilationRequest[*corednsv1alpha1.CoreDNSEntry, *CoreDNSEntryReconciler]
}

func NewRequest(ctx context.Context, r *CoreDNSEntryReconciler, log logging.Logger, obj *corednsv1alpha1.CoreDNSEntry) *ReconcilationRequest {
	var req ReconcilationRequest

	req.Context = ctx
	req.Logger = log
	req.ObjectKey = client.ObjectKeyFromObject(obj) // no deletion handling
	req.Instance = obj
	req.Reconciler = r
	return &req
}

func (r *ReconcilationRequest) handleObject() reconcile.Problem {
	baseerr := r.Validate()
	e := r.Instance

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
	mod := false
	if baseerr != nil {
		mod = r.SetStatusCondition(metav1.Condition{
			Type:    corednsv1alpha1.ValidationConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  corednsv1alpha1.ReasonConfigurarationInvalid,
			Message: baseerr.Error(),
		})
	} else {
		mod = r.SetStatusCondition(metav1.Condition{
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
	if e.Status.Message != msg {
		e.Status.Message = msg
		mod = true
	}
	if e.Status.State != state {
		e.Status.State = state
		mod = true
	}

	var err error
	if mod {
		r.Info("update status to {{state}}", "state", e.Status.State, "message", e.Status.Message)
		err = r.Reconciler.DataPlane.Status().Update(r, e)
	}
	return reconcile.TemporaryProblem(err)
}

func (r *ReconcilationRequest) responsibleForEntry() (*Responsibility, reconcile.Problem) {
	var resp Responsibility

	hist := []string{}
	path := ""
	n := objutils.RefObjectKeyFor(r.Instance, r.Instance.Spec.ZoneRef)
	for {
		var zone corednsv1alpha1.HostedZone
		path = path + "/" + n.Name
		err := r.Reconciler.DataPlane.Get(r, n, &zone)
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

func (r *ReconcilationRequest) Validate() error {
	var err error

	e := r.Instance
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

func (r *ReconcilationRequest) SetStatusCondition(condition metav1.Condition) bool {
	if condition.ObservedGeneration == 0 {
		condition.ObservedGeneration = r.Instance.GetGeneration()
	}
	return meta.SetStatusCondition(&r.Instance.Status.Conditions, condition)
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
