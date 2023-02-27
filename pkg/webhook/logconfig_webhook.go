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

package webhook

import (
	"context"
	"fmt"
	"github.com/loggie-io/loggie/pkg/core/cfg"
	"github.com/loggie-io/loggie/pkg/core/log"
	logconfigv1beta1 "github.com/loggie-io/loggie/pkg/discovery/kubernetes/apis/loggie/v1beta1"
	"github.com/loggie-io/operator/pkg/config"
	"github.com/loggie-io/operator/pkg/utils/files"
	"github.com/loggie-io/operator/pkg/utils/kubernetes"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	InjectorAnnotationKey       = "sidecar.loggie.io/inject"
	InjectorAnnotationValueTrue = "true"

	SidecarContainerName = "loggie"

	EnvKeySystem   = "loggie_config"
	EnvKeyPipeline = "pipeline_config"
)

type SidecarInjection struct {
	Config *config.Sidecar
	client.Client
	decoder *admission.Decoder
}

func (s *SidecarInjection) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := s.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	if !CheckInject(pod.ObjectMeta, s.Config.IgnoreNamespaces) {
		return admission.Allowed("allowed but would not inject Loggie sidecar")
	}

	mutatePod := pod.DeepCopy()
	lgc, paths, err := s.getMatchedLogConfig(mutatePod)
	if err != nil {
		w := fmt.Sprintf("cannot get Pod(%s/%s) matched LogConfig/ClusterLogConfig: %v", mutatePod.Namespace, mutatePod.GenerateName, err)
		log.Warn(w)
		return admission.Allowed("allowed but would not inject Loggie sidecar, " + w)
	}
	if lgc == nil {
		w := fmt.Sprintf("Pod(%s/%s) does not have a matching logconfig/clusterLogConfig", mutatePod.Namespace, mutatePod.GenerateName)
		log.Warn(w)
		return admission.Allowed("allowed but would not inject Loggie sidecar, " + w)
	}

	if err := s.patchWithModeEnv(mutatePod, lgc, paths); err != nil {
		log.Warn("inject pod %s/%s sidecar failed: %v", mutatePod.Namespace, mutatePod.GenerateName, err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	marshaledPod, err := json.Marshal(mutatePod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	log.Info("injecting pod, namespace: %s, GenerateName: %s", mutatePod.Namespace, mutatePod.GenerateName)
	log.Debug("injecting pod yaml: %s", string(marshaledPod))

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// podAnnotator implements admission.DecoderInjector.
// A decoder will be automatically injected.

// InjectDecoder injects the decoder.
func (s *SidecarInjection) InjectDecoder(d *admission.Decoder) error {
	s.decoder = d
	return nil
}

func CheckInject(meta metav1.ObjectMeta, ignoredNamespaces []string) bool {
	if meta.Annotations == nil {
		return false
	}

	if meta.Annotations[InjectorAnnotationKey] != InjectorAnnotationValueTrue {
		return false
	}

	// check namespace
	for _, ns := range ignoredNamespaces {
		if meta.Namespace == ns {
			return false
		}
	}

	return true
}

func (s *SidecarInjection) addPodSidecar(pod *corev1.Pod) error {
	lgc, paths, err := s.getMatchedLogConfig(pod)
	if err != nil {
		return err
	}
	if lgc == nil {
		return errors.New("cannot find pod related logConfig/ClusterLogConfig")
	}
	return s.patchWithModeEnv(pod, lgc, paths)
}

func (s *SidecarInjection) patchWithModeEnv(pod *corev1.Pod, logConfig *logconfigv1beta1.LogConfig, paths []string) error {
	var mounts []corev1.VolumeMount
	var volumes []corev1.Volume
	registryMount, registryVol := registryVolumes()
	logMount, logVol := logVolumes(pod, paths, s.Config.IgnoreContainerNames)
	mounts = append(mounts, registryMount)
	mounts = append(mounts, logMount...)
	volumes = append(volumes, registryVol)
	volumes = append(volumes, logVol...)

	envs, err := configEnvs(logConfig, s.Client, s.Config.SystemConfig)
	if err != nil {
		return err
	}

	sidecar := corev1.Container{
		Name: SidecarContainerName,
		Args: []string{
			"-config.from=env",
			fmt.Sprintf("-config.system=%s", EnvKeySystem),
			fmt.Sprintf("-config.pipeline=%s", EnvKeyPipeline),
		},
		Image:        s.Config.Image,
		Env:          envs,
		VolumeMounts: mounts,
	}
	pod.Spec.Containers = append(pod.Spec.Containers, sidecar)
	pod.Spec.Volumes = append(pod.Spec.Volumes, volumes...)

	return nil
}

// add registry volumeMount and volume
func registryVolumes() (corev1.VolumeMount, corev1.Volume) {
	registryVolName := "registry"
	registryMount := corev1.VolumeMount{
		Name:      registryVolName,
		MountPath: "/data",
	}
	registryVol := corev1.Volume{
		Name: registryVolName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	return registryMount, registryVol
}

// add paths volume for app container and sidecar
func logVolumes(pod *corev1.Pod, paths []string, ignoreContainerNames []string) ([]corev1.VolumeMount, []corev1.Volume) {
	var mounts []corev1.VolumeMount
	var volumes []corev1.Volume
	logPaths := files.CommonPath(paths)
	for i := 0; i < len(logPaths); i++ {
		logVolName := fmt.Sprintf("loggie-logs-%d", i)
		logMount := corev1.VolumeMount{
			Name:      logVolName,
			MountPath: logPaths[i],
		}
		logVol := corev1.Volume{
			Name: logVolName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}

		// add log volumeMounts to app container
		for j, container := range pod.Spec.Containers {
			// if skip mount log volume to this container
			for _, c := range ignoreContainerNames {
				if container.Name == c {
					continue
				}
			}

			applogMount := corev1.VolumeMount{
				Name:      logVolName,
				MountPath: logPaths[i],
			}

			pod.Spec.Containers[j].VolumeMounts = append(pod.Spec.Containers[j].VolumeMounts, applogMount)
		}

		mounts = append(mounts, logMount)
		volumes = append(volumes, logVol)
	}

	return mounts, volumes
}

func configEnvs(logConfig *logconfigv1beta1.LogConfig, cli client.Client, systemConfig string) ([]corev1.EnvVar, error) {
	var envs []corev1.EnvVar
	systemEnv := corev1.EnvVar{
		Name:  EnvKeySystem,
		Value: systemConfig,
	}
	pipes, err := kubernetes.LogConfigToPipelineStr(logConfig, cli)
	if err != nil {
		return nil, err
	}
	pipelineEnv := corev1.EnvVar{
		Name:  EnvKeyPipeline,
		Value: pipes,
	}
	envs = append(envs, systemEnv, pipelineEnv)

	return envs, nil
}

func (s *SidecarInjection) getMatchedLogConfig(pod *corev1.Pod) (logConfig *logconfigv1beta1.LogConfig, path []string, e error) {
	// find pod matched LogConfig
	lgc, err := s.podMatchedLogConfigs(pod)
	if err != nil {
		return nil, nil, err
	}
	if lgc == nil {
		clgc, err := s.podMatchedClusterLogConfigs(pod)
		if err != nil {
			return nil, nil, err
		}
		if clgc == nil {
			return nil, nil, nil
		}
		lgc = clgc.ToLogConfig()
	}

	paths, err := retrievePathsFromSource(lgc.Spec.Pipeline.Sources)
	if err != nil {
		return nil, nil, err
	}

	return lgc, paths, nil
}

type pathsInFileSource struct {
	Paths []string `yaml:"paths,omitempty"`
}

func retrievePathsFromSource(sources string) ([]string, error) {
	src := make([]pathsInFileSource, 0)
	if err := cfg.UnPackFromRaw([]byte(sources), &src).Do(); err != nil {
		return nil, err
	}

	var result []string
	for _, s := range src {
		for _, p := range s.Paths {
			if p == logconfigv1beta1.PathStdout {
				return nil, errors.New("pod stdout logs is not supported in loggie sidecar")
			}
		}

		result = append(result, s.Paths...)
	}

	return result, nil
}

func (s *SidecarInjection) podMatchedLogConfigs(pod *corev1.Pod) (*logconfigv1beta1.LogConfig, error) {
	lgcList := &logconfigv1beta1.LogConfigList{}
	if err := s.Client.List(context.Background(), lgcList, &client.ListOptions{}); err != nil {
		return nil, err
	}

	for _, lgc := range lgcList.Items {
		if lgc.Spec.Selector == nil || lgc.Spec.Selector.Type != logconfigv1beta1.SelectorTypePod {
			continue
		}

		if kubernetes.LabelsSubset(lgc.Spec.Selector.LabelSelector, pod.Labels) {
			return lgc.DeepCopy(), nil
		}
	}

	return nil, nil
}

func (s *SidecarInjection) podMatchedClusterLogConfigs(pod *corev1.Pod) (*logconfigv1beta1.ClusterLogConfig, error) {
	clgcList := &logconfigv1beta1.ClusterLogConfigList{}
	if err := s.Client.List(context.Background(), clgcList, &client.ListOptions{}); err != nil {
		return nil, err
	}

	for _, lgc := range clgcList.Items {
		if lgc.Spec.Selector == nil || lgc.Spec.Selector.Type != logconfigv1beta1.SelectorTypePod {
			continue
		}

		if kubernetes.LabelsSubset(lgc.Spec.Selector.LabelSelector, pod.Labels) {
			return lgc.DeepCopy(), nil
		}
	}

	return nil, nil
}
