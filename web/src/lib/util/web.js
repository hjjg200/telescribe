

export function formatNumber(num, fmt) {
  if(fmt === "" || fmt == undefined) {
    return String(num);
  }

  let rgx      = /\{(?:([0-9]*(?:\.[0-9]+)?)x)?(?:(?:\.([0-9]*))?f)?\}/g;
  let matches  = [...fmt.matchAll(rgx)];
  let literals = fmt.split(rgx);
  let result   = "";

  for(let i = 0; i < matches.length; i++) {
    let m = matches[i];
    let [coef, prcs] = [m[1], m[2]];
    coef = coef ? Number(coef) : 1;
    prcs = prcs ? Number(prcs) : 0;

    let modified = (num * coef).toFixed(prcs);
    result += literals[i].replace(/\\([{}])/, "$1");
    result += modified;
  }
  result += literals[literals.length - 1];

  return result;
}