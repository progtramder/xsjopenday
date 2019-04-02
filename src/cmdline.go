package main

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

var CmdLineHandler = map[string]CLIHandler{
	"quit":    CLIQuit(),
}
