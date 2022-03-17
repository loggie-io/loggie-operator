package logconfig

import (
	"context"
	"github.com/loggie-io/loggie/pkg/core/log"
	logconfigv1beta1 "github.com/loggie-io/loggie/pkg/discovery/kubernetes/apis/loggie/v1beta1"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	AnnotationAutoCreateKey               = "loggie.io/create"
	AnnotationCreateSidecarConfigMapValue = "configmap"

	AutoCreateConfigMapData = "pipelines.yml"
)

type Reconciler struct {
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
		// we would not delete configMap here, instead, adding deletion label and timestamp annotation
		return ctrl.Result{}, nil
	}

	if err = r.createSidecarConfigMap(ctx, lgc, req); err != nil {
		log.Warn("create configMap for sidecar failed: %v", err)
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

type PipelineRawConfig struct {
	Pipelines []ConfigRaw `yaml:"pipelines"`
}

type ConfigRaw struct {
	Name string `yaml:"name,omitempty"`

	Interceptors []map[string]interface{} `yaml:"interceptors,omitempty"`
	Sources      []map[string]interface{} `yaml:"sources,omitempty"`
	Sink         map[string]interface{}   `yaml:"sink,omitempty"`
}

func (r *Reconciler) createSidecarConfigMap(ctx context.Context, lgc *logconfigv1beta1.LogConfig, req ctrl.Request) error {
	if lgc.Annotations[AnnotationAutoCreateKey] != AnnotationCreateSidecarConfigMapValue {
		return nil
	}

	cm, err := r.lgc2cm(ctx, lgc)
	if err != nil {
		log.Warn("create or update configMap from logConfig %s error: %v", req.NamespacedName, err)
		return nil
	}

	getCm := &corev1.ConfigMap{}
	if err = r.Get(ctx, types.NamespacedName{Namespace: cm.Namespace, Name: cm.Name}, getCm); err != nil {
		if !errors.IsNotFound(err) {
			log.Warn("get configMap error: %v", err)
			return err
		}

		// create configMap
		err = r.Create(ctx, cm)
		if err != nil {
			log.Warn("create configMap error: %v", err)
			return err
		}

		log.Info("create configMap %s success", req.NamespacedName)
		return nil
	}

	// update configMap
	if err = r.Update(ctx, cm); err != nil {
		log.Warn("update configMap error: %v", err)
		return err
	}
	log.Info("update configMap %s success", req.NamespacedName)

	return nil
}

func (r *Reconciler) lgc2cm(ctx context.Context, lgc *logconfigv1beta1.LogConfig) (*corev1.ConfigMap, error) {
	sources := lgc.Spec.Pipeline.Sources
	src := make([]map[string]interface{}, 0)
	err := yaml.Unmarshal([]byte(sources), &src)
	if err != nil {
		log.Warn("Unmarshal sources error: %v", err)
		return nil, err
	}

	interceptor, err := r.getInterceptorRef(ctx, lgc.Spec.Pipeline.InterceptorRef)
	if err != nil {
		return nil, err
	}

	sink, err := r.getSinkRef(ctx, lgc.Spec.Pipeline.SinkRef)
	if err != nil {
		return nil, err
	}

	pipelineConfig := &PipelineRawConfig{}

	var pipelines []ConfigRaw
	pipe := ConfigRaw{
		Name:         lgc.Name,
		Sources:      src,
		Interceptors: interceptor,
		Sink:         sink,
	}
	pipelines = append(pipelines, pipe)
	pipelineConfig.Pipelines = pipelines

	cm := &corev1.ConfigMap{}
	cm.Name = lgc.Name
	cm.Namespace = lgc.Namespace

	out, err := yaml.Marshal(pipelineConfig)
	if err != nil {
		log.Error("marshal pipeline error: %v", err)
		return nil, err
	}

	data := make(map[string]string)
	data[AutoCreateConfigMapData] = string(out)
	cm.Data = data

	return cm, nil
}

func (r *Reconciler) getInterceptorRef(ctx context.Context, interceptorRef string) ([]map[string]interface{}, error) {
	if interceptorRef == "" {
		return nil, nil
	}

	icp := &logconfigv1beta1.Interceptor{}
	if err := r.Get(ctx, types.NamespacedName{Name: interceptorRef}, icp); err != nil {
		log.Warn("get interceptor %s error: %v", interceptorRef, err)
		return nil, err
	}

	m := make([]map[string]interface{}, 0)
	if err := yaml.Unmarshal([]byte(icp.Spec.Interceptors), &m); err != nil {
		log.Warn("unmarshal error: %v", err)
		return nil, err
	}

	return m, nil
}

func (r *Reconciler) getSinkRef(ctx context.Context, sinkRef string) (map[string]interface{}, error) {
	if sinkRef == "" {
		return nil, nil
	}

	sink := &logconfigv1beta1.Sink{}
	if err := r.Get(ctx, types.NamespacedName{Name: sinkRef}, sink); err != nil {
		log.Warn("get sink %s error: %v", sinkRef, err)
		return nil, err
	}

	m := make(map[string]interface{}, 0)
	if err := yaml.Unmarshal([]byte(sink.Spec.Sink), &m); err != nil {
		log.Warn("unmarshal error: %v", err)
		return nil, err
	}

	return m, nil
}
