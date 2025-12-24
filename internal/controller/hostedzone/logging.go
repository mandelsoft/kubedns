package hostedzone

import (
	"github.com/mandelsoft/logging"
)

var Realm = logging.NewRealm("controller/hostedzone")

var Log = logging.DefaultContext().Logger(Realm)

func LoggingFor(sub string) logging.Logger {
	if sub == "" {
		return Log
	}
	return logging.DefaultContext().Logger(logging.NewRealm(Realm.Name() + "/" + sub))
}
