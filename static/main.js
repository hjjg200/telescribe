
let chartList = {}

document.addEventListener(
    "DOMContentLoaded",
    () => {
        fetchAndUpdate();

        ["touchstart", "touchmove"].forEach(( typ ) => {
            document.addEventListener( typ, ( ev ) => {
                ev.target.classList.add( "touch" )
            })
        });
        ["touchend"].forEach(( typ ) => {
            document.addEventListener( typ, ( ev ) => {
                ev.target.classList.remove( "touch" )
            })
        });
    }
)

async function fetchAndUpdate() {
    
    let gdcResponse = await fetch(
        "graphDataComposite.json", {
        method: "GET",
        mode: "same-origin",
        cache: "no-cache",
        credentials: "same-origin"
    } )
    let gdc = await gdcResponse.json()

    let cmsResponse = await fetch(
        "clientMonitorStatus.json", {
        method: "GET",
        mode: "same-origin",
        cache: "no-cache",
        credentials: "same-origin"
    } )
    let cms = await cmsResponse.json()
    
    let hostList = document.querySelector( ".host-list" )
    let cmd = gdc.clientMonitorData
    let ca = gdc.clientAliases
    for( let host in cmd ) {
        let alias = ca[host]
        let status = -1
        let escHost = host.replace( /"/g, '\\\"' )

        chartList[host] = {}

        let hostItem = document.createElement( "li" )
        hostList.appendChild( hostItem )
        let hostHeader = document.createElement( "div" )
        hostItem.setAttribute( "data-host", host )
        let hostStatus = document.createElement( "span" )
        hostHeader.appendChild( hostStatus )
        hostStatus.classList.add( "status" )
        let hostName = document.createElement( "span" )
        hostHeader.append( hostName )
        hostName.innerHTML = `${alias} (${host})`
        hostHeader.classList.add( "host-header" )
        hostItem.appendChild( hostHeader )

        let keyList = document.createElement( "ul" )
        hostItem.appendChild( keyList )
        for( let key in cmd[host] ) {
            let escKey = key.replace( /"/g, '\\\"' )
            let slice = cmd[host][key]
            let cmse = cms[host][key]
            let last = slice[slice.length - 1]

            if( cmse == undefined ) {
                cmse = {
                    Status: 0,
                    Value: last.Value
                }
            } else if( last.Timestamp != cmse.Timestamp ) {
                slice.push( cmse )
            }

            let keyStatus = cmse.Status

            let keyItem = document.createElement( "li" )
            keyList.appendChild( keyItem )
            let keyHeader = document.createElement( "div" )
            keyItem.appendChild( keyHeader )
            keyHeader.classList.add( "key-header" )
            keyHeader.setAttribute( "data-status", keyStatus )

            keyItem.addEventListener( "click", ( ev ) => {
                let handler = [...keyHeader.querySelectorAll( "*" )]
                if( ev.target != keyHeader && handler.includes( ev.target ) == false )
                    return
                let childChart = keyItem.querySelector( ".chart" )
                childChart.style.display = isHidden( childChart ) ? "block" : "none"
                let childChartObj = chartList[host][key]
                childChartObj.update()
            } );

            let keyKeySpan = document.createElement( "span" )
            keyHeader.appendChild( keyKeySpan )
            keyKeySpan.innerHTML = key
            let keyKeyValue = document.createElement( "span" )
            let value = cmse.Value
            keyHeader.appendChild( keyKeyValue )
            keyKeyValue.innerHTML = value

            if( keyStatus > status ) {
                status = keyStatus
                hostStatus.setAttribute( "data-status", status )
            }

            let chartDiv = document.createElement( "div" )
            keyItem.appendChild( chartDiv )
            chartDiv.classList.add( "chart" )
            chartDiv.setAttribute( "data-key", key )
            let chartSelector = `.host-list li[data-host="${escHost}"] div[data-key="${escKey}"]`
            let [ chartLabels, chartSeries ] = processDataSlice( 
                slice, {
                    gapThresholdTime: gdc["options.gapThresholdTime"],
                    gapPercent: gdc["options.gapPercent"],
                    momentJsFormat: gdc["options.momentJsFormat"]
                }
            )
            chartList[host][key] = new Chartist.Line(
                chartSelector, {
                    series: [chartSeries],
                    labels: chartLabels
                }, {
                    showPoint: true,
                    fullWidth: true,
                    chartPadding: {
                        /* top: 30, // default 15
                        left: 30, // default 10
                        right: 45 // default 15 */
                    },
                    axisX: {
                        showGrid: false
                    },
                    axisY: {
                        showLabels: false,
                        labelInterpolationFnc: ( v, idx ) => {
                            if( v >= 1e+9 * 0.8 ) {
                                return ( Math.round( v / 1e+9 * 10 ) / 10 ) + "B"
                            } else if( v >= 1e+6 * 0.8 ) {
                                return ( Math.round( v / 1e+6 * 10 ) / 10 ) + "M"
                            } else if( v >= 1e+3 * 0.8 ) {
                                return ( Math.round( v / 1e+3 * 10 ) / 10 ) + "K"
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
}

function processDataSlice( slice, options ) {

    let gtht = options.gapThresholdTime * 60
    let glen = Math.round( slice.length * options.gapPercent / 100 ) + 1
    let gapAdded = false
    let labelStep = Math.floor( slice.length / 5 )
    let series = []
    let labels = []

    let getHours = ( t ) => {
        let now = ( new Date() ).getTime() / 1000
        let diff = now - t
        if( diff < 3600 ) {
            return Math.floor( diff / 60 ) + "m"
        } else {
            return Math.floor( diff / 3600 ) + "h"
        }
    }

    for( let i = 0; i < slice.length; i++ ) {
        let item = {
            meta: moment.unix( slice[i].Timestamp ).format( options.momentJsFormat ),
            value: Math.round( slice[i].Value * 100 ) / 100
        }
        
        if( i < slice.length - 1 && slice[i + 1].Timestamp - slice[i].Timestamp > gtht ) {

            item.meta = "*" + item.meta
            series.push( item )
            labels.push( getHours( slice[i].Timestamp ) )
            gapAdded = true
            // Gap
            for( let j = 0; j < glen; j++ ) {
                series.push( null )
                labels.push( null )
            }

        } else if( i % labelStep == 0 || gapAdded ) {
            if( gapAdded ) {
                gapAdded = false
                item.meta = "*" + item.meta
            }
            series.push( item )
            labels.push( getHours( slice[i].Timestamp ) )
        } else {
            series.push( item )
            labels.push( null )
        }

    }

    return [ labels, series ]

}

function isHidden( elem ) {
    var style = window.getComputedStyle( elem )
    return ( style.display === 'none' )
}