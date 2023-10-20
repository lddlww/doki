package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/doki/common/model"
	"github.com/doki/common/util"
	"github.com/doki/parser"
)

var (
	services = flag.String("services", "mysqld,mongod,redis-server", "collect service log")
	loki     = flag.String("loki", "localhost:3100", "loki server url")
	domain   = flag.String("domain", "dbaas", "domain for instances")
	ipprefix = flag.String("prefix", "", "local ip prefix")
)

func init() {
	fmt.Printf("[INFO]\tdoki start\n")
	flag.Parse()
}

func main() {
	defer fmt.Printf("[INFO]\tdoki finish\n")
	localServices := strings.Split(*services, ",")
	procs, err := util.GetLocalServices(localServices, *ipprefix)
	if err != nil {
		fmt.Printf("[ERROR]\t%v\n", err)
		os.Exit(1)
	}
	for _, v := range procs {
		v.Domain = *domain
	}
	fmt.Printf("[INFO]\tprocs is: %v\n", printProcs(procs))
	parser.Run(procs, *loki)
}

func printProcs(v []*model.Process) string {
	procs := ""
	for _, v := range v {
		procs = fmt.Sprintf("%v\t%v", procs, v)
	}
	return strings.Trim(procs, "\t")
}

// for furture
func usage() {
	fmt.Printf("Usage: %v [mysql=slowlog|redis=addr:pwd|mongodb=log]+\n", os.Args[0])
	os.Exit(1)
}

func argsParse() []model.Arg {
	p := flag.Args()
	if len(p) == 0 {
		usage()
	}
	var args []model.Arg
	for _, v := range p {
		kv := strings.Split(v, "=")
		switch kv[0] {
		case "mysql":
		case "mongodb":
		case "redis":
		default:
			fmt.Println("error: not match key ", kv[0])
			usage()
		}
		var arg = model.Arg{
			Service: kv[0],
			File:    kv[1],
		}
		args = append(args, arg)
	}
	return args
}
