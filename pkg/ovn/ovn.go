
package ovn

import (
	"fmt"
	"github.com/golang/glog"

	kapi "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)


type OvnController struct {
	KClient kubernetes.Interface
	NextPod func() (cache.DeltaType, *kapi.Pod, error)
	NextEndpoints func() (cache.DeltaType, *kapi.Endpoints, error)
}

func (oc *OvnController) Run() {
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
		switch(ev) {
		cache.Added:
			portName := fmt.Sprintf("%s_%s", pod.Namespace, pod.Name)
			_, err := exec.CombinedOutput(OVN_NBCTL, "--wait=sb", "--", "--may-exist", "lsp-add",
                            pod.Spec.NodeName, portName, "--", "lsp-set-addresses",
                            portName, "dynamic", "--", "set",
                            "logical_switch_port", portName,
                            "external-ids:namespace=" + pod.Namespace,
                            "external-ids:pod=true")
			if err != nil {
				glog.Errorf("Error while creating logical port %s - %v", portName, err)
				continue
			}

			out, err := exec.CombinedOutput(OVN_NBCTL, "get", "logical_switch_port", portName, "dynamic_addresses")
			if err != nil {
				glog.Errorf("Error while obtaining addresses for %s - %v", portName, err)
				continue
			}
			addresses := strings.Split(out, " ")
			if len(addresses) != 2 {
				glog.Errorf("Error while obtaining addresses for %s - %v", portName, err)
				continue
			}

			annotation := fmt.Sprintf(`{"ip_address":"%s", "mac_address":"%s", "gateway_ip": "%s"}`, addresses[1], addresses[0], util.GatewayAddress(addresses[1]))
			_, err := oc.SetAnnotationOnPod(&pod, "ovn", annotation)
		cache.Deleted:
		cache.Updated, cache.Sync:
			// do nothing
		}
	}
}

func (oc *OvnController) WatchEndpoints() {
}

func (oc *OvnController) SetAnnotationOnPod(pod *kapi.Pod, key, value string) error {
	glog.Infof("Setting annotations %s=%s on %s", key, value, pod.Name)
	patchData := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`, key, value)
	res, err := oc.KClient.Core().Pods(pod.Namespace).Patch(pod.Name, types.MergePatchType, []byte(patchData))
	if err != nil {
		glog.Errorf("Error in setting annotation on pod %s/%s: %v", pod.Name, pod.Namespace, err)
	}
	if res.Annotations[key] != value {
		fmt.Printf("Annotations not set properly - %v", res)
	}
	return err
}
