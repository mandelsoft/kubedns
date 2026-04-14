package servercomp

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/goutils/sliceutils"
	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/component"
	"github.com/mandelsoft/kubecrtutils/types"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	"github.com/mandelsoft/logging"
	"github.com/spf13/pflag"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const INDEX = "zonednsnames"

func Server() component.Definition {
	return component.Define(server.Component, NewFactory()).
		UseCluster(server.CLUSTER).
		AddForeignIndex(cacheindex.DefineByFactory[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](INDEX, server.CLUSTER, indexerFactory))
}

func indexerFactory(ctx context.Context, logger logging.Logger, set types.Clusters) (cacheindex.TypedIndexerFunc[*corednsv1alpha1.CoreDNSEntry], error) {
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

type Factory struct {
	flagutils.OptionsRef[*server.Options]
	port string
}

func NewFactory() *Factory {
	return &Factory{OptionsRef: *flagutils.NewOptionsRef[*server.Options](server.NewOptions)}
}

func (f *Factory) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&f.port, "dnsapi", "", ":8085", "The port on which to run the DNS API server.")
}

func (f *Factory) CreateComponent(ctx context.Context, comp component.Component) (component.ComponentImplementation, error) {
	c := &Component{
		Logger:    comp,
		comp:      comp,
		cluster:   comp.GetCluster(server.CLUSTER),
		index:     cacheindex.GetTypedIndex[corednsv1alpha1.CoreDNSEntry](comp.GetIndices(), INDEX),
		WebServer: NewWebServer(f.port, comp),
	}
	c.model = zonemodel.New(c)
	c.WebServer.AddEndpoint("/api/v1/zones/", c.handle)
	return c, nil
}
