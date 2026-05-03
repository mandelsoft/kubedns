package hostedzone

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mandelsoft/goutils/sliceutils"
	"github.com/mandelsoft/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/mandelsoft/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const BASE = "dns-service"

type ReconcileContext interface {
	context.Context
	logging.Logger
	cluster.Cluster

	IsSimulate() bool
	GetPlatform() string
	GetKey() client.ObjectKey
	GetObject() *v1alpha1.HostedZone

	Values(m Mode, deleting bool) (map[string]interface{}, error)
}

type reconcileContext struct {
	logging.Logger
	context.Context
	cluster.Cluster
	client.ObjectKey
	Platform string

	apiServer string
	clusterId string

	Simulate bool

	Object *v1alpha1.HostedZone
}

func NewReconcileContext(ctx context.Context, log logging.Logger, server string, clusterId string, platform string, obj *v1alpha1.HostedZone) reconcileContext {
	return reconcileContext{Logger: log, Context: ctx, apiServer: server, clusterId: clusterId, Object: obj, Platform: platform, ObjectKey: client.ObjectKeyFromObject(obj)}
}

func (c reconcileContext) GetId() string {
	if c.clusterId != "" {
		return c.clusterId
	}
	return c.Cluster.GetId()
}

func (c reconcileContext) GetObject() *v1alpha1.HostedZone {
	return c.Object
}

func (c reconcileContext) IsSimulate() bool {
	return c.Simulate
}

func (c reconcileContext) GetPlatform() string {
	return c.Platform
}

func (c reconcileContext) GetKey() client.ObjectKey {
	return c.ObjectKey
}

func (c reconcileContext) GetAPIServerURL() (*url.URL, error) {
	if c.apiServer == "" {
		return c.Cluster.GetAPIServerURL()
	}
	return url.Parse(c.apiServer)
}

func (c reconcileContext) Values(m Mode, deleting bool) (map[string]interface{}, error) {
	return Values(c, m, deleting)
}

func Values(c ReconcileContext, m Mode, deleting bool) (map[string]interface{}, error) {
	key := c.GetKey()
	accname := fmt.Sprintf("%s", BASE)
	depname := m.RuntimeDeploymentName(c, key)

	access := map[string]interface{}{
		"token":  "",
		"cadata": "",
	}

	u, err := c.GetAPIServerURL()
	if err != nil {
		return nil, err
	}
	values := map[string]interface{}{
		"apex": sliceutils.Convert[any](c.GetObject().Spec.DomainNames),
		"dataplane": map[string]interface{}{
			"namespace": key.Namespace,
			"server":    u.String(),
			"name":      accname,
			"label":     accname,
			"access":    access,
			"zone":      key.Name,
			"cluster":   c.GetId(),
		},
		"runtime": map[string]interface{}{
			"platform":  c.GetPlatform(),
			"namespace": m.RuntimeNamespace(c, key),
			"secret": map[string]interface{}{
				"name": m.RuntimeSecretName(c, key),
			},
			"name":     depname,
			"label":    depname,
			"replicas": 1,
		},
	}
	tmp, repeat := m.AccessValues(c, accname, deleting)
	mergeValues(access, tmp)
	err = m.ServerMode().ExtendValues(values)
	if err != nil {
		panic(fmt.Errorf("cannot render server mode values: %w", err))
	}
	data, _ := json.Marshal(values)
	c.Info("values {{values}}", "values", string(data))
	return values, repeat
}

func mergeValues(dst, src map[string]interface{}) map[string]interface{} {
	if dst == nil {
		dst = make(map[string]interface{})
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
