package log

import (
    "os"
    "testing"
    "time"
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

func TestDebugLineNumbers(t *testing.T) {

    lgr := Logger{}
    lgr.AddWriter(os.Stderr, ANSIColorer)
    Debug = true
    DebugFilter, _ = NewFilter(".*")
    func() {
        lgr.Debugln("inside A")
        func() {
            lgr.Debugln("inside B")
            func() {
                lgr.Warnln("inside C")
            }()
        }()
    }()
    lgr.Debugln("D")

}

func TestTimer(t *testing.T) {
    
    lgr := Logger{}
    lgr.AddWriter(os.Stderr, ANSIColorer)
    Debug = true
    DebugFilter, _ = NewFilter(".*")

    lgr.Infoln("TIME")
    for i := 0; i < 10; i++ {
        go func() {
            tm2 := lgr.Timer("A")
            time.Sleep(time.Millisecond * 100)
            tm2.Stop()
        }()
    }
    time.Sleep(time.Millisecond * 220)
    lgr.Infoln("TIME")

}