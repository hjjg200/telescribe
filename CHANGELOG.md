### Alpha 0.12

Focuses on the overall change of looks

- [ ] **TODO:** wrap icon and status togeether
- [x] **Fixed:** font import error in css
- [x] **Fixed:** x-axis tick size
- [ ] **Fixed:** made the hand to be reset to the middle, when the hand is outside the chart rect
- [ ] **Fixed:** EOF errors
- [ ] **Fixed:** infinite re-render bug
- [ ] **Changed:** when writing cache files, write to temp files and rename them to minimalize the odds of files being corrupted
- [x] **Changed:** overall change of web layout and design
- [x] **Changed:** web page title
- [ ] **Changed:** web uses per-key format
- [x] **Deprecated:** plain web resources are replaced with Vue

### Alpha 0.11

Focuses on more extensive use of Vue.js

- [x] **Changed:** now using vue SFC

### Alpha 0.10

- [x] **Added:** duration buttons
- [x] **Changed:** the location of the hand will remain the same after duration changes.
- [x] **Changed:** gaps in the chart now have the fixed width -- not proportional to the duration of the gap; i.e., a gap from Sep 12 to Sep 20 has the same width as a gap from Jan 1 to Oct 1.
- [x] **Changed:** friendly mouse(touch) events for mobile devices; swiping the chart in the mobile devices will let the hand move along as you swipe.
- [x] **Changed:** now use ES6 modules
- [x] **Changed:** shortened monitor keys

### Alpha 0.9

- [x] **Added:** RESTful webhook about fatal status
- [x] **Changed:** server notifies its clients when the config is changed
- [x] **Changed:** separated client config from server config struct

### Alpha 0.8

Focuses on the imporvement of chart performance by using D3.js

- [x] **Added:** network metrics
- [x] **Added:** disk metrics
- [x] **Changed:** chart library changed from Chartist.js to D3.js

### Alpha 0.7

- [x] **Changed:** now web uses Vue.js

### Alpha 0.6

Focuses on better handling of packets and encryption

- [x] **Added:** custom session protocol and handlers
- [x] **Changed:** `package secret` now using elliptic curve and aes256gcm
- [x] **Deprecated:** hybrid encryption with RSA and AES
- [x] **Deprecated:** plain json packet

### Alpha 0.5

Initial commit to github