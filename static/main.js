
let chartList = {}
let chartLabelsMap = {}
let chartSeriesMap = {}
let chartActiveSeriesMap = {}
let g_gdc
let g_cms
let app

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
                let gdc = this.gdc
                let ret = {}
                for(let key in gdc) {
                    if(key.startsWith("options.")) {
                        let split = key.split(".")
                        ret[split[1]] = gdc[key]
                    }
                }
                return ret
            }
        },
        created: function() {
            for(let host in this.caMap) {
                this.checked[host] = {}
            }
        },
        mounted: function() {
            let options = this.options
            for(let host in this.cmdMap) {
                let [labels, seriesMap] = processClientMonitorData(this.cmdMap[host], options)
                chartLabelsMap[host] = labels
                chartSeriesMap[host] = seriesMap
                chartActiveSeriesMap[host] = []
                this.drawChart(host, labels)
            }
        },
        methods: {
            toggleSeries: function(ev, host, key, toggle) {
                if(toggle) {
                    chartActiveSeriesMap[host].push({
                        key: key,
                        series: chartSeriesMap[host][key]
                    })
                    let i = getSeriesIdx(chartActiveSeriesMap[host].length)
                } else {
                    let tmp = []
                    chartActiveSeriesMap[host].forEach(function(val) {
                        if(val.key == key) {
                            on = true
                            return
                        }
                        tmp.push(val)
                    })
                    chartActiveSeriesMap[host] = tmp
                    ev.target.className = ""
                }
                this.updateChart(host)
            },
            updateChart: function(host) {
                let series = []
                let i = 1
                chartActiveSeriesMap[host].forEach(function(val) {
                    let q = formatCheckboxQuery(host, val.key)
                    let a = getSeriesIdx(i++)
                    let cb = document.querySelector(q)
                    cb.className = ""
                    cb.classList.add("series-" + a)
                    series.push(val.series)
                })
                chartList[host].data.series = series
                chartList[host].update()
            },
            maxStatus: function(cms) {
                let max = -1
                for(let key in cms) {
                    let st = cms[key].Status
                    max = st > max ? st : max
                }
                return max
            },
            drawChart: function(host, labels) {
                let query = formatChartQuery(host)
                let labelLength = labels.length
                let labelStep = Math.floor(labelLength / 5)
                chartList[host] = new Chartist.Line(
                    query, {
                        series: [],
                        labels: labels
                    }, {
                        showPoint: true,
                        fullWidth: true,
                        axisX: {
                            showGrid: false,
                            labelInterpolationFnc: (v, idx) => {
                                if(idx % labelStep == 0) {
                                    return getHours(v)
                                }
                                return null
                            }
                        },
                        axisY: {
                            labelInterpolationFnc: (v, idx) => {
                                if(v >= 1e+9 * 0.8) {
                                    return (Math.round(v / 1e+9 * 10) / 10) + "B"
                                } else if(v >= 1e+6 * 0.8) {
                                    return (Math.round(v / 1e+6 * 10) / 10) + "M"
                                } else if(v >= 1e+3 * 0.8) {
                                    return (Math.round(v / 1e+3 * 10) / 10) + "K"
                                } else {
                                    return v
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

})


async function fetchAndUpdate() {
    
    let gdcResponse = await fetch(
        "graphDataComposite.json", {
        method: "GET",
        cache: "no-cache"
    })

    let cmsResponse = await fetch(
        "clientMonitorStatus.json", {
        method: "GET",
        cache: "no-cache"
    })

    g_gdc = await gdcResponse.json()
    g_cms = await cmsResponse.json()

}

function getSeriesIdx(i) {
    return "abcdefghijklmno".charAt(i - 1)
}

function formatChartQuery(host) {
    host = host.replace(/"/g, '\\\"')
    return `#host-list li[data-host="${host}"] .chart`
}

function formatCheckboxQuery(host, key) {
    host = host.replace(/"/g, '\\\"')
    key = key.replace(/"/g, '\\\"')
    return `#host-list li[data-host="${host}"] li[data-key="${key}"] input`
}

function processClientMonitorData(cmd, options) {

    let gtht = options.gapThresholdTime * 60
    let xAxis = []
    let labels = []
    let seriesMap = {}
    {
        let tmp = {}
        for(let key in cmd) {
            let slice = cmd[key]
            for(let idx in slice) {
                let each = slice[idx]
                tmp[each.Timestamp] = null
            }
        }
        for(let t in tmp) {
            xAxis.push(t)
        }
        xAxis.sort()
    }

    // Put gaps
    let tmp = []
    let glen = Math.round(xAxis.length * options.gapPercent / 100) + 1
    let labelStep = Math.floor(xAxis.length / 5)
    let gapAdded = false
    for(let i = 0; i < xAxis.length; i++) {
        let ts = xAxis[i]
        if(i < xAxis.length - 1 && xAxis[i + 1] - ts > gtht) {// Gap
            gapAdded = true
            for(let j = 0; j < glen; j++) {
                labels.push(null)
                tmp.push(null)
            }
        } else if(i % labelStep == 0 || gapAdded) {
            gapAdded = false
            labels.push(getHours(ts))
            tmp.push(ts)
        } else {
            labels.push(null)
            tmp.push(ts)
        }
    }
    xAxis = tmp

    // Series
    for(let key in cmd) {
        let slice = cmd[key]
        let tmp = {}
        let series = []

        for(let idx in slice) {
            let each = slice[idx]
            tmp[each.Timestamp] = {
                value: each.Value,
                meta: moment.unix(each.Timestamp).format(options.momentJsFormat)
            }
        }
        let gapAdded = false
        for(let i = 0; i < xAxis.length; i++) {
            let x = xAxis[i]
            if(x != null && i < xAxis.length - 1 && xAxis[i + 1] == null) {
                series.push({
                    value: tmp[x].value,
                    meta: "*" + tmp[x].meta
                })
                gapAdded = true
            } else if(x == null) {
                series.push({
                    value: null
                })
            } else {
                let each = tmp[x]
                if(gapAdded) {
                    gapAdded = false
                    each.meta = "*" + each.meta
                }
                series.push({
                    value: each.value,
                    meta: each.meta
                })
            }
        }
        seriesMap[key] = series
    }

    return [xAxis, seriesMap]
    
}

function getHours(t) {
    let now = (new Date()).getTime() / 1000
    let diff = now - t
    if(diff < 10800) {
        return Math.floor(diff / 60) + "m"
    } else {
        return Math.floor(diff / 3600) + "h"
    }
}

function isHidden(elem) {
    var style = window.getComputedStyle(elem)
    return (style.display === 'none')
}