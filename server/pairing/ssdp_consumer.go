package pairing

import (
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/koron/go-ssdp"
	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
)

var ssdpConsumerOnce sync.Once

func StartSsdpConsumer(quit <-chan struct{}) <-chan ssdp.Service {
	result := make(chan ssdp.Service, 1)
	ssdpConsumerOnce.Do(func() {
		go startSsdpConsumer(quit, result)
	})
	return result
}

func startSsdpConsumer(quit <-chan struct{}, result chan<- ssdp.Service) {
	logger := logutils.ZapLogger().Named("ssdp consumer")
	logger.Info("Starting ssdp consumer")

	var services []ssdp.Service
	var err error
	i := 1
	for {
		select {
		case <-quit:
			logger.Info("[ssdp consumer] quit")
			return
		default:
			logger.Info("[ssdp consumer] searching for services", zap.Int("searching times", i))
			services, err = ssdp.Search(st, 1, "")
			if err != nil {
				logger.Error("[ssdp consumer] error when search", zap.Error(err))
			}
			if len(services) > 0 {
				logger.Info("[ssdp consumer] found services", zap.Any("services", services))

				serviceMap := map[string]ssdp.Service{}
				for _, service := range services {
					serviceMap[service.USN] = service
				}

				c := http.Client{Timeout: time.Second}

				successCh := make(chan ssdp.Service, 1)
				for _, service := range serviceMap {
					go func(service ssdp.Service) {
						resp, err := c.Get(service.Location)
						if err != nil {
							logger.Error("[ssdp consumer] error when request server", zap.String("location", service.Location), zap.Error(err))
							return
						}
						defer resp.Body.Close()
						content, err := io.ReadAll(resp.Body)
						if err != nil {
							logger.Error("[ssdp consumer] error when read response body", zap.String("location", service.Location), zap.Error(err))
							return
						}
						logger.Info("[ssdp consumer] request server success", zap.String("location", service.Location), zap.String("content", string(content)))
						successCh <- service
					}(service)
				}

				timeout := time.After(maxAge * time.Second)
				select {
				case service := <-successCh:
					logger.Info("[ssdp consumer] found available service", zap.String("usn", service.USN), zap.String("location", service.Location))
					result <- service
					return
				case <-timeout:
					logger.Info("[ssdp consumer] timeout when searching available service")
				}
			}
		}

	}

}
