/*


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

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	webappv1 "operator/example/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GuestbookReconciler reconciles a Guestbook object
type GuestbookReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestbooks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=webapp.my.domain,resources=guestbooks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;list;watch;create;update;patch;delete

func (r *GuestbookReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	_ = context.Background()
	_ = r.Log.WithValues("guestbook", req.NamespacedName)
	webapp := &webappv1.Guestbook{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, webapp)
	if err != nil {
		if errors.IsNotFound(err) {
			r.Log.Info("webapp resouce not found, Ignoring since object must be deleted")
			return reconcile.Result{}, nil
		}

		r.Log.Error(err, "faild to get webapp")
		return reconcile.Result{}, err
	}
	found := &appsv1.Deployment{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: webapp.Name, Namespace: webapp.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		dep := r.DeploymentForWebapp(webapp)
		r.Log.Info("creating a new Deployment", "Deployment.namespace", dep.Namespace, "Deployment.Name", dep.Name)
		err = r.Client.Create(context.TODO(), dep)
		if err != nil {
			r.Log.Error(err, "Failed to create new Deployment", "Deployment.Namespace", dep.Namespace, "Deployment.Name", dep.Name)
			return reconcile.Result{}, err
		}
	}
	if found.Spec.Replicas != &webapp.Spec.Size {
		found.Spec.Replicas = &webapp.Spec.Size
		err = r.Client.Update(context.TODO(), found)
		if err != nil {
			r.Log.Error(err, "Failed to update deployment", "Deployment.Namespace", found.Namespace, "Deployment.Name", found.Name)
			return reconcile.Result{}, err
		}

	}
	// your logic here
	return ctrl.Result{}, nil
}

func (r *GuestbookReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&webappv1.Guestbook{}).
		Owns(&appsv1.Deployment{}).
		Complete(r)
}

func (r *GuestbookReconciler) DeploymentForWebapp(app *webappv1.Guestbook) *appsv1.Deployment {
	ls := labels(app.Name)
	replicas := app.Spec.Size
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Image: "nginx",
						Name:  "webapp",
					}},
				},
			},
		},
	}
	controllerutil.SetControllerReference(app, dep, r.Scheme)
	return dep
}

func labels(name string) map[string]string {
	return map[string]string{"app": "webapp", "cr": name}
}
