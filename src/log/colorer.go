package log

import (
    "fmt"
)

// Color codes

const (
    clRed = iota
    clGreen
    clYellow
    clBlue
    clMagenta
    clCyan
    clGray
)

// logEntry

type logEntry struct {
    color int
    val interface{}
}

var (
    prefixInfo  = Green(" INFO ")
    prefixWarn  = Yellow(" WARN ")
    prefixFatal = Red("FATAL!")
    prefixPanic = Red("PANIC!")
    prefixDebug = Magenta("+DEBUG")
)

func Red(v interface{}) logEntry {
    return logEntry{color: clRed, val: v}
}
func Green(v interface{}) logEntry {
    return logEntry{color: clGreen, val: v}
}
func Yellow(v interface{}) logEntry {
    return logEntry{color: clYellow, val: v}
}
func Blue(v interface{}) logEntry {
    return logEntry{color: clBlue, val: v}
}
func Magenta(v interface{}) logEntry {
    return logEntry{color: clMagenta, val: v}
}
func Cyan(v interface{}) logEntry {
    return logEntry{color: clCyan, val: v}
}
func Gray(v interface{}) logEntry {
    return logEntry{color: clGray, val: v}
}

func(le logEntry) Colorify(clr Colorer) string {

    if clr != nil {
        switch le.color {
        case clRed: return clr.Red(le.val)
        case clGreen: return clr.Green(le.val)
        case clYellow: return clr.Yellow(le.val)
        case clBlue: return clr.Blue(le.val)
        case clMagenta: return clr.Magenta(le.val)
        case clCyan: return clr.Cyan(le.val)
        case clGray: return clr.Gray(le.val)
        }
    }

    // Default
    return fmt.Sprint(le.val)
}

// Colorer

type Colorer interface{
    Red(interface{}) string
    Green(interface{}) string
    Yellow(interface{}) string
    Blue(interface{}) string
    Magenta(interface{}) string
    Cyan(interface{}) string
    Gray(interface{}) string
}

// interface{} to string helper
func i2s(val interface{}) string {
    return fmt.Sprint(val)
}

// ANSI Colorer

type ansiColorer struct{}
var ANSIColorer = ansiColorer{}
const ansiReset = "\033[0m"
func(ac ansiColorer) Red(v interface{}) string {
    return "\033[31;1m" + i2s(v) + ansiReset
}
func(ac ansiColorer) Green(v interface{}) string {
    return "\033[32;1m" + i2s(v) + ansiReset
}
func(ac ansiColorer) Yellow(v interface{}) string {
    return "\033[33;1m" + i2s(v) + ansiReset
}
func(ac ansiColorer) Blue(v interface{}) string {
    return "\033[34;1m" + i2s(v) + ansiReset
}
func(ac ansiColorer) Magenta(v interface{}) string {
    return "\033[35;1m" + i2s(v) + ansiReset
}
func(ac ansiColorer) Cyan(v interface{}) string {
    return "\033[36;1m" + i2s(v) + ansiReset
}
func(ac ansiColorer) Gray(v interface{}) string {
    return "\033[37m" + i2s(v) + ansiReset
}