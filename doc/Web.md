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