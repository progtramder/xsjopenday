package main

import "fmt"

const prompt = "Input <quit> to terminate server:"

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

func recoverSession(bmEvent *BMEvent, s int) int {
	sess := bmEvent.sessions[s]
	sheetName := trimSheetName(sess.Desc)
	rows := bmEvent.report.GetRows(sheetName)
	total := 0
	if len(rows) == 0 {
		return 0
	}

	keyRow := rows[0]
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		info := bminfo{session: s}
		openId := row[0]
		//somehow the column always greater than actual, why?
		//so we have to check if the column is empty
		for j := 1; j < len(row); j++ {
			key := keyRow[j]
			value := row[j]
			if key != "" {
				info.form = append(info.form, Pair{key, value})
			}
		}
		if _, ok := bmEvent.bm[openId]; !ok {
			bmEvent.bm[openId] = info
			total += 1
		}
	}
	return total
}

func recoverEvent(bmEvent *BMEvent) {
	bmEvent.Lock()
	defer bmEvent.Unlock()
	if bmEvent.started {
		ColorRed("Fail: the event is started")
		return
	}
	for i, sess := range bmEvent.sessions {
		ColorGreen("Session: " + sess.Desc)
		total := recoverSession(bmEvent, i)
		bmEvent.sessions[i].number += total
		ColorGreen(fmt.Sprintf("%d records recovered", total))
	}
}

func Recover() {
	ColorGreen("Start recovering ...")
	for _, v := range bmEventList.events {
		ColorGreen("Event: " + v.name)
		recoverEvent(v)
	}
	ColorGreen("Done.")
}

var CmdLineHandler = map[string]CLIHandler{
	"quit":    CLIQuit(),
	"recover": CLIContinue(Recover),
}
