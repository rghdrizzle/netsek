package main

import (
	//"context"
	"flag"
	"fmt"
	"log"
	"time"
	"rghdrizzle/netsek/Controller"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/cache"
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
  ch := make(chan struct{}) // recieve only channel
  informerfactory := informers.NewSharedInformerFactory(clientset,30*time.Second)
  informerfactory.Start(ch)
  cont := Controller.New(*clientset,informerfactory.Apps().V1().Deployments())
  cont.Run(ch)

}
