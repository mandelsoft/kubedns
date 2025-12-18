package setup

import (
	"fmt"
	"os"

	ctrl "sigs.k8s.io/controller-runtime"
)

var SetupLog = ctrl.Log.WithName("setup")

func ExitIfErr(err error, msg string, args ...interface{}) {
	if err != nil {
		fmt.Fprintf(os.Stderr, msg+": %s\n", append(args, err)...)
		os.Exit(1)
	}
}