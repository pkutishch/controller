package controller

import (
	"fmt"
	"time"

	osclient "github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
    _ "github.com/openshift/origin/pkg/quota/admission/runonceduration/api/install"
	"github.com/spf13/pflag"
	kapi "k8s.io/kubernetes/pkg/api"
	oapi "github.com/openshift/origin/pkg/build/api"
	poapi "github.com/openshift/origin/pkg/project/api"
	"k8s.io/kubernetes/pkg/api/meta"
	dep "github.com/openshift/origin/pkg/deploy/api"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/wait"
	"k8s.io/kubernetes/pkg/watch"
	"k8s.io/kubernetes/pkg/api/resource"
)

type Controller struct {
	openshiftClient *osclient.Client
	kubeClient      *kclient.Client
	mapper          meta.RESTMapper
	typer           runtime.ObjectTyper
	f               *clientcmd.Factory
}

func NewController(os *osclient.Client, kc *kclient.Client) *Controller {

	f := clientcmd.New(pflag.NewFlagSet("empty", pflag.ContinueOnError))
	mapper, typer := f.Object()

	return &Controller{
		openshiftClient: os,
		kubeClient:      kc,
		mapper:          mapper,
		typer:           typer,
		f:               f,
	}
}

func (c *Controller) Run(stopChan <-chan struct{}) {
	go wait.Until(func() {
		d, err := c.openshiftClient.DeploymentConfigs(kapi.NamespaceAll).Watch(kapi.ListOptions{})
		if err != nil {
			fmt.Println(err)
		}
		w, err := c.kubeClient.Pods(kapi.NamespaceAll).Watch(kapi.ListOptions{})
		if err != nil {
			fmt.Println(err)
		}
		p, err := c.openshiftClient.Builds(kapi.NamespaceAll).Watch(kapi.ListOptions{})
		if err != nil {
			fmt.Println(err)
		}
		q, err := c.openshiftClient.Projects().Watch(kapi.ListOptions{})
		if err != nil {
			fmt.Println(err)
		}
		res, err := c.kubeClient.ResourceQuotas(kapi.NamespaceAll).Watch(kapi.ListOptions{})
		if err != nil {
			fmt.Println(err)
		}
		if p == nil {
			return
		}
		if w == nil {
			return
		}
		if q == nil {
			return
		}
		for {
			select {
			case event, ok := <-w.ResultChan():
				c.ProcessEvent(event, ok)
		    case event, ok := <-p.ResultChan():
			    c.ProcessEvent(event,ok)
			case event, ok := <-q.ResultChan():
			    c.ProcessEvent(event,ok)
			case event, ok := <-res.ResultChan():
			    c.ProcessEvent(event,ok)
			case event, ok := <-d.ResultChan():
			    c.ProcessEvent(event,ok)
			}
		}
	}, 1*time.Millisecond, stopChan)
}
func (c *Controller) ProcessEvent(event watch.Event, ok bool) {
	if !ok {
		fmt.Println("Error received from watch channel")
	}
	if event.Type == watch.Error {
		fmt.Println("Watch channel error")
	}


	switch t := event.Object.(type) {
	case *kapi.Pod:
		_, err := c.kubeClient.Pods(t.ObjectMeta.Namespace).Get(t.ObjectMeta.Name)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("Pod %s has been %v with status %s message %s\n", t.ObjectMeta.Name, event.Type, t.Status.Phase, t.Status.Message )
	case *oapi.Build:
			fmt.Println(event.Type, t.ObjectMeta.Name, t.ObjectMeta.Namespace)		
	case *poapi.Project:
	    fmt.Println(event.Type, t.Name)
	case *kapi.ResourceQuota:
//		qu := t.Spec.Hard
//		bla := resource.NewQuantityFlagValue()
		data := make(map[kapi.ResourceName]resource.Quantity)
		data = t.Spec.Hard
		for i, v := range t.Spec.Hard {
			data[i] = v
		}
		//fmt.Println(data)
		res := resource.Quantity{}
		res = t.Spec.Hard["limits.memory"]
		ff := resource.NewQuantity(res.Value(), res.Format)
		fmt.Printf("%v: Project %s using now %v RAM\n",event.Type, t.ObjectMeta.Namespace, ff)
	case *dep.DeploymentConfig:
//	    kkk, _ := c.openshiftClient.DeploymentConfigs(t.ObjectMeta.Namespace).Get(t.ObjectMeta.Name)
		fmt.Printf("DeploymentConfig %v %s namespace %s\n", event.Type, t.Name,t.ObjectMeta.Namespace)
	default:
		fmt.Printf("Unknown type\n")
	}
}