package main

import (
	"flag"
	"net"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	ovnfactory "github.com/rajatchopra/ovn-kube/pkg/factory"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	netController := flag.Bool("net-controller", false, "Flag to start the central controller that watches pods/services/policies")
	master := flag.String("init-master", "", "initialize master, requires the hostname as argument")
	node := flag.String("init-node", "", "initialize node, requires the name that node is registered with in kubernetes cluster")
	token := flag.String("token", "", "kubernetes bearer token for ovn service account")
	flag.Parse()

	// uses the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	factory := ovnfactory.NewDefaultFactory(clientset)

	clusterController := factory.CreateClusterController()
	ovnController := factory.CreateOvnController()

	if *node != "" {
		if *token == "" {
			glog.Errorf("Cannot initialize node without service account 'token'. Please provide one with --token argument")
			return
		}
		clusterController.StartClusterNode(*node)
	} 
	if *master != "" {
		// run the cluster controller to init the master
		_, clusterSub, _ := net.ParseCIDR("11.11.0.0/16")
		clusterController.StartClusterMaster(clusterSub, 8)
	}
	if *netController {
		ovnController.Run()
	}
	if *master != "" || *netController {
		// run forever
		select {}
	}
}
