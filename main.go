package main

import (
	"os"
	_ "github.com/openshift/origin/pkg/api/install"
	oapi "github.com/openshift/origin/pkg/build/api"
	papi "github.com/openshift/origin/pkg/project/api"
	osclient "github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"k8s.io/kubernetes/pkg/api/meta"
	"k8s.io/kubernetes/pkg/runtime"
	"github.com/spf13/pflag"
	"k8s.io/kubernetes/pkg/api/resource"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	kapi "k8s.io/kubernetes/pkg/api"
	"log"
	"fmt"
	"k8s.io/kubernetes/pkg/watch"
	"strings"
)
type Controller struct {
	openshiftClient *osclient.Client
	kubeClient      *kclient.Client
	mapper          meta.RESTMapper
	typer           runtime.ObjectTyper
	f               *clientcmd.Factory
}

type Converter interface{}


func conf() *Controller {
		config, err := clientcmd.DefaultClientConfig(pflag.NewFlagSet("empty", pflag.ContinueOnError)).ClientConfig()
		kubeClient, err := kclient.New(config)
		if err != nil {
			log.Printf("Error creating cluster config: %s", err)
			os.Exit(1)
		}
		openshiftClient, err := osclient.New(config)
		if err != nil {
			log.Printf("Error creating OpenShift client: %s", err)
			os.Exit(2)
		}
	f := clientcmd.New(pflag.NewFlagSet("empty", pflag.ContinueOnError))
	mapper, typer := f.Object()

	return &Controller{
		openshiftClient: openshiftClient,
		kubeClient:      kubeClient,
		mapper:          mapper,
		typer:           typer,
		f:               f,
	}

}

func MonPods() <- chan watch.Event {
	c := conf()
	po, err := c.kubeClient.Pods(kapi.NamespaceAll).Watch(kapi.ListOptions{})
	if err != nil {
		log.Println("Error watching pods")
	}
	_, ok := <- po.ResultChan()
	if !ok {
		log.Println("Error watching quota channel")
	}
	return po.ResultChan()
}

func MonQuota() <- chan watch.Event {
	c := conf()
	qo, err := c.kubeClient.ResourceQuotas(kapi.NamespaceAll).Watch(kapi.ListOptions{})
	if err != nil {
		log.Println("Error watching quota")
	}
	_, ok := <- qo.ResultChan()
	if !ok {
		log.Println("Error watching quota channel")
	}
	return qo.ResultChan()
}

func MonProjects() <- chan watch.Event {
	c := conf()
	pr, err := c.openshiftClient.Projects().Watch(kapi.ListOptions{})
	if err != nil {
		log.Println(err)
	}
	return pr.ResultChan()
}

func MonBuilds() <- chan watch.Event {
	//event := watch.Event{}
	c := conf()
	bu, err := c.openshiftClient.Builds(kapi.NamespaceAll).Watch(kapi.ListOptions{})
	if err != nil {
		log.Println(err)
	}

	return bu.ResultChan()

}

func Filter(filt ...interface{}) Converter {
	var sl Converter

	sl = filt
	return sl
}

func main() {
	bu := MonBuilds()
	pr := MonProjects()
	qo := MonQuota()
	po := MonPods()
	var Fl []Converter
	for {
		select {
		case build := <-bu:
			buildIface := build.Object
			buildPoint := buildIface.(*oapi.Build)
			if buildPoint.Status.Phase == oapi.BuildPhaseFailed {
				fmt.Println(buildPoint.Name, buildPoint.ObjectMeta.Namespace, buildPoint.Status.Phase, buildPoint.Status.Reason)
			} else if buildPoint.Status.Phase == oapi.BuildPhaseComplete {
				fmt.Println(buildPoint.Name, buildPoint.ObjectMeta.Namespace, buildPoint.Status.Phase, buildPoint.Status.Reason)
			}
		case proj := <-pr:
			projIface := proj.Object
			projPoint := projIface.(*papi.Project)
			if proj.Type == watch.Added {
				fmt.Println(projPoint.Name, projPoint.Status.Phase)
			}
		case quota := <-qo:
			quotaIface := quota.Object
			quotaPoint := quotaIface.(*kapi.ResourceQuota)
			res := resource.Quantity{}
			res = quotaPoint.Spec.Hard["limits.memory"]
			rram := resource.NewQuantity(res.Value(), res.Format)
			res = quotaPoint.Spec.Hard["limits.cpu"]
			cpu := resource.NewQuantity(res.Value(), res.Format)
			res = quotaPoint.Spec.Hard["pods"]
			po := resource.NewQuantity(res.Value(), res.Format)
			if quota.Type == watch.Modified {
				//fmt.Printf("Project %s has been modified limits: CPU: %v, RAM: %v, PODS: %v\n", quotaPoint.ObjectMeta.Namespace, cpu, rram, po)
				f := Filter(quotaPoint.ObjectMeta.Namespace, cpu, rram, po)
				Fl = append(Fl, f)
				if len(Fl) > 1 {
					fmt.Println(Fl[len(Fl) - 1])
					str := fmt.Sprintf("%v", Fl[len(Fl) - 1])
					data := strings.TrimLeft(str,"[")
					data = strings.TrimRight(data,"]")
					out := strings.Split(data, " ")
					fmt.Printf("Project %s has been modified limits: CPU: %v, RAM: %v, PODS: %v\n", out[0], out[1], out[2], out[3])

					Fl = Fl[:0]
				}
			}
		case pods := <- po:
			podsIface := pods.Object
			podsPoint := podsIface.(*kapi.Pod)
			if podsPoint.Status.Phase == kapi.PodFailed {
				fmt.Printf("Pod %s in project %s failed with reason %v\n", podsPoint.ObjectMeta.Name, podsPoint.ObjectMeta.Namespace, podsPoint.Status.Reason)
			}


		}
	}
}