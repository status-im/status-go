// # 1 "lib/libc/gen/ctype_.c"
// # 1 "<built-in>"
// # 1 "<command-line>"
// # 1 "lib/libc/gen/ctype_.c"
// # 36 "lib/libc/gen/ctype_.c"
// # 1 "./include/ctype.h" 1
// # 43 "./include/ctype.h"
// # 1 "./sys/sys/cdefs.h" 1
// # 41 "./sys/sys/cdefs.h"
// # 1 "./machine/cdefs.h" 1
// # 42 "./sys/sys/cdefs.h" 2
// # 44 "./include/ctype.h" 2
// # 57 "./include/ctype.h"
// typedef void *locale_t;
// 
// 
// 
// 
// 
// extern const char *_ctype_;
// extern const short *_tolower_tab_;
// extern const short *_toupper_tab_;
// 
// 
// int isalnum(int);
// int isalpha(int);
// int iscntrl(int);
// int isdigit(int);
// int isgraph(int);
// int islower(int);
// int isprint(int);
// int ispunct(int);
// int isspace(int);
// int isupper(int);
// int isxdigit(int);
// int tolower(int);
// int toupper(int);
// 
// 
// 
// int isblank(int);
// 
// 
// 
// int isascii(int);
// int toascii(int);
// int _tolower(int);
// int _toupper(int);
// 
// 
// 
// int isalnum_l(int, locale_t);
// int isalpha_l(int, locale_t);
// int isblank_l(int, locale_t);
// int iscntrl_l(int, locale_t);
// int isdigit_l(int, locale_t);
// int isgraph_l(int, locale_t);
// int islower_l(int, locale_t);
// int isprint_l(int, locale_t);
// int ispunct_l(int, locale_t);
// int isspace_l(int, locale_t);
// int isupper_l(int, locale_t);
// int isxdigit_l(int, locale_t);
// int tolower_l(int, locale_t);
// int toupper_l(int, locale_t);
// 
// 
// 
// 
// 
// 
// extern __inline __attribute__((__gnu_inline__)) int isalnum(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & (0x01|0x02|0x04)));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int isalpha(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & (0x01|0x02)));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int iscntrl(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & 0x20));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int isdigit(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & 0x04));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int isgraph(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & (0x10|0x01|0x02|0x04)));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int islower(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & 0x02));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int isprint(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & (0x10|0x01|0x02|0x04|0x80)));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int ispunct(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & 0x10));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int isspace(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & 0x08));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int isupper(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & 0x01));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int isxdigit(int _c)
// {
//  return (_c == -1 ? 0 : ((_ctype_ + 1)[(unsigned char)_c] & (0x04|0x40)));
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int tolower(int _c)
// {
//  if ((unsigned int)_c > 255)
//   return (_c);
//  return ((_tolower_tab_ + 1)[_c]);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int toupper(int _c)
// {
//  if ((unsigned int)_c > 255)
//   return (_c);
//  return ((_toupper_tab_ + 1)[_c]);
// }
// 
// 

// extern __inline __attribute__((__gnu_inline__))
int isblank(int _c)
{
 return (_c == ' ' || _c == '\t');
}



// extern __inline __attribute__((__gnu_inline__)) int isascii(int _c)
// {
//  return ((unsigned int)_c <= 0177);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int toascii(int _c)
// {
//  return (_c & 0177);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int _tolower(int _c)
// {
//  return (_c - 'A' + 'a');
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int _toupper(int _c)
// {
//  return (_c - 'a' + 'A');
// }
// 
// 
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isalnum_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isalnum(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isalpha_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isalpha(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isblank_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isblank(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// iscntrl_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return iscntrl(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isdigit_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isdigit(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isgraph_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isgraph(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// islower_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return islower(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isprint_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isprint(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// ispunct_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return ispunct(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isspace_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isspace(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isupper_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isupper(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// isxdigit_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return isxdigit(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// tolower_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return tolower(_c);
// }
// 
// extern __inline __attribute__((__gnu_inline__)) int
// toupper_l(int _c, locale_t _l __attribute__((__unused__)))
// {
//  return toupper(_c);
// }
// 
// 
// 
// 
// 
// # 37 "lib/libc/gen/ctype_.c" 2
// # 1 "./lib/libc/include/ctype_private.h" 1
// 
// 
// 
// 
// 
// # 5 "./lib/libc/include/ctype_private.h"
// #pragma GCC visibility push(hidden)
// # 5 "./lib/libc/include/ctype_private.h"
// 
// extern const char _C_ctype_[];
// extern const short _C_toupper_[];
// extern const short _C_tolower_[];
// 
// # 9 "./lib/libc/include/ctype_private.h"
// #pragma GCC visibility pop
// # 9 "./lib/libc/include/ctype_private.h"
// 
// # 38 "lib/libc/gen/ctype_.c" 2

const char _C_ctype_[1 + 256] = {
 0,
 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
 0x20, 0x20|0x08, 0x20|0x08, 0x20|0x08, 0x20|0x08, 0x20|0x08, 0x20, 0x20,
 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20, 0x20,
   0x08|(char)0x80, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04, 0x04,
 0x04, 0x04, 0x10, 0x10, 0x10, 0x10, 0x10, 0x10,
 0x10, 0x01|0x40, 0x01|0x40, 0x01|0x40, 0x01|0x40, 0x01|0x40, 0x01|0x40, 0x01,
 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01,
 0x01, 0x01, 0x01, 0x10, 0x10, 0x10, 0x10, 0x10,
 0x10, 0x02|0x40, 0x02|0x40, 0x02|0x40, 0x02|0x40, 0x02|0x40, 0x02|0x40, 0x02,
 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02, 0x02,
 0x02, 0x02, 0x02, 0x10, 0x10, 0x10, 0x10, 0x20,

  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0,
  0, 0, 0, 0, 0, 0, 0, 0
};

const char *_ctype_ = _C_ctype_;
