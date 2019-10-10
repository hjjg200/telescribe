
let chartList = {};
let chartDataset = {};
let chartActiveKeys = {};
let chartXScale = {};
let chartInfo = {};

let g_gdc;
let g_cms;
let app;

// Put average values instead of null

/*

    For bisecting, data should be like
    map[timestamp] [
        {
            key, value
        }
    ]

*/

document.addEventListener("DOMContentLoaded", async function() {

    await fetchAndUpdate();

    app = new Vue({
        el: "#app",
        data: {
            checked: {}
        },
        computed: {
            gdc: function() { return g_gdc },
            cmdMap: function() { return this.gdc.clientMonitorData },
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
            for(let fullName in this.cmsMap) {
                this.checked[fullName] = {};
            }
        },
        mounted: function() {
            let options = this.options;
            for(let fullName in this.cmdMap) {
                let [xScale, dataset] = processChartDataset(
                    this.cmdMap[fullName], this.cmsMap[fullName], options
                );
                chartXScale[fullName] = xScale;
                chartDataset[fullName] = dataset;
                chartActiveKeys[fullName] = [];
                drawChartV2(fullName, {
                    duration: 86400
                });
            }
        },
        methods: {
            toggleSeries: function(ev, fullName, key, toggle) {
                if(toggle) {
                    chartActiveKeys[fullName].push(key);
                } else {
                    let swap = [];
                    chartActiveKeys[fullName].forEach(function(key2) {
                        if(key == key2) return;
                        swap.push(key2);
                    });
                    chartActiveKeys[fullName] = swap;
                    ev.target.className = "";
                }
                this.updateChart(fullName);
            },
            updateChart: function(fullName) {
                let i = 1;
                chartActiveKeys[fullName].forEach(function(key) {
                    let q = formatCheckboxQuery(fullName, key);
                    let a = getSeriesIdx(i++);
                    let cb = document.querySelector(q);
                    cb.className = "";
                    cb.classList.add("series-" + a);
                });
                updateChartSegments(fullName);
            },
            maxStatus: function(cms) {
                let max = -1;
                for(let key in cms) {
                    let st = cms[key].Status;
                    max = st > max ? st : max;
                }
                return max;
            },
            setDuration: function(fullName, duration) {
                drawChartV2(fullName, { duration: duration });
            },
            shortDate: function(t) {
                if(t <= 24 * 3600) {
                    return Math.round(t / 3600) + "h";
                } else {
                    return Math.round(t / 86400) + "d";
                }
            }
        }
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

function formatChartQuery(fullName) {
    fullName = fullName.replace(/"/g, '\\\"');
    return `#host-list li[data-host="${fullName}"] .chart`;
}

function formatCheckboxQuery(fullName, key) {
    fullName = fullName.replace(/"/g, '\\\"');
    key = key.replace(/"/g, '\\\"');
    return `#host-list li[data-host="${fullName}"] li[data-key="${key}"] input`;
}

function hasClass(element, className) {
    return (' ' + element.getAttribute('class') + ' ').indexOf(' ' + className + ' ') > -1;
}

function queryParent(elem, query) {
    while(true) {
        var elem = elem.parentNode;
        if(elem === document) {
            return undefined;
        }
        var parentElem = elem.querySelector(query);
        if(parentElem != undefined) {
            return parentElem;
        }
    }
    return undefined;
}

function queryParentAll(elem, query) {
    while(true) {
        var elem = elem.parentNode;
        if(elem === document) {
            return [];
        }
        var parentElems = elem.querySelectorAll(query);
        if(parentElems.length > 0) {
            return parentElems;
        }
    }
    return [];
}

function processChartDataset(cmd, cms, options) {

    let gtht = options.gapThresholdTime * 60;
    { // Always add the current node
        for(let key in cmd) {
            let len = cmd[key].length;
            let st = cms[key];
            let last = cmd[key][len - 1];
            if(len == 0
            || (last != null && last.Timestamp != st.Timestamp)) {
                cmd[key].push({
                    Timestamp: st.Timestamp,
                    Value: st.Value
                });
            }
        }
    }

    let dataMap = {};
    let dataset = [];
    { // Process Dataset
        for(let key in cmd) {
            let slice = cmd[key];
            for(let i = 0; i < slice.length; i++) {
                let prevItem = slice[i-1];
                let item = slice[i];

                // Gap
                if(prevItem != null) {
                    let diff = item.Timestamp - prevItem.Timestamp;
                    if(diff > gtht) {
                        // Put NaN
                        let avgTimestamp = prevItem.Timestamp + diff / 2;
                        let avgMap = dataMap[avgTimestamp];
                        if(avgMap == null) {
                            dataMap[avgTimestamp] = {
                                [key]: NaN
                            };
                        } else {
                            dataMap[avgTimestamp][key] = NaN;
                        }
                    }
                }

                let currMap = dataMap[item.Timestamp];
                if(currMap == null) {
                    dataMap[item.Timestamp] = {
                        [key]: item.Value // [] is needed to use key as variable rather than "key"
                    };
                } else {
                    dataMap[item.Timestamp][key] = item.Value;
                }

            }
        }

        for(let timestamp in dataMap) {
            let item = {};
            let curr = dataMap[timestamp];
            Object.keys(curr).forEach(function(key) {
                item[key] = curr[key];
            });
            item.timestamp = Number(timestamp);
            dataset.push(item);
        }
    }

    // Sort
    dataset.sort(function(a, b) {
        if(a.timestamp > b.timestamp) return 1;
        if(a.timestamp < b.timestamp) return -1;
        return 0;
    });

    // Gaps
    let xScale, xBaseScale;
    { // Use the gap bounaries to create scale
        let firstT = dataset[0].timestamp;
        let lastT = dataset[dataset.length - 1].timestamp;
        let duration = lastT - firstT;
        let boundaries = [];
        for(let i = 1; i < dataset.length; i++) {
            let prev = dataset[i-1];
            let curr = dataset[i];

            if(curr.timestamp - prev.timestamp > gtht) {
                // Exclude Gap Duration from duration
                duration -= curr.timestamp - prev.timestamp;
                // Boundary
                boundaries.push(
                    prev.timestamp, curr.timestamp
                );
            }
        }
        // Wrap
        boundaries.unshift(firstT);
        boundaries.push(lastT);

        let segNo = boundaries.length / 2;
        let gapNo = segNo - 1;
        let gapEachDuration = duration / (30 + gapNo);
        let gapBoundaries = boundaries.slice(1, -1);
        // | duration | gap duration... |
        let totalDuration = gapEachDuration * gapNo + duration;
        let steps = [];
        let lefts = [];
        let lastRight = 0;
        let lastRightT = firstT;
        //
        for(let i = 0; i < gapNo; i++) {
            let gapStart = gapBoundaries[i*2];
            let gapEnd = gapBoundaries[i*2+1];
            //
            let step = gapEachDuration / (gapEnd - gapStart);
            let left = lastRight + (gapStart - lastRightT) / totalDuration;
            let right = left + gapEachDuration / totalDuration;
            lastRight = right;
            lastRightT = gapEnd;
            //
            steps.push(step, 1); // this step and 1
            lefts.push(left, right);
        }
        // Wrap
        lefts.unshift(0);
        lefts.push(1); // Rightmost
        steps.unshift(1);

        // BaseScale returns 0 to 1
        xBaseScale = function(timestamp) {
            for(let i = 0; i < boundaries.length - 1; i++) {
                let step = steps[i];
                let left = lefts[i];
                let leftT = boundaries[i];
                let rightT = boundaries[i+1];
                if(timestamp <= rightT) {
                    return (timestamp - leftT) * step / totalDuration + left;
                }
            }
            // Beyond domain
            return 1 + (timestamp - lastT) / totalDuration;
        }
        // xDomain
        xScale = function(timestamp) {
            return xBaseScale(timestamp) * (xScale._range[1] - xScale._range[0]) + xScale._range[0];
        };
        xScale._ = function(timestamp) {
            return xBaseScale(timestamp);
        };
        xScale.domain = function() {
            return [firstT, lastT];
        };
        xScale.range = function(range) {
            if(range === undefined) return xScale._range;
            xScale._range = range;
            return this;
        };
        xScale.invert = function(x) {
            // Convert to 0 to 1
            let base = (x - xScale._range[0]) / (xScale._range[1] - xScale._range[0]);
            for(let i = 0; i < lefts.length - 1; i++) {
                let step = steps[i];
                let left = lefts[i];
                let right = lefts[i + 1];
                let leftTimestamp = boundaries[i];
                if(base <= right) {
                    return (base - left) / step * totalDuration + leftTimestamp;
                }
            }
            // Beyond domain
            return (x - 1) * totalDuration + lastT;
        };

    }

    return [xScale, dataset];
    
}

function isHidden(elem) {
    var style = window.getComputedStyle(elem);
    return (style.display === 'none');
}

Number.prototype.date = function(str = "DD HH:mm") {
    return moment.unix(this).format(str);
}

Number.prototype.format = function(str) {
    var fmt = parseNumberFormat(str);
    var num = this;
    num = num * Math.pow(10, fmt.exp);
    // Precision
    if(!isNaN(fmt.precision)) {
        var precision = Math.pow(10, fmt.precision);
        num = Math.round(num * precision) / precision;
    }
    return `${fmt.prefix}${formatComma(num)}${fmt.suffix}`;
};

function parseNumberFormat(str) {
    var rgx = /(.+)?(\{(e([+-]?\d+))?(\.(\d+))?f?\})(.+)?/;
    var m = str.match(rgx);
    var [prefix = "", exp = 0, precision = NaN, suffix = ""] = [m[1], m[4], m[6], m[7]];
    return {
        prefix, exp: parseInt(exp), precision: parseInt(precision), suffix
    };
}

function formatComma(x) {
    var parts = x.toString().split(".");
    parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
    return parts.join(".");
}

// Timeframe
/*
min boundary = max(mintimestamp, mindate)
max boundary = min(maxtimestamp, maxdate)

if min > max or max < min
    if min changed
        min = max - 1 day
    if max changed
        max = min + 1 day
    
hide input show label = MM/DD

Option 1 {clock} MM/DD - MM/DD
Option 2 {clock} MM/DD + 1d 3d 7d 14d 30d All
*/

//
// D3.js
//

/*
chartDuration = 1d
chartStartTimestamp = ...

segNo = ceil(entire time / duration)

entire time
for i 0 i < segNo i++ 
    ent -= duration
    width = left / duration
    left = width * i

    seg attr.start = time
        attr.end = time
    
    axis is drawn entirely at draw

    seg is not drawn
    left < scrollleft and scrollleft - left < width
    scrollleft < left and left - scrollleft < width * 2
        draw chart

    call updateSegs when all is done

    make in such a way that only the duration change requires full redraw

*/

function arraysEqual(a, b) {
    if(a === b) return true;
    if(a == null || b == null) return false;
    if(a.length != b.length) return false;
  
    a = a.slice(0).sort();
    b = b.slice(0).sort();
  
    for (var i = 0; i < a.length; ++i) {
        if(a[i] !== b[i]) return false;
    }
    return true;
}

function updateChartSegments(fullName) {

    // Chart
    var chartQuery = formatChartQuery(fullName);
    var chart = d3.select(chartQuery);
    var entireDataset = chartDataset[fullName];
    var activeKeys = chartActiveKeys[fullName];

    var info = chartInfo[fullName];
    var priorActiveKeys = info.priorActiveKeys;
    var xDuration = info.xDuration;
    var xBoundary = info.xBoundary;
    var chartWidth = info.width;
    var chartHeight = info.height;
    var chartMargin = info.margin;
    var dataWidth = info.dataWidth;
    var dataHeight = info.dataHeight;
    var chartDuration = info.chartDuration;
    var yTicks = info.yTicks;
    // ELements
    var xScale = info.xScale;
    var projection = info.projection;

    // Check Changes
    var keysChanged = !arraysEqual(activeKeys, priorActiveKeys);

    // Info Set
    info.priorActiveKeys = activeKeys.slice(0);

    // Segments
    var segmentsWrap = chart.select(".segments-wrap");
    var scrollLeft = segmentsWrap.node().scrollLeft;

    // Dataset
    var seriesName = d3.scaleOrdinal()
        .domain(activeKeys)
        .range(["a","b","c","d","e","f","g","h","i","j","k","l","m","n","o"].map(function(a) {
            return "series-" + a;
        }));
    info.seriesName = seriesName;
        /*
        scrollLeftTimestamp = xDuration * scrollWidth / DataWidth

        boundary min = max(scrollts - chartduration, min boundary)
        max boundary = min(srollts + charduration * 2, max boundary)
        */
    var scrollLeftTimestamp = xScale.invert(scrollLeft); // scrollLefTime
    // 
    var visibleBoundary = [
        Math.max(xScale.invert(scrollLeft - chartWidth), xBoundary[0]),
        Math.min(xScale.invert(scrollLeft + chartWidth * 2), xBoundary[1])
    ];

    var dataset = [];
    entireDataset.forEach(function(each) {
        var filtered = {};
        var add = false;
        activeKeys.forEach(function(key) {
            if(each[key] == undefined) return;
            filtered[key] = each[key];
            add = true;
        });
        if(!add) return;
        filtered.timestamp = each.timestamp;
        dataset.push(filtered);
    });
    info.dataset = dataset;
    var yBoundary = d3.extent(function() {
        var arr = [];
        dataset.forEach(function(each) {
            for(let key in each) {
                if(key === "timestamp") continue;
                arr.push(each[key]);
            }
        });
        return arr;
    }());
    if(yBoundary[0] == undefined) {
        yBoundary = [0, 1];
    } else if(yBoundary[0] == yBoundary[1]) {
        yBoundary[0] = 0;
    }

    // Visible Dataset
    var visibleDataset = dataset.filter(function(each) {
        if(visibleBoundary[0] <= each.timestamp && each.timestamp <= visibleBoundary[1]) {
            return true;
        }
        return false;
    });

    // Update Y Axis
    var yScale = d3.scaleLinear()
        .domain(yBoundary)
        .range([dataHeight, 0]);
    chartInfo[fullName].yScale = yScale;
        // Remove Y Axis
    chart.select(".y-axis-wrap").remove();
        // New
    var yAxis = chart.append("div")
        .attr("class", "axis-wrap y-axis-wrap")
        .append("div")
            .attr("class", "axis-container y-axis-container")
            .append("svg")
                .attr("height", chartHeight + chartMargin.top + chartMargin.bottom)
                .append("g")
                    .attr("class", "axis y-axis")
                    .attr("transform", `translate(${chartMargin.left}, ${chartMargin.top})`)
                    .call(d3.axisLeft(yScale)
                        .ticks(yTicks)
                        .tickSizeOuter(0)
                        .tickFormat(function(value) {
                            if(value >= 1e+9 * 0.8) {
                                return (Math.round(value / 1e+9 * 10) / 10) + "B";
                            } else if(value >= 1e+6 * 0.8) {
                                return (Math.round(value / 1e+6 * 10) / 10) + "M";
                            } else if(value >= 1e+3 * 0.8) {
                                return (Math.round(value / 1e+3 * 10) / 10) + "K";
                            }
                            return value;
                        }))
                        .attr("font-family", "")
                        .attr("font-size", "");

    // Update grid
    chart.select(".grid").remove();
    var grid = projection.append("g")
        .attr("class", "grid")
        .call(d3.axisLeft(yScale)
            .ticks(yTicks)
            .tickSize(-dataWidth)
            .tickFormat(""));

    // Segments Each
    // Loop
    var segments = segmentsWrap.select(".segments");
    var segmentNodes = segmentsWrap.selectAll(".segment").nodes().reverse();
    var visibleBoundaryPixels = visibleBoundary.map(function(timestamp) {
        return xScale(timestamp);
    });
    segmentNodes.forEach(function(node) {
        var seg = d3.select(node);
        var start = Number(node.getAttribute("data-start"));
        var end = Number(node.getAttribute("data-end"));
        var left = Number(node.getAttribute("data-left"));

        // Check Already Updated
        if(keysChanged === false && seg.selectAll("path").size() > 0) return;
        node.innerHTML = "";

        // Check Visibility
        if(visibleBoundaryPixels[0] <= left && left <= visibleBoundaryPixels[1]) {
            // to be drawn
        } else {
            return;
        }

        // Visible
        var segDataset = visibleDataset.filter(function(each) {
            if(start <= each.timestamp && each.timestamp <= end) {
                return true;
            }
            return false;
        });
        var segDataGroups = activeKeys.map(function(key) {
            return {
                key: key,
                values: segDataset.map(function(d) {
                    if(d[key] == null) return null;
                    return {
                        timestamp: d.timestamp,
                        value: d[key]
                    };
                }).filter(function(d) { return d !== null; })
            };
        });

        // No Data
        if(segDataGroups.length === 0) {
            node.innerHTML = "";
            return;
        }

        // Add Path
        seg.selectAll("paths")
            .data(segDataGroups)
            .enter()
            .append("path")
                .attr("fill", "none")
                .attr("stroke-width", 1)
                .attr("class", function(d) { return seriesName(d.key); })
                .attr("d", function(d) {
                    return d3.line()
                        .defined(function(e) { return !isNaN(e.value); })
                        .x(function(e) { return xScale(e.timestamp) })
                        .y(function(e) { return yScale(e.value) })
                        (d.values);
                });

    });

    // Points
    segments.selectAll(".circles circle").remove();
    var circles = segments.select(".circles");
    circles.selectAll("circles")
            .data(activeKeys)
            .enter()
            .append("circle")
                .attr("class", function(key) { return `${seriesName(key)} point`; })
                .attr("data-series", function(key) { return seriesName(key); })
                .attr("stroke", "none")
                .attr("r", 4)
                .style("opacity", 0);

}

async function drawChartV2(fullName, options) {

    // Client-specific vars
    var activeKeys = chartActiveKeys[fullName];
    var entireDataset = chartDataset[fullName];

    // Vars
    var chartQuery = formatChartQuery(fullName);
    var chart = d3.select(chartQuery);
    var chartDuration = options.duration;
    var chartMargin = {
        top: remToPx(1), left: remToPx(2.5), bottom: remToPx(0.75)
    };
    var chartNode = chart.node();
    var chartRect = {
        width: chartNode.offsetWidth - chartMargin.left,
        height: chartNode.offsetHeight - chartMargin.top - chartMargin.bottom
    };
    var entireDataset = chartDataset[fullName];
    var xScale = chartXScale[fullName];
    var xBoundary = d3.extent(entireDataset, function(each) { return each.timestamp; });
    var xDuration = xBoundary[1] - xBoundary[0];
    // Duration too low
    if(chartDuration == undefined || chartDuration > xDuration) chartDuration = xDuration;
    var dataWidth = chartRect.width * xDuration / chartDuration;
    var segNo = Math.ceil(dataWidth / chartRect.width);
    var dataHeight = chartRect.height;
    var xTicks = segNo * 4;
    var yTicks = 4;
    var info = {};
    chartInfo[fullName] = info;
    info.priorActiveKeys = null;
    info.dataWidth = dataWidth;
    info.dataHeight = dataHeight;
    info.width = chartRect.width;
    info.height = chartRect.height;
    info.margin = chartMargin;
    info.xBoundary = xBoundary;
    info.xDuration = xDuration;
    info.yTicks = yTicks;
    info.chartDuration = chartDuration;

    // Rect
    


    // LAYOUT STACK
    // + xAxis
    // + projection
    // + grid
    // + hand
    // + segments
    //   + path
    // + points
    // + tooltip no pointer events

    // Elements
        // Erase Elements first
    chart.node().innerHTML = "";

    // Background
    var background = chart.append("div")
        .attr("class", "background-wrap")
        .style("width", `${chartRect.width}px`)
        .style("height", `${chartRect.height}px`)
        .style("left", `${chartMargin.left}px`)
        .style("top", `${chartMargin.top}px`)
        .append("div")
            .attr("class", "background-container")
            .append("svg")
                .attr("width", chartRect.width)
                .attr("height", chartRect.height)
                .append("g")
                    .attr("class", "background");
    var focusDate = background.append("g")
        .attr("class", "focus-date")
        .append("text")
            .attr("text-anchor", "middle") // Middle Align
            .attr("x", chartRect.width / 2)
            .attr("y", chartRect.height / 2)
            .attr("dy",  "25%")
            .attr("font-size", chartRect.height / 1.5)
            .style("opacity", 0);

    // Segments
    var segmentsWrap = chart.append("div")
        .attr("class", "segments-wrap")
        .style("width", `${chartRect.width}px`)
        .style("height", `${chartRect.height + chartMargin.bottom}px`)
        .style("left", `${chartMargin.left}px`)
        .style("top", `${chartMargin.top}px`);
    var segmentsContainer = segmentsWrap.append("div")
        .attr("class", "segments-container");
    var segments = segmentsContainer.append("svg")
        .attr("width", dataWidth)
        .attr("height", dataHeight)
        .append("g")
            .attr("class", "segments");
        // Set Scroll Left
    var scrollLeft = Math.max(0, dataWidth - chartRect.width);
    segmentsWrap.node().scrollLeft = scrollLeft;

    // Axis
    xScale = xScale.range([0, dataWidth]);
    chartInfo[fullName].xScale = xScale;
    /*var xAxis = segments.append("g")
        .attr("class", "axis x-axis")
        .attr("transform", `translate(0, ${chartRect.height})`)
        .call(d3.axisBottom(xScale)
            .ticks(xTicks)
            .tickSizeOuter(0)
            .tickFormat(function(timestamp) { return timestamp.date(); }))
                .attr("font-family", "")
                .attr("font-size", "");*/

    // Projection (entire path domain)
    var projection = segments.append("g")
        .attr("class", "projection");
    projection.append("rect")
        .attr("class", "projection-domain")
        .attr("x", 0)
        .attr("y", 0)
        .attr("width", dataWidth)
        .attr("height", dataHeight);
    chartInfo[fullName].projection = projection;
    var hand = projection.append("line")
        .attr("class", "hand")
        .attr("x1", scrollLeft + chartRect.width / 2)
        .attr("x2", scrollLeft + chartRect.width / 2)
        .attr("y1", 0)
        .attr("y2", dataHeight);

    { // Each segment
        for(var i = 0; i < segNo; i++) {
            // Pre
            var segLeft = chartRect.width * i;
            var segStart = xScale.invert(chartRect.width * i);
            var segEnd = Math.min(xScale.invert(chartRect.width * (i + 1)), xBoundary[1]);

            // Main
            var seg = segments.append("g")
                .attr("class", "segment")
                .attr("data-start", segStart)
                .attr("data-end", segEnd)
                .attr("data-left", segLeft);
                //.attr("transform", `translate(${segLeft}, 0)`);

            // Post
        }
    }

    // Points
    var circles = segments.append("g")
        .attr("class", "circles");

    // Overlay
    var overlay = chart.append("div")
        .attr("class", "overlay-wrap")
        .style("width", `${chartRect.width}px`)
        .style("height", `${chartRect.height}px`)
        .style("left", `${chartMargin.left}px`)
        .style("top", `${chartMargin.top}px`)
        .append("div")
            .attr("class", "overlay-container")
            .append("svg")
                .attr("width", chartRect.width)
                .attr("height", chartRect.height)
                .append("g")
                    .attr("class", "overlay");
    // Tooltip // no pointer events
    var tooltipSize = { width: 150, height: 37 };
    var tooltip = overlay.append("g")
        .attr("class", "tooltip")
        .style("opacity", 0);
    tooltip.append("rect")
        .attr("class", "background")
        .attr("height", tooltipSize.height)
        .attr("width", tooltipSize.width)
        .attr("x", 0)
        .attr("y", 0);
    var tooltipTimestamp = tooltip.append("text")
        .attr("class", "timestamp")
        .attr("x", 7)
        .attr("y", 7)
        .attr("dy", "0.71em");
    var tooltipIcon = tooltip.append("rect")
        .attr("height", 10)
        .attr("width", 10)
        .attr("x", 7)
        .attr("y", 20);
    var tooltipValue = tooltip.append("text")
        .attr("class", "value")
        .attr("x", 22)
        .attr("y", 21)
        .attr("dy", "0.71em");

    // Update Segments
    updateChartSegments(fullName);

    // Events

    { // Segments Wrap Scroll
        var handler;
        var node = segmentsWrap.node();
        var interval = 10;
        handler = function() {
            // Pre
            node.removeEventListener("scroll", handler);
            
            // Main
            updateChartSegments(fullName);

            // Post
            setTimeout(function() {
                node.addEventListener("scroll", handler);
            }, interval);
        };
        node.addEventListener("scroll", handler);
    }

    { // Hand, Points and Tooltip
        var bisect = d3.bisector(function(d) { return d.timestamp; }).left;

        chart.on("mouseover", function() {
            var event = d3.event;
            segments.selectAll(".point").style("opacity", 1);
            background.select(".focus-date text").style("opacity", 1);
            if(hasClass(event.target, "point")) {
                overlay.selectAll(".tooltip").style("opacity", 1);
            }
        })
        .on("mouseleave", function() {
            segments.selectAll(".point").style("opacity", 0);
        })
        .on("mouseout", function() {
            var event = d3.event;
            background.select(".focus-date text").style("opacity", 0);
            if(hasClass(event.target, "point")) {
                overlay.selectAll(".tooltip").style("opacity", 0);
            }
        })
        .on("mousemove", function() {
            var event = d3.event;
            var target = event.target;
            var mouse = d3.mouse(projection.node());
            var timestamp = xScale.invert(mouse[0]);
            var activeKeys = chartActiveKeys[fullName];
            var dataset = chartInfo[fullName].dataset;
            var seriesName = chartInfo[fullName].seriesName;
            var yScale = chartInfo[fullName].yScale;
            var scrollLeft = segmentsWrap.node().scrollLeft;
            var i = bisect(dataset, timestamp); // returns the index to the current data item
            var d0 = dataset[i - 1];
            var d1 = dataset[i];
            var d;
            if(d0 == undefined) d = d1;
            else if(d1 == undefined) d= d0;
            else d = timestamp - d0.timestamp > d1.timestamp - timestamp ? d1 : d0;
            var posX = xScale(d.timestamp);
    
            // Hand
            hand.attr("x1", posX).attr("x2", posX);

            // Time
            background.select(".focus-date text").text(d.timestamp.date("MM/DD"));

            // Points
            activeKeys.forEach(function(key) {
                var val = d[key];
                if(val == undefined || isNaN(val)) return;
                var posY = yScale(val);
                var series = seriesName(key);
                segments.select(`.point.${series}`)
                    .attr("data-value", val)
                    .attr("data-timestamp", d.timestamp)
                    .attr("cx", posX)
                    .attr("cy", posY);
            });

            // Tooltip
            var [tw, th] = [tooltipSize.width, tooltipSize.height];
            var tx = Math.min(Math.max(posX - scrollLeft - tw/2, 0), chartRect.width - tw);
            var ty = mouse[1] < th ? mouse[1] + 5 : mouse[1] - tooltipSize.height - 5;
            tooltip.attr("transform", `translate(${tx},${ty})`);
            if(hasClass(target, "point")) {
                //
                tooltipValue.text(target.getAttribute("data-value"));
                tooltipTimestamp.text(
                    Number(target.getAttribute("data-timestamp")).date()
                );
                tooltipIcon.attr("class", target.getAttribute("data-series"));
            }
  
        });
    }

}

function remToPx(rem) {
    return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
}