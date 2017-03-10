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
	ovnController.Run()
	for {
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
			ovnController.SetAnnotationOnPod(&pod, key, value)
		}
		time.Sleep(10 * time.Second)
	}
}
