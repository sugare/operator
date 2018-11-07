package controller

import (
	"fmt"
	"github.com/golang/glog"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type Controller struct {

	Clientset kubernetes.Interface

	Workqueue workqueue.RateLimitingInterface

	Informer cache.SharedIndexInformer

}

func (c *Controller) Run(stopCh <-chan struct{}) {
	defer runtime.HandleCrash()
	defer c.Workqueue.ShutDown()

	go c.Informer.Run(stopCh)
	if !cache.WaitForCacheSync(stopCh, c.HasSynced) {
		runtime.HandleError(fmt.Errorf("Error syncing cache"))
		return
	}
	glog.V(1).Info("Controller.Run: cache sync complete")

	wait.Until(c.runWorker, time.Second, stopCh)

}
//func NewController()  {
//
//}
func (c *Controller) runWorker() {
	glog.Info("runWorker...")
	for c.processNextWorkItem() {
		glog.Info("Controller.runWorker: processing next item")
	}
}

func (c *Controller) processNextWorkItem() bool {

	glog.Info("Controller.processNextItem: start")
	key, quit := c.Workqueue.Get()

	if quit {
		return false
	}

	defer c.Workqueue.Done(key)

	keyRaw := key.(string)
	_, exists, err := c.Informer.GetIndexer().GetByKey(keyRaw)

	if err != nil {
		if c.Workqueue.NumRequeues(key) < 5 {
			glog.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, retrying", key, err)
			c.Workqueue.AddRateLimited(key)
		} else {
			glog.Errorf("Controller.processNextItem: Failed processing item with key %s with error %v, no more retries", key, err)
			c.Workqueue.Forget(key)
			runtime.HandleError(err)
		}
	}

	if !exists {
		glog.Infof("Controller.processNextItem: object deleted detected: %s", keyRaw)
		c.Workqueue.Forget(key)
	} else {
		glog.Infof("Controller.processNextItem: object created detected: %s", keyRaw)
		c.Workqueue.Forget(key)
	}


	return true
}

func (c *Controller) HasSynced() bool {
	return c.Informer.HasSynced()
}