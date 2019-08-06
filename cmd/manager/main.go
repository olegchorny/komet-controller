package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	"net/http"
	"gopkg.in/go-playground/webhooks.v5/github"
	cloudv1alpha1 "github.com/plerionio/komet-controller/pkg/apis/cloud/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/api/errors"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/plerionio/komet-controller/pkg/apis"
	"github.com/plerionio/komet-controller/pkg/controller"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/spf13/pflag"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)
const (
	path = "/webhooks"
)
// Change below variables to serve metrics on different host or port.
var (
	metricsHost       = "0.0.0.0"
	metricsPort int32 = 8383
)
var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("Version of operator-sdk: %v", sdkVersion.Version))
}

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "komet-controller-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := tekton.AddToScheme(mgr.GetScheme()); err != nil {
	      log.Error(err, "")
	      os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create Service object to expose the metrics port.
	_, err = metrics.ExposeMetricsPort(ctx, metricsPort)
	if err != nil {
		log.Info(err.Error())
	}



	log.Info("Starting the gitHook.")
	go gitHook(mgr)

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

func gitHook(mgr manager.Manager) {
	secret := getEnv("WEBHOOK_SECRET", "test")

//	hook, _ := github.New(github.Options.Secret("enimal"))
	hook, _ := github.New(github.Options.Secret(secret))

	http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		payload, err := hook.Parse(r, github.ReleaseEvent, github.PullRequestEvent, github.PushEvent)
		if err != nil {
			if err == github.ErrEventNotFound {
				// ok event wasn;t one of the ones asked to be parsed
			}
		}
		switch payload.(type) {

		case github.PushPayload:
			push := payload.(github.PushPayload)
			// Do whatever you want from here...
			komet := newKometForCR(push.Repository.Name, push.Repository.HTMLURL, push.After, push.Repository.FullName, push.Ref, push.Pusher.Email)

			log.Info("New Komet detected, processing", "GitHub URL:",komet.Spec.GitSource)
//			log.Info(komet.Spec.GitSource)

			foundKomet := &cloudv1alpha1.Komet{}
			client := mgr.GetClient()
			err := client.Get(context.TODO(), types.NamespacedName{"komet","komet"}, foundKomet)
			if err != nil && errors.IsNotFound(err) && push.After[0:7] != "0000000" {
			log.Info("Creating a new Komet", "komet.Namespace", komet.Namespace, "komet.Name", komet.Name)
//			komet.SetFinalizers(append(komet.GetFinalizers(), "new-finalizer"))
			err = client.Delete(context.TODO(), komet)
			err = client.Create(context.TODO(), komet)

			//err = client.Patch(context.TODO(), komet)
		if err != nil {
			log.Error(err, "No need  to create a new komet for given repo and commit")
		}
	// todo: create ns if needed
	//		err = client.Create(context.TODO(), ns)
		// created successfully - don't requeue
		// return reconcile.Result{}, nil
		} else if err != nil {
			//err = client.Update(context.TODO(), komet)
			//if err != nil {
			//	log.Error(err, "Failed to create komet")
			//}
			log.Info("No action required")
		}


			//log.Info("New Komet detected, GitHub URL:",push.Repository.Name)
			//fmt.Printf("%+v", push)
			w.Header().Set("Content-Type","application/json")
			w.Write([]byte(komet.Spec.GitSource))
			w.Write([]byte(push.Repository.Name))
//			w.Write([]byte(komet.Spec.GitSource))

		case github.ReleasePayload:
			release := payload.(github.ReleasePayload)
			// Do whatever you want from here...
			fmt.Printf("%+v", release)

		case github.PullRequestPayload:
			pullRequest := payload.(github.PullRequestPayload)
			// Do whatever you want from here...
			fmt.Printf("%+v", pullRequest)
		}
	})
	http.ListenAndServe(":3000", nil)
}

func newKometForCR(n string, g string, c string, f string, r string, a string)  *cloudv1alpha1.Komet {

//	labels := map[string]string{
//		"app": cr.Name,
//	}

b := strings.SplitAfter(r, "/")[2]

	k := cloudv1alpha1.Komet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      n+"-"+b,
			Namespace: "komet",
//			Labels:    labels,
		},
		Spec: cloudv1alpha1.KometSpec{
			GitSource: g,
			GitBranch: c,
			DockerImage: "ochorny/"+n,
//			DockerImage: "docker.pkg.github.com/"+f+"/"+n,
			Version: b+"-"+c[0:7],
			Author: a,
		},
	}

//	ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: b}}

	return &k //, &ns
}

func getEnv(key, fallback string) string {
    if value, ok := os.LookupEnv(key); ok {
        return value
    }
    return fallback
}
