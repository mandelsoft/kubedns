package setup

import (
	"fmt"
	"os"

	"github.com/go-logr/logr"
	"github.com/mandelsoft/logging"
	"github.com/mandelsoft/logging/logrusl"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var Log logging.Logger

// because there are many (partially private Logger variables
// used by kubebuilder and controller runtime initialized
// at different times it is not possible just to shift
// the level, we have to manipulate the commonly used sink, instead.
func init() {
	base := logrusl.Human().NewLogr()

	logging.DefaultContext().SetBaseLogger(base)

	Log = logging.DefaultContext().Logger(logging.NewRealm("controller-runtime"))

	ctrl.SetLogger(LoggerWithShiftedSinkLevel(Log.V(0), 3))
	log.Log = ctrl.Log

	ctrl.Log.V(3).Info("logging initialized")
}

type delegatingSink struct {
	diff int
	orig logr.LogSink
}

func LoggerWithShiftedSinkLevel(l logr.Logger, diff int) logr.Logger {
	return logr.New(ShiftSinkLevel(l.GetSink(), diff)).V(l.GetV())
}

func ShiftSinkLevel(sink logr.LogSink, diff int) logr.LogSink {
	return &delegatingSink{diff, sink}
}

func (d *delegatingSink) Init(info logr.RuntimeInfo) {
	d.orig.Init(info)
}

func (d *delegatingSink) Enabled(level int) bool {
	return d.orig.Enabled(level + d.diff)
}

func (d *delegatingSink) Info(level int, msg string, keysAndValues ...any) {
	d.orig.Info(level+d.diff, msg, keysAndValues...)
}

func (d *delegatingSink) Error(err error, msg string, keysAndValues ...any) {
	d.orig.Error(err, msg, keysAndValues...)
}

func (d *delegatingSink) WithValues(keysAndValues ...any) logr.LogSink {
	return &delegatingSink{d.diff, d.orig.WithValues(keysAndValues...)}
}

func (d *delegatingSink) WithName(name string) logr.LogSink {
	return &delegatingSink{d.diff, d.orig.WithName(name)}
}

////////////////////////////////////////////////////////////////////////////////

func ExitIfErr(err error, msg string, args ...interface{}) {
	if err != nil {
		fmt.Fprintf(os.Stderr, msg+": %s\n", append(args, err)...)
		os.Exit(1)
	}
}
