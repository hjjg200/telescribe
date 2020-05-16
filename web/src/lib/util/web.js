

// https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/String/split
// If separator is a regular expression that contains capturing parentheses (),
// matched results are included in the array.
const formatRegex = /\{(?:(\.[0-9]*)?(f))?\}/g;
let defaultFormat = "{}";
export class NumberFormatter {

  static defaultFormat(fmt) {
    if(fmt) {
      defaultFormat = fmt;
      return;
    }
    return defaultFormat;
  }

  constructor(fmt) {
    fmt = fmt || this.constructor.defaultFormat();

    let parsed = parsedFormatters[fmt];
    if(parsed) return parsed.format.bind(parsed);

    this.info = parseFormat(fmt);
    parsedFormatters[fmt] = this;

    return this.format.bind(this);
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
  isFloat(f) {
    if(f) {
      this.info.isFloat = f;
      return this;
    }
    return this.info.isFloat;
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
    let prcsNaN = isNaN(this.precision());

    if(this.isFloat()) { // floating point

      if(!prcsNaN)
        modified = modified.toFixed(this.precision());
      modified = formatComma(modified);

    } else { // abbreviation format

      let abbreviated = false;
      const fmts = [[1.0e+3, 'K'], [1.0e+6, 'M'], [1.0e+9, 'B'], [1.0e+12, 'T']];
      for(let i = 0; i < fmts.length; i++) {
        let fmt = fmts[i];
        if(modified < fmt[0] * 1.0e+3 && modified >= fmt[0]) {
          abbreviated = true;
          modified /= fmt[0];
          let digits = Math.floor(Math.log10(modified)) + 1;
          let prcs   = Math.min(3 - digits, 2);
          modified = modified.toFixed(prcs);
          modified = formatComma(modified) + fmt[1];
        }
      }

      if(!abbreviated) {
        if(modified >= 1.0e+15 || modified.toString().length > 5) {
          modified = modified.toExponential(2);
        }
      }

    }

    return `${this.info.prefix}${modified}${this.info.suffix}`;
  }
}

export function formatNumber(format, value) {
  return (new NumberFormatter(format))(value);
}

let parsedFormatters = {};

function parseFormat(fmt) {
  let splits = fmt.split(formatRegex);
  let [prefix, precision, f, suffix] = splits;

  if(precision) {
    precision = precision.substring(1);
    precision = precision === "" ? 0 : Number(precision);
  } else {
    precision = NaN;
  }
  let isFloat = f === "f";

  [prefix, suffix] = [prefix, suffix].map(d => d ? d.replace(/\\([{}])/, "$1") : "");
  return {prefix, precision, isFloat, suffix};
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

// SPLIT ---

let whitespaceRegexp = /\s+/g;
export function splitWhitespace(str) {
  return str.split(whitespaceRegexp);
}