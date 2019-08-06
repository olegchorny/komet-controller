package komet

import (
	"context"
	"time"
	"strings"
	cloudv1alpha1 "github.com/plerionio/komet-controller/pkg/apis/cloud/v1alpha1"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_komet")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Komet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileKomet{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("komet-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Komet
	err = c.Watch(&source.Kind{Type: &cloudv1alpha1.Komet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resources and requeue the owner Komet
	err = c.Watch(&source.Kind{Type: &tekton.PipelineRun{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cloudv1alpha1.Komet{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &tekton.Pipeline{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cloudv1alpha1.Komet{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &tekton.PipelineResource{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cloudv1alpha1.Komet{},
	})
	if err != nil {
		return err
	}
	return nil
}

// blank assignment to verify that ReconcileKomet implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileKomet{}

// ReconcileKomet reconciles a Komet object
type ReconcileKomet struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Komet object and makes changes based on the state read
// and what is in the Komet.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileKomet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Komet")

	// Fetch the Komet instance
	instance := &cloudv1alpha1.Komet{}
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

	// Define a new Pod object
	//pod := newPodForCR(instance)
	git := newGitForCR(instance)
	pipe := newPipeForCR(instance)
	run := newRunForCR(instance)
	image := newImageForCR(instance)

	// Set Komet instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, git, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	if err := controllerutil.SetControllerReference(instance, pipe, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	if err := controllerutil.SetControllerReference(instance, run, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	if err := controllerutil.SetControllerReference(instance, image, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Git already exists
	foundGit := &tekton.PipelineResource{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: git.Name, Namespace: git.Namespace}, foundGit)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new git", "git.Namespace", git.Namespace, "git.Name", git.Name)
		err = r.client.Create(context.TODO(), git)
		if err != nil {
			return reconcile.Result{}, err
		}

		// created successfully - don't requeue
		// return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Image already exists
	foundImage := &tekton.PipelineResource{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: image.Name, Namespace: image.Namespace}, foundImage)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new image", "image.Namespace", image.Namespace, "image.Name", image.Name)
		err = r.client.Create(context.TODO(), image)
		if err != nil {
			return reconcile.Result{}, err
		}

		// created successfully - don't requeue
		// return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Wait couple seconds to make sure all PipelineResources are ready
	time.Sleep(3 * time.Second)

	// Check if this Pipeline already exists
	foundPipe := &tekton.Pipeline{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pipe.Name, Namespace: pipe.Namespace}, foundPipe)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pipeline", "Pipeline.Namespace", pipe.Namespace, "Pipeline.Name", pipe.Name)
		err = r.client.Create(context.TODO(), pipe)
		if err != nil {
			return reconcile.Result{}, err
		}

		// created successfully - don't requeue
		// return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Check if this PipelineRun already exists
	foundRun := &tekton.PipelineRun{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: run.Name, Namespace: run.Namespace}, foundRun)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new PipelineRun", "PipelineRun.Namespace", run.Namespace, "PipelineRun.Name", run.Name)
		err = r.client.Create(context.TODO(), run)
		if err != nil {
			return reconcile.Result{}, err
		}

		// created successfully - don't requeue
		// return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Everything already exists - don't requeue
	reqLogger.Info("Skip reconcile: All iteams already exist")
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *cloudv1alpha1.Komet) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
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

func newRunForCR(cr *cloudv1alpha1.Komet) *tekton.PipelineRun {

	labels := map[string]string{
		"app": cr.Name,
	}

	tasks := []tekton.PipelineTask{
		{
			Name: "build",
			TaskRef: tekton.TaskRef{
				Name: "kaniko",
				Kind: "Task",
			},
			Resources: &tekton.PipelineTaskResources{
				Inputs: []tekton.PipelineTaskInputResource{
					{
						Name:     "source",
						Resource: "source-repo",
					},
				//	{
				//		Name:     "image",
				//		Resource: "docker-image",
				//	},
				},
				Outputs: []tekton.PipelineTaskOutputResource{
					{
						Name:     "image",
						Resource: "docker-image",
					},
				},
			},
		},
	}

	res := []tekton.PipelineDeclaredResource{
		{
			Name: "source-repo",
			Type: tekton.PipelineResourceTypeGit,
		},
		{
			Name: "docker-image",
			Type: tekton.PipelineResourceTypeImage,
		},
	}

	p := tekton.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-run-" + cr.Spec.Version,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: tekton.PipelineSpec{
			Resources: res,
			Tasks:     tasks,
			//			Params: []ParamSpec{},

		},
	}

	ref := []tekton.PipelineResourceBinding{
		{
			Name: "source-repo",
			ResourceRef: tekton.PipelineResourceRef{
				Name: cr.Name+"-git-"+cr.Spec.Version,
			},
		},
		{
			Name: "docker-image",
			ResourceRef: tekton.PipelineResourceRef{
				Name: cr.Name+"-image-"+cr.Spec.Version,
			},
		},
	}
	return &tekton.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-run-" + cr.Spec.Version,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: tekton.PipelineRunSpec{
			PipelineRef: tekton.PipelineRef{
				Name: p.ObjectMeta.Name, //??
			},
			Resources: ref,
		},
	}
}

func newPipeForCR(cr *cloudv1alpha1.Komet) *tekton.Pipeline {

	labels := map[string]string{
		"app": cr.Name,
	}

	tasks := []tekton.PipelineTask{
		{
			Name: "build",
			TaskRef: tekton.TaskRef{
				Name: "kaniko",
				Kind: "Task",
			},
			Resources: &tekton.PipelineTaskResources{
				Inputs: []tekton.PipelineTaskInputResource{
					{
						Name:     "source",
						Resource: "source-repo",
					},
//					{
//						Name:     "image",
//						Resource: "docker-image",
//					},
				},
				Outputs: []tekton.PipelineTaskOutputResource{
					{
						Name:     "image",
						Resource: "docker-image",
					},
				},
			},
		},
		{
			Name: "deploy",
			TaskRef: tekton.TaskRef{
				Name: "knctl-deploy",
				Kind: "Task",
			},
			Resources: &tekton.PipelineTaskResources{
//				Inputs: []tekton.PipelineTaskInputResource{
//					{
//						Name:     "source",
//						Resource: "source-repo",
//					},
//				},
				Inputs: []tekton.PipelineTaskInputResource{
					{
						Name:     "image",
						Resource: "docker-image",
						From: []string{"build"},
					},
				},
			},
			Params: []tekton.Param{
				{
					Name:  "service",
					Value: strings.Split(cr.Name,"-")[0]+"-"+strings.Split(cr.Name,"-")[1],

				},
				{
					Name:  "namespace",
					Value: strings.Split(cr.Spec.Version, "-")[0],
				},
				{
					Name:  "env",
					Value: "SERVICES_DOMAIN="+strings.Split(cr.Name,"-")[2]+".svc.cluster.local",
				},

			},
//			RunAfter: []string{
//				"build",
//			},
		},
		{
			Name: "mailgun",
			TaskRef: tekton.TaskRef{
				Name: "mailgun",
				Kind: "Task",
			},
			Params: []tekton.Param{
				{
					Name:  "to",
					Value: cr.Spec.Author,

				},
				{
					Name:  "text",
					Value: "Komet released: http://" + strings.Split(cr.Name,"-")[0]+"-"+strings.Split(cr.Name,"-")[1]+"."+strings.Split(cr.Name,"-")[2]+".komet.plerion.io",
				},

			},
			RunAfter: []string{
				"deploy",
			},
		},
	}

	res := []tekton.PipelineDeclaredResource{
		{
			Name: "source-repo",
			Type: tekton.PipelineResourceTypeGit,
		},
		{
			Name: "docker-image",
			Type: tekton.PipelineResourceTypeImage,
		},
	}

	p := tekton.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-run-" + cr.Spec.Version,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: tekton.PipelineSpec{
			Resources: res,
			Tasks:     tasks,
			//			Params: []ParamSpec{},

		},
	}

	return &p
}

func newGitForCR(cr *cloudv1alpha1.Komet) *tekton.PipelineResource {

	labels := map[string]string{
		"app": cr.Name,
	}

	g := tekton.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-git-" + cr.Spec.Version,
//			Name:      "git",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: tekton.PipelineResourceSpec{
			Type: tekton.PipelineResourceTypeGit,
			Params: []tekton.Param{
				{
					Name:  "revision",
					Value: cr.Spec.GitBranch,
				},
				{
					Name:  "url",
					Value: cr.Spec.GitSource,
				},
			},
		},
	}

	return &g
}


func newImageForCR(cr *cloudv1alpha1.Komet) *tekton.PipelineResource {

	labels := map[string]string{
		"app": cr.Name,
	}

	i := tekton.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-image-" + cr.Spec.Version,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: tekton.PipelineResourceSpec{
			Type: tekton.PipelineResourceTypeImage,
			Params: []tekton.Param{
				{
					Name:  "url",
					Value: cr.Spec.DockerImage + ":" + cr.Spec.Version,
				},
			},
		},
	}

	return &i
}
