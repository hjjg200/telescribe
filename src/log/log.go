package log

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
    "runtime"
    "strings"
)

type Logger struct {
    writers []Writer
}

type Writer struct {
    w io.Writer
    color bool
}

const (
    clGreen = "\033[32;1m"
    clYellow = "\033[33;1m"
    clRed = "\033[31;1m"
    clReset = "\033[0m"
    prefixInfo = clGreen + "INFO" + clReset
    prefixWarn = clYellow + "WARN" + clReset
    prefixFatal = clRed + "FATAL!" + clReset
    prefixPanic = clRed + "PANIC!" + clReset
    prefixDebug = "+DEBUG"
)

var logStartTime time.Time
var Debug = false

func secondsFromStart() string {
    var t time.Time
    if logStartTime == t {
        logStartTime = time.Now()
        return fmt.Sprintf("%11d", time.Now().Unix())
    }
    return fmt.Sprintf("%11.3f", float64(time.Now().Sub(logStartTime)) / float64(time.Second))
}

func (lgr *Logger) AddWriter(w io.Writer, cl bool) {
    lgr.writers = append(lgr.writers, Writer{ w, cl })
}

func (lgr *Logger) Infoln(args ...interface{}) {
    lgr.println(prefixInfo, args...)
}

func (lgr *Logger) Warnln(args ...interface{}) {
    if Debug {
        args = append([]interface{}{ lgr.caller(2) }, args...)
    }
    lgr.println(prefixWarn, args...)
}

func (lgr *Logger) Fatalln(args ...interface{}) {
    if Debug {
        args = append([]interface{}{ lgr.caller(2) }, args...)
    }
    lgr.println(prefixFatal, args...)
    os.Exit(1)
}

func (lgr *Logger) Panicln(args ...interface{}) {
    lgr.print(prefixPanic, "")
    panic(fmt.Sprintln(args...))
}

func (lgr *Logger) Debugln(args ...interface{}) {
    if !Debug {
        return
    }
    args = append([]interface{}{ lgr.caller(2) }, args...)
    lgr.println(prefixDebug, args...)
}

func (lgr *Logger) caller(skip int) string {
    pc, _, _, ok := runtime.Caller(skip)
    if !ok  {
        return ""
    }
    fn := runtime.FuncForPC(pc)
    f, l := fn.FileLine(pc)
    dir := filepath.Base(filepath.Dir(f))
    f = dir + "/" + filepath.Base(f)
    n := filepath.Base(fn.Name())
    return fmt.Sprintf("%s[%s:%d]", n, f, l)
}

func (lgr *Logger) println(prefix string, args ...interface{}) {
    args = append(args, "\n")
    lgr.print(prefix, args...)
}

func (lgr *Logger) print(prefix string, args ...interface{}) {
    for _, w := range lgr.writers {
        w.print(prefix, args...)
    }
}

func (w *Writer) print(prefix string, args ...interface{}) {

    if !w.color {
        prefix = stripAnsiColor(prefix)
    }

    out := "[" + padPrefix(prefix, 6) + "] "
    out += secondsFromStart()
    out += " - "

    for i := range args {
        if i > 0 {
            out += " "
        }
        out += fmt.Sprint(args[i])
    }

    fmt.Fprint(w.w, out)

}

func padPrefix(str string, width int) string {

    strLen := len(stripAnsiColor(str))
    if strLen > width {
        return str
    }

    for diff := width - strLen; diff > 0; diff-- {
        if diff % 2 == 1 {
            str += " "
        } else {
            str = " " + str
        }
    }
    return str
}

func stripAnsiColor(str string) string {

    for i := 0; i < 5; i++ {
        pos := strings.Index(str, "\033")
        if pos == -1 {
            break
        }
        pos2 := strings.Index(str[pos:], "m") + pos
        if pos2 == -1 {
            break
        }
        if pos2 == len(str) - 1 {
            str = str[:pos]
        } else {
            str = str[:pos] + str[pos2 + 1:]
        }
    }
    return str

}