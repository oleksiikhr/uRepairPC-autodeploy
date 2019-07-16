package logger

import (
	"fmt"
	"time"

	"github.com/urepairpc/autodeploy/pkg/telegram"
)

func Info(message string) {
	telegram.SendMe(message + "\n#info")
	fmt.Printf("[%s, info], %s\n", t(), message)
}

func Warning(message string) {
	telegram.SendMe(message + "\n#warning")
	fmt.Printf("[%s, warning], %s\n", t(), message)
}

func Error(err error) {
	telegram.SendMe(err.Error() + "\n#error")
	fmt.Printf("[%s, error] %s\n", t(), err)
}

func Panic(err error) {
	telegram.SendMe(err.Error() + "\n#panic")
	fmt.Printf("[%s, panic] %s\n", t(), err)
	panic(err)
}

func t() string {
	return time.Now().Format("01/02/06 15:04")
}
