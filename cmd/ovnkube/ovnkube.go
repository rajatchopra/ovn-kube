package main

import (
	"flag"
	"fmt"
	"net"
	"strings"

	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"

	ovnfactory "github.com/rajatchopra/ovn-kube/pkg/factory"
)

func main() {
	// auth flags
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	server := flag.String("apiserver", "https://localhost:8443", "url to the kubernetes apiserver")
	rootCAFile := flag.String("ca-cert", "", "CA cert for the api server")
	token := flag.String("token", "", "Bearer token to use for establishing ovn infrastructure")

	// mode flags
	netController := flag.Bool("net-controller", false, "Flag to start the central controller that watches pods/services/policies")
	master := flag.String("init-master", "", "initialize master, requires the hostname as argument")
	node := flag.String("init-node", "", "initialize node, requires the name that node is registered with in kubernetes cluster")

	flag.Parse()

	// Process auth flags
	var config *restclient.Config
	var err error
	if *kubeconfig != "" {
		// uses the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else if *server != "" && *token != "" && ((*rootCAFile != "") || !strings.HasPrefix(*server, "https")) {
		config, err = CreateConfig(*server, *token, *rootCAFile)
	} else {
		err = fmt.Errorf("Provide kubeconfig file or give server/token/tls credentials")
	}
	if err != nil {
		panic(err.Error())
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	// create factory and start the controllers asked for
	factory := ovnfactory.NewDefaultFactory(clientset)

	clusterController := factory.CreateClusterController()
	ovnController := factory.CreateOvnController()

	if *node != "" {
		if *token == "" {
			panic("Cannot initialize node without service account 'token'. Please provide one with --token argument")
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

func CreateConfig(server, token, rootCAFile string) (*restclient.Config, error) {
	tlsClientConfig := restclient.TLSClientConfig{}
	if rootCAFile != "" {
		if _, err := certutil.NewPool(rootCAFile); err != nil {
			return nil, err
		} else {
			tlsClientConfig.CAFile = rootCAFile
		}
	}

	return &restclient.Config{
		Host:            server,
		BearerToken:     string(token),
		TLSClientConfig: tlsClientConfig,
	}, nil
}
