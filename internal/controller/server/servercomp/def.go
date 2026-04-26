package servercomp

import (
	"context"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/component"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	"github.com/spf13/pflag"
)

const INDEX_ZONENAMES = "zonednsnames"
const INDEX_ZONEIPS = "zoneips"

func Server() component.Definition {
	return component.Define(server.Component, NewFactory()).
		UseCluster(server.CLUSTER).
		AddForeignIndex(cacheindex.DefineByFactory[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](INDEX_ZONENAMES, server.CLUSTER, indexerFactoryNames)).
		AddForeignIndex(cacheindex.DefineByFactory[*corednsv1alpha1.CoreDNSEntry, corednsv1alpha1.CoreDNSEntry](INDEX_ZONEIPS, server.CLUSTER, indexerFactoryIPs))
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
		Logger:      comp,
		comp:        comp,
		cluster:     comp.GetCluster(server.CLUSTER),
		indexByName: cacheindex.GetTypedIndex[corednsv1alpha1.CoreDNSEntry](comp.GetIndices(), INDEX_ZONENAMES),
		indexByIP:   cacheindex.GetTypedIndex[corednsv1alpha1.CoreDNSEntry](comp.GetIndices(), INDEX_ZONEIPS),
		WebServer:   NewWebServer(f.port, comp),
	}
	c.model = zonemodel.New(c)
	c.WebServer.AddEndpoint(PATH_ZONES, c.handleNames)
	c.WebServer.AddEndpoint(PATH_IPS, c.handleIPs)
	return c, nil
}
