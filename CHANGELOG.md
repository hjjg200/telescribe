### Alpha 0.13

Overall refinement

- [ ] **Added:** `web` debug mode for time measuring
- [ ] **Fixed:** `web` infinite re-render bug
- [x] **Fixed:** `server` EOF errors; low priority, minute error
- [ ] **Fixed:** `server` performance(possible leakage) issue, which might be related to http
- [ ] **Changed:** `web` uses per-key format
- [x] **Changed:** `server` manual HTTP request handling for preserving remote address

### Alpha 0.12

Focuses on the overall change of looks

- [x] **TODO:** `web` wrap icon and status togeether
- [x] **Added:** `web` dropdown component
- [x] **Fixed:** `web` font import error in css
- [x] **Fixed:** `web` x-axis tick size
- [x] **Fixed:** `web` made the hand to be reset to the middle, when the hand is outside the chart rect
- [x] **Changed:** `server` when writing cache files, write to temp files and rename them to minimalize the odds of files being corrupted
- [x] **Changed:** `web` overall change of web layout and design
- [x] **Changed:** `web` page title
- [x] **Deprecated:** `web` plain web resources are replaced with Vue

### Alpha 0.11

Focuses on more extensive use of Vue.js

- [x] **Changed:** `web` now using vue SFC

### Alpha 0.10

- [x] **Added:** `web` duration buttons
- [x] **Changed:** `web` the location of the hand will remain the same after duration changes.
- [x] **Changed:** `web` gaps in the chart now have the fixed width -- not proportional to the duration of the gap; i.e., a gap from Sep 12 to Sep 20 has the same width as a gap from Jan 1 to Oct 1.
- [x] **Changed:** `web` friendly mouse(touch) events for mobile devices; swiping the chart in the mobile devices will let the hand move along as you swipe.
- [x] **Changed:** `web` now use ES6 modules
- [x] **Changed:** `web` shortened monitor keys

### Alpha 0.9

- [x] **Added:** `server` RESTful webhook about fatal status
- [x] **Changed:** `server` `client` server notifies its clients when the config is changed
- [x] **Changed:** `server` separated client config from server config struct

### Alpha 0.8

Focuses on the imporvement of chart performance by using D3.js

- [x] **Added:** `metrics` network metrics
- [x] **Added:** `metrics` disk metrics
- [x] **Changed:** `web` chart library changed from Chartist.js to D3.js

### Alpha 0.7

- [x] **Changed:** `web` now web uses Vue.js

### Alpha 0.6

Focuses on better handling of packets and encryption

- [x] **Added:** `packet` custom session protocol and handlers
- [x] **Changed:** `packet` now using elliptic curve and aes256gcm
- [x] **Deprecated:** `packet` hybrid encryption with RSA and AES
- [x] **Deprecated:** `packet` plain json packet

### Alpha 0.5

Initial commit to github