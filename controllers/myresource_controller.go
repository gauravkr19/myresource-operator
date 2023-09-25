/*
Copyright 2023.

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
	gauravkr19devv1alpha1 "github.com/gauravkr19/myresource/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// var log = log.Log.WithName("my-resource-controller")

// MyResourceReconciler reconciles a MyResource object
type MyResourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=gauravkr19.dev,resources=myresources,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=gauravkr19.dev,resources=myresources/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=gauravkr19.dev,resources=myresources/finalizers,verbs=update

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the MyResource object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile

func (r *MyResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the MyResource custom resource
	myResource := &gauravkr19devv1alpha1.MyResource{}
	if err := r.Client.Get(ctx, req.NamespacedName, myResource); err != nil {
		logger.Error(err, "Failed to fetch MyResource")
		return ctrl.Result{}, err
	}

	// Create the target namespace for the workload if it doesn't exist
	if err := r.ensureNamespaceExists(ctx, myResource.Spec.TargetNamespace); err != nil {
		return ctrl.Result{}, err
	}

	// PVC Extension
	if myResource.Spec.PVCExtensionNeeded {
		if err := r.extendPVC(ctx, myResource); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create PVC; ignore error if PVC exists
	pvcName := myResource.Name + "-db" + "-pvc"
	if err := r.createPVC(ctx, myResource, pvcName); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create or update PVC")
		return ctrl.Result{}, err
	}

	// Create or update the Secret for Deployment
	secretName := myResource.Name + "-app" + "-secret"
	secKey := "somekey"
	secVal := "someval"
	if err := r.createSecret(ctx, myResource, secretName, secKey, secVal); err != nil {
		logger.Error(err, "Failed to create or update Secret")
		return ctrl.Result{}, err
	}
	// Create or update the Secret for Statefulset
	secretName = myResource.Name + "-db" + "-secret"
	if err := r.createSecret(ctx, myResource, secretName, secKey, secVal); err != nil {
		logger.Error(err, "Failed to create or update Secret")
		return ctrl.Result{}, err
	}

	// Create Service for Deployment
	deploymentServiceName := myResource.Name + "-app" + "-deployment-service"
	if err := r.createServiceApp(ctx, myResource, deploymentServiceName); err != nil {
		logger.Error(err, "Failed to create Deployment Service")
		return ctrl.Result{}, err
	}

	// Create Service for StatefulSet
	statefulSetServiceName := myResource.Name + "-db" + "-statefulset-service"
	if err := r.createServiceDB(ctx, myResource, statefulSetServiceName); err != nil {
		logger.Error(err, "Failed to create StatefulSet Service")
		return ctrl.Result{}, err
	}

	// Reconcile the Deployment
	if err := r.createOrUpdateDeployment(ctx, myResource); err != nil {
		logger.Error(err, "Failed to reconcile Deployment")
		return ctrl.Result{}, err
	}

	// Reconcile the StatefulSet
	if err := r.createOrUpdateStatefulSet(ctx, myResource); err != nil {
		logger.Error(err, "Failed to reconcile StatefulSet")
		return ctrl.Result{}, err
	}

	logger.Info("Reconciliation completed successfully")
	return ctrl.Result{}, nil
}

// ******************** End of Reconcile func ************************

// Create Namespace for the workloads if it does not exist
func (r *MyResourceReconciler) ensureNamespaceExists(ctx context.Context, namespaceName string) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespaceName,
		},
	}
	// Try to create the namespace; ignore errors if it already exists
	if err := r.Create(ctx, ns); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// Extend PVC if required
func (r *MyResourceReconciler) extendPVC(ctx context.Context, myResource *gauravkr19devv1alpha1.MyResource) error {
	logger := log.FromContext(ctx)

	// Fetch the PVC to be extended
	pvc := &corev1.PersistentVolumeClaim{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: myResource.Spec.TargetNamespace, Name: myResource.Name + "-db" + "-pvc"}, pvc); err != nil {
		return err
	}

	// Convert newSize to a resource.Quantity
	desiredSize, err := resource.ParseQuantity(myResource.Spec.NewPVCSize)
	if err != nil {
		return err
	}

	// Compare the PVC's storage size with the desired size
	currentSize := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if currentSize.Cmp(desiredSize) == 0 {
		logger.Info("PVC is already extended to the desired size")
		return nil
	}

	// Update the PVC's storage size
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = desiredSize

	if err := r.Update(ctx, pvc); err != nil {
		logger.Error(err, "Failed to update PVC")
		return err
	}

	logger.Info("PVC extended to the desired size")
	return nil
}

// DEPLOYMENT
func (r *MyResourceReconciler) createOrUpdateDeployment(ctx context.Context, myResource *gauravkr19devv1alpha1.MyResource) error {
	logger := log.FromContext(ctx)
	// dbHostVal := ""

	logger.Info("Reconciling Deployment...")

	// Define your Deployment based on myResource specifications
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myResource.Name + "-deployment-crudapp",
			Namespace: myResource.Spec.TargetNamespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &myResource.Spec.DeploymentReplicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": myResource.Name + "-app"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": myResource.Name + "-app"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  myResource.Name + "-app" + "-container",
							Image: myResource.Spec.Image,
							Env: []corev1.EnvVar{{
								Name: "POSTGRES_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: myResource.Name + "-app" + "-secret",
										},
										Key: "POSTGRES_PASSWORD",
									},
								}},
								{Name: "DB_HOST",
									Value: "myresource-sample-db-statefulset-service.myapp-namespace.svc.cluster.local",
								},
								{Name: "DB_NAME",
									Value: "my_database",
								},
								{Name: "DB_USER",
									Value: "postgres",
								},
								{Name: "DB_PASSWORD",
									Value: "postgres",
								}}, // End of Env listed values and Env definition
						},
					},
					// Volumes: []corev1.Volume{
					// 	{
					// 		Name: "secret-volume",
					// 		VolumeSource: corev1.VolumeSource{
					// 			Secret: &corev1.SecretVolumeSource{
					// 				SecretName: secretName,
					// 			},
					// 		},
					// 	},
					// },
				},
			},
		},
	}

	// Use the custom CreateOrUpdate function to create or update the Deployment
	if err := r.CreateOrUpdate(ctx, deployment); err != nil {
		logger.Error(err, "Failed to create or update Deployment")
		return err
	}
	ctrl.SetControllerReference(myResource, deployment, r.Scheme)
	logger.Info("Deployment reconciliation completed successfully")
	return nil
}

func (r *MyResourceReconciler) CreateOrUpdate(ctx context.Context, obj client.Object) error {
	logger := log.FromContext(ctx)

	logger.Info("Creating or updating resource...")

	// Try to create the resource. If it already exists, this will result in an error.
	err := r.Client.Create(ctx, obj)
	if err != nil && errors.IsAlreadyExists(err) {
		// The resource already exists; try to update it.
		err = r.Client.Update(ctx, obj)
	}

	if err != nil {
		logger.Error(err, "Failed to create or update resource")
	} else {
		logger.Info("Resource creation or update completed successfully")
	}

	return err
}

func (r *MyResourceReconciler) createSecret(ctx context.Context, myResource *gauravkr19devv1alpha1.MyResource, secretName string, secKey string, secVal string) error {
	logger := log.FromContext(ctx)
	secKey = "POSTGRES_PASSWORD"
	secVal = "postgres"

	logger.Info("Creating Secret...")

	sec := make(map[string]string)
	sec[secKey] = secVal

	// Define the Secret based on myResource specifications
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: myResource.Spec.TargetNamespace},
		Type:       corev1.SecretTypeOpaque,
		StringData: sec,
	}
	// Create the Secret
	if err := r.Client.Create(ctx, secret); err != nil {
		logger.Error(err, "Failed to create Secret")
		return err
	}

	// Set the owner reference for the
	ctrl.SetControllerReference(myResource, secret, r.Scheme)

	logger.Info("Secret creation completed successfully")
	return nil
}

// STATEFULSET
func (r *MyResourceReconciler) createOrUpdateStatefulSet(ctx context.Context, myResource *gauravkr19devv1alpha1.MyResource) error {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling StatefulSet...")

	// Define your StatefulSet based on myResource specifications
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      myResource.Name + "-db" + "-statefulset",
			Namespace: myResource.Spec.TargetNamespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    &myResource.Spec.StatefulSetReplicas,
			ServiceName: myResource.Name + "-db" + "-statefulset-service",
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": myResource.Name + "-db"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": myResource.Name + "-db"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  myResource.Name + "-db" + "-container",
							Image: myResource.Spec.ImageDB,
							Env: []corev1.EnvVar{{
								Name: "POSTGRES_PASSWORD",
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: myResource.Name + "-db" + "-secret",
										},
										Key: "POSTGRES_PASSWORD",
									},
								}},
								{Name: "POSTGRES_USER",
									Value: "postgres",
								},
								{Name: "POSTGRES_DB",
									Value: "my_database",
								}}, // End of Env listed values and Env definition
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "database-volume",
									MountPath: "/data",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "database-volume",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: myResource.Name + "-db" + "-pvc",
								},
							},
						},
					},
				},
			},
		},
	}

	// Use the custom CreateOrUpdate function to create or update the StatefulSet
	if err := r.CreateOrUpdate(ctx, statefulSet); err != nil {
		logger.Error(err, "Failed to create or update StatefulSet")
		return err
	}
	ctrl.SetControllerReference(myResource, statefulSet, r.Scheme)
	logger.Info("StatefulSet reconciliation completed successfully")
	return nil
}

func (r *MyResourceReconciler) createPVC(ctx context.Context, myResource *gauravkr19devv1alpha1.MyResource, pvcName string) error {
	logger := log.FromContext(ctx)

	logger.Info("Creating PVC...")

	// Define the PVC based on myResource specifications
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: myResource.Spec.TargetNamespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(myResource.Spec.PVCSize),
				},
			},
		},
	}
	// Create the PVC, ignore error if PVC exists
	if err := r.Client.Create(ctx, pvc); err != nil && !errors.IsAlreadyExists(err) {
		logger.Error(err, "Failed to create PVC")
		return err
	}

	logger.Info("PVC creation completed successfully")
	return nil
}

func (r *MyResourceReconciler) createServiceApp(ctx context.Context, myResource *gauravkr19devv1alpha1.MyResource, serviceName string) error {
	logger := log.FromContext(ctx)
	logger.Info("Creating Service...")

	// Define your Service based on myResource specifications
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: myResource.Spec.TargetNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": myResource.Name + "-app"},
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
					Name: "http",
				},
			},
		},
	}
	// Set the owner reference for the Service
	ctrl.SetControllerReference(myResource, service, r.Scheme)

	// Create the Service
	if err := r.Client.Create(ctx, service); err != nil {
		logger.Error(err, "Failed to create Service")
		return err
	}
	logger.Info("Service creation completed successfully")
	return nil
}
func (r *MyResourceReconciler) createServiceDB(ctx context.Context, myResource *gauravkr19devv1alpha1.MyResource, serviceName string) error {
	logger := log.FromContext(ctx)
	logger.Info("Creating Service...")
	// none type svc - Type: corev1.ServiceTypeNone,

	// Define your Service based on myResource specifications
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceName,
			Namespace: myResource.Spec.TargetNamespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": myResource.Name + "-db"},
			Type:     corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{
				{
					Port: 5432,
					Name: "tcp",
				},
			},
		},
	}
	// Set the owner reference for the Service
	ctrl.SetControllerReference(myResource, service, r.Scheme)

	// Create the Service
	if err := r.Client.Create(ctx, service); err != nil {
		logger.Error(err, "Failed to create Service")
		return err
	}
	logger.Info("Service creation completed successfully")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MyResourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&gauravkr19devv1alpha1.MyResource{}).
		Owns(&appsv1.Deployment{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.PersistentVolumeClaim{}).
		Complete(r)
}
