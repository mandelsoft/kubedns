package main

import (
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/ctrlmgmt"
	"github.com/spf13/pflag"
)

func main() {

	mgmtDef := ctrlmgmt.NewDefinition().
		AddCluster(
			cluster.NewDefinition("dataplane", "user interface").WithFallback(cluster.DEFAULT),
			cluster.NewDefinition("runtime", "runtime cluster").WithFallback("dataplane"),
		)

	opts := &flagutils.DefaultOptionSet{}
	opts.Add(mgmtDef)

	flags := pflag.NewFlagSet("cli", pflag.ExitOnError)
	opts.AddFlags(flags)

	fmt.Println(flags.FlagUsages())

}
