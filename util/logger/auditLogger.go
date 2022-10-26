package logger

import (
	"ethattacksim/interfaces"
	"fmt"
	"os"
)

var auditLogger *AuditLogger

type AuditLogger struct {
	file           *os.File
	printToConsole bool
}

func InitAuditLogger(file *os.File, printToConsole bool) {
	auditLogger = &AuditLogger{file: file, printToConsole: printToConsole}
	if auditLogger != nil {
		// write csv headers
		_, _ = auditLogger.file.Write([]byte(fmt.Sprintf("%v ; %v ; %v ; %v ; %v ; %v\n", "time", "nodeId", "eventType", "from->to", "id", "text")))
	}
}

func (logger *AuditLogger) log(text string, now int64) {
	toPrint := []byte(fmt.Sprintf("%v ; %v\n", now, text))
	_, _ = logger.file.Write(toPrint)
	if logger.printToConsole {
		_, _ = os.Stdout.Write(toPrint)
	}
}

func Audit(nodeId string, t string, id string, text string, now int64) {
	if auditLogger != nil {
		auditLogger.log(fmt.Sprintf("%v ; %v ; ; %v ; %v", nodeId, t, id, text), now)
	}
}

func AuditEvent(nodeId string, t interfaces.IEventType, id string, text string, now int64) {
	if auditLogger != nil {
		auditLogger.log(fmt.Sprintf("%v ; %v ; ; %v ; %v", nodeId, t, id, text), now)
	}
}

func AuditEventSent(nodeId string, peerId string, t interfaces.IEventType, id string, text string, now int64) {
	if auditLogger != nil {
		auditLogger.log(fmt.Sprintf("%v ; %v ; %v->%v ; %v ; %v", nodeId, t, nodeId, peerId, id, text), now)
	}
}

func AuditEventReceived(nodeId string, peerId string, t interfaces.IEventType, id string, text string, now int64) {
	if auditLogger != nil {
		auditLogger.log(fmt.Sprintf("%v ; %v ; %v->%v ; %v ; %v", nodeId, t, peerId, nodeId, id, text), now)
	}
}
