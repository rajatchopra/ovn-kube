
package ovn

import (
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/client-go/tools/cache"
)


type OvnController struct {
	NextPod func() (cache.DeltaType, *kapi.Pod, error)
	NextEndpoints func() (cache.DeltaType, *kapi.Endpoints, error)
}
