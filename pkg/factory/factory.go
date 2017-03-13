package factory

import (
	"fmt"
	"strings"
	"time"

	//kapi "k8s.io/apimachinery/pkg/api"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	kapi "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/cache"

	"github.com/rajatchopra/ovn-kube/pkg/kube"
	"github.com/rajatchopra/ovn-kube/pkg/ovn"
)

// OvnControllerFactory initializes and manages the kube watches that drive an ovn controller
type OvnControllerFactory struct {
	KClient        kubernetes.Interface
	ResyncInterval time.Duration
	Namespace      string
	Labels         labels.Selector
	Fields         fields.Selector
}

// NewDefaultOvnControllerFactory initializes a default ovn controller factory.
func NewDefaultOvnControllerFactory(c kubernetes.Interface) *OvnControllerFactory {
	return &OvnControllerFactory{
		KClient:        c,
		ResyncInterval: 10 * time.Minute,
		Namespace:      kapi.NamespaceAll,
		Labels:         labels.Everything(),
		Fields:         fields.Everything(),
	}
}

func (factory *OvnControllerFactory) newEventQueue(client cache.Getter, resourceName string, expectedType interface{}, namespace string) *cache.DeltaFIFO {
	rn := strings.ToLower(resourceName)
	lw := cache.NewListWatchFromClient(client, rn, namespace, fields.Everything())
	keyFunc := cache.DeletionHandlingMetaNamespaceKeyFunc
	knownObjectStore := cache.NewStore(keyFunc)
	eventQueue := cache.NewDeltaFIFO(
		keyFunc,
		nil,
		knownObjectStore)
	// Repopulate event queue every sync Interval
	// Existing items in the event queue will have watch.Modified event type
	cache.NewReflector(lw, expectedType, eventQueue, factory.ResyncInterval).Run()
	return eventQueue
}

type watchEvent struct {
	Event cache.DeltaType
	Obj   interface{}
}

// Create begins listing and watching against the API server for the desired route and endpoint
// resources. It spawns child goroutines that cannot be terminated.
func (factory *OvnControllerFactory) Create() *ovn.OvnController {

	endpointsEventQueue := factory.newEventQueue(factory.KClient.Core().RESTClient(), "endpoints", &kapi.Endpoints{}, factory.Namespace)
	podsEventQueue := factory.newEventQueue(factory.KClient.Core().RESTClient(), "pods", &kapi.Pod{}, factory.Namespace)
	nodesEventQueue := factory.newEventQueue(factory.KClient.Core().RESTClient(), "nodes", &kapi.Node{}, factory.Namespace)

	return &ovn.OvnController{
		NextPod: func() (cache.DeltaType, *kapi.Pod, error) {
			we := &watchEvent{}
			podsEventQueue.Pop(func(obj interface{}) error {
				delta, ok := obj.(cache.Deltas)
				if !ok {
					fmt.Printf("Object %v not cache.Delta type", obj)
				}
				we.Obj = delta.Newest().Object
				we.Event = delta.Newest().Type
				return nil
			})
			return we.Event, we.Obj.(*kapi.Pod), nil
		},
		NextEndpoints: func() (cache.DeltaType, *kapi.Endpoints, error) {
			we := &watchEvent{}
			endpointsEventQueue.Pop(func(obj interface{}) error {
				delta, ok := obj.(cache.Deltas)
				if !ok {
					fmt.Printf("Object %v not cache.Delta type", obj)
				}
				we.Obj = delta.Newest().Object
				we.Event = delta.Newest().Type
				return nil
			})
			return we.Event, we.Obj.(*kapi.Endpoints), nil
		},
		NextNode: func() (cache.DeltaType, *kapi.Node, error) {
			we := &watchEvent{}
			nodesEventQueue.Pop(func(obj interface{}) error {
				delta, ok := obj.(cache.Deltas)
				if !ok {
					fmt.Printf("Object %v not cache.Delta type", obj)
				}
				we.Obj = delta.Newest().Object
				we.Event = delta.Newest().Type
				return nil
			})
			return we.Event, we.Obj.(*kapi.Node), nil
		},
		Kube: &kube.Kube{KClient: factory.KClient},
	}
}
