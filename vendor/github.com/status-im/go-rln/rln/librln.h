#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

typedef struct RLN_Bn256 RLN_Bn256;

/**
 * Buffer struct is taken from
 * https://github.com/celo-org/celo-threshold-bls-rs/blob/master/crates/threshold-bls-ffi/src/ffi.rs
 */
typedef struct Buffer {
  const uint8_t *ptr;
  uintptr_t len;
} Buffer;

bool new_circuit_from_params(uintptr_t merkle_depth,
                             const struct Buffer *parameters_buffer,
                             struct RLN_Bn256 **ctx);

bool get_root(const struct RLN_Bn256 *ctx, struct Buffer *output_buffer);

bool update_next_member(struct RLN_Bn256 *ctx, const struct Buffer *input_buffer);

bool delete_member(struct RLN_Bn256 *ctx, uintptr_t index);

bool generate_proof(const struct RLN_Bn256 *ctx,
                    const struct Buffer *input_buffer,
                    struct Buffer *output_buffer);

bool verify(const struct RLN_Bn256 *ctx, const struct Buffer *proof_buffer, uint32_t *result_ptr);

bool signal_to_field(const struct RLN_Bn256 *ctx,
                     const struct Buffer *inputs_buffer,
                     struct Buffer *output_buffer);

bool key_gen(const struct RLN_Bn256 *ctx, struct Buffer *input_buffer);
