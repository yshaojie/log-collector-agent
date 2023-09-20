package v2

import (
	"context"
	v1 "github.com/yshaojie/log-collector/api/v1"
	"github.com/yshaojie/log-collector/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	"log-collector-agent/internal"
	"sigs.k8s.io/controller-runtime/pkg/client"
	reconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type serverlogController struct {
	client     client.Client
	logService internal.LogService
}

func (c *serverlogController) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	serverLog := &v1.ServerLog{}
	err := c.client.Get(ctx, request.NamespacedName, serverLog)
	if errors.IsNotFound(err) {
		return reconcile.Result{}, nil
	}
	if err != nil {
		return reconcile.Result{}, err
	}
	//处理删除逻辑
	if !serverLog.ObjectMeta.DeletionTimestamp.IsZero() {
		return c.HandleDelete(serverLog)
	}

	return c.HandleUpdate(serverLog)
}

func (c serverlogController) HandleUpdate(serverlog *v1.ServerLog) (reconcile.Result, error) {
	err := c.logService.HandlerUpdate(serverlog)
	if err != nil {
		return reconcile.Result{}, err
	}
	serverlog.Status.Phase = v1.ServerLogRunning
	err = c.client.Status().Update(context.TODO(), serverlog)
	//更新解决冲突
	if errors.IsConflict(err) {
		return reconcile.Result{Requeue: true}, nil
	}
	return reconcile.Result{}, err
}

func (c serverlogController) HandleDelete(serverlog *v1.ServerLog) (reconcile.Result, error) {
	err := c.logService.HandlerDelete(serverlog)
	if err != nil {
		return reconcile.Result{}, err
	}
	serverlog.Finalizers = removeFsyncFinalizer(serverlog.Finalizers, utils.FinalizerNameAgentHolder)
	err = c.client.Update(context.TODO(), serverlog)
	//更新解决冲突
	if errors.IsConflict(err) {
		return reconcile.Result{Requeue: true}, nil
	}
	return reconcile.Result{}, err
}

func removeFsyncFinalizer(finalizers []string, removed string) []string {
	if finalizers == nil {
		return []string{}
	}
	var result []string
	for _, finalizer := range finalizers {
		if finalizer == removed {
			continue
		}

		result = append(result, finalizer)
	}
	return result
}
