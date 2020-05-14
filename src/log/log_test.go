package log

import (
    "os"
    "testing"
)

func TestANSIColorer(t *testing.T) {
    lgr := Logger{}
    lgr.AddWriter(os.Stderr, ANSIColorer)

    lgr.Infoln("abc")

    lgr.Warnln(Red("red"), Green("green"), Gray("gray"), Yellow("yellow"))
    lgr.Infoln(Cyan(123), Blue(5.67), Magenta([]byte("efgh")))
}

func TestNoColorer(t *testing.T) {
    lgr := Logger{}
    lgr.AddWriter(os.Stderr, nil)

    lgr.Infoln("info")
    lgr.Warnln("warn")
    lgr.Infoln(Red("no red"), Green("no green"))
}

func TestDebugFilter(t *testing.T) {
    lgr := Logger{}
    lgr.AddWriter(os.Stderr, ANSIColorer)

    repeat := func() {
        lgr.Debugln("abc1", "abc1")
        lgr.Debugln("def2", "def2")
        lgr.Debugln("xyz3", "xyz3")
    }

    Debug = true

    lgr.Infoln("Empty filter")
    repeat()

    DebugFilter, _ = NewFilter("(abc|def)\\d")
    
    lgr.Infoln("Filter = (abc|def)\\d")
    repeat()

    DebugFilter, _ = NewFilter(".+3")
    
    lgr.Infoln("Filter = .+3")
    repeat()
    
}