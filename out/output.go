package out

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	infoBoldC    = color.New(color.Bold, color.FgHiBlue)
	infoC        = color.New(color.FgHiBlue)
	warningBoldC = color.New(color.Bold, color.FgHiYellow)
	warningC     = color.New(color.FgHiYellow)
	errorBoldC   = color.New(color.Bold, color.FgHiRed)
	errorC       = color.New(color.FgHiRed)
	Red          = errorC
	outPrefix    = "[dp-cli]"
)

type Log func(msg string, args ...interface{})

func cliPrefix(c *color.Color) {
	c.Printf("%s ", outPrefix)
}

func Info(msg string) {
	cliPrefix(infoBoldC)
	fmt.Printf("%s\n", msg)
}

func Warn(msg string) {
	cliPrefix(warningBoldC)
	fmt.Printf("%s\n", msg)
}

func InfoAppend(msg string) {
	infoC.Print(msg)
}

func InfoF(msg string, args ...interface{}) {
	cliPrefix(infoBoldC)
	fmt.Printf(msg, args...)
}

func Error(err error) {
	cliPrefix(errorBoldC)
	fmt.Printf("%s\n", err.Error())
}

func InfoFHighlight(msg string, args ...interface{}) {
	cliPrefix(infoBoldC)
	highlight(infoC, msg, args...)
}

func WarnFHighlight(msg string, args ...interface{}) {
	cliPrefix(warningBoldC)
	highlight(warningC, msg, args...)
}

func ErrorFHighlight(msg string, args ...interface{}) {
	cliPrefix(errorBoldC)
	highlight(errorC, msg, args...)
}

func highlight(c *color.Color, formattedMsg string, args ...interface{}) {
	var highlighted []interface{}

	for _, val := range args {
		highlighted = append(highlighted, c.SprintFunc()(val))
	}

	formattedMsg = fmt.Sprintf(formattedMsg, highlighted...)
	fmt.Printf("%s\n", formattedMsg)
}
