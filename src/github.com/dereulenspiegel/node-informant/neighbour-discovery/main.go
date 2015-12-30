package main

import (
	"flag"
	"log"
	"net"
	"time"

	"github.com/dereulenspiegel/node-informant/announced"
	"github.com/dereulenspiegel/node-informant/utils"
)

var (
	ifaceName     = flag.String("iface", "", "Network interface")
	queryString   = flag.String("query", "", "Query string")
	deflate       = flag.Bool("deflate", false, "Is the received data compressed")
	port          = flag.Int("port", 12444, "Port to listen to responses on")
	timeout       = flag.Int("timeout", -1, "Timeout after i seconds")
	targetAddress = flag.String("target", "", "Query a single device via unicast")

	requester *announced.Requester
)

func UseAnnounced() {
	localRequester, err := announced.NewRequester(*ifaceName, *port)
	requester = localRequester
	if err != nil {
		log.Printf("Error creating requester: %v", err)
		return
	}
	if *queryString == "" {
		log.Fatalf("No query string specified")
	}
	if *targetAddress != "" {
		addr := &net.UDPAddr{
			Port: 1001,
			IP:   net.ParseIP(*targetAddress),
		}
		requester.QueryUnicast(addr, *queryString)
	} else {
		requester.Query(*queryString)
	}
	for response := range requester.ReceiveChan {
		if *deflate {
			decompressedData, err := utils.Deflate(response.Payload)
			if err != nil {
				log.Printf("Error decompressing response data: %v", err)
			} else {
				log.Printf("Received response from %s: %s", response.ClientAddr.String(), string(decompressedData))
			}
		} else {
			log.Printf("Received response from %s: %s", response.ClientAddr.String(), string(response.Payload))
		}
	}
}

func main() {
	flag.Parse()
	if *timeout > 0 {
		go func() {
			time.Sleep(time.Second * time.Duration(*timeout))
			log.Printf("Closing requester")
			requester.Close()
		}()
	}
	UseAnnounced()
}
