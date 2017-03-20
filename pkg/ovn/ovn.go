package ovn

import (
	"github.com/golang/glog"

	"github.com/rajatchopra/ovn-kube/pkg/kube"
	kapi "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

type OvnController struct {
	Kube kube.KubeInterface

	NextPod       func() (cache.DeltaType, *kapi.Pod, error)
	NextEndpoints func() (cache.DeltaType, *kapi.Endpoints, error)

	gatewayCache map[string]string
}

const (
	OVN_NBCTL = "ovn-nbctl"
)

func (oc *OvnController) Run() {
	oc.gatewayCache = make(map[string]string)
	go oc.WatchPods()
	go oc.WatchEndpoints()
}

func (oc *OvnController) WatchPods() {
	for {
		ev, pod, err := oc.NextPod()
		if err != nil {
			glog.Errorf("Error in watching pods: %v", err)
			continue
		}
		switch ev {
		case cache.Added:
			oc.addLogicalPort(pod)
		case cache.Deleted:
			oc.deleteLogicalPort(pod)
		case cache.Updated, cache.Sync:
			// do nothing
		}
	}
}

func (oc *OvnController) WatchEndpoints() {
	for {
		ev, ep, err := oc.NextEndpoints()
		if err != nil {
			glog.Errorf("Error in obtaining next endpoint event- %v", err)
			continue
		}
		glog.V(4).Infof("Endpoint event %v, %v", ev, ep)
		switch ev {
		case cache.Added:
			err = oc.addEndpoints(ep)
		case cache.Deleted:
			err = oc.deleteEndpoints(ep)
		case cache.Updated, cache.Sync:
			// TODO: check what has changed
		}
		if err != nil {
			glog.Errorf("Error in processing endpoint: %v", err)
		}
	}
}
