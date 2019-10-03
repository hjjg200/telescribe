
let chartList = {};
let chartLabelsMap = {};
let chartSeriesMap = {};
let chartActiveSeriesMap = {};
let g_gdc;
let g_cms;
let app;

// Put average values instead of null

document.addEventListener("DOMContentLoaded", async function() {

    await fetchAndUpdate()

    app = new Vue({
        el: "#app",
        data: {
            checked: {}
        },
        computed: {
            gdc: function() { return g_gdc },
            cmdMap: function() { return this.gdc.clientMonitorData },
            caMap: function() { return this.gdc.clientAliases },
            cmsMap: function() { return g_cms },
            options: function() {
                let gdc = this.gdc;
                let ret = {};
                for(let key in gdc) {
                    if(key.startsWith("options.")) {
                        let split = key.split(".");
                        ret[split[1]] = gdc[key];
                    }
                }
                return ret;
            }
        },
        created: function() {
            for(let host in this.caMap) {
                this.checked[host] = {};
            }
        },
        mounted: function() {
            let options = this.options
            for(let host in this.cmdMap) {
                let [labels, seriesMap] = processClientMonitorData(
                    this.cmdMap[host], this.cmsMap[host], options
                )
                chartLabelsMap[host] = labels;
                chartSeriesMap[host] = seriesMap;
                chartActiveSeriesMap[host] = [];
                this.drawChart(host, labels);
            }
        },
        methods: {
            toggleSeries: function(ev, host, key, toggle) {
                if(toggle) {
                    chartActiveSeriesMap[host].push({
                        key: key,
                        series: chartSeriesMap[host][key]
                    });
                    let i = getSeriesIdx(chartActiveSeriesMap[host].length);
                } else {
                    let tmp = [];
                    chartActiveSeriesMap[host].forEach(function(val) {
                        if(val.key == key) {
                            on = true;
                            return;
                        }
                        tmp.push(val);
                    });
                    chartActiveSeriesMap[host] = tmp;
                    ev.target.className = "";
                }
                this.updateChart(host);
            },
            updateChart: function(host) {
                let series = [];
                let i = 1;
                chartActiveSeriesMap[host].forEach(function(val) {
                    let q = formatCheckboxQuery(host, val.key);
                    let a = getSeriesIdx(i++);
                    let cb = document.querySelector(q);
                    cb.className = "";
                    cb.classList.add("series-" + a);
                    series.push(val.series);
                });
                chartList[host].data.series = series;
                chartList[host].update();
            },
            maxStatus: function(cms) {
                let max = -1;
                for(let key in cms) {
                    let st = cms[key].Status;
                    max = st > max ? st : max;
                }
                return max;
            },
            drawChart: function(host, labels) {
                let query = formatChartQuery(host);
                let labelLength = labels.length;
                let labelStep = Math.floor(labelLength / 7);
                let gtht = this.options.gapThresholdTime * 60;
                chartList[host] = new Chartist.Line(
                    query, {
                        series: [],
                        labels: labels
                    }, {
                        showPoint: true,
                        fullWidth: true,
                        lineSmooth: false,
                        axisX: {
                            showGrid: false,
                            labelInterpolationFnc: (v, idx) => {
                                if(idx % labelStep == 0 && v != null) {
                                    return getHours(v);
                                }
                                return null;
                            }
                        },
                        axisY: {
                            labelInterpolationFnc: (v, idx) => {
                                if(v >= 1e+9 * 0.8) {
                                    return (Math.round(v / 1e+9 * 10) / 10) + "B";
                                } else if(v >= 1e+6 * 0.8) {
                                    return (Math.round(v / 1e+6 * 10) / 10) + "M";
                                } else if(v >= 1e+3 * 0.8) {
                                    return (Math.round(v / 1e+3 * 10) / 10) + "K";
                                } else {
                                    return v;
                                }
                            }
                        },
                        plugins: [
                            Chartist.plugins.tooltip()
                        ]
                    }
                )
            }
        }
    })

    //
    document.querySelectorAll(".chart").forEach(function(el) {
        el.addEventListener("mouseover", function(ev) {
            if(hasClass(ev.target, "ct-point")) {
                let series = ev.target.parentElement.getAttribute('class').match(/ct-series-\w/)[0];
                let g = document.querySelector("g." + series);
                g.classList.add("active");
            }
        })
        el.addEventListener("mouseout", function(ev) {
            if(hasClass(ev.target, "ct-point")) {
                let series = ev.target.parentElement.getAttribute('class').match(/ct-series-\w/)[0];
                let g = document.querySelector("g." + series);
                g.classList.remove("active");
            }
        })
    })

})


async function fetchAndUpdate() {
    
    let gdcResponse = await fetch(
        "graphDataComposite.json", {
        method: "GET",
        cache: "no-cache"
    });

    let cmsResponse = await fetch(
        "clientMonitorStatus.json", {
        method: "GET",
        cache: "no-cache"
    });

    g_gdc = await gdcResponse.json();
    g_cms = await cmsResponse.json();

}

function getSeriesIdx(i) {
    return "abcdefghijklmno".charAt(i - 1);
}

function formatChartQuery(host) {
    host = host.replace(/"/g, '\\\"');
    return `#host-list li[data-host="${host}"] .chart`;
}

function formatCheckboxQuery(host, key) {
    host = host.replace(/"/g, '\\\"');
    key = key.replace(/"/g, '\\\"');
    return `#host-list li[data-host="${host}"] li[data-key="${key}"] input`;
}

function hasClass(element, className) {
    return (' ' + element.getAttribute('class') + ' ').indexOf(' ' + className + ' ') > -1;
}

function processClientMonitorData(cmd, cms, options) {

    let gtht = options.gapThresholdTime * 60;
    let xAxis = [];
    let seriesMap = {};
    { // Get all timestamps
        let tmp = {};
        for(let key in cmd) {
            let slice = cmd[key];
            // last one
            if(slice.length == 0 || (slice.length > 0 && slice[slice.length - 1].Timestamp != cms.Timestamp)) {
                slice.push({
                    Timestamp: cms.Timestamp,
                    Value: cms.Value
                });
            }

            for(let idx in slice) {
                let each = slice[idx];
                tmp[each.Timestamp] = null;
            }
        }
        for(let t in tmp) {
            xAxis.push(t);
        }
        xAxis.sort();
    }

    // Put gaps
    let tmp = [];
    let glen = Math.round(xAxis.length * options.gapPercent / 100) + 1;
    let labelStep = Math.floor(xAxis.length / 5);
    let gapAdded = false;
    for(let i = 0; i < xAxis.length; i++) {
        let ts = xAxis[i];
        if(i < xAxis.length - 1 && xAxis[i + 1] - ts > gtht) {// Gap
            gapAdded = true;
            for(let j = 0; j < glen; j++) {
                tmp.push(null);
            }
        } else if(i % labelStep == 0 || gapAdded) {
            gapAdded = false;
            tmp.push(ts);
        } else {
            tmp.push(ts);
        }
    }
    xAxis = tmp;

    // Series
    for(let key in cmd) {
        let slice = cmd[key];
        let tmp = {};
        let series = [];

        for(let idx in slice) {
            let each = slice[idx];
            tmp[each.Timestamp] = {
                value: each.Value,
                meta: moment.unix(each.Timestamp).format(options.momentJsFormat)
            };
        }

        for(let i = 0; i < xAxis.length; i++) {

            let aX = xAxis[i];
            let bX = (i < xAxis.length - 1) ? xAxis[i + 1] : null;

            if(aX != null
            && bX != null
            && tmp[aX] != null) {
                
                let aI = tmp[aX];
                series.push({
                    x: aX,
                    y: aI.value,
                    meta: aI.meta
                });

                if(tmp[bX] == null) {
                    let wta = whatToAppend(xAxis.slice(i + 1), aI, tmp, options);
                    wta.forEach(function(item) {
                        series.push(item);
                    });
                    i += wta.length;
                }

            } else if(aX != null && tmp[aX] != null) {
                let aI = tmp[aX];
                series.push({
                    x: aX,
                    y: aI.value,
                    meta: aI.meta
                });
            } else {
                series.push({
                    x: null,
                    y: null
                });
            }

        }
        
        seriesMap[key] = series;
    }

    return [xAxis, seriesMap];
    
}

function whatToAppend(axis, start, tmp, options) {

    let startVal = start.value;
    let count = 0;
    let ret = [];
    let endVal = start.value;

    for(let i = 0; i < axis.length; i++) {
        count++;

        let x = axis[i];
        if(x == null) break;

        let item = tmp[x];
        if(item != null) {
            endVal = item.value;
            count -= 1; // Don't add this point
            break;
        }
    }

    let step = (endVal - startVal) / count;
    for(let i = 0; i < count; i++) {
        let x = axis[i];
        if(x == null) {
            ret.push({
                x: null,
                y: null
            });
            break;
        }
        ret.push({
            x: x,
            y: startVal + step * (i + 1),
            meta: ""
        });
    }

    return ret;

}

function getHours(t) {
    let now = (new Date()).getTime() / 1000;
    let diff = now - t;
    if(diff < 10800) {
        return Math.floor(diff / 60) + "m";
    } else {
        return Math.floor(diff / 3600) + "h";
    }
}

function isHidden(elem) {
    var style = window.getComputedStyle(elem);
    return (style.display === 'none');
}

function chartSplitIntoSegments (pathCoordinates, valueData, options) {
    var defaultOptions = {
        increasingX: false,
        fillHoles: false,
        gapThresholdTime: undefined
    };

    options = Chartist.extend({}, defaultOptions, options);

    var segments = [];
    var hole = true;
    var gtht = options.gapThresholdTime;
    let lastPointX = undefined;

    for(var i = 0; i < pathCoordinates.length; i += 2) {

        let val = valueData[i / 2].value;

        // If this value is a "hole" we set the hole flag
        if(Chartist.getMultiValue(valueData[i / 2].value) === undefined) {
            // if(valueData[i / 2].value === undefined) {
            if(!options.fillHoles) {
                hole = true;
            }
        } else {
            if(options.increasingX && i >= 2 && pathCoordinates[i] <= pathCoordinates[i-2]) {
                // X is not increasing, so we need to make sure we start a new segment
                hole = true;
            }

            // If it's a valid value we need to check if we're coming out of a hole and create a new empty segment
            if(gtht != undefined && lastPointX != undefined) {
                if(val.x - lastPointX > gtht) {
                    //hole = true;
                }
            }

            if(hole) {
                segments.push({
                pathCoordinates: [],
                valueData: []
                });
                // As we have a valid value now, we are not in a "hole" anymore
                hole = false;
            }

            // Add to the segment pathCoordinates and valueData
            segments[segments.length - 1].pathCoordinates.push(pathCoordinates[i], pathCoordinates[i + 1]);
            segments[segments.length - 1].valueData.push(valueData[i / 2]);

        }

        if(val.y != undefined) {
            lastPointX = val.x;
        }

    }

    return segments;
}

function chartInterpolationCardinal (options) {
    var defaultOptions = {
      tension: 0, //1,
      fillHoles: false,
      gapThresholdTime: undefined
    };

    options = Chartist.extend({}, defaultOptions, options);

    var t = Math.min(1, Math.max(0, options.tension)),
      c = 1 - t;

    return function cardinal(pathCoordinates, valueData) {
        // First we try to split the coordinates into segments
        // This is necessary to treat "holes" in line charts
        var segments = chartSplitIntoSegments(pathCoordinates, valueData, {
            fillHoles: options.fillHoles,
            gapThresholdTime: options.gapThresholdTime
        });

        if(!segments.length) {
            // If there were no segments return 'Chartist.Interpolation.none'
            return Chartist.Interpolation.none()([]);
        } else if(segments.length > 1) {
            // If the split resulted in more that one segment we need to interpolate each segment individually and join them
            // afterwards together into a single path.
            var paths = [];
            // For each segment we will recurse the cardinal function
            segments.forEach(function(segment) {
            paths.push(cardinal(segment.pathCoordinates, segment.valueData));
            });
            // Join the segment path data into a single path and return
            return Chartist.Svg.Path.join(paths);
        } else {
            // If there was only one segment we can proceed regularly by using pathCoordinates and valueData from the first
            // segment
            pathCoordinates = segments[0].pathCoordinates;
            valueData = segments[0].valueData;

            // If less than two points we need to fallback to no smoothing
            if(pathCoordinates.length <= 4) {
            return Chartist.Interpolation.none()(pathCoordinates, valueData);
            }

            var path = new Chartist.Svg.Path().move(pathCoordinates[0], pathCoordinates[1], false, valueData[0]),
            z;

            for (var i = 0, iLen = pathCoordinates.length; iLen - 2 * !z > i; i += 2) {
            var p = [
                {x: +pathCoordinates[i - 2], y: +pathCoordinates[i - 1]},
                {x: +pathCoordinates[i], y: +pathCoordinates[i + 1]},
                {x: +pathCoordinates[i + 2], y: +pathCoordinates[i + 3]},
                {x: +pathCoordinates[i + 4], y: +pathCoordinates[i + 5]}
            ];
            if (z) {
                if (!i) {
                p[0] = {x: +pathCoordinates[iLen - 2], y: +pathCoordinates[iLen - 1]};
                } else if (iLen - 4 === i) {
                p[3] = {x: +pathCoordinates[0], y: +pathCoordinates[1]};
                } else if (iLen - 2 === i) {
                p[2] = {x: +pathCoordinates[0], y: +pathCoordinates[1]};
                p[3] = {x: +pathCoordinates[2], y: +pathCoordinates[3]};
                }
            } else {
                if (iLen - 4 === i) {
                p[3] = p[2];
                } else if (!i) {
                p[0] = {x: +pathCoordinates[i], y: +pathCoordinates[i + 1]};
                }
            }

            path.curve(
                (t * (-p[0].x + 6 * p[1].x + p[2].x) / 6) + (c * p[2].x),
                (t * (-p[0].y + 6 * p[1].y + p[2].y) / 6) + (c * p[2].y),
                (t * (p[1].x + 6 * p[2].x - p[3].x) / 6) + (c * p[2].x),
                (t * (p[1].y + 6 * p[2].y - p[3].y) / 6) + (c * p[2].y),
                p[2].x,
                p[2].y,
                false,
                valueData[(i + 2) / 2]
            );
            }

            return path;
        }
    }
}