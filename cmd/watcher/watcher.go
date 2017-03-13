package main

import (
	"flag"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	ovnfactory "github.com/rajatchopra/ovn-kube/pkg/factory"
	ovn "github.com/rajatchopra/ovn-kube/pkg/ovn"
)

func main() {
	kubeconfig := flag.String("kubeconfig", "./config", "absolute path to the kubeconfig file")
	testAnnotations := flag.Bool("annotate", false, "test annotations on pods interactively")
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

	ovnController := ovnfactory.NewDefaultOvnControllerFactory(clientset).Create()

	if *testAnnotations {
		SetRandomAnnotations(clientset, ovnController)
	} else {
		ovnController.Run()
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
