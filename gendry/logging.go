package gendry

// LeveledLogger is a simple interface for logging messages at different "levels"
type LeveledLogger interface {
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Warnf(string, ...interface{})
	Errorf(string, ...interface{})
}
