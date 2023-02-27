/*
Copyright 2023 Loggie.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logconfig

import (
	"context"
	"github.com/loggie-io/loggie/pkg/core/log"
	logconfigv1beta1 "github.com/loggie-io/loggie/pkg/discovery/kubernetes/apis/loggie/v1beta1"
	"github.com/loggie-io/operator/pkg/config"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	Config *config.Config
	client.Client
	Scheme *runtime.Scheme
}

func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Info("reconciling logConfig %s", req.NamespacedName)

	lgc := &logconfigv1beta1.LogConfig{}
	err := r.Get(ctx, req.NamespacedName, lgc)
	if err != nil {
		log.Info("unable to get logConfig %s", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if lgc.DeletionTimestamp != nil {
		log.Info("logConfig %s is deleting", req.NamespacedName)

		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&logconfigv1beta1.LogConfig{}).
		Owns(&logconfigv1beta1.Sink{}).
		Owns(&logconfigv1beta1.Interceptor{}).
		Complete(r)
}
