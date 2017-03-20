package ovn

import (
	"fmt"
	"github.com/golang/glog"
	"os/exec"

	kapi "k8s.io/client-go/pkg/api/v1"
)

const (
	TCP_LB = "k8s-cluster-lb-tcp"
	UDP_LB = "k8s-cluster-lb-udp"
)

func (ovn *OvnController) getLoadBalancer(protocol kapi.Protocol) string {
	if protocol == kapi.ProtocolTCP {
		return TCP_LB
	} else if protocol == kapi.ProtocolUDP {
		return UDP_LB
	}
	return ""
}

func (ovn *OvnController) createLoadBalancerVIP(lb string, serviceIP string, port int32, ips []string, targetPort int32) error {
	// With service_ip:port as a VIP, create an entry in 'load_balancer'
	// key is of the form "IP:port" (with quotes around)
	key := fmt.Sprintf("\"%s:%d\"", serviceIP, port)

	if len(ips) == 0 {
		_, err := exec.Command(OVN_NBCTL, "remove", "load_balancer", lb, "vips", key).CombinedOutput()
		return err
	}

	var commaSeparatedEndpoints string
	for i, ep := range ips {
		comma := ","
		if i == 0 {
			comma = ""
		}
		commaSeparatedEndpoints += fmt.Sprintf("%s%s:%d", comma, ep, targetPort)
	}
	target := fmt.Sprintf("vips:%s=\"%s\"", commaSeparatedEndpoints)

	out, err := exec.Command(OVN_NBCTL, "set", "load_balancer", lb, target).CombinedOutput()
	if err != nil {
		glog.Errorf("Error in creating load balancer: %v(%v)", string(out), err)
	}
	return err
}

func (ovn *OvnController) addEndpoints(ep *kapi.Endpoints) error {
	// get service
	svc, err := ovn.Kube.GetService(ep.Name, ep.Namespace)
	if err != nil {
		return err
	}
	tcpPortMap := make(map[int32]([]string))
	udpPortMap := make(map[int32]([]string))
	for _, s := range ep.Subsets {
		for _, ip := range s.Addresses {
			for _, port := range s.Ports {
				var ips []string
				var portMap map[int32]([]string)
				var ok bool
				if port.Protocol == kapi.ProtocolUDP {
					portMap = tcpPortMap
				} else if port.Protocol == kapi.ProtocolTCP {
					portMap = udpPortMap
				}
				if ips, ok = portMap[port.Port]; !ok {
					ips = make([]string, 0)
				}
				ips = append(ips, ip.IP)
				portMap[port.Port] = ips
			}
		}
	}

	for targetPort, ips := range tcpPortMap {
		for _, svcPort := range svc.Spec.Ports {
			if svcPort.Protocol == kapi.ProtocolTCP && svcPort.TargetPort.IntVal == targetPort {
				err := ovn.createLoadBalancerVIP(TCP_LB, svc.Spec.ClusterIP, svcPort.Port, ips, targetPort)
				if err != nil {
					return err
				}
			}
		}
	}
	for targetPort, ips := range udpPortMap {
		for _, svcPort := range svc.Spec.Ports {
			if svcPort.Protocol == kapi.ProtocolTCP && svcPort.TargetPort.IntVal == targetPort {
				err := ovn.createLoadBalancerVIP(UDP_LB, svc.Spec.ClusterIP, svcPort.Port, ips, targetPort)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (ovn *OvnController) deleteEndpoints(ep *kapi.Endpoints) error {
	svc, err := ovn.Kube.GetService(ep.Name, ep.Namespace)
	if err != nil {
		return err
	}
	for _, svcPort := range svc.Spec.Ports {
		lb := ovn.getLoadBalancer(svcPort.Protocol)
		key := fmt.Sprintf("\"%s:%d\"", svc.Spec.ClusterIP, svcPort.Port)
		_, err := exec.Command(OVN_NBCTL, "remove", "load_balancer", lb, "vips", key).CombinedOutput()
		if err != nil {
			glog.Errorf("Error in deleting endpoints: %v", err)
		}
	}
	return nil
}
