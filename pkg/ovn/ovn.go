package ovn

import (
	"fmt"
	"github.com/golang/glog"
	"os/exec"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kapi "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

type OvnController struct {
	KClient kubernetes.Interface

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

func (oc *OvnController) getGatewayFromSwitch(logical_switch string) (string, error) {
	var gateway_ip string
	if gateway_ip, ok := oc.gatewayCache[logical_switch]; !ok {
		gateway_ip_bytes, err := exec.Command(OVN_NBCTL, "--if-exists", "get",
			"logical_switch", logical_switch,
			"external_ids:gateway_ip").Output()
		if err != nil {
			return "", err
		}
		gateway_ip = strings.TrimSpace(string(gateway_ip_bytes))
		oc.gatewayCache[logical_switch] = gateway_ip
	}
	return gateway_ip, nil
}

func (oc *OvnController) deleteLogicalPort(pod *kapi.Pod) {
	return
}

func (oc *OvnController) addLogicalPort(pod *kapi.Pod) {
	portName := fmt.Sprintf("%s_%s", pod.Namespace, pod.Name)
	_, err := exec.Command(OVN_NBCTL, "--wait=sb", "--", "--may-exist", "lsp-add",
		pod.Spec.NodeName, portName, "--", "lsp-set-addresses",
		portName, "dynamic", "--", "set",
		"logical_switch_port", portName,
		"external-ids:namespace="+pod.Namespace,
		"external-ids:pod=true").Output()
	if err != nil {
		glog.Errorf("Error while creating logical port %s - %v", portName, err)
		return
	}

	gateway_ip, err := oc.getGatewayFromSwitch(pod.Spec.NodeName)
	if err != nil {
		glog.Errorf("Error obtaining gateway address for switch %s", pod.Spec.NodeName)
		return
	}

	count := 30
	var out []byte
	for count > 0 {
		out, err = exec.Command(OVN_NBCTL, "get", "logical_switch_port", portName, "dynamic_addresses").Output()
		if err == nil {
			break
		}
		glog.V(4).Infof("Error while obtaining addresses for %s - %v", portName, err)
		time.Sleep(time.Second)
	}
	if count == 0 {
		glog.Errorf("Error while obtaining addresses for %s", portName)
		return
	}

	addresses := strings.Split(string(out), " ")
	if len(addresses) != 2 {
		glog.Errorf("Error while obtaining addresses for %s", portName)
		return
	}

	annotation := fmt.Sprintf(`{"ip_address":"%s", "mac_address":"%s", "gateway_ip": "%s"}`, addresses[1], addresses[0], gateway_ip)
	err = oc.SetAnnotationOnPod(pod, "ovn", annotation)
	if err != nil {
		glog.Errorf("Failed to set annotation on pod %s - %v", pod.Name, err)
	}
	return
}

func (oc *OvnController) WatchEndpoints() {
	for {
		ev, ep, err := oc.NextEndpoints()
		if err != nil {
			glog.Errorf("Error in obtaining next endpoint event- %v", err)
			continue
		}
		glog.V(4).Infof("Endpoint event %v, %v", ev, ep)
	}
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
