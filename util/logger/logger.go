package logger

import (
	"os"
)

type Logger struct {
	file           *os.File
	printToConsole bool
}

func NewLogger(file *os.File, printToConsole bool) *Logger {
	return &Logger{file: file, printToConsole: printToConsole}
}

func (logger *Logger) Write(p []byte) (n int, err error) {
	_, _ = logger.file.Write(p)
	if logger.printToConsole {
		_, _ = os.Stdout.Write(p)
	}
	return len(p), nil
}
