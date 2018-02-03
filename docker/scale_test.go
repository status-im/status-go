package scale

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"flag"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/ethereum/go-ethereum/whisper/shhclient"
	"github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/docker/project"
	"github.com/stretchr/testify/suite"
)

var keep = flag.Bool("keep", false, "keep the cluster after tests are finished.")

type Whisp struct {
	Name    string
	Rpc     string
	Metrics string
}

func MakeWhisps(containers []types.Container) []Whisp {
	whisps := []Whisp{}
	for _, container := range containers {
		w := Whisp{Name: container.Names[0]}
		for _, port := range container.Ports {
			if port.PrivatePort == 8080 {
				w.Metrics = fmt.Sprintf("http://%s:%d/metrics", port.IP, port.PublicPort)
			} else if port.PrivatePort == 8545 {
				w.Rpc = fmt.Sprintf("http://%s:%d", port.IP, port.PublicPort)
			}
		}
		whisps = append(whisps, w)
	}
	return whisps
}

func TestWhisperScale(t *testing.T) {
	suite.Run(t, new(WhisperScaleSuite))
}

type WhisperScaleSuite struct {
	suite.Suite

	p      project.Project
	whisps []Whisp
}

func (w *WhisperScaleSuite) SetupSuite() {
	flag.Parse()
}

func (s *WhisperScaleSuite) SetupTest() {
	cli, err := client.NewEnvClient()
	s.NoError(err)
	s.p = project.New("wnode-test-cluster", cli)
	s.NoError(s.p.Up(project.UpOpts{
		Scale: map[string]int{"wnode": 2},
		Wait:  true,
	}))
	containers, err := s.p.Containers(project.FilterOpts{SvcName: "wnode"})
	s.NoError(err)
	s.whisps = MakeWhisps(containers)
}

func (s *WhisperScaleSuite) TearDownTest() {
	if !*keep {
		s.NoError(s.p.Down()) // make it optional and wait
	}
}

func (s *WhisperScaleSuite) TestSymKeyMessaging() {
	msgNum := 10
	interval := 400 * time.Millisecond
	whispCount := 2
	var wg sync.WaitGroup
	if len(s.whisps) < whispCount {
		whispCount = len(s.whisps)
	}
	wg.Add(whispCount)
	for i := 0; i < whispCount; i++ {
		w := s.whisps[i]
		c, err := shhclient.Dial(w.Rpc)
		s.NoError(err)
		for {
			// wait till whisper is ready
			_, err := c.Info(context.TODO())
			if err != nil {
				time.Sleep(500 * time.Millisecond)
				continue
			}
			break
		}
		go func(c *shhclient.Client) {
			defer wg.Done()
			symkey, err := c.NewSymmetricKey(context.TODO())
			s.NoError(err)
			info, err := c.Info(context.TODO())
			s.NoError(err)
			for j := 0; j < msgNum; j++ {
				s.NoError(c.Post(context.TODO(), whisperv5.NewMessage{
					SymKeyID:  symkey,
					PowTarget: info.MinPow,
					PowTime:   200,
					Topic:     whisperv5.TopicType{0x03, 0x02, 0x02, 0x05},
					Payload:   []byte("hello"),
				}))
				time.Sleep(interval)
			}
		}(c)
	}
	wg.Wait()
}
