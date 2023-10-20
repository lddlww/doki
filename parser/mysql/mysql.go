package mysql

import (
	"context"
	"fmt"
	"sync"

	"github.com/doki/common/model"
	"github.com/doki/common/util"
	"github.com/honeycombio/honeytail/event"
	"github.com/honeycombio/honeytail/parsers/mysql"
)

var _ model.Parser = (*Mysql)(nil)

type Mysql struct {
	ins  *model.InsInfo
	proc *model.Process
}

func NewParser(v *model.Process) model.Parser {
	return &Mysql{
		proc: v,
	}
}

func (m *Mysql) GetLogFile() (string, error) {
	ins, err := util.GetMysqlInsInfo(m.proc)
	if err != nil {
		return "", err
	}
	m.ins = ins
	return ins.LogFile, nil
}

func (m *Mysql) GetLabels() map[string]string {
	labels := map[string]string{
		//"appid":  m.ins.AppID,
		//"subapp": m.ins.SubApp,
		//"uuid":   m.ins.UUID,
		"job":    m.proc.Name,
		"ip":     m.proc.Ip,
		"port":   fmt.Sprintf("%d", m.proc.Port),
		"type":   "slowlog",
		"domain": m.proc.Domain,
	}
	return labels
}

func (m *Mysql) Process(ctx context.Context, lines <-chan string, sends chan<- model.Send, errCh chan<- error) {
	p := mysql.Parser{}
	events := make(chan event.Event, 50)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		wg.Done()
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
			fmt.Printf("[INFO]\tget mysqld event: %v\n", event)
			sends <- model.Send{
				Timestamp: event.Timestamp,
				Data:      event.Data,
			}
		}
	}
}
