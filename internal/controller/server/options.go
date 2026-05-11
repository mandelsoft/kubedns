package server

import (
	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/flagutils/pflags"
	corednsv1alpha1 "github.com/mandelsoft/kubedns/api/coredns/v1alpha1"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func From(opts flagutils.OptionSetProvider) *Options {
	return flagutils.GetFrom[*Options](opts)
}

type Options struct {
	Class *string
	Slave bool
}

func NewOptions() *Options {
	return &Options{}
}

func (o *Options) AddFlags(fs *pflag.FlagSet) {
	pflags.StringRefVar(fs, &o.Class, "server-class", nil, "class name to be served by REST server")
	fs.BoolVar(&o.Slave, "slave-mode", false, "run server in slave mode")
}

func (o *Options) IsMaster(conditions []metav1.Condition) bool {
	// check for plain mode.
	// This means the controller is explictly managed and not by an aaS controller
	// managing additional conditions
	if !o.Slave {
		plain := true
		for _, c := range conditions {
			if c.Type != corednsv1alpha1.ServerConditionType {
				return false
				break
			}
		}
		return plain
	}
	return false

}
