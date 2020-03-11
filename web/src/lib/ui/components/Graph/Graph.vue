<template>
  <article class="graph"></article>
</template>

<script>

/*

Things Required for Plotting Graph
* Boundaries for gap analyzing
* Duration for adjusting visible amount of data
* Data for putting paths
* Color for choosing what color to use for paths
* Formatter for formatting values shown on tooltip
* Options for adjusting details of plotting

Structure
* Boundaries
  * Timestamps
* Duration
  * Seconds
* Dataset
  * Data
    * Timestamps
    * Values
  * Color
    * CSS-Style color
  * Formatter
    * num => formattedString
* Options
  * Long date and short date
  * xTick and yTick count

Plot (dataset)
* Draw

Draw
* Draw grid and ticks
* Draw y ticks
* Plot visible

Plot Visible
* Draw paths and circles in the visible part

Recolor (key, newColor)
* Change the color of paths and circles that are already drawn

Update Data (key, data)
* Replace data
* Draw

Set Formatter (key, formatter)


*/

// Static functions
function r2px(n) {return remToPx(n);}
function remToPx(rem) {
  return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
}

function keysEqual(a, b) {
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

function addThrottledAsyncEvent(elem, type, handler, interval) {
  var wrap = function(event) {
    elem.removeEventListener(type, wrap);
    handler(event).then(function() {
      setTimeout(function() {
        elem.addEventListener(type, wrap);
      }, interval);
    });
  };
  elem.addEventListener(type, wrap);
}

// D3
import {event, select, mouse, customEvent} from "d3-selection";
import {axisBottom, axisLeft} from "d3-axis";
import {line} from "d3-shape";
import {scaleLinear} from "d3-scale";
import {extent, bisector} from "d3-array";

const d3 = {get event() {return event;}, select, axisBottom, axisLeft, extent, scaleLinear, bisector, line, mouse, customEvent};

export default {
  name: "Graph",
  props: {
  },
  data() {
    this._options = {
      // TODO $root webcfg is not part of this library
      formatDateShort: this.$root.webCfg["format.date.short"],
      formatDateLong: this.$root.webCfg["format.date.long"]
    };
    return {
      dataset:    {},
      duration:   undefined,
      boundaries: undefined
    };
  },

  watch: {
    duration() {this._draw();},
    boundaries() {this._draw();}
  },

  computed: {
    keys() {
      return Object.keys(this.dataset);
    }
  },

  methods: {

    plot(dataset) {
      this.dataset = dataset;
      this._draw();
    },

    recolor(key, color) {
      d3.select(this._points[key]).attr("fill", color);
      this._lines[key].forEach(
        line => d3.select(line).attr("stroke", color)
      );
    },

    updateData(key, data) {
      this.dataset[key].data = data;
      this._draw();
    },

    setFormatter(key, fmt) {
      this.dataset[key].formatter = fmt;
    },

    async _draw() {

      // Check vars
      if(this.boundaries == undefined || this.duration == undefined)
        return;

      // xScale and Boundaries
      this._xBoundary   = d3.extent(this.boundaries);
      this._priorXScale = this._xScale;
      this._xScale      = this._scale();

      // Accessing Purposes
      var $ = this;
      
    // Vars
      var graph         = d3.select(this.$el);
      var graphNode     = graph.node();
      var graphDuration = this.duration * 60; // Into seconds
      var graphMargin   = {
        top: remToPx(0.5),
        left: remToPx(3),
        bottom: remToPx(2)
      };
      var graphRect = {
        width: graphNode.offsetWidth - graphMargin.left,
        height: graphNode.offsetHeight - graphMargin.top - graphMargin.bottom
      };
      var xScale    = this._xScale;
      var xBoundary = this._xBoundary;
      // Total x-axis duration
      var xDuration = xScale.totalDuration;
      // Prevent duration being longer than actual data
      graphDuration  = Math.min(graphDuration, xDuration);
      // Total width needed for data plotting
      var dataWidth  = graphRect.width * xDuration / graphDuration;
      // Plotted data height
      var dataHeight = graphRect.height;
      var dw_cw      = dataWidth / graphRect.width;
      // Total number of segments
      var segNo      = Math.ceil(dw_cw);
      // Total count of x-axis ticks
      var xTicks     = Math.round(dw_cw * 4);
      // Y-axis ticks count
      var yTicks     = 4;
      this._dataWidth     = dataWidth;
      this._dataHeight    = dataHeight;
      this._width         = graphRect.width;
      this._height        = graphRect.height;
      this._margin        = graphMargin;
      this._xDuration     = xDuration;
      this._xTicks        = xTicks;
      this._yTicks        = yTicks;
      this._graphDuration = graphDuration;
      // Prior: Reset at draw
      this._priorKeys     = null;
    
    // Element Var
      // Scroll left is max(0, total width minus graph width)
      var scrollLeft = Math.max(0, dataWidth - graphRect.width);
      // X of hand is middle of the graph
      var handX      = scrollLeft + graphRect.width / 2;

    // Axis Range
      // Set the output range for x scale: 0 to total width
      xScale.range([0, dataWidth]);
    
    // PRIOR VALUES
      var prior = false;
      {
        var segmentsWrap = graph.select(".segments-wrap");
        var priorHandXEnRectPercent;
        var priorHandT;

        // If segmentsWrap already exists
        if(segmentsWrap.size() > 0) {
          prior = true;
    
          var node        = segmentsWrap.node();
          var hand        = segmentsWrap.select(".hand");
          var priorHandX  = Number(hand.attr("x1"));
          var priorXScale = this._priorXScale;
    
          // Hand position relative to the visible rect
          priorHandXEnRectPercent = (priorHandX - node.scrollLeft) / node.offsetWidth;
          // Find timestamp for the hand x position
          priorHandT = priorXScale.invert(priorHandX);
    
          // Restore Scroll Left and Hand X
          handX      = xScale(priorHandT);
          scrollLeft = Math.max(0, handX - graphRect.width * priorHandXEnRectPercent);

          // When the hand is outside the graph, default to default position
          if(!(scrollLeft <= handX && handX <= scrollLeft + graphRect.width)) {
            handX = scrollLeft + graphRect.width / 2;
          }
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
      graph.node().innerHTML = "";
    
    // Background
      var background = graph.append("div")
        .attr("class", "background-wrap")
        .style("width",  `${graphRect.width}px`)
        .style("height", `${graphRect.height}px`)
        .style("left",   `${graphMargin.left}px`)
        .style("top",    `${graphMargin.top}px`)
        .append("div")
          .attr("class", "background-container")
          .append("svg")
            .attr("width",  graphRect.width)
            .attr("height", graphRect.height)
            .append("g")
              .attr("class", "background");
      // Disabled for now
      var focusDate = background.append("g")
        .attr("class", "focus-date")
        .append("text")
          .attr("text-anchor", "middle")
          .attr("x", (graphRect.width + graphMargin.left) / 2 - graphMargin.left)
          .attr("y", graphRect.height / 2)
          .attr("dy", "25%")
          .attr("font-size", graphRect.height / 1.5)
          .style("opacity", 0);
    
    // Segments
      var segmentsWrap = graph.append("div")
        .attr("class", "segments-wrap")
        .style("width",  `${graphRect.width}px`)
        .style("height", `${graphRect.height + graphMargin.top + graphMargin.bottom}px`)
        .style("left",   `${graphMargin.left}px`)
        .style("top",    `0px`);
      var segmentsContainer = segmentsWrap.append("div")
        .attr("class", "segments-container");
      var segments = segmentsContainer.append("svg")
        .attr("width",  dataWidth)
        .attr("height", dataHeight)
        .append("g")
          .attr("class",     "segments")
          .attr("transform", `translate(0, ${graphMargin.top})`);
      // Set Scroll Left
      segmentsWrap.node().scrollLeft = scrollLeft;
    
    // Axis
      var xAxis = segments.append("g")
        .attr("class",     "axis x-axis")
        .attr("transform", `translate(0, ${graphRect.height})`)
        .call(d3.axisBottom(xScale)
          .tickValues(xScale.ticks(xTicks).tickValues())
          .tickSizeOuter(0)
          .tickFormat(timestamp => timestamp.date($._options.formatDateShort)))
            .attr("font-family", "")
            .attr("font-size",   "");
    
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
    
    // For each segment
      {
        for(var i = 0; i < segNo; i++) {
          // Pre ---
          var segLeft  = graphRect.width * i;
          // Start is the timestamp for the start of this segment
          var segStart = xScale.invert(segLeft);
          // End is the timestamp for the end
          var segEnd   = Math.min(xScale.invert(segLeft + graphRect.width), xBoundary[1]);
          // Main ---
          var seg = segments.append("g")
            .attr("class", "segment")
            .attr("data-start", segStart)
            .attr("data-end",   segEnd)
            .attr("data-left",  segLeft);
          // Post ---
        }
      }
    
    // Overlay
      // Overlay is for everything that is above the plotted data
      var overlay = graph.append("div")
        .attr("class", "overlay-wrap")
        .style("width",  `${graphRect.width}px`)
        .style("height", `${graphRect.height}px`)
        .style("left",   `${graphMargin.left}px`)
        .style("top",    `${graphMargin.top}px`)
        .append("div")
          .attr("class", "overlay-container")
          .append("svg")
            .attr("width",  graphRect.width)
            .attr("height", graphRect.height)
            .append("g")
              .attr("class", "overlay");
    // Tooltip // no pointer events
      var tooltipSize = {width: r2px(13), height: r2px(2.625)};
      //       | 0.375 6/16
      //       | 0.875 14/16
      // 0.375 | 0.25  4/16
      //       | 0.75  12/16
      //       | 0.375 6/16
      var tooltip = overlay.append("g")
        .attr("class", "tooltip")
        .style("opacity", 0);
      tooltip.append("rect")
        .attr("class", "background")
        .attr("height", tooltipSize.height)
        .attr("width",  tooltipSize.width)
        .attr("x", 0)
        .attr("y", 0);
      var tooltipIcon = tooltip.append("rect")
        .attr("height", r2px(14/16))
        .attr("width",  r2px(14/16))
        .attr("x",      r2px(6/16))
        .attr("y",      r2px(6/16));
      var tooltipValue = tooltip.append("text")
        .attr("class", "value")
        .attr("text-anchor", "middle")
        .attr("x",  r2px(26/16))
        .attr("y",  r2px(6/16))
        .attr("dy", r2px(12/16));
      var tooltipTimestamp = tooltip.append("text")
        .attr("class", "timestamp")
        .attr("text-anchor", "middle")
        .attr("x",  r2px(6/16))
        .attr("y",  r2px(24/16))
        .attr("dy", r2px(11/16));
    
    // Draw Y Scale
      var yBoundary = d3.extent(function() {
        var arr = [];
        for(let key in $.dataset) {
          var {data} = $.dataset[key];
          data.forEach(i => arr.push(i.value));
        }
        return arr;
      }());
      if(yBoundary[0] == undefined)         yBoundary = [0, 1];
      else if(yBoundary[0] == yBoundary[1]) yBoundary[0] = 0;

      // Y Scale
      var yScale = d3.scaleLinear()
        .domain(yBoundary)
        .range([dataHeight, 0]);
      this._yScale = yScale;
      var yAxis = graph.append("div")
        .attr("class", "axis-wrap y-axis-wrap")
        .append("div")
          .attr("class", "axis-container y-axis-container")
          .append("svg")
            .attr("height", graphRect.height + graphMargin.top + graphMargin.bottom)
            .append("g")
              .attr("class", "axis y-axis")
              .attr("transform", `translate(${graphMargin.left}, ${graphMargin.top})`)
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

      // Draw Grid
      graph.select(".grid").remove();
      var grid = projection.append("g")
        .attr("class", "grid")
        .call(d3.axisLeft(yScale)
          .ticks(yTicks)
          .tickSize(-dataWidth)
          .tickFormat(""));

      // Paths
      this._lines = {};
      this.keys.forEach(key => $._lines[key] = []);

      // Points
      this._points = {};
      var circles = segments.append("g")
        .attr("class", "circles");
      circles.selectAll("circles")
        .data(this.keys)
        .enter()
        .append("circle")
          .each(function(key) {$._points[key] = this;})
          .attr("fill",     key => $.dataset[key].color)
          .attr("stroke",   "none")
          .attr("r",        4)
          .attr("cx",       -100)
          .style("opacity", 0);

    // Plot Visible Parts
      await this._plotVisible();
    
    // EVENTS ---
      // Segments Wrap Scroll
      addThrottledAsyncEvent(
        segmentsWrap.node(), "scroll", $._plotVisible, 10
      );
      // Window Resize
      addThrottledAsyncEvent(
        window, "resize", $._draw, 100
      );

      { // Hand, Points and Tooltip and Touch Interface
        var isMouseDown = false;
        var isTouch     = false;
        // Bisect is used to get the nearest point to a timestamp
        var bisect = function(slice, timestamp, accessor) {
          var bs = d3.bisector(accessor).left;
          var i   = bs(slice, timestamp);
          var d0  = slice[i-1];
          var d1  = slice[i];
          if(d0 === undefined && d1 === undefined) return undefined;
          else if(d0 === undefined) return d1;
          else if(d1 === undefined) return d0;
          else return timestamp - accessor(d0) > accessor(d1) - timestamp ? d1 : d0;
        };

        // Moving hand
        var mouseHandler = function() {

          var event      = d3.event;
          var target     = event.target;
          var onPoint    = target.hasClass("point");
          var mouse      = d3.mouse(projection.node());
          var [mX, mY]   = mouse;
          var timestamp  = xScale.invert(mX);
          var dataset    = $.dataset;
          var yScale     = $._yScale;
          var scrollLeft = segmentsWrap.node().scrollLeft;

          // Event Type Check
          if(event.type === "mousemove" && !(isTouch || isMouseDown))
            return;

          // Make Points Visible
          segments.selectAll(".point").style("opacity", 1);
          background.select(".focus-date text").style("opacity", 1);

          // If mouse is on a point, show the tooltip
          if(onPoint) {
            overlay.selectAll(".tooltip").style("opacity", 1);
          }

          //
          var handX;
    
          // Points
          for(let key in dataset) {
            let {data} = dataset[key];
            var elem   = bisect(data, timestamp, function(d) { return d.timestamp; });
            // No nearest found, return
            if(elem === undefined || isNaN(elem.value)) return;
            var cX = xScale(elem.timestamp);
            var cY = yScale(elem.value);
            d3.select($._points[key])
              .attr("data-timestamp", elem.timestamp)
              .attr("data-value",     elem.value)
              .attr("cx", cX)
              .attr("cy", cY);

            // Hand X is nearest point to the mouse from all data
            if(handX === undefined) handX =  cX;
            else if(Math.abs(handX - mX) > Math.abs(cX - mX)) handX = cX;
          }

          // No nearest X found
          if(handX === undefined) return;
          
          // Hand timestamp
          var handT = xScale.invert(handX);
          // Move hand
          hand.attr("x1", handX).attr("x2", handX);

          // Time (disabled for now)
          // background.select(".focus-date text").text(handT.date("MM/DD"));
    
          // Tooltip
          var [tw, th] = [tooltipSize.width, tooltipSize.height];
          var tx       = Math.min(Math.max(mX - scrollLeft - tw/2, 0), graphRect.width - tw);
          var ty       = mY < th ? mY + 5 : mY - th - 5;
          tooltip.attr("transform", `translate(${tx},${ty})`);
          if(onPoint) {
            let ttKey = target.getAttribute("data-key");
            let ttVal = Number(target.getAttribute("data-value"));
            let ttTs  = Number(target.getAttribute("data-timestamp"));
            let ttCl  = $.dataset[ttKey].color;
            let ttFmt = $.dataset[ttKey].formatter;
            tooltipValue.text(ttFmt(ttVal));
            tooltipTimestamp.text(
              // TODO prototype functions are not part of library
              ttTs.date($._options.formatDateLong)
            );
            tooltipIcon.style("fill", ttCl);
          }
        };
    
        //
        { // Touch Interface
          window.addEventListener("touchstart", function handler() {
            isTouch = true;
            window.removeEventListener("touchstart", handler);
          });
          // Dispatch mouse event as you scroll
          var node     = segmentsWrap.node();
          var lastLeft = node.scrollLeft;
          var handler  = async function(event) {
            if(isTouch) {
              var left  = node.scrollLeft;
              var rect  = node.getBoundingClientRect();
              var handX = Number(hand.attr("x1"));
              // Arbitrary Coords
              event.clientX = (handX - left) + (left - lastLeft) + rect.left;
              event.clientY = rect.top + rect.height / 2;
              lastLeft      = left;
              d3.customEvent(event, mouseHandler);
            }
          }
          addThrottledAsyncEvent(node, "scroll", handler, 1);
        }
    
        // Add Handlers

        // https://github.com/d3/d3-selection/blob/v1.4.1/README.md#selection_on
        // The type may be optionally followed by a period (.) and a name;
        // the optional name allows multiple callbacks to be registered to
        // receive events of the same type, such as click.foo and click.bar.
        // To specify multiple typenames, separate typenames with spaces,
        // such as input change or click.foo click.bar.
        graph
          .on("mouseout", function() {
            var event = d3.event;
            background.select(".focus-date text").style("opacity", 0);
            if(event.target.hasClass("point")) {
              overlay.selectAll(".tooltip").style("opacity", 0);
            }
          })
          .on("mousemove",   mouseHandler)
          .on("mousedown.a", mouseHandler)
          .on("mousedown.b", () => {isMouseDown = true;})
          .on("mouseup",     () => {isMouseDown = false;});
      }

    },

    _scale() {

      // Custom scale is used to make the length of all gaps the same
      // in order to reduce the whitespace made by long-term gaps
      let epsilon       = 1e-5;
      let boundaries    = this.boundaries;
      let firstT        = boundaries[0];
      let lastT         = boundaries[boundaries.length - 1];
      let duration      = lastT - firstT;
      let segNo         = boundaries.length / 2;
      let gapNo         = segNo - 1;
      let gapBoundaries = boundaries.slice(1, -1);
      
      // Exclude Gap Duration from duration
      for(let i = 0; i < gapBoundaries.length; i+=2) {
        let start  = gapBoundaries[i];
        let end    = gapBoundaries[i+1];
        duration  -= end - start;
      }

      // |     total duration      |
      // | duration | gap duration |
      let gapEachDuration = (this.duration * 60) / 9; // To seconds
      let totalDuration   = gapEachDuration * gapNo + duration;
      let steps           = [];
      let lefts           = [];
      let lastRight       = 0;
      let lastRightT      = firstT;
      //
      for(let i = 0; i < gapNo; i++) {
        let gapStart = gapBoundaries[i*2];
        let gapEnd   = gapBoundaries[i*2+1] - epsilon; // In order to exclude the non-gap point at the gap end
        //
        let step  = gapEachDuration / (gapEnd - gapStart);
        let left  = lastRight + (gapStart - lastRightT) / totalDuration;
        let right = left + gapEachDuration / totalDuration;
        //
        lastRight  = right;
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
        // Private
        $._steps      = steps;
        $._lefts      = lefts;
        $._boundaries = boundaries;
        $._tickValues = boundaries; // Default tick values
        $._firstT     = firstT;
        $._lastT      = lastT;
        $._ = function(timestamp) {
          for(let i = 0; i < $._boundaries.length - 1; i++) {
            let step   = $._steps[i];
            let left   = $._lefts[i];
            let leftT  = $._boundaries[i];
            let rightT = $._boundaries[i+1];
            if(timestamp <= rightT) {
              return (timestamp - leftT) * step / $.totalDuration + left;
            }
          }
          // Beyond domain
          return 1 + (timestamp - $._lastT) / $.totalDuration;
        };
        // Public
        $.duration      = duration;
        $.totalDuration = totalDuration;
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
            let step  = $._steps[i];
            let left  = $._lefts[i];
            let right = $._lefts[i + 1];
            let leftT = $._boundaries[i];
            if(base <= right) {
              return (base - left) / step * $.totalDuration + leftT;
            }
          }
          // Beyond domain
          return (x - 1) * $.totalDuration + $._lastT;
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

    },

    async _plotVisible() {

      // Shorthand
      var $ = this;

      // Graph
      var graph         = d3.select(this.$el);
      var priorKeys     = this._priorKeys;
      var xDuration     = this._xDuration;
      var xBoundary     = this._xBoundary;
      var yBoundary     = this._yBoundary;
      var graphWidth    = this._width;
      var graphHeight   = this._height;
      var graphMargin   = this._margin;
      var dataWidth     = this._dataWidth;
      var dataHeight    = this._dataHeight;
      var graphDuration = this._graphDuration;
      var xTicks        = this._xTicks;
      var yTicks        = this._yTicks;
      // ELements
      var xScale        = this._xScale;
      var yScale        = this._yScale;
      var projection    = this._projection;

      // Info Set
      this._priorKeys = this.keys.slice(0);

      // Segments
      var segmentsWrap = graph.select(".segments-wrap");
      var scrollLeft   = segmentsWrap.node().scrollLeft;

      // Dataset
      //
      var scrollLeftTimestamp = xScale.invert(scrollLeft); // scrollLefTime
      // 
      var visibleBoundary = [
        Math.max(xScale.invert(scrollLeft - graphWidth), xBoundary[0]),
        Math.min(xScale.invert(scrollLeft + graphWidth * 2), xBoundary[1])
      ];

      // Visible Dataset
      var visibleDataset = {};
      for(let key in this.dataset) {
        visibleDataset[key] = this.dataset[key].data.filter(
          i => visibleBoundary[0] <= i.timestamp && i.timestamp <= visibleBoundary[1]
        );
      }

      // Segments Each
      // Loop
      var segments     = segmentsWrap.select(".segments");
      var segmentNodes = segmentsWrap.selectAll(".segment").nodes().reverse(); // Reverse() to start from the most recent segment
      var visibleSegmentsLefts = [ // Segments whose data range are in the visible boundaries
        scrollLeft - graphWidth, scrollLeft + graphWidth
      ];
      segmentNodes.forEach(function(node) {
        var seg   = d3.select(node);
        var start = Number(node.getAttribute("data-start"));
        var end   = Number(node.getAttribute("data-end"));
        var left  = Number(node.getAttribute("data-left"));

        // Check Already Drawn
        if(seg.selectAll("path").size() > 0) return;

        // Check Visibility
        if(!(visibleSegmentsLefts[0] <= left && left <= visibleSegmentsLefts[1]))
          return;

        // Segment Dataset
        var segDataGroups = [];
        for(let key in visibleDataset) {
          let data = visibleDataset[key].filter(i => {
            return start <= i.timestamp && i.timestamp <= end;
          });
          if(data.length > 0) {
            segDataGroups.push({key, data});
          }
        }

        // If No Data
        if(segDataGroups.length === 0)
          return;

        // Add Path
        seg.selectAll("paths")
          .data(segDataGroups)
          .enter()
          .append("path")
            .each(function(d) {$._lines[d.key].push(this);})
            .attr("stroke-width", 1)
            .attr("stroke", d => $.dataset[d.key].color)
            .attr("fill",   "none")
            .attr("d", d => d3.line()
                .defined(e => !isNaN(e.value))
                .x(e => xScale(e.timestamp))
                .y(e => yScale(e.value))
                (d.data)
            );

      });

    }

  }
}
</script>