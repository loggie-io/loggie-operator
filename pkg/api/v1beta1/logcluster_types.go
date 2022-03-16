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

package v1beta1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	LogClusterTypeDaemonSet  = "DaemonSet"
	LogClusterTypeDeployment = "Deployment"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LogClusterSpec defines the desired state of LogCluster
type LogClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of LogCluster. Edit logcluster_types.go to remove/update
	Image        string                  `json:"image,omitempty"`
	Type         string                  `json:"type,omitempty"`
	Replicas     int                     `json:"replicas,omitempty"` // only type=Deployment
	Resources    v1.ResourceRequirements `json:"resources,omitempty"`
	NodeSelector map[string]string       `json:"nodeSelector,omitempty"`
	Volumes      Volumes                 `json:"volumes"`
}

type Volumes struct {
	PipelineConfigDir string   `json:"pipelineConfigDir,omitempty"`
	KubeletRootDir    string   `json:"kubeletRootDir,omitempty"`
	DockerRootDir     string   `json:"dockerRootDir,omitempty"`
	HostPaths         []string `json:"hostPaths,omitempty"`
}

// LogClusterStatus defines the observed state of LogCluster
type LogClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// LogCluster is the Schema for the logclusters API
type LogCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LogClusterSpec   `json:"spec,omitempty"`
	Status LogClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// LogClusterList contains a list of LogCluster
type LogClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LogCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LogCluster{}, &LogClusterList{})
}
