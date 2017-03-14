package cluster

import (
	"net"
	"time"

	"github.com/golang/glog"
	kapi "k8s.io/client-go/pkg/api/v1"
)

func (cluster *OvnClusterController) StartClusterNode(name string) error {
	count := 30
	var err error
	var node *kapi.Node
	var subnet *net.IPNet

	for count > 0 {
		if count != 30 {
			count--
			time.Sleep(time.Second)
		}

		// setup the node, create the logical switch
		node, err = cluster.Kube.GetNode(name)
		if err != nil {
			glog.Errorf("Error starting node %s, no node found - %v", name, err)
			continue
		}

		sub, ok := node.Annotations[OVN_HOST_SUBNET]
		if !ok {
			glog.Errorf("Error starting node %s, no annotation found on node for subnet - %v", name, err)
			continue
		}
		_, subnet, err = net.ParseCIDR(sub)
		if err != nil {
			glog.Errorf("Invalid hostsubnet found for node %s - %v", node.Name, err)
			return err
		}
		break
	}

	glog.Infof("Node %s ready for ovn initialization with subnet %s", node.Name, subnet.String())

	return err
}
