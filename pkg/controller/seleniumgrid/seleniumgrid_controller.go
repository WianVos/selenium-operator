package seleniumgrid

import (
	"context"
	"fmt"
	"reflect"
	"time"

	testv1alpha1 "github.com/WianVos/selenium-operator/pkg/apis/test/v1alpha1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_seleniumgrid")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new SeleniumGrid Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileSeleniumGrid{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("seleniumgrid-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource SeleniumGrid
	err = c.Watch(&source.Kind{Type: &testv1alpha1.SeleniumGrid{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner SeleniumGrid
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &testv1alpha1.SeleniumGrid{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileSeleniumGrid implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileSeleniumGrid{}

// ReconcileSeleniumGrid reconciles a SeleniumGrid object
type ReconcileSeleniumGrid struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a SeleniumGrid object and makes changes based on the state read
// and what is in the SeleniumGrid.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileSeleniumGrid) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling SeleniumGrid")

	// Fetch the SeleniumGrid instance
	gridInstance := &testv1alpha1.SeleniumGrid{}
	err := r.client.Get(context.TODO(), request.NamespacedName, gridInstance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Start by handeling the grid pod first
	// Define a new Pod object
	gridPod := newPodForGrid(gridInstance)

	// Set SeleniumGrid instance as the owner and controller
	if err := controllerutil.SetControllerReference(gridInstance, gridPod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: gridPod.Name, Namespace: gridPod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", gridPod.Namespace, "Pod.Name", gridPod.Name)
		err = r.client.Create(context.TODO(), gridPod)
		// if we run into an error ... return the results with an error
		if err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Grid Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)

	} else if err != nil {
		return reconcile.Result{}, err
	}

	// check if the requested number of worker pods are actually running

	// Next lets handle the workers ..
	// we can only start the workers once the grid hub is available and reachable by means of a service
	// so that we can register the workers with the hub
	// if the hub is not available we are going to return and requeue this request

	if waitForPodReady(found, 10, 5) == false {
		reqLogger.Info("Requeing while waiting for Pod readiness", "Pod.Namespace", gridPod.Namespace, "Pod.Name", gridPod.Name)
		return reconcile.Result{Requeue: true}, nil
	}

	// once the hub is ready we are going to devine a service
	gridService := newServiceForHub(gridInstance)

	// Set SeleniumGrid instance as the owner and controller
	if err := controllerutil.SetControllerReference(gridInstance, gridService, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	foundService := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: gridService.Name, Namespace: gridService.Namespace}, foundService)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service", "Pod.Namespace", gridService.Namespace, "Pod.Name", gridService.Name)
		err = r.client.Create(context.TODO(), gridService)
		// if we run into an error ... return the results with an error
		if err != nil {
			reqLogger.Error(err, "Failed to create service", "service.name", gridService.Name)
			return reconcile.Result{}, err
		}

		reqLogger.Info("Grid Service already exists", "Pod.Namespace", foundService.Namespace, "Pod.Name", foundService.Name)

	} else if err != nil {
		return reconcile.Result{}, err
	}

	// lets attach a service so that users can reach the grid easily
	reqLogger.Info("TEST")
	routeToService := newRouteToService(gridInstance)

	if err := controllerutil.SetControllerReference(gridInstance, routeToService, r.scheme); err != nil {
		fmt.Printf("%+v\n", routeToService)

		return reconcile.Result{}, corev1.ErrIntOverflowGenerated
	}

	foundRouteToService := &routev1.Route{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: gridService.Name, Namespace: gridService.Namespace}, foundRouteToService)

	reqLogger.Info("TEST2")

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new route to service", gridService.Namespace, "Pod.Name", gridService.Name)
		err = r.client.Create(context.TODO(), routeToService)
		// if we run into an error ... return the results with an error
		if err != nil {
			reqLogger.Error(err, "Failed to create route to service", routeToService.Name, routeToService.Name)
			return reconcile.Result{}, err
		}

		reqLogger.Info("Route to Service already exists", "Pod.Namespace", foundService.Namespace, "Pod.Name", foundService.Name)

	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Lets get all worker pods
	workerList := &corev1.PodList{}
	lbs := map[string]string{
		"app":         gridInstance.Name,
		"clusterRole": "worker",
	}
	labelSelector := labels.SelectorFromSet(lbs)
	listOps := &client.ListOptions{Namespace: gridInstance.Namespace, LabelSelector: labelSelector}
	if err = r.client.List(context.TODO(), workerList, listOps); err != nil {
		return reconcile.Result{}, err
	}

	// Count the workers that are pending or running as available
	var available []corev1.Pod
	for _, worker := range workerList.Items {
		if worker.ObjectMeta.DeletionTimestamp != nil {
			continue
		}
		if worker.Status.Phase == corev1.PodRunning || worker.Status.Phase == corev1.PodPending {
			available = append(available, worker)
		}
	}
	availableNames := []string{}
	for _, worker := range available {
		availableNames = append(availableNames, worker.ObjectMeta.Name)
	}

	// Update the status if necessary
	status := testv1alpha1.SeleniumGridStatus{
		ChromeNodeList: availableNames,
	}
	if !reflect.DeepEqual(found.Status, status) {
		gridInstance.Status = status
		err = r.client.Status().Update(context.TODO(), gridInstance)
		if err != nil {
			reqLogger.Error(err, "Failed to update selenium worker status")
			return reconcile.Result{}, err
		}
	}

	// if the number of available pods is higher then the requested number of replicas then take some down
	numAvailable := int32(len(available))
	if numAvailable > gridInstance.Spec.ChromeNodes {
		reqLogger.Info("Scaling down pods", "Currently available", numAvailable, "Required replicas", gridInstance.Spec.ChromeNodes)
		diff := numAvailable - gridInstance.Spec.ChromeNodes
		dpods := available[:diff]
		for _, dpod := range dpods {
			err = r.client.Delete(context.TODO(), &dpod)
			if err != nil {
				reqLogger.Error(err, "Failed to delete pod", "pod.name", dpod.Name)
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// if the number of requested pods is more then the actual number of pods ... add some
	if numAvailable < gridInstance.Spec.ChromeNodes {
		reqLogger.Info("Scaling up pods", "Currently available", numAvailable, "Required replicas", gridInstance.Spec.ChromeNodes)
		// Define a new Pod object
		pod := newPodForWorker(gridInstance, gridPod.Name)
		// Set PodSet instance as the owner and controller
		if err := controllerutil.SetControllerReference(gridInstance, pod, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			reqLogger.Error(err, "Failed to create pod", "pod.name", pod.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	return reconcile.Result{}, nil
}

func newPodForGrid(cr *testv1alpha1.SeleniumGrid) *corev1.Pod {

	n := cr.Name
	v := cr.Spec.HubVersion
	p := int32(4444)
	cImage := "selenium/hub:" + v

	labels := map[string]string{
		"app":         n,
		"clusterRole": "grid",
		"version":     v,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-grid",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  n + "-hub",
					Image: cImage,
					Ports: []corev1.ContainerPort{{
						ContainerPort: p,
						Name:          "selenium",
					}},
					Resources: getResourceRequirements(getResourceList(cr.Spec.HubCPU, cr.Spec.HubMemory), getResourceList("", "")),
					LivenessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/wd/hub/status",
								Port: intstr.FromInt(int(p)),
							},
						},
						InitialDelaySeconds: 30,
						TimeoutSeconds:      5,
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path: "/wd/hub/status",
								Port: intstr.FromInt(int(p)),
							},
						},
						InitialDelaySeconds: 1,
						TimeoutSeconds:      5,
					},
				},
			},
		},
	}

}

func newPodForWorker(cr *testv1alpha1.SeleniumGrid, g string) *corev1.Pod {
	n := cr.Name
	v := cr.Spec.HubVersion
	p := int32(5555)
	cImage := "selenium/node-chrome-debug:" + v

	labels := map[string]string{
		"app":         cr.Name,
		"clusterRole": "worker",
		"version":     v,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: cr.Name + "-worker",
			Namespace:    cr.Namespace,
			Labels:       labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  n + "-node-chrome",
					Image: cImage,
					Ports: []corev1.ContainerPort{{
						ContainerPort: p,
					}},
					// VolumeMounts: []corev1.VolumeMount{{
					// 	MountPath: "/dev/shm",
					// 	Name:      "dshm",
					// }},
					Env: []corev1.EnvVar{{
						Name:  "HUB_HOST",
						Value: n,
					}, {
						Name:  "HUB_PORT",
						Value: "4444",
					}},
					Resources: getResourceRequirements(getResourceList(cr.Spec.HubCPU, cr.Spec.HubMemory), getResourceList("", "")),
				},
			},
		},
	}
}

func newServiceForHub(cr *testv1alpha1.SeleniumGrid) *corev1.Service {
	p := int32(4444)

	labels := map[string]string{
		"app": cr.Name,
	}

	selector := map[string]string{
		"app":         cr.Name,
		"clusterRole": "grid",
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{{
				Port:       p,
				TargetPort: intstr.FromInt(int(p)),
			}},
		},
	}
}

func newRouteToService(cr *testv1alpha1.SeleniumGrid) *routev1.Route {

	var termination routev1.TLSTerminationType

	termination = routev1.TLSTerminationEdge

	labels := map[string]string{
		"app": cr.Name,
	}

	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: cr.Name,
			},
			TLS: &routev1.TLSConfig{
				Termination: termination,
			},

			// Port: routev1.RoutePort{{
			// 	TargetPort: intstr.FromInt(int(p)),
			// }},
		},
	}

}

//func newRouteToService
///
// //spec:
//       volumes:
//       - name: dshm
//         emptyDir:
//           medium: Memory
//       containers:
//       - name: selenium-node-chrome
//         image: selenium/node-chrome-debug:3.141
//         ports:
//           - containerPort: 5555
//         volumeMounts:
//           - mountPath: /dev/shm
//             name: dshm
//         env:
//           - name: HUB_HOST
//             value: "selenium-hub"
//           - name: HUB_PORT
//             value: "4444"
//         resources:
//           limits:
//             memory: "1000Mi"
// 			cpu: ".5"
// /

//Helper methods below

func getResourceList(cpu, memory string) v1.ResourceList {
	res := v1.ResourceList{}
	if cpu != "" {
		res[v1.ResourceCPU] = resource.MustParse(cpu)
	}
	if memory != "" {
		res[v1.ResourceMemory] = resource.MustParse(memory)
	}
	return res
}

func getResourceRequirements(requests, limits v1.ResourceList) v1.ResourceRequirements {
	res := v1.ResourceRequirements{}
	res.Requests = requests
	res.Limits = limits
	return res
}

func podReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			if c.Status == corev1.ConditionTrue {
				return true
			}
		}
	}
	return false
}

func waitForPodReady(p *corev1.Pod, t, r int) bool {
	for i := 0; r > i; i++ {
		if podReady(p) {
			return true
		}
		time.Sleep(time.Duration(t) * time.Second)
	}
	return false
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
// func newPodForCR(cr *testv1alpha1.SeleniumGrid) *corev1.Pod {
// 	labels := map[string]string{
// 		"app": cr.Name,
// 	}
// 	return &corev1.Pod{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      cr.Name + "-pod",
// 			Namespace: cr.Namespace,
// 			Labels:    labels,
// 		},
// 		Spec: corev1.PodSpec{
// 			Containers: []corev1.Container{
// 				{
// 					Name:    "busybox",
// 					Image:   "busybox",
// 					Command: []string{"sleep", "3600"},
// 				},
// 			},
// 		},
// 	}
// }
