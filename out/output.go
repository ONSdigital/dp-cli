package out

import (
	"fmt"

	"github.com/fatih/color"
)

var (
	infoC          = color.New(color.Bold, color.FgHiBlue)
	warningC       = color.New(color.Bold, color.FgHiYellow)
	strongWarningC = color.New(color.Bold, color.FgHiRed)
	outPrefix      = "[dp-cli]"
)

func Info(msg string) {
	infoC.Printf("%s %s\n", outPrefix, msg)
}
func InfoAppend(msg string) {
	infoC.Print(msg)
}

func InfoF(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args...)
	infoC.Printf("%s %s", outPrefix, msg)
}

func Error(err error) {
	strongWarningC.Printf("%s %s", outPrefix, err.Error())
}

func InfoFHighlight(msg string, args ...interface{}) {
	highlight(infoC, msg, args...)
}

func WarnFHighlight(msg string, args ...interface{}) {
	highlight(warningC, msg, args...)
}

func ErrorFHighlight(msg string, args ...interface{}) {
	highlight(strongWarningC, msg, args...)
}

func highlight(c *color.Color, formattedMsg string, args ...interface{}) {
	var highlighted []interface{}
	highlightFunc := c.SprintFunc()

	for _, val := range args {
		highlighted = append(highlighted, highlightFunc(val))
	}

	formattedMsg = fmt.Sprintf(formattedMsg, highlighted...)
	fmt.Printf("%s %s\n", highlightFunc(outPrefix), formattedMsg)
}
