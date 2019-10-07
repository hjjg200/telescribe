
let chartList = {};
let chartTimestampsMap = {};
let chartSeriesMap = {};
let chartActiveSeriesMap = {};

let g_gdc;
let g_cms;
let app;

// Put average values instead of null

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
            let options = this.options
            for(let fullName in this.cmdMap) {
                let [timestamps, seriesMap] = processClientMonitorData(
                    this.cmdMap[fullName], this.cmsMap[fullName], options
                );
                chartTimestampsMap[fullName] = timestamps;
                chartSeriesMap[fullName] = seriesMap;
                chartActiveSeriesMap[fullName] = [];
                this.updateChart(fullName);
            }
        },
        methods: {
            toggleSeries: function(ev, fullName, key, toggle) {
                if(toggle) {
                    chartActiveSeriesMap[fullName].push({
                        key: key,
                        series: chartSeriesMap[fullName][key]
                    });
                    let i = getSeriesIdx(chartActiveSeriesMap[fullName].length);
                } else {
                    let tmp = [];
                    chartActiveSeriesMap[fullName].forEach(function(val) {
                        if(val.key == key) {
                            on = true;
                            return;
                        }
                        tmp.push(val);
                    });
                    chartActiveSeriesMap[fullName] = tmp;
                    ev.target.className = "";
                }
                this.updateChart(fullName);
            },
            updateChart: function(fullName) {
                let series = [];
                let i = 1;
                chartActiveSeriesMap[fullName].forEach(function(val) {
                    let q = formatCheckboxQuery(fullName, val.key);
                    let a = getSeriesIdx(i++);
                    let cb = document.querySelector(q);
                    cb.className = "";
                    cb.classList.add("series-" + a);
                    series.push(val.series);
                });
                let timestamps = chartTimestampsMap[fullName];
                let query = formatChartQuery(fullName);
                drawD3Chart(query, timestamps, series);
            },
            maxStatus: function(cms) {
                let max = -1;
                for(let key in cms) {
                    let st = cms[key].Status;
                    max = st > max ? st : max;
                }
                return max;
            },
            drawChart: function(fullName, timestamps) {
            }
        }
    })

    //
    let dragged = false;
    let dragStartT = {};
    let dragEndT = {};
    document.addEventListener("mouseup", function(ev) {
        document.querySelectorAll(".selection.active").forEach(function(el) {
            let chart = queryParent(el, ".chart");
            let fullName = chart.getAttribute("data-fullName");
            el.classList.remove("active");
            let start = Math.min(dragStartT[fullName], dragEndT[fullName]);
            let end = Math.max(dragStartT[fullName], dragEndT[fullName]);
        });
        dragged = false;
    });

    document.querySelectorAll(".chart").forEach(function(el) {
        let dragStartX = null;
        let fullName = el.getAttribute("data-fullName");
        let idx;
        let getChart = function() { return chartList[fullName]; }

        el.addEventListener("mousemove", function(ev) {
            let grid = getChart().grid;
            let hand = getChart().hand;
            let selection = getChart().selection;
            let rect = grid.getBoundingClientRect();
            let pX = ev.pageX;
            let absX = pX - rect.x;

            // Only in the grid
            if(pX > rect.x && pX < rect.x + rect.width) {
                // Hand
                hand.setAttribute("x1", absX);
                hand.setAttribute("x2", absX);

                // Get nearest timestamp
                let timestamps = chartTimestampsMap[fullName];
                idx = Math.round(absX / rect.width * timestamps.length);

                // Dragged
                if(dragged) {
                    dragEndT[fullName] = idx;
                    if(pX < dragStartX) {
                        selection.setAttribute("x", absX);
                        selection.setAttribute("width", dragStartX - pX);
                    } else {
                        selection.setAttribute("width", pX - dragStartX);
                    }
                }
            }

        });

        el.addEventListener("mousedown", function(ev) {
            let selection = getChart().selection;
            let grid = getChart().grid;
            let rect = grid.getBoundingClientRect();
            let pX = ev.pageX;
            dragged = true;
            dragStartX = pX;
            dragStartT[fullName] = idx;
            selection.setAttribute("x", pX - rect.x);
            selection.setAttribute("width", 0);
            selection.classList.add("active");
        });

        el.addEventListener("mouseover", function(ev) {
            /*if(hasClass(ev.target, "ct-point")) {
                let series = ev.target.parentElement.getAttribute('class').match(/ct-series-\w/)[0];
                let g = document.querySelector("g." + series);
                g.classList.add("active");
            }*/
        });
        el.addEventListener("mouseout", function(ev) {
            /*if(hasClass(ev.target, "ct-point")) {
                let series = ev.target.parentElement.getAttribute('class').match(/ct-series-\w/)[0];
                let g = document.querySelector("g." + series);
                g.classList.remove("active");
            }*/
        });
    });

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

function processClientMonitorData(cmd, cms, options) {

    let gtht = options.gapThresholdTime * 60;
    let xAxis = [];
    let seriesMap = {};
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

    let rcMap = {};
    { // Get all timestamps
        let tmp = {};
        for(let key in cmd) {
            let slice = cmd[key];
            rcMap[key] = {};
            for(let idx in slice) {
                let each = slice[idx];
                tmp[each.Timestamp] = null;
                rcMap[key][each.Timestamp] = {
                    Value: each.Value,
                    Timestamp: each.Timestamp
                };
            }
        }
        for(let t in tmp) {
            xAxis.push(t);
        }
        xAxis.sort();
    }

    // Series
    for(let key in cmd) {
        let slice = cmd[key];
        let tmp = rcMap[key];
        let series = [];

        let lastX = null;
        for(let i = 0; i < xAxis.length; i++) {
            let aX = Number(xAxis[i]);

            if(tmp[aX] != null) {

                // Put gap
                if(lastX != null && aX - lastX > gtht) {
                    series.push({
                        Value: null,
                        Timestamp: lastX + (aX - lastX)/2
                    });
                }

                let aI = tmp[aX];
                series.push({
                    Value: aI.Value,
                    Timestamp: aI.Timestamp
                });
                lastX = aX;

            }

        }
        
        seriesMap[key] = series;
    }

    return [xAxis, seriesMap];
    
}

function getHours(t) {
    return moment.unix(t).format("DD HH:mm")
}

function isHidden(elem) {
    var style = window.getComputedStyle(elem);
    return (style.display === 'none');
}

//
// D3.js
//

function remToPx(rem) {
    return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
}

function getMaxValue(series) {
    ret = -Infinity; // minus infinity
    if(series.length == 0)
        return 1;
    series.forEach(function(slice) {
        for(let i = 0; i < slice.length; i++) {
            ret = Math.max(ret, slice[i].Value);
        }
    });
    return ret;
}

function getMinValue(series) {
    ret = Infinity; // minus infinity
    if(series.length == 0)
        return 0;
    series.forEach(function(slice) {
        for(let i = 0; i < slice.length; i++) {
            ret = Math.min(ret, slice[i].Value);
        }
    });
    return ret;
}

function getMaxTimestamp(ts) {
    ret = -Infinity;
    for(let i = 0; i < ts.length; i++) {
        if(ts[i] == null) continue;
        ret = Math.max(ret, ts[i]);
    }
    return ret;
}

function getMinTimestamp(ts) {
    ret = Infinity;
    for(let i = 0; i < ts.length; i++) {
        if(ts[i] == null) continue;
        ret = Math.min(ret, ts[i]);
    }
    return ret;
}

async function drawD3Chart(query, timestamps, series) {
    let elem = document.querySelector(query);
    elem.innerHTML = "";
    let fullName = elem.getAttribute("data-fullName");
    chartList[fullName] = {};
    let margin = {top: remToPx(1), left: remToPx(3.5), bottom: remToPx(2) };
    let height = elem.offsetHeight - margin.bottom - margin.top;
    let width = elem.offsetWidth - margin.left;

    let svg = d3.select(query)
        .append("svg")
            .attr("height", height + margin.bottom + margin.top)
            .attr("width", width + margin.left)
            .append("g")
                .attr("transform", `translate(${margin.left}, ${margin.top})`);

    // For axis
    let minY = getMinValue(series);
    let maxY = getMaxValue(series);
    let minX = getMinTimestamp(timestamps);
    let maxX = getMaxTimestamp(timestamps);

    // Add X axis --> it is a date format
    var x = d3.scaleLinear()
        .domain([minX, maxX])
        .range([0, width]);
    svg.append("g")
        .attr("transform", "translate(0," + (height+0) + ")")
        .classed("axis", true)
        .call(d3.axisBottom(x)
            .ticks(4)
            .tickSizeOuter(0)
            .tickFormat(function(d) { return getHours(d);}))
        .attr("font-family", "")
        .attr("font-size", "");

    // Add Y axis
    var y = d3.scaleLinear()
        .domain([minY, maxY])
        .range([height, 0]);
    svg.append("g")
        .attr("transform", "translate(0,0)")
        .classed("axis", true)
        .call(d3.axisLeft(y).ticks(4).tickSizeOuter(0))
        .attr("font-family", "")
        .attr("font-size", "");

    // Y axis grid
    svg.append("g")
        .classed("grid", true)
        .call(d3.axisLeft(y).ticks(4).tickSize(-width).tickFormat(""));
    let grid = elem.querySelector(".grid");

    // Selection
    svg.append("rect")
        .attr("x", width/2)
        .attr("y", 0)
        .attr("height", height)
        .classed("selection", true);
    let selection = elem.querySelector(".selection");
    
    // X axis hand
    svg.append("line")
        .attr("x1", width/2)
        .attr("x2", width/2)
        .attr("y1", 0)
        .attr("y2", height)
        .classed("hand", true);
    let hand = elem.querySelector(".hand");
    
    chartList[fullName] = {
        svg: svg,
        minY: minY,
        maxY: maxY,
        minX: minX,
        maxX: maxX,
        xFunc: x,
        yFunc: y,
        grid: grid,
        selection: selection,
        hand: hand
    };

    //
    for(let i = 0; i < series.length; i++) {
        let slice = series[i];
        // Trim data out of scope
        slice = slice.filter(function(v) { return v.Timestamp >= minX && v.Timestamp <= maxX; });
        let seriesName = "series-" + getSeriesIdx(i + 1);
        let gaplessSlice = slice.filter(function(v) { return v.Value !== null; });

        //
        let seriesG = svg.append("g")
            .classed("series", true);
        
        // Add the line
        seriesG.append("path")
            .datum(slice)
            .attr("fill", "none")
            .attr("stroke-width", 1)
            .classed(seriesName, true)
            .attr("d", d3.line()
                .x(function(d) { return x(d.Timestamp) })
                .y(function(d) { return y(d.Value) })
                .defined(function(d) { return d.Value !== null; })
            );

        // Add the points
        seriesG.selectAll("circles")
            .data(gaplessSlice)
            .enter()
            .append("circle")
            .classed(seriesName, true)
            .attr("stroke", "none")
            .attr("cx", function(d) { return x(d.Timestamp) })
            .attr("cy", function(d) { return y(d.Value) })
            .attr("r", 4);
    }

}