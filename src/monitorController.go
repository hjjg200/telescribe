package main

import (
    "crypto/sha256"
    "encoding/binary"
    "fmt"
    "math"
    "./monitor"
   // "./secret"

    . "github.com/hjjg200/go-act"
)

func init() {
    // Monitor error callback
    monitor.ErrorCallback = func(err error) {
        EventLogger.Warnln("monitor:", err)
    }
}

// STATUS ---

const (
    MonitorStatusNormal  = 0
    MonitorStatusWarning = 8
    MonitorStatusFatal   = 16
)

// DATA ---

type MonitorDataTableBox struct {
    Boundaries []byte
    DataMap    map[string/* monitorKey */] []byte
}

type MonitorDataMap map[string/* monitorKey */] MonitorData
type MonitorData []MonitorDatum
const MonitorDatumVersion uint8 = 1
const MonitorDatumSize int = 20
type MonitorDatum struct {
    Timestamp int64 // timestamp is int64 as go uses int64 for unix timetamps
    Value float64
    Per int32
}

// TODO these functions are liable to panic
func(md MonitorData) From() int64 {
    return md[0].Timestamp - int64(md[0].Per)
}
func(md MonitorData) To() int64 {
    return md[len(md) - 1].Timestamp
}
func(md MonitorData) Duration() int64 {
    return md.To() - md.From()
}
func(md MonitorData) MidTime() float64 {
    return float64(md.To() + md.From()) / 2.0
}
func(md MonitorData) MinMax() (float64, float64) {
    min, max := math.Inf(0), math.Inf(-1)
    for _, datum := range md {
        val := datum.Value
        if val < min {min = val}
        if val > max {max = val}
    }
    return min, max
}

// Aggregate Types ---

const (
    monitorAggregateKeyMean = "mean"
    monitorAggregateKeyMin = "min"
    monitorAggregateKeyMax = "max"
    monitorAggregateKeySum = "sum"
)

var monitorAggregateTypesMap = map[string] monitorAggregateFunc{
    monitorAggregateKeyMean: monitorAggregateMean,
    monitorAggregateKeyMin: monitorAggregateMin,
    monitorAggregateKeyMax: monitorAggregateMax,
    monitorAggregateKeySum: monitorAggregateSum,
}

type monitorAggregateFunc func(MonitorData) float64
func monitorAggregateMean(md MonitorData) float64 {
    accm := 0.0
    for _, datum := range md {
        accm += datum.Value
    }
    return accm / float64(len(md))
}
func monitorAggregateMin(md MonitorData) float64 {
    min := math.Inf(0)
    for _, datum := range md {
        if datum.Value < min {
            min = datum.Value
        }
    }
    return min
}
func monitorAggregateMax(md MonitorData) float64 {
    max := math.Inf(-1)
    for _, datum := range md {
        if datum.Value > max {
            max = datum.Value
        }
    }
    return max
}
func monitorAggregateSum(md MonitorData) float64 {
    sum := 0.0
    for _, datum := range md {
        sum += datum.Value
    }
    return sum
}


// Index ---
type MonitorDataIndex struct {
    Uuid   string  `json:"uuid"`
    Length int     `json:"length"`
    From   int64   `json:"from"`
    To     int64   `json:"to"`
    Min    float64 `json:"min"`
    Max    float64 `json:"max"`
}
type MonitorDataIndexes []MonitorDataIndex
type MonitorDataIndexesMap map[string/* monitorKey */] MonitorDataIndexes

/*

Indexes are all json formatted
Indexes must be updated when a new chunk of data is created
Copy portion of data(server) -> created uuid for that portion(MC)
-> encode it and store it(server) -> create index(MC) -> remove that portion atomically(server)


*/

func CreateUuidForMonitorData(md MonitorData) string {
    serial := SerializeMonitorData(md)
    h := sha256.New()
    h.Write(serial)
    return fmt.Sprintf("%x", h.Sum(nil))
}

func CreateIndexForMonitorData(md MonitorData) MonitorDataIndex {

    mi := MonitorDataIndex{}
    if len(md) == 0 {
        return mi
    }

    // Set
    mi.Uuid        = CreateUuidForMonitorData(md)
    mi.Length      = len(md)
    mi.From, mi.To = md.From(), md.To()
    mi.Min, mi.Max = md.MinMax()

    return mi

}

func(mdIndexes MonitorDataIndexes) MinMax() (float64, float64) {
    min, max := math.Inf(0), math.Inf(-1)
    for _, mi := range mdIndexes {
        if mi.Min < min {min = mi.Min}
        if mi.Max > max {max = mi.Max}
    }
    return min, max
}

func(mdIndexes MonitorDataIndexes) Append(rhs ...MonitorDataIndex) MonitorDataIndexes {

    // Copy
    copied := make(MonitorDataIndexes, len(rhs))
    copy(copied, MonitorDataIndexes(rhs))

    // If nil or zero
    // because of this, the function returns indexes, not utilizing reference
    if mdIndexes == nil || len(mdIndexes) == 0 {
        return copied
    }

    return append(mdIndexes, copied...)

}

// sort.Interface
func(md MonitorData) Len() int { return len(md) }
func(md MonitorData) Less(i, j int) bool { return md[i].Timestamp < md[j].Timestamp }
func(md MonitorData) Swap(i, j int) { md[i], md[j] = md[j], md[i] }

// CONFIG ---

type MonitorConfig struct {
    Absolute     bool   `json:"absolute"`
    Alias        string `json:"alias"`
    Constant     bool   `json:"constant"`
    Format       string `json:"format"`
    FatalRange   Range  `json:"fatalRange"`
    WarningRange Range  `json:"warningRange"`
}
type MonitorConfigMap map[string/* monitorKey */] MonitorConfig

func(mCfg MonitorConfig) StatusOf(val float64) int {
    switch {
    case mCfg.FatalRange.Includes(val):
        return MonitorStatusFatal
    case mCfg.WarningRange.Includes(val):
        return MonitorStatusWarning
    }
    return MonitorStatusNormal
}

// KEY ---

type monitorKey struct {
    base, param, idx string
}
var parsedMonitorKeys = make(map[string] monitorKey)

func ParseMonitorKey(mKey string) (base, param, idx string) {
    if m, ok := parsedMonitorKeys[mKey]; ok {
        base, param, idx = m.base, m.param, m.idx
    } else {
        base, param, idx        = monitor.ParseWrapperKey(mKey)
        parsedMonitorKeys[mKey] = monitorKey{
            base, param, idx,
        }
    }
    return
}
func FormatMonitorKey(base, param, idx string) string {
    return monitor.FormatWrapperKey(base, param, idx)
}

// SERIALIZATION ---

func SerializeMonitorData(md MonitorData) []byte {

    // Type version check
    Assert(MonitorDatumVersion == 1, "MonitorDatum version must be 1")

    // Byte
    lenMd    := len(md)
    lenTotal := 1 + lenMd * MonitorDatumSize // version byte + data bytes
    serial   := make([]byte, lenTotal)

    // Version byte
    serial[0] = byte(MonitorDatumVersion)

    // Data
    le   := binary.LittleEndian
    base := 1 // from 1st position
    for _, datum := range md {
        le.PutUint64(serial[base:base + 8],       uint64(datum.Timestamp))
        le.PutUint64(serial[base + 8:base + 16],  math.Float64bits(datum.Value))
        le.PutUint32(serial[base + 16:base + 20], uint32(datum.Per))

        base += MonitorDatumSize
    }

    return serial

}

func DeserializeMonitorData(serial []byte) (md MonitorData, err error) {

    // Type version check
    Assert(MonitorDatumVersion == 1, "MonitorDatum version must be 1")

    defer Catch(&err) // recover from panic from this point

    // Version byte
    version := serial[0]
    Assert(uint8(version) == MonitorDatumVersion, "MonitorData version must match the current version")

    // Make
    mdPart := serial[1:]
    lenMd  := len(mdPart) / MonitorDatumSize
    md = make(MonitorData, lenMd)

    // Parse
    le := binary.LittleEndian
    for i := range md {
        base := i * MonitorDatumSize
        md[i].Timestamp = int64(le.Uint64(mdPart[base:base + 8]))
        md[i].Value = math.Float64frombits(le.Uint64(mdPart[base + 8:base + 16]))
        md[i].Per = int32(le.Uint32(mdPart[base + 16:base + 20]))
    }

    return md, nil

}
