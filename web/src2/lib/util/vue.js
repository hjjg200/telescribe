
// Return css based on the given object 
export function cssify(obj) {
  var css = "";
  for(let [k, v] of Object.entries(obj)) {
    css += `${k}:${v};`;
  }
  return css;
}