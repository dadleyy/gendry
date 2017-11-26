package gendry

import "io"
import "os"
import "log"
import "fmt"
import "strings"
import "log/syslog"

// LeveledLogger is a simple interface for logging messages at different "levels"
type LeveledLogger interface {
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}

// LogLabel is used when creating the underlying log.Logger struct
type LogLabel string

// LogOutput is accepted as an argument to the NewLogger function to control which output to write to
type LogOutput io.Writer

var logOutput io.Writer

// NewLogger returns an implementation of a leveled logger.
func NewLogger(options ...interface{}) LeveledLogger {
	out := logOutput
	label := ""

	for _, o := range options {
		writer, ok := o.(LogOutput)

		if ok {
			out = io.Writer(writer)
			continue
		}

		if l, ok := o.(LogLabel); ok {
			label = fmt.Sprintf("%v ", strings.TrimSpace(string(l)))
		}
	}

	if out != nil {
		logger := log.New(out, label, 0)
		return &leveledLogger{logger}
	}

	net, addr, tag := os.Getenv("SYSLOG_NETWORK"), os.Getenv("SYSLOG_ADDRESS"), os.Getenv("SYSLOG_TAG")

	// If syslog is not configured fully, use stdout.
	if net == "" || addr == "" || tag == "" {
		logOutput = os.Stdout
		logger := log.New(logOutput, label, 0)
		return &leveledLogger{logger}
	}

	sys, err := syslog.Dial(net, addr, syslog.LOG_WARNING|syslog.LOG_DAEMON, tag)

	if err != nil {
		panic(err)
	}

	logOutput = sys
	logger := log.New(logOutput, label, 0)
	return &leveledLogger{logger}
}

type leveledLogger struct {
	*log.Logger
}

func (l *leveledLogger) Debugf(template string, elements ...interface{}) {
	l.Printf("[debug] %s", fmt.Sprintf(template, elements...))
}

func (l *leveledLogger) Infof(template string, elements ...interface{}) {
	l.Printf("[info] %s", fmt.Sprintf(template, elements...))
}

func (l *leveledLogger) Warnf(template string, elements ...interface{}) {
	l.Printf("[warn] %s", fmt.Sprintf(template, elements...))
}

func (l *leveledLogger) Errorf(template string, elements ...interface{}) {
	l.Printf("[error] %s", fmt.Sprintf(template, elements...))
}
