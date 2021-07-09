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

package syncer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/imdario/mergo"
	"github.com/presslabs/controller-util/mergo/transformers"
	"github.com/presslabs/controller-util/syncer"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/radondb/radondb-mysql-kubernetes/cluster"
	"github.com/radondb/radondb-mysql-kubernetes/cluster/container"
	"github.com/radondb/radondb-mysql-kubernetes/utils"
)

// The wait time limit for pod upgrade.
const waitLimit = 2 * 60 * 60

// StatefulSetSyncer used to operate statefulset.
type StatefulSetSyncer struct {
	*cluster.Cluster

	cli client.Client

	sfs *appsv1.StatefulSet

	// configmap resourceVersion.
	cmRev string

	// secret resourceVersion.
	sctRev string
}

// NewStatefulSetSyncer returns a pointer to StatefulSetSyncer.
func NewStatefulSetSyncer(cli client.Client, c *cluster.Cluster, cmRev, sctRev string) *StatefulSetSyncer {
	return &StatefulSetSyncer{
		Cluster: c,
		cli:     cli,
		sfs: &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.GetNameForResource(utils.StatefulSet),
				Namespace: c.Namespace,
			},
		},
		cmRev:  cmRev,
		sctRev: sctRev,
	}
}

// Object returns the object for which sync applies.
func (s *StatefulSetSyncer) Object() interface{} { return s.sfs }

// GetObject returns the object for which sync applies
func (s *StatefulSetSyncer) GetObject() interface{} { return s.sfs }

// Owner returns the object owner or nil if object does not have one.
func (s *StatefulSetSyncer) ObjectOwner() runtime.Object { return s.Unwrap() }

// GetOwner returns the object owner or nil if object does not have one.
func (s *StatefulSetSyncer) GetOwner() runtime.Object { return s.Unwrap() }

// Sync persists data into the external store.
func (s *StatefulSetSyncer) Sync(ctx context.Context) (syncer.SyncResult, error) {
	var err error
	var kind string
	result := syncer.SyncResult{}

	result.Operation, err = s.creteOrUpdate(ctx)

	key := client.ObjectKeyFromObject(s.sfs)
	gvk, gvkErr := apiutil.GVKForObject(s.sfs, s.cli.Scheme())
	if gvkErr != nil {
		kind = fmt.Sprintf("%T", s.sfs)
	} else {
		kind = gvk.String()
	}

	if errors.Is(err, syncer.ErrOwnerDeleted) {
		log.Info(string(result.Operation), "key", key, "kind", kind, "error", err)
		err = nil
	} else if errors.Is(err, syncer.ErrIgnore) {
		log.V(1).Info("syncer skipped", "key", key, "kind", kind, "error", err)
		err = nil
	} else if err != nil {
		result.SetEventData("Warning", basicEventReason(s.Name, err),
			fmt.Sprintf("%s %s failed syncing: %s", kind, key, err))
		log.Error(err, string(result.Operation), "key", key, "kind", kind)
	} else {
		result.SetEventData("Normal", basicEventReason(s.Name, err),
			fmt.Sprintf("%s %s %s successfully", kind, key, result.Operation))
		log.V(1).Info(string(result.Operation), "key", key, "kind", kind)
	}
	return result, err
}

// creteOrUpdate creates or updates the statefulset in the Kubernetes cluster.
// see https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/controller/controllerutil?utm_source=gopls#CreateOrUpdate
func (s *StatefulSetSyncer) creteOrUpdate(ctx context.Context) (controllerutil.OperationResult, error) {
	var err error
	if err = s.cli.Get(ctx, client.ObjectKeyFromObject(s.sfs), s.sfs); err != nil {
		if !k8serrors.IsNotFound(err) {
			return controllerutil.OperationResultNone, err
		}

		if err = s.mutate(); err != nil {
			return controllerutil.OperationResultNone, err
		}

		if err = s.cli.Create(ctx, s.sfs); err != nil {
			return controllerutil.OperationResultNone, err
		} else {
			return controllerutil.OperationResultCreated, nil
		}
	}

	existing := s.sfs.DeepCopyObject()
	if err = s.mutate(); err != nil {
		return controllerutil.OperationResultNone, err
	}

	if equality.Semantic.DeepEqual(existing, s.sfs) {
		return controllerutil.OperationResultNone, nil
	}

	if err := s.cli.Update(ctx, s.sfs); err != nil {
		return controllerutil.OperationResultNone, err
	}

	if err := s.updatePod(ctx, s.sfs); err != nil {
		return controllerutil.OperationResultNone, err
	}

	return controllerutil.OperationResultUpdated, nil
}

// updatePod update the pods.
func (s *StatefulSetSyncer) updatePod(ctx context.Context, sfs *appsv1.StatefulSet) error {
	if sfs.Status.UpdatedReplicas >= sfs.Status.Replicas {
		return nil
	}

	log.Info("statefulSet was changed, run update")

	if sfs.Status.ReadyReplicas < sfs.Status.Replicas {
		log.Info("can't start/continue 'update': waiting for all replicas are ready")
		return nil
	}

	pods := corev1.PodList{}
	if err := s.cli.List(ctx,
		&pods,
		&client.ListOptions{
			Namespace:     sfs.Namespace,
			LabelSelector: s.GetLabels().AsSelector(),
		},
	); err != nil {
		return err
	}

	// get the leader pod.
	var leaderPod corev1.Pod
	for _, pod := range pods.Items {
		if pod.ObjectMeta.Labels["healthy"] != "yes" {
			return fmt.Errorf("can't start/continue 'update': pod[%s] is unhealthy", pod.Name)
		}

		if pod.ObjectMeta.Labels["role"] == "leader" && leaderPod.Name == "" {
			leaderPod = pod
			log.Info("get leader pod", "pod name", leaderPod.Name)
			continue
		}

		if err := s.applyNWait(ctx, &pod); err != nil {
			return err
		}
	}

	if leaderPod.Name != "" {
		log.Info("apply changes to leader pod", "pod name", leaderPod.Name)
		if err := s.applyNWait(ctx, &leaderPod); err != nil {
			return err
		}
	}

	log.Info("update finished")
	return nil
}

// mutate set the statefulset.
func (s *StatefulSetSyncer) mutate() error {
	s.sfs.Spec.ServiceName = s.GetNameForResource(utils.StatefulSet)
	s.sfs.Spec.Replicas = s.Spec.Replicas
	s.sfs.Spec.Selector = metav1.SetAsLabelSelector(s.GetSelectorLabels())
	s.sfs.Spec.UpdateStrategy = appsv1.StatefulSetUpdateStrategy{
		Type: appsv1.OnDeleteStatefulSetStrategyType,
	}

	s.sfs.Spec.Template.ObjectMeta.Labels = s.GetLabels()
	for k, v := range s.Spec.PodSpec.Labels {
		s.sfs.Spec.Template.ObjectMeta.Labels[k] = v
	}
	s.sfs.Spec.Template.ObjectMeta.Labels["role"] = "candidate"
	s.sfs.Spec.Template.ObjectMeta.Labels["healthy"] = "no"

	s.sfs.Spec.Template.Annotations = s.Spec.PodSpec.Annotations
	if len(s.sfs.Spec.Template.ObjectMeta.Annotations) == 0 {
		s.sfs.Spec.Template.ObjectMeta.Annotations = make(map[string]string)
	}
	if s.Spec.MetricsOpts.Enabled {
		s.sfs.Spec.Template.ObjectMeta.Annotations["prometheus.io/scrape"] = "true"
		s.sfs.Spec.Template.ObjectMeta.Annotations["prometheus.io/port"] = fmt.Sprintf("%d", utils.MetricsPort)
	}
	s.sfs.Spec.Template.ObjectMeta.Annotations["config_rev"] = s.cmRev
	s.sfs.Spec.Template.ObjectMeta.Annotations["secret_rev"] = s.sctRev

	err := mergo.Merge(&s.sfs.Spec.Template.Spec, s.ensurePodSpec(), mergo.WithTransformers(transformers.PodSpec))
	if err != nil {
		return err
	}
	s.sfs.Spec.Template.Spec.Tolerations = s.Spec.PodSpec.Tolerations

	if s.Spec.Persistence.Enabled {
		if s.sfs.Spec.VolumeClaimTemplates, err = s.EnsureVolumeClaimTemplates(s.cli.Scheme()); err != nil {
			return err
		}
	}

	// set owner reference only if owner resource is not being deleted, otherwise the owner
	// reference will be reset in case of deleting with cascade=false.
	if s.Unwrap().GetDeletionTimestamp().IsZero() {
		if err := controllerutil.SetControllerReference(s.Unwrap(), s.sfs, s.cli.Scheme()); err != nil {
			return err
		}
	} else if ctime := s.Unwrap().GetCreationTimestamp(); ctime.IsZero() {
		// the owner is deleted, don't recreate the resource if does not exist, because gc
		// will not delete it again because has no owner reference set
		return fmt.Errorf("owner is deleted")
	}
	return nil
}

// ensurePodSpec used to ensure the podspec.
func (s *StatefulSetSyncer) ensurePodSpec() corev1.PodSpec {
	initSidecar := container.EnsureContainer(utils.ContainerInitSidecarName, s.Cluster)
	initMysql := container.EnsureContainer(utils.ContainerInitMysqlName, s.Cluster)
	initContainers := []corev1.Container{initSidecar, initMysql}

	mysql := container.EnsureContainer(utils.ContainerMysqlName, s.Cluster)
	xenon := container.EnsureContainer(utils.ContainerXenonName, s.Cluster)
	containers := []corev1.Container{mysql, xenon}
	if s.Spec.MetricsOpts.Enabled {
		containers = append(containers, container.EnsureContainer(utils.ContainerMetricsName, s.Cluster))
	}
	if s.Spec.PodSpec.SlowLogTail {
		containers = append(containers, container.EnsureContainer(utils.ContainerSlowLogName, s.Cluster))
	}
	if s.Spec.PodSpec.SlowLogTail {
		containers = append(containers, container.EnsureContainer(utils.ContainerAuditLogName, s.Cluster))
	}

	return corev1.PodSpec{
		InitContainers:     initContainers,
		Containers:         containers,
		Volumes:            s.EnsureVolumes(),
		SchedulerName:      s.Spec.PodSpec.SchedulerName,
		ServiceAccountName: s.GetNameForResource(utils.ServiceAccount),
		Affinity:           s.Spec.PodSpec.Affinity,
		PriorityClassName:  s.Spec.PodSpec.PriorityClassName,
		Tolerations:        s.Spec.PodSpec.Tolerations,
	}
}

func (s *StatefulSetSyncer) applyNWait(ctx context.Context, pod *corev1.Pod) error {
	if pod.ObjectMeta.Labels["controller-revision-hash"] == s.sfs.Status.UpdateRevision {
		log.Info("pod is already updated", "pod name", pod.Name)
	} else {
		if err := s.cli.Delete(ctx, pod); err != nil {
			return err
		}
	}

	// wait the pod restart.
	return retry(time.Second*10, time.Duration(waitLimit)*time.Second, func() (bool, error) {
		err := s.cli.Get(ctx, types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, pod)
		if err != nil && !k8serrors.IsNotFound(err) {
			return false, err
		}

		var xenonReady, mysqlReady bool
		for _, container := range pod.Status.ContainerStatuses {
			if container.Name == "xenon" {
				xenonReady = container.Ready
			}
			if container.Name == "mysql" {
				mysqlReady = container.Ready
			}
		}

		if pod.Status.Phase == corev1.PodFailed {
			return false, fmt.Errorf("pod %s is in failed phase", pod.Name)
		}

		if pod.Status.Phase == corev1.PodRunning && xenonReady && mysqlReady &&
			pod.ObjectMeta.Labels["controller-revision-hash"] == s.sfs.Status.UpdateRevision &&
			pod.ObjectMeta.Labels["healthy"] == "yes" {
			log.Info("pod is running", "pod name", pod.Name)
			return true, nil
		}

		return false, nil
	})
}

// retry runs func "f" every "in" time until "limit" is reached
// it also doesn't have an extra tail wait after the limit is reached
// and f func runs first time instantly
func retry(in, limit time.Duration, f func() (bool, error)) error {
	fdone, err := f()
	if err != nil {
		return err
	}
	if fdone {
		return nil
	}

	done := time.NewTimer(limit)
	defer done.Stop()
	tk := time.NewTicker(in)
	defer tk.Stop()

	for {
		select {
		case <-done.C:
			return fmt.Errorf("reach pod wait limit")
		case <-tk.C:
			fdone, err := f()
			if err != nil {
				return err
			}
			if fdone {
				return nil
			}
		}
	}
}

func basicEventReason(objKindName string, err error) string {
	if err != nil {
		return fmt.Sprintf("%sSyncFailed", strcase.ToCamel(objKindName))
	}

	return fmt.Sprintf("%sSyncSuccessfull", strcase.ToCamel(objKindName))
}
