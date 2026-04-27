package servercomp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mandelsoft/kubecrtutils/cluster"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/kubedns/api/server/v1"
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
)

const PATH_IPS = "/api/v1/ips/"

func (c *Component) handleIPs(w http.ResponseWriter, r *http.Request) {
	p := strings.TrimPrefix(r.URL.Path, PATH_IPS)

	fields := strings.Split(p, "/")
	if len(fields) != 4 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	zk := zonemodel.NewZoneKey(fields[0], fields[1], fields[2])
	c.Info("request {{ip}} for zone {{key}}", "key", zk, "ip", fields[3])
	z := c.model.GetZone(zk)
	if z == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	infos, err := z.ResolveIP(c.logger, fields[3])
	if err != nil {
		c.SendError(err, w, http.StatusInternalServerError)
		return
	}

	if len(infos) == 0 {
		c.SendError(fmt.Errorf("name not in domain"), w, http.StatusBadRequest)
		return
	}
	var answer v1.Answer

	for _, i := range infos {
		var info v1.Info
		info.Names = i.EntryNames
		if i.Zone != nil {
			var zone corednsv1alpha1.HostedZone
			err := i.Zone.GetSource().(cluster.Cluster).Get(context.Background(), i.Zone.GetKey().NamespacedName, &zone)
			if err != nil {
				c.SendError(err, w, http.StatusInternalServerError)
				return
			}
			info.Zone.Names = i.ZoneNames
			info.Zone.NameServers = zone.Status.NameServers
			info.Zone.EMail = zone.Spec.EMail
			info.Zone.MinimumTTL = zone.Spec.MinimumTTL
			info.Zone.Expire = zone.Spec.Expire
			info.Zone.Refresh = zone.Spec.Refresh
		}
		for _, r := range i.Entries {
			CopyR(r.Spec.NS, &info.Records.NS)
			CopyR(r.Spec.A, &info.Records.A)
			CopyR(r.Spec.AAAA, &info.Records.AAAA)
			CopyR(r.Spec.TXT, &info.Records.TXT)
			if info.Records.CNAME == "" {
				info.Records.CNAME = r.Spec.CNAME
			}
			if r.Spec.SRV != nil {
				info.Records.SRV = append(info.Records.SRV, *r.Spec.SRV)
			}
		}
		answer.Infos = append(answer.Infos, info)
	}

	d, _ := json.Marshal(&answer)
	_, err = io.Copy(w, bytes.NewReader(d))
	if err != nil {
		c.logger.Error("%s", err.Error())
	}
}
