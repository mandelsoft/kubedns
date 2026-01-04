package kubecrtutils

import (
	"github.com/mandelsoft/logging"
)

var (
	Realm      = logging.DefineRealm("kubecrt", "controller management")
	LogContext = logging.DefaultContext().AttributionContext().WithContext(Realm)
	Log        = LogContext.Logger()
)
