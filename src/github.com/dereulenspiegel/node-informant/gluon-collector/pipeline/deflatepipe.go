package pipeline

import (
	log "github.com/Sirupsen/logrus"
	"github.com/dereulenspiegel/node-informant/announced"
	"github.com/dereulenspiegel/node-informant/utils"
)

type DeflatePipe struct {
}

func (d *DeflatePipe) Process(in chan announced.Response) chan announced.Response {
	out := make(chan announced.Response)
	go func() {
		for response := range in {
			decompressedData, err := utils.Deflate(response.Payload)
			if err != nil {
				log.WithFields(log.Fields{
					"error":   err,
					"client":  response.ClientAddr,
					"payload": response.Payload,
				}).Error("Error deflating response")
			} else {
				response.Payload = decompressedData
				out <- response
			}
		}

	}()
	return out
}
