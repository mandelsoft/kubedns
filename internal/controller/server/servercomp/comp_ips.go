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
	"github.com/mandelsoft/kubedns/internal/controller/server/zonemodel"
	v1 "github.com/mandelsoft/kubedns/internal/controller/server/zonemodel/api/v1"
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
	var answers []*v1.Answer

	for _, info := range infos {
		var answer v1.Answer
		answer.Names = info.EntryNames
		if info.Zone != nil {
			var zone corednsv1alpha1.HostedZone
			err := info.Zone.GetSource().(cluster.Cluster).Get(context.Background(), info.Zone.GetKey().NamespacedName, &zone)
			if err != nil {
				c.SendError(err, w, http.StatusInternalServerError)
				return
			}
			answer.Zone.Names = info.ZoneNames
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
		answers = append(answers, &answer)
	}

	d, _ := json.Marshal(answers)
	_, err = io.Copy(w, bytes.NewReader(d))
	if err != nil {
		c.logger.Error("%s", err.Error())
	}
}
