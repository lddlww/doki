package model

type Process struct {
	Domain string
	Name   string
	Port   int
	Ip     string
	Pid    int
}

type InsInfo struct {
	UUID     string
	AppID    string
	SubApp   string
	LogFile  string
	UserName string
	Password string
}
