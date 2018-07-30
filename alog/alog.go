package alog

import (
	"fmt"
	"sync"
	"time"
)

const (
	logLevelFatal = iota
	logLevelError
	logLevelWarning
	logLevelInfo
	logLevelDebug
)

var logLevelStr = []string{"Fatal", "Error", "warning", "Info", "Debug"}

type logMsg struct {
	ts    time.Time
	level int
	msg   string
	next  *logMsg
}

type alog struct {
	header *logMsg
	tail   *logMsg
	mutex  sync.Mutex
	cond   *sync.Cond
	wg     sync.WaitGroup
	num    int
	maxNum int
	level  int
	stop   bool
}

var Log = &alog{}

func init() {
	Log.cond = sync.NewCond(&Log.mutex)
	Log.level = logLevelDebug
	Log.maxNum = 10000
	go Log.worker()
}

func Stop() {
	Log.mutex.Lock()
	Log.stop = true
	Log.mutex.Unlock()

	Log.cond.Signal()
	Log.wg.Wait()
}

func SetLevel(level int) {
	Log.level = level
}

func SetMaxNum(maxNum int) {
	Log.maxNum = maxNum
}

func Fatal(format string, a ...interface{}) {
	Log.add(logLevelFatal, format, a...)
}

func Error(format string, a ...interface{}) {
	Log.add(logLevelError, format, a...)
}

func Warning(format string, a ...interface{}) {
	Log.add(logLevelWarning, format, a...)
}

func Info(format string, a ...interface{}) {
	Log.add(logLevelInfo, format, a...)
}

func Debug(format string, a ...interface{}) {
	Log.add(logLevelDebug, format, a...)
}

func (self *alog) worker() {
	self.wg.Add(1)
	for {
		self.mutex.Lock()
		for self.header == nil && self.stop == false {
			self.cond.Wait()
		}
		if self.header != nil {
			header := self.header
			self.header = nil
			self.tail = nil
			self.num = 0
			self.mutex.Unlock()
			self.process(header)
		} else {
			self.mutex.Unlock()
			break
		}
	}
	self.wg.Done()
}

func (self *alog) skip(level int) bool {
	if level > self.level {
		// skip log when level disabled
		return true
	}

	return false
}

func (self *alog) add(level int, format string, a ...interface{}) {
	if self.skip(level) {
		return
	}
	l := &logMsg{}
	l.ts = time.Now()
	l.level = level
	l.msg = fmt.Sprintf(format, a...)
	self.mutex.Lock()
	if self.num > self.maxNum && level > logLevelError {
		self.mutex.Unlock()
		return
	}
	if self.tail != nil {
		self.tail.next = l
	} else {
		self.header = l
	}
	self.tail = l
	self.num++
	self.mutex.Unlock()
	self.cond.Signal()
}

func (self *alog) process(header *logMsg) {
	for header != nil {
		if !self.skip(header.level) {
			fmt.Printf("[%s][%s]%s\n", header.ts.Format("2006-01-02 15:04:05.000"), logLevelStr[header.level], header.msg)
		}
		header = header.next
	}
}
