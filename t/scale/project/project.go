/*
Package project provides light-weight wrapper around docker-compose and docker client.

Main purpose of the package is to bootstrap a docker-compose cluster with parametrized parameters,
wait till containers in cluster are ready, get containers ip addresses and tear down a cluster.
*/
package project

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Project is a wrapper around docker-compose project.
type Project struct {
	Path string
	Name string

	client *client.Client
}

// UpOpts used to provide options for docker-compose up.
type UpOpts struct {
	Scale map[string]int
	Wait  time.Duration
}

// New initializes a Project.
func New(fullpath, name string, client *client.Client) Project {
	return Project{
		Path:   fullpath,
		Name:   strings.ToLower(name),
		client: client,
	}
}

// Up runs docker-compose up with options and waits till containers are running.
func (p Project) Up(opts UpOpts) error {
	args := []string{"-f", p.Path, "up", "-d"}
	if len(opts.Scale) > 0 {
		args = append(args, "--scale")
		for service, value := range opts.Scale {
			args = append(args, fmt.Sprintf("%s=%d", service, value))
		}
	}
	cmd := exec.Command("docker-compose", args...) // nolint (gas)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.New(string(out))
	}
	return p.wait(opts.Wait)

}

// Down runs docker-compose down.
func (p Project) Down() error {
	out, err := exec.Command("docker-compose", "-f", p.Path, "down").CombinedOutput() // nolint (gas)
	if err != nil {
		return errors.New(string(out))
	}
	return nil
}

// FilterOpts used to parametrize a query for a list of containers.
type FilterOpts struct {
	SvcName string
}

// Containers queries docker for containers and filters results according to FiltersOpts.
func (p Project) Containers(f FilterOpts) (rst []types.Container, err error) {
	containers, err := p.client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return
	}
	name := p.Name
	if len(f.SvcName) > 0 {
		name = strings.Join([]string{p.Name, f.SvcName}, "_")
	}
	for _, container := range containers {
		for _, cname := range container.Names {
			if strings.Contains(cname, name) {
				rst = append(rst, container)
			}
		}
	}
	return rst, err
}

func (p Project) wait(timeout time.Duration) error {
	timer := time.After(timeout)
	for {
		containers, err := p.Containers(FilterOpts{})
		if err != nil {
			return err
		}
		for _, c := range containers {
			if c.State != "running" {
				break
			}
			return nil
		}
		time.Sleep(300 * time.Millisecond)
		select {
		case <-timer:
			return errors.New("docker compose timeout")
		default:
		}
	}
}
