package monitor

// #include <unistd.h>
import "C"

var (
    // clock ticks - _SC_CLK_TCK
    //   The number of clock ticks per second.  The corresponding
    //   variable is obsolete.  It was of course called CLK_TCK.
    //   (Note: the macro CLOCKS_PER_SEC does not give information: it
    //   must equal 1000000.)
    c_SC_CLK_TCK = C.sysconf(C._SC_CLK_TCK)

    // PAGESIZE - _SC_PAGESIZE
    //   Size of a page in bytes.  Must not be less than 1.
    c_PAGE_SIZE = C.sysconf(C._SC_PAGESIZE)
)
