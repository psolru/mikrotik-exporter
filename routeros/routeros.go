package routeros

import "gopkg.in/routeros.v2"

// Client - describes RouterOS command runner interface
type Client interface {
	Run(sentence ...string) (*routeros.Reply, error)
	Close()
	Async() <-chan error
}
