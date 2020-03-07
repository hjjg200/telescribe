<template>
  <article class="graph"></article>
</template>

<script>

// Static functions
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


/*
graph
- 

dataset [key]
- color
- data
*/

export default {
  name: "Graph",
  props: {
  },
  watch: {
    duration(newVal) {
      if(this.drawn || this.boundaries != undefined) {
        this._xScale = this._scale();
        this._draw();
      }
    },
    dataset(newVal) {
      this.update();
    },
    boundaries(newVal) {
      if(this.duration == undefined) return;
      this._xBoundary = this.$d3.extent(newVal);
      this._xScale = this._scale();
      this._draw(true);
    }
  },
  data() {
    return {
      duration: undefined,
      dataset: {},
      boundaries: undefined,
      drawn: false
    };
  },
  computed: {
    keys() {
      return Object.keys(this.dataset);
    }
  },

  created() {

    //
    var $ = this;

    // Public

    // Global Event
    window.addEventListener("resize", function() {
      $._resize_handler();
    });

  },

  methods: {

    async _draw(reset = false) {

      this.drawn = true;

      // Shorthand access
      var $ = this;
    
    // Vars
      var chart         = this.$d3.select(this.$el);
      var chartDuration = this.duration * 60; // Into seconds
      var chartMargin   = {
        top: remToPx(0.5),
        left: remToPx(2.5),
        bottom: remToPx(2)
      };
      var chartNode = chart.node();
      var chartRect = {
        width: chartNode.offsetWidth - chartMargin.left,
        height: chartNode.offsetHeight - chartMargin.top - chartMargin.bottom
      };
      var xScale    = this._xScale;
      var xBoundary = this._xBoundary;
      var xDuration = xScale.totalDuration;
      // Duration too low
      if(chartDuration == undefined || chartDuration > xDuration) chartDuration = xDuration;
      var dataWidth  = chartRect.width * xDuration / chartDuration;
      var dataHeight = chartRect.height;
      var dw_cw      = dataWidth / chartRect.width;
      var segNo      = Math.ceil(dw_cw);
      var xTicks     = Math.round(dw_cw * 4);
      var yTicks     = 4;
      this._dataWidth     = dataWidth;
      this._dataHeight    = dataHeight;
      this._width         = chartRect.width;
      this._height        = chartRect.height;
      this._margin        = chartMargin;
      this._xDuration     = xDuration;
      this._yTicks        = yTicks;
      this._chartDuration = chartDuration;
      this._priorKeys     = null;
      this._priorXScale   = null;
    
    // Element Var
      var scrollLeft = Math.max(0, dataWidth - chartRect.width);
      var handX      = scrollLeft + chartRect.width / 2;
    
    // PRIOR VALUES
      var prior = false;
      var priorHandXEnRectPercent;
      var priorHandXPercent;
      {
        var segmentsWrap = chart.select(".segments-wrap");
        if(segmentsWrap.size() > 0 && !reset) {
          prior = true;
    
          var priorXScale   = this._priorXScale;
          this._priorXScale = xScale;

          var node       = segmentsWrap.node();
          var hand       = segmentsWrap.select(".hand");
          var priorHandX = Number(hand.attr("x1"));
    
          // Relative to segments-wrap node
          priorHandXEnRectPercent = (priorHandX - node.scrollLeft) / node.offsetWidth;
          // Absolute
          priorHandT = priorXScale.invert(priorHandX);
    
          // Scroll Left and Hand X
          handX      = xScale(priorHandT);
          scrollLeft = Math.max(0, handX - chartRect.width * priorHandXEnRectPercent);

          // When the hand is outside the chart
          if(!(scrollLeft <= handX && handX <= scrollLeft + chartRect.width)) {
            handX = scrollLeft + chartRect.width / 2;
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
        .call(this.$d3.axisBottom(xScale)
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
        $._resize_handler = function() {
          clearTimeout($._resize_timer);
          $._resize_timer = setTimeout(function() {
            $._draw();
          }, 100);
        };
      }

      { // Hand, Points and Tooltip and Touch Interface
        var isMouseDown = false;
        var isTouch     = false;
        var bisect = function(slice, timestamp, accessor) {
          var bs = $.$d3.bisector(accessor).left;
          var i   = bs(slice, timestamp);
          var d0  = slice[i-1];
          var d1  = slice[i];
          if(d0 === undefined && d1 === undefined) return undefined;
          else if(d0 === undefined) return d1;
          else if(d1 === undefined) return d0;
          else return timestamp - accessor(d0) > accessor(d1) - timestamp ? d1 : d0;
        };

        // Moving hand
        // - mousedown
        // - mousemove when mouse is down
        // - scroll
        var mouseHandler = function() {

          var event      = $.$d3.event;
          var target     = event.target;
          var mouse      = $.$d3.mouse(projection.node());
          var [mX, mY]   = mouse;
          var timestamp  = xScale.invert(mX);
          var dataset    = $.dataset;
          var seriesName = $._seriesName;
          var yScale     = $._yScale;
          var scrollLeft = segmentsWrap.node().scrollLeft;

          // Event Type Check
          if(event.type === "mousemove" && !(isTouch || isMouseDown))
            return;

          //
          segments.selectAll(".point").style("opacity", 1);
          background.select(".focus-date text").style("opacity", 1);
          if(event.target.hasClass("point")) {
            overlay.selectAll(".tooltip").style("opacity", 1);
          }

          //
          var handX;
    
          // Points
          for(let key in dataset) {
            let data = dataset[key];
            var elem = bisect(data, timestamp, function(d) { return d.timestamp; });
            if(elem === undefined || isNaN(elem.value)) return;
            var cX     = xScale(elem.timestamp);
            var cY     = yScale(elem.value);
            var series = seriesName(key);
            segments.select(`.point.${series}`)
              .attr("data-timestamp", elem.timestamp)
              .attr("data-value", elem.value)
              .attr("cx", cX)
              .attr("cy", cY);

            // Hand X
            if(handX === undefined) handX =  cX;
            else if(Math.abs(handX - mX) > Math.abs(cX - mX)) handX = cX;
          }

          // No nearest X
          if(handX === undefined) return;
          
          // Hand
          var handT = xScale.invert(handX);
          hand.attr("x1", handX).attr("x2", handX);

          // Time
          background.select(".focus-date text").text(handT.date("MM/DD"));
    
          // Tooltip
          var [tw, th] = [tooltipSize.width, tooltipSize.height];
          var tx      = Math.min(Math.max(mX - scrollLeft - tw/2, 0), chartRect.width - tw);
          var ty      = mY < th ? mY + 5 : mY - tooltipSize.height - 5;
          tooltip.attr("transform", `translate(${tx},${ty})`);
          if(target.hasClass("point")) {
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
          window.addEventListener("touchstart", function handler() {
            isTouch = true;
            window.removeEventListener("touchstart", handler);
          });
          // Dispatch mouse event as you scroll
          var node = segmentsWrap.node();
          var lastLeft = node.scrollLeft;
          var interval = 1;
          var handler = function handler(event) {
            // Pre
            node.removeEventListener("scroll", handler);
            // Main
            if(isTouch) {
              var left = node.scrollLeft;
              var rect = node.getBoundingClientRect();
              var handX = Number(hand.attr("x1"));
              // Arbitrary Coords
              event.clientX = (handX - left) + (left - lastLeft) + rect.left;
              event.clientY = rect.top + rect.height / 2;
              lastLeft = left;
              $.$d3.customEvent(event, mouseHandler);
            }
            // Post
            setTimeout(function() {
              node.addEventListener("scroll", handler);
            }, interval);
          };
          node.addEventListener("scroll", handler);
        }
    
        // Add Handlers
        chart
          .on("mouseout", function() {
            var event = $.$d3.event;
            background.select(".focus-date text").style("opacity", 0);
            if(event.target.hasClass("point")) {
              overlay.selectAll(".tooltip").style("opacity", 0);
            }
          })
          .on("mousemove", mouseHandler)
          .on("mousedown", mouseHandler)
          .on("mousedown", () => {isMouseDown = true;})
          .on("mouseup",   () => {isMouseDown = false;});
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
        let start = gapBoundaries[i];
        let end   = gapBoundaries[i+1];
        duration -= end - start;
      }

      // |     total duration      |
      // | duration | gap duration |
      let gapEachDuration = duration / (30 + gapNo);
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

    async update() {

      if(!this.drawn) return;

      // Shorthand
      var $ = this;

      // Chart
      var chart = this.$d3.select(this.$el);

      var priorKeys     = this._priorKeys;
      var xDuration     = this._xDuration;
      var xBoundary     = this._xBoundary;
      var chartWidth    = this._width;
      var chartHeight   = this._height;
      var chartMargin   = this._margin;
      var dataWidth     = this._dataWidth;
      var dataHeight    = this._dataHeight;
      var chartDuration = this._chartDuration;
      var yTicks        = this._yTicks;
      // ELements
      var xScale     = this._xScale;
      var projection = this._projection;

      // Check Changes
      var keysChanged = !keysEqual(this.keys, priorKeys);

      // Info Set
      this._priorKeys = this.keys.slice(0);

      // Segments
      var segmentsWrap = chart.select(".segments-wrap");
      var scrollLeft = segmentsWrap.node().scrollLeft;

      // Dataset
      var seriesName = this.$d3.scaleOrdinal()
        .domain(this.keys)
        .range("abcdefghijklmnopqrstuvwxyz".split("").map(function(a) {
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

      // DATASET
      
      var yBoundary = this.$d3.extent(function() {
        var arr = [];
        for(let key in $.dataset) {
          var data = $.dataset[key];
          data.forEach(i => {
            arr.push(i.value);
          });
        }
        return arr;
      }());
      if(yBoundary[0] == undefined) {
        yBoundary = [0, 1];
      } else if(yBoundary[0] == yBoundary[1]) {
        yBoundary[0] = 0;
      }

      // Visible Dataset
      var visibleDataset = {};
      for(let key in this.dataset) {
        visibleDataset[key] = this.dataset[key].filter(i => {
          return visibleBoundary[0] <= i.timestamp && i.timestamp <= visibleBoundary[1];
        });
      }

      // Update Y Axis
      var yScale = this.$d3.scaleLinear()
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
              .call(this.$d3.axisLeft(yScale)
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
        .call(this.$d3.axisLeft(yScale)
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
        var seg = $.$d3.select(node);
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
          // invisible
          return;
        }

        // Segment Dataset
        var segDataGroups = [];
        for(let key in visibleDataset) {
          let data = visibleDataset[key].filter(i => {
            return start <= i.timestamp && i.timestamp <= end;
          });
          if(data.length > 0) {
            segDataGroups.push({
              key: key,
              data: data
            });
          }
        }

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
              return $.$d3.line()
                .defined(function(e) { return !isNaN(e.value); })
                .x(function(e) { return xScale(e.timestamp) })
                .y(function(e) { return yScale(e.value) })
                (d.data);
            });

      });

      // Points
      // Regenerate points when keys were changed
      if(keysChanged === true) {
        segments.selectAll(".circles circle").remove();
        var circles = segments.select(".circles");
        circles.selectAll("circles")
            .data(this.keys)
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

  }
}
</script>