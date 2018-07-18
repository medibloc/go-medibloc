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
	"github.com/medibloc/go-medibloc/medlet/pb"
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
			"file": file,
			"err":  err,
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

//DefaultConfig returns default config.
func DefaultConfig() *medletpb.Config {
	return &medletpb.Config{
		Global: &medletpb.GlobalConfig{
			ChainId: 1,
			Datadir: "data.db",
		},
		Network: &medletpb.NetworkConfig{
			Seed:                       nil,
			Listen:                     []string{"127.0.0.1:9900", "127.0.0.1:9910"},
			PrivateKey:                 "",
			NetworkId:                  0,
			RouteTableSyncLoopInterval: 2000,
		},
		Chain: &medletpb.ChainConfig{
			Genesis:             "",
			Keydir:              "",
			StartMine:           false,
			Coinbase:            "",
			Miner:               "",
			Passphrase:          "",
			SignatureCiphers:    nil,
			BlockCacheSize:      128,
			TailCacheSize:       128,
			BlockPoolSize:       128,
			TransactionPoolSize: 262144,
			Privkey:             "",
		},
		Rpc: &medletpb.RPCConfig{
			RpcListen:        []string{"127.0.0.1:9920"},
			HttpListen:       []string{"127.0.0.1:9921"},
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
			SeedingMinChunkSize:        1,
			SeedingMaxChunkSize:        100,
			SeedingMaxConcurrentPeers:  5,
			DownloadChunkSize:          50,
			DownloadMaxConcurrentTasks: 5,
			DownloadChunkCacheSize:     10,
			MinimumPeers:               1,
			RequestInterval:            1,
			FinisherTimeout:            5,
			SyncActivationHeight:       100,
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
