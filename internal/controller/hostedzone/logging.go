package hostedzone

import (
	"github.com/mandelsoft/logging"
)

var Realm = logging.NewRealm("hostedzone")

var Log = logging.DefaultContext().Logger(Realm)
