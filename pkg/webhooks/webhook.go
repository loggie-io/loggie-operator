package webhooks

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/loggie-io/operator/pkg/api/v1beta1"
	"github.com/loggie-io/operator/pkg/constant"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"strings"
)

func SetupWithWebhook(mgr ctrl.Manager) {
	hookServer := mgr.GetWebhookServer()
	mgr.GetLogger().Info("Register MutatingWebhook /loggie-pod-inject-sidecar")
	hookServer.Register("/loggie-pod-inject-sidecar", &webhook.Admission{
		Handler: &PodInject{
			Client: mgr.GetClient(),
		},
	})
}

type PodInject struct {
	client.Client
	decoder *admission.Decoder
}

func (r *PodInject) InjectDecoder(d *admission.Decoder) error {
	r.decoder = d
	return nil
}

func (r *PodInject) Handle(ctx context.Context, req admission.Request) admission.Response {
	// watch pod
	pod := &corev1.Pod{}
	lcs := v1beta1.LogClusterList{}
	if err := r.decoder.Decode(req, pod); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}
	for _, c := range pod.Spec.Containers {
		if c.Name == constant.LoggieOperatorAgentName {
			// agent already exists
			marshaledPod, err := json.Marshal(pod)
			if err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
			return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
		}
	}
	if err := r.List(ctx, &lcs); err == nil {
		for _, lc := range lcs.Items {
			if lc.Spec.Type != constant.LogClusterTypeSideCar {
				continue
			}
			pipeCms := corev1.ConfigMapList{}
			// Gets a ConfigMap with the same name as LogCluster
			if err := r.List(ctx, &pipeCms, client.HasLabels{constant.LoggieOperatorLabel}); err != nil {
				return admission.Errored(http.StatusBadRequest, fmt.Errorf("ConfigMap %s  not found", lc.Name))
			}
			for _, pipeCm := range pipeCms.Items {
				if pipeCm.Name == lc.Name && hasLabels(pod.Labels, pipeCm.Labels) {
					// inject loggie container
					pipeCfg, ok := pipeCm.Data[constant.AutoCreateConfigMapData]
					if !ok {
						return admission.Errored(http.StatusBadRequest, fmt.Errorf("pipelines.yaml config %s  not found in  %s", constant.AutoCreateConfigMapData, pipeCm.Name))
					}
					loggieContainer := corev1.Container{}
					loggieContainer.Name = constant.LoggieOperatorAgentName
					loggieContainer.Image = constant.LoggieAgentImageName
					loggieContainer.Env = append(loggieContainer.Env, corev1.EnvVar{
						Name:  constant.AutoCreateSystemConfigMapData,
						Value: lc.Spec.SystemConfig,
					})
					loggieContainer.Env = append(loggieContainer.Env, corev1.EnvVar{
						Name:  constant.AutoCreateConfigMapData,
						Value: pipeCfg,
					})
					for _, c := range pod.Spec.Containers {
						//loggie and business container mount the same log volume
						loggieContainer.VolumeMounts = append(loggieContainer.VolumeMounts, c.VolumeMounts...)
					}
					// data persistence
					pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
						Name: "register",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					})
					// TODO loggie args
					//loggieContainer.Args = append(loggieContainer.Args, []string{
					//	"-meta.nodeName=$(NODE_NAME)",
					//	"-config.system=loggie.yaml",
					//	"-config.pipeline=pipelines.yaml",
					//	"-config.type=env",
					//}...)
					loggieContainer.VolumeMounts = append(loggieContainer.VolumeMounts, corev1.VolumeMount{
						Name:      "registry",
						MountPath: "/loggie",
					})

					pod.Spec.Containers = append(pod.Spec.Containers, loggieContainer)
				}
			}

		}
	}

	// marshal pod
	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

func getNameSpaceEev() string {
	ns := os.Getenv("NAMESPACE")
	if ns == "" {
		return "loggie-system"
	}
	return ns
}

func hasLabels(podLabel map[string]string, cmLabel map[string]string) bool {
	found := false
	for cmK, cmV := range cmLabel {
		if !strings.HasPrefix(constant.LoggieOperatorLabel+"-", cmK) {
			continue
		}
		if podV, ok := podLabel[strings.SplitN(cmK, "-", 2)[1]]; !ok || podV != cmV {
			return false
		}
		found = true
	}
	return found
}
