package log

import (
	"fmt"
	"sync"
)

var instance *Logger
var once sync.Once

func InitLog(fileName, rollType string, logLevel int) error {
	once.Do(func() {
		if instance != nil {
			return
		}
		instance = &Logger{
			fileName: fileName,
			rollType: rollType,
			logLevel: logLevel,
		}
		instance.init()
		instance.run()
	})

	return nil
}

func getInstance() *Logger {
	if instance == nil {
		panic("you should initialize log before use")
	}
	return instance
}

func Debug(format string, a ...interface{}) {
	getInstance().debug(format, a...)
}

func Info(format string, a ...interface{}) {
	getInstance().info(format, a...)
}

func Warn(format string, a ...interface{}) {
	getInstance().warn(format, a...)
}

func Error(format string, a ...interface{}) {
	getInstance().error(format, a...)
}

func Fatal(format string, a ...interface{}) {
	getInstance().fatal(format, a...)
}

func StateInfo() string {
	return fmt.Sprintf("chanel len=%d, buf_len=%d", len(getInstance().ch), getInstance().buf.Len())
}

func Quit() {
	getInstance().quit()
}
