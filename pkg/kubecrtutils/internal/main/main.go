package main

import (
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/ctrlmgmt"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/objutils"
	"github.com/spf13/pflag"
)

func main() {

	mgmtDef := ctrlmgmt.Define("coredns.mandelsoft.org", "dataplane").
		AddCluster(
			cluster.Define("dataplane", "user interface").WithFallback(cluster.DEFAULT),
			cluster.Define("runtime", "runtime cluster").WithFallback("dataplane"),
		)

	opts := &flagutils.DefaultOptionSet{}
	opts.Add(mgmtDef)

	flags := pflag.NewFlagSet("cli", pflag.ExitOnError)
	opts.AddFlags(flags)

	fmt.Println(flags.FlagUsages())

	fmt.Printf("40 : %s (%d)\n", gen(40), len(gen(40)))
	fmt.Printf("50 : %s (%d)\n", gen(50), len(gen(50)))
	fmt.Printf("60 : %s (%d)\n", gen(60), len(gen(60)))

}

func gen(length int) string {
	return objutils.GenerateUniqueName("dns-service", "mandelsoft", "my-very-long-object-name", length)
}
