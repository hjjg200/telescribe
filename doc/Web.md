# Web

This documentation explains the specifications for the web elements.

## Config

|Item|Description|
|-|-|
|`durations`|The array of duration choices for graph plotting; in minutes|
|`format.value`|The default moment.js format used for monitored values|
|`format.date.long`|moment.js format used for long date|
|`format.date.short`|moment.js format used for short date|


## Format

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
|0.00123|1.0e-3|Exponential format is used when it gets longer than 4 characters including the decimal dot|
|1.02|1.02||
|123|123||
|1,234|1.23K|Only up to the second decimal place is expressed|
|999,000|999K|Units(K, M, B, T) change at 1 thousand, 1 million, 1 billion, and 1 trillion|
|1234 trillion|1.23e+15|When it is 1,000 trillion or higher, the exponential format is used|

### Regular Expression

```regexp
\{(?:(\.[0-9]*)?(f))?\}
```


## Layout

There are two statuses for layout:

* **Narrow:** when the width is < 35rem
* **Wide:** when the width is >= 35rem


## Client




## Sidebar

* Fixtures

|Item|Description|
|-|-|
|Expand Button|It is used to expand or collapse the sidebar|
|Toggle Button|It is used to toggle the visibility of the sidebar|

* Per-layout Specs

|Item|Narrow|Wide|
|-|-|-|
|Sidebar|Always expanded; visibility toggleable|Always visible; collapsed or expanded|
|Expand Button|Invisible|Visible at the top of the sidebar|
|Toggle Button|Visible at the top right corner|Invisible|

### SidebarLabel

Labels act as separators in a sidebar and they are shown as horizontal rules when the sidebar is collapsed.

### SidebarItem

Items typically are mapped to their own client and when clicked the assigned client becomes visible. The background color is randomly chosen by its client id and the text is the first two letters of the client alias in lowercase.

* Background Color

|Item|Range|
|-|-|
|Hue|0 to 50 and 200 to 255 and 345 to 360|
|Saturation|25% to 30% and 70% to 95%|
|Lightness|50%|

* Content

An `h4` element must be present and it will become the title of the item. And if a span element follows, it will be considered as the description for the item.

```html
<SidebarItem>
  <h4>Title</h4>
  <span>Description</span>
</SidebarItem>
```


## Gtaph

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
