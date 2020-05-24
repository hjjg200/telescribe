package main

import (
    "bufio"
    "bytes"
    "compress/gzip"
    "encoding/gob"
    "math"
    "./monitor"

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
type MonitorDatum struct {
    Timestamp int64
    Value float64
    Per int
}

// sort.Interface
func (md MonitorData) Len() int { return len(md) }
func (md MonitorData) Less(i, j int) bool { return md[i].Timestamp < md[j].Timestamp }
func (md MonitorData) Swap(i, j int) { md[i], md[j] = md[j], md[i] }
// For LTTB
func (datum MonitorDatum) X() float64 { return float64(datum.Timestamp) }
func (datum MonitorDatum) Y() float64 { return datum.Value }

// CONFIG ---

type MonitorConfig struct {
    Alias        string `json:"alias"`
    Constant     bool   `json:"constant"`
    Format       string `json:"format"`
    FatalRange   Range  `json:"fatalRange"`
    WarningRange Range  `json:"warningRange"`
    Relative     bool   `json:"relative"`
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


// COMPRESSION ---

func CompressMonitorData(md MonitorData) (cmp []byte, err error) {

    defer Catch(&err)

    buf := bytes.NewBuffer(nil)
    gzw := gzip.NewWriter(buf)
    enc := gob.NewEncoder(gzw)

    // |       entire content       |
    // | type | timestamps | values |

    if len(md) > 0 {

        enc.Encode("float64")

        timestamps := make([]int64, len(md))
        values     := make([]float64, len(md))
        pers       := make([]int, len(md))
        for i := range md {
            timestamps[i] = md[i].Timestamp
            values[i]     = md[i].Value
            pers[i]       = md[i].Per
        }

        enc.Encode(timestamps)
        enc.Encode(values)
        enc.Encode(pers)

    } else {

        enc.Encode("nil")

    }

    // Return
    gzw.Close()
    cmp = buf.Bytes()
    return

}

func DecompressMonitorData(cmp []byte) (md MonitorData, err error) {

    defer Catch(&err)

    rd       := bytes.NewReader(cmp)
    gzr, err := gzip.NewReader(bufio.NewReader(rd))
    Try(err)
    dec      := gob.NewDecoder(gzr)

    // Type
    var typ string
    dec.Decode(&typ)

    // Value
    switch typ {
    case "float64":

        timestamps := make([]int64, 0)
        values     := make([]float64, 0)
        pers       := make([]int, 0)

        // Timestamp
        dec.Decode(&timestamps)
        dec.Decode(&values)
        dec.Decode(&pers)
        
        // Assign
        md = make(MonitorData, len(timestamps))
        for i := 0; i < len(timestamps); i++ {
            md[i] = MonitorDatum{
                Timestamp: timestamps[i],
                Value:     values[i],
                Per:       pers[i],
            }
        }

    case "nil":

        // Return empty
        md = make(MonitorData, 0)

    }

    return

}

// Implementation of Largest-Triangle-Three-Buckets down-sampling algorithm
// https://github.com/dgryski/go-lttb
func LttbMonitorData(md MonitorData, threshold int) MonitorData {

    if threshold >= len(md) || threshold == 0 {
        return md
    }

    sampled := make(MonitorData, 0, threshold)

    // Bucket size. Leave room for start and end data points
    every := float64(len(md) - 2) / float64(threshold - 2)

    sampled = append(sampled, md[0]) // Always add the first point

    bucketStart := 1
    bucketCenter := int(math.Floor(every)) + 1

    var a int

    for i := 0; i < threshold - 2; i++ {

        bucketEnd := int(math.Floor(float64(i + 2) * every)) + 1

        // Calculate point average for next bucket (containing c)
        avgRangeStart := bucketCenter
        avgRangeEnd := bucketEnd

        if avgRangeEnd >= len(md) {
            avgRangeEnd = len(md)
        }

        avgRangeLength := float64(avgRangeEnd - avgRangeStart)

        var avgX, avgY float64
        for ; avgRangeStart < avgRangeEnd; avgRangeStart++ {
            avgX += md[avgRangeStart].X()
            avgY += md[avgRangeStart].Y()
        }
        avgX /= avgRangeLength
        avgY /= avgRangeLength

        // Get the range for this bucket
        rangeOffs := bucketStart
        rangeTo := bucketCenter

        // Point a
        pointAX := md[a].X()
        pointAY := md[a].Y()

        maxArea := -1.0

        var nextA int
        for ; rangeOffs < rangeTo; rangeOffs++ {
            // Calculate triangle area over three buckets
            area := (pointAX - avgX) * (md[rangeOffs].Y() - pointAY) -
                (pointAX - md[rangeOffs].X()) * (avgY - pointAY)
            // We only care about the relative area here.
            // Calling math.Abs() is slower than squaring
            area *= area
            if area > maxArea {
                maxArea = area
                nextA = rangeOffs // Next a is this b
            }
        }

        sampled = append(sampled, md[nextA]) // Pick this point from the bucket
        a = nextA                               // This a is the next a (chosen b)

        bucketStart = bucketCenter
        bucketCenter = bucketEnd
    }

    sampled = append(sampled, md[len(md) - 1]) // Always add last

    return sampled
}