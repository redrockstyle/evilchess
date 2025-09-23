package gconf

import (
	"encoding/json"
	"fmt"
	"os"
)

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

func NewGUIConfig() (*Config, error) {
	file := "evilchess.json"

	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		def := defaultConfig()
		return &def, nil
	} else if err != nil {
		return nil, err
	}

	conf, err := os.Open(file)
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

	return &c, nil
}

func (c *Config) Save() error {
	file := "evilchess.json"
	jsonData, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		return err
	}
	err = os.WriteFile(file, jsonData, 0644)
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
