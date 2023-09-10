package internal

import (
	"errors"
	apiv1 "github.com/yshaojie/log-collector/api/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/klog/v2"
)

type LogService struct {
}

func (s LogService) RUN(oncomplete func()) {
	if oncomplete != nil {
		oncomplete()
	}
}

func (s LogService) SyncHandler(log apiv1.ServerLog) {

}

func (s LogService) HandlerDelete(log *apiv1.ServerLog) error {
	klog.V(5).Infof("servet log %q is deleted...", log.Name)
	if rand.Int()%1 == 0 {
		return errors.New("handler error...")
	}
	return nil
}

func (s LogService) HandlerUpdate(log *apiv1.ServerLog) error {
	klog.Infof("servet log %q is updated...", log.Name)
	if rand.Int()%1 == 0 {
		return errors.New("handler error...")
	}
	return nil
}
