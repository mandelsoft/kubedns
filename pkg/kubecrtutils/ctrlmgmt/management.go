package ctrlmgmt

import (
	"context"
	"fmt"

	"github.com/mandelsoft/flagutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/cluster"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/controller"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/index"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/internal"
	"github.com/mandelsoft/kubedns/pkg/kubecrtutils/options/manageropts"
	"github.com/mandelsoft/logging"
	ctrl "sigs.k8s.io/controller-runtime"
)

func NewControllerManagerByOpts(ctx context.Context, opts flagutils.OptionSetProvider) (ControllerManager, error) {
	def := From(opts)

	if def == nil {
		return nil, fmt.Errorf("no management definition found")
	}

	if def.GetError() != nil {
		return nil, fmt.Errorf("management definition: %w", def.GetError())
	}

	copts := cluster.From(opts)
	if copts == nil {
		return nil, fmt.Errorf("no clusters found in options")
	}
	clusters := copts.GetClusters()

	mopts := manageropts.From(opts)
	if mopts == nil {
		return nil, fmt.Errorf("no manager options found")
	}

	manager, err := mopts.GetManager(ctx, opts)
	if err != nil {
		return nil, err
	}

	var indices index.Indices
	iopts := index.From(opts)
	if iopts != nil {
		indices, err = iopts.GetIndices(ctx, clusters)
		if err != nil {
			return nil, err
		}
	} else {
		indices = index.NewIndices()
	}
	cm := &_controllermanager{
		Element:  internal.NewElement(def.GetName()),
		logger:   kubecrtutils.LogContext.WithContext(logging.NewRealm(kubecrtutils.Realm.Name() + "/" + def.GetName())).Logger(),
		clusters: clusters,
		manager:  manager,
		main:     clusters.Get(mopts.GetMain()),
		indices:  indices,
	}

	cntropts := controller.From(opts)
	if cntropts == nil {
		return nil, fmt.Errorf("no controller definitions found")
	}
	cntr, err := cntropts.Apply(ctx, cm)
	if err != nil {
		return nil, err
	}
	cm.controllers = cntr
	return cm, nil
}

type _controllermanager struct {
	internal.Element
	logger      logging.Logger
	main        cluster.Cluster
	manager     ctrl.Manager
	clusters    cluster.Clusters
	indices     index.Indices
	controllers controller.Controllers
}

func (cm *_controllermanager) GetLogger() logging.Logger {
	return cm.logger
}

func (cm *_controllermanager) GetClusters() cluster.Clusters {
	return cm.clusters
}

func (cm *_controllermanager) GetIndices() index.Indices {
	return cm.indices
}

func (cm *_controllermanager) GetManager() ctrl.Manager {
	return cm.manager
}

func (cm *_controllermanager) GetMainCLuster() cluster.Cluster {
	return cm.main
}

func (cm *_controllermanager) GetCluster(name string) cluster.Cluster {
	return cm.clusters.Get(name)
}

func (cm *_controllermanager) GetIndex(name string) cluster.Index {
	return cm.indices.Get(name)
}
