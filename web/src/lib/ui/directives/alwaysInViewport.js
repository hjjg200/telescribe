
// Util
import {addThrottledAsyncEvent, addDebouncedAsyncEvent} from '../util/util.js';

let overflow = {
  width: "overflowX", height: "overflowY"
};

function bind(el, {expression}) {
  let tr = {x: 0, y: 0};
  let initRect; // initial rect
  let flex = expression === "flexible";

  let copyRect = ({x, y, width, height}) => {
    return {x, y, width, height};
  }

  let move = async() => {
    let wwh = {
      width: window.innerWidth,
      height: window.innerHeight
    }
    let rect = el.getBoundingClientRect(); // el rect

    // size check
    if(rect.width == 0 || rect.height == 0) return;

    let newRect = copyRect(rect);
    if(initRect == undefined) initRect = copyRect(rect);
    
    if(flex) {
      ["width", "height"].forEach(i => {
        if(newRect[i] > wwh[i]) {
          newRect[i] = wwh[i];
          el.style[overflow[i]] = "auto";
          el.style[i] = `${wwh[i]}px`;
        }
      });
    }

    [["width", "x"], ["height", "y"]].forEach(a => {
      let [wh, xy] = a;

      // restore to initial size
      if(initRect[wh] < wwh[wh]) {
        newRect[wh] = initRect[wh];
        el.style[overflow[wh]] = "";
        el.style[wh] = "";
      }

      //
      newRect[xy] -= tr[xy]; // to 0, 0
      newRect[xy] -= Math.max(0, (newRect[xy] + newRect[wh]) - wwh[wh]);
      newRect[xy] = Math.max(0, newRect[xy]);
      tr[xy] += newRect[xy] - rect[xy];

    });

    el.style.transform = `translate(${tr.x}px, ${tr.y}px)`;
  };

  addThrottledAsyncEvent(
    window, "scroll", move, 10
  );
  addThrottledAsyncEvent(
    window, "resize", move, 10
  );

}

const directive = {
  name: "always-in-viewport",
  bind
};

export default directive;