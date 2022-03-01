package proxy

import (
	"fmt"
	"sync"

	"github.com/hashicorp/consul-k8s/cli/common"
	"k8s.io/client-go/kubernetes"
)

// Command is the `get proxy` command.
type Command struct {
	*common.BaseCommand

	kubernetes kubernetes.Interface

	once sync.Once
	help string
}

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

func (c *Command) init() {}
