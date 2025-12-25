package hostedzone

import (
	"context"
	"fmt"

	"github.com/mandelsoft/logging"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const BASE = "dns-service"

type ReconcileContext struct {
	logging.Logger
	context.Context
	client.ObjectKey
	DataPlaneURL string
	Platform     string

	Simulate bool
}

func NewReconcileContext(ctx context.Context, log logging.Logger, server string, platform string, key client.ObjectKey) ReconcileContext {
	return ReconcileContext{Logger: log, Context: ctx, ObjectKey: key, DataPlaneURL: server, Platform: platform}
}

func (r *ReconcileContext) Values(m Mode) (map[string]interface{}, error) {
	accname := fmt.Sprintf("%s", BASE)
	depname := m.RuntimeDeploymentName(r.ObjectKey)

	access := map[string]interface{}{
		"token":  "",
		"cadata": "",
	}

	values := map[string]interface{}{
		"dataplane": map[string]interface{}{
			"namespace": r.Namespace,
			"server":    r.DataPlaneURL,
			"name":      accname,
			"label":     accname,
			"access":    access,
			"zone":      r.Name,
		},
		"runtime": map[string]interface{}{
			"platform":  r.Platform,
			"namespace": m.RuntimeNamespace(r.ObjectKey),
			"secret": map[string]interface{}{
				"name": m.RuntimeSecretName(r.ObjectKey),
			},
			"name":     depname,
			"label":    depname,
			"replicas": 1,
		},
	}
	tmp, repeat := m.AccessValues(*r, accname)
	if repeat != nil {
		r.Logger.Info("no access info -> must repeat")
	}
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
