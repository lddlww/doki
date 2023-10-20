package mongodb

import (
	"context"
	"fmt"
	"sync"

	"github.com/doki/common/model"
	"github.com/doki/common/util"
	"github.com/honeycombio/honeytail/event"
	"github.com/honeycombio/honeytail/parsers/mongodb"
)

var _ model.Parser = (*Mongodb)(nil)

type Mongodb struct {
	proc *model.Process
	ins  *model.InsInfo
}

func NewParser(proc *model.Process) model.Parser {
	return &Mongodb{
		proc: proc,
	}
}

func (m *Mongodb) GetLabels() map[string]string {
	labels := map[string]string{
		"job":  m.proc.Name,
		"port": fmt.Sprintf("%d", m.proc.Port),
		"ip":   m.proc.Ip,
		//"uuid":   m.ins.UUID,
		//"appid":  m.ins.AppID,
		//"subapp": m.ins.SubApp,
		"type":   "log",
		"domain": m.proc.Domain,
	}

	return labels
}
func (m *Mongodb) GetLogFile() (string, error) {
	ins, err := util.GetMongodbInsInfo(m.proc)
	if err != nil {
		return "", err
	}
	m.ins = ins

	return ins.LogFile, nil
}
func (m *Mongodb) Process(ctx context.Context, lines <-chan string, send chan<- model.Send, errCh chan<- error) {
	p := mongodb.Parser{}
	events := make(chan event.Event, 50)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		p.ProcessLines(lines, events, nil)
	}()

	for {
		select {
		case <-ctx.Done():
			wg.Wait()
			close(events)
			//get last event?
			return
		case event := <-events:
			send <- model.Send{
				Timestamp: event.Timestamp,
				Data:      event.Data,
			}
		}
	}
}
