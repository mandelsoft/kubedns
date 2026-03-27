package common

import (
	"context"
	errors2 "errors"
	"fmt"
	"net"

	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils/reconcile"
	"github.com/mandelsoft/kubecrtutils/objutils"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/logging"
)

func ValidateEntryData(e *corednsv1alpha1.CoreDNSEntry) error {
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

func ValidateEntry(ctx context.Context, c cluster.Cluster, logger logging.Logger, e *corednsv1alpha1.CoreDNSEntry) (*Responsibility, error, reconcile.Problem) {
	var resp *Responsibility
	var ok bool
	var prob reconcile.Problem

	baseerr := ValidateEntryData(e)

	if e.Spec.ZoneRef != "" {
		resp, ok, prob = GetRootInfoForEntry(ctx, c, logger, objutils.RefObjectKeyFor(e, e.Spec.ZoneRef))
		if !ok {
			// temporary problem
			return nil, baseerr, prob
		}
		if prob != nil {
			logger.Info("problem determining responsibility", "problem", prob.Message)
			baseerr = errors2.Join(fmt.Errorf("zone error: %s", prob.Message()), baseerr)
		} else {
			logger.Info("responsibility info: {{responsible}}", "responsible", resp)
		}
	} else {
		baseerr = errors2.Join(fmt.Errorf("zone reference required"), baseerr)
	}

	if baseerr != nil {
		logger.Info("found problem {{error}}", "error", baseerr)
	}
	return resp, baseerr, nil
}
