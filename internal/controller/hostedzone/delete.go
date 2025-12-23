package hostedzone

import (
	"errors"

	"github.com/mandelsoft/kubedns/pkg/render"
)

func (r *ReconcileRequest) DeleteExternalResources() error {

	values, _ := r.Values(r.reconciler.Mode)

	dataplane, runtime, err := render.Render(r.reconciler.Manifests, values)
	if err != nil {
		return err
	}

	for _, data := range runtime {
		err = errors.Join(err, r.Delete("runtime", r.reconciler.Runtime, data))
	}

	for _, data := range dataplane {
		err = errors.Join(err, r.Delete("dataplane", r.reconciler.DataPlane, data))
	}

	if err == nil {
		err = r.reconciler.Mode.Cleanup(r.ReconcileContext, values["dataplane"].(map[string]interface{})["name"].(string))
	}
	return err
}
