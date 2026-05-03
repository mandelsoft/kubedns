package hostedzone

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubecrtutils/controller/controllerutils"
	"github.com/spf13/pflag"
)

const SERVERMODE_DATAPLANE = "dataplane" // use dataplane access for promary DNS server

const dataplaneServerDir = "assets/server/dataplane"

func init() {
	ServerModes.Register(SERVERMODE_DATAPLANE, NewDataplaneServerFactory())
}

var (
	_ flagutils.Options              = (*dataplaneServerFactory)(nil)
	_ controllerutils.DefaultElement = (*dataplaneServerFactory)(nil)
)

type dataplaneServerFactory struct {
	serverFactorySupport
}

func NewDataplaneServerFactory() ServerModeFactory {
	return &dataplaneServerFactory{
		*newServerFactorySupport(SERVERMODE_DATAPLANE, "mandelsoft/coredns:latest", "primary DNS server accessing dataplane"),
	}
}

func (f *dataplaneServerFactory) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&f.image, "image-kubedyndns", f.image, "image for primary dns server using REST API")
}

func (f *dataplaneServerFactory) Create(ctx context.Context, cfg *Options) (ServerMode, error) {
	b, err := newServerModeSupport(f.name, dataplaneServerDir, f.image)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", f.name, err)
	}
	return &dataplaneServer{*b}, nil
}

func (f *dataplaneServerFactory) IsDefault() bool {
	return true
}

////////////////////////////////////////////////////////////////////////////////

type dataplaneServer struct {
	servermodeSupport
}

func (s *dataplaneServer) RequireDataplaneAccess() bool {
	return true
}

func (s *dataplaneServer) ExtendValues(values map[string]interface{}) error {
	m := values["dataplane"].(map[string]interface{})
	m["required"] = true
	return s.servermodeSupport.AddValues(values)
}
