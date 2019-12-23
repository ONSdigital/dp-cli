package out

import (
	"fmt"

	"github.com/fatih/color"
)

func Info(msg string) {
	color.Green(fmt.Sprintf("[dp-cli] %s", msg))
}
func InfoAppend(msg string) {
	fmt.Print(color.GreenString(msg))
}

func InfoF(msg string, args ...interface{}) {
	msg = fmt.Sprintf(msg, args)
	color.Green(fmt.Sprintf("[dp-cli] %s", msg))
}

func Error(err error) {
	color.Red(fmt.Sprintf("[dp-cli] %s", err.Error()))
}
