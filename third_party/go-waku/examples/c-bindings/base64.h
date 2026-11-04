
#ifndef _BASE64_H_
#define _BASE64_H_

#include <stdlib.h>

size_t b64_encoded_size(size_t inlen);

char *b64_encode(const unsigned char *in, size_t len);

size_t b64_decoded_size(const char *in);

int b64_decode(const char *in, unsigned char *out, size_t outlen);

#endif
