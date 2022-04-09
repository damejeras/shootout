/*
Copyright 2022.

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

package controllers

import (
	"context"
	"strconv"
	"strings"

	cowboysv1 "github.com/damejeras/shootout/operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	indexField = ".metadata.controller"
)

// ShootoutReconciler reconciles a Shootout object
type ShootoutReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=cowboys.mejeras.lt,resources=shootouts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cowboys.mejeras.lt,resources=shootouts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cowboys.mejeras.lt,resources=shootouts/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ShootoutReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	shootout := &cowboysv1.Shootout{}
	if err := r.Get(ctx, req.NamespacedName, shootout); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "unable to fetch Shootout")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var podList corev1.PodList
	if err := r.List(ctx, &podList, client.InNamespace(req.Namespace), client.MatchingFields{indexField: req.Name}); err != nil {
		logger.Error(err, "unable to list child Pods")
		return ctrl.Result{}, err
	}

	var redisPodIP, arbiterPodIP string
	var redisPod, arbiterPod *corev1.Pod

	for i := range podList.Items {
		switch podList.Items[i].Name {
		case shootout.Name + "-redis":
			redisPod = &podList.Items[i]
		case shootout.Name + "-arbiter":
			arbiterPod = &podList.Items[i]
		}
	}

	if redisPod == nil {
		podDef := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      shootout.Name + "-redis",
				Namespace: shootout.Namespace,
				Labels: map[string]string{
					"shootout": shootout.Name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "redis",
						Image: "redis",
					},
				},
				RestartPolicy: corev1.RestartPolicyOnFailure,
			},
		}

		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, podDef, func() error {
			return ctrl.SetControllerReference(shootout, podDef, r.Scheme)
		}); err != nil {
			logger.Error(err, "error creating redis pod")
			return ctrl.Result{}, err
		}

		return r.Update(shootout)
	} else {
		if redisPod.Status.Phase == corev1.PodRunning {
			redisPodIP = redisPod.Status.PodIP
		}
	}

	if redisPodIP == "" {
		// we have to wait for redis to start
		return r.Update(shootout)
	} else if arbiterPod == nil {
		// lets start arbiter pod
		podDef := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      shootout.Name + "-arbiter",
				Namespace: shootout.Namespace,
				Labels: map[string]string{
					"shootout": shootout.Name,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:  "arbiter",
						Image: "shootout-arbiter:latest",
						Env: []corev1.EnvVar{
							{
								Name:  "COMPETITORS",
								Value: strconv.Itoa(len(shootout.Spec.Shooters)),
							},
							{
								Name:  "REDIS_ADDR",
								Value: redisPodIP + ":6379",
							},
						},
						ImagePullPolicy: corev1.PullNever,
					},
				},
				RestartPolicy: corev1.RestartPolicyNever,
			},
		}

		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, podDef, func() error {
			return ctrl.SetControllerReference(shootout, podDef, r.Scheme)
		}); err != nil {
			logger.Error(err, "error creating arbiter pod")
			return ctrl.Result{}, err
		}

		return r.Update(shootout)
	} else {
		if arbiterPod.Status.Phase == corev1.PodRunning {
			arbiterPodIP = arbiterPod.Status.PodIP
		}
	}

	if arbiterPodIP == "" {
		// we have to wait for arbiter to start
		return r.Update(shootout)
	}

	var containers []corev1.Container
	// we have arbiter and redis running, lets start shooter pods
	for i := range shootout.Spec.Shooters {
		containers = append(containers, corev1.Container{
			Name:  "shooter-" + strings.ToLower(shootout.Spec.Shooters[i].Name),
			Image: "shootout-shooter:latest",
			Env: []corev1.EnvVar{
				{
					Name:  "COMPETITORS",
					Value: strconv.Itoa(len(shootout.Spec.Shooters)),
				},
				{
					Name:  "REDIS_ADDR",
					Value: redisPodIP + ":6379",
				},
				{
					Name:  "ARBITER_ADDR",
					Value: "http://" + arbiterPodIP + ":8888",
				},
				{
					Name:  "SHOOTER_NAME",
					Value: shootout.Spec.Shooters[i].Name,
				},
				{
					Name:  "SHOOTER_HEALTH",
					Value: strconv.Itoa(shootout.Spec.Shooters[i].Health),
				},
				{
					Name:  "SHOOTER_DAMAGE",
					Value: strconv.Itoa(shootout.Spec.Shooters[i].Damage),
				},
			},
			ImagePullPolicy: corev1.PullNever,
		})
	}

	shooterPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      shootout.Name + "-shooters",
			Namespace: shootout.Namespace,
			Labels: map[string]string{
				"shootout": shootout.Name,
			},
		},
		Spec: corev1.PodSpec{
			Containers:    containers,
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	if _, err := ctrl.CreateOrUpdate(ctx, r.Client, shooterPod, func() error {
		return ctrl.SetControllerReference(shootout, shooterPod, r.Scheme)
	}); err != nil {
		logger.Error(err, "error creating shooter pod")
		return ctrl.Result{}, err
	}

	// TODO: collect corpses

	return r.Update(shootout)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ShootoutReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, indexField, func(rawObj client.Object) []string {
		pod := rawObj.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != cowboysv1.GroupVersion.String() || owner.Kind != "Shootout" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&cowboysv1.Shootout{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

func (r *ShootoutReconciler) Update(shootout *cowboysv1.Shootout) (ctrl.Result, error) {
	if err := r.Status().Update(context.Background(), shootout); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
