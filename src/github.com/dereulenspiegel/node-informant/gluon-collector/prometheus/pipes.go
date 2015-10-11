package prometheus

import (
	log "github.com/Sirupsen/logrus"
	"github.com/dereulenspiegel/node-informant/gluon-collector/data"
	stat "github.com/prometheus/client_golang/prometheus"
)

type NodeCountPipe struct {
	Store data.Nodeinfostore
}

func (n *NodeCountPipe) Process(in chan data.ParsedResponse) chan data.ParsedResponse {
	out := make(chan data.ParsedResponse)
	go func() {
		for response := range in {
			_, err := n.Store.GetNodeStatusInfo(response.NodeId())
			if err != nil {
				TotalNodes.Inc()
			}
			out <- response
		}
	}()
	return out
}

type ClientCountPipe struct {
	Store data.Nodeinfostore
}

func (c *ClientCountPipe) Process(in chan data.ParsedResponse) chan data.ParsedResponse {
	out := make(chan data.ParsedResponse)
	go func() {
		for response := range in {
			if response.Type() == "statistics" {
				newStats, _ := response.ParsedData().(*data.StatisticsStruct)
				oldStats, err := c.Store.GetStatistics(response.NodeId())
				var addValue float64
				if err == nil {
					addValue = float64(newStats.Clients.Total - oldStats.Clients.Total)
				} else {
					addValue = float64(newStats.Clients.Total)
				}
				log.Debugf("Adding %f clients", addValue)
				TotalClientCounter.Add(addValue)
			}
			out <- response
		}
	}()
	return out
}

type TrafficCountPipe struct {
	Store data.Nodeinfostore
}

func collectTrafficBytes(counter stat.Counter, oldTraffic, newTraffic *data.TrafficObject) {
	var value float64
	if oldTraffic != nil {
		value = float64(newTraffic.Bytes - oldTraffic.Bytes)
	} else {
		value = float64(newTraffic.Bytes)
	}
	counter.Add(value)
}

func (t *TrafficCountPipe) Process(in chan data.ParsedResponse) chan data.ParsedResponse {
	out := make(chan data.ParsedResponse)
	go func() {
		for response := range in {
			if response.Type() == "statistics" {
				newStats, _ := response.ParsedData().(*data.StatisticsStruct)
				oldStats, _ := t.Store.GetStatistics(response.NodeId())

				if oldStats.Traffic == nil {
					oldStats.Traffic = &data.TrafficStruct{}
				}
				collectTrafficBytes(TotalNodeTrafficTx, oldStats.Traffic.Tx, newStats.Traffic.Tx)
				collectTrafficBytes(TotalNodeTrafficRx, oldStats.Traffic.Rx, newStats.Traffic.Rx)
				collectTrafficBytes(TotalNodeMgmtTrafficRx, oldStats.Traffic.MgmtRx, newStats.Traffic.MgmtRx)
				collectTrafficBytes(TotalNodeMgmtTrafficTx, oldStats.Traffic.MgmtTx, newStats.Traffic.MgmtTx)
			}
			out <- response
		}
	}()
	return out
}