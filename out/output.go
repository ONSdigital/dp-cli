package out

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ONSdigital/dp-cli/config"
	"github.com/fatih/color"
	"github.com/pkg/term"
)

var (
	infoBoldC    = color.New(color.Bold, color.FgHiBlue)
	infoC        = color.New(color.FgHiBlue)
	warningBoldC = color.New(color.Bold, color.FgHiYellow)
	warningC     = color.New(color.FgHiYellow)
	errorBoldC   = color.New(color.Bold, color.FgHiRed)
	errorC       = color.New(color.FgHiRed)
	outPrefix    = "[dp]"
)

type Level int

const (
	INFO Level = iota + 1
	WARN
	ERROR
)

func getColor(lvl Level) *color.Color {
	switch lvl {
	case ERROR:
		return errorC
	case WARN:
		return warningC
	default:
		return infoC
	}
}

func GetLevel(env config.Environment) Level {
	if env.Name == "production" {
		return ERROR
	}
	return INFO
}

func Write(lvl Level, msg string) {
	cliPrefix(getColor(lvl))
	fmt.Printf("%s\n", msg)
}

func WriteF(lvl Level, msg string, args ...interface{}) {
	cliPrefix(getColor(lvl))
	fmt.Printf(msg, args...)
}

func Highlight(lvl Level, msg string, args ...interface{}) {
	c := getColor(lvl)
	cliPrefix(c)
	highlight(c, os.Stdout, true, msg, args...)
}

func HighlightRaw(lvl Level, msg string, args ...interface{}) {
	c := getColor(lvl)
	cliPrefix(c)
	highlight(c, os.Stdout, false, msg, args...)
}

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
	highlight(infoC, os.Stdout, true, msg, args...)
}

func WarnFHighlight(msg string, args ...interface{}) {
	cliPrefix(warningBoldC)
	highlight(warningC, os.Stdout, true, msg, args...)
}

func ErrorFHighlight(msg string, args ...interface{}) {
	cliPrefix(errorBoldC)
	highlight(errorC, os.Stdout, true, msg, args...)
}

func highlight(c *color.Color, file io.Writer, newline bool, formattedMsg string, args ...interface{}) {
	var highlighted []interface{}

	for _, val := range args {
		highlighted = append(highlighted, c.SprintFunc()(val))
	}

	fmt.Fprintf(file, formattedMsg, highlighted...)
	if newline {
		fmt.Println("")
	}
}

func InfoE(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "INFO "+format+"\n", args...)
}

func WarnE(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "WARN "+format+"\n", args...)
}

func ErrorE(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR "+format+"\n", args...)
}

func FatalE(rc int, format string, args ...interface{}) {
	ErrorE(format, args...)
	os.Exit(rc)
}

func YesOrNo(msg string, args ...interface{}) (byte, error) {
	defaultKey := byte('y')
	otherKeys := "nq"

	for {
		HighlightRaw(INFO, msg, args...)
		fmt.Printf(
			" [%s%s] ",
			warningBoldC.SprintFunc()(string(defaultKey)),
			infoC.SprintFunc()(otherKeys),
		)
		readKey, err := getChar()
		if err != nil {
			return readKey, err
		}

		if readKey == '\r' {
			fmt.Println()
		} else {
			fmt.Println(string(readKey))
		}
		if readKey == '\r' || readKey == ' ' || readKey == defaultKey {
			return defaultKey, nil
		} else if strings.Contains(otherKeys, string(readKey)) {
			return readKey, nil
		}
	}
}

// returns a typed byte
func getChar() (b byte, err error) {
	var t *term.Term
	if t, err = term.Open("/dev/tty"); err != nil {
		return
	}
	defer t.Close()

	term.RawMode(t)
	defer t.Restore()

	bytes := make([]byte, 3)
	var readCount int
	if readCount, err = t.Read(bytes); err != nil {
		return
	}
	if readCount == 1 {
		b = bytes[0]
	} else {
		err = errors.New("too many chars read")
	}
	return
}
