# Web

This documentation explains the specifications for the web elements.

## Naming

In the web source, Javascript and HTML, when a variable, the name of which is specified in the documentation, is used in under a component or such, the part which matches the name of that parent component is omitted; e.g., when `clientInfo` is used as data of `Client` component, it is named as just `info` not `clientInfo`.

## Config

|Go|Javascript|HTML|
|-|-|-|
|`webCfg`|`webConfig`|`web-config`|

|Item|Default|Description|
|-|-|-|
|`durations`|`[180,1440,7200]`|The array of duration choices for graph plotting; in minutes|
|`format.value`|`{.2f}`|The default number format used for monitored values|
|`format.yAxis`|`{}`|The format used for figures on y axis|
|`format.date.long`|`Y-MM-DD[T]HH:mm:ssZ`|moment.js format used for long date|
|`format.date.short`|`MMM DD HH:mm`|moment.js format used for short date and it is the default format for dates|


## Format

|Go|Javascript|HTML|
|-|-|-|
|`format`|`format`|`format`|

A format is a string expression that is used to modify how monitored values look on the web UI.

|Format|1,503.6|Note|
|-|-|-|
|`{f}`|1,503.6|Curly brackets are used to represent the value|
|`{.4f}`|1,503.6000|The formatted value will contain trailing zeros|
|`{.f}`|1,504|When the precision number is omitted, 0 is assumed|
|`{f}%`|1,503.6%|Prefix and suffix can be defined outside the brackets|
|`\{{f}\}`|{1,503.6}|Escape the brackets when you want to use them as string|
|`{}`|1.50K|When an f is not given in the brackets, *the abbreviation format* is assumed|

### Abbreviation Format

The abbreviation format is designed for the figures on the axis ticks of graphs.

|Number|Abbreviated|Note|
|-|-|-|
|0.00123|1.23e-3|It cannot get longer than 4 characters including the decimal dot|
|1.02|1.02|No changes as it is exactly 4 characters long|
|123|123|Nothing special|
|1,234|1.23K|Expressed is only up to the second decimal place|
|999,000|999K|Units(K, M, B, T) change at 1 thousand, 1 million, 1 billion, and 1 trillion|
|1234 trillion|1.23e+15|When it is 1,000 trillion or higher, the exponential format is used|

### Regular Expression

```regexp
\{(?:(\.[0-9]*)?(f))?\}
```

The regular expression works as follows:

|Group|Description|
|-|-|
|1|The prefix of the format|
|2|The precision of the format; it always includes a dot at the start|
|3|`f` means it is floating point style format -- abbreviation style otherwise|
|4|The suffix of the format|

### NumberFormatter

```js
import {NumberFormatter} from '@/lib/util/web.js';
```

`NumberFormatter` is a utility class that helps format numbers into strings. It has the following methods.

|Name|Static?|Description|
|-|-|-|
|`prefix(string)`|No|Gets the prefix if the argument is undefined, sets the prefix otherwise|
|`precision(number)`|No|Getter and setter for the precision|
|`isFloat(boolean)`|No|Getter and setter for the floating point style format boolean|
|`suffix(string)`|No|Getter and setter for the suffix|
|`clone()`|No|Clones the formatter|
|`format(number)`|No|Formats the given number|
|`defaultFormat(string)`|Yes|Getter and setter for the default number format. The initial default format is `{}`|


## Gtaph

|Javascript|HTML|
|-|-|
|`graph`|`graph`|

A graph component is used to plot a line chart for provided dataset.

|Prop|Description|
|-|-|
|Duration|Defines how much time of data the graph shows in the visible space|
|Boundaries|The timestamps of the start, the end, and the start and end of gaps|
|Dataset|The compound of options and data used for plotting|

### Dataset

Dataset is a **Monitor.Key** to data compound object map, and the the content of that map is as follows:

* `data`: an array of timestamp and value objects
* `color`: css-style color used for lines and points
* `formatter`: a function used to format numbers shown on tooltip


### Hand

When moving the hand, the nearest points of the active items become visible provided the value for that point is not NaN; when it is NaN, the point becomes invisible and it is excluded from the focused values. The focused timestamps are an array of two timestamps; the first timestamp is the minimum timestamp of the active points; and the second timestamp is the maximum of them.