nstall operator SDK

Very helpful docs
https://sdk.operatorframework.io/docs/golang/quickstart/


Create a new Project called selenium-operator
operator-sdk new selenium-operator --repo=github.com/WianVos/selenium-operator 



https://sdk.operatorframework.io/docs/golang/references/project-layout/


Add the api for our selenium grid to the operator
operator-sdk add api --api-version=test.selenium.com/v1alpha1 --kind=SeleniumGrid



Next we need to start defining the types we need for our Selenium Grid . 
We will start simple with a grid and 2 worker nodes 

now we need to define a type that will hold the minimum specs and status we need 

```
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SeleniumGridSpec defines the desired state of SeleniumGrid
type SeleniumGridSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	ChromeNodes int32 `json:"chromeNodes"`
}

// SeleniumGridStatus defines the observed state of SeleniumGrid
type SeleniumGridStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Nodes []string `json:"nodes"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SeleniumGrid is the Schema for the seleniumgrids API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=seleniumgrids,scope=Namespaced
type SeleniumGrid struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SeleniumGridSpec   `json:"spec,omitempty"`
	Status SeleniumGridStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SeleniumGridList contains a list of SeleniumGrid
type SeleniumGridList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SeleniumGrid `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SeleniumGrid{}, &SeleniumGridList{})
}
```


After modifying the *_types.go file always run the following command to update the generated code for that resource type:

```
operator-sdk generate k8s
```

The sdk will run deepcopy procedures to generate a lot of the additional code we are going to need to actually run this operator (which we may dive into in a later post). For now just roll with what is given :-)



next we need to generate the custom resource definitions again (these can be found under /deploy in your project directory and i do encourage you to familiorize yourself with them as they explain a lot of what will happen within your kubernetes cluster later on). Also take note of the other files in the /deploy directory as they define all the stuff you need to run your operator such as the service account, role and rolebinding file. 


```
operator-sdk generate crds
```

!!! every time you update your types, please also regenerate the crds !!!

Now we are going to add the much needed controller to our project 

``` 
operator-sdk add controller --api-version=test.selenium.com/v1alpha1 --kind=SeleniumGrid 
```

This command will generate /pkg/controller/seleniumgrid/seleniumgrid_controller.go for us (amoung a few other files)

The most important part of this file that you should take note of is the reconcille function. This will run every time an event containing our api takes place. 

now the generated controller will start simple.It contains logic to start a single busybox pod and to check that it is still running. 

thing to note here is the instance variable 
```
// Fetch the SeleniumGrid instance
	instance := &testv1alpha1.SeleniumGrid{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
```

this gets instantiated as an empty testv1alpha1.SeleniumGrid type 
and then the actual requested instance is marshalled into it

now let's deploy the first version of our operator (to get some footing) and to see what it does . Now do keep in mind that you do need a running openshift cluster for this . (code ready containers will do , if u have a fast laptop and like warmth) 

```
oc login -u kubeadmin -p jh3kL-Te6cD-BKDG7-3rvSu https://api.crc.testing:6443
kubectl create -f deploy/crds/test.selenium.com_seleniumgrids_crd.yaml
```

now for development purposes we are going to run our operator outside of the cluster. 

to do this us the next procedure and adapt for your operator as needed 

```
export OPERATOR_NAME=selenium-operator
operator-sdk run --local --watch-namespace=default
```

resulting is the oerator running in the foreground and the log output will be something like this: 

```
{"level":"info","ts":1591000473.227359,"logger":"cmd","msg":"Operator Version: 0.0.1"}
{"level":"info","ts":1591000473.227412,"logger":"cmd","msg":"Go Version: go1.14.3"}
{"level":"info","ts":1591000473.227419,"logger":"cmd","msg":"Go OS/Arch: darwin/amd64"}
{"level":"info","ts":1591000473.227423,"logger":"cmd","msg":"Version of operator-sdk: v0.17.1"}
{"level":"info","ts":1591000473.229097,"logger":"leader","msg":"Trying to become the leader."}
{"level":"info","ts":1591000473.229117,"logger":"leader","msg":"Skipping leader election; not running in a cluster."}
{"level":"info","ts":1591000475.3830779,"logger":"controller-runtime.metrics","msg":"metrics server is starting to listen","addr":"0.0.0.0:8383"}
{"level":"info","ts":1591000475.383224,"logger":"cmd","msg":"Registering Components."}
{"level":"info","ts":1591000475.383351,"logger":"cmd","msg":"Skipping CR metrics server creation; not running in a cluster."}
{"level":"info","ts":1591000475.3833642,"logger":"cmd","msg":"Starting the Cmd."}
{"level":"info","ts":1591000475.3835368,"logger":"controller-runtime.manager","msg":"starting metrics server","path":"/metrics"}
{"level":"info","ts":1591000475.3835578,"logger":"controller-runtime.controller","msg":"Starting EventSource","controller":"seleniumgrid-controller","source":"kind source: /, Kind="}
{"level":"info","ts":1591000475.487744,"logger":"controller-runtime.controller","msg":"Starting EventSource","controller":"seleniumgrid-controller","source":"kind source: /, Kind="}
{"level":"info","ts":1591000475.591618,"logger":"controller-runtime.controller","msg":"Starting Controller","controller":"seleniumgrid-controller"}
{"level":"info","ts":1591000475.5916958,"logger":"controller-runtime.controller","msg":"Starting workers","controller":"seleniumgrid-controller","worker count":1}
```

now lets see what happens if we test this controller a little bit 

Luckily the sdk has created a yaml for us to use when trying the operator . 
deploy it with kubctl like so 

```
kubectl apply -f ./deploy/crds/test.selenium.com_v1alpha1_seleniumgrid_cr.yaml
```

as soon as we deploy a valid yaml the controller will start reconcilling the requested resource

and this will result in a single busybox pod being created in the designated namespace on our openshift/kubernetes cluster. 

not very functional yet , but we will get there shortly. 

now we need to think about some logic: 

we need to start a selenium grid pod .. and see if it is good and running smoothly
next we need to start worker nodes and connect them to the grid node 

first things first ... lest see if we can modify our controller so that we can see our one created pod show up as the grid node (we will keep using busy box as a mock for now) and then see if we can start additional busyboxes to represent our workernods ... again using busybox as a mock for the time being. 




so basically our generated code almost suffices. 

```
// Fetch the SeleniumGrid instance
	instance := &testv1alpha1.SeleniumGrid{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
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
	pod := newPodForGrid(instance)

	// Set SeleniumGrid instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		// if we run into an error ... return the results with an error
		if err != nil {
			return reconcile.Result{}, err
		}

		reqLogger.Info("Grid Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)

	} else if err != nil {
		return reconcile.Result{}, err
	}


    ```

    one thing i did adapt was the newPodForCR function which actually generates the object we need to pass to kubernetes to create the needed grid pod . 

    ```
    func newPodForGrid(cr *testv1alpha1.SeleniumGrid) *corev1.Pod {
	labels := map[string]string{
		"app":         cr.Name,
		"clusterRole": "grid",
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
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}

}  
```

things to note here .. i added the label clusterRole and appended the name with "-grid" 
i left the rest of the function unchanged from what was already there by name of 
newPodForCR which was generated by the SDK. 

now in addition i also created one for the worker nodes: 

``` 
func newPodForWorker(cr *testv1alpha1.SeleniumGrid) *corev1.Pod {
	labels := map[string]string{
		"app":         cr.Name,
		"clusterRole": "worker",
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName:      cr.Name + "-worker",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
``` 

again adapting the clusterRole label and name (this time we use GenerateName so kubernetes will append our input with a uuid to keep things unique)

We will change these later as we start putting in some actual selenium containers. 

now we need to add some logic to the reconciler to make sure that the workernodes are handled correctly. 

First we need to add some code to retrieve all the worker nodes from kubernetes
```
// Lets get all worker pods
	workerList := &corev1.PodList{}
	lbs := map[string]string{
		"app":         instance.Name,
		"clusterRole": "worker",
	}
	labelSelector := labels.SelectorFromSet(lbs)
	listOps := &client.ListOptions{Namespace: instance.Namespace, LabelSelector: labelSelector}
	if err = r.client.List(context.TODO(), workerList, listOps); err != nil {
		return reconcile.Result{}, err
	}
```

second we need to determine if these are running or pending (are they ready or getting ready??)

```
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
	numAvailable := int32(len(available))
	availableNames := []string{}
	for _, worker := range available {
		availableNames = append(availableNames, worker.ObjectMeta.Name)
	}
```

The next thing we need to do is to update the status field we defined earlier in the seleniumgrid_types.go file to reflect the actual worker nodes running or pending

```
// Update the status if necessary
	status := testv1alpha1.SeleniumGridStatus{
		ChromeNodeList: availableNames,
	}
	if !reflect.DeepEqual(found.Status, status) {
		instance.Status = status
		err = r.client.Status().Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update selenium worker status")
			return reconcile.Result{}, err
		}
	}
    ```

now we are going to see if the number of pods running is equal to the number of requested pods and act accordingly . We are going to take pods down if we have more then requested and we are going to create new ones if we have less. 

first lets handle to many pods 

```
// if the number of available pods is higher then the requested number of replicas then take some down
	numAvailable := int32(len(available))
	if numAvailable > instance.Spec.ChromeNodes {
		reqLogger.Info("Scaling down pods", "Currently available", numAvailable, "Required replicas", instance.Spec.ChromeNodes)
		diff := numAvailable - instance.Spec.ChromeNodes
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
```

and then 

then handle to few pods. 

```
// if the number of requested pods is more then the actual number of pods ... add some
	if numAvailable < instance.Spec.ChromeNodes {
		reqLogger.Info("Scaling up pods", "Currently available", numAvailable, "Required replicas", instance.Spec.ChromeNodes)
		// Define a new Pod object
		pod := newPodForWorker(instance)
		// Set PodSet instance as the owner and controller
		if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
			return reconcile.Result{}, err
		}
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			reqLogger.Error(err, "Failed to create pod", "pod.name", pod.Name)
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}
```

now our our controller is ready for it's first test run (in actuallity i have tested the above taken steps in a step by step way to make sure i didn't have to fix a crap load of typos and syntax/type errors once i got upto running the operator)

so go ahead and run this controller 
(make sure your logged into a openshift cluster (codeready containers will do for now))

```
export OPERATOR_NAME=selenium-operator
operator-sdk run --local --watch-namespace=default
```

now in another terminal i've also logged into the kubernetes cluster. To test if our operator is doing what it is supposed to we can make it start our grid and worker nodes by applying the generated test yaml

```
kubectl apply -f ./deploy/crds/test.selenium.com_v1alpha1_seleniumgrid_cr.yaml
```

which contains: 
```
apiVersion: test.selenium.com/v1alpha1
kind: SeleniumGrid
metadata:
  name: example-seleniumgrid
spec:
  # Add fields here
  chromeNodes: 3
```

this should make the operator start one grid and three worker pods

now if we would apply that yaml again but change the cromeNodes number it should result in het number of nodes being adjusted up or down accordingly


