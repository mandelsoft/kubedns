package servercomp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mandelsoft/goutils/sliceutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/api/server/v1"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
)

const PATH_ZONES = "/api/v1/zones/"

func prepareInfo(info *zonemodel.Info) (*v1.Info, error) {
	var i v1.Info
	i.Names = info.EntryNames
	if info.Zone != nil {
		var zone corednsv1alpha1.HostedZone
		err := info.Zone.GetSource().(cluster.Cluster).Get(context.Background(), info.Zone.GetKey().NamespacedName, &zone)
		if err != nil {
			return nil, err
		}
		i.Zone.NameServers = zone.Status.NameServers
		i.Zone.EMail = zone.Spec.EMail
		i.Zone.MinimumTTL = zone.Spec.MinimumTTL
		i.Zone.Expire = zone.Spec.Expire
		i.Zone.Refresh = zone.Spec.Refresh
		i.Zone.Retry = zone.Spec.Retry
		i.Zone.SerialId = info.Zone.GetSerialId()
	}
	i.Zone.Names = info.ZoneNames
	for _, r := range info.Entries {
		CopyR(r.Spec.NS, &i.Records.NS)
		CopyR(r.Spec.A, &i.Records.A)
		CopyR(r.Spec.AAAA, &i.Records.AAAA)
		CopyR(r.Spec.TXT, &i.Records.TXT)
		if i.Records.CNAME == "" {
			i.Records.CNAME = r.Spec.CNAME
		}
		if r.Spec.SRV != nil {
			i.Records.SRV = append(i.Records.SRV, *r.Spec.SRV)
		}
		i.Records.TTL = r.Spec.TTL
	}
	return &i, nil
}

func (c *Component) handleNames(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, PATH_ZONES)

	fields := strings.Split(p, "/")
	if len(fields) != 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	zk := zonemodel.NewZoneKey(fields[0], fields[1], fields[2])
	c.Info("request {{fqdn}} for zone {{key}}", "key", zk, "fqdn", fields[3])
	z := c.model.GetZone(zk)
	if z == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	info, err := z.ResolveFQDN(c.logger, fields[3])
	if err != nil {
		c.SendError(err, w, http.StatusInternalServerError)
		return
	}

	if info == nil {
		c.SendError(fmt.Errorf("name not in domain"), w, http.StatusBadRequest)
		return
	}

	i, err := prepareInfo(info)
	if err == nil {
		answer := &v1.Answer{Infos: sliceutils.AsSlice(i)}
		d, _ := json.Marshal(answer)
		_, err = io.Copy(w, bytes.NewReader(d))
		if err != nil {
			c.logger.Error("%s", err.Error())
		}
	} else {
		c.SendError(err, w, http.StatusInternalServerError)
	}
}
