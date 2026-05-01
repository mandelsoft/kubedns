package hostedzone

import (
	"context"
)

const SERVERMODE_RESTAPI = "restapi" // use dataplane access for promary DNS server

const restapiServerDir = "assets/server/restapi"

func init() {
	ServerModes.Register(SERVERMODE_RESTAPI, ServerModeFactory(NewRestAPIServer))
}

type restapiServer struct {
	servermodeSupport
	endpoint string
}

func NewRestAPIServer(_ context.Context, opts *Options) (ServerMode, error) {
	s, err := newServerModeSupport(restapiServerDir)
	if err != nil {
		return nil, err
	}
	s.image = opts.Restdyndns
	return &restapiServer{*s, opts.RestEndpoint}, nil
}

func (s *restapiServer) RequireDataplaneAccess() bool {
	return false
}

func (s *restapiServer) AddValues(values map[string]interface{}) error {
	m := values["runtime"].(map[string]interface{})
	// m["requiredataplane"] = false
	m["restendpoint"] = s.endpoint
	return s.servermodeSupport.AddValues(values)
}
