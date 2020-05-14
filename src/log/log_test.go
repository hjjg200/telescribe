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