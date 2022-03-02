package proxyconfig

import (
	"fmt"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"github.com/hashicorp/consul-k8s/cli/common/flag"
	"k8s.io/client-go/kubernetes"
)

// Command is the proxyconfig command.
type Command struct {
	*common.BaseCommand

	set *flag.Sets

	flagPodName string
	Namespace   string
	fullConfig  bool
	format      string

	kubernetes kubernetes.Interface

	once sync.Once
	help string
}

// Run queries the Kubernetes Pod specified and returns its proxy configuration.
func (c *Command) Run(args []string) int {
	c.once.Do(c.init)

	fmt.Println("You ran the proxy command!")

	return 0
}

func (c *Command) Help() string {
	return c.help
}

func (c *Command) Synopsis() string {
	return ""
}

func (c *Command) init() {
	// Set up the flags.
	c.set = flag.NewSets()
	f := c.set.NewSet("Command Options")
	f.StringVar(&flag.StringVar{
		Name: "pod",
	})

}
