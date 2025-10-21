package glang

import (
	"encoding/json"
	"errors"
	"evilchess/src/ui/gui/gbase/gconf"
	"evilchess/src/ui/gui/gbase/gos"
	"fmt"
)

type LangType int

const (
	EN LangType = iota
	RU
	ZZ
)

func LangTypeByString(lang string) LangType {
	switch lang {
	case "en":
		return EN
	case "ru":
		return RU
	default:
	}
	return ZZ
}

func (t LangType) String() string {
	switch t {
	case EN:
		return "en"
	case RU:
		return "ru"
	default:
	}
	return ""
}

type GUILangWorker struct {
	workdir string
	lang    LangType
	dict    map[string]string
}

// create object LangWorker and set EN lang
func NewGUILangWorker(workdir string, cfg *gconf.Config) (*GUILangWorker, error) {
	lw := &GUILangWorker{
		dict:    make(map[string]string),
		workdir: workdir,
	}
	t := LangTypeByString(cfg.Lang)
	if t == ZZ {
		return nil, errors.New("unsupported lang")
	}
	if err := lw.SetLang(t); err != nil {
		return nil, err
	}
	return lw, nil
}

func (lw *GUILangWorker) GetLang() LangType {
	return lw.lang
}

func (lw *GUILangWorker) SetLang(l LangType) error {
	lw.lang = l
	data, err := gos.ReadFile(lw.workdir + "/" + lw.langTypeToJsonName())
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &lw.dict)
}

func (lw *GUILangWorker) T(key string) string {
	if v, ok := lw.dict[key]; ok {
		return v
	}
	return fmt.Sprintf("%s", key) // if key is not found
}

func (lw *GUILangWorker) langTypeToJsonName() string {
	switch lw.lang {
	case EN:
		return "en.json"
	case RU:
		return "ru.json"
	default:
		return ""
	}
}
