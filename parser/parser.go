package parser

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/afiskon/promtail-client/promtail"
	"github.com/doki/common/model"

	"github.com/doki/parser/mongodb"
	"github.com/doki/parser/mysql"
	"github.com/doki/parser/redis"
	"github.com/go-logfmt/logfmt"
	"github.com/honeycombio/honeytail/tail"
)

// for further to add parser, like proxysql,redis-sentinel,clickhouse and so on
func newParser(proc *model.Process) model.Parser {
	switch proc.Name {
	case "mysqld":
		return mysql.NewParser(proc)
	case "mongod":
		return mongodb.NewParser(proc)
	case "redis-server":
		return redis.NewParser(proc)
	default:
		return nil
	}
}
func Run(procs []*model.Process, loki string) error {
	ctx, cancel := context.WithCancel(context.Background())
	stopCh := make(chan struct{}, len(procs))
	for _, v := range procs {
		v := v
		go func() {
			parse(ctx, v, loki, stopCh)
		}()
	}
	sch := make(chan os.Signal, 1)
	signal.Notify(sch, syscall.SIGINT, syscall.SIGTERM)
	var count int
	for {
		select {
		case <-sch:
			cancel()
			fmt.Printf("[INFO]\tget signal, and gracefully exit\n")
		case <-stopCh:
			count++
			if count == len(procs) {
				fmt.Printf("[INFO]\tall parser had already exit, ready to exit now\n")
				cancel()
				return nil
			}
		}
	}
}

func parse(ctx context.Context, v *model.Process, loki string, stopCh chan struct{}) {
	defer func() {
		stopCh <- struct{}{}
	}()
	parser := newParser(v)
	if parser == nil {
		fmt.Printf("[ERROR]\t%v parser is not found\n", v.Name)
		return
	}
	file, err := parser.GetLogFile()
	if err != nil {
		fmt.Printf("[ERROR]\t%v\n", err)
		return
	}

	fmt.Printf("[INFO]\tstart parser %v,listen logfile: %v\n", v.Name, file)
	tailConfig := tail.Config{
		Paths: []string{file},
		Options: tail.TailOptions{
			ReadFrom:  "end",
			StateFile: fmt.Sprintf("%s_%d", file, v.Port),
		},
	}
	linesch, _ := tail.GetEntries(ctx, tailConfig)
	sends := make(chan model.Send, 100)
	var lines chan string
	if linesch != nil {
		lines = linesch[0]
	}
	errCh := make(chan error, 5)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		parser.Process(ctx, lines, sends, errCh)
	}()

	lokiURL := fmt.Sprintf("http://%s/api/prom/push", loki)
	l := buildLabels(parser.GetLabels())
	fmt.Printf("[INFO]\t%d labels is: %v\n", v.Port, l)
	lokiConfig := promtail.ClientConfig{
		PushURL:            lokiURL,
		Labels:             l,
		BatchWait:          time.Second * 2,
		BatchEntriesNumber: 50,
		SendLevel:          promtail.INFO,
		PrintLevel:         promtail.ERROR,
	}
	lokiClient, err := promtail.NewClientProto(lokiConfig)
	if err != nil {
		fmt.Printf("[ERROR]\t%v\n", err)
		return
	}
	defer lokiClient.Shutdown()

	//send to loki
	for {
		select {
		case <-ctx.Done():
			fmt.Printf("[INFO]\t%v parser exit\n", v.Name)
			wg.Wait()
			return
		case err := <-errCh:
			fmt.Printf("[ERROR]\t%v\n", err)
			close(lines)
			return
		case send := <-sends:
			send2Loki(lokiClient, send)
		}
	}
}

func buildLabels(data map[string]string) string {
	var labels []string
	for k, v := range data {
		labels = append(labels, fmt.Sprintf(`%s="%s"`, k, v))
	}
	l := fmt.Sprintf(`{%s}`, strings.Join(labels, ","))
	return l
}

func send2Loki(loki promtail.Client, send model.Send) error {
	data := send.Data

	w := &bytes.Buffer{}
	enc := logfmt.NewEncoder(w)
	for k, v := range data {
		_, ok := v.(map[string]interface{})
		if ok {
			bv, _ := json.Marshal(v)
			v = string(bv)
		}
		_, ok = v.([]interface{})
		if ok {
			bv, _ := json.Marshal(v)
			v = string(bv)
		}
		enc.EncodeKeyval(k, v)
	}
	enc.EncodeKeyval("time", send.Timestamp)
	fmt.Printf("[INFO]\tsend data: %v\n", w.String())
	loki.Infof(w.String())
	return nil
}
