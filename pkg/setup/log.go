package setup

import (
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
)

var SetupLog = ctrl.Log.WithName("setup")

func ExitIfErr(err error, msg string, args ...interface{}) {
	if err != nil {
		SetupLog.Error(err, msg, args...)
		os.Exit(1)
	}
}