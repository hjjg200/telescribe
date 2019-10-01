package tc

import (
    "fmt"
)

func Try(err error) {
    if err != nil {
        panic(err)
    }
}

func Catch(err *error) {
    r := recover()
    if r != nil {
        *err = fmt.Errorf("%v", r)
    }
}

func Assert(t bool, msg string) {
    if !t {
        panic(msg)
    }
}