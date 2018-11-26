package aLog

import (
	"fmt"
	"sync"
	"time"
)

// log level
const (
	logLevelFatal = iota
	logLevelError
	logLevelWarning
	logLevelInfo
	logLevelDebug
)

// log level names
var logLevelStr = []string{"Fatal", "Error", "Warning", "Info", "Debug"}

// one log msg
type logMsg struct {
	ts    time.Time
	level int
	msg   string
	next  *logMsg
}

// the log class
type aLog struct {
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

// the log instance
var aLogIns = &aLog{}

func init() {
	aLogIns.cond = sync.NewCond(&aLogIns.mutex)
	aLogIns.level = logLevelDebug
	aLogIns.maxNum = 10000
	go aLogIns.worker()
}

// stop log
func Stop() {
	aLogIns.mutex.Lock()
	aLogIns.stop = true
	aLogIns.mutex.Unlock()

	aLogIns.cond.Signal()
	aLogIns.wg.Wait()
}

// set log level, all logs big than level will not output
func SetLevel(level int) {
	aLogIns.level = level
}

// set max queued log number
func SetMaxNum(maxNum int) {
	aLogIns.maxNum = maxNum
}

func Fatal(format string, a ...interface{}) {
	aLogIns.add(logLevelFatal, format, a...)
}

func Error(format string, a ...interface{}) {
	aLogIns.add(logLevelError, format, a...)
}

func Warning(format string, a ...interface{}) {
	aLogIns.add(logLevelWarning, format, a...)
}

func Info(format string, a ...interface{}) {
	aLogIns.add(logLevelInfo, format, a...)
}

func Debug(format string, a ...interface{}) {
	aLogIns.add(logLevelDebug, format, a...)
}

func (self *aLog) worker() {
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

func (self *aLog) skip(level int) bool {
	if level > self.level {
		// skip log when level disabled
		return true
	}

	return false
}

func (self *aLog) add(level int, format string, a ...interface{}) {
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

func (self *aLog) process(header *logMsg) {
	for header != nil {
		if !self.skip(header.level) {
			fmt.Printf("[%s][%s]%s\n", header.ts.Format("2006-01-02 15:04:05.000"), logLevelStr[header.level], header.msg)
		}
		header = header.next
	}
}
