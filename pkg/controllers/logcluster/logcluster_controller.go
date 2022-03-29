/*
Copyright 2021.

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

package logcluster

import (
	"context"
	"encoding/json"
	"github.com/loggie-io/loggie/pkg/core/cfg"
	"github.com/loggie-io/loggie/pkg/core/log"
	"github.com/loggie-io/loggie/pkg/core/sysconfig"
	"github.com/loggie-io/operator/pkg/api/v1beta1"
	"github.com/loggie-io/operator/pkg/constant"
	templ2 "github.com/loggie-io/operator/pkg/controllers/templ"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// LogClusterReconciler reconciles a LogCluster object
type LogClusterReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	DaemonsetTempl  *templ2.Template
	DeploymentTempl *templ2.Template
	ConfigMapTempl  *templ2.Template
}

//+kubebuilder:rbac:groups=operator.loggie.io,resources=logclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.loggie.io,resources=logclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.loggie.io,resources=logclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the LogCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *LogClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	lc := &v1beta1.LogCluster{}
	err := r.Get(ctx, req.NamespacedName, lc)
	if err != nil {
		log.Info("unable to get logCluster %s", req.NamespacedName)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if lc.DeletionTimestamp != nil {
		log.Info("logCluster %s is deleting", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// create or update DaemonSet/Deployment
	switch lc.Spec.Type {
	case constant.LogClusterTypeDeployment:
		return r.createOrUpdateDeployment(ctx, req, lc)
	case constant.LogClusterTypeDaemonSet:
		return r.createOrUpdateDaemonSet(ctx, req, lc)
	case constant.LogClusterTypeSideCar:
		// validate loggie system config
		return ctrl.Result{}, cfg.UnpackRawDefaultsAndValidate([]byte(lc.Spec.SystemConfig), &sysconfig.Config{})
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LogClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1beta1.LogCluster{}).
		Complete(r)
}

func (r *LogClusterReconciler) createOrUpdateDaemonSet(ctx context.Context, req ctrl.Request, lc *v1beta1.LogCluster) (ctrl.Result, error) {

	// create ConfigMap
	result, err := r.createConfigMap(ctx, req, lc)
	if err != nil {
		return result, err
	}

	// get DaemonSet
	ds := v1.DaemonSet{}
	if err := r.Get(ctx, req.NamespacedName, &ds); err != nil {
		if errors.IsNotFound(err) {
			// create DaemonSet
			return r.createDaemonSet(ctx, lc)
		}

		log.Info("unable to get DaemonSet %s", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// update DaemonSet
	return r.updateDaemonSet(ctx, lc)
}

func (r *LogClusterReconciler) createDaemonSet(ctx context.Context, lc *v1beta1.LogCluster) (ctrl.Result, error) {

	ds := &v1.DaemonSet{}
	if err := genResources(ctx, lc, r.DaemonsetTempl, ds); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Create(ctx, ds); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *LogClusterReconciler) updateDaemonSet(ctx context.Context, lc *v1beta1.LogCluster) (ctrl.Result, error) {

	ds := &v1.DaemonSet{}
	if err := genResources(ctx, lc, r.DaemonsetTempl, ds); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Update(ctx, ds); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *LogClusterReconciler) genDaemonSet(ctx context.Context, lc *v1beta1.LogCluster) (*v1.DaemonSet, error) {

	outYaml, err := r.DaemonsetTempl.Render(lc)
	if err != nil {
		log.Error("render template error: %v", err)
		return nil, err
	}

	ds := &v1.DaemonSet{}
	outJson, err := yaml.YAMLToJSON(outYaml)
	if err != nil {
		log.Error("yaml to json error: %v", err)
		return nil, err
	}
	err = json.Unmarshal(outJson, ds)
	if err != nil {
		log.Error("unmarshal json error: %v", err)
		return nil, err
	}

	return ds, nil
}

func genResources(ctx context.Context, lc *v1beta1.LogCluster, templ *templ2.Template, resource interface{}) error {

	outYaml, err := templ.Render(lc)
	if err != nil {
		log.Error("render template error: %v", err)
		return err
	}

	outJson, err := yaml.YAMLToJSON(outYaml)
	if err != nil {
		log.Error("yaml to json error: %v", err)
		return err
	}
	err = json.Unmarshal(outJson, resource)
	if err != nil {
		log.Error("unmarshal json error: %v", err)
		return err
	}
	return nil
}

func (r *LogClusterReconciler) createOrUpdateDeployment(ctx context.Context, req ctrl.Request, lc *v1beta1.LogCluster) (ctrl.Result, error) {

	// get Deployment
	ds := v1.Deployment{}
	if err := r.Get(ctx, req.NamespacedName, &ds); err != nil {
		if errors.IsNotFound(err) {
			// create DaemonSet
			return r.createDeployment(ctx, lc)
		}

		log.Info("unable to get Deployment %s", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	// update Deployment
	return r.updateDeployment(ctx, lc)
}

func (r *LogClusterReconciler) createDeployment(ctx context.Context, lc *v1beta1.LogCluster) (ctrl.Result, error) {
	dp := &v1.Deployment{}
	if err := genResources(ctx, lc, r.DeploymentTempl, dp); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.Client.Create(ctx, dp); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LogClusterReconciler) updateDeployment(ctx context.Context, lc *v1beta1.LogCluster) (ctrl.Result, error) {

	dp := &v1.Deployment{}
	if err := genResources(ctx, lc, r.DeploymentTempl, dp); err != nil {
		return ctrl.Result{}, err
	}
	if err := r.Client.Update(ctx, dp); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *LogClusterReconciler) createConfigMap(ctx context.Context, req ctrl.Request, lc *v1beta1.LogCluster) (ctrl.Result, error) {

	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, req.NamespacedName, cm); err != nil {
		if errors.IsNotFound(err) {
			// create configMap for loggie
			if err := genResources(ctx, lc, r.ConfigMapTempl, cm); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.Client.Create(ctx, cm); err != nil {
				return ctrl.Result{}, err
			}
		}
		log.Info("unable to get ConfigMap %s", req.NamespacedName)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}
