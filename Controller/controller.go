package Controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	network "k8s.io/kubernetes/pkg/apis/networking"
	"k8s.io/apimachinery/pkg/util/wait"
	appinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {

	clientset     kubernetes.Interface
	lister        applisters.DeploymentLister
	depCacheSyncd cache.InformerSynced            // to check if the cache has been synced or not
	queue         workqueue.RateLimitingInterface // for pushing events
}

func New(clientset kubernetes.Clientset, depInformer appinformer.DeploymentInformer) *controller {
	c:= &controller{
		clientset:     &clientset,
		lister:        depInformer.Lister(),
		depCacheSyncd: depInformer.Informer().HasSynced,
		queue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "netsek"),
	}
	depInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    c.handleAdd,
			DeleteFunc: c.handleDelete,
		},
	)
	return c
}

// This runs the netsek controller
func (c *controller) Run(ch <-chan struct{}) {
	fmt.Println("Controller Starting")
	if !cache.WaitForCacheSync(ch, c.depCacheSyncd) { // since the informer maintains a local cahce, we will have to wait for that cache to be synced or inialized if it is the first time
		fmt.Println("Error watiting for cache to be synced")
	}
	go wait.Until(c.worker, 1*time.Second, ch) // this calls a specific function for a duration until the channel is closed
	<-ch                                       // waits for something to be recieved but since nothing will the go routine above will not return
}

func (c *controller) worker() {
	for c.processItem() {

	}
}
func (c *controller) processItem() bool {
	item, shutdown := c.queue.Get()
	if shutdown{
		return false
	}

	key, err := cache.MetaNamespaceKeyFunc(item)
	if err!=nil{
		fmt.Println("Error getting key from cache:", err)
	}
	ns, name, err := cache.SplitMetaNamespaceKey(key)
	if err!=nil{
		fmt.Println("Error getting namespace, name from key:", err)
	}
	err = c.syncDeployment() // things we want to once the deployment is created
	if err!=nil{
		//re try
		fmt.Println("Error syncing deploymemt:", err)
		return false
	}

	return true
}
func (c *controller) syncDeployment(ns,name string) error{
	// create network policy
	//np:= network.NetworkPolicy{}


	//create service
	svc := corev1.Service{}
	_, err := c.clientset.CoreV1().Services(ns).Create(context.Background(),&svc,metav1.CreateOptions{})
	if err!=nil{
		fmt.Println("creating service error:", err)
	} 
	return nil
}

func (c *controller) handleAdd(obj interface{}) {
	c.queue.Add(obj)
}
func (c *controller) handleDelete(obj interface{}) {
	c.queue.Add(obj)
}
