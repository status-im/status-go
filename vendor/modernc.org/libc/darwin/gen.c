#include <stdio.h>
#include <ctype.h>

// typedef struct {
// 	char		__magic[8];	/* Magic saying what version we are */
// 	char		__encoding[32];	/* ASCII name of this encoding */
// 
// 	__darwin_rune_t	(*__sgetrune)(const char *, __darwin_size_t, char const **);
// 	int		(*__sputrune)(__darwin_rune_t, char *, __darwin_size_t, char **);
// 	__darwin_rune_t	__invalid_rune;
// 
// 	__uint32_t	__runetype[_CACHED_RUNES];
// 	__darwin_rune_t	__maplower[_CACHED_RUNES];
// 	__darwin_rune_t	__mapupper[_CACHED_RUNES];
// 
// 	/*
// 	 * The following are to deal with Runes larger than _CACHED_RUNES - 1.
// 	 * Their data is actually contiguous with this structure so as to make
// 	 * it easier to read/write from/to disk.
// 	 */
// 	_RuneRange	__runetype_ext;
// 	_RuneRange	__maplower_ext;
// 	_RuneRange	__mapupper_ext;
// 
// 	void		*__variable;	/* Data which depends on the encoding */
// 	int		__variable_len;	/* how long that data is */
// 
// 	/*
// 	 * extra fields to deal with arbitrary character classes
// 	 */
// 	int		__ncharclasses;
// 	_RuneCharClass	*__charclasses;
// } _RuneLocale;


#define SZ(a) (sizeof(a)/sizeof(a[0]))

int main() {
	printf("#include <ctype.h>\n\n");

	printf(
"	__maskrune(__darwin_ct_rune_t _c, unsigned long _f)\n"
"{\n"
"	return (int)_DefaultRuneLocale.__runetype[_c & 0xff] & (__uint32_t)_f;\n"
"}\n"
);
	printf("\n__darwin_ct_rune_t __toupper(__darwin_ct_rune_t c) { return toupper(c); }\n");
	printf("\n__darwin_ct_rune_t __tolower(__darwin_ct_rune_t c) { return tolower(c); }\n");

	printf("_RuneLocale _DefaultRuneLocale = {\n");

	printf("\t.__magic = {");
	for (int i = 0; i < SZ(_DefaultRuneLocale.__magic); i++) {
		printf("%i, ", _DefaultRuneLocale.__magic[i]);
	}
	printf("},\n");

	printf("\t.__encoding = {");
	for (int i = 0; i < SZ(_DefaultRuneLocale.__encoding); i++) {
		printf("%i, ", _DefaultRuneLocale.__encoding[i]);
	}
	printf("},\n");

	printf("\t.__invalid_rune = 0x%x,\n", (unsigned)_DefaultRuneLocale.__invalid_rune);

	printf("\t.__runetype = {");
	for (int i = 0; i < SZ(_DefaultRuneLocale.__runetype); i++) {
		if (i%16 == 0) {
			printf("\n\t\t");
		}
		printf("0x%x, ", _DefaultRuneLocale.__runetype[i]);
	}
	printf("\n\t},\n");

	printf("\t.__maplower = {");
	for (int i = 0; i < SZ(_DefaultRuneLocale.__maplower); i++) {
		if (i%16 == 0) {
			printf("\n\t\t");
		}
		printf("0x%x, ", _DefaultRuneLocale.__maplower[i]);
	}
	printf("\n\t},\n");

	printf("\t.__mapupper= {");
	for (int i = 0; i < SZ(_DefaultRuneLocale.__mapupper); i++) {
		if (i%16 == 0) {
			printf("\n\t\t");
		}
		printf("0x%x, ", _DefaultRuneLocale.__mapupper[i]);
	}
	printf("\n\t},\n");

	printf("\n};\n");
	printf("\n_RuneLocale *_CurrentRuneLocale = &_DefaultRuneLocale;\n");
}
