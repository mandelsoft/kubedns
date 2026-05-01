package hostedzone

import (
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils"
	"github.com/mandelsoft/kubedns/pkg/render"
	"sigs.k8s.io/yaml"
)

type ServerModeFactory = controllerutils.FactoryFunc[*Options, ServerMode]

var ServerModes = controllerutils.NewRegistry[*Options, ServerMode]("Server mode")

type ServerMode interface {
	ImageName() string
	RequireDataplaneAccess() bool
	AddValues(map[string]interface{}) error
}

////////////////////////////////////////////////////////////////////////////////

type servermodeSupport struct {
	manifests map[string][]byte
	image     string
}

func newServerModeSupport(d string) (*servermodeSupport, error) {
	m, err := GetManifestsFromDir(d)
	if err != nil {
		return nil, err
	}
	return &servermodeSupport{m, ""}, nil
}

func (s *servermodeSupport) ImageName() string {
	return s.image
}

func (s *servermodeSupport) AddValues(values map[string]interface{}) error {
	m := values["runtime"].(map[string]interface{})
	m["image"] = s.image

	rendered, err := render.Render(s.manifests, values, "corefile")
	if err != nil {
		return err
	}

	var val string
	err = yaml.Unmarshal(rendered.Other["corefile.yaml"], &val)
	if err != nil {
		return err
	}
	m["corefile"] = val
	return nil
}
