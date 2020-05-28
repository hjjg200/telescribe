#include <iostream>
#include <bitset>

#ifndef CHAR_BIT
#define CHAR_BIT 8
#endif

#define PROPAGATE_ALL_ONES(x) \
  ((sizeof (x) < sizeof (uintmax_t) \
    && (~ (x) == (sizeof (x) < sizeof (int) \
                  ? - (1 << (sizeof (x) * CHAR_BIT)) \
                  : 0))) \
   ? UINTMAX_MAX : (uintmax_t) (x))

int main(int argc, char ** argv) {

    std::cout << "UINTMAX_MAX: " << UINTMAX_MAX << std::endl;

    int i = 0;
    std::bitset<sizeof(int)> bit_i(i);
    std::cout << "2(int(0)): " << bit_i << std::endl;

    i = ~i;
    std::bitset<sizeof(int)> bit_ii(i);
    std::cout << "2(~int(0)): " << bit_ii << std::endl;

    auto pii = PROPAGATE_ALL_ONES(i);
    std::bitset<sizeof(pii)> bit_pii(pii);
    std::cout << "10(PROPAGATE_ALL_ONES(~int(0))): " << pii << std::endl;
    std::cout << "2(PROPAGATE_ALL_ONES(~int(0))): " << bit_pii << std::endl;

    return 0;

}