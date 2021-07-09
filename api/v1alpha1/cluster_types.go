/*
Copyright 2021 RadonDB.

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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Replicas is the number of pods.
	// +optional
	// +kubebuilder:validation:Enum=0;2;3;5
	// +kubebuilder:default:=3
	Replicas *int32 `json:"replicas,omitempty"`

	// MysqlOpts is the options of MySQL container.
	// +optional
	// +kubebuilder:default:={rootPassword: "", rootHost: "127.0.0.1", user: "qc_usr", password: "Qing@123", database: "qingcloud", initTokuDB: true, resources: {limits: {cpu: "500m", memory: "1Gi"}, requests: {cpu: "100m", memory: "256Mi"}}}
	MysqlOpts MysqlOpts `json:"mysqlOpts,omitempty"`

	// XenonOpts is the options of xenon container.
	// +optional
	// +kubebuilder:default:={image: "zhyass/xenon:1.1.5-alpha", admitDefeatHearbeatCount: 5, electionTimeout: 10000, resources: {limits: {cpu: "100m", memory: "256Mi"}, requests: {cpu: "50m", memory: "128Mi"}}}
	XenonOpts XenonOpts `json:"xenonOpts,omitempty"`

	// XenonOpts is the options of metrics container.
	// +optional
	// +kubebuilder:default:={image: "prom/mysqld-exporter:v0.12.1", resources: {limits: {cpu: "100m", memory: "128Mi"}, requests: {cpu: "10m", memory: "32Mi"}}, enabled: false}
	MetricsOpts MetricsOpts `json:"metricsOpts,omitempty"`

	// Represents the MySQL version that will be run. The available version can be found here:
	// This field should be set even if the Image is set to let the operator know which mysql version is running.
	// Based on this version the operator can take decisions which features can be used.
	// +optional
	// +kubebuilder:default:="5.7"
	MysqlVersion string `json:"mysqlVersion,omitempty"`

	// Pod extra specification.
	// +optional
	// +kubebuilder:default:={imagePullPolicy: "IfNotPresent", resources: {requests: {cpu: "10m", memory: "32Mi"}}, sidecarImage: "zhyass/sidecar:0.1", busyboxImage: "busybox:1.32"}
	PodSpec PodSpec `json:"podSpec,omitempty"`

	// PVC extra specifiaction.
	// +optional
	// +kubebuilder:default:={enabled: true, accessModes: {"ReadWriteOnce"}, size: "10Gi"}
	Persistence Persistence `json:"persistence,omitempty"`
}

// MysqlOpts defines the options of MySQL container.
type MysqlOpts struct {
	// Password for the root user.
	// +optional
	// +kubebuilder:default:=""
	RootPassword string `json:"rootPassword,omitempty"`

	// The root user's host.
	// +optional
	// +kubebuilder:validation:Enum="127.0.0.1";"%"
	// +kubebuilder:default:="127.0.0.1"
	RootHost string `json:"rootHost,omitempty"`

	// Username of new user to create.
	// +optional
	// +kubebuilder:default:="qc_usr"
	User string `json:"user,omitempty"`

	// Password for the new user.
	// +optional
	// +kubebuilder:default:="Qing@123"
	Password string `json:"password,omitempty"`

	// Name for new database to create.
	// +optional
	// +kubebuilder:default:="qingcloud"
	Database string `json:"database,omitempty"`

	// InitTokuDB represents if install tokudb engine.
	// +optional
	// +kubebuilder:default:=true
	InitTokuDB bool `json:"initTokuDB,omitempty"`

	// A map[string]string that will be passed to my.cnf file.
	// +optional
	MysqlConf MysqlConf `json:"mysqlConf,omitempty"`

	// The compute resource requirements.
	// +optional
	// +kubebuilder:default:={limits: {cpu: "500m", memory: "1Gi"}, requests: {cpu: "100m", memory: "256Mi"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// XenonOpts defines the options of xenon container.
type XenonOpts struct {
	// To specify the image that will be used for xenon container.
	// +optional
	// +kubebuilder:default:="zhyass/xenon:1.1.5-alpha"
	Image string `json:"image,omitempty"`

	// High available component admit defeat heartbeat count.
	// +optional
	// +kubebuilder:default:=5
	AdmitDefeatHearbeatCount *int32 `json:"admitDefeatHearbeatCount,omitempty"`

	// High available component election timeout. The unit is millisecond.
	// +optional
	// +kubebuilder:default:=10000
	ElectionTimeout *int32 `json:"electionTimeout,omitempty"`

	// The compute resource requirements.
	// +optional
	// +kubebuilder:default:={limits: {cpu: "100m", memory: "256Mi"}, requests: {cpu: "50m", memory: "128Mi"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

// MetricsOpts defines the options of metrics container.
type MetricsOpts struct {
	// To specify the image that will be used for metrics container.
	// +optional
	// +kubebuilder:default:="prom/mysqld-exporter:v0.12.1"
	Image string `json:"image,omitempty"`

	// The compute resource requirements.
	// +optional
	// +kubebuilder:default:={limits: {cpu: "100m", memory: "128Mi"}, requests: {cpu: "10m", memory: "32Mi"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Enabled represents if start a metrics container.
	// +optional
	// +kubebuilder:default:=false
	Enabled bool `json:"enabled,omitempty"`
}

// MysqlConf defines type for extra cluster configs. It's a simple map between
// string and string.
type MysqlConf map[string]string

// PodSpec defines type for configure cluster pod spec.
type PodSpec struct {
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +kubebuilder:default:="IfNotPresent"
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	Labels            map[string]string   `json:"labels,omitempty"`
	Annotations       map[string]string   `json:"annotations,omitempty"`
	Affinity          *corev1.Affinity    `json:"affinity,omitempty"`
	PriorityClassName string              `json:"priorityClassName,omitempty"`
	Tolerations       []corev1.Toleration `json:"tolerations,omitempty"`
	SchedulerName     string              `json:"schedulerName,omitempty"`

	// The compute resource requirements.
	// +optional
	// +kubebuilder:default:={requests: {cpu: "10m", memory: "32Mi"}}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// To specify the image that will be used for sidecar container.
	// +optional
	// +kubebuilder:default:="zhyass/sidecar:0.1"
	SidecarImage string `json:"sidecarImage,omitempty"`

	// The busybox image.
	// +optional
	// +kubebuilder:default:="busybox:1.32"
	BusyboxImage string `json:"busyboxImage,omitempty"`

	// SlowLogTail represents if tail the mysql slow log.
	// +optional
	// +kubebuilder:default:=false
	SlowLogTail bool `json:"slowLogTail,omitempty"`

	// AuditLogTail represents if tail the mysql audit log.
	// +optional
	// +kubebuilder:default:=false
	AuditLogTail bool `json:"auditLogTail,omitempty"`
}

// Persistence is the desired spec for storing mysql data. Only one of its
// members may be specified.
type Persistence struct {
	// Create a volume to store data.
	// +optional
	// +kubebuilder:default:=true
	Enabled bool `json:"enabled,omitempty"`

	// AccessModes contains the desired access modes the volume should have.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1
	// +optional
	// +kubebuilder:default:={"ReadWriteOnce"}
	AccessModes []corev1.PersistentVolumeAccessMode `json:"accessModes,omitempty"`

	// Name of the StorageClass required by the claim.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`

	//Size of persistent volume claim.
	// +optional
	// +kubebuilder:default:="10Gi"
	Size string `json:"size,omitempty"`
}

// ClusterConditionType defines type for cluster condition type.
type ClusterConditionType string

const (
	// ClusterInit indicates whether the cluster is initializing.
	ClusterInit ClusterConditionType = "Initializing"
	// ClusterReady indicates whether all containers in the pod are ready.
	ClusterReady ClusterConditionType = "Ready"
	// ClusterError indicates whether the cluster encountered an error.
	ClusterError ClusterConditionType = "Error"
)

// ClusterCondition defines type for cluster conditions.
type ClusterCondition struct {
	// Type of cluster condition, values in (\"Initializing\", \"Ready\", \"Error\").
	Type ClusterConditionType `json:"type"`
	// Status of the condition, one of (\"True\", \"False\", \"Unknown\").
	Status corev1.ConditionStatus `json:"status"`

	// The last time this Condition type changed.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
	// One word, camel-case reason for current status of the condition.
	Reason string `json:"reason,omitempty"`
	// Full text reason for current status of the condition.
	Message string `json:"message,omitempty"`
}

// NodeStatus defines type for status of a node into cluster.
type NodeStatus struct {
	// Name of the node.
	Name string `json:"name"`
	// Full text reason for current status of the node.
	Message string `json:"message,omitempty"`
	// Conditions contains the list of the node conditions fulfilled.
	Conditions []NodeCondition `json:"conditions,omitempty"`
}

// NodeCondition defines type for representing node conditions.
type NodeCondition struct {
	// Type of the node condition.
	Type NodeConditionType `json:"type"`
	// Status of the node, one of (\"True\", \"False\", \"Unknown\").
	Status corev1.ConditionStatus `json:"status"`
	// The last time this Condition type changed.
	LastTransitionTime metav1.Time `json:"lastTransitionTime"`
}

// NodeConditionType defines type for node condition type.
type NodeConditionType string

const (
	// NodeConditionLagged represents if the node is lagged.
	NodeConditionLagged NodeConditionType = "Lagged"
	// NodeConditionLeader represents if the node is leader or not.
	NodeConditionLeader NodeConditionType = "Leader"
	// NodeConditionReadOnly repesents if the node is read only or not
	NodeConditionReadOnly NodeConditionType = "ReadOnly"
	// NodeConditionReplicating represents if the node is replicating or not.
	NodeConditionReplicating NodeConditionType = "Replicating"
)

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// ReadyNodes represents number of the nodes that are in ready state.
	ReadyNodes int `json:"readyNodes,omitempty"`
	// State
	State ClusterConditionType `json:"state,omitempty"`
	// Conditions contains the list of the cluster conditions fulfilled.
	Conditions []ClusterCondition `json:"conditions,omitempty"`
	// Nodes contains the list of the node status fulfilled.
	Nodes []NodeStatus `json:"nodes,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.replicas,statuspath=.status.readyNodes
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type == 'Ready')].status",description="The cluster status"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".spec.replicas",description="The number of desired nodes"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:shortName=mysql
// Cluster is the Schema for the clusters API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterSpec   `json:"spec,omitempty"`
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Cluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
