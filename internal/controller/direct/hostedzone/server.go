package hostedzone

import (
	"fmt"

	"github.com/mandelsoft/kubecrtutils/controller/controllerutils"
	"github.com/mandelsoft/kubedns/pkg/render"
	"github.com/spf13/pflag"
	"sigs.k8s.io/yaml"
)

type ServerModeFactory controllerutils.Factory[*Options, ServerMode]

var ServerModes = controllerutils.NewRegistry[*Options, ServerMode]("server-mode", "mode for primary DNS server")

type ServerMode interface {
	GetName() string
	ImageName() string
	RequireDataplaneAccess() bool
	ExtendValues(map[string]interface{}) error
}

////////////////////////////////////////////////////////////////////////////////

type serverFactorySupport struct {
	image string
	name  string
	desc  string
}

func newServerFactorySupport(name, image, desc string) *serverFactorySupport {
	return &serverFactorySupport{image: image, name: name, desc: desc}
}

func (s *serverFactorySupport) GetName() string {
	return s.name
}

func (s *serverFactorySupport) Description() string {
	return s.desc
}

func (s *serverFactorySupport) ImageName() string {
	return s.image
}

func (s *serverFactorySupport) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&s.image, "image-"+s.name, "", s.image, s.desc)
}

////////////////////////////////////////////////////////////////////////////////

type servermodeSupport struct {
	name      string
	manifests map[string][]byte
	image     string
}

func newServerModeSupport(n string, d string, im string) (*servermodeSupport, error) {
	if im == "" {
		return nil, fmt.Errorf("image required")
	}
	m, err := GetManifestsFromDir(d)
	if err != nil {
		return nil, err
	}
	return &servermodeSupport{n, m, im}, nil
}

func (s *servermodeSupport) GetName() string {
	return s.name
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
