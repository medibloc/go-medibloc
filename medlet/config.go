// Copyright (C) 2018  MediBloc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>

package medlet

import (
	"io/ioutil"
	"os"

	"github.com/gogo/protobuf/proto"
	medletpb "github.com/medibloc/go-medibloc/medlet/pb"
	log "github.com/medibloc/go-medibloc/util/logging"
	"github.com/sirupsen/logrus"
)

// LoadConfig loads configuration from the file.
func LoadConfig(file string) (*medletpb.Config, error) {
	if file == "" {
		return DefaultConfig(), nil
	}

	if !pathExist(file) {
		err := createDefaultConfigFile(file)
		if err != nil {
			return nil, err
		}
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Console().WithFields(logrus.Fields{
			"file": file,
			"err":  err,
		}).Error("Failed to read the config file.")
		return nil, err
	}
	content := string(b)

	pb := new(medletpb.Config)
	if err := proto.UnmarshalText(content, pb); err != nil {
		log.Console().WithFields(logrus.Fields{
			"content": content,
			"file":    file,
			"err":     err,
		}).Error("Failed to parse the config file.")
		return nil, err
	}
	return pb, nil
}

func createDefaultConfigFile(filename string) error {
	if err := ioutil.WriteFile(filename, []byte(defaultConfigString()), 0644); err != nil {
		log.Console().WithFields(logrus.Fields{
			"file": filename,
			"err":  err,
		}).Error("Failed to create the config file.")
		return err
	}
	return nil
}

// DefaultConfig returns default config.
func DefaultConfig() *medletpb.Config {
	return &medletpb.Config{
		Global: &medletpb.GlobalConfig{
			ChainId: 1,
			Datadir: "data.db",
		},
		Network: &medletpb.NetworkConfig{
			Listens:            []string{"/ip4/0.0.0.0/tcp/9900/"},
			NetworkKeyFile:     "",
			Seeds:              nil,
			BootstrapPeriod:    5,
			MinimumConnections: 5,
			CacheFile:          "net.cache",
			CachePeriod:        60 * 3,

			ConnMgrLowWaterMark:  900,
			ConnMgrHighWaterMark: 600,
			ConnMgrGracePeriod:   20,

			MaxReadConcurrency:  100,
			MaxWriteConcurrency: 100,
		},
		Chain: &medletpb.ChainConfig{
			Genesis:             "genesis.conf",
			StartMine:           false,
			SignatureCiphers:    nil,
			BlockCacheSize:      16384,
			TailCacheSize:       16384,
			BlockPoolSize:       16384,
			TransactionPoolSize: 65536,
			Proposers:           make([]*medletpb.ProposerConfig, 0),
		},
		Rpc: &medletpb.RPCConfig{
			RpcListen:        []string{"0.0.0.0:9920"},
			HttpListen:       []string{"0.0.0.0:9921"},
			HttpModule:       nil,
			ConnectionLimits: 0,
		},
		Stats: &medletpb.StatsConfig{
			EnableMetrics:   false,
			ReportingModule: nil,
			Influxdb: &medletpb.InfluxdbConfig{
				Host:     "",
				Port:     0,
				Db:       "",
				User:     "",
				Password: "",
			},
			MetricsTags: nil,
		},
		Misc: &medletpb.MiscConfig{
			DefaultKeystoreFileCiper: "",
		},
		App: &medletpb.AppConfig{
			LogLevel: "debug",
			LogFile:  "logs",
			LogAge:   0,
			Pprof: &medletpb.PprofConfig{
				HttpListen: "",
				Cpuprofile: "",
				Memprofile: "",
			},
			Version: "",
		},
		Sync: &medletpb.SyncConfig{
			ResponseTimeLimit:    2,
			NumberOfRetries:      5,
			ActiveDownloadLimit:  10,
			SyncActivationHeight: 64,
			SyncActivationLibGap: 64,
		},
	}
}

func defaultConfigString() string {
	return proto.MarshalTextString(DefaultConfig())
}

func pathExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
