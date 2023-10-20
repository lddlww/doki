package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/doki/common/model"
	"github.com/go-ini/ini"
	"github.com/shirou/gopsutil/process"
)

//must exist ins_key file

func GetMongodbInsInfo(v *model.Process) (*model.InsInfo, error) {
	proc, err := process.NewProcess(int32(v.Pid))
	if err != nil {
		return nil, err
	}
	port := fmt.Sprintf("%d", v.Port)
	cmd, err := proc.Cmdline()
	if err != nil {
		return nil, err
	}
	file := strings.TrimSpace(strings.Split(cmd, "-f")[1])
	fd, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer fd.Close()
	var dbpath, logfile string
	row := bufio.NewScanner(fd)
	for row.Scan() {
		line := row.Text()
		if strings.HasPrefix(line, "dbpath") {
			dbpath = strings.TrimSpace(strings.Split(line, "=")[1])
		}
		if strings.HasPrefix(line, "logpath") {
			logfile = strings.TrimSpace(strings.Split(line, "=")[1])
		}
	}
	if dbpath == "" {
		return nil, fmt.Errorf("mongodb dbpath not found")
	}
	flag := Exist(logfile)
	if !flag {
		return nil, fmt.Errorf("%v file not  exist", logfile)
	}

	//keyFile := fmt.Sprintf("%s/%s", dbpath, "ins_key")
	ins, err := parseFile2(dbpath, string(port))
	if err != nil {
		return nil, err
	}
	if ins.LogFile != "" {
		return ins, nil
	}
	ins.LogFile = logfile
	return ins, nil
}

func GetMysqlInsInfo(v *model.Process) (*model.InsInfo, error) {
	proc, err := process.NewProcess(int32(v.Pid))
	if err != nil {
		return nil, err
	}
	port := fmt.Sprintf("%d", v.Port)
	cwd, err := proc.Cwd()
	if err != nil {
		return nil, err
	}
	//keyFile := fmt.Sprintf("%s/%s", cwd, "ins_key")
	ins, err := parseFile2(cwd, string(port))
	if err != nil {
		return nil, err
	}
	if ins.LogFile != "" {
		return ins, nil
	}
	logfile := fmt.Sprintf("%s/%s", cwd, "slow_query.txt")
	flag := Exist(logfile)
	if !flag {
		return nil, fmt.Errorf("%v file not  exist", logfile)
	}
	ins.LogFile = logfile

	return ins, nil
}

func GetRedisInsInfo(v *model.Process) (*model.InsInfo, error) {
	proc, err := process.NewProcess(int32(v.Pid))
	if err != nil {
		return nil, err
	}
	cwd, err := proc.Cwd()
	if err != nil {
		return nil, err
	}
	port := fmt.Sprintf("%d", v.Port)
	//keyFile := fmt.Sprintf("%s/%s", cwd, "ins_key")
	ins, err := parseFile(cwd, string(port))
	if err != nil {
		return nil, err
	}
	if ins.Password != "" {
		return ins, nil
	}

	password, err := GetKV(cwd+"/redis.conf", "requirepass")
	if err != nil {
		return nil, err
	}

	ins.Password = password

	return ins, nil
}

func parseFile2(file, port string) (*model.InsInfo, error) {
	return &model.InsInfo{}, nil
}

func parseFile(file, port string) (*model.InsInfo, error) {
	keyFile := fmt.Sprintf("%s/%s", file, "ins_key")
	cfg, err := ini.Load(keyFile)
	if err != nil {
		return nil, err
	}
	int_id := cfg.Section(port).Key("int_id").String()
	appid := cfg.Section(port).Key("appid").String()
	username := cfg.Section(port).Key("username").String()
	password := cfg.Section(port).Key("password").String()
	logfile := cfg.Section(port).Key("logfile").String()
	var subapp string
	app := strings.Split(appid, "-")
	if len(app) > 1 {
		appid = app[0]
		subapp = strings.Join(app[1:], "-")
	}
	ins := &model.InsInfo{
		UUID:     int_id,
		AppID:    appid,
		SubApp:   subapp,
		UserName: username,
		Password: password,
		LogFile:  logfile,
	}
	return ins, nil
}

func GetKV(file, prefix string) (string, error) {
	fd, err := os.Open(file)
	if err != nil {
		return "", err
	}
	defer fd.Close()
	br := bufio.NewReader(fd)
	for {
		line, _, err := br.ReadLine()
		if err == io.EOF {
			break
		}
		if !strings.HasPrefix(string(line), prefix) {
			continue
		}
		ret := strings.Split(string(line), " ")[1]
		ret = strings.Trim(ret, "\"")
		return ret, nil
	}
	return "", fmt.Errorf("not found key: %s", prefix)
}

func Exist(file string) bool {
	_, err := os.Stat(file)
	return err == nil || os.IsExist(err)
}
