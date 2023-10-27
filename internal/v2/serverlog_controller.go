package v2

import (
	"context"
	v1 "github.com/yshaojie/log-collector/api/v1"
	"github.com/yshaojie/log-collector/pkg/utils"
	"k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"log-collector-agent/internal"
	"sigs.k8s.io/controller-runtime/pkg/client"
	controllerutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/finalizer"
	reconcile "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type serverlogController struct {
	client     client.Client
	logService internal.LogService
	finalizers finalizer.Finalizers
}

var _ finalizer.Finalizer = &serverlogController{}
var _ reconcile.Reconciler = &serverlogController{}

func NewServerlogController(client client.Client) *serverlogController {
	c := &serverlogController{
		client:     client,
		logService: internal.LogService{},
		finalizers: finalizer.NewFinalizers(),
	}
	c.finalizers.Register(utils.FinalizerNameAgentHolder, c)
	return c
}

func (c *serverlogController) Finalize(ctx context.Context, object client.Object) (finalizer.Result, error) {
	log, ok := object.(*v1.ServerLog)
	if !ok {
		return finalizer.Result{}, errors.FromObject(log)
	}
	//清理外部资源
	err := c.deleteExternalResources(log)
	return finalizer.Result{}, err
}

func (c *serverlogController) deleteExternalResources(log *v1.ServerLog) error {
	return c.logService.HandlerDelete(log)
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
	updated := false
	statusUpdated := false
	var errs []error
	finalize, err := c.finalizers.Finalize(ctx, serverLog)
	errs = append(errs, err)
	updated = updated || finalize.Updated
	statusUpdated = statusUpdated || finalize.StatusUpdated
	if serverLog.GetDeletionTimestamp().IsZero() {
		update, err := c.HandleUpdate(serverLog)
		updated = update || updated
		errs = append(errs, err)
	}

	if updated {
		err = c.client.Update(context.TODO(), serverLog)
		errs = append(errs, err)
	}
	if statusUpdated {
		err = c.client.Status().Update(context.TODO(), serverLog)
		errs = append(errs, err)
	}

	return reconcile.Result{}, kerrors.NewAggregate(errs)
}

func (c serverlogController) HandleUpdate(serverlog *v1.ServerLog) (bool, error) {
	err := c.logService.HandlerUpdate(serverlog)
	if err != nil {
		return false, err
	}
	serverlog.Status.Phase = v1.ServerLogRunning
	return true, err
}

func (c serverlogController) HandleDelete(serverlog *v1.ServerLog) (reconcile.Result, error) {
	err := c.logService.HandlerDelete(serverlog)
	if err != nil {
		return reconcile.Result{}, err
	}
	updated := controllerutil.RemoveFinalizer(serverlog, utils.FinalizerNameAgentHolder)
	if updated {
		err = c.client.Update(context.TODO(), serverlog)
		//更新解决冲突
		if errors.IsConflict(err) {
			return reconcile.Result{Requeue: true}, nil
		}
	}

	return reconcile.Result{}, err
}
