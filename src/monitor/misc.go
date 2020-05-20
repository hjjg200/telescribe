package monitor

// #include <unistd.h>
import "C"

var (
    c_SC_CLK_TCK = C.sysconf(C._SC_CLK_TCK)
)