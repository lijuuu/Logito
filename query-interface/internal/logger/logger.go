package logger

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

var (
	initColor    = color.New(color.FgCyan, color.Bold)
	queryColor   = color.New(color.FgGreen, color.Bold)
	dbColor      = color.New(color.FgBlue, color.Bold)
	esColor      = color.New(color.FgMagenta, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	successColor = color.New(color.FgGreen)
	infoColor    = color.New(color.FgYellow)
)

func Init(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	initColor.Printf("[%s] INIT: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

func Query(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	queryColor.Printf("[%s] QUERY: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

func Database(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	dbColor.Printf("[%s] DB: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

func Elasticsearch(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	esColor.Printf("[%s] ES: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

func Error(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	errorColor.Printf("[%s] ERROR: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

func Success(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	successColor.Printf("[%s] SUCCESS: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

func Info(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	infoColor.Printf("[%s] INFO: %s\n", timestamp, fmt.Sprintf(msg, args...))
}
