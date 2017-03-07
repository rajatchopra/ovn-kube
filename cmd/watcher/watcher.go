package main

import (
	"flag"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	ovnkube "github.com/rajatchopra/ovn-kube/pkg/kube"
	ovn "github.com/rajatchopra/ovn-kube/pkg/ovn"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
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
	
	ovnController := ovnkube.NewDefaultOvnControllerFactory(clientset).Create()

	go checkPods(ovnController)
	for {
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
		time.Sleep(10 * time.Second)
	}
}

func checkPods(ovnController *ovn.OvnController) {
	for {
		ev, pod, err := ovnController.NextPod()
		if err != nil {
			fmt.Printf("Error in pod watch: %v", ev)
			continue
		}
		fmt.Printf(" Got event: %v for pod %s in namespace %s\n", ev, pod.Name, pod.Namespace)
	}
}
