
let _defaults = {
  gapThresholdTime: 0
};
let _options = {};

export class Chart {

  constructor(target, dataset) {
    // Target
    if(typeof target === "string") this.target = document.querySelector(target); // Query
    else this.target = target; // Element
    // Public
    this.dataset = dataset;
    // Private
    this._duration = 3 * 3600;
    this._keys = [];
    // Pre
    this._xScale = this._scale();

    this._draw();
  }

// Private
  // DRAW
  async _draw() {
    
    // Shorthand access
    var $ = this;

    // Client-specific vars
    var activeKeys = this._keys;
    var entireDataset = this.dataset;
  
  // Vars
    var chart = d3.select(this.target);
    var chartDuration = this._duration;
    var chartMargin = {
      top: Chart.remToPx(0.5),
      left: Chart.remToPx(2.5),
      bottom: Chart.remToPx(2)
    };
    var chartNode = chart.node();
    var chartRect = {
      width: chartNode.offsetWidth - chartMargin.left,
      height: chartNode.offsetHeight - chartMargin.top - chartMargin.bottom
    };
    var xScale = this._xScale;
    var xBoundary = d3.extent(entireDataset, function(each) { return each.timestamp; });
    var xDuration = xBoundary[1] - xBoundary[0];
    // Duration too low
    if(chartDuration == undefined || chartDuration > xDuration) chartDuration = xDuration;
    var dataWidth = chartRect.width * xDuration / chartDuration;
    var segNo = Math.ceil(dataWidth / chartRect.width);
    var dataHeight = chartRect.height;
    var xTicks = segNo * 4;
    var yTicks = 4;
    this._priorActiveKeys = null;
    this._dataWidth = dataWidth;
    this._dataHeight = dataHeight;
    this._width = chartRect.width;
    this._height = chartRect.height;
    this._margin = chartMargin;
    this._xBoundary = xBoundary;
    this._xDuration = xDuration;
    this._yTicks = yTicks;
    this._chartDuration = chartDuration;
  
  // Element Var
    var scrollLeft = Math.max(0, dataWidth - chartRect.width);
    var handX = scrollLeft + chartRect.width / 2;
  
  // PRIOR VALUES
    var prior = false;
    var priorHandXEnRectPercent;
    var priorHandXPercent;
    {
      var segmentsWrap = chart.select(".segments-wrap");
      if(segmentsWrap.size() > 0) {
        prior = true;
  
        var node = segmentsWrap.node();
        var hand = segmentsWrap.select(".hand");
        var x = Number(hand.attr("x1"));
  
        priorHandXEnRectPercent = (x - node.scrollLeft) / node.offsetWidth;
        priorHandXPercent = x / node.scrollWidth;
  
        // Scroll Left and Hand X
        handX = priorHandXPercent * dataWidth;
        scrollLeft = Math.max(0, handX - chartRect.width * priorHandXEnRectPercent);
      }
    }
  
  // LAYOUT STACK
  // + xAxis
  // + projection
  // + grid
  // + hand
  // + segments
  //  + path
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
        .attr("x", (chartRect.width + chartMargin.left) / 2 - chartMargin.left)
        .attr("y", chartRect.height / 2)
        .attr("dy", "25%")
        .attr("font-size", chartRect.height / 1.5)
        .style("opacity", 0);
  
  // Segments
    var segmentsWrap = chart.append("div")
      .attr("class", "segments-wrap")
      .style("width", `${chartRect.width}px`)
      .style("height", `${chartRect.height + chartMargin.top + chartMargin.bottom}px`)
      .style("left", `${chartMargin.left}px`)
      .style("top", `0px`);
    var segmentsContainer = segmentsWrap.append("div")
      .attr("class", "segments-container");
    var segments = segmentsContainer.append("svg")
      .attr("width", dataWidth)
      .attr("height", dataHeight)
      .append("g")
        .attr("class", "segments")
        .attr("transform", `translate(0, ${chartMargin.top})`);
    // Set Scroll Left
    segmentsWrap.node().scrollLeft = scrollLeft;
  
  // Axis
    xScale = xScale.range([0, dataWidth]);
    var xAxis = segments.append("g")
      .attr("class", "axis x-axis")
      .attr("transform", `translate(0, ${chartRect.height})`)
      .call(d3.axisBottom(xScale)
        .tickValues(xScale.ticks(xTicks).tickValues())
        .tickSizeOuter(0)
        .tickFormat(function(timestamp) { return timestamp.date(); }))
          .attr("font-family", "")
          .attr("font-size", "");
  
  // Projection (entire path domain)
    var projection = segments.append("g")
      .attr("class", "projection");
    projection.append("rect")
      .attr("class", "projection-domain")
      .attr("x", 0)
      .attr("y", 0)
      .attr("width", dataWidth)
      .attr("height", dataHeight);
    this._projection = projection;
  // Hand
    var hand = projection.append("line")
      .attr("class", "hand")
      .attr("x1", handX)
      .attr("x2", handX)
      .attr("y1", 0)
      .attr("y2", dataHeight);
  
  // Each segment
    {
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
    await this.update();
  
  // EVENTS
    { // Segments Wrap Scroll
      var interval = 10;
      var bind = function(type, elem) {
        var handler;
        handler = function() {
          // Pre
          elem.removeEventListener(type, handler);
          // Main
          $.update().then(function() {
            // Post
            setTimeout(function() {
              elem.addEventListener(type, handler);
            }, interval);
          });
        };
        elem.addEventListener(type, handler);
      }
      bind("scroll", segmentsWrap.node());
    }
    { // Window Resize
      var handler;
      var timer;
      handler = function() {
        clearTimeout(timer);
        timer = setTimeout(function() {
          $._draw();
        }, 500);
      };
      window.addEventListener("resize", handler);
    }
    { // Hand, Points and Tooltip and Touch Interface
      var bisect = d3.bisector(function(d) { return d.timestamp; }).left;
      var mouseHandler = function() {
        var event = d3.event;
        var target = event.target;
        var mouse = d3.mouse(projection.node());
        var timestamp = xScale.invert(mouse[0]);
        var activeKeys = $._keys;
        var activeDataset = $._activeDataset;
        var seriesName = $._seriesName;
        var yScale = $._yScale;
        var scrollLeft = segmentsWrap.node().scrollLeft;
        //
        segments.selectAll(".point").style("opacity", 1);
        background.select(".focus-date text").style("opacity", 1);
        if(hasClass(event.target, "point")) {
          overlay.selectAll(".tooltip").style("opacity", 1);
        }
        //
        var i = bisect(activeDataset, timestamp); // returns the index to the current data item
        var d0 = activeDataset[i - 1];
        var d1 = activeDataset[i];
        var d;
        if(d0 == undefined && d1 == undefined) return;
        else if(d0 == undefined) d = d1;
        else if(d1 == undefined) d = d0;
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
      };
  
      //
      { // Touch Interface
        var touch = false;
        window.addEventListener("touchstart", function handler() {
          touch = true;
          window.removeEventListener("touchstart", handler);
        });
        // Dispatch mouse event as you scroll
        var node = segmentsWrap.node();
        var lastLeft = node.scrollLeft;
        var interval = 1;
        handler = function(event) {
          // Pre
          node.removeEventListener("scroll", handler);
          // Main
          if(touch) {
            var left = node.scrollLeft;
            var rect = node.getBoundingClientRect();
            var handX = Number(hand.attr("x1"));
            // Arbitrary Coords
            event.clientX = (handX - left) + (left - lastLeft) + rect.left;
            event.clientY = rect.top + rect.height / 2;
            lastLeft = left;
            d3.customEvent(event, mouseHandler);
          }
          // Post
          setTimeout(function() {
            node.addEventListener("scroll", handler);
          }, interval);
        };
        node.addEventListener("scroll", handler);
      }
  
      // Add Handlers
      chart.on("mouseover", mouseHandler)
      .on("mouseout", function() {
        var event = d3.event;
        background.select(".focus-date text").style("opacity", 0);
        if(hasClass(event.target, "point")) {
          overlay.selectAll(".tooltip").style("opacity", 0);
        }
      })
      .on("mousemove", mouseHandler);
    }
  
  }

  // UPDATE
  async update() {
    // Shorthand
    var $ = this;

    // Chart
    var chart = d3.select(this.target);
    var entireDataset = this.dataset;
    var activeKeys = this._keys;

    var priorActiveKeys = this._priorActiveKeys;
    var xDuration = this._xDuration;
    var xBoundary = this._xBoundary;
    var chartWidth = this._width;
    var chartHeight = this._height;
    var chartMargin = this._margin;
    var dataWidth = this._dataWidth;
    var dataHeight = this._dataHeight;
    var chartDuration = this._chartDuration;
    var yTicks = this._yTicks;
    // ELements
    var xScale = this._xScale;
    var projection = this._projection;

    // Check Changes
    var keysChanged = !Chart.keysEqual(activeKeys, priorActiveKeys);

    // Info Set
    this._priorActiveKeys = activeKeys.slice(0);

    // Segments
    var segmentsWrap = chart.select(".segments-wrap");
    var scrollLeft = segmentsWrap.node().scrollLeft;

    // Dataset
    var seriesName = d3.scaleOrdinal()
      .domain(activeKeys)
      .range("abcdefghijklmno".split("").map(function(a) {
        return "series-" + a;
      }));
    this._seriesName = seriesName;
    //
    var scrollLeftTimestamp = xScale.invert(scrollLeft); // scrollLefTime
    // 
    var visibleBoundary = [
      Math.max(xScale.invert(scrollLeft - chartWidth), xBoundary[0]),
      Math.min(xScale.invert(scrollLeft + chartWidth * 2), xBoundary[1])
    ];

    var activeDataset = [];
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
      activeDataset.push(filtered);
    });
    this._activeDataset = activeDataset;
    var yBoundary = d3.extent(function() {
      var arr = [];
      activeDataset.forEach(function(each) {
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
    var visibleDataset = activeDataset.filter(function(each) {
      if(visibleBoundary[0] <= each.timestamp && each.timestamp <= visibleBoundary[1]) {
        return true;
      }
      return false;
    });

    // Update Y Axis
    var yScale = d3.scaleLinear()
      .domain(yBoundary)
      .range([dataHeight, 0]);
    this._yScale = yScale;
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
              .tickSize(5)
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
    var segmentNodes = segmentsWrap.selectAll(".segment").nodes().reverse(); // Reverse() to start from the most recent segment
    var visibleSegmentsLefts = [ // Segments whose data range are in the visible boundaries
      scrollLeft - chartWidth, scrollLeft + chartWidth
    ];
    segmentNodes.forEach(function(node) {
      var seg = d3.select(node);
      var start = Number(node.getAttribute("data-start"));
      var end = Number(node.getAttribute("data-end"));
      var left = Number(node.getAttribute("data-left"));

      // Check Already Updated
      if(keysChanged === false && seg.selectAll("path").size() > 0) return;
      // Erase First
      node.innerHTML = "";

      // Check Visibility
      if(visibleSegmentsLefts[0] <= left && left <= visibleSegmentsLefts[1]) {
        // to be drawn
      } else {
        return;
      }

      // Segment Dataset
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

      // If No Data
      if(segDataGroups.length === 0) {
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
    // Regenerate points when keys were changed
    if(keysChanged === true) {
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
            .attr("cx", -100)
            .style("opacity", 0);
    }
  }

  // SCALE
  _scale() {
    let opt = Chart.options();
    let dataset = this.dataset;
    let firstT = dataset[0].timestamp;
    let lastT = dataset[dataset.length - 1].timestamp;
    let duration = lastT - firstT;
    let boundaries = [];
    let gtht = opt.gapThresholdTime * 60;
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

    let epsilon = 1e-5;
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
      let gapEnd = gapBoundaries[i*2+1] - epsilon; // In order to exclude the non-gap point at the gap end
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

    // xScale
    var create;
    create = function() {
      var $;
      $ = function(timestamp) {
        return $._(timestamp) * ($._range[1] - $._range[0]) + $._range[0];
      };
      $._steps = steps;
      $._lefts = lefts;
      $._boundaries = boundaries;
      $._tickValues = boundaries; // Default tick values
      $._firstT = firstT;
      $._lastT = lastT;
      $._totalDuration = totalDuration;
      $._ = function(timestamp) {
        for(let i = 0; i < $._boundaries.length - 1; i++) {
          let step = $._steps[i];
          let left = $._lefts[i];
          let leftT = $._boundaries[i];
          let rightT = $._boundaries[i+1];
          if(timestamp <= rightT) {
            return (timestamp - leftT) * step / $._totalDuration + left;
          }
        }
        // Beyond domain
        return 1 + (timestamp - $._lastT) / $._totalDuration;
      };
      $.copy = function() {
        var copy = create();
        copy.range($._range);
        return copy;
      }
      $.domain = function() {
        return [$._firstT, $._lastT];
      };
      $.range = function(range) {
        if(range === undefined) return $._range;
        $._range = range;
        return $;
      };
      $.normalize = function(x) {
        return (x - $._range[0]) / ($._range[1] - $._range[0]);
      };
      $.invert = function(x) {
        // Convert to 0 to 1
        let base = $.normalize(x);
        for(let i = 0; i < $._lefts.length - 1; i++) {
          let step = $._steps[i];
          let left = $._lefts[i];
          let right = $._lefts[i + 1];
          let leftT = $._boundaries[i];
          if(base <= right) {
            return (base - left) / step * $._totalDuration + leftT;
          }
        }
        // Beyond domain
        return (x - 1) * $._totalDuration + $._lastT;
      };
      $.ticks = function(n) {
        var tv = [];
        if(n <= 2) return;
        // Exclude 0 and 1
        for(var i = 1; i < n - 1; i++) {
          tv.push(
            $.invert((1 * i / (n - 1)) * ($._range[1] - $._range[0]) + $._range[0])
          );
        }
        $._tickValues = tv;
        return $;
      };
      $.tickValues = function() {
        return $._tickValues;
      };
      return $;
    };

    // Assign
    return create();

  }

// Public
  keys(arr) {
    if(arr === undefined) return this._keys;
    this._keys = arr;
    this.update();
    return this;
  }

  duration(d) {
    if(d === undefined) return this._duration;
    this._duration = d;
    this._draw();
    return this;
  }

// Static
  static options(opt) {
    if(opt === undefined) return _options;
    _options = Object.assign({}, _defaults, opt);
  }
  static remToPx(rem) {
    return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
  }
  static keysEqual(a, b) {
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
  static processDataset(rawDataset) {
    let opt = Chart.options();
    let gtht = opt.gapThresholdTime * 60;
    let dataMap = {};
    let dataset = [];
    { // Process Dataset
      for(let key in rawDataset) {
        let rows = rawDataset[key];
        for(let i = 0; i < rows.length; i++) {
          let prevRow = rows[i-1];
          let row = rows[i];
          // Gap
          if(prevRow !== undefined) {
            let diff = row.Timestamp - prevRow.Timestamp;
            if(diff > gtht) {
              // Put NaN
              let avgTimestamp = prevRow.Timestamp + diff / 2;
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

          let currMap = dataMap[row.Timestamp];
          if(currMap == null) {
            dataMap[row.Timestamp] = {
              [key]: row.Value // [] is needed to use key as variable rather than "key"
            };
          } else {
            dataMap[row.Timestamp][key] = row.Value;
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
  
    return dataset;
  }

}