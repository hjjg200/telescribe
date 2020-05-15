# Util

This documentation explains the specifications for the utility-purpose elements.

## Range

|Go|Javascript|HTML|
|-|-|-|
|rng|range|range|

A range represents a numeric range which is used to examine the status of monitored values. Commas are used to express multiple ranges in one string and colons are used to express ranges.

|Example|Description|
|-|-|
|`1`|x = 1|
|`1:`|x >= 1|
|`1:2`|1 <= x <= 2|
|`:2`|x <= 2|
|`1:3,5:`|1 <= x <= 3 and 5 <= x|