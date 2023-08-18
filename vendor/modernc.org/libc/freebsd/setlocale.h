
#ifndef _SETLOCALE_H_
#define	_SETLOCALE_H_

#define ENCODING_LEN 31
#define CATEGORY_LEN 11

extern char *_PathLocale;

int	__detect_path_locale(void);
int	__wrap_setrunelocale(const char *);

#endif /* !_SETLOCALE_H_ */
