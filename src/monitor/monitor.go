package monitor

import (
    "fmt"
    "reflect"
    "strconv"
    "strings"
)

type wrapper struct {
    body interface{}
}

const KeyCpuCount = "general-cpuCount"
const KeyCpuUsage = "general-cpuUsage"
const KeyMemoryTotal = "general-memoryTotal"
const KeySwapTotal = "general-swapTotal"
const KeyMemoryUsage = "general-memoryUsage"
const KeySwapUsage = "general-swapUsage"
const KeyLoadAverage = "general-loadAverage"
const KeyLoadAveragePerCpu = "general-loadAveragePerCpu"
const KeyCustomCommand = "custom-command"

// Remove string range part
// Improve range
//   3:,:10

var Wrappers = map[string] wrapper {
    KeyCpuCount: wrapper{ GetCpuCount },
    KeyCpuUsage: wrapper{ GetCpuUsage },
    KeyMemoryTotal: wrapper{ GetMemoryTotal },
    KeySwapTotal: wrapper{ GetSwapTotal },
    KeyMemoryUsage: wrapper{ GetMemoryUsage },
    KeySwapUsage: wrapper{ GetSwapUsage },
    KeyLoadAverage: wrapper{ GetLoadAverage },
    KeyLoadAveragePerCpu: wrapper{ GetLoadAveragePerCpu },
    KeyCustomCommand: wrapper{ CustomCommand },
}

func Getter( longKey string ) ( func() map[string] float64, bool ) {

    base, param, idx := ParseWrapperKey( longKey )
    baseWrapper, ok := Wrappers[base]
    if !ok {
        return nil, false
    }

    return func() map[string] float64 {
        out := baseWrapper.Get( param )
        switch cast := out.( type ) {
        case float64:
            if idx != "" {
                return nil
            }
            return map[string] float64{
                longKey: cast,
            }
        case map[string] float64:
            if idx != "" {
                return map[string] float64{
                    longKey: cast[idx],
                }
            }
            ret := make( map[string] float64 )
            for i, v := range cast {
                ret[FormatWrapperKey( base, param, i )] = v
            }
            return ret
        }
        return nil
    }, true

}

func FormatWrapperKey( base, param, idx string ) string {
    qtchars := `")([]`
    key := base
    if param != "" {
        if strings.ContainsAny( param, qtchars ) {
            param = strconv.Quote( param )
        }
        key += "(" + param + ")"
    }
    if idx != "" {
        if strings.ContainsAny( idx, qtchars ) {
            idx = strconv.Quote( idx )
        }
        key += "[" + idx + "]"
    }
    return key
}

func ParseWrapperKey( key string ) ( base, param, idx string ) {

    const (
        pBase = iota
        pParam
        pIdx
        pNil
    )

    parseMode := pBase
    quoted := false
    escaped := false

    for _, r := range key {
        switch r {
        case '\\':
            if !escaped {
                escaped = true
                continue
            } else {
                escaped = false
            }
        case '"':
            if escaped {
                escaped = false
            } else {
                quoted = !quoted
                continue
            }
        case '(':
            if !quoted {
                parseMode = pParam
                continue
            }
        case ')':
            if !quoted {
                parseMode = pNil
                continue
            }
        case '[':
            if !quoted {
                parseMode = pIdx
                continue
            }
        case ']':
            if !quoted {
                parseMode = pNil
                continue
            }
        }
        switch parseMode {
        case pBase: base += string( r )
        case pParam: param += string( r )
        case pIdx: idx += string( r )
        }
    }
    
    return

}

func ( w wrapper ) Get( param string ) interface{} {

    fn := reflect.ValueOf( w.body )
    ins := make( []reflect.Value, 0 )
    if param != "" {
        ins = append( ins, reflect.ValueOf( param ) )
    }
    outs := fn.Call( ins )
    val := outs[0]

    if len( outs ) > 1 {
        if outs[1].Interface() != nil {
            return nil
        }
    }

    switch cast := val.Interface().( type ) {
    case float64:
        return cast
    case []float64:
        ret := make( map[string] float64 )
        for i := 0; i < len( cast ); i++ {
            ret[fmt.Sprint( i )] = cast[i]
        }
        return ret
    case map[string] float64:
        return cast
    }

    return nil

}