package main

import (
	"flag"
	"fmt"
	"net"
	"time"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	restclient "k8s.io/client-go/rest"
	certutil "k8s.io/client-go/util/cert"

	ovnfactory "github.com/rajatchopra/ovn-kube/pkg/factory"
	ovn "github.com/rajatchopra/ovn-kube/pkg/ovn"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	server := flag.String("apiserver", "https://localhost:8443", "url to the kubernetes apiserver")
	rootCAFile := flag.String("ca-cert", "", "CA cert for the api server")
	token := flag.String("token", "", "Bearer token to use for establishing ovn infrastructure")

	testAnnotations := flag.Bool("annotate", false, "test annotations on pods interactively")
	master := flag.Bool("master", true, "run in master mode")
	node := flag.String("node", "", "run to initialize the given hostname")
	flag.Parse()

	var config *restclient.Config
	var err error
	if (*kubeconfig != "") {
		// uses the current context in kubeconfig
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	} else if (*server != "" && *token != "" && ((*rootCAFile != "") || !strings.HasPrefix(*server, "https"))) {
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

	factory := ovnfactory.NewDefaultFactory(clientset)

	ovnController := factory.CreateOvnController()
	clusterController := factory.CreateClusterController()

	if *testAnnotations {
		SetRandomAnnotations(clientset, ovnController)
	} else if *node != "" {
		clusterController.StartClusterNode(*node)
	} else {
		ovnController.Run()

		if *master {
			// run the cluster controller to init the master
			_, clusterSub, _ := net.ParseCIDR("11.11.0.0/16")
			clusterController.StartClusterMaster(clusterSub, 8)
		}
		select {}
	}
}

func SetRandomAnnotations(clientset *kubernetes.Clientset, ovnController *ovn.OvnController) {
	for {
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		for _, pod := range pods.Items {
			var key, value string
			fmt.Println("Enter key for pod ", pod.Name)
			n, err := fmt.Scanf("%s", &key)
			if n != 1 || err != nil {
				fmt.Printf("Didn't scan properly %v, %v", n, err)
				continue
			}
			fmt.Println("Enter value for key ", key, " for pod", pod.Name)
			n, err = fmt.Scanf("%s", &value)
			if n != 1 || err != nil {
				fmt.Printf("Didn't scan properly %v, %v", n, err)
				continue
			}
			ovnController.Kube.SetAnnotationOnPod(&pod, key, value)
		}
		time.Sleep(10 * time.Second)
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
