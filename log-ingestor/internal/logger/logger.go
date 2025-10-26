package logger

import (
	"fmt"
	"time"

	"github.com/fatih/color"
)

var (
	initColor    = color.New(color.FgCyan, color.Bold)
	workerColor  = color.New(color.FgGreen, color.Bold)
	dbColor      = color.New(color.FgBlue, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	successColor = color.New(color.FgGreen)
	infoColor    = color.New(color.FgYellow)
)

func Init(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	initColor.Printf("[%s] INIT: %s\n", timestamp, fmt.Sprintf(msg, args...))
}

func Worker(workerNo int, batchSize int, msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	workerColor.Printf("[%s] WORKER-%d (batch:%d): %s\n", timestamp, workerNo, batchSize, fmt.Sprintf(msg, args...))
}

func Database(msg string, args ...interface{}) {
	timestamp := time.Now().Format("15:04:05")
	dbColor.Printf("[%s] DB: %s\n", timestamp, fmt.Sprintf(msg, args...))
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
