#pragma once
#include "compiler.h"
#include "endian.h"
#include "fbcash.h"
#include <stdio.h>

#define ENABLE_SSE 0

#if defined(_M_X64) && ENABLE_SSE
#include <smmintrin.h>
#endif

#ifdef __cplusplus
extern "C" {
#endif

// compile time settings
#define NODE_WORDS (64/4)
#define MIX_WORDS (FBCASH_MIX_BYTES/4)
#define MIX_NODES (MIX_WORDS / NODE_WORDS)
#include <stdint.h>

typedef union node {
	uint8_t bytes[NODE_WORDS * 4];
	uint32_t words[NODE_WORDS];
	uint64_t double_words[NODE_WORDS / 2];

#if defined(_M_X64) && ENABLE_SSE
	__m128i xmm[NODE_WORDS/4];
#endif

} node;

static inline uint8_t fbcash_h256_get(fbcash_h256_t const* hash, unsigned int i)
{
	return hash->b[i];
}

static inline void fbcash_h256_set(fbcash_h256_t* hash, unsigned int i, uint8_t v)
{
	hash->b[i] = v;
}

static inline void fbcash_h256_reset(fbcash_h256_t* hash)
{
	memset(hash, 0, 32);
}

// Returns if hash is less than or equal to boundary (2^256/difficulty)
static inline bool fbcash_check_difficulty(
	fbcash_h256_t const* hash,
	fbcash_h256_t const* boundary
)
{
	// Boundary is big endian
	for (int i = 0; i < 32; i++) {
		if (fbcash_h256_get(hash, i) == fbcash_h256_get(boundary, i)) {
			continue;
		}
		return fbcash_h256_get(hash, i) < fbcash_h256_get(boundary, i);
	}
	return true;
}

/**
 *  Difficulty quick check for POW preverification
 *
 * @param header_hash      The hash of the header
 * @param nonce            The block's nonce
 * @param mix_hash         The mix digest hash
 * @param boundary         The boundary is defined as (2^256 / difficulty)
 * @return                 true for succesful pre-verification and false otherwise
 */
bool fbcash_quick_check_difficulty(
	fbcash_h256_t const* header_hash,
	uint64_t const nonce,
	fbcash_h256_t const* mix_hash,
	fbcash_h256_t const* boundary
);

struct fbcash_light {
	void* cache;
	uint64_t cache_size;
	uint64_t block_number;
};

/**
 * Allocate and initialize a new fbcash_light handler. Internal version
 *
 * @param cache_size    The size of the cache in bytes
 * @param seed          Block seedhash to be used during the computation of the
 *                      cache nodes
 * @return              Newly allocated fbcash_light handler or NULL in case of
 *                      ERRNOMEM or invalid parameters used for @ref fbcash_compute_cache_nodes()
 */
fbcash_light_t fbcash_light_new_internal(uint64_t cache_size, fbcash_h256_t const* seed);

/**
 * Calculate the light client data. Internal version.
 *
 * @param light          The light client handler
 * @param full_size      The size of the full data in bytes.
 * @param header_hash    The header hash to pack into the mix
 * @param nonce          The nonce to pack into the mix
 * @return               The resulting hash.
 */
fbcash_return_value_t fbcash_light_compute_internal(
	fbcash_light_t light,
	uint64_t full_size,
	fbcash_h256_t const header_hash,
	uint64_t nonce
);

struct fbcash_full {
	FILE* file;
	uint64_t file_size;
	node* data;
};

/**
 * Allocate and initialize a new fbcash_full handler. Internal version.
 *
 * @param dirname        The directory in which to put the DAG file.
 * @param seedhash       The seed hash of the block. Used in the DAG file naming.
 * @param full_size      The size of the full data in bytes.
 * @param cache          A cache object to use that was allocated with @ref fbcash_cache_new().
 *                       Iff this function succeeds the fbcash_full_t will take memory
 *                       memory ownership of the cache and free it at deletion. If
 *                       not then the user still has to handle freeing of the cache himself.
 * @param callback       A callback function with signature of @ref fbcash_callback_t
 *                       It accepts an unsigned with which a progress of DAG calculation
 *                       can be displayed. If all goes well the callback should return 0.
 *                       If a non-zero value is returned then DAG generation will stop.
 * @return               Newly allocated fbcash_full handler or NULL in case of
 *                       ERRNOMEM or invalid parameters used for @ref fbcash_compute_full_data()
 */
fbcash_full_t fbcash_full_new_internal(
	char const* dirname,
	fbcash_h256_t const seed_hash,
	uint64_t full_size,
	fbcash_light_t const light,
	fbcash_callback_t callback
);

void fbcash_calculate_dag_item(
	node* const ret,
	uint32_t node_index,
	fbcash_light_t const cache
);

void fbcash_quick_hash(
	fbcash_h256_t* return_hash,
	fbcash_h256_t const* header_hash,
	const uint64_t nonce,
	fbcash_h256_t const* mix_hash
);

uint64_t fbcash_get_datasize(uint64_t const block_number);
uint64_t fbcash_get_cachesize(uint64_t const block_number);

/**
 * Compute the memory data for a full node's memory
 *
 * @param mem         A pointer to an fbcash full's memory
 * @param full_size   The size of the full data in bytes
 * @param cache       A cache object to use in the calculation
 * @param callback    The callback function. Check @ref fbcash_full_new() for details.
 * @return            true if all went fine and false for invalid parameters
 */
bool fbcash_compute_full_data(
	void* mem,
	uint64_t full_size,
	fbcash_light_t const light,
	fbcash_callback_t callback
);

#ifdef __cplusplus
}
#endif
