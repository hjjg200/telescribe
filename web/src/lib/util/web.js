

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/split
// If separator is a regular expression that contains capturing parentheses (),
// matched results are included in the array.
const formatRegex = /\{(?:(\.[0-9]*)?f)?\}/g;
export class NumberFormatter {
  constructor(fmt) {
    fmt = fmt || "";
    this.info = parseFormat(fmt);
  }

  clone() {
    let n = new NumberFormatter();
    n.info = {...this.info};
    return n;
  }

  prefix(str) {
    if(str) {
      this.info.prefix = str;
      return this;
    }
    return this.info.prefix;
  }
  precision(prcs) {
    if(prcs) {
      this.info.precision = prcs;
      return this;
    }
    return this.info.precision;
  }
  suffix(str) {
    if(str) {
      this.info.suffix = str;
      return this;
    }
    return this.info.suffix;
  }

  format(num) {
    let modified = num;
    if(!isNaN(this.info.precision))
      modified = modified.toFixed(this.info.precision);
    modified = formatComma(modified);
  
    return `${this.info.prefix}${modified}${this.info.suffix}`;
  }
}

function parseFormat(fmt) {
  let splits = fmt.split(formatRegex);
  let [prefix, precision, suffix] = splits;

  if(precision) {
    precision = precision.substring(1);
    precision = precision === "" ? 0 : Number(precision);
  } else {
    precision = NaN;
  }

  [prefix, suffix] = [prefix, suffix].map(d => d ? d.replace(/\\([{}])/, "$1") : "");
  return {prefix, precision, suffix};
}

function formatComma(x) {
  var parts = x.toString().split(".");
  parts[0] = parts[0].replace(/\B(?=(\d{3})+(?!\d))/g, ",");
  return parts.join(".");
}

// STATUS ---

export function statusIconOf(val, ignoreGreen = false) {
  let st = val;
  if(typeof val === "object") {
    st = Math.max(
      ...Object.values(val).map(d => d.status)
    );
  }

  if(isNaN(st)) return '';
  
  if(st === 0 && !ignoreGreen) return 'green-light';
  else if(st === 8)            return 'warning';
  else if(st === 16)           return 'error';

}