package log

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"
    "runtime"
)

var logStartTime time.Time

func secondsFromStart() string {
    if logStartTime == (time.Time{}) {
        logStartTime = time.Now()
        return fmt.Sprintf("%11d", time.Now().Unix())
    }
    return fmt.Sprintf("%11.3f", float64(time.Now().Sub(logStartTime)) / float64(time.Second))
}

// LOGGER ---

type Logger struct {
    writers []Writer
}

func NewLogger() *Logger {
    return &Logger{}
}

func(lgr *Logger) AddWriter(w io.Writer, clr Colorer) {
    lgr.writers = append(lgr.writers, Writer{ w, clr })
}

func(lgr *Logger) Infoln(args ...interface{}) {
    lgr.println(prefixInfo, args...)
}

func(lgr *Logger) Warnln(args ...interface{}) {
    if Debug {
        args = append(args, lgr.callers(3)...)
    }
    lgr.println(prefixWarn, args...)
}

func(lgr *Logger) Fatalln(args ...interface{}) {
    if Debug {
        args = append(args, lgr.callers(3)...)
    }
    lgr.println(prefixFatal, args...)
    os.Exit(1)
}

func(lgr *Logger) Panicln(args ...interface{}) {
    lgr.print(prefixPanic, "")
    panic(fmt.Sprintln(args...))
}

func(lgr *Logger) println(prefix logEntry, args ...interface{}) {
    args = append(args, "\n")
    lgr.print(prefix, args...)
}

func(lgr *Logger) print(prefix logEntry, args ...interface{}) {
    for _, w := range lgr.writers {
        w.print(prefix, args...)
    }
}

// WRITER ---

type Writer struct {
    w io.Writer
    clr Colorer
}

func(w *Writer) print(prefix logEntry, args ...interface{}) {

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

func(lgr *Logger) callers(skip int) []interface{} {

    pcs    := make([]uintptr, 128)
    count  := runtime.Callers(skip, pcs)
    frames := runtime.CallersFrames(pcs[0:count - 1])
    ret    := make([]interface{}, 0)

    for {

        frame, more := frames.Next()
        if !more {
            break
        }

        f, l := frame.File, frame.Line
        dir  := filepath.Base(filepath.Dir(f))
        f     = dir + "/" + filepath.Base(f)
        n    := filepath.Base(frame.Func.Name())

        ret   = append(
            []interface{}{fmt.Sprintf("\n  %s[%s:%d]", n, f, l)}, ret...
        )

    }

    return ret

}
