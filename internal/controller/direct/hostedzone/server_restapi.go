package hostedzone

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/spf13/pflag"
)

const SERVERMODE_RESTAPI = "restapi" // use dataplane access for promary DNS server

const restapiServerDir = "assets/server/restapi"

func init() {
	ServerModes.Register(SERVERMODE_RESTAPI, NewRestAPIServerFactory())
}

var (
	_ flagutils.Options = (*restapiServerFactory)(nil)
)

type restapiServerFactory struct {
	serverFactorySupport
	endpoint string
}

func NewRestAPIServerFactory() ServerModeFactory {
	return &restapiServerFactory{
		*newServerFactorySupport(SERVERMODE_RESTAPI, "mandelsoft/restdnyndns-coredns:latest", "primary DNS server accessing REST API"),
		"http://dns-rest-api.dns-service.svc.cluster.local",
	}
}

func (f *restapiServerFactory) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&f.image, "image-restdyndns", f.image, "image for primary DNS server using REST API")
	fs.StringVar(&f.endpoint, "rest-endpoint", f.endpoint, "REST API endpoint for primary DNS sever")
}

func (f *restapiServerFactory) Create(ctx context.Context, cfg *Options) (ServerMode, error) {
	if f.endpoint == "" {
		return nil, fmt.Errorf("no REST API endpoint specified")
	}
	b, err := newServerModeSupport(f.name, restapiServerDir, f.image)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", f.name, err)
	}
	return &restapiServer{*b, f.endpoint}, nil
}

////////////////////////////////////////////////////////////////////////////////

type restapiServer struct {
	servermodeSupport
	endpoint string
}

func (s *restapiServer) RequireDataplaneAccess() bool {
	return false
}

func (s *restapiServer) ExtendValues(values map[string]interface{}) error {
	m := values["runtime"].(map[string]interface{})
	// m["requiredataplane"] = false
	m["restendpoint"] = s.endpoint
	return s.servermodeSupport.AddValues(values)
}
