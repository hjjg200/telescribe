package main

import (
    "bufio"
    "bytes"
    "compress/gzip"
    "encoding/gob"
    "fmt"
    "math"
    "strconv"
    "strings"
    "regexp"
    "./monitor"
)

type Range string

/*
MonitorInfo
- Ranges
MonitorDataSlice
- Count
- Elapsed
- Last
- Slice

*/

type GrpahDataComposite struct {
    GapThresholdTime int `json:"options.gapThresholdTime"`
    GapPercent float64 `json:"options.gapPercent"`
    MomentJSFormat string `json:"options.momentJsFormat"`
    ClientAliases map[string] string `json:"clientAliases"`
    ClientMonitorData map[string] map[string] MonitorDataSlice `json:"clientMonitorData"`
}

type MonitorDataSlice []MonitorDataSliceElem
type MonitorDataSliceElem struct {
    Timestamp int64
    Value float64
}

type MonitorStatusSlice []MonitorStatusSliceElem
type MonitorStatusSliceElem struct {
    Status int
    Timestamp int64
    Value float64
}

// sort.Interface
func (mds MonitorDataSlice) Len() int {
    return len(mds)
}

func (mds MonitorDataSlice) Less(i, j int) bool {
    return mds[i].Timestamp < mds[j].Timestamp
}

func (mds MonitorDataSlice) Swap(i, j int) {
    mds[i], mds[j] = mds[j], mds[i]
}

//
func (mdse MonitorDataSliceElem) X() float64 {
    return float64(mdse.Timestamp)
}

func (mdse MonitorDataSliceElem) Y() float64 {
    return mdse.Value
}

type MonitorInfo struct {
    FatalRange Range `json:"fatalRange"`
    WarningRange Range `json:"warningRange"`
}

const (
    MonitorStatusNormal = 8 * iota
    MonitorStatusWarning
    MonitorStatusFatal
)

var parsedRanges map[Range] func(float64) bool
var commaSplitRegexp = regexp.MustCompile("\\s*,\\s*")
func SplitComma(str string) []string {
    return commaSplitRegexp.Split(str, -1)
}

func init() {
    parsedRanges = make(map[Range] func(float64) bool)
}

func ParseMonitorrKey(key string) (base, param, idx string) {
    return monitor.ParseWrapperKey(key)
}

func FormatMonitorrKey(base, param, idx string) string {
    return monitor.FormatWrapperKey(base, param, idx)
}

func (r Range) Parse() {

    // Prepare Splits
    commaSplits := SplitComma(string(r))
    numSplits := make([][]float64, len(commaSplits))
    for i := range commaSplits {

        isRanged := strings.Contains(commaSplits[i], ":")
        splits := strings.Split(commaSplits[i], ":")

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

func (mi MonitorInfo) StatusOf(val float64) int {
    switch {
    case mi.FatalRange.Includes(val):
        return MonitorStatusFatal
    case mi.WarningRange.Includes(val):
        return MonitorStatusWarning
    }
    return MonitorStatusNormal
}

func CompressMonitorDataSlice(mds MonitorDataSlice) (cmp []byte, err error) {

    defer func() {
        r := recover()
        if r != nil {
            err = fmt.Errorf("%v", r)
        }
    }()

    buf := bytes.NewBuffer(nil)
    gzw := gzip.NewWriter(buf)
    enc := gob.NewEncoder(gzw)

    // | type | timestamps | values | 

    if len(mds) > 0 {
        enc.Encode("float64")
        timestamps := make([]int64, len(mds))
        slice := make([]float64, len(mds))
        for i := range mds {
            slice[i] = mds[i].Value
            timestamps[i] = mds[i].Timestamp
        }
        enc.Encode(timestamps)
        enc.Encode(slice)
    } else {
        enc.Encode("nil")
    }

    // Return
    gzw.Close()
    cmp = buf.Bytes()
    return

}

func DecompressMonitorDataSlice(b []byte) (mds MonitorDataSlice, err error) {

    defer func() {
        r := recover()
        if r != nil {
            err = fmt.Errorf("%v", r)
        }
    }()

    rd := bytes.NewReader(b)
    gzr, err := gzip.NewReader(bufio.NewReader(rd))
    if err != nil {
        return
    }
    dec := gob.NewDecoder(gzr)

    //
    var t string
    timestamps := make([]int64, 0)
    put := func(slice []float64) {
        mds = make([]MonitorDataSliceElem, len(slice))
        for i := 0; i < len(slice); i++ {
            mds[i] = MonitorDataSliceElem{
                Timestamp: timestamps[i],
                Value: slice[i],
            }
        }
    }
    // Type
    dec.Decode(&t)
    // Timestamp
    dec.Decode(&timestamps)
    // Value
    switch t {
    case "float64":
        slice := make([]float64, 0)
        dec.Decode(&slice)
        put(slice)
    case "nil":
        return
    }

    return

}

// Implementation of Largest-Triangle-Three-Buckets down-sampling algorithm
// https://github.com/dgryski/go-lttb
func LTTBMonitorDataSlice(mds MonitorDataSlice, threshold int) MonitorDataSlice {

    if threshold >= len(mds) || threshold == 0 {
        return mds
    }

    sampled := make(MonitorDataSlice, 0, threshold)

    // Bucket size. Leave room for start and end data points
    every := float64(len(mds) - 2) / float64(threshold - 2)

    sampled = append(sampled, mds[0]) // Always add the first point

    bucketStart := 1
    bucketCenter := int(math.Floor(every)) + 1

    var a int

    for i := 0; i < threshold - 2; i++ {

        bucketEnd := int(math.Floor(float64(i + 2) * every)) + 1

        // Calculate point average for next bucket (containing c)
        avgRangeStart := bucketCenter
        avgRangeEnd := bucketEnd

        if avgRangeEnd >= len(mds) {
            avgRangeEnd = len(mds)
        }

        avgRangeLength := float64(avgRangeEnd - avgRangeStart)

        var avgX, avgY float64
        for ; avgRangeStart < avgRangeEnd; avgRangeStart++ {
            avgX += mds[avgRangeStart].X()
            avgY += mds[avgRangeStart].Y()
        }
        avgX /= avgRangeLength
        avgY /= avgRangeLength

        // Get the range for this bucket
        rangeOffs := bucketStart
        rangeTo := bucketCenter

        // Point a
        pointAX := mds[a].X()
        pointAY := mds[a].Y()

        maxArea := -1.0

        var nextA int
        for ; rangeOffs < rangeTo; rangeOffs++ {
            // Calculate triangle area over three buckets
            area := (pointAX - avgX) * (mds[rangeOffs].Y() - pointAY) -
                (pointAX - mds[rangeOffs].X()) * (avgY - pointAY)
            // We only care about the relative area here.
            // Calling math.Abs() is slower than squaring
            area *= area
            if area > maxArea {
                maxArea = area
                nextA = rangeOffs // Next a is this b
            }
        }

        sampled = append(sampled, mds[nextA]) // Pick this point from the bucket
        a = nextA                               // This a is the next a (chosen b)

        bucketStart = bucketCenter
        bucketCenter = bucketEnd
    }

    sampled = append(sampled, mds[len(mds) - 1]) // Always add last

    return sampled
}