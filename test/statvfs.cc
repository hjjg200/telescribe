#include <iostream>
#include <sys/statvfs.h>

// int statvfs(const char *path, struct statvfs *buf);

int main(int argc, char ** argv) {

    if(argc < 2) return 1;

    struct statvfs fs;
    std::cout << statvfs(argv[1], &fs) << std::endl;
    std::cout << fs.f_blocks << fs.f_bfree << std::endl;

    return 0;

}