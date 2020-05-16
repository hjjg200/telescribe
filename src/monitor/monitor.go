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
const KeyMemorySize = "memory-size"
const KeyMemorySizeMB = "memory-size-mb"
const KeyMemorySizeGB = "memory-size-gb"
const KeyMemoryUsage = "memory-usage"
const KeySwapSize = "swap-size"
const KeySwapSizeMB = "swap-size-mb"
const KeySwapSizeGB = "swap-size-gb"
const KeySwapUsage = "swap-usage"
// Load
const KeyLoadAverage = "load"
const KeyLoadAveragePerCpu = "load-perCpu"
// Disk
const KeyDiskWrites = "disk-writes"
const KeyMountWrites = "mount-writes"
const KeyDiskReads = "disk-reads"
const KeyMountReads  = "mount-reads"
const KeyDiskWriteBytes = "disk-writeBytes"
const KeyMountWriteBytes = "mount-writeBytes"
const KeyDiskReadBytes = "disk-readBytes"
const KeyMountReadBytes = "mount-readBytes"
const KeyDiskUsage = "disk-usage"
const KeyMountUsage  = "mount-usage"
const KeyDiskSize = "disk-size"
const KeyDiskSizeMB = "disk-size-mb"
const KeyDiskSizeGB = "disk-size-gb"
const KeyDiskSizeTB = "disk-size-tb"
const KeyMountSize = "mount-size"
const KeyMountSizeMB = "mount-size-mb"
const KeyMountSizeGB = "mount-size-gb"
const KeyMountSizeTB = "mount-size-tb"
// Network
const KeyNetworkIn = "network-in"
const KeyNetworkInPackets = "network-inPackets"
const KeyNetworkOut = "network-out"
const KeyNetworkOutPackets = "network-outPackets"
// Misc
const KeyCustomCommand = "command"

var Wrappers = map[string] wrapper {
    // CPU
    KeyCpuCount: Wrap(GetCpuCount, 1.0),
    KeyCpuUsage: Wrap(GetCpuUsage, 1.0),
    // Memory
    KeyMemorySize: Wrap(GetMemoryTotal, 1.0),
    KeyMemorySizeMB: Wrap(GetMemoryTotal, 1.0e-3),
    KeyMemorySizeGB: Wrap(GetMemoryTotal, 1.0e-6),
    KeyMemoryUsage: Wrap(GetMemoryUsage, 1.0),
    KeySwapSize: Wrap(GetSwapTotal, 1.0),
    KeySwapSizeMB: Wrap(GetSwapTotal, 1.0e-3),
    KeySwapSizeGB: Wrap(GetSwapTotal, 1.0e-6),
    KeySwapUsage: Wrap(GetSwapUsage, 1.0),
    // Load
    KeyLoadAverage: Wrap(GetLoadAverage, 1.0),
    KeyLoadAveragePerCpu: Wrap(GetLoadAveragePerCpu, 1.0),
    // Disk
    KeyDiskWrites: Wrap(GetDiskWrites, 1.0),
    KeyMountWrites: Wrap(GetMountWrites, 1.0),
    KeyDiskReads: Wrap(GetDiskReads, 1.0),
    KeyMountReads: Wrap(GetMountReads, 1.0),
    KeyDiskWriteBytes: Wrap(GetDiskWriteBytes, 1.0),
    KeyMountWriteBytes: Wrap(GetMountWriteBytes, 1.0),
    KeyDiskReadBytes: Wrap(GetDiskReadBytes, 1.0),
    KeyMountReadBytes: Wrap(GetMountReadBytes, 1.0),
    KeyDiskUsage: Wrap(GetDiskUsage, 1.0),
    KeyMountUsage: Wrap(GetMountUsage, 1.0),
    KeyDiskSize: Wrap(GetDiskSize, 1.0),
    KeyDiskSizeMB: Wrap(GetDiskSize, 1.0e-3),
    KeyDiskSizeGB: Wrap(GetDiskSize, 1.0e-6),
    KeyDiskSizeTB: Wrap(GetDiskSize, 1.0e-9),
    KeyMountSize: Wrap(GetMountSize, 1.0),
    KeyMountSizeMB: Wrap(GetMountSize, 1.0e-3),
    KeyMountSizeGB: Wrap(GetMountSize, 1.0e-6),
    KeyMountSizeTB: Wrap(GetMountSize, 1.0e-9),
    // Network
    KeyNetworkIn: Wrap(GetNetworkIn, 1.0),
    KeyNetworkInPackets: Wrap(GetNetworkInPackets, 1.0),
    KeyNetworkOut: Wrap(GetNetworkOut, 1.0),
    KeyNetworkOutPackets: Wrap(GetNetworkOutPackets, 1.0),
   // Misc
    KeyCustomCommand: Wrap(CustomCommand, 1.0),
}

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