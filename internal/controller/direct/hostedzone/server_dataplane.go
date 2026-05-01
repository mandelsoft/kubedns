package hostedzone

import (
	"context"
)

const SERVERMODE_DATAPLANE = "dataplane" // use dataplane access for promary DNS server

const dataplaneServerDir = "assets/server/dataplane"

func init() {
	ServerModes.Register(SERVERMODE_DATAPLANE, ServerModeFactory(NewDataplaneServer))
}

type dataplaneServer struct {
	servermodeSupport
}

func NewDataplaneServer(_ context.Context, opts *Options) (ServerMode, error) {
	s, err := newServerModeSupport(dataplaneServerDir)
	if err != nil {
		return nil, err
	}
	s.image = opts.Kubedyndns
	return &dataplaneServer{*s}, nil
}

func (s *dataplaneServer) RequireDataplaneAccess() bool {
	return true
}

func (s *dataplaneServer) AddValues(values map[string]interface{}) error {
	m := values["dataplane"].(map[string]interface{})
	m["required"] = true
	return s.servermodeSupport.AddValues(values)
}
