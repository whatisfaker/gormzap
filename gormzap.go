package gormzap

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"regexp"
	"time"
	"unicode"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	sqlRegexp                = regexp.MustCompile(`\?`)
	numericPlaceHolderRegexp = regexp.MustCompile(`\$\d+`)
)

type OutputLevel int8

const (
	OutputLevelDebug OutputLevel = iota - 1
	OutputLevelInfo
)

var (
	DefaultOutputLevel OutputLevel = OutputLevelDebug
)

func isPrintable(s string) bool {
	for _, r := range s {
		if !unicode.IsPrint(r) {
			return false
		}
	}
	return true
}

func toZapFields(values ...interface{}) []zapcore.Field {
	messages := make([]zapcore.Field, 0)
	if len(values) > 1 {
		var (
			sql             string
			formattedValues []string
			level           = values[0]
			source          = fmt.Sprintf("%v", values[1])
		)
		messages = append(messages, zap.String("source", source))
		if level == "sql" {
			// duration
			messages = append(messages, zap.String("duration", fmt.Sprintf("%.2fms", float64(values[2].(time.Duration).Nanoseconds()/1e4)/100.0)))
			// sql

			for _, value := range values[4].([]interface{}) {
				indirectValue := reflect.Indirect(reflect.ValueOf(value))
				if indirectValue.IsValid() {
					value = indirectValue.Interface()
					if t, ok := value.(time.Time); ok {
						formattedValues = append(formattedValues, fmt.Sprintf("'%v'", t.Format("2006-01-02 15:04:05")))
					} else if b, ok := value.([]byte); ok {
						if str := string(b); isPrintable(str) {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", str))
						} else {
							formattedValues = append(formattedValues, "'<binary>'")
						}
					} else if r, ok := value.(driver.Valuer); ok {
						if value, err := r.Value(); err == nil && value != nil {
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
						} else {
							formattedValues = append(formattedValues, "NULL")
						}
					} else {
						switch value.(type) {
						case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
							formattedValues = append(formattedValues, fmt.Sprintf("%v", value))
						default:
							formattedValues = append(formattedValues, fmt.Sprintf("'%v'", value))
						}
					}
				} else {
					formattedValues = append(formattedValues, "NULL")
				}
			}

			// differentiate between $n placeholders or else treat like ?
			if numericPlaceHolderRegexp.MatchString(values[3].(string)) {
				sql = values[3].(string)
				for index, value := range formattedValues {
					placeholder := fmt.Sprintf(`\$%d([^\d]|$)`, index+1)
					sql = regexp.MustCompile(placeholder).ReplaceAllString(sql, value+"$1")
				}
			} else {
				formattedValuesLength := len(formattedValues)
				for index, value := range sqlRegexp.Split(values[3].(string), -1) {
					sql += value
					if index < formattedValuesLength {
						sql += formattedValues[index]
					}
				}
			}

			messages = append(messages, zap.String("sql", sql))
			messages = append(messages, zap.Int64("affected_rows", values[5].(int64)))
		} else {
			for _, v := range values[2:] {
				messages = append(messages, zap.Any("other", v))
			}
		}
	}

	return messages
}

// New create logger object for *gorm.DB from *zap.Logger
func New(zap *zap.Logger) *Logger {
	return &Logger{
		zap: zap,
	}
}

// Logger is an alternative implementation of *gorm.Logger
type Logger struct {
	zap *zap.Logger
}

// Print passes arguments to Println
func (l *Logger) Print(values ...interface{}) {
	l.Println(values)
}

// Println format & print log
func (l *Logger) Println(values []interface{}) {
	if DefaultOutputLevel == OutputLevelDebug {
		l.zap.Debug("gormzap", toZapFields(values...)...)
	}else{
		l.zap.Info("gormzap", toZapFields(values...)...)
	}
}
