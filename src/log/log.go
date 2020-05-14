package log

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
    "runtime"
)

type logEntry struct {
    color int
    val interface{}
}

type Logger struct {
    writers []Writer
}

type Writer struct {
    w io.Writer
    clr Colorer
}

var (
    prefixInfo = logEntry{color: clGreen, val: " INFO "}
    prefixWarn = logEntry{color: clYellow, val: " WARN "}
    prefixFatal = logEntry{color: clRed, val: "FATAL!"}
    prefixPanic = logEntry{color: clRed, val: "PANIC!"}
    prefixDebug = logEntry{color: clMagenta, val: "+DEBUG"}
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

func (lgr *Logger) AddWriter(w io.Writer, clr Colorer) {
    lgr.writers = append(lgr.writers, Writer{ w, clr })
}

func (lgr *Logger) Infoln(args ...interface{}) {
    lgr.println(prefixInfo, args...)
}

func (lgr *Logger) Warnln(args ...interface{}) {
    if Debug {
        args = append(lgr.callers(3), args...)
    }
    lgr.println(prefixWarn, args...)
}

func (lgr *Logger) Fatalln(args ...interface{}) {
    if Debug {
        args = append(lgr.callers(3), args...)
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
    args = append(lgr.callers(3), args...)
    lgr.println(prefixDebug, args...)
}

func (lgr *Logger) println(prefix logEntry, args ...interface{}) {
    args = append(args, "\n")
    lgr.print(prefix, args...)
}

func (lgr *Logger) print(prefix logEntry, args ...interface{}) {
    for _, w := range lgr.writers {
        w.print(prefix, args...)
    }
}

func (w *Writer) print(prefix logEntry, args ...interface{}) {

    out := "[" + prefix.Colorify(w.clr) + "] "
    out += secondsFromStart()
    out += " - "

    for i := range args {
        if i > 0 {
            out += " "
        }
        switch v := args[i].(type) {
        case logEntry:
            out += v.Colorify(w.clr)
        default:
            out += fmt.Sprint(args[i])
        }
    }

    fmt.Fprint(w.w, out)

}

func (lgr *Logger) callers(skip int) []interface{} {
    pcs := make([]uintptr, 128)
    count := runtime.Callers(skip, pcs)
    ret := make([]interface{}, 0)
    for i := count - 1; i >= 0; i-- {
        pc := pcs[i]
        fn := runtime.FuncForPC(pc)
        f, l := fn.FileLine(pc)
        dir := filepath.Base(filepath.Dir(f))
        f = dir + "/" + filepath.Base(f)
        n := filepath.Base(fn.Name())
        ret = append(ret, fmt.Sprintf("%s[%s:%d]\n  ", n, f, l))
    }
    return ret
}
