package pairing

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/koron/go-ssdp"
	"go.uber.org/zap"

	"github.com/status-im/status-go/logutils"
	"github.com/status-im/status-go/server"
)

const st = "urn:service:localPairProvider"

var ssdpProviderOnce sync.Once
var sendAlivePeriod = time.Duration(3)
var maxAge = time.Duration(600)

// StartSsdpProvider starts SSDP server
// for the MVP, this should only be called on mobile since mobile network has less restriction
// to be simple, we start SSDP provider only once
func StartSsdpProvider(quit <-chan struct{}) {
	go startSsdpProvider(quit)
}

func startSsdpProvider(quit <-chan struct{}) {
	ssdpProviderOnce.Do(func() {
		logger := logutils.ZapLogger().Named("ssdp provider")
		logger.Info("Starting ssdp provider")

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte("Hello World!"))
			if err != nil {
				logger.Error("[ssdp provider] http server error when write", zap.Error(err))
			}
		})
		//TODO we can add a handler for /startTime to return the time when the server started
		// so client can know when the service expires

		// Create a listener on a random port.
		listener, err := net.Listen("tcp", "0.0.0.0:0")
		if err != nil {
			logger.Error("[ssdp provider] http server error when listen", zap.Error(err))
		}

		// Get the actual port used by the listener.
		tcpAddr := listener.Addr().(*net.TCPAddr)
		logger.Info("[ssdp provider] http server is listening", zap.Int("port", tcpAddr.Port))

		// Use the listener with http.Serve.
		go func() {
			if err := http.Serve(listener, nil); err != nil {
				logger.Error("[ssdp provider] http server error when serve", zap.Error(err))
			}
		}()

		ips, err := server.GetLocalAddressesForPairingServer()
		if err != nil {
			logger.Error("[ssdp provider] error when get local addresses", zap.Error(err))
		}
		logger.Info("[ssdp provider] local addresses", zap.Any("ips", ips))

		deviceName, err := server.GetDeviceName()
		if err != nil {
			logger.Error("[ssdp provider] error when get device name", zap.Error(err))
		}
		for _, ip := range ips {
			go func(ip net.IP) { // Make sure to capture the loop variable
				usn := uuid.New().String()
				location := fmt.Sprintf("http://%s:%d/", ip.String(), tcpAddr.Port)
				ad, err := ssdp.Advertise(
					st,
					usn,
					location,
					deviceName,
					int(maxAge))
				if err != nil {
					logger.Error("[ssdp provider] error when advertise", zap.Error(err))
					return
				}
				logger.Info("[ssdp provider] advertise success", zap.String("usn", usn), zap.String("location", location))

				// Start the timer for sending the Alive() signal periodically
				ticker := time.NewTicker(sendAlivePeriod * time.Second)
				// Set a timeout for when to send the Bye() signal and close the advertiser
				timeout := time.After(maxAge * time.Second)
				handleShutdown := func() {
					logger.Info("[ssdp provider] shutdown")
					ticker.Stop()
					if err := listener.Close(); err != nil {
						logger.Error("msg: [ssdp provider] error when close http server listener", zap.Error(err))
					}
					if err := ad.Bye(); err != nil {
						logger.Error("msg: [ssdp provider] error when send bye", zap.Error(err))
					}
					if err := ad.Close(); err != nil {
						logger.Error("msg: [ssdp provider] error when close", zap.Error(err))
					}
				}
				for {
					select {
					case <-ticker.C:
						// Send alive signal to refresh advertisement
						err = ad.Alive()
						if err != nil {
							logger.Error("[ssdp provider] error when send alive", zap.Error(err))
						}
					case <-timeout:
						handleShutdown()
						return
					case <-quit:
						handleShutdown()
						return
					}
				}
			}(ip)
		}

	})
}
