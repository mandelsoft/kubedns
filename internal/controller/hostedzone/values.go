package hostedzone

import (
	"context"
	"fmt"

	"github.com/mandelsoft/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const BASE = "dns-service"

type ReconcileContext interface {
	context.Context
	logging.Logger

	IsSimulate() bool
	GetPlatform() string
	GetDataPlaneURL() string
	GetKey() client.ObjectKey

	Values(m Mode, deleting bool) (map[string]interface{}, error)
}

type reconcileContext struct {
	logging.Logger
	context.Context
	client.ObjectKey
	DataPlaneURL string
	Platform     string

	Simulate bool
}

func NewReconcileContext(ctx context.Context, log logging.Logger, server string, platform string, key client.ObjectKey) reconcileContext {
	return reconcileContext{Logger: log, Context: ctx, ObjectKey: key, DataPlaneURL: server, Platform: platform}
}

func (c reconcileContext) IsSimulate() bool {
	return c.Simulate
}

func (c reconcileContext) GetPlatform() string {
	return c.Platform
}

func (c reconcileContext) GetDataPlaneURL() string {
	return c.DataPlaneURL
}

func (c reconcileContext) GetKey() client.ObjectKey {
	return c.ObjectKey
}

func (c reconcileContext) Values(m Mode, deleting bool) (map[string]interface{}, error) {
	return Values(c, m, deleting)
}

func Values(c ReconcileContext, m Mode, deleting bool) (map[string]interface{}, error) {
	key := c.GetKey()
	accname := fmt.Sprintf("%s", BASE)
	depname := m.RuntimeDeploymentName(key)

	access := map[string]interface{}{
		"token":  "",
		"cadata": "",
	}

	values := map[string]interface{}{
		"dataplane": map[string]interface{}{
			"namespace": key.Namespace,
			"server":    c.GetDataPlaneURL(),
			"name":      accname,
			"label":     accname,
			"access":    access,
			"zone":      key.Name,
		},
		"runtime": map[string]interface{}{
			"platform":  c.GetPlatform(),
			"namespace": m.RuntimeNamespace(key),
			"secret": map[string]interface{}{
				"name": m.RuntimeSecretName(key),
			},
			"name":     depname,
			"label":    depname,
			"replicas": 1,
		},
	}
	tmp, repeat := m.AccessValues(c, accname, deleting)
	mergeValues(access, tmp)
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
