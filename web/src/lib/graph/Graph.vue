<template>
  <article class="ui-graph"></article>
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


TODO: make line stroke width 2 during mouse move

*/

// Const
const defaultOptions = {
  accessors: {
    x: d => d.x,
    y: d => d.y
  },
  formatters: {
    xAxis: x => x,
    yAxis: y => y,
    x: d => d.x,
    y: d => d.y
  }
};

// Static functions
function r2px(n) {return remToPx(n);}
function remToPx(rem) {
  return rem * parseFloat(getComputedStyle(document.documentElement).fontSize);
}
function sliceFromTo(arr, from, to, accessor = e => e) {
  let fromIdx = arr.findIndex(i => accessor(i) >= from);
  let toIdx   = arr.slice(fromIdx).findIndex(i => accessor(i) >= to);

  toIdx = toIdx === -1
    ? (arr.length ? arr.length : 0)
    : toIdx + fromIdx;
  fromIdx = fromIdx > 0
    ? fromIdx - 1
    : 0;
  return arr.slice(fromIdx, toIdx + 1); // slice doesn't include end
}

// TODO: make this independent from external code
//  + change timestamp to x
//  + change value to y
//  + format -- not as string but as function

// Util
function addThrottledAsyncEvent(elem, type, handler, interval) {
  let running = false;
  let wrap = function(event) {
    if(!running) {
      running = true;
      handler(event).then(function() {
        setTimeout(function() {
          running = false;
        }, interval);
      });
    }
  };
  elem.addEventListener(type, wrap);
}

function addDebouncedAsyncEvent(elem, type, handler, interval) {
  let timer;
  let wrap = function(event) {
    clearTimeout(timer);
    timer = setTimeout(() => handler(event), interval);
  }
  elem.addEventListener(type, wrap);
}

function d3SyncThrottle(handler, interval) {
  let running = false;
  return function() {
    if(!running) {
      running = true;
      let context = this,
        args = arguments;
      
      handler.apply(context, args);
      setTimeout(function() {
        running = false;
      }, interval);
    }
  };
}

function d3SyncDebounce(handler, interval) {
  let timer = null;
  return function() {
    let context = this,
      args = arguments,
      evt = d3.event;
    clearTimeout(timer);
    timer = setTimeout(function() {
      d3.event = evt;
      handler.apply(context, args);
    }, interval);
  };
}

// D3
import {event, selectAll, select, mouse, customEvent} from "d3-selection";
import {axisBottom, axisLeft} from "d3-axis";
import {line} from "d3-shape";
import {scaleLinear} from "d3-scale";
import {extent, bisector} from "d3-array";

const d3 = {get event() {return event;}, selectAll, select, axisBottom, axisLeft, extent, scaleLinear, bisector, line, mouse, customEvent};

export default {

  name: "Graph",

  props: {
    boundaries: {type: Array,  default: []},
    dataset:    {type: Object, default: {}},
    duration:   Number,
    model:      {type: Object, default: {}},
    options:    {type: Object, default: {}}
  },

  // v-model
  // This is for accessing focused key-values from parent components
  model: {
    prop: "model",
    event: "change"
  },

  watch: {
    duration()   {this._draw();},
    boundaries() {this._draw();},
    dataset()    {this._draw();},
    options()    {this._draw();}
  },

  data() {
    return {
      segmentIdAUtoIncrement: 0
    };
  },

  computed: {
    keys() {
      return Object.keys(this.dataset);
    },
    computedOptions() {
      return Object.assign({}, defaultOptions, this.options);
    }
  
  },

  mounted() {
    // Window Resize
    // Add at mounted to prevent multiple event listeners
    addDebouncedAsyncEvent(
      window, "resize", this._draw, 100
    );
    this._draw();
  },

  methods: {

    asx(d) { // x accessor
      return this.computedOptions.accessors.x(d);
    },
    asy(d) { // y accessor
      return this.computedOptions.accessors.y(d);
    },

    cachedDataset() {
      
      let $ = this;
      let graph         = d3.select(this.$el);
      let segmentsWrap = graph.select(".segments-wrap");
      let segments     = segmentsWrap.select(".segments");
      let segmentNodes = segmentsWrap.selectAll(".segment").nodes();
      let cachedDataset = {};

      for(let key in $.dataset) {
        cachedDataset[key] = [];
      }

      segmentNodes.forEach(function(node) {
        if(!node.hasAttribute("data-id")) return;

        let segId          = node.getAttribute("data-id");
        let segmentDataset = $._segmentsDataset[segId];
        for(let key in $.dataset) {
          cachedDataset[key] = cachedDataset[key].concat(segmentDataset[key]);
        }
      });

      return cachedDataset;
      
    },

    async _draw() {

      // Validate variables
      if(this.boundaries == undefined || this.boundaries.length < 2)
        return;

      if(this.duration == undefined || this.duration == 0)
        return;

      // Accessing purposes
      let $ = this;
      
      // Vars
      let graph         = d3.select(this.$el);
      let graphNode     = graph.node();
      let graphDuration = this.duration;
      let graphMargin   = {
        top: remToPx(0.5),
        left: remToPx(3),
        bottom: remToPx(2)
      };
      let graphRect = {
        width:  graphNode.offsetWidth - graphMargin.left,
        height: graphNode.offsetHeight - graphMargin.top - graphMargin.bottom
      };

      // Scale and boudaries
      this._xBoundary   = d3.extent(this.boundaries);
      let xBoundary     = this._xBoundary;
      this._priorXScale = this._xScale;
      this._xScale      = this._scale();
      let xScale        = this._xScale;

      // Total x-axis duration
      let xDuration = xScale.totalDuration;

      // Prevent duration being longer than actual data
      graphDuration = Math.min(graphDuration, xDuration);
      this._xDuration     = xDuration;
      this._graphDuration = graphDuration;

      // Total width needed for data plotting
      let dataWidth  = graphRect.width * xDuration / graphDuration;
      // Plotted data height
      let dataHeight = graphRect.height;
      let dw_cw      = dataWidth / graphRect.width;
      // Total number of segments
      let segNo      = Math.ceil(dw_cw);
      // Total count of x-axis ticks
      let xTicks     = Math.round(dw_cw * 4);
      // Y-axis ticks count
      let yTicks     = 4;

      // Assign for later use
      this._dataWidth  = dataWidth;
      this._dataHeight = dataHeight;
      this._width      = graphRect.width;
      this._height     = graphRect.height;
      this._margin     = graphMargin;
      this._xTicks     = xTicks;
      this._yTicks     = yTicks;
    
      // Scroll left and graph hand
      // Scroll left is max(0, total width minus graph width)
      let scrollLeft = Math.max(0, dataWidth - graphRect.width);
      // X of hand defaults to middle of the graph
      let handX      = scrollLeft + graphRect.width / 2;

      // Axis range
      // Set the output range for x scale: 0 to total width
      xScale.range([0, dataWidth]);
    
      // PRIOR VALUES
      let prior = false;
      {
        let segmentsWrap = graph.select(".segments-wrap");
        // Percent of hand position relative to visible rect
        let priorHandXEnRectPercent;
        let priorHandT;

        // If segmentsWrap already exists, it means it was drawn before
        if(segmentsWrap.size() > 0) {
          prior = true;
    
          let node        = segmentsWrap.node();
          let hand        = segmentsWrap.select(".hand");
          let priorHandX  = + hand.attr("x1");
          // Prior x scale is required for inverting prior position
          let priorXScale = this._priorXScale;
    
          // Hand position relative to the visible rect
          priorHandXEnRectPercent = (priorHandX - node.scrollLeft) / node.offsetWidth;
          // Find timestamp for the hand x position
          priorHandT = priorXScale.invert(priorHandX);
    
          // Restore scroll left and hand x position
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
      //   + paths
      // + circles
      // + tooltip
    
      // Elements
      // Erase Elements first
      graph.node().innerHTML = "";
    
      // Background
      let background = graph.append("div")
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
    
      // Segments
      let segmentsDataset = {};
      $._segmentsDataset = segmentsDataset;
      let segmentsWrap = graph.append("div")
        .attr("class",   "segments-wrap")
        .style("width",  `${graphRect.width}px`)
        .style("height", `${graphRect.height + graphMargin.top + graphMargin.bottom}px`)
        .style("left",   `${graphMargin.left}px`)
        .style("top",    `0px`);
      let segmentsContainer = segmentsWrap.append("div")
        .attr("class", "segments-container");
      let segments = segmentsContainer.append("svg")
        .attr("width",  dataWidth)
        .attr("height", dataHeight)
        .append("g")
          .attr("class",     "segments")
          .attr("transform", `translate(0, ${graphMargin.top})`);
      // Set scroll left
      segmentsWrap.node().scrollLeft = scrollLeft;
    
      // Axis
      let xAxis = segments.append("g")
        .attr("class",     "axis x-axis")
        .attr("transform", `translate(0, ${graphRect.height})`)
        .call(d3.axisBottom(xScale)
          .tickValues(xScale.ticks(xTicks).tickValues())
          .tickSizeOuter(0)
          .tickFormat($.computedOptions.formatters.xAxis))
            .attr("font-family", "")
            .attr("font-size",   "");
    
      // Projection (entire path domain)
      let projection = segments.append("g")
        .attr("class", "projection");
      projection.append("rect")
        .attr("class", "projection-domain")
        .attr("x", 0)
        .attr("y", 0)
        .attr("width",  dataWidth)
        .attr("height", dataHeight);
      this._projection = projection;
      // Hand
      let hand = projection.append("line")
        .attr("class", "hand")
        .attr("x1", handX)
        .attr("x2", handX)
        .attr("y1", 0)
        .attr("y2", dataHeight);
    
      // For each segment
      {
        for(let i = 0; i < segNo; i++) {
          // Pre ---
          let segLeft  = graphRect.width * i;
          // Start is the timestamp for the start of this segment
          let segStart = xScale.invert(segLeft);
          // End is the timestamp for the end
          let segEnd   = Math.min(xScale.invert(segLeft + graphRect.width), xBoundary[1]);
          // Main ---
          let seg = segments.append("g")
            .attr("class", "segment")
            .attr("data-start", segStart)
            .attr("data-end",   segEnd)
            .attr("data-left",  segLeft);
        }
      }
    
      // Overlay
      // It is for everything that is above the plotted data
      let overlay = graph.append("div")
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

      // Tooltip
      // no pointer events
      let tooltipSize = {width: r2px(13), height: r2px(2.625)};
      //       | 0.375 6/16
      //       | 0.875 14/16
      // 0.375 | 0.25  4/16
      //       | 0.75  12/16
      //       | 0.375 6/16
      let tooltip = overlay.append("g")
        .attr("class", "tooltip")
        .style("opacity", 0);
      tooltip.append("rect")
        .attr("class",  "background")
        .attr("height", tooltipSize.height)
        .attr("width",  tooltipSize.width)
        .attr("x", 0)
        .attr("y", 0);
      let tooltipIcon = tooltip.append("rect")
        .attr("height", r2px(14/16))
        .attr("width",  r2px(14/16))
        .attr("x",      r2px(6/16))
        .attr("y",      r2px(6/16));
      let tooltipY = tooltip.append("text")
        .attr("class", "y")
        .attr("text-anchor", "left")
        .attr("x",  r2px(26/16))
        .attr("y",  r2px(6/16))
        .attr("dy", r2px(12/16));
      let tooltipX = tooltip.append("text")
        .attr("class", "x")
        .attr("text-anchor", "left")
        .attr("x",  r2px(6/16))
        .attr("y",  r2px(24/16))
        .attr("dy", r2px(11/16));
    
      // Draw y scale
      let yBoundary = d3.extent(function() {
        let arr = [];
        for(let key in $.dataset) {
          let {data, minMaxes} = $.dataset[key];
          let mm = minMaxes.y;

          if(mm !== undefined) {
            // if min max is present
            arr.push(mm.min, mm.max);
          } else if(data !== undefined) {
            // if data is present
            data.forEach(i => arr.push(
              $.asy(i)
            ));
          }
        }
        return arr;
      }());
      if(yBoundary[0] == undefined)         yBoundary = [0, 1];
      else if(yBoundary[0] == yBoundary[1]) yBoundary[0] = 0;

      // Y scale
      let yScale = d3.scaleLinear()
        .domain(yBoundary)
        .range([dataHeight, 0]);
      this._yScale = yScale;
      let yTickValues = yScale.ticks(5).slice(1, -1);
      let yAxis = graph.append("div")
        .attr("class", "axis-wrap y-axis-wrap")
        .append("div")
          .attr("class", "axis-container y-axis-container")
          .append("svg")
            .attr("height", graphRect.height + graphMargin.top + graphMargin.bottom)
            .append("g")
              .attr("class", "axis y-axis")
              .attr("transform", `translate(${graphMargin.left}, ${graphMargin.top})`)
              .call(d3.axisLeft(yScale)
                .tickValues(yTickValues)
                .tickSize(5)
                .tickSizeOuter(0)
                .tickFormat($.computedOptions.formatters.yAxis))
                .attr("font-family", "")
                .attr("font-size", "");

      // Draw Grid
      graph.select(".grid").remove();
      let grid = projection.append("g")
        .attr("class", "grid")
        .call(d3.axisLeft(yScale)
          .tickValues(yTickValues)
          .tickSize(-dataWidth)
          .tickFormat(""));

      // Paths
      this._lines = {};
      this.keys.forEach(key => $._lines[key] = []);

      // Points
      this._points = {};
      let circles = segments.append("g")
        .attr("class", "circles");
      circles.selectAll("circles")
        .data(this.keys)
        .enter()
        .append("circle")
          .each(function(key) {$._points[key] = this;})
          .attr("class",    "point")
          .attr("data-key", key => key)
          .attr("fill",     key => $.dataset[key].color)
          .attr("stroke",   "none")
          .attr("r",        4)
          .attr("cx",       -100);

      // Plot Visible Parts
      await this._plotVisible();
    
      // EVENTS ---
      // Segments Wrap Scroll
      addThrottledAsyncEvent(
        segmentsWrap.node(), "scroll", $._plotVisible, 10
      );

      { // Hand, Points and Tooltip and Touch Interface
        let isMouseDown = false;
        let isTouch     = false;

        // Bisect is used to get the nearest point to a timestamp
        let bisect = function(slice, x, accessor) {
          let bs = d3.bisector(accessor).left;
          let i   = bs(slice, x);
          let d0  = slice[i-1];
          let d1  = slice[i];
          if(d0 === undefined && d1 === undefined) return undefined;
          else if(d0 === undefined) return d1;
          else if(d1 === undefined) return d0;
          else return x - accessor(d0) > accessor(d1) - x ? d1 : d0;
        };

        // Moving hand
        let mouseHandler = function() {

          let event      = d3.event;
          let target     = event.target;
          let onPoint    = target.classList.contains("point");
          // No onLine events as paths trigger mouse events in a rectangle shape.
          let mouse      = d3.mouse(projection.node());
          let [mX, mY]   = mouse;
          let x          = xScale.invert(mX);
          let dataset    = $.dataset;
          let yScale     = $._yScale;
          let scrollLeft = segmentsWrap.node().scrollLeft;
          let newModel   = {};

          // Reset stroke width
          d3.selectAll(".line")
            .attr("stroke-width", 1);

          // If mouse is on a point
          if(onPoint) {
            // Key
            let key = target.getAttribute("data-key");

            // Show the tooltip
            overlay.selectAll(".tooltip").style("opacity", 1);

            // Thicken
            $._lines[key].forEach(
              line => d3.select(line)
                .attr("stroke-width", 2)
            );
          }

          //
          let handX;
    
          // Points
          let cachedDataset = $._cachedDataset;
          for(let key in cachedDataset) {
            let data = cachedDataset[key];
            let elem = bisect(data, x, $.asx);
            let elX  = $.asx(elem);
            let elY  = $.asy(elem);

            // No nearest found, return
            if(elem === undefined || isNaN(elY)) {
              // Blur Key
              d3.select($._points[key])
                .attr("cx", -100);
              return;
            }
            
            // Focus Key
            let cX = xScale(elX);
            let cY = yScale(elY);
            d3.select($._points[key])
              .attr("cx", cX)
              .attr("cy", cY);
            
            // Set focused value
            newModel[key] = elem;

            // Hand X is nearest point to the mouse from all data
            if(handX === undefined) handX =  cX;
            else if(Math.abs(handX - mX) > Math.abs(cX - mX)) handX = cX;
          }

          // Emit
          $.$emit("change", newModel);

          // No nearest X found
          if(handX === undefined) return;
          
          // Move hand
          hand.attr("x1", handX).attr("x2", handX);

          // Tooltip
          let [tw, th] = [tooltipSize.width, tooltipSize.height];
          let tx       = Math.min(Math.max(mX - scrollLeft - tw/2, 0), graphRect.width - tw);
          let ty       = mY < th ? mY + 5 : mY - th - 5;
          tooltip.attr("transform", `translate(${tx},${ty})`);
          if(onPoint) {
            let ttKey  = target.getAttribute("data-key");
            let ttEl   = $.model[ttKey];
            let ttCl   = $.dataset[ttKey].color;
            // In case of no formatters, default to default formatters
            let ttXfmt = $.dataset[ttKey].formatters.x || $.computedOptions.formatters.x;
            let ttYfmt = $.dataset[ttKey].formatters.y || $.computedOptions.formatters.y;
          
            tooltipX.text(ttXfmt(ttEl));
            tooltipY.text(ttYfmt(ttEl));
            tooltipIcon.style("fill", ttCl);
          }
        };
    
        //
        { // Touch Interface

          // Check for touch
          window.addEventListener("touchstart", function handler() {
            isTouch = true;
            window.removeEventListener("touchstart", handler);
          });

          // Dispatch mouse event as you scroll
          let node     = segmentsWrap.node();
          let lastLeft = node.scrollLeft;
          let handler  = async function(event) {
            if(isTouch) {
              let left  = node.scrollLeft;
              let rect  = node.getBoundingClientRect();
              let handX = + hand.attr("x1");
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
            let event = d3.event;
            if(event.target.classList.contains("point")) {
              overlay.selectAll(".tooltip").style("opacity", 0);
            }
          })
          .on("mousemove",   d3SyncThrottle(mouseHandler, 3))
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
      let gapEachDuration = Math.min(duration, this.duration) / 9;
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
      let create;
      create = function() {
        let $;
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
          let copy = create();
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
          let tv = [];
          if(n <= 2) return;
          // Exclude 0 and 1
          for(let i = 1; i < n - 1; i++) {
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
      let $ = this;

      // Graph
      let graph         = d3.select(this.$el);
      let xDuration     = this._xDuration;
      let xBoundary     = this._xBoundary;
      let yBoundary     = this._yBoundary;
      let graphWidth    = this._width;
      let graphHeight   = this._height;
      let graphMargin   = this._margin;
      let dataWidth     = this._dataWidth;
      let dataHeight    = this._dataHeight;
      let graphDuration = this._graphDuration;
      let xTicks        = this._xTicks;
      let yTicks        = this._yTicks;
      // ELements
      let xScale        = this._xScale;
      let yScale        = this._yScale;
      let projection    = this._projection;

      // Segments
      let segmentsWrap        = graph.select(".segments-wrap");
      let scrollLeft          = segmentsWrap.node().scrollLeft;
      let scrollLeftTimestamp = xScale.invert(scrollLeft); // scrollLefTime

      // Visible boundary
      let visibleBoundary = [
        Math.max(xScale.invert(scrollLeft - graphWidth), xBoundary[0]),
        Math.min(xScale.invert(scrollLeft + graphWidth * 2), xBoundary[1])
      ];
      this.visibleBoundary = visibleBoundary;

      // Segments Each
      // Loop
      let segments     = segmentsWrap.select(".segments");
      let segmentNodes = segmentsWrap.selectAll(".segment").nodes().reverse(); // reverse() to start from the most recent segment
      let visibleSegmentsLefts = [ // Segments whose data range are in the visible boundaries
        scrollLeft - graphWidth, scrollLeft + graphWidth
      ];
      let numCached = 0;

      for(let i = 0; i < segmentNodes.length; i++) {

        let node  = segmentNodes[i];
        let seg   = d3.select(node);
        let start = + node.getAttribute("data-start");
        let end   = + node.getAttribute("data-end");
        let left  = + node.getAttribute("data-left");

        // If it is already drawn
        if(node.hasAttribute("data-id"))
        //if(seg.selectAll("path").size() > 0)
          continue;

        // Check visibility
        if(!(visibleSegmentsLefts[0] <= left && left <= visibleSegmentsLefts[1]))
          continue;

        // Assign id
        let segId = `${$.segmentIdAUtoIncrement++}`;
        node.setAttribute("data-id", segId);
        numCached++;

        // Segment dataset
        let segDataGroups = [];
        let segmentDataset = {};
        $._segmentsDataset[segId] = segmentDataset;
        for(let key in this.dataset) {
          let each = this.dataset[key];
          let getter = each.getters.byX;
          let data = [];

          if(each.data !== undefined) {
            data = sliceFromTo(each.data[key], start, end, $.asx);
          } else if(getter !== undefined) {
            data = await getter(start, end);
          }

          // Assign data
          if(data.length > 0) {
            segDataGroups.push({key, data});
            segmentDataset[key] = data;
          }
        }

        // If no data
        if(segDataGroups.length === 0)
          continue;

        // Add paths
        seg.selectAll("paths")
          .data(segDataGroups)
          .enter()
          .append("path")
            .each(function(d) {$._lines[d.key].push(this);})
            .attr("class",        "line")
            .attr("fill",         "none")
            .attr("data-key",     d => d.key)
            .attr("stroke",       d => $.dataset[d.key].color)
            .attr("stroke-width", 1)
            .attr("d", d => d3.line()
              .defined(e => !isNaN($.asy(e)))
              .x(e => xScale($.asx(e)))
              .y(e => yScale($.asy(e)))
              (d.data)
            );

      }

      if(numCached > 0) $._cachedDataset = $.cachedDataset();

    }

  }
}
</script>