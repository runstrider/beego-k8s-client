//此代码是根据已有的资源进行监控，然后实现自己逻辑进行controller
package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"time"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

//自定义controller
type MyController struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller
}

//队列item
type QueueItem struct {
	Key    string
	Type   cache.DeltaType
	Object interface{}
}

//新建controller
func NewMyController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *MyController {
	return &MyController{
		informer: informer,
		indexer:  indexer,
		queue:    queue,
	}
}

//顺序处理queue中item
func (c *MyController) processNextItem() bool {
	//从队列中pop item
	item, quite := c.queue.Get()
	if quite {
		return false
	}
	//For safe parallel processing
	defer c.queue.Done(item)

	//Logic processing
	err := c.syncToStdout(item.(QueueItem))
	// handle the err
	c.handleErr(err, item)
	return true
}

//item处理逻辑
func (c *MyController) syncToStdout(item QueueItem) error {
	//查看缓存中是否存在item
	obj, exists, err := c.indexer.Get(item.Key)
	if err != nil {
		glog.Errorf("Fetching object with key %s from store failed with %v", item.Key, err)
		return err
	}

	if !exists {
		endPoints, ok := item.Object.(*v1.Endpoints)
		if !ok {
			fmt.Printf("Key %s [%s]\n", item.Key, item.Type)
		} else {
			data, _ := json.MarshalIndent(item.Object, "", " ")
			fmt.Println("Endpoints %s [%s] key %s obj [%s]\n", endPoints.GetName(), item.Type, item.Key, string(data))
		}

	} else {
		endpoints := obj.(*v1.Endpoints)
		fmt.Println("Endpoints %s [%s]\n", endpoints.GetName(), item.Type)
	}
	return nil
}

//错误处理函数
func (c *MyController) handleErr(err error, key interface{}) {
	if err == nil {
		// Forget about the item to ensure not to delay on the key.
		c.queue.Forget(key)
		return
	}

	// This controller retries 5 times if something goes wrong. After that, it stops trying.
	if c.queue.NumRequeues(key) < 5 {
		glog.Infof("Errof syncing endpoints %v: %v", key, err)

		// Re-enqueue the key rate limited. Based on the rate limiter on the queue and the re-enqueue history, the key will be processed later again.
		//通过这个函数进行添加会使failed数组加1，用于计算NumRequeues。这里当limiter说ok的时候才会添加，而add函数则是直接添加。
		c.queue.AddRateLimited(key)
		return
	}

	// After 5 times, report to an external entity that, even after several retries, we could not successfully process this key.
	c.queue.Forget(key)
	runtime.HandleError(err)
	glog.Infof("Dropping endpoints %q out of the queue: %v", key, err)
}

func (c *MyController) Run(threads int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer c.queue.ShutDown()
	glog.Info("Starting mycontroller!")

	go c.informer.Run(stopCh)

	// 为了减少重复性工作和出错，先sync
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced) {
		runtime.HandleError(fmt.Errorf("Time out waiting for caches to sync"))
		return
	}

	for i := 0; i < threads; i++ {
		//根据传入线程数启动处理，每隔1s继续执行，知道stopCh被关闭
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<-stopCh
	glog.Info("Stopping MyController")
}

// 启动worker处理
func (c *MyController) runWorker() {
	for c.processNextItem() {

	}
}

func (c *MyController) Start() {
	kubeconfig := "/home/ubunut/.kube/config"

	// creates the client conf
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatal(err)
	}

	//create the endpoints watcher
	endpointsListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "endpoints", "kube-system", fields.Everything())

	// create the workqueue
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	//绑定queue到一个cache
	indexer, informer := cache.NewIndexerInformer(endpointsListWatcher, &v1.Endpoints{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(QueueItem{key, cache.Added, nil})
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(QueueItem{key, cache.Updated, nil})
			}

			e1 := old.(*v1.Endpoints)
			e2 := new.(*v1.Endpoints)
			e1Data, _ := json.MarshalIndent(e1, "", " ")
			e2Data, _ := json.MarshalIndent(e2, "", " ")

			fmt.Println(string(e1Data), string(e2Data))
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this key function.
			// 如果是未知错误导致删除的对行为 DeletedFinalStateUnknown, 而不是对应的对象，所以处理的时候需要特别注意
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(QueueItem{key, cache.Deleted, obj})
			}

		},
	}, cache.Indexers{})

	controller := NewMyController(queue, indexer, informer)

	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)

	//wait forever
	select {}

}
