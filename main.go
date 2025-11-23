package main

import (
	//"context"
	"flag"
	"fmt"
	"log"
	"time"

	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

func main(){
  kubeconfig := flag.String("kubeconfig","/root/.kube/config","location of kubeconfig")
  config , err := clientcmd.BuildConfigFromFlags("",*kubeconfig)
  if err!=nil{
    fmt.Println("Error while fetching kubeconfig:",err.Error())
    config, err = rest.InClusterConfig() // whenever a pod gets created, a default service account is mounted on the pod, so we use this to communicate with the k8s api
    if err!=nil{
      fmt.Println("error getting inclusterconfig")
    }
  }
  clientset , err := kubernetes.NewForConfig(config)
  if err!=nil{
    log.Fatal("Error while creating clientset:",err)
  }
  informerfactory := informers.NewSharedInformerFactory(clientset,30*time.Second)

  podinformer := informerfactory.Core().V1().Pods()

  podinformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
	AddFunc: func(new interface{}){
		fmt.Println("add was called")
	},
	UpdateFunc: func(old,new interface{}){
		fmt.Println("update was called")
	},
	DeleteFunc: func(obj interface{}){
		fmt.Println("delete was called")
	},

  })
  informerfactory.Start(wait.NeverStop) // when the cahce is initialized
  informerfactory.WaitForCacheSync(wait.NeverStop)
  pod, err := podinformer.Lister().Pods("default").Get("default") // The informer has a lister through which we can list. This call goes to the informer and gets a response from the cache instead of the api server
  fmt.Println(pod)

}
