package kube

import (
	"fmt"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	kapi "k8s.io/client-go/pkg/api/v1"
)

type KubeInterface interface {
	SetAnnotationOnPod(pod *kapi.Pod, key, value string) error
	GetPod(namespace, name string) (*kapi.Pod, error)
}

type Kube struct {
	KClient kubernetes.Interface
}

func (k *Kube) SetAnnotationOnPod(pod *kapi.Pod, key, value string) error {
	glog.Infof("Setting annotations %s=%s on %s", key, value, pod.Name)
	patchData := fmt.Sprintf(`{"metadata":{"annotations":{"%s":"%s"}}}`, key, value)
	res, err := k.KClient.Core().Pods(pod.Namespace).Patch(pod.Name, types.MergePatchType, []byte(patchData))
	if err != nil {
		glog.Errorf("Error in setting annotation on pod %s/%s: %v", pod.Name, pod.Namespace, err)
	}
	if res.Annotations[key] != value {
		fmt.Printf("Annotations not set properly - %v", res.Annotations)
	}
	return err
}

func (k *Kube) GetPod(namespace, name string) (*kapi.Pod, error) {
	return k.KClient.Core().Pods(namespace).Get(name, metav1.GetOptions{})
}
