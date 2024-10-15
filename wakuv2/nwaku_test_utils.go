package wakuv2

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/miekg/dns"
)

type NwakuInfo struct {
	ListenAddresses []string `json:"listenAddresses"`
	EnrUri          string   `json:"enrUri"`
}

func GetNwakuInfo(host *string, port *int) (NwakuInfo, error) {
	nwakuRestPort := 8645
	if port != nil {
		nwakuRestPort = *port
	}
	envNwakuRestPort := os.Getenv("NWAKU_REST_PORT")
	if envNwakuRestPort != "" {
		v, err := strconv.Atoi(envNwakuRestPort)
		if err != nil {
			return NwakuInfo{}, err
		}
		nwakuRestPort = v
	}

	nwakuRestHost := "localhost"
	if host != nil {
		nwakuRestHost = *host
	}
	envNwakuRestHost := os.Getenv("NWAKU_REST_HOST")
	if envNwakuRestHost != "" {
		nwakuRestHost = envNwakuRestHost
	}

	resp, err := http.Get(fmt.Sprintf("http://%s:%d/debug/v1/info", nwakuRestHost, nwakuRestPort))
	if err != nil {
		return NwakuInfo{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NwakuInfo{}, err
	}

	var data NwakuInfo
	err = json.Unmarshal(body, &data)
	if err != nil {
		return NwakuInfo{}, err
	}

	return data, nil
}

func CreateFakeDnsServer(txtRecord string) (*dns.Server, error) {
    handleDNSRequest := func(w dns.ResponseWriter, r *dns.Msg) {
        fmt.Println("----------- received dns request -------")
		msg := dns.Msg{}
        msg.SetReply(r)
        msg.Authoritative = true

        for _, q := range r.Question {
            switch q.Qtype {
            case dns.TypeA:
                rr, err := dns.NewRR(fmt.Sprintf("%s A 127.0.0.1", q.Name))
                if err != nil {
                    log.Printf("Failed to create A RR: %v", err)
                    continue
                }
                msg.Answer = append(msg.Answer, rr)
            case dns.TypeTXT:
                rr, err := dns.NewRR(fmt.Sprintf("%s TXT \"%s\"", q.Name, txtRecord))
                if err != nil {
                    log.Printf("Failed to create TXT RR: %v", err)
                    continue
                }
                msg.Answer = append(msg.Answer, rr)
            }
        }

        err := w.WriteMsg(&msg)
        if err != nil {
            log.Printf("Failed to write message: %v", err)
        }
    }

    // Create a new DNS server mux
    dns.HandleFunc(".", handleDNSRequest)
    
    // Create the server
    server := &dns.Server{Addr: ":53", Net: "udp"}

    return server, nil
}

