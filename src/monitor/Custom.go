package monitor

import (
    "fmt"
    "os/exec"
    "strconv"
    "strings"
)

func CustomCommand( cmd string ) ( float64, error ) {
    if cmd == "" {
        return 0.0, fmt.Errorf( "Given command is empty" )
    }
    ex := exec.Command( "bash", "-c", cmd )
    out, err := ex.Output()
    if err != nil {
        return 0.0, err
    }
    str := string( out )
    str = strings.Trim( str, " \r\n\t" ) // remove \t, \r, \n, <space>
    f, err := strconv.ParseFloat( str, 64 )
    if err != nil {
        return 0.0, err
    }
    return f, nil
}