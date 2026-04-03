package component

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mandelsoft/kubecrtutils/cacheindex"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubecrtutils/component"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Component struct {
	name string
	*WebServer

	index   cacheindex.TypedIndex[corednsv1alpha1.CoreDNSEntry]
	cluster cluster.ClusterEquivalent
	model   *zonemodel.Model
}

var _ manager.Runnable = (*Component)(nil)

var _ zonemodel.Index = (*Component)(nil)

func (c *Component) GetName() string {
	return c.name
}

func (c *Component) GetModel() *zonemodel.Model {
	return c.model
}

func (c *Component) LookupRelativeDomainName(ctx context.Context, zk zonemodel.ZoneKey, rel string) ([]corednsv1alpha1.CoreDNSEntry, error) {
	return c.index.GetTyped(ctx, zk.Namespace, IndexKey(zk, rel))
}

var _ component.Component = (*Component)(nil)

func IndexKey(zk zonemodel.ZoneKey, rdn string) string {
	return fmt.Sprintf("%s/%s/%s", zk.Namespace, zk.Name, rdn)
}

func (c *Component) handle(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, "/api/v1/zones/")

	fields := strings.Split(p, "/")
	if len(fields) != 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	zk := zonemodel.NewZoneKey(fields[0], fields[1], fields[2])
	z := c.model.GetZone(zk)
	if z == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	info, err := z.Resolve(c.logger, fields[3])
	if err != nil {
		c.SendError(err, w)
		return
	}

	var answer v1.Answer

	answer.Name = info.EntryName
	if info.Zone != nil {
		var zone corednsv1alpha1.HostedZone
		err := info.Zone.GetSource().(cluster.Cluster).Get(context.Background(), info.Zone.GetKey().NamespacedName, &zone)
		if err != nil {
			c.SendError(err, w)
			return
		}
		answer.Zone.Name = info.ZoneName
		answer.Zone.NameServers = zone.Status.NameServers
		answer.Zone.EMail = zone.Spec.EMail
		answer.Zone.MinimumTTL = zone.Spec.MinimumTTL
		answer.Zone.Expire = zone.Spec.Expire
		answer.Zone.Refresh = zone.Spec.Refresh
	}
	for _, r := range info.Entries {
		CopyR(r.Spec.NS, &answer.Records.NS)
		CopyR(r.Spec.A, &answer.Records.A)
		CopyR(r.Spec.AAAA, &answer.Records.AAAA)
		CopyR(r.Spec.TXT, &answer.Records.TXT)
		if answer.Records.CNAME == "" {
			answer.Records.CNAME = r.Spec.CNAME
		}
		if r.Spec.SRV != nil {
			answer.Records.SRV = append(answer.Records.SRV, *r.Spec.SRV)
		}
	}

	d, _ := json.Marshal(&answer)
	_, err = io.Copy(w, bytes.NewReader(d))
	if err != nil {
		c.logger.Error("%s", err.Error())
	}
	w.WriteHeader(http.StatusOK)
}

func (c *Component) SendError(err error, w http.ResponseWriter) {
	var e = v1.Error{
		Error: err.Error(),
	}
	d, _ := json.Marshal(&e)
	_, err = io.Copy(w, bytes.NewReader(d))
	if err != nil {
		c.logger.Error("%s", err.Error())
	}
	w.WriteHeader(http.StatusInternalServerError)
}

func CopyR(list []string, tgt *[]string) {
	if len(list) != 0 {
		*tgt = append(*tgt, list...)
	}
}
