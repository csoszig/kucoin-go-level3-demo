package helper

import (
	"log"
	"os"
)

var logger = CreateLogger()
var fi *os.File

func Info(format string, v ...interface{}) {
	logger.Printf("[Info] "+format+"\n", v...)
}

func Warn(format string, v ...interface{}) {
	logger.Printf("\033[33m[Warn] "+format+"\033[0m\n", v...)
}

func Error(format string, v ...interface{}) {
	logger.Printf("\033[31m[Error] "+format+"\033[0m\n", v...)
}

func Fatal(format string, v ...interface{}) {
	logger.Fatalf("\033[31m[Fatal] "+format+"\033[0m\n", v...)
}

func CreateLogger() *log.Logger {
	f, err := os.OpenFile("text.log",
	os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
	}
	fi = f
	// defer f.Close()
	return log.New(f, "", log.LstdFlags)
}

func CloseLogger() {
	fi.Close()
}
