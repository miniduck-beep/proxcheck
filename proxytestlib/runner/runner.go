package runner

import (
	"bytes"
	"fmt"
	"log"
	"os"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
)

type XrayRunner struct {
	instance   *core.Instance
	configFile string
}

func NewXrayRunner(configFile string) *XrayRunner {
	return &XrayRunner{
		configFile: configFile,
	}
}

func (r *XrayRunner) Start() error {
	configBytes, err := os.ReadFile(r.configFile)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	xrayConfig, err := serial.DecodeJSONConfig(bytes.NewReader(configBytes))
	if err != nil {
		return fmt.Errorf("error decoding config: %v", err)
	}

	coreConfig, err := xrayConfig.Build()
	if err != nil {
		return fmt.Errorf("error building config: %v", err)
	}

	instance, err := core.New(coreConfig)
	if err != nil {
		return fmt.Errorf("error creating Xray instance: %v", err)
	}

	if err := instance.Start(); err != nil {
		return fmt.Errorf("error starting Xray: %v", err)
	}

	r.instance = instance
	log.Println("Xray instance started successfully")

	return nil
}

func (r *XrayRunner) Stop() error {
	if r.instance != nil {
		err := r.instance.Close()
		r.instance = nil
		if err != nil {
			return fmt.Errorf("error stopping Xray: %v", err)
		}
		log.Println("Xray instance stopped successfully")
	}
	return nil
}

func (r *XrayRunner) IsRunning() bool {
	return r.instance != nil
}