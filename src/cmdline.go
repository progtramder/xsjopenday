package main

import (
	"encoding/json"
	"io/ioutil"
)

const prompt = ""

type CLIHandler interface {
	Handle() int
}

type CLIContinue func()

func (h CLIContinue) Handle() int {
	h()
	return Continue()
}

type CLIDoQuit func()

func (h CLIDoQuit) Handle() int {
	h()
	return Quit()
}

type quit struct{}

func (h quit) Handle() int {
	return Quit()
}
func CLIQuit() CLIHandler {
	return quit{}
}

func Quit() int {
	return 0
}

func Continue() int {
	return 1
}

type _loginfo struct {
	Token    string `json:"token"`
	Student  string `json:"student"`
	Gender   string `json:"gender"`
	Phone    string `json:"phone"`
	Category string `json:"category"`
	Session  int    `json:"session"`
}

func RecoverForMigration() {
	data := struct {
		Data []_loginfo `json:"data"`
	}{}

	bmEvent := bmEventList.GetEvent("主旨演讲报名")
	b, err := ioutil.ReadFile(systembasePath + "/log.json")
	if err != nil {
		ColorRed(err.Error())
		return
	}
	json.Unmarshal(b, &data)

	sessLen := len(bmEvent.sessions)
	sessions := make([]int, sessLen)
	bm := map[string]bminfo{}
	for _, v := range data.Data {
		if v.Session >= sessLen {
			ColorRed("fatal error: log data does not match current event")
			return
		}
		sessions[v.Session] += 1
		bm[v.Token] = bminfo{v.Student, v.Gender, v.Phone, v.Category, v.Session}
		register(bmEvent.name, v.Token, v.Student, v.Gender, v.Phone, v.Category, v.Session)
	}

	bmEvent.Lock()
	defer bmEvent.Unlock()
	for i := range sessions {
		bmEvent.sessions[i].number = sessions[i]
	}
	bmEvent.bm = bm
	ColorGreen("recovered from log.json")
}

var CmdLineHandler = map[string]CLIHandler{
	"quit":    CLIQuit(),
	"recover": CLIContinue(RecoverForMigration),
}
