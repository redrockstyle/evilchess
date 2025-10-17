package gconf

import (
	"encoding/json"
	"evilchess/ui/gui/gbase/gos"
	"fmt"
)

type Config struct {
	Theme     string `json:"theme"`           // light/dark
	Engine    string `json:"engine"`          // internal/external
	Lang      string `json:"language"`        // en/ru
	UCIPath   string `json:"uci_path"`        // path to external engine
	Strength  int    `json:"engine_strength"` // strength engine
	UseClock  bool   `json:"use_clock"`       // true/false
	UseEngine bool   `json:"use_engine"`      // true/false
	Clock     int    `json:"clock"`           // chess clock time
	PlayAs    string `json:"play_as"`         // white/random/black
	Training  bool   `json:"training_mode"`   // true/false
	WindowH   int    `json:"window_h"`        // window height
	WindowW   int    `json:"window_w"`        // window width
	Debug     bool   `json:"debug"`           // true/false
}

func defaultConfig() Config {
	return Config{
		Theme:     "light",
		Engine:    "internal",
		Lang:      "en",
		UCIPath:   "",
		Strength:  4,
		UseClock:  true,
		UseEngine: true,
		Clock:     3,
		PlayAs:    "random",
		Training:  false,
		WindowH:   800,
		WindowW:   1000,
		Debug:     false,
	}
}

func NewGUIConfig() (*Config, error) {
	file := "evilchess.json"

	_, err := gos.Stat(file)
	if gos.IsNotExist(err) {
		def := defaultConfig()
		return &def, nil
	} else if err != nil {
		return nil, err
	}

	conf, err := gos.Open(file)
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
	err = gos.WriteFile(file, jsonData, 0644)
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
	if c.Strength < 0 || c.Strength > 10 {
		c.Strength = def.Strength
	}
	if c.Clock < 0 || c.Clock > 60 {
		c.Clock = def.Clock
	}
	if c.PlayAs != "white" && c.PlayAs != "random" && c.PlayAs != "black" {
		c.PlayAs = def.PlayAs
	}
	if c.WindowH < def.WindowH || c.WindowW < def.WindowW {
		c.WindowH = def.WindowH
		c.WindowW = def.WindowW
	}
}
