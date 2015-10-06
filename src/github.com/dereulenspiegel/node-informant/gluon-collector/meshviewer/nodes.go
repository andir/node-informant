package meshviewer

import (
	"encoding/json"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	conf "github.com/dereulenspiegel/node-informant/gluon-collector/config"
	"github.com/dereulenspiegel/node-informant/gluon-collector/data"
	"github.com/dereulenspiegel/node-informant/gluon-collector/httpserver"
)

const TimeFormat string = time.RFC3339

type NodeFlags struct {
	Gateway bool `json:"gateway"`
	Online  bool `json:"online"`
}
type NodesJsonNode struct {
	Nodeinfo   data.NodeInfo     `json:"nodeinfo"`
	Statistics *StatisticsStruct `json:"statistics"`
	Flags      NodeFlags         `json:"flags"`
	Lastseen   string            `json:"lastseen"`
	Firstseen  string            `json:"firstseen"`
}

type NodesJson struct {
	Timestamp string                   `json:"timestamp"`
	Version   int                      `json:"version"`
	Nodes     map[string]NodesJsonNode `json:"nodes"`
}

type NodesJsonGenerator struct {
	Store           data.Nodeinfostore
	CachedNodesJson string
}

func (n *NodesJsonGenerator) Routes() []httpserver.Route {
	var nodesRoutes = []httpserver.Route{
		httpserver.Route{"NodesJson", "GET", "/nodes.json", n.GetNodesJsonRest},
	}
	return nodesRoutes
}

func (n *NodesJsonGenerator) GetNodesJsonRest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(n.CachedNodesJson))
}

func convertToMeshviewerStatistics(in *data.StatisticsStruct) StatisticsStruct {
	return StatisticsStruct{
		Clients:     in.Clients.Total,
		Gateway:     in.Gateway,
		Loadavg:     in.LoadAverage,
		MemoryUsage: (float64(in.Memory.Free) / float64(in.Memory.Total)),
		RootfsUsage: in.RootFsUsage,
		Traffic:     &in.Traffic,
		Uptime:      in.Uptime,
	}
}

func isOnline(status *data.NodeStatusInfo) bool {
	now := time.Now()
	lastseen, err := time.Parse(TimeFormat, status.Lastseen)
	if err != nil {
		log.WithFields(log.Fields{
			"err":        err,
			"timeString": status.Lastseen,
		}).Error("Error while parsing lastseen time to determine online status")
	}
	var updateInterval int = 300
	if conf.Global != nil {
		updateInterval = conf.Global.UInt("announced.interval.statistics", 300)
	}
	if (now.Unix() - lastseen.Unix()) > int64((updateInterval * 3)) {
		status.Online = false
	} else {
		status.Online = true
	}
	return status.Online
}

func (n *NodesJsonGenerator) GetNodesJson() NodesJson {
	timestamp := time.Now().Format(TimeFormat)
	nodes := make(map[string]NodesJsonNode)
	for _, nodeInfo := range n.Store.GetNodeInfos() {
		nodeId := nodeInfo.NodeId
		var stats StatisticsStruct
		if storedStats, err := n.Store.GetStatistics(nodeId); err == nil {
			stats = convertToMeshviewerStatistics(&storedStats)
		} else {
			stats = StatisticsStruct{}
		}
		status, _ := n.Store.GetNodeStatusInfo(nodeId)
		flags := NodeFlags{
			Online:  isOnline(&status),
			Gateway: status.Gateway,
		}
		node := NodesJsonNode{
			Nodeinfo:   nodeInfo,
			Statistics: &stats,
			Lastseen:   status.Lastseen,
			Firstseen:  status.Firstseen,
			Flags:      flags,
		}
		nodes[nodeId] = node
	}
	nodesJson := NodesJson{
		Timestamp: timestamp,
		Version:   1,
		Nodes:     nodes,
	}
	return nodesJson
}

func (n *NodesJsonGenerator) UpdateNodesJson() {
	nodesJson := n.GetNodesJson()

	data, err := json.Marshal(&nodesJson)
	if err != nil {
		log.WithFields(log.Fields{
			"error":  err,
			"value":  err.(*json.UnsupportedValueError).Value,
			"string": err.(*json.UnsupportedValueError).Str,
		}).Errorf("Error encoding nodes.json")
		return
	}
	n.CachedNodesJson = string(data)
}