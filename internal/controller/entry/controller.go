/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package entry

import (
	"context"
	errors2 "errors"
	"fmt"
	"net"
	"slices"

	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/common"
	"github.com/mandelsoft/kubedns/pkg/objutils"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// CoreDNSEntryReconciler reconciles a CoreDNSEntry object
type CoreDNSEntryReconciler struct {
	*common.Reconciler
}

// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=coredns.mandelsoft.org,resources=corednsentries/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the CoreDNSEntry object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *CoreDNSEntryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := Log.WithName(req.String()).WithValues("object", req.NamespacedName)

	var obj corednsv1alpha1.CoreDNSEntry
	err := r.DataPlane.Get(ctx, req.NamespacedName, &obj)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("entry has been deleted")
			err = nil
		}
		return ctrl.Result{}, err
	}

	e := &obj

	baseerr := r.Validate(e)

	var root *corednsv1alpha1.HostedZone

	if e.Spec.ZoneRef != "" {
		resp, err := r.responsibleForEntry(ctx, e)
		if err != nil {
			return ctrl.Result{}, err
		}
		log.Info("responsible {{responsible}}", "responsible", resp)

		if resp.Error != "" {
			baseerr = errors2.Join(fmt.Errorf("zone error: %s", resp.Error), baseerr)
		} else {
			root = resp.Root
		}
	} else {
		baseerr = errors2.Join(fmt.Errorf("zone reference required"), baseerr)
	}

	mod := false
	if baseerr != nil {
		mod = r.SetStatusCondition(e, metav1.Condition{
			Type:    corednsv1alpha1.ValidationConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  corednsv1alpha1.ReasonConfigurarationInvalid,
			Message: baseerr.Error(),
		})
	} else {
		mod = r.SetStatusCondition(e, metav1.Condition{
			Type:    corednsv1alpha1.ValidationConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  corednsv1alpha1.ReasonConfigurarationValid,
			Message: "configuration valid",
		})
	}

	// create summary
	state := "Ready"
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

	if state == "Ready" {
		if root.Status.State != "Ready" {
			root.Status.State = root.Status.State
			root.Status.Message = root.Status.Message
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
	if mod {
		log.Info("update status {{state}}", "state", e.Status.State, "message", e.Status.Message)
		err = r.DataPlane.Status().Update(ctx, e)
	}
	return ctrl.Result{}, err
}

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

func (cntr *CoreDNSEntryReconciler) responsibleForEntry(ctx context.Context, e *corednsv1alpha1.CoreDNSEntry) (*Responsibility, error) {
	var resp Responsibility

	hist := []string{}
	path := ""
	n := objutils.RefObjectKeyFor(e, e.Spec.ZoneRef)
	for {
		var zone corednsv1alpha1.HostedZone
		path = path + "/" + n.Name
		err := cntr.DataPlane.Get(ctx, n, &zone)
		if err != nil {
			if errors.IsNotFound(err) {
				resp.Error = fmt.Sprintf("zone %q not found", n.Name)
				return &resp, nil
			}
			return nil, err
		}
		if resp.Zone == nil {
			resp.Zone = &zone
		}
		hist = append(hist, n.Name)

		if zone.Spec.ParentRef == "" {
			return &resp, nil
		}

		if slices.Contains(hist, zone.Spec.ParentRef) {
			resp.Error = fmt.Sprintf("reference cycle %v", path+"/"+zone.Spec.ParentRef)
			return &resp, nil
		}
		n = objutils.RefObjectKeyFor(&zone, zone.Spec.ParentRef)
	}
}

func (r *CoreDNSEntryReconciler) Validate(e *corednsv1alpha1.CoreDNSEntry) error {
	var err error

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

func (r *CoreDNSEntryReconciler) SetStatusCondition(o *corednsv1alpha1.CoreDNSEntry, condition metav1.Condition) bool {
	if condition.ObservedGeneration == 0 {
		condition.ObservedGeneration = o.GetGeneration()
	}
	return meta.SetStatusCondition(&o.Status.Conditions, condition)
}
