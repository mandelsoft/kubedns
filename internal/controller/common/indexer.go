package common

import (
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
)

const IndexKeyZoneParent = "hostedzone.parent"

func ParentIndexer(o *corednsv1alpha1.HostedZone) []string {
	if o.Spec.ParentRef == "" {
		return nil
	}
	return []string{o.Spec.ParentRef}
}

const IndexKeyEntryZone = "corednsentry.zone"

func ZoneIndexer(res *corednsv1alpha1.CoreDNSEntry) []string {
	if res.Spec.ZoneRef == "" {
		return nil
	}
	return []string{res.Spec.ZoneRef}
}
