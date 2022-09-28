package logs

import (
	"bufio"
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"
)

const ERROR_LOG_FILE_SIZE_MAX = 1024 * 1024

type logFormatter func(entry *log.Entry) ([]byte, error)

func (f logFormatter) Format(entry *log.Entry) ([]byte, error) {
	return f(entry)
}

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetReportCaller(true)
	log.SetFormatter(&log.TextFormatter{
		CallerPrettyfier: func(frame *runtime.Frame) (function string, file string) {
			return frame.Function, ""
		},
	})
	log.SetFormatter(logFormatter(func(entry *log.Entry) ([]byte, error) {
		var b *bytes.Buffer
		if entry.Buffer != nil {
			b = entry.Buffer
		} else {
			b = &bytes.Buffer{}
		}

		b.WriteString(entry.Time.Format(time.RFC3339))
		b.WriteByte(' ')
		b.WriteString(strings.ToUpper(entry.Level.String()))

		b.WriteByte(' ')
		if caller, ok := entry.Data["caller"]; ok {
			b.WriteString(fmt.Sprintf("%s", caller))
		} else if entry.HasCaller() {
			b.WriteString(entry.Caller.Function)
		} else {
			b.WriteByte('-')
		}

		b.WriteByte(' ')
		b.WriteString(entry.Message)

		keys := make([]string, 0, len(entry.Data))
		for k := range entry.Data {
			if k == "caller" {
				continue
			}
			keys = append(keys, k)
		}

		sort.Strings(keys)

		for _, k := range keys {
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteByte('=')
			value := entry.Data[k]
			stringVal, ok := value.(string)
			if !ok {
				stringVal = fmt.Sprint(value)
			}
			b.WriteString(stringVal)
		}

		b.WriteByte('\n')
		return b.Bytes(), nil
	}))
}

var logFilePath string

func InitLoggerFile(filesDir string, fileName string) {
	logFile, err := os.Create(path.Join(filesDir, fileName))
	if err != nil {
		log.WithError(err).Error("🧨 file for logs doesn't created")
		return
	}
	logFilePath = logFile.Name()
	log.Infof("🐿 file for logs was created(%s)", logFilePath)
	log.SetOutput(io.MultiWriter(os.Stderr, bufio.NewWriter(logFile)))
}

func PreserveLogFile(filesDir string, fileName string) {
	if len(logFilePath) < 1 {
		log.Debug("Can't preserve log; log file path is not defined")
		return
	}
	errorLogPath := path.Join(filesDir, fileName)
	log.Infof("Preserve log file %s", errorLogPath)
	CopyFile(logFilePath, errorLogPath, ERROR_LOG_FILE_SIZE_MAX)
}

func GetLogFilePath() string {
	return logFilePath
}

type Logger interface {
	Trace(message string)
	Debug(message string)
	Info(message string)
	Warn(message string)
	Error(message string)
}

type LoggerStruct struct {
	logEntry *log.Entry
}

func (l *LoggerStruct) Trace(message string) {
	l.logEntry.Trace(message)
}

func (l *LoggerStruct) Debug(message string) {
	l.logEntry.Debug(message)
}

func (l *LoggerStruct) Info(message string) {
	l.logEntry.Info(message)
}

func (l *LoggerStruct) Warn(message string) {
	l.logEntry.Warn(message)
}

func (l *LoggerStruct) Error(message string) {
	l.logEntry.Error(message)
}

func New(caller string) Logger {
	return &LoggerStruct{
		logEntry: log.WithField("caller", caller),
	}
}

func CopyFile(src string, dst string, maxSize int64) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		log.WithError(err).Error("unable to copy: error opening src")
		return err
	}

	srcFileStat, err := sourceFile.Stat()
	if err != nil {
		log.WithError(err).Error("unable to copy: error stat src")
	}

	dstFile, err := os.Create(dst)
	if err != nil {
		log.WithError(err).Error("unable to copy: error opening dst")
		return err
	}

	if maxSize > 0 && srcFileStat.Size() > int64(maxSize) {
		seekTo := srcFileStat.Size() - maxSize
		_, err := sourceFile.Seek(seekTo, 0)
		if err != nil {
			log.WithError(err).Error(fmt.Sprintf("unable to copy: error seeking to %d", seekTo))
			return err
		}
	}

	buf := make([]byte, 1024*512)
	for {
		n, err := sourceFile.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}
		if _, err := dstFile.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}
