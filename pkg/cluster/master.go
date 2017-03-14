package cluster

import (
	"fmt"
	"net"

	"github.com/golang/glog"

	utilwait "k8s.io/apimachinery/pkg/util/wait"
	kapi "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/openshift/origin/pkg/util/netutils"
)

func (cluster *OvnClusterController) StartClusterMaster(clusterNetwork *net.IPNet, hostSubnetLength uint32) error {
	subrange := make([]string, 0)
	existingNodes, err := cluster.Kube.GetNodes()
	if err != nil {
		glog.Errorf("Error in initializing/fetching subnets: %v", err)
		return err
	}
	for _, node := range existingNodes.Items {
		hostsubnet, ok := node.Annotations[OVN_HOST_SUBNET]
		if ok {
			subrange = append(subrange, hostsubnet)
		}
	}

	cluster.masterSubnetAllocator, err = netutils.NewSubnetAllocator(clusterNetwork.String(), hostSubnetLength, subrange)
	if err != nil {
		return err
	}

	go utilwait.Forever(cluster.watchNodes, 0)
	return nil
}

func (cluster *OvnClusterController) addNode(node *kapi.Node) error {
	// Create new subnet
	sn, err := cluster.masterSubnetAllocator.GetNetwork()
	if err != nil {
		return fmt.Errorf("Error allocating network for node %s: %v", node.Name, err)
	}

	err = cluster.Kube.SetAnnotationOnNode(node, OVN_HOST_SUBNET, sn.String())
	if err != nil {
		cluster.masterSubnetAllocator.ReleaseNetwork(sn)
		return fmt.Errorf("Error creating subnet %s for node %s: %v", sn.String(), node.Name, err)
	}
	glog.Infof("Created HostSubnet %s", sn.String())
	return nil
}

func (cluster *OvnClusterController) deleteNode(node *kapi.Node) error {
	sub, ok := node.Annotations[OVN_HOST_SUBNET]
	if !ok {
		return fmt.Errorf("Error in obtaining host subnet for node %q for deletion", node.Name)
	}

	_, subnet, err := net.ParseCIDR(sub)
	if err != nil {
		return fmt.Errorf("Error in parsing hostsubnet - %v", err)
	}
	err = cluster.masterSubnetAllocator.ReleaseNetwork(subnet)
	if err != nil {
		return fmt.Errorf("Error deleting subnet %v for node %q: %v", sub, node.Name, err)
	}

	glog.Infof("Deleted HostSubnet %s for node %s", sub, node.Name)
	return nil
}

func (cluster *OvnClusterController) watchNodes() {
	ev, node, err := cluster.NextNode()
	switch ev {
	case cache.Added:
		glog.V(5).Infof("Added event for Node %q", node.Name)

		err = cluster.addNode(node)
		if err != nil {
			glog.Errorf("error creating subnet for node %s: %v", node.Name, err)
		}
	case cache.Deleted:
		glog.V(5).Infof("Delete event for Node %q", node.Name)

		err = cluster.deleteNode(node)
		if err != nil {
			glog.Errorf("Error deleting node %s: %v", node.Name, err)
		}
	case cache.Sync, cache.Updated:
		// do nothing
	}
}
