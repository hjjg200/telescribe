
let chartList = {};
let chartDataset = {};
let chartActiveKeys = {};
let chartActiveBoundary = {};

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
                let dataset = processChartDataset(
                    this.cmdMap[fullName], this.cmsMap[fullName], options
                );
                chartDataset[fullName] = dataset;
                chartActiveKeys[fullName] = [];
                chartActiveBoundary[fullName] = [NaN, NaN];
                this.updateChart(fullName);
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
                let query = formatChartQuery(fullName);
                drawD3Chart(query);
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

    return dataset.sort(function(a, b) {
        if(a.timestamp > b.timestamp) return 1;
        if(a.timestamp < b.timestamp) return -1;
        return 0;
    });
    
}

function getHours(t) {
    return moment.unix(t).format("DD HH:mm");
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

async function drawD3Chart(query) {
    let chart = document.querySelector(query);
    chart.innerHTML = "";
    let fullName = chart.getAttribute("data-fullName");
    chartList[fullName] = {};
    let activeKeys = chartActiveKeys[fullName];
    let activeBoundary = chartActiveBoundary[fullName];
    let margin = {top: remToPx(1), left: remToPx(2.5), bottom: remToPx(2) };
    let height = chart.offsetHeight - margin.bottom - margin.top;
    let width = chart.offsetWidth - margin.left;

    let svg = d3.select(query)
        .append("svg")
            .attr("height", height + margin.bottom + margin.top)
            .attr("width", width + margin.left)
            .append("g")
                .attr("transform", `translate(${margin.left}, ${margin.top})`);

    // For axis
    let entireDataset = chartDataset[fullName];
    let boundarySet = !isNaN(activeBoundary[0]) && !isNaN(activeBoundary[1]);
    let dataset = entireDataset.filter(function(d) {
        if(boundarySet) {
            if(d.timestamp < activeBoundary[0] || d.timestamp > activeBoundary[1]) return false;
        }
        for(let key in d) {
            if(key === "timestamp") continue;
            if(activeKeys.indexOf(key) !== -1) return true;
        }
        return false;
    });
    let dataGroups = activeKeys
        .map(function(key) {
            return {
                key: key,
                values: dataset.map(function(d) {
                    if(d[key] == null) return null;
                    return {
                        timestamp: d.timestamp,
                        value: d[key]
                    };
                }).filter(function(d) { return d !== null; })
            };
        });

    // Map
    var seriesIndexes = d3.scaleOrdinal()
        .domain(chartActiveKeys[fullName])
        .range(["a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o"]);

    // Add X axis
    var x = d3.scaleLinear()
        .domain(d3.extent(dataset, function(d) { return d.timestamp; }))
        .range([0, width]);
    svg.append("g")
        .attr("transform", `translate(0, ${height})`)
        .classed("axis", true)
        .call(d3.axisBottom(x)
            .ticks(4)
            .tickSizeOuter(0)
            .tickFormat(function(d) { return getHours(d);}))
        .attr("font-family", "")
        .attr("font-size", "");

    // Add Y axis
    var y = d3.scaleLinear()
        .domain(d3.extent(function() {
            var arr = [];
            dataGroups.forEach(function(d) {
                var values = d.values.filter(function(d) { return !isNaN(d.value); } );
                d3.extent(values, function(e) {
                    return e.value;
                }).forEach(function(f) {
                    arr.push(f);
                });
            });
            return d3.extent(arr);
        }()))
        .range([height, 0]);
    svg.append("g")
        .attr("transform", "translate(0,0)")
        .classed("axis", true)
        .call(d3.axisLeft(y)
            .ticks(4)
            .tickSizeOuter(0)
            .tickFormat(function(d) {
                if(d >= 1e+9 * 0.8) {
                    return (Math.round(d / 1e+9 * 10) / 10) + "B";
                } else if(d >= 1e+6 * 0.8) {
                    return (Math.round(d / 1e+6 * 10) / 10) + "M";
                } else if(d >= 1e+3 * 0.8) {
                    return (Math.round(d / 1e+3 * 10) / 10) + "K";
                }
                return d;
            }))
        .attr("font-family", "")
        .attr("font-size", "");

    // Y axis grid
    svg.append("g")
        .classed("grid", true)
        .call(d3.axisLeft(y).ticks(4).tickSize(-width).tickFormat(""));
    let grid = chart.querySelector(".grid");

    // Selection
    var selection = svg.append("rect")
        .attr("x", width/2)
        .attr("y", 0)
        .attr("height", height)
        .classed("selection", true);
    
    // X axis hand
    svg.append("line")
        .attr("x1", width/2)
        .attr("x2", width/2)
        .attr("y1", 0)
        .attr("y2", height)
        .classed("hand", true);
    let hand = chart.querySelector(".hand");

    // Overlay
    svg.append("rect")
        .attr("class", "overlay")
        .attr("x", 0)
        .attr("y", 0)
        .attr("height", height)
        .attr("width", width)
        .attr("fill", "none")
        .style("pointer-events", "all");
    let overlay = chart.querySelector(".overlay");

/*
https://www.d3-graph-gallery.com/graph/line_several_group.html
https://stackoverflow.com/questions/25367987/d3-js-get-nearest-x-y-coordinates-for-area-chart
http://bl.ocks.org/mikehadlow/93b471e569e31af07cd3
https://bl.ocks.org/fabiomainardi/00fd581dc5ba92d99eec
https://github.com/d3/d3-shape/blob/v1.3.5/README.md#line_defined
*/
    svg.append("g")
        .attr("class", "paths")
        .selectAll("paths")
        .data(dataGroups)
        .enter()
        .append("path")
            .attr("fill", "none")
            .attr("stroke-width", 1)
            .attr("class", function(d) { return "series-" + seriesIndexes(d.key); })
            .attr("d", function(d) {
                return d3.line()
                    .defined(function(e) { return !isNaN(e.value); })
                    .x(function(e) { return x(e.timestamp) })
                    .y(function(e) { return y(e.value) })
                    (d.values);
            });

    // X axis hand points
    var circles = svg.selectAll("circles")
        .data(dataGroups)
        .enter()
        .append("circle")
            .attr("class", function(d) { return `series-${seriesIndexes(d.key)} point`; })
            .attr("data-series", function(d) { return seriesIndexes(d.key); })
            .attr("stroke", "none")
            .attr("cx", function(d) { return x(d.values[0].timestamp) })
            .attr("cy", function(d) { return y(d.values[0].value) })
            .attr("r", 4)
            .style("opacity", 0);

    // Tooltip // no pointer events
    var tooltipSize = {
        height: 37,
        width: 150
    };
    var tooltip = svg.append("g")
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

    // EVENTS
    var circleHandler = function(circle) {
        tooltipIcon.attr("class", `series-${circle.getAttribute("data-series")}`);
        tooltipTimestamp.text(getHours(circle.getAttribute("data-timestamp")));
        tooltipValue.text(circle.getAttribute("data-value"));
    }
    circles.on("mouseover", function(d) {
        circleHandler(this);
        tooltip.style("opacity", 1);
    })
    .on("mouseout", function(d) {
        tooltip.style("opacity", 0);
    })
    .on("mousemove", function() {
        circleHandler(this);
    });

    // Overlay
    var dragged = false;
    var dragStartTimestamp = undefined;
    var dragEndTimestamp = undefined;
    var bisect = d3.bisector(function(d) { return d.timestamp; }).left;
    svg.on("mouseout", function() {
        svg.selectAll(".point").style("opacity", 0);
    })
    .on("mouseover", function() {
        svg.selectAll(".point").style("opacity", 1);
    })
    .on("mousedown", function() {
        dragged = true;
        dragEndTimestamp = undefined;
        selection.attr("opacity", 1);
    })
    .on("mouseup", function() {
        var event = d3.event;

        // Return
        if(dragStartTimestamp == dragEndTimestamp || dragEndTimestamp === undefined) return;

        // Filter range
        if(event.shiftKey) {
            // Reset
            chartActiveBoundary[fullName] = [NaN, NaN];
        } else {
            chartActiveBoundary[fullName] = d3.extent([dragStartTimestamp, dragEndTimestamp]);
        }

        dragged = false;
        dragStartTimestamp = undefined;
        selection.attr("opacity", 0)
            .attr("width", 0);

        // Filter
        drawD3Chart(query);
    })
    .on("mousemove", function() {
        var mouse = d3.mouse(overlay);
        var mouseTimestamp = x.invert(mouse[0]);
        var i = bisect(dataset, mouseTimestamp); // returns the index to the current data item

        var d0 = dataset[i - 1];
        var d1 = dataset[i];
        // work out which date value is closest to the mouse
        var d;
        if(d0 == undefined)
            d = d1;
        else
            d = mouseTimestamp - d0.timestamp > d1.timestamp - mouseTimestamp ? d1 : d0;

        var posX = x(d.timestamp);

        // Hand
        svg.select(".hand")
            .attr("x1", posX)
            .attr("x2", posX);

        // Selection
        if(dragged) {
            if(dragStartTimestamp === undefined) {
                selection.attr("x", posX);
                dragStartTimestamp = d.timestamp;
            }

            //
            var dragStartX = x(dragStartTimestamp);
            dragEndTimestamp = d.timestamp;
            if(d.timestamp < dragStartTimestamp) {
                selection.attr("x", posX)
                    .attr("width", dragStartX - posX);
            } else {
                selection.attr("x", dragStartX)
                    .attr("width", posX - dragStartX);
            }
        }

        // Tooltip
        var tx = mouse[0] - tooltipSize.width / 2;
        var ty = mouse[1] - tooltipSize.height - 5;
        tooltip.attr("transform", `translate(${tx} ,${ty})`);

        chartActiveKeys[fullName].forEach(function(key) {
            let seriesName = "series-" + seriesIndexes(key);
            var posY;
            if(d[key] == null || isNaN(d[key])) return;
            
            posY = y(d[key]);
            svg.select(`.point.${seriesName}`)
                .attr("data-value", d[key])
                .attr("data-timestamp", d.timestamp)
                .attr("cx", posX)
                .attr("cy", posY);
        });
    });

}