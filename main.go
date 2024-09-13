package main

import (
	"flag"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

// Controller 使用client-go实现控制器
type Controller struct {
	indexer cache.Indexer
	queue   workqueue.RateLimitingInterface
	informer cache.Controller
}


// NewController 创建一个新的控制器
func NewController(queue workqueue.RateLimitingInterface, indexer cache.Indexer, informer cache.Controller) *Controller {
	return &Controller{
		informer: informer,
		indexer: indexer,
		queue: queue,
	}
}

func (c *Controller) processNetItem() bool {
	// 等待,直到工作队列有新项目
	key, quit := c.queue.Get()
	if quit{
		return false
	}

	// 告诉队列我们已经完成了这个键值的处理,并发安全的
	defer c.queue.Done(key)


	// 实现自己的业务逻辑,这里打印key的名称
	err := c.syncToStdout(key.(string))
	// 如果在业务逻辑执行过程中出现错误,则处理错误
	c.handleErr(err, key)
	return false

}

// syncToStdout是控制器的业务逻辑. 在这个控制器中,它只是打印
// 将pod的信息发送给stdout.发生错误时,它只是简单的返回错误.
func (c *Controller) syncToStdout(key string) error {
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		klog.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}

	if !exists{
		fmt.Printf("Pod %s does not exist anymore\n", key)
	} else {
		fmt.Printf("Sync/Add/Update for Pod %s\n", obj.(*v1.Pod).GetName())
	}

	return nil
}

// handlerErr检查是否发生了错误,并确保稍后重试
func (c *Controller) handleErr(err error, key interface{}){
	if err == nil {
		// 本质上是从队列的限流器中去除当前key,意思是标记当前key处理完成
		c.queue.Forget(key)
		return
	}

	// 如果出现错误,控制器会重试5次,在那之后,它停止尝试
	if c.queue.NumRequeues(key) < 5{
		klog.Infof("Error syncing pod %v: %v",key, err)


		// 重新入队进行排队
		c.queue.AddRateLimited(key)
		return
	}

	c.queue.Forget(key)
	// 在多次重试之后,仍然无法成功处理当前key,从而抛出错误
	runtime.HandleError(err)
	klog.Infof("Dropping pod %q out of the queue: %v", key, err)
}

// 开始观察和同步,这里触发Reflector进行list&watch
func (c *Controller) Run(threadindess int, stopCh chan struct{})  {
	defer runtime.HandleCrash()
	defer c.queue.ShuttingDown()
	klog.Info("Starting Pod controller")

	go c.informer.Run(stopCh)

	// 在开始处理队列中的项之前,等待所有涉及的缓存先同步完成
	if !cache.WaitForCacheSync(stopCh, c.informer.HasSynced){
		runtime.HandleError(fmt.Errorf("Timed out wating for caches to sync"))
		return
	}

	for i :=  0; i < threadindess; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	<- stopCh
	klog.Infof("Stopping Pod controller")

}

func (c *Controller) runWorker()  {
	for c.processNetItem(){

	}
}

func main()  {
	var kubeconfig string
	var master string

	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "", "master url")
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	// 创建一个对pod 资源进行 List&Watcher的实例
	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", v1.NamespaceDefault, fields.Everything())

	// 创建一个工作队列workqueue,使用默认限流策略
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())

	// 将工作队列workQueue绑定到缓存Indexer中.这样我们就能确保每当缓存更新是,pod键就会添加到workqueue中.
	indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer使用增量队列,因此对于删除,我们也需要将key传递到队列中
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil{
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	controller := NewController(queue, indexer, informer)

	// 启动控制器
	stop := make(chan struct{})
	defer close(stop)
	go controller.Run(1, stop)
	select {

	}
}

