## doki

database observability log component which  auto collect MySQL,Redis,MongoDB logs or slowlog and sends logs to grafana loki 

## run (--prefix is local ip prefix)
go build -o doki

./doki --prefix="172.27"



