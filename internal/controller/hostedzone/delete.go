package hostedzone

import (
	"errors"
	"fmt"

	"github.com/mandelsoft/kubedns/pkg/render"
)

func (r *ReconcileRequest) DeleteExternalResources() error {

	values, _ := r.Values(r.reconciler.Mode, true)

	r.Info("rendering manifests to determine objects to be deleted")
	dataplane, runtime, err := render.Render(r.reconciler.Manifests, values)
	if err != nil {
		return err
	}

	r.Info("deleting runtime resources")
	for _, data := range runtime {
		err = errors.Join(err, r.Delete(r.reconciler.Runtime, data))
	}

	r.Info("deleting dataplane resources")
	for _, data := range dataplane {
		err = errors.Join(err, r.Delete(r.reconciler.DataPlane, data))
	}

	if err == nil {
		r.Info("cleanup deployment mode")
		err = r.reconciler.Mode.Cleanup(r.ReconcileContext, values["dataplane"].(map[string]interface{})["name"].(string))
		if err != nil {
			err = fmt.Errorf("error cleanup  mode: %s", err.Error())
		}
	}
	return err
}
