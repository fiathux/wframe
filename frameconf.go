/* Program error code verson manager
 * config file support
 * Qujie Tech 2019-06-03
 * Fiathux Su
 */

package wframe

import (
	"gopkg.in/yaml.v2"
	"os"
)

// default config  file name
const defaultConfFile = "conf.yaml"

// InstConfig defined baisc configure object
type InstConfig struct {
	Listen      string                `yaml:"Listen,omitempty"`
	ServiceName string                `yaml:"ServiceName,omitempty"`
	LimitPost   uint                  `yaml:"LimitPost,omitempty"`
	MaxRedirect uint                  `yaml:"MaxRedirect,omitempty"`
	Includes    map[string]string     `yaml:"Includes,omitempty"`
	Logs        map[string]frmLogConf `yaml:"Logs,omitempty"`
	Debuging    bool                  `yaml:"DebugInterface,omitempty"`
}

////////////////////// functions //////////////////////

// create new framwork-config struct with default values
func newFrmConf() *InstConfig {
	return &InstConfig{
		"127.0.0.1:8080",
		"DefaultService",
		2097152,
		5,
		nil,
		nil,
		false,
	}
}

// load config object from config file
func loadConf(filepath string, confobj interface{}) error {
	fp, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer fp.Close()
	yamlDec := yaml.NewDecoder(fp)
	yamlDec.SetStrict(true)
	if err := yamlDec.Decode(confobj); err != nil {
		return err
	}
	return nil
}
