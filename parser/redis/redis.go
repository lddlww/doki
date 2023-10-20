package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/doki/common/model"
	"github.com/doki/common/util"
	"github.com/go-redis/redis"
)

var _ model.Parser = (*Redis)(nil)

type Redis struct {
	proc *model.Process
	ins  *model.InsInfo
	pos  int64
}

func NewParser(v *model.Process) model.Parser {
	pos := time.Now().Add(-time.Minute * 2).Unix()
	return &Redis{
		proc: v,
		pos:  pos,
	}
}

func (r *Redis) GetLogFile() (string, error) {
	ins, err := util.GetRedisInsInfo(r.proc)
	if err != nil {
		return "", err
	}
	r.ins = ins

	return "", nil
}

func (r *Redis) GetLabels() map[string]string {
	labels := map[string]string{
		"job": r.proc.Name,
		//"appid":  r.ins.AppID,
		//"subapp": r.ins.SubApp,
		"uuid":   r.ins.UUID,
		"ip":     r.proc.Ip,
		"port":   fmt.Sprintf("%d", r.proc.Port),
		"type":   "slowlog",
		"domain": r.proc.Domain,
	}
	return labels
}

func (r *Redis) Process(ctx context.Context, lines <-chan string, sends chan<- model.Send, errCh chan<- error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", r.proc.Ip, r.proc.Port),
		Password: r.ins.Password,
	})
	err := client.Ping().Err()
	if err != nil {
		errCh <- err
		return
	}
	defer client.Close()

	r.parse(client, sends)
	ticker := time.NewTicker(time.Second * 30)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.parse(client, sends)
		}
	}
}

func (r *Redis) parse(client *redis.Client, sends chan<- model.Send) {
	//get all slowlog
	ret := client.Do("slowlog", "get")
	pos := time.Now().Unix()
	slowlogs := ret.Val().([]interface{})
	for _, v := range slowlogs {
		v := v.([]interface{})
		if len(v) != 6 {
			continue
		}
		pos := v[1].(int64)
		if r.pos > pos {
			continue
		}
		var data = make(map[string]interface{})
		data["id"] = v[0]
		data["ts"] = v[1]
		data["duration"] = v[2]
		data["query"] = v[3].([]interface{})
		data["clientAddr"] = v[4]
		ts := time.Unix(v[1].(int64), 0)
		sends <- model.Send{
			Timestamp: ts,
			Data:      data,
		}
	}
	r.pos = pos
}
