package Controller

import (
	"context"
	"fmt"
	"time"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	appinformer "k8s.io/client-go/informers/apps/v1"
	"k8s.io/client-go/kubernetes"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	//network "k8s.io/kubernetes/pkg/apis/networking"
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
	defer c.queue.Forget(item) // forget the item once it is processed
	key, err := cache.MetaNamespaceKeyFunc(item)
	if err!=nil{
		fmt.Println("Error getting key from cache:", err)
	}
	ns, name, err := cache.SplitMetaNamespaceKey(key) // name here is the name of the deployment object
	if err!=nil{
		fmt.Println("Error getting namespace, name from key:", err)
	}
	// check if the object has been deleted from the cluster
	_, err = c.clientset.AppsV1().Deployments(ns).Get(context.Background(),name,metav1.GetOptions{})
	if apierrors.IsNotFound(err){
		fmt.Printf("Deployment %s not found; Handling delete event for deployment",name)
		// delete service
		err := c.clientset.CoreV1().Services(ns).Delete(context.Background(),name,metav1.DeleteOptions{})
		if err!=nil{
			fmt.Println("Error deleting service for deployment:" ,err)
			return false
		}
		// delete network policy
		err = c.clientset.NetworkingV1().NetworkPolicies(ns).Delete(context.Background(),name,metav1.DeleteOptions{})
		if err !=nil{
			fmt.Println("Error deleting network policy for deployment:", err)
			return false
		}
		return true
	}
	err = c.syncDeployment(ns,name) // things we want to once the deployment is created
	if err!=nil{
		//re try
		fmt.Println("Error syncing deploymemt:", err)
		return false
	}

	return true
}
func (c *controller) syncDeployment(ns string,name string) error{
	// In production we have to create annotations like owner reference and what created it so we can delete the objects created only by the controller and not any other objects created by the user in case of the object having same labels

	// create network policy
	err := c.createNetworkPolicy(ns,name)
	if err!=nil{
		fmt.Println("Error creating network polciy:",err)
		return err
	}

	dep, err:= c.lister.Deployments(ns).Get(name)
	labels := getPodLabels(dep)
	//create service
	// we will have to modify this to make the port dynamic based on the deployment
	svc := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: dep.Name,
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
					{Name: "http",
					Port: 80,
			},
		},
		 Selector: labels,
		},
	}
	_, err = c.clientset.CoreV1().Services(ns).Create(context.Background(),&svc,metav1.CreateOptions{})
	if err!=nil{
		fmt.Println("creating service error:", err)
	} 
	return nil
}
// Creates network policy for the deployment which blocks all ingress and egress
func (c *controller)createNetworkPolicy(ns string,name string) error{
	dep, err:= c.lister.Deployments(ns).Get(name)
	labels := getPodLabels(dep)
	np:= v1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: dep.Name,
			Namespace: ns,
		},
		Spec: v1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: labels,
			},
			Ingress: []v1.NetworkPolicyIngressRule{

			},
			Egress: []v1.NetworkPolicyEgressRule{

			},
		},
	}
	_,err= c.clientset.NetworkingV1().NetworkPolicies(ns).Create(context.Background(),&np,metav1.CreateOptions{})
	if err!=nil{
		return err
	}
	return nil
}
// Add the object to the queue to further process the object and perform business logic
func (c *controller) handleAdd(obj interface{}) {
	c.queue.Add(obj)
}
func (c *controller) handleDelete(obj interface{}) {
	c.queue.Add(obj)
}
// This function gets the labels from the deployment object which then can be used as selectors in other k8s objects such as service
func getPodLabels(dep *appsv1.Deployment)map[string]string{
	return dep.Spec.Template.Labels
}