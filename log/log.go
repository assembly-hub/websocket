package log

import (
	"github.com/assembly-hub/log"
	"github.com/assembly-hub/log/empty"
)

var Log = empty.NoLog

func SetDefLog(log log.Log) {
	if log == nil {
		Log = empty.NoLog
	} else {
		Log = log
	}
}
