package servercomp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mandelsoft/goutils/generics"
	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/cluster/clustercontext"
	"github.com/mandelsoft/kubecrtutils/component"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/api/server/v1"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	"github.com/mandelsoft/logging"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Component struct {
	logging.Logger
	comp component.Component
	*WebServer

	indexByName cacheindex.TypedIndex[corednsv1alpha1.CoreDNSEntry]
	indexByIP   cacheindex.TypedIndex[corednsv1alpha1.CoreDNSEntry]
	cluster     cluster.ClusterEquivalent
	model       *zonemodel.Model
}

var (
	_ manager.Runnable                  = (*Component)(nil)
	_ component.ComponentImplementation = (*Component)(nil)
	_ zonemodel.Index                   = (*Component)(nil)
)

func (c *Component) GetComponent() component.Component {
	return c.comp
}

func (c *Component) GetModel() *zonemodel.Model {
	return c.model
}

func (c *Component) LookupRelativeDomainName(src zonemodel.Source, zk zonemodel.ZoneKey, rel string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	c.Info("lookup entries for zone {{key}} rdn {{rdn}}", "key", zk, "rdn", rel)
	return c.indexByName.GetTyped(clustercontext.WithCluster(context.Background(), generics.Cast[cluster.Cluster](src)), zk.Namespace, IndexKey(zk, rel))
}

func (c *Component) LookupIP(src zonemodel.Source, zk zonemodel.ZoneKey, ip string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	c.Info("lookup entries for zone {{key}} ip {{ip}}", "key", zk, "ip", ip)
	return c.indexByIP.GetTyped(clustercontext.WithCluster(context.Background(), generics.Cast[cluster.Cluster](src)), zk.Namespace, IndexKey(zk, ip))
}

func IndexKey(zk zonemodel.ZoneKey, rdn string) string {
	return fmt.Sprintf("%s/%s/%s", zk.Namespace, zk.Name, rdn)
}

func (c *Component) SendError(err error, w http.ResponseWriter, statusCode int) {
	var e = v1.Error{
		Error: err.Error(),
	}
	w.WriteHeader(statusCode)
	d, _ := json.Marshal(&e)
	_, err = io.Copy(w, bytes.NewReader(d))
	if err != nil {
		c.logger.Error("%s", err.Error())
	}
}

func CopyR(list []string, tgt *[]string) {
	if len(list) != 0 {
		*tgt = append(*tgt, list...)
	}
}
