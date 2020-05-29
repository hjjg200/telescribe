package monitor

import (
    "fmt"
    "reflect"
    "strconv"
    "strings"
)

type wrapper struct {
    body interface{}
    coef float64
}

func Wrap(body interface{}, coef float64) wrapper {
    return wrapper{body, coef}
}

// CPU
const KeyCpuCount = "cpu-count"
const KeyCpuUsage = "cpu-usage"
// Memory
const KeyMemorySize =   "memory-size"
const KeyMemorySizeMB = "memory-size-mb"
const KeyMemorySizeGB = "memory-size-gb"
const KeyMemoryUsage =  "memory-usage"
const KeySwapSize =     "swap-size"
const KeySwapSizeMB =   "swap-size-mb"
const KeySwapSizeGB =   "swap-size-gb"
const KeySwapUsage =    "swap-usage"
// Load
const KeyLoadAverage =       "load"
const KeyLoadAveragePerCpu = "load-perCpu"
// Disk
const KeyDevWrites =      "dev-writes"
const KeyDevReads =       "dev-reads"
const KeyDevWriteBytes =  "dev-writeBytes"
const KeyDevReadBytes =   "dev-readBytes"
const KeyDevUsage =       "dev-usage"
const KeyDevSize =        "dev-size"
const KeyDevSizeMB =      "dev-size-mb"
const KeyDevSizeGB =      "dev-size-gb"
const KeyDevSizeTB =      "dev-size-tb"
const KeyDevIoUsage =     "dev-io-usage"
const KeyDevsWrites =     "devs-writes" // DEVS
const KeyDevsReads =      "devs-reads"
const KeyDevsWriteBytes = "devs-writeBytes"
const KeyDevsReadBytes =  "devs-readBytes"
const KeyDevsUsage =      "devs-usage"
const KeyDevsSize =       "devs-size"
const KeyDevsSizeMB =     "devs-size-mb"
const KeyDevsSizeGB =     "devs-size-gb"
const KeyDevsSizeTB =     "devs-size-tb"
const KeyDevsIoUsage =    "devs-io-usage"
// Network
const KeyNetworkIn =         "network-in"
const KeyNetworkInPackets =  "network-inPackets"
const KeyNetworkOut =        "network-out"
const KeyNetworkOutPackets = "network-outPackets"
// Process
const KeyProcessCpuUsage =    "process-cpu-usage"
const KeyProcessMemoryUsage = "process-memory-usage"
const KeyProcessSwapUsage =   "process-swap-usage"
const KeyProcessReadBytes =   "process-read-bytes"
const KeyProcessWriteBytes =  "process-write-bytes"
// Misc
const KeyCustomCommand = "command"

var Wrappers = map[string] wrapper {
    // CPU
    KeyCpuCount: Wrap(GetCpuCount, 1.0),
    KeyCpuUsage: Wrap(GetCpuUsage, 1.0),
    // Memory
    KeyMemorySize:   Wrap(GetMemoryTotal, 1.0),
    KeyMemorySizeMB: Wrap(GetMemoryTotal, 1.0e-3),
    KeyMemorySizeGB: Wrap(GetMemoryTotal, 1.0e-6),
    KeyMemoryUsage:  Wrap(GetMemoryUsage, 1.0),
    KeySwapSize:     Wrap(GetSwapTotal, 1.0),
    KeySwapSizeMB:   Wrap(GetSwapTotal, 1.0e-3),
    KeySwapSizeGB:   Wrap(GetSwapTotal, 1.0e-6),
    KeySwapUsage:    Wrap(GetSwapUsage, 1.0),
    // Load
    KeyLoadAverage:       Wrap(GetLoadAverage, 1.0),
    KeyLoadAveragePerCpu: Wrap(GetLoadAveragePerCpu, 1.0),
    // Disk
    KeyDevWrites:      Wrap(GetDevWrites, 1.0),
    KeyDevReads:       Wrap(GetDevReads, 1.0),
    KeyDevWriteBytes:  Wrap(GetDevWriteBytes, 1.0),
    KeyDevReadBytes:   Wrap(GetDevReadBytes, 1.0),
    KeyDevUsage:       Wrap(GetDevUsage, 1.0),
    KeyDevSize:        Wrap(GetDevSize, 1.0),
    KeyDevSizeMB:      Wrap(GetDevSize, 1.0e-3),
    KeyDevSizeGB:      Wrap(GetDevSize, 1.0e-6),
    KeyDevSizeTB:      Wrap(GetDevSize, 1.0e-9),
    KeyDevIoUsage:     Wrap(GetDevIoUsage, 1.0),
    KeyDevsWrites:     Wrap(GetDevsWrites, 1.0),
    KeyDevsReads:      Wrap(GetDevsReads, 1.0),
    KeyDevsWriteBytes: Wrap(GetDevsWriteBytes, 1.0),
    KeyDevsReadBytes:  Wrap(GetDevsReadBytes, 1.0),
    KeyDevsUsage:      Wrap(GetDevsUsage, 1.0),
    KeyDevsSize:       Wrap(GetDevsSize, 1.0),
    KeyDevsSizeMB:     Wrap(GetDevsSize, 1.0e-3),
    KeyDevsSizeGB:     Wrap(GetDevsSize, 1.0e-6),
    KeyDevsSizeTB:     Wrap(GetDevsSize, 1.0e-9),
    KeyDevsIoUsage:    Wrap(GetDevsIoUsage, 1.0),
    // Network
    KeyNetworkIn:         Wrap(GetNetworkIn, 1.0),
    KeyNetworkInPackets:  Wrap(GetNetworkInPackets, 1.0),
    KeyNetworkOut:        Wrap(GetNetworkOut, 1.0),
    KeyNetworkOutPackets: Wrap(GetNetworkOutPackets, 1.0),
    // Process
    KeyProcessCpuUsage:    Wrap(GetProcessCpuUsage, 1.0),
    KeyProcessMemoryUsage: Wrap(GetProcessMemoryUsage, 1.0),
    KeyProcessSwapUsage:   Wrap(GetProcessSwapUsage, 1.0),
    KeyProcessReadBytes:   Wrap(GetProcessReadBytes, 1.0),
    KeyProcessWriteBytes:  Wrap(GetProcessWriteBytes, 1.0),
    // Misc
    KeyCustomCommand: Wrap(CustomCommand, 1.0),
}

const (
    railRead = iota
    railWrite
)

var ErrorCallback = func(err error) {
    fmt.Println("monitor:", err)
}

func emitError(err error) {
    if ErrorCallback != nil {
        ErrorCallback(err)
    }
}

func Getter(longKey string) (func() map[string] float64, bool) {

    base, param, idx := ParseWrapperKey(longKey)
    baseWrapper, ok := Wrappers[base]
    if !ok {
        return nil, false
    }

    return func() map[string] float64 {
        out := baseWrapper.Get(param)
        switch cast := out.(type) {
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
            ret := make(map[string] float64)
            for i, v := range cast {
                ret[FormatWrapperKey(base, param, i)] = v
            }
            return ret
        }
        return nil
    }, true

}

func FormatWrapperKey(base, param, idx string) string {
    qtchars := `")([]`
    key := base
    if param != "" {
        if strings.ContainsAny(param, qtchars) {
            param = strconv.Quote(param)
        }
        key += "(" + param + ")"
    }
    if idx != "" {
        if strings.ContainsAny(idx, qtchars) {
            idx = strconv.Quote(idx)
        }
        key += "[" + idx + "]"
    }
    return key
}

func ParseWrapperKey(key string) (base, param, idx string) {

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
        case pBase: base += string(r)
        case pParam: param += string(r)
        case pIdx: idx += string(r)
        }
    }
    
    return

}

func (w wrapper) Get(param string) interface{} {

    defer recover()

    fn := reflect.ValueOf(w.body)
    ins := make([]reflect.Value, 0)
    if param != "" {
        ins = append(ins, reflect.ValueOf(param))
    }
    outs := fn.Call(ins)
    val := outs[0]

    // Check if it has the second return, which must be error, and it is not nil
    if len(outs) > 1 {
        if outs[1].Interface() != nil {
            return nil
        }
    }

    // Coefficient
    coef := w.coef

    switch cast := val.Interface().(type) {
    case float64:
        return cast * coef
    case []float64:
        ret := make(map[string] float64)
        for i := 0; i < len(cast); i++ {
            ret[fmt.Sprint(i)] = cast[i] * coef
        }
        return ret
    case map[string] float64:
        for key := range cast {
            cast[key] *= coef
        }
        return cast
    }

    return nil

}