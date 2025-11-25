package Controller

import (
	"k8s.io/client-go/kubernetes"
	applisters "k8s.io/client-go/listers/apps/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type controller struct {
	clientset kubernetes.Interface
	lister applisters.DeploymentLister
	depCacheSyncd cache.InformerSynced
	queue  workqueue.RateLimitingInterface
}

func New() *controller{
	return &controller{}
}