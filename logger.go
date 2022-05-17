package log

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/fatih/color"
)

type Logger struct {
	fileName string
	rollType string
	logLevel int

	ch          chan *LogMsg
	curFileName string
	buf         *bytes.Buffer
}

const (
	MSG_QUIT int = iota
	MSG_LOG
)

const (
	LV_DEBUG = iota
	LV_INFO
	LV_WARN
	LV_ERROR
	LV_FATAL
)

type LogMsg struct {
	msgType   int
	msg       string
	timestamp time.Time
}

func (l *Logger) init() {
	l.ch = make(chan *LogMsg, 100000)
	l.buf = bytes.NewBuffer([]byte{})
}

func (l *Logger) checkRolling(logTime time.Time) bool {
	if l.fileName == "console" {
		return false
	}
	now := ""
	if l.rollType == "DAY" {
		now = logTime.Format("2006-01-02")
	} else if l.rollType == "HOUR" {
		now = logTime.Format("2006-01-02T15")
	} else {
		return false
	}

	ext := path.Ext(l.fileName)

	pref := strings.TrimSuffix(l.fileName, ext)

	fn := pref + ext + "." + now

	if l.curFileName == "" {
		file, err := os.OpenFile(l.fileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0755)
		if err != nil {
			panic(fmt.Sprintf("open file %v failed %v", l.fileName, err))
		}
		defer file.Close()
		buf := bufio.NewReader(file)
		lineBytes, _, err := buf.ReadLine()
		if err != nil {
			return false
		}
		if len(lineBytes) == 0 {
			return false
		}
		line := string(lineBytes)
		strs := strings.Split(line, " ")
		if len(strs) <= 0 {
			return false
		}
		datetime := strs[0]
		hasPref := strings.HasPrefix(datetime, now)
		if !hasPref {
			closeErr := file.Close() //close first
			if closeErr != nil {
				fmt.Printf("close file %v failed: %v\n", l.fileName, closeErr)
			}
			//rename
			newFileName := pref + ext + "." + datetime[:len(now)]
			renameResult := os.Rename(l.fileName, newFileName)
			if renameResult != nil {
				fmt.Printf("initial rename file from %v to %v failed: %v\n",
					l.fileName, newFileName, renameResult)
			}
		}
		l.curFileName = fn
		file.Close()
	}
	if l.curFileName != fn {
		renameResult := os.Rename(l.fileName, l.curFileName)
		if renameResult != nil {
			fmt.Printf("it's time, but rename file from %v to %v failed: %v\n",
				l.fileName, l.curFileName, renameResult)
		}
		l.curFileName = fn
		return true
	}
	return false
}

func (l *Logger) getLogPref(curTime time.Time) string {
	now := curTime.Format("2006-01-02T15:04:05")
	nano := curTime.UnixNano()
	ms := int64(nano/(1000*1000)) % 1000
	pref := fmt.Sprintf("%s.%03d", now, ms)

	return pref
}

func (l *Logger) saveBufferToFile() error {
	if l.buf.Len() <= 0 {
		return nil
	}
	data := l.buf.Bytes()
	defer l.buf.Reset()

	file, err := os.OpenFile(l.fileName, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0755)
	if err != nil {
		fmt.Printf("open file %v failed %v\n", l.fileName, err)
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		fmt.Printf("write to file %v failed %v\n", l.fileName, err)
		return err
	}
	return nil
}

func (l *Logger) run() {
	buf := l.buf
	go func() {
		for {
			select {
			case msgData := <-l.ch:
				if msgData.msgType == MSG_QUIT {
					l.saveBufferToFile()
					os.Exit(1)
				}
				rolled := l.checkRolling(msgData.timestamp)
				if rolled {
					l.saveBufferToFile()
				}
				msg := msgData.msg
				buf.WriteString(msg)
				if buf.Len() >= 1024*1024 {
					l.saveBufferToFile()
				}
			default:
				l.saveBufferToFile()
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
}

func (l *Logger) logFormat(curTime time.Time, format string, a ...interface{}) string {
	pref := l.getLogPref(curTime)
	msg := pref + " " + fmt.Sprintf(format, a...) + "\n"
	return msg
}

func (l *Logger) info(format string, a ...interface{}) {
	if l.logLevel > LV_INFO {
		return
	}
	nowTime := time.Now()
	msg := l.logFormat(nowTime, "INFO "+format, a...)
	if l.fileName == "console" {
		color.New(color.FgGreen).Printf(msg)
		return
	}
	l.ch <- &LogMsg{
		msg:       msg,
		msgType:   MSG_LOG,
		timestamp: nowTime,
	}
}

func (l *Logger) warn(format string, a ...interface{}) {
	if l.logLevel > LV_WARN {
		return
	}
	nowTime := time.Now()
	msg := l.logFormat(nowTime, "WARN "+format, a...)
	if l.fileName == "console" {
		color.New(color.FgHiYellow).Printf(msg)
		return
	}
	l.ch <- &LogMsg{
		msg:       msg,
		msgType:   MSG_LOG,
		timestamp: time.Now(),
	}
}

func (l *Logger) error(format string, a ...interface{}) {
	if l.logLevel > LV_ERROR {
		return
	}
	nowTime := time.Now()
	msg := l.logFormat(nowTime, "ERROR "+format, a...)
	if l.fileName == "console" {
		color.New(color.FgHiRed).Printf(msg)
		return
	}
	l.ch <- &LogMsg{
		msg:       msg,
		msgType:   MSG_LOG,
		timestamp: time.Now(),
	}
}

func (l *Logger) fatal(format string, a ...interface{}) {
	if l.logLevel > LV_FATAL {
		return
	}
	nowTime := time.Now()
	msg := l.logFormat(nowTime, "FATAL "+format, a...)
	if l.fileName == "console" {
		color.New(color.FgHiRed).Printf(msg)
		l.quit()
		return
	}
	l.ch <- &LogMsg{
		msg:       msg,
		msgType:   MSG_LOG,
		timestamp: time.Now(),
	}
	l.quit()
}

func (l *Logger) debug(format string, a ...interface{}) {
	if l.logLevel > LV_DEBUG {
		return
	}
	nowTime := time.Now()
	msg := l.logFormat(nowTime, "DEBUG "+format, a...)
	if l.fileName == "console" {
		color.New(color.FgBlue).Printf(msg)
		return
	}
	l.ch <- &LogMsg{
		msg:       msg,
		msgType:   MSG_LOG,
		timestamp: time.Now(),
	}
}

func (l *Logger) quit() {
	if l.fileName == "console" {
		os.Exit(1)
	} else {
		l.ch <- &LogMsg{
			msgType:   MSG_QUIT,
			timestamp: time.Now(),
		}
	}
}

func (l *Logger) Log(msg string) {
	nowTime := time.Now()
	msg = l.logFormat(nowTime, "INFO "+msg)
	if l.fileName == "console" {
		fmt.Printf(msg)
		return
	}
	l.ch <- &LogMsg{
		msg:       msg,
		msgType:   MSG_LOG,
		timestamp: time.Now(),
	}
}
