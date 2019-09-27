package monitor

// /etc/mtab
// df -kP
// When both the -k and -P options are specified, the following header line shall be written (in the POSIX locale):
// "Filesystem 1024-blocks Used Available Capacity Mounted on\n"
// When the -P option is specified without the -k option, the following header line shall be written (in the POSIX locale):
// "Filesystem 512-blocks Used Available Capacity Mounted on\n"