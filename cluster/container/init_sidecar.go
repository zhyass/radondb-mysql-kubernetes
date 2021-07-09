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

package container

import (
	"strconv"

	corev1 "k8s.io/api/core/v1"

	"github.com/radondb/radondb-mysql-kubernetes/cluster"
	"github.com/radondb/radondb-mysql-kubernetes/utils"
)

// initSidecar used for init-sidecar container.
type initSidecar struct {
	*cluster.Cluster

	// The name of the init-mysql container.
	name string
}

// getName get the container name.
func (c *initSidecar) getName() string {
	return c.name
}

// getImage get the container image.
func (c *initSidecar) getImage() string {
	return c.Spec.PodSpec.SidecarImage
}

// getCommand get the container command.
func (c *initSidecar) getCommand() []string {
	return []string{"sidecar", "init"}
}

// getEnvVars get the container env.
func (c *initSidecar) getEnvVars() []corev1.EnvVar {
	sctName := c.GetNameForResource(utils.Secret)
	envs := []corev1.EnvVar{
		{
			Name: "POD_HOSTNAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "metadata.name",
				},
			},
		},
		{
			Name:  "NAMESPACE",
			Value: c.Namespace,
		},
		{
			Name:  "SERVICE_NAME",
			Value: c.GetNameForResource(utils.HeadlessSVC),
		},
		{
			Name:  "ADMIT_DEFEAT_HEARBEAT_COUNT",
			Value: strconv.Itoa(int(*c.Spec.XenonOpts.AdmitDefeatHearbeatCount)),
		},
		{
			Name:  "ELECTION_TIMEOUT",
			Value: strconv.Itoa(int(*c.Spec.XenonOpts.ElectionTimeout)),
		},
		{
			Name:  "MY_MYSQL_VERSION",
			Value: c.GetMySQLVersion(),
		},
		getEnvVarFromSecret(sctName, "MYSQL_ROOT_PASSWORD", "root-password", false),
		getEnvVarFromSecret(sctName, "MYSQL_DATABASE", "mysql-database", true),
		getEnvVarFromSecret(sctName, "MYSQL_USER", "mysql-user", true),
		getEnvVarFromSecret(sctName, "MYSQL_PASSWORD", "mysql-password", true),
		getEnvVarFromSecret(sctName, "MYSQL_REPL_USER", "replication-user", true),
		getEnvVarFromSecret(sctName, "MYSQL_REPL_PASSWORD", "replication-password", true),
		getEnvVarFromSecret(sctName, "METRICS_USER", "metrics-user", true),
		getEnvVarFromSecret(sctName, "METRICS_PASSWORD", "metrics-password", true),
		getEnvVarFromSecret(sctName, "OPERATOR_USER", "operator-user", true),
		getEnvVarFromSecret(sctName, "OPERATOR_PASSWORD", "operator-password", true),
	}

	if c.Spec.MysqlOpts.InitTokuDB {
		envs = append(envs, corev1.EnvVar{
			Name:  "INIT_TOKUDB",
			Value: "1",
		})
	}

	return envs
}

// getLifecycle get the container lifecycle.
func (c *initSidecar) getLifecycle() *corev1.Lifecycle {
	return nil
}

// getResources get the container resources.
func (c *initSidecar) getResources() corev1.ResourceRequirements {
	return c.Spec.PodSpec.Resources
}

// getPorts get the container ports.
func (c *initSidecar) getPorts() []corev1.ContainerPort {
	return nil
}

// getLivenessProbe get the container livenessProbe.
func (c *initSidecar) getLivenessProbe() *corev1.Probe {
	return nil
}

// getReadinessProbe get the container readinessProbe.
func (c *initSidecar) getReadinessProbe() *corev1.Probe {
	return nil
}

// getVolumeMounts get the container volumeMounts.
func (c *initSidecar) getVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      utils.ConfVolumeName,
			MountPath: utils.ConfVolumeMountPath,
		},
		{
			Name:      utils.ConfMapVolumeName,
			MountPath: utils.ConfMapVolumeMountPath,
		},
		{
			Name:      utils.ScriptsVolumeName,
			MountPath: utils.ScriptsVolumeMountPath,
		},
		{
			Name:      utils.XenonVolumeName,
			MountPath: utils.XenonVolumeMountPath,
		},
		{
			Name:      utils.InitFileVolumeName,
			MountPath: utils.InitFileVolumeMountPath,
		},
	}

	if c.Spec.MysqlOpts.InitTokuDB {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      utils.SysVolumeName,
				MountPath: utils.SysVolumeMountPath,
			},
		)
	}

	if c.Spec.Persistence.Enabled {
		volumeMounts = append(volumeMounts,
			corev1.VolumeMount{
				Name:      utils.DataVolumeName,
				MountPath: utils.DataVolumeMountPath,
			},
		)
	}

	return volumeMounts
}
