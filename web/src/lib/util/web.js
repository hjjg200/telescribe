

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/split
// If separator is a regular expression that contains capturing parentheses (),
// matched results are included in the array.
const formatRegex = /\{(?:([0-9]*(?:\.[0-9]+)?)x)?(?:(\.[0-9]*)?f)?\}/g;
export function formatNumber(num, fmt) {
  if(fmt === "" || fmt == undefined) {
    return String(num);
  }

  let splits   = fmt.split(formatRegex);
  let matches  = [];
  splits.filter((d, i) => i % 3 != 0).map(
    (d, i, arr) => i % 2 == 0 ? matches.push([d, arr[i+1]]) : false
  );
  let literals = splits.filter((d, i) => i % 3 == 0).map(d => d.replace(/\\([{}])/, "$1"));
  let result   = "";

  for(let i = 0; i < matches.length; i++) {
    let [coef, prcs] = matches[i];
    coef = coef ? Number(coef) : 1;
    
    // Precision
    let modified = num * coef;
    if(prcs) {
      prcs = prcs.substring(1);
      prcs = prcs === "" ? 0 : Number(prcs);
      modified = modified.toFixed(prcs);
    }

    result += literals[i];
    result += formatComma(modified);
  }
  result += literals[literals.length - 1];

  return result;
}

function formatComma(x) {
  var parts = x.toString().split(".");
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return parts.join(".");
}