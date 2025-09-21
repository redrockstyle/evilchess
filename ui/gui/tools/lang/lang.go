package lang

import (
	"encoding/json"
	"fmt"
	"os"
)

type LangType int

const (
	EN LangType = iota
	RU
)

type GUILangWorker struct {
	workdir string
	lang    LangType
	dict    map[string]string
}

// create object LangWorker and set EN lang
func NewGUILangWorker(workdir string) (*GUILangWorker, error) {
	lw := &GUILangWorker{
		dict:    make(map[string]string),
		workdir: workdir,
	}
	if err := lw.SetLang(EN); err != nil {
		return nil, err
	}
	return lw, nil
}

func (lw *GUILangWorker) GetLang() LangType {
	return lw.lang
}

func (lw *GUILangWorker) SetLang(l LangType) error {
	lw.lang = l
	data, err := os.ReadFile(lw.workdir + "/" + lw.langTypeToJsonName())
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
