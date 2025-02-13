// +build ingore

// #	@(#)COPYRIGHT	8.2 (Berkeley) 3/21/94
// 
// The compilation of software known as FreeBSD is distributed under the
// following terms:
// 
// Copyright (c) 1992-2021 The FreeBSD Project.
// 
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
// 
// THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED.  IN NO EVENT SHALL THE AUTHOR OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
// OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
// HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
// LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
// OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
// SUCH DAMAGE.
// 
// The 4.4BSD and 4.4BSD-Lite software is distributed under the following
// terms:
// 
// All of the documentation and software included in the 4.4BSD and 4.4BSD-Lite
// Releases is copyrighted by The Regents of the University of California.
// 
// Copyright 1979, 1980, 1983, 1986, 1988, 1989, 1991, 1992, 1993, 1994
// 	The Regents of the University of California.  All rights reserved.
// 
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions
// are met:
// 1. Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
// 2. Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
// 3. All advertising materials mentioning features or use of this software
//    must display the following acknowledgement:
// This product includes software developed by the University of
// California, Berkeley and its contributors.
// 4. Neither the name of the University nor the names of its contributors
//    may be used to endorse or promote products derived from this software
//    without specific prior written permission.
// 
// THIS SOFTWARE IS PROVIDED BY THE REGENTS AND CONTRIBUTORS ``AS IS'' AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED.  IN NO EVENT SHALL THE REGENTS OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
// OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
// HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
// LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
// OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
// SUCH DAMAGE.
// 
// The Institute of Electrical and Electronics Engineers and the American
// National Standards Committee X3, on Information Processing Systems have
// given us permission to reprint portions of their documentation.
// 
// In the following statement, the phrase ``this text'' refers to portions
// of the system documentation.
// 
// Portions of this text are reprinted and reproduced in electronic form in
// the second BSD Networking Software Release, from IEEE Std 1003.1-1988, IEEE
// Standard Portable Operating System Interface for Computer Environments
// (POSIX), copyright C 1988 by the Institute of Electrical and Electronics
// Engineers, Inc.  In the event of any discrepancy between these versions
// and the original IEEE Standard, the original IEEE Standard is the referee
// document.
// 
// In the following statement, the phrase ``This material'' refers to portions
// of the system documentation.
// 
// This material is reproduced with permission from American National
// Standards Committee X3, on Information Processing Systems.  Computer and
// Business Equipment Manufacturers Association (CBEMA), 311 First St., NW,
// Suite 500, Washington, DC 20001-2178.  The developmental work of
// Programming Language C was completed by the X3J11 Technical Committee.
// 
// The views and conclusions contained in the software and documentation are
// those of the authors and should not be interpreted as representing official
// policies, either expressed or implied, of the Regents of the University
// of California.
// 
// 
// NOTE: The copyright of UC Berkeley's Berkeley Software Distribution ("BSD")
// source has been updated.  The copyright addendum may be found at
// ftp://ftp.cs.berkeley.edu/pub/4bsd/README.Impt.License.Change and is
// included below.
// 
// July 22, 1999
// 
// To All Licensees, Distributors of Any Version of BSD:
// 
// As you know, certain of the Berkeley Software Distribution ("BSD") source
// code files require that further distributions of products containing all or
// portions of the software, acknowledge within their advertising materials
// that such products contain software developed by UC Berkeley and its
// contributors.
// 
// Specifically, the provision reads:
// 
// "     * 3. All advertising materials mentioning features or use of this software
//       *    must display the following acknowledgement:
//       *    This product includes software developed by the University of
//       *    California, Berkeley and its contributors."
// 
// Effective immediately, licensees and distributors are no longer required to
// include the acknowledgement within advertising materials.  Accordingly, the
// foregoing paragraph of those BSD Unix files containing it is hereby deleted
// in its entirety.
// 
// William Hoskins
// Director, Office of Technology Licensing
// University of California, Berkeley

// Preprocessed and manualy edited https://github.com/freebsd/freebsd-src/blob/main/lib/libc/locale/table.c


///	__asm__(".ident\t\"" 
///	"$FreeBSD$" 
///	"\"");
///	
///	typedef signed char __int8_t;
///	typedef unsigned char __uint8_t;
///	typedef short __int16_t;
///	typedef unsigned short __uint16_t;
///	typedef int __int32_t;
///	typedef unsigned int __uint32_t;
///	
///	typedef long __int64_t;
typedef unsigned long __uint64_t;
///	
///	typedef __int32_t __clock_t;
///	typedef __int64_t __critical_t;
///	
///	typedef double __double_t;
///	typedef float __float_t;
///	
///	typedef __int64_t __intfptr_t;
///	typedef __int64_t __intptr_t;
///	
///	typedef __int64_t __intmax_t;
///	typedef __int32_t __int_fast8_t;
///	typedef __int32_t __int_fast16_t;
///	typedef __int32_t __int_fast32_t;
///	typedef __int64_t __int_fast64_t;
///	typedef __int8_t __int_least8_t;
///	typedef __int16_t __int_least16_t;
///	typedef __int32_t __int_least32_t;
///	typedef __int64_t __int_least64_t;
///	
///	typedef __int64_t __ptrdiff_t;
///	typedef __int64_t __register_t;
///	typedef __int64_t __segsz_t;
typedef __uint64_t __size_t;
///	typedef __int64_t __ssize_t;
///	typedef __int64_t __time_t;
///	typedef __uint64_t __uintfptr_t;
///	typedef __uint64_t __uintptr_t;
///	
///	typedef __uint64_t __uintmax_t;
///	typedef __uint32_t __uint_fast8_t;
///	typedef __uint32_t __uint_fast16_t;
///	typedef __uint32_t __uint_fast32_t;
///	typedef __uint64_t __uint_fast64_t;
///	typedef __uint8_t __uint_least8_t;
///	typedef __uint16_t __uint_least16_t;
///	typedef __uint32_t __uint_least32_t;
///	typedef __uint64_t __uint_least64_t;
///	
///	typedef __uint64_t __u_register_t;
///	typedef __uint64_t __vm_offset_t;
///	typedef __uint64_t __vm_paddr_t;
///	typedef __uint64_t __vm_size_t;
///	
///	typedef int ___wchar_t;
///	
///	typedef __int32_t __blksize_t;
///	typedef __int64_t __blkcnt_t;
///	typedef __int32_t __clockid_t;
///	typedef __uint32_t __fflags_t;
///	typedef __uint64_t __fsblkcnt_t;
///	typedef __uint64_t __fsfilcnt_t;
///	typedef __uint32_t __gid_t;
///	typedef __int64_t __id_t;
///	typedef __uint64_t __ino_t;
///	typedef long __key_t;
///	typedef __int32_t __lwpid_t;
///	typedef __uint16_t __mode_t;
///	typedef int __accmode_t;
///	typedef int __nl_item;
///	typedef __uint64_t __nlink_t;
///	typedef __int64_t __off_t;
///	typedef __int64_t __off64_t;
///	typedef __int32_t __pid_t;
///	typedef __int64_t __rlim_t;
///	
///	
///	typedef __uint8_t __sa_family_t;
///	typedef __uint32_t __socklen_t;
///	typedef long __suseconds_t;
///	typedef struct __timer *__timer_t;
///	typedef struct __mq *__mqd_t;
///	typedef __uint32_t __uid_t;
///	typedef unsigned int __useconds_t;
///	typedef int __cpuwhich_t;
///	typedef int __cpulevel_t;
///	typedef int __cpusetid_t;
///	typedef __int64_t __daddr_t;
///	
typedef int __ct_rune_t;
typedef __ct_rune_t __rune_t;
///	typedef __ct_rune_t __wint_t;
///	
///	
///	
///	typedef __uint_least16_t __char16_t;
///	typedef __uint_least32_t __char32_t;
///	
///	
///	
///	
///	
///	
///	
///	typedef struct {
///	 long long __max_align1 __attribute__((__aligned__(_Alignof(long long))));
///	
///	 long double __max_align2 __attribute__((__aligned__(_Alignof(long double))));
///	
///	} __max_align_t;
///	
///	typedef __uint64_t __dev_t;
///	
///	typedef __uint32_t __fixpt_t;
///	
///	
///	
///	
///	
///	typedef union {
///	 char __mbstate8[128];
///	 __int64_t _mbstateL;
///	} __mbstate_t;
///	
///	typedef __uintmax_t __rman_res_t;
///	
///	
///	
///	
///	
///	
///	typedef __builtin_va_list __va_list;
///	
///	
///	
///	
///	
///	
///	typedef __va_list __gnuc_va_list;
///	
///	
///	
///	
///	unsigned long ___runetype(__ct_rune_t) __attribute__((__pure__));
///	__ct_rune_t ___tolower(__ct_rune_t) __attribute__((__pure__));
///	__ct_rune_t ___toupper(__ct_rune_t) __attribute__((__pure__));
///	
///	
///	extern int __mb_sb_limit;

typedef struct {
 __rune_t __min;
 __rune_t __max;
 __rune_t __map;
 unsigned long *__types;
} _RuneEntry;

typedef struct {
 int __nranges;
 _RuneEntry *__ranges;
} _RuneRange;

typedef struct {
 char __magic[8];
 char __encoding[32];

 __rune_t (*__sgetrune)(const char *, __size_t, char const **);
 int (*__sputrune)(__rune_t, char *, __size_t, char **);
 __rune_t __invalid_rune;

 unsigned long __runetype[(1 <<8 )];
 __rune_t __maplower[(1 <<8 )];
 __rune_t __mapupper[(1 <<8 )];






 _RuneRange __runetype_ext;
 _RuneRange __maplower_ext;
 _RuneRange __mapupper_ext;

 void *__variable;
 int __variable_len;
} _RuneLocale;
///	
///	extern const _RuneLocale _DefaultRuneLocale;
///	extern const _RuneLocale *_CurrentRuneLocale;
///	
///	
///	
///	extern _Thread_local const _RuneLocale *_ThreadRuneLocale;
///	static __inline const _RuneLocale *__getCurrentRuneLocale(void)
///	{
///	
///	 if (_ThreadRuneLocale)
///	  return _ThreadRuneLocale;
///	 return _CurrentRuneLocale;
///	}
///	
///	
///	
///	
///	
///	static __inline int
///	__maskrune(__ct_rune_t _c, unsigned long _f)
///	{
///	 return ((_c < 0 || _c >= (1 <<8 )) ? ___runetype(_c) :
///	  (__getCurrentRuneLocale())->__runetype[_c]) & _f;
///	}
///	
///	static __inline int
///	__sbmaskrune(__ct_rune_t _c, unsigned long _f)
///	{
///	 return (_c < 0 || _c >= __mb_sb_limit) ? 0 :
///	        (__getCurrentRuneLocale())->__runetype[_c] & _f;
///	}
///	
///	static __inline int
///	__istype(__ct_rune_t _c, unsigned long _f)
///	{
///	 return (!!__maskrune(_c, _f));
///	}
///	
///	static __inline int
///	__sbistype(__ct_rune_t _c, unsigned long _f)
///	{
///	 return (!!__sbmaskrune(_c, _f));
///	}
///	
///	static __inline int
///	__isctype(__ct_rune_t _c, unsigned long _f)
///	{
///	 return (_c < 0 || _c >= 128) ? 0 :
///	        !!(_DefaultRuneLocale.__runetype[_c] & _f);
///	}
///	
///	static __inline __ct_rune_t
///	__toupper(__ct_rune_t _c)
///	{
///	 return (_c < 0 || _c >= (1 <<8 )) ? ___toupper(_c) :
///	        (__getCurrentRuneLocale())->__mapupper[_c];
///	}
///	
///	static __inline __ct_rune_t
///	__sbtoupper(__ct_rune_t _c)
///	{
///	 return (_c < 0 || _c >= __mb_sb_limit) ? _c :
///	        (__getCurrentRuneLocale())->__mapupper[_c];
///	}
///	
///	static __inline __ct_rune_t
///	__tolower(__ct_rune_t _c)
///	{
///	 return (_c < 0 || _c >= (1 <<8 )) ? ___tolower(_c) :
///	        (__getCurrentRuneLocale())->__maplower[_c];
///	}
///	
///	static __inline __ct_rune_t
///	__sbtolower(__ct_rune_t _c)
///	{
///	 return (_c < 0 || _c >= __mb_sb_limit) ? _c :
///	        (__getCurrentRuneLocale())->__maplower[_c];
///	}
///	
///	static __inline int
///	__wcwidth(__ct_rune_t _c)
///	{
///	 unsigned int _x;
///	
///	 if (_c == 0)
///	  return (0);
///	 _x = (unsigned int)__maskrune(_c, 0xe0000000L|0x00040000L);
///	 if ((_x & 0xe0000000L) != 0)
///	  return ((_x & 0xe0000000L) >> 30);
///	 return ((_x & 0x00040000L) != 0 ? 1 : -1);
///	}
///	
///	
///	
///	int isalnum(int);
///	int isalpha(int);
///	int iscntrl(int);
///	int isdigit(int);
///	int isgraph(int);
///	int islower(int);
///	int isprint(int);
///	int ispunct(int);
///	int isspace(int);
///	int isupper(int);
///	int isxdigit(int);
///	int tolower(int);
///	int toupper(int);
///	
///	
///	int isascii(int);
///	int toascii(int);
///	
///	
///	
///	int isblank(int);
///	
///	
///	
///	int digittoint(int);
///	int ishexnumber(int);
///	int isideogram(int);
///	int isnumber(int);
///	int isphonogram(int);
///	int isrune(int);
///	int isspecial(int);
///	
///	
///	
///	
///	
///	typedef struct _xlocale *locale_t;
///	
///	
///	
///	
///	unsigned long ___runetype_l(__ct_rune_t, locale_t) __attribute__((__pure__));
///	__ct_rune_t ___tolower_l(__ct_rune_t, locale_t) __attribute__((__pure__));
///	__ct_rune_t ___toupper_l(__ct_rune_t, locale_t) __attribute__((__pure__));
///	_RuneLocale *__runes_for_locale(locale_t, int*);
///	
///	inline int
///	__sbmaskrune_l(__ct_rune_t __c, unsigned long __f, locale_t __loc);
///	inline int
///	__sbistype_l(__ct_rune_t __c, unsigned long __f, locale_t __loc);
///	
///	inline int
///	__sbmaskrune_l(__ct_rune_t __c, unsigned long __f, locale_t __loc)
///	{
///	 int __limit;
///	 _RuneLocale *runes = __runes_for_locale(__loc, &__limit);
///	 return (__c < 0 || __c >= __limit) ? 0 :
///	        runes->__runetype[__c] & __f;
///	}
///	
///	inline int
///	__sbistype_l(__ct_rune_t __c, unsigned long __f, locale_t __loc)
///	{
///	 return (!!__sbmaskrune_l(__c, __f, __loc));
///	}
///	
///	
///	
///	
///	
///	
///	
///	inline int isalnum_l(int, locale_t); inline int isalnum_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00000100L|0x00000400L|0x00400000L, __l); }
///	inline int isalpha_l(int, locale_t); inline int isalpha_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00000100L, __l); }
///	inline int isblank_l(int, locale_t); inline int isblank_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00020000L, __l); }
///	inline int iscntrl_l(int, locale_t); inline int iscntrl_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00000200L, __l); }
///	inline int isdigit_l(int, locale_t); inline int isdigit_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00000400L, __l); }
///	inline int isgraph_l(int, locale_t); inline int isgraph_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00000800L, __l); }
///	inline int ishexnumber_l(int, locale_t); inline int ishexnumber_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00010000L, __l); }
///	inline int isideogram_l(int, locale_t); inline int isideogram_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00080000L, __l); }
///	inline int islower_l(int, locale_t); inline int islower_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00001000L, __l); }
///	inline int isnumber_l(int, locale_t); inline int isnumber_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00000400L|0x00400000L, __l); }
///	inline int isphonogram_l(int, locale_t); inline int isphonogram_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00200000L, __l); }
///	inline int isprint_l(int, locale_t); inline int isprint_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00040000L, __l); }
///	inline int ispunct_l(int, locale_t); inline int ispunct_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00002000L, __l); }
///	inline int isrune_l(int, locale_t); inline int isrune_l(int __c, locale_t __l) { return __sbistype_l(__c, 0xFFFFFF00L, __l); }
///	inline int isspace_l(int, locale_t); inline int isspace_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00004000L, __l); }
///	inline int isspecial_l(int, locale_t); inline int isspecial_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00100000L, __l); }
///	inline int isupper_l(int, locale_t); inline int isupper_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00008000L, __l); }
///	inline int isxdigit_l(int, locale_t); inline int isxdigit_l(int __c, locale_t __l) { return __sbistype_l(__c, 0x00010000L, __l); }
///	
///	inline int digittoint_l(int, locale_t);
///	inline int tolower_l(int, locale_t);
///	inline int toupper_l(int, locale_t);
///	
///	inline int digittoint_l(int __c, locale_t __l)
///	{ return __sbmaskrune_l((__c), 0xFF, __l); }
///	
///	inline int tolower_l(int __c, locale_t __l)
///	{
///	 int __limit;
///	 _RuneLocale *__runes = __runes_for_locale(__l, &__limit);
///	 return (__c < 0 || __c >= __limit) ? __c :
///	        __runes->__maplower[__c];
///	}
///	inline int toupper_l(int __c, locale_t __l)
///	{
///	 int __limit;
///	 _RuneLocale *__runes = __runes_for_locale(__l, &__limit);
///	 return (__c < 0 || __c >= __limit) ? __c :
///	        __runes->__mapupper[__c];
///	}
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	typedef __mbstate_t mbstate_t;
///	
///	
///	
///	
///	typedef __size_t size_t;
///	
///	
///	
///	
///	
///	typedef __va_list va_list;
///	
///	
///	
///	
///	
///	
///	typedef ___wchar_t wchar_t;
///	
///	
///	
///	
///	
///	typedef __wint_t wint_t;
///	
///	typedef struct __sFILE FILE;
///	
///	struct tm;
///	
///	
///	wint_t btowc(int);
///	wint_t fgetwc(FILE *);
///	wchar_t *
///	 fgetws(wchar_t * restrict, int, FILE * restrict);
///	wint_t fputwc(wchar_t, FILE *);
///	int fputws(const wchar_t * restrict, FILE * restrict);
///	int fwide(FILE *, int);
///	int fwprintf(FILE * restrict, const wchar_t * restrict, ...);
///	int fwscanf(FILE * restrict, const wchar_t * restrict, ...);
///	wint_t getwc(FILE *);
///	wint_t getwchar(void);
///	size_t mbrlen(const char * restrict, size_t, mbstate_t * restrict);
///	size_t mbrtowc(wchar_t * restrict, const char * restrict, size_t,
///	     mbstate_t * restrict);
///	int mbsinit(const mbstate_t *);
///	size_t mbsrtowcs(wchar_t * restrict, const char ** restrict, size_t,
///	     mbstate_t * restrict);
///	wint_t putwc(wchar_t, FILE *);
///	wint_t putwchar(wchar_t);
///	int swprintf(wchar_t * restrict, size_t n, const wchar_t * restrict,
///	     ...);
///	int swscanf(const wchar_t * restrict, const wchar_t * restrict, ...);
///	wint_t ungetwc(wint_t, FILE *);
///	int vfwprintf(FILE * restrict, const wchar_t * restrict,
///	     __va_list);
///	int vswprintf(wchar_t * restrict, size_t n, const wchar_t * restrict,
///	     __va_list);
///	int vwprintf(const wchar_t * restrict, __va_list);
///	size_t wcrtomb(char * restrict, wchar_t, mbstate_t * restrict);
///	wchar_t *wcscat(wchar_t * restrict, const wchar_t * restrict);
///	wchar_t *wcschr(const wchar_t *, wchar_t) __attribute__((__pure__));
///	int wcscmp(const wchar_t *, const wchar_t *) __attribute__((__pure__));
///	int wcscoll(const wchar_t *, const wchar_t *);
///	wchar_t *wcscpy(wchar_t * restrict, const wchar_t * restrict);
///	size_t wcscspn(const wchar_t *, const wchar_t *) __attribute__((__pure__));
///	size_t wcsftime(wchar_t * restrict, size_t, const wchar_t * restrict,
///	     const struct tm * restrict);
///	size_t wcslen(const wchar_t *) __attribute__((__pure__));
///	wchar_t *wcsncat(wchar_t * restrict, const wchar_t * restrict,
///	     size_t);
///	int wcsncmp(const wchar_t *, const wchar_t *, size_t) __attribute__((__pure__));
///	wchar_t *wcsncpy(wchar_t * restrict , const wchar_t * restrict, size_t);
///	wchar_t *wcspbrk(const wchar_t *, const wchar_t *) __attribute__((__pure__));
///	wchar_t *wcsrchr(const wchar_t *, wchar_t) __attribute__((__pure__));
///	size_t wcsrtombs(char * restrict, const wchar_t ** restrict, size_t,
///	     mbstate_t * restrict);
///	size_t wcsspn(const wchar_t *, const wchar_t *) __attribute__((__pure__));
///	wchar_t *wcsstr(const wchar_t * restrict, const wchar_t * restrict)
///	     __attribute__((__pure__));
///	size_t wcsxfrm(wchar_t * restrict, const wchar_t * restrict, size_t);
///	int wctob(wint_t);
///	double wcstod(const wchar_t * restrict, wchar_t ** restrict);
///	wchar_t *wcstok(wchar_t * restrict, const wchar_t * restrict,
///	     wchar_t ** restrict);
///	long wcstol(const wchar_t * restrict, wchar_t ** restrict, int);
///	unsigned long
///	  wcstoul(const wchar_t * restrict, wchar_t ** restrict, int);
///	wchar_t *wmemchr(const wchar_t *, wchar_t, size_t) __attribute__((__pure__));
///	int wmemcmp(const wchar_t *, const wchar_t *, size_t) __attribute__((__pure__));
///	wchar_t *wmemcpy(wchar_t * restrict, const wchar_t * restrict, size_t);
///	wchar_t *wmemmove(wchar_t *, const wchar_t *, size_t);
///	wchar_t *wmemset(wchar_t *, wchar_t, size_t);
///	int wprintf(const wchar_t * restrict, ...);
///	int wscanf(const wchar_t * restrict, ...);
///	
///	
///	extern FILE *__stdinp;
///	extern FILE *__stdoutp;
///	extern FILE *__stderrp;
///	
///	int vfwscanf(FILE * restrict, const wchar_t * restrict,
///	     __va_list);
///	int vswscanf(const wchar_t * restrict, const wchar_t * restrict,
///	     __va_list);
///	int vwscanf(const wchar_t * restrict, __va_list);
///	float wcstof(const wchar_t * restrict, wchar_t ** restrict);
///	long double
///	 wcstold(const wchar_t * restrict, wchar_t ** restrict);
///	
///	
///	long long
///	 wcstoll(const wchar_t * restrict, wchar_t ** restrict, int);
///	
///	unsigned long long
///	  wcstoull(const wchar_t * restrict, wchar_t ** restrict, int);
///	
///	
///	
///	
///	int wcswidth(const wchar_t *, size_t);
///	int wcwidth(wchar_t);
///	
///	
///	
///	
///	size_t mbsnrtowcs(wchar_t * restrict, const char ** restrict, size_t,
///	     size_t, mbstate_t * restrict);
///	FILE *open_wmemstream(wchar_t **, size_t *);
///	wchar_t *wcpcpy(wchar_t * restrict, const wchar_t * restrict);
///	wchar_t *wcpncpy(wchar_t * restrict, const wchar_t * restrict, size_t);
///	wchar_t *wcsdup(const wchar_t *) __attribute__((__malloc__));
///	int wcscasecmp(const wchar_t *, const wchar_t *);
///	int wcsncasecmp(const wchar_t *, const wchar_t *, size_t n);
///	size_t wcsnlen(const wchar_t *, size_t) __attribute__((__pure__));
///	size_t wcsnrtombs(char * restrict, const wchar_t ** restrict, size_t,
///	     size_t, mbstate_t * restrict);
///	
///	
///	
///	wchar_t *fgetwln(FILE * restrict, size_t * restrict);
///	size_t wcslcat(wchar_t *, const wchar_t *, size_t);
///	size_t wcslcpy(wchar_t *, const wchar_t *, size_t);
///	
///	
///	
///	
///	
///	int wcscasecmp_l(const wchar_t *, const wchar_t *,
///	      locale_t);
///	int wcsncasecmp_l(const wchar_t *, const wchar_t *, size_t,
///	      locale_t);
///	int wcscoll_l(const wchar_t *, const wchar_t *, locale_t);
///	size_t wcsxfrm_l(wchar_t * restrict,
///	      const wchar_t * restrict, size_t, locale_t);
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	struct lconv {
///	 char *decimal_point;
///	 char *thousands_sep;
///	 char *grouping;
///	 char *int_curr_symbol;
///	 char *currency_symbol;
///	 char *mon_decimal_point;
///	 char *mon_thousands_sep;
///	 char *mon_grouping;
///	 char *positive_sign;
///	 char *negative_sign;
///	 char int_frac_digits;
///	 char frac_digits;
///	 char p_cs_precedes;
///	 char p_sep_by_space;
///	 char n_cs_precedes;
///	 char n_sep_by_space;
///	 char p_sign_posn;
///	 char n_sign_posn;
///	 char int_p_cs_precedes;
///	 char int_n_cs_precedes;
///	 char int_p_sep_by_space;
///	 char int_n_sep_by_space;
///	 char int_p_sign_posn;
///	 char int_n_sign_posn;
///	};
///	
///	
///	struct lconv *localeconv(void);
///	char *setlocale(int, const char *);
///	
///	
///	
///	
///	locale_t duplocale(locale_t base);
///	void freelocale(locale_t loc);
///	locale_t newlocale(int mask, const char *locale, locale_t base);
///	const char *querylocale(int mask, locale_t loc);
///	locale_t uselocale(locale_t loc);
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	wint_t btowc_l(int, locale_t);
///	wint_t fgetwc_l(FILE *, locale_t);
///	wchar_t *fgetws_l(wchar_t * restrict, int, FILE * restrict,
///	       locale_t);
///	wint_t fputwc_l(wchar_t, FILE *, locale_t);
///	int fputws_l(const wchar_t * restrict, FILE * restrict,
///	      locale_t);
///	int fwprintf_l(FILE * restrict, locale_t,
///	       const wchar_t * restrict, ...);
///	int fwscanf_l(FILE * restrict, locale_t,
///	       const wchar_t * restrict, ...);
///	wint_t getwc_l(FILE *, locale_t);
///	wint_t getwchar_l(locale_t);
///	size_t mbrlen_l(const char * restrict, size_t,
///	      mbstate_t * restrict, locale_t);
///	size_t mbrtowc_l(wchar_t * restrict,
///	       const char * restrict, size_t,
///	       mbstate_t * restrict, locale_t);
///	int mbsinit_l(const mbstate_t *, locale_t);
///	size_t mbsrtowcs_l(wchar_t * restrict,
///	       const char ** restrict, size_t,
///	       mbstate_t * restrict, locale_t);
///	wint_t putwc_l(wchar_t, FILE *, locale_t);
///	wint_t putwchar_l(wchar_t, locale_t);
///	int swprintf_l(wchar_t * restrict, size_t n, locale_t,
///	       const wchar_t * restrict, ...);
///	int swscanf_l(const wchar_t * restrict, locale_t,
///	      const wchar_t * restrict, ...);
///	wint_t ungetwc_l(wint_t, FILE *, locale_t);
///	int vfwprintf_l(FILE * restrict, locale_t,
///	       const wchar_t * restrict, __va_list);
///	int vswprintf_l(wchar_t * restrict, size_t n, locale_t,
///	       const wchar_t * restrict, __va_list);
///	int vwprintf_l(locale_t, const wchar_t * restrict,
///	       __va_list);
///	size_t wcrtomb_l(char * restrict, wchar_t,
///	       mbstate_t * restrict, locale_t);
///	size_t wcsftime_l(wchar_t * restrict, size_t,
///	       const wchar_t * restrict,
///	       const struct tm * restrict, locale_t);
///	size_t wcsrtombs_l(char * restrict,
///	       const wchar_t ** restrict, size_t,
///	       mbstate_t * restrict, locale_t);
///	double wcstod_l(const wchar_t * restrict,
///	       wchar_t ** restrict, locale_t);
///	long wcstol_l(const wchar_t * restrict,
///	       wchar_t ** restrict, int, locale_t);
///	unsigned long wcstoul_l(const wchar_t * restrict,
///	       wchar_t ** restrict, int, locale_t);
///	int wcswidth_l(const wchar_t *, size_t, locale_t);
///	int wctob_l(wint_t, locale_t);
///	int wcwidth_l(wchar_t, locale_t);
///	int wprintf_l(locale_t, const wchar_t * restrict, ...);
///	int wscanf_l(locale_t, const wchar_t * restrict, ...);
///	int vfwscanf_l(FILE * restrict, locale_t,
///	       const wchar_t * restrict, __va_list);
///	int vswscanf_l(const wchar_t * restrict, locale_t,
///	       const wchar_t *restrict, __va_list);
///	int vwscanf_l(locale_t, const wchar_t * restrict,
///	       __va_list);
///	float wcstof_l(const wchar_t * restrict,
///	       wchar_t ** restrict, locale_t);
///	long double wcstold_l(const wchar_t * restrict,
///	       wchar_t ** restrict, locale_t);
///	long long wcstoll_l(const wchar_t * restrict,
///	       wchar_t ** restrict, int, locale_t);
///	unsigned long long wcstoull_l(const wchar_t * restrict,
///	       wchar_t ** restrict, int, locale_t);
///	size_t mbsnrtowcs_l(wchar_t * restrict,
///	       const char ** restrict, size_t, size_t,
///	       mbstate_t * restrict, locale_t);
///	size_t wcsnrtombs_l(char * restrict,
///	       const wchar_t ** restrict, size_t, size_t,
///	       mbstate_t * restrict, locale_t);
///	
///	
///	
///	
///	
///	struct lconv *localeconv_l(locale_t);
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	typedef __rune_t rune_t;
///	
///	typedef struct {
///	 int quot;
///	 int rem;
///	} div_t;
///	
///	typedef struct {
///	 long quot;
///	 long rem;
///	} ldiv_t;
///	
///	
///	
///	
///	
///	double atof_l(const char *, locale_t);
///	int atoi_l(const char *, locale_t);
///	long atol_l(const char *, locale_t);
///	long long atoll_l(const char *, locale_t);
///	int mblen_l(const char *, size_t, locale_t);
///	size_t mbstowcs_l(wchar_t * restrict,
///	       const char * restrict, size_t, locale_t);
///	int mbtowc_l(wchar_t * restrict,
///	       const char * restrict, size_t, locale_t);
///	double strtod_l(const char *, char **, locale_t);
///	float strtof_l(const char *, char **, locale_t);
///	long strtol_l(const char *, char **, int, locale_t);
///	long double strtold_l(const char *, char **, locale_t);
///	long long strtoll_l(const char *, char **, int, locale_t);
///	unsigned long strtoul_l(const char *, char **, int, locale_t);
///	unsigned long long strtoull_l(const char *, char **, int, locale_t);
///	size_t wcstombs_l(char * restrict,
///	       const wchar_t * restrict, size_t, locale_t);
///	int wctomb_l(char *, wchar_t, locale_t);
///	
///	int ___mb_cur_max_l(locale_t);
///	
///	
///	extern int __mb_cur_max;
///	extern int ___mb_cur_max(void);
///	
///	
///	_Noreturn void abort(void);
///	int abs(int) __attribute__((__const__));
///	int atexit(void (* )(void));
///	double atof(const char *);
///	int atoi(const char *);
///	long atol(const char *);
///	void *bsearch(const void *, const void *, size_t,
///	     size_t, int (*)(const void * , const void *));
///	void *calloc(size_t, size_t) __attribute__((__malloc__)) __attribute__((__warn_unused_result__))
///	      __attribute__((__alloc_size__(1, 2)));
///	div_t div(int, int) __attribute__((__const__));
///	_Noreturn void exit(int);
///	void free(void *);
///	char *getenv(const char *);
///	long labs(long) __attribute__((__const__));
///	ldiv_t ldiv(long, long) __attribute__((__const__));
///	void *malloc(size_t) __attribute__((__malloc__)) __attribute__((__warn_unused_result__)) __attribute__((__alloc_size__(1)));
///	int mblen(const char *, size_t);
///	size_t mbstowcs(wchar_t * restrict , const char * restrict, size_t);
///	int mbtowc(wchar_t * restrict, const char * restrict, size_t);
///	void qsort(void *, size_t, size_t,
///	     int (* )(const void *, const void *));
///	int rand(void);
///	void *realloc(void *, size_t) __attribute__((__warn_unused_result__)) __attribute__((__alloc_size__(2)));
///	void srand(unsigned);
///	double strtod(const char * restrict, char ** restrict);
///	float strtof(const char * restrict, char ** restrict);
///	long strtol(const char * restrict, char ** restrict, int);
///	long double
///	  strtold(const char * restrict, char ** restrict);
///	unsigned long
///	  strtoul(const char * restrict, char ** restrict, int);
///	int system(const char *);
///	int wctomb(char *, wchar_t);
///	size_t wcstombs(char * restrict, const wchar_t * restrict, size_t);
///	
///	typedef struct {
///	 long long quot;
///	 long long rem;
///	} lldiv_t;
///	
///	
///	long long
///	  atoll(const char *);
///	
///	long long
///	  llabs(long long) __attribute__((__const__));
///	
///	lldiv_t lldiv(long long, long long) __attribute__((__const__));
///	
///	long long
///	  strtoll(const char * restrict, char ** restrict, int);
///	
///	unsigned long long
///	  strtoull(const char * restrict, char ** restrict, int);
///	
///	
///	_Noreturn void _Exit(int);
///	
///	
///	
///	
///	
///	
///	void * aligned_alloc(size_t, size_t) __attribute__((__malloc__)) __attribute__((__alloc_align__(1)))
///	     __attribute__((__alloc_size__(2)));
///	int at_quick_exit(void (*)(void));
///	_Noreturn void
///	 quick_exit(int);
///	
///	
///	
///	
///	
///	char *realpath(const char * restrict, char * restrict);
///	
///	
///	int rand_r(unsigned *);
///	
///	
///	int posix_memalign(void **, size_t, size_t);
///	int setenv(const char *, const char *, int);
///	int unsetenv(const char *);
///	
///	
///	
///	int getsubopt(char **, char *const *, char **);
///	
///	char *mkdtemp(char *);
///	
///	
///	
///	int mkstemp(char *);
///	
///	long a64l(const char *);
///	double drand48(void);
///	
///	double erand48(unsigned short[3]);
///	
///	
///	char *initstate(unsigned int, char *, size_t);
///	long jrand48(unsigned short[3]);
///	char *l64a(long);
///	void lcong48(unsigned short[7]);
///	long lrand48(void);
///	
///	char *mktemp(char *);
///	
///	
///	long mrand48(void);
///	long nrand48(unsigned short[3]);
///	int putenv(char *);
///	long random(void);
///	unsigned short
///	 *seed48(unsigned short[3]);
///	char *setstate( char *);
///	void srand48(long);
///	void srandom(unsigned int);
///	
///	
///	
///	int grantpt(int);
///	int posix_openpt(int);
///	char *ptsname(int);
///	int unlockpt(int);
///	
///	
///	
///	int ptsname_r(int, char *, size_t);
///	
///	
///	
///	extern const char *malloc_conf;
///	extern void (*malloc_message)(void *, const char *);
///	
///	void abort2(const char *, int, void **) __attribute__((__noreturn__));
///	__uint32_t
///	  arc4random(void);
///	void arc4random_buf(void *, size_t);
///	__uint32_t
///	  arc4random_uniform(__uint32_t);
///	
///	
///	
///	
///	
///	
///	char *getbsize(int *, long *);
///	
///	char *cgetcap(char *, const char *, int);
///	int cgetclose(void);
///	int cgetent(char **, char **, const char *);
///	int cgetfirst(char **, char **);
///	int cgetmatch(const char *, const char *);
///	int cgetnext(char **, char **);
///	int cgetnum(char *, const char *, long *);
///	int cgetset(const char *);
///	int cgetstr(char *, const char *, char **);
///	int cgetustr(char *, const char *, char **);
///	
///	int daemon(int, int);
///	int daemonfd(int, int);
///	char *devname(__dev_t, __mode_t);
///	char *devname_r(__dev_t, __mode_t, char *, int);
///	char *fdevname(int);
///	char *fdevname_r(int, char *, int);
///	int getloadavg(double [], int);
///	const char *
///	  getprogname(void);
///	
///	int heapsort(void *, size_t, size_t,
///	     int (* )(const void *, const void *));
///	
///	
///	
///	
///	
///	
///	int l64a_r(long, char *, int);
///	int mergesort(void *, size_t, size_t, int (*)(const void *, const void *));
///	
///	
///	
///	int mkostemp(char *, int);
///	int mkostemps(char *, int, int);
///	int mkostempsat(int, char *, int, int);
///	void qsort_r(void *, size_t, size_t, void *,
///	     int (*)(void *, const void *, const void *));
///	int radixsort(const unsigned char **, int, const unsigned char *,
///	     unsigned);
///	void *reallocarray(void *, size_t, size_t) __attribute__((__warn_unused_result__))
///	     __attribute__((__alloc_size__(2, 3)));
///	void *reallocf(void *, size_t) __attribute__((__warn_unused_result__)) __attribute__((__alloc_size__(2)));
///	int rpmatch(const char *);
///	void setprogname(const char *);
///	int sradixsort(const unsigned char **, int, const unsigned char *,
///	     unsigned);
///	void srandomdev(void);
///	long long
///	 strtonum(const char *, long long, long long, const char **);
///	
///	
///	__int64_t
///	  strtoq(const char *, char **, int);
///	__uint64_t
///	  strtouq(const char *, char **, int);
///	
///	extern char *suboptarg;
///	
///	
///	
///	
///	
///	
///	typedef size_t rsize_t;
///	
///	
///	
///	
///	typedef int errno_t;
///	
///	
///	
///	typedef void (*constraint_handler_t)(const char * restrict,
///	    void * restrict, errno_t);
///	
///	constraint_handler_t set_constraint_handler_s(constraint_handler_t handler);
///	
///	_Noreturn void abort_handler_s(const char * restrict, void * restrict,
///	    errno_t);
///	
///	void ignore_handler_s(const char * restrict, void * restrict, errno_t);
///	
///	errno_t qsort_s(void *, rsize_t, rsize_t,
///	    int (*)(const void *, const void *, void *), void *);
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	typedef __int8_t int8_t;
///	
///	
///	
///	
///	typedef __int16_t int16_t;
///	
///	
///	
///	
///	typedef __int32_t int32_t;
///	
///	
///	
///	
///	typedef __int64_t int64_t;
///	
///	
///	
///	
///	typedef __uint8_t uint8_t;
///	
///	
///	
///	
///	typedef __uint16_t uint16_t;
///	
///	
///	
///	
///	typedef __uint32_t uint32_t;
///	
///	
///	
///	
///	typedef __uint64_t uint64_t;
///	
///	
///	
///	
///	typedef __intptr_t intptr_t;
///	
///	
///	
///	typedef __uintptr_t uintptr_t;
///	
///	
///	
///	typedef __intmax_t intmax_t;
///	
///	
///	
///	typedef __uintmax_t uintmax_t;
///	
///	
///	typedef __int_least8_t int_least8_t;
///	typedef __int_least16_t int_least16_t;
///	typedef __int_least32_t int_least32_t;
///	typedef __int_least64_t int_least64_t;
///	
///	typedef __uint_least8_t uint_least8_t;
///	typedef __uint_least16_t uint_least16_t;
///	typedef __uint_least32_t uint_least32_t;
///	typedef __uint_least64_t uint_least64_t;
///	
///	typedef __int_fast8_t int_fast8_t;
///	typedef __int_fast16_t int_fast16_t;
///	typedef __int_fast32_t int_fast32_t;
///	typedef __int_fast64_t int_fast64_t;
///	
///	typedef __uint_fast8_t uint_fast8_t;
///	typedef __uint_fast16_t uint_fast16_t;
///	typedef __uint_fast32_t uint_fast32_t;
///	typedef __uint_fast64_t uint_fast64_t;
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	struct pthread;
///	struct pthread_attr;
///	struct pthread_cond;
///	struct pthread_cond_attr;
///	struct pthread_mutex;
///	struct pthread_mutex_attr;
///	struct pthread_once;
///	struct pthread_rwlock;
///	struct pthread_rwlockattr;
///	struct pthread_barrier;
///	struct pthread_barrier_attr;
///	struct pthread_spinlock;
///	
///	typedef struct pthread *pthread_t;
///	
///	
///	typedef struct pthread_attr *pthread_attr_t;
///	typedef struct pthread_mutex *pthread_mutex_t;
///	typedef struct pthread_mutex_attr *pthread_mutexattr_t;
///	typedef struct pthread_cond *pthread_cond_t;
///	typedef struct pthread_cond_attr *pthread_condattr_t;
///	typedef int pthread_key_t;
///	typedef struct pthread_once pthread_once_t;
///	typedef struct pthread_rwlock *pthread_rwlock_t;
///	typedef struct pthread_rwlockattr *pthread_rwlockattr_t;
///	typedef struct pthread_barrier *pthread_barrier_t;
///	typedef struct pthread_barrierattr *pthread_barrierattr_t;
///	typedef struct pthread_spinlock *pthread_spinlock_t;
///	
///	
///	
///	
///	
///	
///	
///	typedef void *pthread_addr_t;
///	typedef void *(*pthread_startroutine_t)(void *);
///	
///	
///	
///	
///	struct pthread_once {
///	 int state;
///	 pthread_mutex_t mutex;
///	};
///	
///	
///	
///	typedef unsigned char u_char;
///	typedef unsigned short u_short;
///	typedef unsigned int u_int;
///	typedef unsigned long u_long;
///	
///	typedef unsigned short ushort;
///	typedef unsigned int uint;
///	
///	typedef __uint8_t u_int8_t;
///	typedef __uint16_t u_int16_t;
///	typedef __uint32_t u_int32_t;
///	typedef __uint64_t u_int64_t;
///	
///	typedef __uint64_t u_quad_t;
///	typedef __int64_t quad_t;
///	typedef quad_t * qaddr_t;
///	
///	typedef char * caddr_t;
///	typedef const char * c_caddr_t;
///	
///	
///	typedef __blksize_t blksize_t;
///	
///	
///	
///	typedef __cpuwhich_t cpuwhich_t;
///	typedef __cpulevel_t cpulevel_t;
///	typedef __cpusetid_t cpusetid_t;
///	
///	
///	typedef __blkcnt_t blkcnt_t;
///	
///	
///	
///	
///	typedef __clock_t clock_t;
///	
///	
///	
///	
///	typedef __clockid_t clockid_t;
///	
///	
///	
///	typedef __critical_t critical_t;
///	typedef __daddr_t daddr_t;
///	
///	
///	typedef __dev_t dev_t;
///	
///	
///	
///	
///	typedef __fflags_t fflags_t;
///	
///	
///	
///	typedef __fixpt_t fixpt_t;
///	
///	
///	typedef __fsblkcnt_t fsblkcnt_t;
///	typedef __fsfilcnt_t fsfilcnt_t;
///	
///	
///	
///	
///	typedef __gid_t gid_t;
///	
///	
///	
///	
///	typedef __uint32_t in_addr_t;
///	
///	
///	
///	
///	typedef __uint16_t in_port_t;
///	
///	
///	
///	
///	typedef __id_t id_t;
///	
///	
///	
///	
///	typedef __ino_t ino_t;
///	
///	
///	
///	
///	typedef __key_t key_t;
///	
///	
///	
///	
///	typedef __lwpid_t lwpid_t;
///	
///	
///	
///	
///	typedef __mode_t mode_t;
///	
///	
///	
///	
///	typedef __accmode_t accmode_t;
///	
///	
///	
///	
///	typedef __nlink_t nlink_t;
///	
///	
///	
///	
///	typedef __off_t off_t;
///	
///	
///	
///	
///	typedef __off64_t off64_t;
///	
///	
///	
///	
///	typedef __pid_t pid_t;
///	
///	
///	
///	typedef __register_t register_t;
///	
///	
///	typedef __rlim_t rlim_t;
///	
///	
///	
///	typedef __int64_t sbintime_t;
///	
///	typedef __segsz_t segsz_t;
///	
///	
///	
///	
///	
///	
///	
///	typedef __ssize_t ssize_t;
///	
///	
///	
///	
///	typedef __suseconds_t suseconds_t;
///	
///	
///	
///	
///	typedef __time_t time_t;
///	
///	
///	
///	
///	typedef __timer_t timer_t;
///	
///	
///	
///	
///	typedef __mqd_t mqd_t;
///	
///	
///	
///	typedef __u_register_t u_register_t;
///	
///	
///	typedef __uid_t uid_t;
///	
///	
///	
///	
///	typedef __useconds_t useconds_t;
///	
///	
///	
///	
///	
///	typedef unsigned long cap_ioctl_t;
///	
///	
///	
///	
///	struct cap_rights;
///	
///	typedef struct cap_rights cap_rights_t;
///	
///	typedef __uint64_t kpaddr_t;
///	typedef __uint64_t kvaddr_t;
///	typedef __uint64_t ksize_t;
///	typedef __int64_t kssize_t;
///	
///	typedef __vm_offset_t vm_offset_t;
///	typedef __uint64_t vm_ooffset_t;
///	typedef __vm_paddr_t vm_paddr_t;
///	typedef __uint64_t vm_pindex_t;
///	typedef __vm_size_t vm_size_t;
///	
///	typedef __rman_res_t rman_res_t;
///	
///	static __inline __uint16_t
///	__bitcount16(__uint16_t _x)
///	{
///	
///	 _x = (_x & 0x5555) + ((_x & 0xaaaa) >> 1);
///	 _x = (_x & 0x3333) + ((_x & 0xcccc) >> 2);
///	 _x = (_x + (_x >> 4)) & 0x0f0f;
///	 _x = (_x + (_x >> 8)) & 0x00ff;
///	 return (_x);
///	}
///	
///	static __inline __uint32_t
///	__bitcount32(__uint32_t _x)
///	{
///	
///	 _x = (_x & 0x55555555) + ((_x & 0xaaaaaaaa) >> 1);
///	 _x = (_x & 0x33333333) + ((_x & 0xcccccccc) >> 2);
///	 _x = (_x + (_x >> 4)) & 0x0f0f0f0f;
///	 _x = (_x + (_x >> 8));
///	 _x = (_x + (_x >> 16)) & 0x000000ff;
///	 return (_x);
///	}
///	
///	
///	static __inline __uint64_t
///	__bitcount64(__uint64_t _x)
///	{
///	
///	 _x = (_x & 0x5555555555555555) + ((_x & 0xaaaaaaaaaaaaaaaa) >> 1);
///	 _x = (_x & 0x3333333333333333) + ((_x & 0xcccccccccccccccc) >> 2);
///	 _x = (_x + (_x >> 4)) & 0x0f0f0f0f0f0f0f0f;
///	 _x = (_x + (_x >> 8));
///	 _x = (_x + (_x >> 16));
///	 _x = (_x + (_x >> 32)) & 0x000000ff;
///	 return (_x);
///	}
///	
///	
///	
///	
///	
///	typedef struct __sigset {
///	 __uint32_t __bits[4];
///	} __sigset_t;
///	
///	
///	
///	struct timeval {
///	 time_t tv_sec;
///	 suseconds_t tv_usec;
///	};
///	
///	
///	
///	
///	
///	struct timespec {
///	 time_t tv_sec;
///	 long tv_nsec;
///	};
///	
///	
///	struct itimerspec {
///	 struct timespec it_interval;
///	 struct timespec it_value;
///	};
///	
///	
///	typedef unsigned long __fd_mask;
///	
///	typedef __fd_mask fd_mask;
///	
///	
///	
///	
///	typedef __sigset_t sigset_t;
///	
///	typedef struct fd_set {
///	 __fd_mask __fds_bits[(((1024) + (((sizeof(__fd_mask) * 8)) - 1)) / ((sizeof(__fd_mask) * 8)))];
///	} fd_set;
///	
///	
///	int pselect(int, fd_set *restrict, fd_set *restrict, fd_set *restrict,
///	 const struct timespec *restrict, const sigset_t *restrict);
///	
///	
///	
///	int select(int, fd_set *, fd_set *, fd_set *, struct timeval *);
///	
///	
///	
///	
///	static __inline int
///	__major(dev_t _d)
///	{
///	 return (((_d >> 32) & 0xffffff00) | ((_d >> 8) & 0xff));
///	}
///	
///	static __inline int
///	__minor(dev_t _d)
///	{
///	 return (((_d >> 24) & 0xff00) | (_d & 0xffff00ff));
///	}
///	
///	static __inline dev_t
///	__makedev(int _Major, int _Minor)
///	{
///	 return (((dev_t)(_Major & 0xffffff00) << 32) | ((_Major & 0xff) << 8) |
///	     ((dev_t)(_Minor & 0xff00) << 24) | (_Minor & 0xffff00ff));
///	}
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	
///	int ftruncate(int, off_t);
///	
///	
///	
///	off_t lseek(int, off_t, int);
///	
///	
///	
///	void * mmap(void *, size_t, int, int, int, off_t);
///	
///	
///	
///	int truncate(const char *, off_t);
///	
///	
///	
///	
///	
///	
///	
///	
///	static __inline int atomic_cmpset_char(volatile u_char *dst, u_char expect, u_char src) { u_char res; __asm volatile( "	" "lock ; " "		" "	cmpxchg %3,%1 ;	" "# atomic_cmpset_" "char" "	" : "=@cce" (res), "+m" (*dst), "+a" (expect) : "r" (src) : "memory", "cc"); return (res); } static __inline int atomic_fcmpset_char(volatile u_char *dst, u_char *expect, u_char src) { u_char res; __asm volatile( "	" "lock ; " "		" "	cmpxchg %3,%1 ;		" "# atomic_fcmpset_" "char" "	" : "=@cce" (res), "+m" (*dst), "+a" (*expect) : "r" (src) : "memory", "cc"); return (res); };
///	static __inline int atomic_cmpset_short(volatile u_short *dst, u_short expect, u_short src) { u_char res; __asm volatile( "	" "lock ; " "		" "	cmpxchg %3,%1 ;	" "# atomic_cmpset_" "short" "	" : "=@cce" (res), "+m" (*dst), "+a" (expect) : "r" (src) : "memory", "cc"); return (res); } static __inline int atomic_fcmpset_short(volatile u_short *dst, u_short *expect, u_short src) { u_char res; __asm volatile( "	" "lock ; " "		" "	cmpxchg %3,%1 ;		" "# atomic_fcmpset_" "short" "	" : "=@cce" (res), "+m" (*dst), "+a" (*expect) : "r" (src) : "memory", "cc"); return (res); };
///	static __inline int atomic_cmpset_int(volatile u_int *dst, u_int expect, u_int src) { u_char res; __asm volatile( "	" "lock ; " "		" "	cmpxchg %3,%1 ;	" "# atomic_cmpset_" "int" "	" : "=@cce" (res), "+m" (*dst), "+a" (expect) : "r" (src) : "memory", "cc"); return (res); } static __inline int atomic_fcmpset_int(volatile u_int *dst, u_int *expect, u_int src) { u_char res; __asm volatile( "	" "lock ; " "		" "	cmpxchg %3,%1 ;		" "# atomic_fcmpset_" "int" "	" : "=@cce" (res), "+m" (*dst), "+a" (*expect) : "r" (src) : "memory", "cc"); return (res); };
///	static __inline int atomic_cmpset_long(volatile u_long *dst, u_long expect, u_long src) { u_char res; __asm volatile( "	" "lock ; " "		" "	cmpxchg %3,%1 ;	" "# atomic_cmpset_" "long" "	" : "=@cce" (res), "+m" (*dst), "+a" (expect) : "r" (src) : "memory", "cc"); return (res); } static __inline int atomic_fcmpset_long(volatile u_long *dst, u_long *expect, u_long src) { u_char res; __asm volatile( "	" "lock ; " "		" "	cmpxchg %3,%1 ;		" "# atomic_fcmpset_" "long" "	" : "=@cce" (res), "+m" (*dst), "+a" (*expect) : "r" (src) : "memory", "cc"); return (res); };
///	
///	
///	
///	
///	
///	static __inline u_int
///	atomic_fetchadd_int(volatile u_int *p, u_int v)
///	{
///	
///	 __asm volatile(
///	 "	" "lock ; " "		"
///	 "	xaddl	%0,%1 ;		"
///	 "# atomic_fetchadd_int"
///	 : "+r" (v),
///	   "+m" (*p)
///	 : : "cc");
///	 return (v);
///	}
///	
///	
///	
///	
///	
///	static __inline u_long
///	atomic_fetchadd_long(volatile u_long *p, u_long v)
///	{
///	
///	 __asm volatile(
///	 "	" "lock ; " "		"
///	 "	xaddq	%0,%1 ;		"
///	 "# atomic_fetchadd_long"
///	 : "+r" (v),
///	   "+m" (*p)
///	 : : "cc");
///	 return (v);
///	}
///	
///	static __inline int
///	atomic_testandset_int(volatile u_int *p, u_int v)
///	{
///	 u_char res;
///	
///	 __asm volatile(
///	 "	" "lock ; " "		"
///	 "	btsl	%2,%1 ;		"
///	 "# atomic_testandset_int"
///	 : "=@ccc" (res),
///	   "+m" (*p)
///	 : "Ir" (v & 0x1f)
///	 : "cc");
///	 return (res);
///	}
///	
///	static __inline int
///	atomic_testandset_long(volatile u_long *p, u_int v)
///	{
///	 u_char res;
///	
///	 __asm volatile(
///	 "	" "lock ; " "		"
///	 "	btsq	%2,%1 ;		"
///	 "# atomic_testandset_long"
///	 : "=@ccc" (res),
///	   "+m" (*p)
///	 : "Jr" ((u_long)(v & 0x3f))
///	 : "cc");
///	 return (res);
///	}
///	
///	static __inline int
///	atomic_testandclear_int(volatile u_int *p, u_int v)
///	{
///	 u_char res;
///	
///	 __asm volatile(
///	 "	" "lock ; " "		"
///	 "	btrl	%2,%1 ;		"
///	 "# atomic_testandclear_int"
///	 : "=@ccc" (res),
///	   "+m" (*p)
///	 : "Ir" (v & 0x1f)
///	 : "cc");
///	 return (res);
///	}
///	
///	static __inline int
///	atomic_testandclear_long(volatile u_long *p, u_int v)
///	{
///	 u_char res;
///	
///	 __asm volatile(
///	 "	" "lock ; " "		"
///	 "	btrq	%2,%1 ;		"
///	 "# atomic_testandclear_long"
///	 : "=@ccc" (res),
///	   "+m" (*p)
///	 : "Jr" ((u_long)(v & 0x3f))
///	 : "cc");
///	 return (res);
///	}
///	
///	static __inline void
///	__storeload_barrier(void)
///	{
///	
///	 __asm volatile("lock; addl $0,-8(%%rsp)" : : : "memory", "cc");
///	}
///	
///	static __inline void
///	atomic_thread_fence_acq(void)
///	{
///	
///	 __asm volatile(" " : : : "memory");
///	}
///	
///	static __inline void
///	atomic_thread_fence_rel(void)
///	{
///	
///	 __asm volatile(" " : : : "memory");
///	}
///	
///	static __inline void
///	atomic_thread_fence_acq_rel(void)
///	{
///	
///	 __asm volatile(" " : : : "memory");
///	}
///	
///	static __inline void
///	atomic_thread_fence_seq_cst(void)
///	{
///	
///	 __storeload_barrier();
///	}
///	
///	
///	
///	static __inline void atomic_set_char(volatile u_char *p, u_char v){ __asm volatile("lock ; " "orb %b1,%0" : "+m" (*p) : "iq" (v) : "cc"); } static __inline void atomic_set_barr_char(volatile u_char *p, u_char v){ __asm volatile("lock ; " "orb %b1,%0" : "+m" (*p) : "iq" (v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_clear_char(volatile u_char *p, u_char v){ __asm volatile("lock ; " "andb %b1,%0" : "+m" (*p) : "iq" (~v) : "cc"); } static __inline void atomic_clear_barr_char(volatile u_char *p, u_char v){ __asm volatile("lock ; " "andb %b1,%0" : "+m" (*p) : "iq" (~v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_add_char(volatile u_char *p, u_char v){ __asm volatile("lock ; " "addb %b1,%0" : "+m" (*p) : "iq" (v) : "cc"); } static __inline void atomic_add_barr_char(volatile u_char *p, u_char v){ __asm volatile("lock ; " "addb %b1,%0" : "+m" (*p) : "iq" (v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_subtract_char(volatile u_char *p, u_char v){ __asm volatile("lock ; " "subb %b1,%0" : "+m" (*p) : "iq" (v) : "cc"); } static __inline void atomic_subtract_barr_char(volatile u_char *p, u_char v){ __asm volatile("lock ; " "subb %b1,%0" : "+m" (*p) : "iq" (v) : "memory", "cc"); } struct __hack;
///	
///	static __inline void atomic_set_short(volatile u_short *p, u_short v){ __asm volatile("lock ; " "orw %w1,%0" : "+m" (*p) : "ir" (v) : "cc"); } static __inline void atomic_set_barr_short(volatile u_short *p, u_short v){ __asm volatile("lock ; " "orw %w1,%0" : "+m" (*p) : "ir" (v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_clear_short(volatile u_short *p, u_short v){ __asm volatile("lock ; " "andw %w1,%0" : "+m" (*p) : "ir" (~v) : "cc"); } static __inline void atomic_clear_barr_short(volatile u_short *p, u_short v){ __asm volatile("lock ; " "andw %w1,%0" : "+m" (*p) : "ir" (~v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_add_short(volatile u_short *p, u_short v){ __asm volatile("lock ; " "addw %w1,%0" : "+m" (*p) : "ir" (v) : "cc"); } static __inline void atomic_add_barr_short(volatile u_short *p, u_short v){ __asm volatile("lock ; " "addw %w1,%0" : "+m" (*p) : "ir" (v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_subtract_short(volatile u_short *p, u_short v){ __asm volatile("lock ; " "subw %w1,%0" : "+m" (*p) : "ir" (v) : "cc"); } static __inline void atomic_subtract_barr_short(volatile u_short *p, u_short v){ __asm volatile("lock ; " "subw %w1,%0" : "+m" (*p) : "ir" (v) : "memory", "cc"); } struct __hack;
///	
///	static __inline void atomic_set_int(volatile u_int *p, u_int v){ __asm volatile("lock ; " "orl %1,%0" : "+m" (*p) : "ir" (v) : "cc"); } static __inline void atomic_set_barr_int(volatile u_int *p, u_int v){ __asm volatile("lock ; " "orl %1,%0" : "+m" (*p) : "ir" (v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_clear_int(volatile u_int *p, u_int v){ __asm volatile("lock ; " "andl %1,%0" : "+m" (*p) : "ir" (~v) : "cc"); } static __inline void atomic_clear_barr_int(volatile u_int *p, u_int v){ __asm volatile("lock ; " "andl %1,%0" : "+m" (*p) : "ir" (~v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_add_int(volatile u_int *p, u_int v){ __asm volatile("lock ; " "addl %1,%0" : "+m" (*p) : "ir" (v) : "cc"); } static __inline void atomic_add_barr_int(volatile u_int *p, u_int v){ __asm volatile("lock ; " "addl %1,%0" : "+m" (*p) : "ir" (v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_subtract_int(volatile u_int *p, u_int v){ __asm volatile("lock ; " "subl %1,%0" : "+m" (*p) : "ir" (v) : "cc"); } static __inline void atomic_subtract_barr_int(volatile u_int *p, u_int v){ __asm volatile("lock ; " "subl %1,%0" : "+m" (*p) : "ir" (v) : "memory", "cc"); } struct __hack;
///	
///	static __inline void atomic_set_long(volatile u_long *p, u_long v){ __asm volatile("lock ; " "orq %1,%0" : "+m" (*p) : "er" (v) : "cc"); } static __inline void atomic_set_barr_long(volatile u_long *p, u_long v){ __asm volatile("lock ; " "orq %1,%0" : "+m" (*p) : "er" (v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_clear_long(volatile u_long *p, u_long v){ __asm volatile("lock ; " "andq %1,%0" : "+m" (*p) : "er" (~v) : "cc"); } static __inline void atomic_clear_barr_long(volatile u_long *p, u_long v){ __asm volatile("lock ; " "andq %1,%0" : "+m" (*p) : "er" (~v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_add_long(volatile u_long *p, u_long v){ __asm volatile("lock ; " "addq %1,%0" : "+m" (*p) : "er" (v) : "cc"); } static __inline void atomic_add_barr_long(volatile u_long *p, u_long v){ __asm volatile("lock ; " "addq %1,%0" : "+m" (*p) : "er" (v) : "memory", "cc"); } struct __hack;
///	static __inline void atomic_subtract_long(volatile u_long *p, u_long v){ __asm volatile("lock ; " "subq %1,%0" : "+m" (*p) : "er" (v) : "cc"); } static __inline void atomic_subtract_barr_long(volatile u_long *p, u_long v){ __asm volatile("lock ; " "subq %1,%0" : "+m" (*p) : "er" (v) : "memory", "cc"); } struct __hack;
///	
///	
///	
///	
///	
///	static __inline u_char atomic_load_acq_char(volatile u_char *p) { u_char res; res = *p; __asm volatile(" " : : : "memory"); return (res); } struct __hack; static __inline void atomic_store_rel_char(volatile u_char *p, u_char v) { __asm volatile(" " : : : "memory"); *p = v; } struct __hack;
///	static __inline u_short atomic_load_acq_short(volatile u_short *p) { u_short res; res = *p; __asm volatile(" " : : : "memory"); return (res); } struct __hack; static __inline void atomic_store_rel_short(volatile u_short *p, u_short v) { __asm volatile(" " : : : "memory"); *p = v; } struct __hack;
///	static __inline u_int atomic_load_acq_int(volatile u_int *p) { u_int res; res = *p; __asm volatile(" " : : : "memory"); return (res); } struct __hack; static __inline void atomic_store_rel_int(volatile u_int *p, u_int v) { __asm volatile(" " : : : "memory"); *p = v; } struct __hack;
///	static __inline u_long atomic_load_acq_long(volatile u_long *p) { u_long res; res = *p; __asm volatile(" " : : : "memory"); return (res); } struct __hack; static __inline void atomic_store_rel_long(volatile u_long *p, u_long v) { __asm volatile(" " : : : "memory"); *p = v; } struct __hack;
///	
///	static __inline u_int
///	atomic_swap_int(volatile u_int *p, u_int v)
///	{
///	
///	 __asm volatile(
///	 "	xchgl	%1,%0 ;		"
///	 "# atomic_swap_int"
///	 : "+r" (v),
///	   "+m" (*p));
///	 return (v);
///	}
///	
///	static __inline u_long
///	atomic_swap_long(volatile u_long *p, u_long v)
///	{
///	
///	 __asm volatile(
///	 "	xchgq	%1,%0 ;		"
///	 "# atomic_swap_long"
///	 : "+r" (v),
///	   "+m" (*p));
///	 return (v);
///	}
///	
///	
///	
///	
///	
///	extern char *_PathLocale;
///	
///	int __detect_path_locale(void);
///	int __wrap_setrunelocale(const char *);
///	
///	
///	enum {
///	 XLC_COLLATE = 0,
///	 XLC_CTYPE,
///	 XLC_MONETARY,
///	 XLC_NUMERIC,
///	 XLC_TIME,
///	 XLC_MESSAGES,
///	 XLC_LAST
///	};
///	
///	_Static_assert(XLC_LAST - XLC_COLLATE == 6, "XLC values should be contiguous");
///	_Static_assert(XLC_COLLATE == 
///	
///	                             1 
///	
///	                                        - 1,
///	               "XLC_COLLATE doesn't match the LC_COLLATE value.");
///	_Static_assert(XLC_CTYPE == 
///	
///	                           2 
///	
///	                                    - 1,
///	               "XLC_CTYPE doesn't match the LC_CTYPE value.");
///	_Static_assert(XLC_MONETARY == 
///	
///	                              3 
///	
///	                                          - 1,
///	               "XLC_MONETARY doesn't match the LC_MONETARY value.");
///	_Static_assert(XLC_NUMERIC == 
///	
///	                             4 
///	
///	                                        - 1,
///	               "XLC_NUMERIC doesn't match the LC_NUMERIC value.");
///	_Static_assert(XLC_TIME == 
///	
///	                          5 
///	
///	                                  - 1,
///	               "XLC_TIME doesn't match the LC_TIME value.");
///	_Static_assert(XLC_MESSAGES == 
///	
///	                              6 
///	
///	                                          - 1,
///	               "XLC_MESSAGES doesn't match the LC_MESSAGES value.");
///	
///	struct xlocale_refcounted {
///	
///	 long retain_count;
///	
///	 void(*destructor)(void*);
///	};
///	
///	
///	
///	
///	
///	
///	
///	struct xlocale_component {
///	 struct xlocale_refcounted header;
///	
///	 char locale[31 +1];
///	
///	 char version[12];
///	};
///	
///	
///	
///	
///	struct _xlocale {
///	 struct xlocale_refcounted header;
///	
///	 struct xlocale_component *components[XLC_LAST];
///	
///	
///	 int monetary_locale_changed;
///	
///	
///	 int using_monetary_locale;
///	
///	
///	 int numeric_locale_changed;
///	
///	
///	 int using_numeric_locale;
///	
///	
///	 int using_time_locale;
///	
///	
///	 int using_messages_locale;
///	
///	 struct lconv lconv;
///	
///	 char *csym;
///	};
///	
///	
///	
///	
///	__attribute__((unused)) static void*
///	xlocale_retain(void *val)
///	{
///	 struct xlocale_refcounted *obj = val;
///	 atomic_add_long(&(obj->retain_count), 1);
///	 return (val);
///	}
///	
///	
///	
///	
///	__attribute__((unused)) static void
///	xlocale_release(void *val)
///	{
///	 struct xlocale_refcounted *obj = val;
///	 long count;
///	
///	 count = atomic_fetchadd_long(&(obj->retain_count), -1) - 1;
///	 if (count < 0 && obj->destructor != 
///	
///	                                    ((void *)0)
///	
///	                                        )
///	  obj->destructor(obj);
///	}
///	
///	
///	
///	
///	
///	extern void* __collate_load(const char*, locale_t);
///	extern void* __ctype_load(const char*, locale_t);
///	extern void* __messages_load(const char*, locale_t);
///	extern void* __monetary_load(const char*, locale_t);
///	extern void* __numeric_load(const char*, locale_t);
///	extern void* __time_load(const char*, locale_t);
///	
///	extern struct _xlocale __xlocale_global_locale;
///	extern struct _xlocale __xlocale_C_locale;
///	
///	
///	
///	
///	void __set_thread_rune_locale(locale_t loc);
///	
///	
///	
///	
///	extern int __has_thread_locale;
///	
///	
///	
///	
///	
///	extern _Thread_local locale_t __thread_locale;
///	
///	
///	
///	
///	
///	
///	
///	static inline locale_t __get_locale(void)
///	{
///	
///	 if (!__has_thread_locale) {
///	  return (&__xlocale_global_locale);
///	 }
///	 return (__thread_locale ? __thread_locale : &__xlocale_global_locale);
///	}
///	
///	
///	
///	
///	
///	static inline locale_t get_real_locale(locale_t locale)
///	{
///	 switch ((intptr_t)locale) {
///	  case 0: return (&__xlocale_C_locale);
///	  case -1: return (&__xlocale_global_locale);
///	  default: return (locale);
///	 }
///	}
///	
///	
///	
///	
///	
///	
///	
///	
///	struct xlocale_ctype {
///	 struct xlocale_component header;
///	 _RuneLocale *runes;
///	 size_t (*__mbrtowc)(wchar_t * 
///	
///	                              restrict
///	
///	                                        , const char * 
///	
///	                                                       restrict
///	
///	                                                                 ,
///	  size_t, mbstate_t * 
///	
///	                     restrict
///	
///	                               );
///	 int (*__mbsinit)(const mbstate_t *);
///	 size_t (*__mbsnrtowcs)(wchar_t * 
///	
///	                                 restrict
///	
///	                                           , const char ** 
///	
///	                                                           restrict
///	
///	                                                                     ,
///	  size_t, size_t, mbstate_t * 
///	
///	                             restrict
///	
///	                                       );
///	 size_t (*__wcrtomb)(char * 
///	
///	                           restrict
///	
///	                                     , wchar_t, mbstate_t * 
///	
///	                                                            restrict
///	
///	                                                                      );
///	 size_t (*__wcsnrtombs)(char * 
///	
///	                              restrict
///	
///	                                        , const wchar_t ** 
///	
///	                                                           restrict
///	
///	                                                                     ,
///	  size_t, size_t, mbstate_t * 
///	
///	                             restrict
///	
///	                                       );
///	 int __mb_cur_max;
///	 int __mb_sb_limit;
///	
///	 __mbstate_t mblen;
///	
///	 __mbstate_t mbrlen;
///	
///	 __mbstate_t mbrtoc16;
///	
///	 __mbstate_t mbrtoc32;
///	
///	 __mbstate_t mbrtowc;
///	
///	 __mbstate_t mbsnrtowcs;
///	
///	 __mbstate_t mbsrtowcs;
///	
///	 __mbstate_t mbtowc;
///	
///	 __mbstate_t c16rtomb;
///	
///	 __mbstate_t c32rtomb;
///	
///	 __mbstate_t wcrtomb;
///	
///	 __mbstate_t wcsnrtombs;
///	
///	 __mbstate_t wcsrtombs;
///	
///	 __mbstate_t wctomb;
///	};
///	
///	extern struct xlocale_ctype __xlocale_global_ctype;
///	
///	
///	
///	
///	int _none_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                     __attribute__((__visibility__("hidden")))
///	
///	                                                             ;
///	int _UTF8_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                     __attribute__((__visibility__("hidden")))
///	
///	                                                             ;
///	int _EUC_CN_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                       __attribute__((__visibility__("hidden")))
///	
///	                                                               ;
///	int _EUC_JP_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                       __attribute__((__visibility__("hidden")))
///	
///	                                                               ;
///	int _EUC_KR_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                       __attribute__((__visibility__("hidden")))
///	
///	                                                               ;
///	int _EUC_TW_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                       __attribute__((__visibility__("hidden")))
///	
///	                                                               ;
///	int _GB18030_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                        __attribute__((__visibility__("hidden")))
///	
///	                                                                ;
///	int _GB2312_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                       __attribute__((__visibility__("hidden")))
///	
///	                                                               ;
///	int _GBK_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                    __attribute__((__visibility__("hidden")))
///	
///	                                                            ;
///	int _BIG5_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                     __attribute__((__visibility__("hidden")))
///	
///	                                                             ;
///	int _MSKanji_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                        __attribute__((__visibility__("hidden")))
///	
///	                                                                ;
///	int _ascii_init(struct xlocale_ctype *, _RuneLocale *) 
///	
///	                                                      __attribute__((__visibility__("hidden")))
///	
///	                                                              ;
///	
///	typedef size_t (*mbrtowc_pfn_t)(wchar_t * 
///	
///	                                         restrict
///	
///	                                                   ,
///	    const char * 
///	
///	                restrict
///	
///	                          , size_t, mbstate_t * 
///	
///	                                                restrict
///	
///	                                                          );
///	typedef size_t (*wcrtomb_pfn_t)(char * 
///	
///	                                      restrict
///	
///	                                                , wchar_t,
///	    mbstate_t * 
///	
///	               restrict
///	
///	                         );
///	size_t __mbsnrtowcs_std(wchar_t * 
///	
///	                                 restrict
///	
///	                                           , const char ** 
///	
///	                                                           restrict
///	
///	                                                                     ,
///	    size_t, size_t, mbstate_t * 
///	
///	                               restrict
///	
///	                                         , mbrtowc_pfn_t);
///	size_t __wcsnrtombs_std(char * 
///	
///	                              restrict
///	
///	                                        , const wchar_t ** 
///	
///	                                                           restrict
///	
///	                                                                     ,
///	    size_t, size_t, mbstate_t * 
///	
///	                               restrict
///	
///	                                         , wcrtomb_pfn_t);
///	

const _RuneLocale _DefaultRuneLocale = {
    

   "RuneMagi"

                ,
    "NONE",
    

   ((void *)0)

       ,
    

   ((void *)0)

       ,
    0xFFFD,

    { 

            0x00000200L

                    ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
        

       0x00000200L

               ,
  

 0x00000200L

         |

          0x00004000L

                  |

                   0x00020000L

                           ,
  

 0x00000200L

         |

          0x00004000L

                  ,
  

 0x00000200L

         |

          0x00004000L

                  ,
  

 0x00000200L

         |

          0x00004000L

                  ,
  

 0x00000200L

         |

          0x00004000L

                  ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
        

       0x00000200L

               ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
        

       0x00000200L

               ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
  

 0x00000200L

         ,
        

       0x00004000L

               |

                0x00020000L

                        |

                         0x00040000L

                                 ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
        

       0x00002000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
        

       0x00000400L

               |

                0x00040000L

                        |

                         0x00000800L

                                 |

                                  0x00010000L

                                          |

                                           0x00400000L

                                                   |0,
  

 0x00000400L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00010000L

                                    |

                                     0x00400000L

                                             |1,
  

 0x00000400L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00010000L

                                    |

                                     0x00400000L

                                             |2,
  

 0x00000400L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00010000L

                                    |

                                     0x00400000L

                                             |3,
  

 0x00000400L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00010000L

                                    |

                                     0x00400000L

                                             |4,
  

 0x00000400L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00010000L

                                    |

                                     0x00400000L

                                             |5,
  

 0x00000400L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00010000L

                                    |

                                     0x00400000L

                                             |6,
  

 0x00000400L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00010000L

                                    |

                                     0x00400000L

                                             |7,
        

       0x00000400L

               |

                0x00040000L

                        |

                         0x00000800L

                                 |

                                  0x00010000L

                                          |

                                           0x00400000L

                                                   |8,
  

 0x00000400L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00010000L

                                    |

                                     0x00400000L

                                             |9,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
        

       0x00002000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 ,
  

 0x00008000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |10,
  

 0x00008000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |11,
  

 0x00008000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |12,
  

 0x00008000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |13,
  

 0x00008000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |14,
  

 0x00008000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |15,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
        

       0x00008000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 |

                                  0x00000100L

                                          ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
        

       0x00008000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 |

                                  0x00000100L

                                          ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
        

       0x00008000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 |

                                  0x00000100L

                                          ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00008000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
        

       0x00002000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 ,
  

 0x00001000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |10,
  

 0x00001000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |11,
  

 0x00001000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |12,
  

 0x00001000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |13,
  

 0x00001000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |14,
  

 0x00001000L

         |

          0x00010000L

                  |

                   0x00040000L

                           |

                            0x00000800L

                                    |

                                     0x00000100L

                                             |15,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
        

       0x00001000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 |

                                  0x00000100L

                                          ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
        

       0x00001000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 |

                                  0x00000100L

                                          ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
        

       0x00001000L

               |

                0x00040000L

                        |

                         0x00000800L

                                 |

                                  0x00000100L

                                          ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00001000L

         |

          0x00040000L

                  |

                   0x00000800L

                           |

                            0x00000100L

                                    ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00002000L

         |

          0x00040000L

                  |

                   0x00000800L

                           ,
  

 0x00000200L

         ,
    },
    { 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
      0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
      0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
      0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
      0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f,
 0x40, 'a', 'b', 'c', 'd', 'e', 'f', 'g',
      'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o',
 'p', 'q', 'r', 's', 't', 'u', 'v', 'w',
      'x', 'y', 'z', 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
 0x60, 'a', 'b', 'c', 'd', 'e', 'f', 'g',
      'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o',
 'p', 'q', 'r', 's', 't', 'u', 'v', 'w',
      'x', 'y', 'z', 0x7b, 0x7c, 0x7d, 0x7e, 0x7f,
 0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87,
      0x88, 0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f,
 0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97,
      0x98, 0x99, 0x9a, 0x9b, 0x9c, 0x9d, 0x9e, 0x9f,
 0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7,
      0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf,
 0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7,
      0xb8, 0xb9, 0xba, 0xbb, 0xbc, 0xbd, 0xbe, 0xbf,
 0xc0, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7,
      0xc8, 0xc9, 0xca, 0xcb, 0xcc, 0xcd, 0xce, 0xcf,
 0xd0, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7,
      0xd8, 0xd9, 0xda, 0xdb, 0xdc, 0xdd, 0xde, 0xdf,
 0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
      0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef,
 0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7,
      0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff,
    },
    { 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
      0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
      0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,
 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x26, 0x27,
      0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37,
      0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d, 0x3e, 0x3f,
 0x40, 'A', 'B', 'C', 'D', 'E', 'F', 'G',
      'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O',
 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W',
      'X', 'Y', 'Z', 0x5b, 0x5c, 0x5d, 0x5e, 0x5f,
 0x60, 'A', 'B', 'C', 'D', 'E', 'F', 'G',
      'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O',
 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'W',
      'X', 'Y', 'Z', 0x7b, 0x7c, 0x7d, 0x7e, 0x7f,
 0x80, 0x81, 0x82, 0x83, 0x84, 0x85, 0x86, 0x87,
      0x88, 0x89, 0x8a, 0x8b, 0x8c, 0x8d, 0x8e, 0x8f,
 0x90, 0x91, 0x92, 0x93, 0x94, 0x95, 0x96, 0x97,
      0x98, 0x99, 0x9a, 0x9b, 0x9c, 0x9d, 0x9e, 0x9f,
 0xa0, 0xa1, 0xa2, 0xa3, 0xa4, 0xa5, 0xa6, 0xa7,
      0xa8, 0xa9, 0xaa, 0xab, 0xac, 0xad, 0xae, 0xaf,
 0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7,
      0xb8, 0xb9, 0xba, 0xbb, 0xbc, 0xbd, 0xbe, 0xbf,
 0xc0, 0xc1, 0xc2, 0xc3, 0xc4, 0xc5, 0xc6, 0xc7,
      0xc8, 0xc9, 0xca, 0xcb, 0xcc, 0xcd, 0xce, 0xcf,
 0xd0, 0xd1, 0xd2, 0xd3, 0xd4, 0xd5, 0xd6, 0xd7,
      0xd8, 0xd9, 0xda, 0xdb, 0xdc, 0xdd, 0xde, 0xdf,
 0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
      0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef,
 0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7,
      0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff,
    },
};


const _RuneLocale *_CurrentRuneLocale = &_DefaultRuneLocale;

///	_RuneLocale *
///	__runes_for_locale(locale_t locale, int *mb_sb_limit)
///	{
///	 (locale = get_real_locale(locale));
///	 struct xlocale_ctype *c = ((struct xlocale_ctype*)(locale)->components[XLC_CTYPE]);
///	 *mb_sb_limit = c->__mb_sb_limit;
///	 return c->runes;
///	}
