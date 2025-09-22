package gconf

import (
	"encoding/json"
	"fmt"
	"os"
)

type GUIConfigWorker struct {
	Path   string
	Config Config
}

type Config struct {
	Theme   string `json:"theme"`    // light/dark
	Engine  string `json:"engine"`   // internal/external
	Lang    string `json:"language"` // en/ru
	UCIPath string `json:"uci_path"` // path to external engine
	WindowH int    `json:"window_h"` //
	WindowW int    `json:"window_w"` //
	Debug   bool   `json:"debug"`    // true/false
}

func defaultConfig() Config {
	return Config{
		Theme:   "light",
		Engine:  "internal",
		Lang:    "en",
		UCIPath: "",
		WindowH: 800,
		WindowW: 1000,
		Debug:   false,
	}
}

func NewGUIConfigWorker(pathToConfig string) (*GUIConfigWorker, error) {
	confPath := "evilchess.json"

	_, err := os.Stat(confPath)
	if os.IsNotExist(err) {
		return &GUIConfigWorker{Path: confPath, Config: defaultConfig()}, nil
	} else if err != nil {
		return nil, err
	}

	conf, err := os.Open(confPath)
	if err != nil {
		return nil, err
	}
	defer conf.Close()

	dec := json.NewDecoder(conf)
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("error decode config: %s", err)
	}
	correctableConfig(&c)

	return &GUIConfigWorker{Path: confPath, Config: c}, nil
}

func (c *GUIConfigWorker) Save() error {
	jsonData, err := json.MarshalIndent(c.Config, "", "    ")
	if err != nil {
		return err
	}
	err = os.WriteFile(c.Path+"/evilchess.json", jsonData, 0644)
	if err != nil {
		return err
	}
	return nil
}

func correctableConfig(c *Config) {
	def := defaultConfig()
	if c.Theme == "" || (c.Theme != "light" && c.Theme != "dark") {
		c.Theme = def.Theme
	}
	if (c.Engine == "external" && c.UCIPath == "") ||
		(c.Engine == "" || (c.Engine != "internal" && c.Engine != "external")) {
		c.Engine = def.Engine
		c.UCIPath = ""
	}
	if c.Lang != "en" && c.Lang != "ru" {
		c.Lang = "en"
	}
	if c.WindowH < def.WindowH || c.WindowW < def.WindowW {
		c.WindowH = def.WindowH
		c.WindowW = def.WindowW
	}
}
