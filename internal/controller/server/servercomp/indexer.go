package servercomp

import (
	"context"
	"slices"

	"github.com/mandelsoft/goutils/sliceutils"
	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/component"
	"github.com/mandelsoft/kubecrtutils/types"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	"github.com/mandelsoft/logging"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func indexerFactoryNames(ctx context.Context, logger logging.Logger, set types.Clusters) (cacheindex.TypedIndexerFunc[*corednsv1alpha1.CoreDNSEntry], error) {
	opts := component.DefinitionFromContext(ctx).GetOptions().(*Factory)

	return func(obj *corednsv1alpha1.CoreDNSEntry) []string {
		if obj.Spec.ZoneRef == "" {
			logger.Info("omit unassigned entry from index", "entry", client.ObjectKeyFromObject(obj))

			return nil
		}
		c := meta.FindStatusCondition(obj.Status.Conditions, corednsv1alpha1.ServerConditionType)
		if opts.Options.Slave {
			if c == nil || c.Status == metav1.ConditionFalse {
				if c == nil {
					logger.Info("omit unvalidated entry from index", "entry", client.ObjectKeyFromObject(obj))
				} else {
					logger.Info("omit invalid entry from index: {{problem}}", "problem", c.Message, "entry", client.ObjectKeyFromObject(obj))
				}
				return nil
			}
		} else {
			if obj.Status.State != "Ok" && obj.Status.State != "Ready" {
				logger.Info("omit invalid entry from index: {{problem}}", "problem", obj.Status.Message, "entry", client.ObjectKeyFromObject(obj))
			}
		}
		key := obj.Namespace + "/" + obj.Spec.ZoneRef
		names := sliceutils.Transform(obj.Spec.DNSNames, zonemodel.Rdn)
		for _, n := range names {
			logger.Info("cache {{dnsname}}[{{entry}}] -> {{key}}\n", "dnsname", n, "entry", client.ObjectKeyFromObject(obj), "key", key)
		}
		return sliceutils.Transform(names, func(item string) string { return key + "/" + item })
	}, nil
}

func indexerFactoryIPs(ctx context.Context, logger logging.Logger, set types.Clusters) (cacheindex.TypedIndexerFunc[*corednsv1alpha1.CoreDNSEntry], error) {
	opts := component.DefinitionFromContext(ctx).GetOptions().(*Factory)

	return func(obj *corednsv1alpha1.CoreDNSEntry) []string {
		if obj.Spec.ZoneRef == "" {
			logger.Info("omit unassigned entry from index", "entry", client.ObjectKeyFromObject(obj))

			return nil
		}
		c := meta.FindStatusCondition(obj.Status.Conditions, corednsv1alpha1.ServerConditionType)
		if opts.Options.Slave {
			if c == nil || c.Status == metav1.ConditionFalse {
				if c == nil {
					logger.Info("omit unvalidated entry from index", "entry", client.ObjectKeyFromObject(obj))
				} else {
					logger.Info("omit invalid entry from index: {{problem}}", "problem", c.Message, "entry", client.ObjectKeyFromObject(obj))
				}
				return nil
			}
		} else {
			if obj.Status.State != "Ok" && obj.Status.State != "Ready" {
				logger.Info("omit invalid entry from index: {{problem}}", "problem", obj.Status.Message, "entry", client.ObjectKeyFromObject(obj))
			}
		}
		key := obj.Namespace + "/" + obj.Spec.ZoneRef
		ips := append(slices.Clone(obj.Spec.A), obj.Spec.AAAA...)
		if len(ips) > 0 {
			for _, n := range ips {
				logger.Info("cache {{ip}}[{{entry}}] -> {{key}}\n", "ip", n, "entry", client.ObjectKeyFromObject(obj), "key", key)
			}
			return sliceutils.Transform(ips, func(item string) string { return key + "/" + item })
		}
		return nil
	}, nil
}
