/*
Copyright 2021 RadonDB.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sidecar

import (
	"fmt"
	"strconv"

	"github.com/blang/semver"
	"github.com/go-ini/ini"

	"github.com/radondb/radondb-mysql-kubernetes/utils"
)

// Config of the sidecar.
type Config struct {
	// The hostname of the pod.
	HostName string
	// The namespace where the pod is in.
	NameSpace string
	// The name of the headless service.
	ServiceName string

	// The password of the root user.
	RootPassword string

	// Username of new user to create.
	User string
	// Password for the new user.
	Password string
	// Name for new database to create.
	Database string

	// The name of replication user.
	ReplicationUser string
	// The password of the replication user.
	ReplicationPassword string

	// The name of metrics user.
	MetricsUser string
	// The password of metrics user.
	MetricsPassword string

	// The name of operator user.
	OperatorUser string
	// The password of operator user.
	OperatorPassword string

	// InitTokuDB represents if install tokudb engine.
	InitTokuDB bool

	// MySQLVersion represents the MySQL version that will be run.
	MySQLVersion semver.Version

	// The parameter in xenon means admit defeat count for hearbeat.
	AdmitDefeatHearbeatCount int32
	// The parameter in xenon means election timeout(ms)。
	ElectionTimeout int32
}

// NewConfig returns a pointer to Config.
func NewConfig() *Config {
	mysqlVersion, err := semver.Parse(getEnvValue("MYSQL_VERSION"))
	if err != nil {
		log.Info("MYSQL_VERSION is not a semver version")
		mysqlVersion, _ = semver.Parse(utils.MySQLDefaultVersion)
	}

	initTokuDB := false
	if len(getEnvValue("INIT_TOKUDB")) > 0 {
		initTokuDB = true
	}

	admitDefeatHearbeatCount, err := strconv.ParseInt(getEnvValue("ADMIT_DEFEAT_HEARBEAT_COUNT"), 10, 32)
	if err != nil {
		admitDefeatHearbeatCount = 5
	}
	electionTimeout, err := strconv.ParseInt(getEnvValue("ELECTION_TIMEOUT"), 10, 32)
	if err != nil {
		electionTimeout = 10000
	}

	return &Config{
		HostName:    getEnvValue("POD_HOSTNAME"),
		NameSpace:   getEnvValue("NAMESPACE"),
		ServiceName: getEnvValue("SERVICE_NAME"),

		RootPassword: getEnvValue("MYSQL_ROOT_PASSWORD"),

		Database: getEnvValue("MYSQL_DATABASE"),
		User:     getEnvValue("MYSQL_USER"),
		Password: getEnvValue("MYSQL_PASSWORD"),

		ReplicationUser:     getEnvValue("MYSQL_REPL_USER"),
		ReplicationPassword: getEnvValue("MYSQL_REPL_PASSWORD"),

		MetricsUser:     getEnvValue("METRICS_USER"),
		MetricsPassword: getEnvValue("METRICS_PASSWORD"),

		OperatorUser:     getEnvValue("OPERATOR_USER"),
		OperatorPassword: getEnvValue("OPERATOR_PASSWORD"),

		InitTokuDB: initTokuDB,

		MySQLVersion: mysqlVersion,

		AdmitDefeatHearbeatCount: int32(admitDefeatHearbeatCount),
		ElectionTimeout:          int32(electionTimeout),
	}
}

// buildExtraConfig build a ini file for mysql.
func (cfg *Config) buildExtraConfig(filePath string) (*ini.File, error) {
	conf := ini.Empty()
	sec := conf.Section("mysqld")

	id, err := generateServerID(cfg.HostName)
	if err != nil {
		return nil, err
	}
	if _, err := sec.NewKey("server-id", strconv.Itoa(id)); err != nil {
		return nil, err
	}

	if _, err := sec.NewKey("init-file", filePath); err != nil {
		return nil, err
	}

	return conf, nil
}

// buildXenonConf build a config file for xenon.
func (cfg *Config) buildXenonConf() []byte {
	pingTimeout := cfg.ElectionTimeout / cfg.AdmitDefeatHearbeatCount
	heartbeatTimeout := cfg.ElectionTimeout / cfg.AdmitDefeatHearbeatCount
	requestTimeout := cfg.ElectionTimeout / cfg.AdmitDefeatHearbeatCount

	version := "mysql80"
	if cfg.MySQLVersion.Major == 5 {
		if cfg.MySQLVersion.Minor == 6 {
			version = "mysql56"
		} else {
			version = "mysql57"
		}
	}

	var masterSysVars, slaveSysVars string
	if cfg.InitTokuDB {
		masterSysVars = "tokudb_fsync_log_period=default;sync_binlog=default;innodb_flush_log_at_trx_commit=default"
		slaveSysVars = "tokudb_fsync_log_period=1000;sync_binlog=1000;innodb_flush_log_at_trx_commit=1"
	} else {
		masterSysVars = "sync_binlog=default;innodb_flush_log_at_trx_commit=default"
		slaveSysVars = "sync_binlog=1000;innodb_flush_log_at_trx_commit=1"
	}

	hostName := fmt.Sprintf("%s.%s.%s", cfg.HostName, cfg.ServiceName, cfg.NameSpace)

	str := fmt.Sprintf(`{
    "log": {
        "level": "INFO"
    },
    "server": {
        "endpoint": "%s:%d",
        "peer-address": "%s:%d",
        "enable-apis": true
    },
    "replication": {
        "passwd": "%s",
        "user": "%s"
    },
    "rpc": {
        "request-timeout": %d
    },
    "mysql": {
        "admit-defeat-ping-count": 3,
        "admin": "root",
        "ping-timeout": %d,
        "passwd": "%s",
        "host": "localhost",
        "version": "%s",
        "master-sysvars": "%s",
        "slave-sysvars": "%s",
        "port": 3306,
        "monitor-disabled": true
    },
    "raft": {
        "election-timeout": %d,
        "admit-defeat-hearbeat-count": %d,
        "heartbeat-timeout": %d,
        "meta-datadir": "/var/lib/xenon/",
        "leader-start-command": "/scripts/leader-start.sh",
        "leader-stop-command": "/scripts/leader-stop.sh",
        "semi-sync-degrade": true,
        "purge-binlog-disabled": true,
        "super-idle": false
    }
}
`, hostName, utils.XenonPort, hostName, utils.XenonPeerPort, cfg.ReplicationPassword, cfg.ReplicationUser, requestTimeout,
		pingTimeout, cfg.RootPassword, version, masterSysVars, slaveSysVars, cfg.ElectionTimeout,
		cfg.AdmitDefeatHearbeatCount, heartbeatTimeout)
	return utils.StringToBytes(str)
}

// buildInitSql used to build init.sql. The file run after the mysql init.
func (cfg *Config) buildInitSql() []byte {
	sql := fmt.Sprintf(`SET @@SESSION.SQL_LOG_BIN=0;
CREATE DATABASE IF NOT EXISTS %s;
DELETE FROM mysql.user WHERE user in ('%s', '%s', '%s', '%s');
GRANT REPLICATION SLAVE, REPLICATION CLIENT ON *.* TO '%s'@'%%' IDENTIFIED BY '%s';
GRANT SELECT, PROCESS, REPLICATION CLIENT ON *.* TO '%s'@'%%' IDENTIFIED BY '%s';
GRANT SUPER, PROCESS, RELOAD, CREATE, SELECT ON *.* TO '%s'@'%%' IDENTIFIED BY '%s';
GRANT ALL ON %s.* TO '%s'@'%%' IDENTIFIED BY '%s';
FLUSH PRIVILEGES;
`, cfg.Database, cfg.ReplicationUser, cfg.MetricsUser, cfg.OperatorUser, cfg.User, cfg.ReplicationUser, cfg.ReplicationPassword,
		cfg.MetricsUser, cfg.MetricsPassword, cfg.OperatorUser, cfg.OperatorPassword, cfg.Database, cfg.User, cfg.Password)

	return utils.StringToBytes(sql)
}

// buildClientConfig used to build client.conf.
func (cfg *Config) buildClientConfig() (*ini.File, error) {
	conf := ini.Empty()
	sec := conf.Section("client")

	if _, err := sec.NewKey("host", "127.0.0.1"); err != nil {
		return nil, err
	}

	if _, err := sec.NewKey("port", fmt.Sprintf("%d", utils.MysqlPort)); err != nil {
		return nil, err
	}

	if _, err := sec.NewKey("user", cfg.OperatorUser); err != nil {
		return nil, err
	}

	if _, err := sec.NewKey("password", cfg.OperatorPassword); err != nil {
		return nil, err
	}

	return conf, nil
}
