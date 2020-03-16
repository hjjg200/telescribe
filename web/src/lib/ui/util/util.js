
export function colorify(str) {
  let hash = 0;
  for(let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash);
  }
  let hue = (hash & 0x000FFF) % 120;
  hue = hue + (hue > 55 ? 290 : 200);
  hue = hue % 360;
  let sat = (hash & 0xFFF000 >> 12) % 25;
  sat = sat + (sat > 5 ? 70 : 25);
  return `hsl(${hue}, ${sat}%, 50%)`;
}

export function getChildrenTextContent(children) {
  return children.map(function (node) {
    return node.children
    ? getChildrenTextContent(node.children)
    : node.text
  }).join('');
}

export function addThrottledAsyncEvent(elem, type, handler, interval) {
  var running = false;
  var wrap = function(event) {
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

export function addDebouncedAsyncEvent(elem, type, handler, interval) {
  var timer;
  var wrap = function(event) {
    clearTimeout(timer);
    timer = setTimeout(() => handler(event), interval);
  }
  elem.addEventListener(type, wrap);
}