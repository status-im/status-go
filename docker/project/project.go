package project

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Project struct {
	Path string
	Name string

	client *client.Client
}

type UpOpts struct {
	Scale map[string]int
	Wait  time.Duration
}

func New(basepath string, client *client.Client) Project {
	parts := strings.Split(basepath, "-")
	name := strings.Join(parts, "")
	return Project{
		Path:   filepath.Join(basepath, "docker-compose.yml"),
		Name:   strings.ToLower(name),
		client: client,
	}
}

func (p Project) Up(opts UpOpts) error {
	args := []string{"-f", p.Path, "up", "-d"}
	if len(opts.Scale) > 0 {
		args = append(args, "--scale")
		for service, value := range opts.Scale {
			args = append(args, fmt.Sprintf("%s=%d", service, value))
		}
	}
	cmd := exec.Command("docker-compose", args...)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return p.Wait(opts.Wait)

}

func (p Project) Down() error {
	return exec.Command("docker-compose", "-f", p.Path, "down").Run()
}

type FilterOpts struct {
	SvcName string
}

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

func (p Project) Wait(timeout time.Duration) error {
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
