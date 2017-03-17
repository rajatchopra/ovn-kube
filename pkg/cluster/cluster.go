package cluster

import (
	"net"

	"github.com/openshift/origin/pkg/util/netutils"
	"github.com/rajatchopra/ovn-kube/pkg/kube"
	kapi "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"
)

type OvnClusterController struct {
	Kube                  kube.KubeInterface
	masterSubnetAllocator *netutils.SubnetAllocator

	KubeServer       string
	CACert           string
	Token            string
	ClusterIPNet     *net.IPNet
	HostSubnetLength uint32

	NextNode func() (cache.DeltaType, *kapi.Node, error)
}

const (
	OVN_HOST_SUBNET = "ovn_host_subnet"
)
