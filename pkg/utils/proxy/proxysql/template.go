package proxysql

type Config struct {
	AdminPort string
	SQLPort   string
	Backends  []Backend
}
type Backend struct {
	Host, Port, MaxConn string
}

const PerconaMysqlConfigTemplate = `
datadir="/var/lib/proxysql"

admin_variables=
{
	mysql_ifaces="0.0.0.0:{{.AdminPort}}"
	refresh_interval=2000
	web_enabled=false
	web_port=6080
	stats_credentials="stats:admin"
}

mysql_variables=
{
	threads=4
	max_connections=2048
	default_query_delay=0
	default_query_timeout=36000000
	have_compress=true
	poll_timeout=2000
	interfaces="0.0.0.0:{{.SQLPort}};/tmp/proxysql.sock"
	default_schema="information_schema"
	stacksize=1048576
	server_version="5.7.28"
	connect_timeout_server=10000
	monitor_history=60000
	monitor_connect_interval=200000
	monitor_ping_interval=200000
	ping_interval_server_msec=10000
	ping_timeout_server=200
	commands_stats=true
	sessions_sort=true
	monitor_username="$MONITOR_USERNAME"
	monitor_password="$MONITOR_PASSWORD"
	monitor_galera_healthcheck_interval=2000
	monitor_galera_healthcheck_timeout=800
}

mysql_galera_hostgroups =
(
	{
		writer_hostgroup=10
		backup_writer_hostgroup=20
		reader_hostgroup=30
		offline_hostgroup=9999
		max_writers=2
		writer_is_also_reader=2
		max_transactions_behind=30
		active=1
	}
)

mysql_servers =
(
  {{- range .Backends }}
	{ address="{{.Host}}", port={{.Port}}, hostgroup=10, max_connections={{.MaxConn}} },
  {{- end }}
)

mysql_query_rules =
(
	{
		rule_id=100
		active=1
		match_pattern="^SELECT .* FOR UPDATE"
		destination_hostgroup=10
		apply=1
	},
	{
		rule_id=200
		active=1
		match_pattern="^SELECT .*"
		destination_hostgroup=20
		apply=1
	},
	{
		rule_id=300
		active=1
		match_pattern=".*"
		destination_hostgroup=10
		apply=1
	}
)
`
