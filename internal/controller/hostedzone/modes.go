package hostedzone

type Mode interface {
	GetRuntimeNamespace(*ReconcileRequest) string
	AccessValues(*ReconcileRequest) map[string]interface{}
}

////////////////////////////////////////////////////////////////////////////////

type ModeImpl struct {
	*HostedZoneReconciler
}

func (m *ModeImpl) AccessValues(*ReconcileRequest) map[string]interface{} {
	return nil
}

func (m *ModeImpl) GetRuntimeNamespace(req *ReconcileRequest) string {
	if req.reconciler.Options.RuntimeNamespace != "" {
		return req.reconciler.Options.RuntimeNamespace
	}
	return req.instance.Namespace
}

////////////////////////////////////////////////////////////////////////////////

type LocalMode struct {
	ModeImpl
}

var _ Mode = (*LocalMode)(nil)

////////////////////////////////////////////////////////////////////////////////

type RuntimeMode struct {
	ModeImpl
}

var _ Mode = (*RuntimeMode)(nil)

func (r *RuntimeMode) AccessValues(req *ReconcileRequest) map[string]interface{} {
	// TODO implement me
	panic("implement me")
}
