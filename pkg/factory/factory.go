package factory

import (
	"time"

	utilwait "k8s.io/apimachinery/pkg/util/wait"
	informerfactory "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/rajatchopra/ovn-kube/pkg/cluster"
	"github.com/rajatchopra/ovn-kube/pkg/kube"
	"github.com/rajatchopra/ovn-kube/pkg/ovn"
)

// Factory initializes and manages the kube watches that drive an ovn controller
type Factory struct {
	KClient        kubernetes.Interface
	IFactory       informerfactory.SharedInformerFactory
	ResyncInterval time.Duration
}

// NewDefaultFactory initializes a default ovn controller factory.
func NewDefaultFactory(c kubernetes.Interface) *Factory {
	resyncInterval := 10 * time.Minute
	return &Factory{
		KClient:        c,
		ResyncInterval: resyncInterval,
		IFactory:       informerfactory.NewSharedInformerFactory(c, resyncInterval),
	}
}

// Create begins listing and watching against the API server for the desired route and endpoint
// resources. It spawns child goroutines that cannot be terminated.
func (factory *Factory) CreateOvnController() *ovn.OvnController {

	podInformer := factory.IFactory.Core().V1().Pods()
	endpointsInformer := factory.IFactory.Core().V1().Endpoints()

	return &ovn.OvnController{
		StartPodWatch: func(handler cache.ResourceEventHandler) {
			podInformer.Informer().AddEventHandler(handler)
			podInformer.Informer().Run(utilwait.NeverStop)
		},
		StartEndpointWatch: func(handler cache.ResourceEventHandler) {
			endpointsInformer.Informer().AddEventHandler(handler)
			endpointsInformer.Informer().Run(utilwait.NeverStop)
		},
		Kube: &kube.Kube{KClient: factory.KClient},
	}
}

func (factory *Factory) CreateClusterController() *cluster.OvnClusterController {
	nodeInformer := factory.IFactory.Core().V1().Nodes()
	return &cluster.OvnClusterController{
		StartNodeWatch: func(handler cache.ResourceEventHandler) {
			nodeInformer.Informer().AddEventHandler(handler)
			nodeInformer.Informer().Run(utilwait.NeverStop)
		},
		//NodeInformer: nodeInformer,
		Kube: &kube.Kube{KClient: factory.KClient},
	}
}
