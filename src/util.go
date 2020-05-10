package main

import (
    "math"
    "strconv"
    "strings"
)

//
// RANGE
//
type Range string
var parsedRanges = make(map[Range] func(float64) bool)

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
                        EventLogger.Warnln(r, "is a malformed range!")
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
