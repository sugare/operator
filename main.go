package main

import (
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"operator/pkg/client/clientset/versioned"
	"operator/pkg/client/informers/externalversions/demo/v1"
	"operator/pkg/controller"
	"os"
	"os/signal"
	"syscall"
)

func main()  {
	client, democlient := getKubernetesClient()

	informer := v1.NewDemoInformer(
		democlient,
		metav1.NamespaceAll,
		0,
		cache.Indexers{"namespace":cache.MetaNamespaceIndexFunc},
	)

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// convert the resource object into a key (in this case
			// we are just doing it in the format of 'namespace/name')
			key, err := cache.MetaNamespaceKeyFunc(obj)
			glog.Infof("Add Demo: %s", key)
			if err == nil {
				// add the key to the queue for the handler to get
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)

			glog.Infof("Update Demo: %s", key)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {

			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			glog.Infof("Delete Demo: %s", key)
			if err == nil {
				queue.Add(key)
			}
		},
	})
	stopCh := make(chan struct{})

	cltr := controller.Controller{
		client,
		queue,
		informer,

	}

	defer close(stopCh)

	go cltr.Run(stopCh)

	sigTerm := make(chan os.Signal, 1)
	signal.Notify(sigTerm, syscall.SIGTERM)
	signal.Notify(sigTerm, syscall.SIGINT)
	<-sigTerm
}

// retrieve the Kubernetes cluster client from outside of the cluster
func getKubernetesClient() (kubernetes.Interface, versioned.Interface) {
	// construct the path to resolve to `~/.kube/config`
	kubeConfigPath := os.Getenv("HOME") + "/.kube/config"

	// create the config from the path
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		glog.Fatalf("getClusterConfig: %v", err)
	}

	// generate the client based off of the config
	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("getClusterConfig: %v", err)
	}

	demoClient, err := versioned.NewForConfig(config)
	if err != nil {
		glog.Fatalf("getClusterConfig: %v", err)
	}

	return client, demoClient
}