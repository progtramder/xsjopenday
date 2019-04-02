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

type chanRecover struct{}

func (self *chanRecover) handle() {
	ColorGreen("Start recovering ...")
	for _, v := range bmEventList.events {
		ColorGreen("Event: " + v.name)
		recoverEvent(v)
	}
	ColorGreen("Done.")
}

func recoverEvent(bmEvent *BMEvent) {
	bmEvent.Lock()
	defer bmEvent.Unlock()
	for i, sess := range bmEvent.sessions {
		ColorGreen("Session: " + sess.Desc)
		sheetName := trimSheetName(sess.Desc)
		rows := bmEvent.report.GetRows(sheetName)
		for _, row := range rows {
			// row data: openid, student, gender, phone, category
			bm := bminfo{}
			bm.session = i
			bm.student = row[1]
			bm.gender = row[2]
			bm.phone = row[3]
			bm.category = row[4]
			bmEvent.bm[row[0]] = bm
		}
		bmEvent.sessions[i].number += len(rows)
		ColorGreen(fmt.Sprintf("%d records recovered", len(rows)))
	}
}

func Recover() {
	dbChannel <- &chanRecover{}
}

var CmdLineHandler = map[string]CLIHandler{
	"quit":    CLIQuit(),
	"recover": CLIContinue(Recover),
}
