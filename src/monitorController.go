package main

import (
    "bufio"
    "bytes"
    "compress/gzip"
    "encoding/gob"
    "math"
    "strconv"
    "strings"
    "./monitor"

    . "github.com/hjjg200/go-act"
)

const (
    MonitorStatusNormal = 8 * iota
    MonitorStatusWarning
    MonitorStatusFatal
)

type Range string

type MonitorDataTableBox struct {
    Boundaries []byte
    DataMap map[string/* key */] []byte
}

type MonitorDataMap map[string/* key */] MonitorData
type MonitorData []MonitorDatum
type MonitorDatum struct {
    Timestamp int64
    Value float64
}

// sort.Interface
func (md MonitorData) Len() int { return len(md) }
func (md MonitorData) Less(i, j int) bool { return md[i].Timestamp < md[j].Timestamp }
func (md MonitorData) Swap(i, j int) { md[i], md[j] = md[j], md[i] }
// For LTTB
func (datum MonitorDatum) X() float64 { return float64(datum.Timestamp) }
func (datum MonitorDatum) Y() float64 { return datum.Value }

type MonitorConfig struct {
    FatalRange Range `json:"fatalRange"`
    WarningRange Range `json:"warningRange"`
    Format string `json:"format"`
}

func init() {
    parsedRanges = make(map[Range] func(float64) bool)

    // Monitor error callback
    monitor.ErrorCallback = func(err error) {
        Logger.Warnln("monitor:", err)
    }
}

//
// RANGES
//
var parsedRanges map[Range] func(float64) bool

func (r Range) Parse() {

    // Prepare Splits
    commaSplits := SplitComma(string(r))
    numSplits   := make([][]float64, len(commaSplits))
    for i := range commaSplits {

        isRanged := strings.Contains(commaSplits[i], ":")
        splits   := strings.Split(commaSplits[i], ":")

        if len(splits) == 1 && isRanged  {
            splits = append(splits, "")
        }

        numSplits[i] = make([]float64, len(splits))
        for j := range splits {
            var num float64
            var err error
            if splits[j] == "" {
                // Empty is the either end of number
                if j == 0 {
                    num = -math.MaxFloat64
                } else {
                    num = math.MaxFloat64
                }
            } else {
                // If not empty
                num, err = strconv.ParseFloat(splits[j], 64)
                if err != nil {
                    parsedRanges[r] = func(val float64) bool {
                        Logger.Warnln(r, "is a malformed range!")
                        return false
                    }
                    return
                }
            }
            numSplits[i][j] = num
        }
    }

    // Assign
    parsedRanges[r] = func(val float64) bool {
        //
        for _, split := range numSplits {
            if len(split) == 1 {
                if val == split[0] {
                    return true
                }
            } else {
                if val >= split[0] && val <= split[1] {
                    return true
                }
            }
        }
        return false
    }

}

func (r Range) Includes(val float64) bool {
    pr, ok := parsedRanges[r]
    if !ok || pr == nil {
        r.Parse()
        pr = parsedRanges[r]
    }
    return pr(val)
}

func (mCfg MonitorConfig) StatusOf(val float64) int {
    switch {
    case mCfg.FatalRange.Includes(val):
        return MonitorStatusFatal
    case mCfg.WarningRange.Includes(val):
        return MonitorStatusWarning
    }
    return MonitorStatusNormal
}

//
// Monitor Keys
//
func ParseMonitorrKey(key string) (base, param, idx string) {
    return monitor.ParseWrapperKey(key)
}

func FormatMonitorrKey(base, param, idx string) string {
    return monitor.FormatWrapperKey(base, param, idx)
}

//
// Compression
//
func CompressMonitorData(md MonitorData) (cmp []byte, err error) {

    defer Catch(&err)

    buf := bytes.NewBuffer(nil)
    gzw := gzip.NewWriter(buf)
    enc := gob.NewEncoder(gzw)

    // | type | timestamps | values | 

    if len(md) > 0 {
        enc.Encode("float64")
        timestamps := make([]int64, len(md))
        values     := make([]float64, len(md))
        for i := range md {
            timestamps[i] = md[i].Timestamp
            values[i] = md[i].Value
        }
        enc.Encode(timestamps)
        enc.Encode(values)
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

    //
    var typ string
    timestamps := make([]int64, 0)
    put        := func(slice []float64) {
        md = make(MonitorData, len(slice))
        for i := 0; i < len(slice); i++ {
            md[i] = MonitorDatum{
                Timestamp: timestamps[i],
                Value: slice[i],
            }
        }
    }
    // Type
    dec.Decode(&typ)
    // Timestamp
    dec.Decode(&timestamps)
    // Value
    switch typ {
    case "float64":
        values := make([]float64, 0)
        dec.Decode(&values)
        put(values)
    case "nil":
        // Return empty
        md = make(MonitorData, 0)
        return
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