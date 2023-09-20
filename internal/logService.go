package internal

import (
	apiv1 "github.com/yshaojie/log-collector/api/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
)

type LogService struct {
}

func (s LogService) RUN(oncomplete func()) {
	if oncomplete != nil {
		oncomplete()
	}
}

func (s LogService) HandlerDelete(log *apiv1.ServerLog) error {
	key, err := cache.MetaNamespaceKeyFunc(log)
	if err != nil {
		return err
	}
	klog.Infof("servet log %q is deleted...", key)
	return nil
}

func (s LogService) HandlerUpdate(log *apiv1.ServerLog) error {
	key, err := cache.MetaNamespaceKeyFunc(log)
	if err != nil {
		return err
	}
	klog.Infof("servet log %q is updated...", key)
	return nil
}
