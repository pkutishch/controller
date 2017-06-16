package main

import (
	"fmt"
	"log"
	"os"

	"github.com/damemi/controller/pkg/controller"
	_ "github.com/openshift/origin/pkg/api/install"
	osclient "github.com/openshift/origin/pkg/client"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"

	"github.com/spf13/pflag"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
)

func main() {
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

	c := controller.NewController(openshiftClient, kubeClient)
	stopChan := make(chan struct{})
	c.Run(stopChan)
	<-stopChan

	fmt.Printf("Connected to OpenShift!\n")
}
