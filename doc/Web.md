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

|Format|1.503|Note|
|-|-|-|
|`{}` or `{f}`|1.503|Curly brackets are used to represent the value|
|`{.4f}`|1.5030|The formatted value will contain trailing zeros|
|`{.f}`|2|When the precision number is omitted, 0 is assumed|
|`{f}%`|1.503%|Prefix and suffix can be defined outside the brackets|
|`\{{f}\}`|{1.503}|Escape the brackets when you want to use them as string|


## Layout

There are two statuses for layout:

* **Narrow:** when the width is < 35rem
* **Wide:** when the width is >= 35rem


## Sidebar

|Layout|Description|
|-|-|
|Narrow|It is always at the expanded status and visibility is toggled by a button|
|Wide|It is normally at the collapsed status and is expandable by a button|

### SidebarItem

Items typically are mapped to their own client and when clicked the assigned client becomes visible. The background color is randomly chosen by its client id and the text is the first two letters of the client alias in lowercase.

#### Background Color

|Item|Range|
|-|-|
|Hue|0 to 50 and 200 to 255 and 345 to 360|
|Saturation|25% to 30% and 70% to 95%|
|Lightness|50%|

