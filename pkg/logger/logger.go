package logger

import (
  "fmt"
  "time"

  "github.com/uRepairPC/autodeploy/pkg/telegram"
)

// Info just notice
func Info(message string) {
  telegram.SendMe(message + "\n#info")
  fmt.Printf("[%s, info], %s\n", t(), message)
}

// Warning indicates a possible or impending danger, problem
func Warning(message string) {
  telegram.SendMe(message + "\n#warning")
  fmt.Printf("[%s, warning], %s\n", t(), message)
}

// Error is an action which is inaccurate or incorrect
func Error(err error) {
  telegram.SendMe(err.Error() + "\n#error")
  fmt.Printf("[%s, error] %s\n", t(), err)
}

// Panic is a critical error
func Panic(err error) {
  telegram.SendMe(err.Error() + "\n#panic")
  fmt.Printf("[%s, panic] %s\n", t(), err)
  panic(err)
}

func t() string {
  return time.Now().Format("01/02/06 15:04")
}
