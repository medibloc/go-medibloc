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
func LoadConfig(file string) *medletpb.Config {
	if file == "" {
		return defaultConfig()
	}

	if !pathExist(file) {
		createDefaultConfigFile(file)
	}

	b, err := ioutil.ReadFile(file)
	if err != nil {
		log.Console().WithFields(logrus.Fields{
			"file": file,
			"err":  err,
		}).Fatal("Failed to read the config file.")
	}
	content := string(b)

	pb := new(medletpb.Config)
	if err := proto.UnmarshalText(content, pb); err != nil {
		log.Console().WithFields(logrus.Fields{
			"file": file,
			"err":  err,
		}).Fatal("Failed to parse the config file.")
	}
	return pb
}

func createDefaultConfigFile(filename string) {
	if err := ioutil.WriteFile(filename, []byte(defaultConfigString()), 0644); err != nil {
		log.Console().WithFields(logrus.Fields{
			"file": filename,
			"err":  err,
		}).Fatal("Failed to create the config file.")
	}
}

func defaultConfig() *medletpb.Config {
	return &medletpb.Config{
		Network: &medletpb.NetworkConfig{
			Seed:       nil,
			Listen:     []string{"127.0.0.1:9900", "127.0.0.1:9910"},
			PrivateKey: "",
			NetworkId:  0,
		},
		Chain: &medletpb.ChainConfig{
			ChainId:          1010,
			Genesis:          "",
			Datadir:          "",
			Keydir:           "",
			StartMine:        false,
			Coinbase:         "",
			Miner:            "",
			Passphrase:       "",
			SignatureCiphers: nil,
		},
		Rpc: &medletpb.RPCConfig{
			RpcListen:        []string{"127.0.0.1:9920"},
			HttpListen:       []string{"127.0.0.1:9921"},
			HttpModule:       nil,
			ConnectionLimits: 0,
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
	}
}

func defaultConfigString() string {
	return proto.MarshalTextString(defaultConfig())
}

func pathExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
