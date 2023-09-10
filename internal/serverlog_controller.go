package internal

import (
	"context"
	"fmt"
	loginformersv1 "github.com/yshaojie/log-collector/pkg/informers/v1"
	v12 "github.com/yshaojie/log-collector/pkg/listers/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"time"
)

const (
	serverlogDeletionGracePeriod = 1 * time.Minute
)

type ServerLogStructController struct {
	client          clientset.Interface
	lister          v12.ServerLogLister
	serverLogSynced cache.InformerSynced
	workqueue       workqueue.RateLimitingInterface
	logService      LogService
}

func NewServerLogStructController(options Options, informer loginformersv1.ServerLogInformer) (*ServerLogStructController, error) {
	queue := workqueue.NewRateLimitingQueueWithConfig(workqueue.DefaultControllerRateLimiter(), workqueue.RateLimitingQueueConfig{
		Name: "serverlog",
	})
	controller := &ServerLogStructController{
		serverLogSynced: informer.Informer().HasSynced,
		workqueue:       queue,
		lister:          informer.Lister(),
	}
	informer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.enqueue(obj)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.enqueue(newObj)
		},
		DeleteFunc: func(obj interface{}) {
			controller.enqueue(obj)
		},
	})
	return controller, nil
}

func (c ServerLogStructController) RUN(ctx context.Context, workers int) {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()
	klog.Infof("Starting log-collector-controller controller")
	defer klog.Infof("Shutting down log-collector-controller controller")
	if !cache.WaitForNamedCacheSync("log-collector-controller", ctx.Done(), c.serverLogSynced) {
		return
	}

	for i := 0; i < workers; i++ {
		go wait.UntilWithContext(ctx, c.worker, 5*time.Second)
	}

	<-ctx.Done()
}

func (c ServerLogStructController) worker(ctx context.Context) {
	for c.processNextWorkItem() {
	}
}

func (c ServerLogStructController) processNextWorkItem() bool {
	item, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		err := c.syncServerLog(obj.(string))
		if err == nil {
			c.workqueue.Forget(item)
		}
		return err
	}(item)

	if err != nil {
		c.workqueue.AddRateLimited(item)
		utilruntime.HandleError(err)
	}

	return true
}

func (c ServerLogStructController) enqueue(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("Couldn't get key for object %+v: %v", obj, err))
		return
	}
	c.workqueue.Add(key)
}

func (c ServerLogStructController) syncServerLog(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}
	if err != nil {
		if errors.IsNotFound(err) {
			return err
		}
	}
	serverLog, err := c.lister.ServerLogs(namespace).Get(name)
	if serverLog == nil {
		return nil
	}
	if serverLog.GetObjectMeta().GetDeletionTimestamp() != nil {
		return c.logService.HandlerDelete(serverLog)
	} else {
		return c.logService.HandlerUpdate(serverLog)
	}
}
