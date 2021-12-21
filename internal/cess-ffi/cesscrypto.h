/* cesscrypto Header */

#ifdef __cplusplus
extern "C" {
#endif


#ifndef cesscrypto_H
#define cesscrypto_H

/* Generated with cbindgen:0.14.0 */

#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

#define DIGEST_BYTES 96

#define PRIVATE_KEY_BYTES 32

#define PUBLIC_KEY_BYTES 48

#define SIGNATURE_BYTES 96

typedef enum {
  FCPResponseStatus_FCPNoError = 0,
  FCPResponseStatus_FCPUnclassifiedError = 1,
  FCPResponseStatus_FCPCallerError = 2,
  FCPResponseStatus_FCPReceiverError = 3,
} FCPResponseStatus;

typedef enum {
  cess_RegisteredAggregationProof_SnarkPackV1,
} cess_RegisteredAggregationProof;

typedef enum {
  cess_RegisteredPoStProof_StackedDrgWinning2KiBV1,
  cess_RegisteredPoStProof_StackedDrgWinning8MiBV1,
  cess_RegisteredPoStProof_StackedDrgWinning32MiBV1,
  cess_RegisteredPoStProof_StackedDrgWinning512MiBV1,
  cess_RegisteredPoStProof_StackedDrgWinning32GiBV1,
  cess_RegisteredPoStProof_StackedDrgWinning64GiBV1,
  cess_RegisteredPoStProof_StackedDrgWindow2KiBV1,
  cess_RegisteredPoStProof_StackedDrgWindow8MiBV1,
  cess_RegisteredPoStProof_StackedDrgWindow32MiBV1,
  cess_RegisteredPoStProof_StackedDrgWindow512MiBV1,
  cess_RegisteredPoStProof_StackedDrgWindow32GiBV1,
  cess_RegisteredPoStProof_StackedDrgWindow64GiBV1,
} cess_RegisteredPoStProof;

typedef enum {
  cess_RegisteredSealProof_StackedDrg2KiBV1,
  cess_RegisteredSealProof_StackedDrg8MiBV1,
  cess_RegisteredSealProof_StackedDrg32MiBV1,
  cess_RegisteredSealProof_StackedDrg512MiBV1,
  cess_RegisteredSealProof_StackedDrg32GiBV1,
  cess_RegisteredSealProof_StackedDrg64GiBV1,
  cess_RegisteredSealProof_StackedDrg2KiBV1_1,
  cess_RegisteredSealProof_StackedDrg8MiBV1_1,
  cess_RegisteredSealProof_StackedDrg32MiBV1_1,
  cess_RegisteredSealProof_StackedDrg512MiBV1_1,
  cess_RegisteredSealProof_StackedDrg32GiBV1_1,
  cess_RegisteredSealProof_StackedDrg64GiBV1_1,
} cess_RegisteredSealProof;

typedef struct {
  uint8_t inner[SIGNATURE_BYTES];
} cess_BLSSignature;

/**
 * AggregateResponse
 */
typedef struct {
  cess_BLSSignature signature;
} cess_AggregateResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  size_t proof_len;
  const uint8_t *proof_ptr;
} cess_AggregateProof;

typedef struct {
  uint8_t inner[32];
} cess_32ByteArray;

typedef struct {
  cess_32ByteArray comm_r;
  cess_32ByteArray comm_d;
  uint64_t sector_id;
  cess_32ByteArray ticket;
  cess_32ByteArray seed;
} cess_AggregationInputs;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  const uint8_t *proof_ptr;
  size_t proof_len;
  const cess_AggregationInputs *commit_inputs_ptr;
  size_t commit_inputs_len;
} cess_SealCommitPhase2Response;

typedef struct {
  const char *error_msg;
  FCPResponseStatus status_code;
} cess_ClearCacheResponse;

/**
 * AggregateResponse
 */
typedef struct {
  cess_BLSSignature signature;
} cess_ZeroSignatureResponse;

typedef struct {
  const char *error_msg;
  FCPResponseStatus status_code;
  uint8_t commitment[32];
} cess_FauxRepResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  uint8_t ticket[32];
} cess_FinalizeTicketResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  uint8_t comm_d[32];
} cess_GenerateDataCommitmentResponse;

typedef struct {
  const char *error_msg;
  FCPResponseStatus status_code;
  const uint64_t *ids_ptr;
  size_t ids_len;
  const uint64_t *challenges_ptr;
  size_t challenges_len;
  size_t challenges_stride;
} cess_GenerateFallbackSectorChallengesResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  uint8_t comm_p[32];
  /**
   * The number of unpadded bytes in the original piece plus any (unpadded)
   * alignment bytes added to create a whole merkle tree.
   */
  uint64_t num_bytes_aligned;
} cess_GeneratePieceCommitmentResponse;

typedef struct {
  size_t proof_len;
  const uint8_t *proof_ptr;
} cess_VanillaProof;

typedef struct {
  const char *error_msg;
  cess_VanillaProof vanilla_proof;
  FCPResponseStatus status_code;
} cess_GenerateSingleVanillaProofResponse;

typedef struct {
  cess_RegisteredPoStProof registered_proof;
  size_t proof_len;
  const uint8_t *proof_ptr;
} cess_PartitionSnarkProof;

typedef struct {
  const char *error_msg;
  cess_PartitionSnarkProof partition_proof;
  size_t faulty_sectors_len;
  const uint64_t *faulty_sectors_ptr;
  FCPResponseStatus status_code;
} cess_GenerateSingleWindowPoStWithVanillaResponse;

typedef struct {
  cess_RegisteredPoStProof registered_proof;
  size_t proof_len;
  const uint8_t *proof_ptr;
} cess_PoStProof;

typedef struct {
  const char *error_msg;
  size_t proofs_len;
  const cess_PoStProof *proofs_ptr;
  size_t faulty_sectors_len;
  const uint64_t *faulty_sectors_ptr;
  FCPResponseStatus status_code;
} cess_GenerateWindowPoStResponse;

typedef struct {
  const char *error_msg;
  size_t proofs_len;
  const cess_PoStProof *proofs_ptr;
  FCPResponseStatus status_code;
} cess_GenerateWinningPoStResponse;

typedef struct {
  const char *error_msg;
  FCPResponseStatus status_code;
  const uint64_t *ids_ptr;
  size_t ids_len;
} cess_GenerateWinningPoStSectorChallenge;

typedef struct {
  const char *error_msg;
  FCPResponseStatus status_code;
  size_t num_partition;
} cess_GetNumPartitionForFallbackPoStResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  size_t devices_len;
  const char *const *devices_ptr;
} cess_GpuDeviceResponse;

typedef struct {
  uint8_t inner[DIGEST_BYTES];
} cess_BLSDigest;

/**
 * HashResponse
 */
typedef struct {
  cess_BLSDigest digest;
} cess_HashResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
} cess_InitLogFdResponse;

typedef struct {
  const char *error_msg;
  cess_PoStProof proof;
  FCPResponseStatus status_code;
} cess_MergeWindowPoStPartitionProofsResponse;

typedef struct {
  uint8_t inner[PRIVATE_KEY_BYTES];
} cess_BLSPrivateKey;

/**
 * PrivateKeyGenerateResponse
 */
typedef struct {
  cess_BLSPrivateKey private_key;
} cess_PrivateKeyGenerateResponse;

typedef struct {
  uint8_t inner[PUBLIC_KEY_BYTES];
} cess_BLSPublicKey;

/**
 * PrivateKeyPublicKeyResponse
 */
typedef struct {
  cess_BLSPublicKey public_key;
} cess_PrivateKeyPublicKeyResponse;

/**
 * PrivateKeySignResponse
 */
typedef struct {
  cess_BLSSignature signature;
} cess_PrivateKeySignResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  const uint8_t *seal_commit_phase1_output_ptr;
  size_t seal_commit_phase1_output_len;
} cess_SealCommitPhase1Response;

typedef struct {
  const char *error_msg;
  FCPResponseStatus status_code;
  const uint8_t *seal_pre_commit_phase1_output_ptr;
  size_t seal_pre_commit_phase1_output_len;
} cess_SealPreCommitPhase1Response;

typedef struct {
  const char *error_msg;
  FCPResponseStatus status_code;
  cess_RegisteredSealProof registered_proof;
  uint8_t comm_d[32];
  uint8_t comm_r[32];
} cess_SealPreCommitPhase2Response;

/**
 *
 */
typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  const char *string_val;
} cess_StringResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
} cess_UnsealRangeResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  bool is_valid;
} cess_VerifyAggregateSealProofResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  bool is_valid;
} cess_VerifySealResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  bool is_valid;
} cess_VerifyWindowPoStResponse;

typedef struct {
  FCPResponseStatus status_code;
  const char *error_msg;
  bool is_valid;
} cess_VerifyWinningPoStResponse;

typedef struct {
  uint8_t comm_p[32];
  const char *error_msg;
  uint64_t left_alignment_unpadded;
  FCPResponseStatus status_code;
  uint64_t total_write_unpadded;
} cess_WriteWithAlignmentResponse;

typedef struct {
  uint8_t comm_p[32];
  const char *error_msg;
  FCPResponseStatus status_code;
  uint64_t total_write_unpadded;
} cess_WriteWithoutAlignmentResponse;

typedef struct {
  uint64_t num_bytes;
  uint8_t comm_p[32];
} cess_PublicPieceInfo;

typedef struct {
  cess_RegisteredPoStProof registered_proof;
  const char *cache_dir_path;
  uint8_t comm_r[32];
  const char *replica_path;
  uint64_t sector_id;
} cess_PrivateReplicaInfo;

typedef struct {
  cess_RegisteredPoStProof registered_proof;
  uint8_t comm_r[32];
  uint64_t sector_id;
} cess_PublicReplicaInfo;

/**
 * Aggregate signatures together into a new signature
 *
 * # Arguments
 *
 * * `flattened_signatures_ptr` - pointer to a byte array containing signatures
 * * `flattened_signatures_len` - length of the byte array (multiple of SIGNATURE_BYTES)
 *
 * Returns `NULL` on error. Result must be freed using `destroy_aggregate_response`.
 */
cess_AggregateResponse *cess_aggregate(const uint8_t *flattened_signatures_ptr,
                                       size_t flattened_signatures_len);

cess_AggregateProof *cess_aggregate_seal_proofs(cess_RegisteredSealProof registered_proof,
                                                cess_RegisteredAggregationProof registered_aggregation,
                                                const cess_32ByteArray *comm_rs_ptr,
                                                size_t comm_rs_len,
                                                const cess_32ByteArray *seeds_ptr,
                                                size_t seeds_len,
                                                const cess_SealCommitPhase2Response *seal_commit_responses_ptr,
                                                size_t seal_commit_responses_len);

cess_ClearCacheResponse *cess_clear_cache(uint64_t sector_size, const char *cache_dir_path);

/**
 * Returns a zero signature, used as placeholder in CESS.
 *
 * The return value is a pointer to a compressed signature in bytes, of length `SIGNATURE_BYTES`
 */
cess_ZeroSignatureResponse *cess_create_zero_signature(void);

/**
 * Deallocates a AggregateProof
 *
 */
void cess_destroy_aggregate_proof(cess_AggregateProof *ptr);

void cess_destroy_aggregate_response(cess_AggregateResponse *ptr);

void cess_destroy_clear_cache_response(cess_ClearCacheResponse *ptr);

void cess_destroy_fauxrep_response(cess_FauxRepResponse *ptr);

void cess_destroy_finalize_ticket_response(cess_FinalizeTicketResponse *ptr);

void cess_destroy_generate_data_commitment_response(cess_GenerateDataCommitmentResponse *ptr);

void cess_destroy_generate_fallback_sector_challenges_response(cess_GenerateFallbackSectorChallengesResponse *ptr);

void cess_destroy_generate_piece_commitment_response(cess_GeneratePieceCommitmentResponse *ptr);

void cess_destroy_generate_single_vanilla_proof_response(cess_GenerateSingleVanillaProofResponse *ptr);

void cess_destroy_generate_single_window_post_with_vanilla_response(cess_GenerateSingleWindowPoStWithVanillaResponse *ptr);

void cess_destroy_generate_window_post_response(cess_GenerateWindowPoStResponse *ptr);

void cess_destroy_generate_winning_post_response(cess_GenerateWinningPoStResponse *ptr);

void cess_destroy_generate_winning_post_sector_challenge(cess_GenerateWinningPoStSectorChallenge *ptr);

void cess_destroy_get_num_partition_for_fallback_post_response(cess_GetNumPartitionForFallbackPoStResponse *ptr);

void cess_destroy_gpu_device_response(cess_GpuDeviceResponse *ptr);

void cess_destroy_hash_response(cess_HashResponse *ptr);

void cess_destroy_init_log_fd_response(cess_InitLogFdResponse *ptr);

void cess_destroy_merge_window_post_partition_proofs_response(cess_MergeWindowPoStPartitionProofsResponse *ptr);

void cess_destroy_private_key_generate_response(cess_PrivateKeyGenerateResponse *ptr);

void cess_destroy_private_key_public_key_response(cess_PrivateKeyPublicKeyResponse *ptr);

void cess_destroy_private_key_sign_response(cess_PrivateKeySignResponse *ptr);

void cess_destroy_seal_commit_phase1_response(cess_SealCommitPhase1Response *ptr);

void cess_destroy_seal_commit_phase2_response(cess_SealCommitPhase2Response *ptr);

void cess_destroy_seal_pre_commit_phase1_response(cess_SealPreCommitPhase1Response *ptr);

void cess_destroy_seal_pre_commit_phase2_response(cess_SealPreCommitPhase2Response *ptr);

void cess_destroy_string_response(cess_StringResponse *ptr);

void cess_destroy_unseal_range_response(cess_UnsealRangeResponse *ptr);

/**
 * Deallocates a VerifyAggregateSealProofResponse.
 *
 */
void cess_destroy_verify_aggregate_seal_response(cess_VerifyAggregateSealProofResponse *ptr);

/**
 * Deallocates a VerifySealResponse.
 *
 */
void cess_destroy_verify_seal_response(cess_VerifySealResponse *ptr);

void cess_destroy_verify_window_post_response(cess_VerifyWindowPoStResponse *ptr);

/**
 * Deallocates a VerifyPoStResponse.
 *
 */
void cess_destroy_verify_winning_post_response(cess_VerifyWinningPoStResponse *ptr);

void cess_destroy_write_with_alignment_response(cess_WriteWithAlignmentResponse *ptr);

void cess_destroy_write_without_alignment_response(cess_WriteWithoutAlignmentResponse *ptr);

void cess_destroy_zero_signature_response(cess_ZeroSignatureResponse *ptr);

/**
 * Frees the memory of the returned value of `cess_create_zero_signature`.
 */
void cess_drop_signature(uint8_t *sig);

cess_FauxRepResponse *cess_fauxrep(cess_RegisteredSealProof registered_proof,
                                   const char *cache_dir_path,
                                   const char *sealed_sector_path);

cess_FauxRepResponse *cess_fauxrep2(cess_RegisteredSealProof registered_proof,
                                    const char *cache_dir_path,
                                    const char *existing_p_aux_path);

/**
 * Returns the merkle root for a sector containing the provided pieces.
 */
cess_GenerateDataCommitmentResponse *cess_generate_data_commitment(cess_RegisteredSealProof registered_proof,
                                                                   const cess_PublicPieceInfo *pieces_ptr,
                                                                   size_t pieces_len);

/**
 * TODO: document
 *
 */
cess_GenerateFallbackSectorChallengesResponse *cess_generate_fallback_sector_challenges(cess_RegisteredPoStProof registered_proof,
                                                                                        cess_32ByteArray randomness,
                                                                                        const uint64_t *sector_ids_ptr,
                                                                                        size_t sector_ids_len,
                                                                                        cess_32ByteArray prover_id);

/**
 * Returns the merkle root for a piece after piece padding and alignment.
 * The caller is responsible for closing the passed in file descriptor.
 */
cess_GeneratePieceCommitmentResponse *cess_generate_piece_commitment(cess_RegisteredSealProof registered_proof,
                                                                     int piece_fd_raw,
                                                                     uint64_t unpadded_piece_size);

/**
 * TODO: document
 *
 */
cess_GenerateSingleVanillaProofResponse *cess_generate_single_vanilla_proof(cess_PrivateReplicaInfo replica,
                                                                            const uint64_t *challenges_ptr,
                                                                            size_t challenges_len);

/**
 * TODO: document
 *
 */
cess_GenerateSingleWindowPoStWithVanillaResponse *cess_generate_single_window_post_with_vanilla(cess_RegisteredPoStProof registered_proof,
                                                                                                cess_32ByteArray randomness,
                                                                                                cess_32ByteArray prover_id,
                                                                                                const cess_VanillaProof *vanilla_proofs_ptr,
                                                                                                size_t vanilla_proofs_len,
                                                                                                size_t partition_index);

/**
 * TODO: document
 *
 */
cess_GenerateWindowPoStResponse *cess_generate_window_post(cess_32ByteArray randomness,
                                                           const cess_PrivateReplicaInfo *replicas_ptr,
                                                           size_t replicas_len,
                                                           cess_32ByteArray prover_id);

/**
 * TODO: document
 *
 */
cess_GenerateWindowPoStResponse *cess_generate_window_post_with_vanilla(cess_RegisteredPoStProof registered_proof,
                                                                        cess_32ByteArray randomness,
                                                                        cess_32ByteArray prover_id,
                                                                        const cess_VanillaProof *vanilla_proofs_ptr,
                                                                        size_t vanilla_proofs_len);

/**
 * TODO: document
 *
 */
cess_GenerateWinningPoStResponse *cess_generate_winning_post(cess_32ByteArray randomness,
                                                             const cess_PrivateReplicaInfo *replicas_ptr,
                                                             size_t replicas_len,
                                                             cess_32ByteArray prover_id);

/**
 * TODO: document
 *
 */
cess_GenerateWinningPoStSectorChallenge *cess_generate_winning_post_sector_challenge(cess_RegisteredPoStProof registered_proof,
                                                                                     cess_32ByteArray randomness,
                                                                                     uint64_t sector_set_len,
                                                                                     cess_32ByteArray prover_id);

/**
 * TODO: document
 *
 */
cess_GenerateWinningPoStResponse *cess_generate_winning_post_with_vanilla(cess_RegisteredPoStProof registered_proof,
                                                                          cess_32ByteArray randomness,
                                                                          cess_32ByteArray prover_id,
                                                                          const cess_VanillaProof *vanilla_proofs_ptr,
                                                                          size_t vanilla_proofs_len);

/**
 * Returns an array of strings containing the device names that can be used.
 */
cess_GpuDeviceResponse *cess_get_gpu_devices(void);

/**
 * Returns the number of user bytes that will fit into a staged sector.
 *
 */
uint64_t cess_get_max_user_bytes_per_staged_sector(cess_RegisteredSealProof registered_proof);

/**
 * TODO: document
 *
 */
cess_GetNumPartitionForFallbackPoStResponse *cess_get_num_partition_for_fallback_post(cess_RegisteredPoStProof registered_proof,
                                                                                      size_t num_sectors);

/**
 * Returns the identity of the circuit for the provided PoSt proof type.
 *
 */
cess_StringResponse *cess_get_post_circuit_identifier(cess_RegisteredPoStProof registered_proof);

/**
 * Returns the CID of the Groth parameter file for generating a PoSt.
 *
 */
cess_StringResponse *cess_get_post_params_cid(cess_RegisteredPoStProof registered_proof);

/**
 * Returns the path from which the proofs library expects to find the Groth
 * parameter file used when generating a PoSt.
 *
 */
cess_StringResponse *cess_get_post_params_path(cess_RegisteredPoStProof registered_proof);

/**
 * Returns the CID of the verifying key-file for verifying a PoSt proof.
 *
 */
cess_StringResponse *cess_get_post_verifying_key_cid(cess_RegisteredPoStProof registered_proof);

/**
 * Returns the path from which the proofs library expects to find the verifying
 * key-file used when verifying a PoSt proof.
 *
 */
cess_StringResponse *cess_get_post_verifying_key_path(cess_RegisteredPoStProof registered_proof);

/**
 * Returns the version of the provided seal proof.
 *
 */
cess_StringResponse *cess_get_post_version(cess_RegisteredPoStProof registered_proof);

/**
 * Returns the identity of the circuit for the provided seal proof.
 *
 */
cess_StringResponse *cess_get_seal_circuit_identifier(cess_RegisteredSealProof registered_proof);

/**
 * Returns the CID of the Groth parameter file for sealing.
 *
 */
cess_StringResponse *cess_get_seal_params_cid(cess_RegisteredSealProof registered_proof);

/**
 * Returns the path from which the proofs library expects to find the Groth
 * parameter file used when sealing.
 *
 */
cess_StringResponse *cess_get_seal_params_path(cess_RegisteredSealProof registered_proof);

/**
 * Returns the CID of the verifying key-file for verifying a seal proof.
 *
 */
cess_StringResponse *cess_get_seal_verifying_key_cid(cess_RegisteredSealProof registered_proof);

/**
 * Returns the path from which the proofs library expects to find the verifying
 * key-file used when verifying a seal proof.
 *
 */
cess_StringResponse *cess_get_seal_verifying_key_path(cess_RegisteredSealProof registered_proof);

/**
 * Returns the version of the provided seal proof type.
 *
 */
cess_StringResponse *cess_get_seal_version(cess_RegisteredSealProof registered_proof);

/**
 * Compute the digest of a message
 *
 * # Arguments
 *
 * * `message_ptr` - pointer to a message byte array
 * * `message_len` - length of the byte array
 */
cess_HashResponse *cess_hash(const uint8_t *message_ptr, size_t message_len);

/**
 * Verify that a signature is the aggregated signature of the hashed messages
 *
 * # Arguments
 *
 * * `signature_ptr`             - pointer to a signature byte array (SIGNATURE_BYTES long)
 * * `messages_ptr`              - pointer to an array containing the pointers to the messages
 * * `messages_sizes_ptr`        - pointer to an array containing the lengths of the messages
 * * `messages_len`              - length of the two messages arrays
 * * `flattened_public_keys_ptr` - pointer to a byte array containing public keys
 * * `flattened_public_keys_len` - length of the array
 */
int cess_hash_verify(const uint8_t *signature_ptr,
                     const uint8_t *flattened_messages_ptr,
                     size_t flattened_messages_len,
                     const size_t *message_sizes_ptr,
                     size_t message_sizes_len,
                     const uint8_t *flattened_public_keys_ptr,
                     size_t flattened_public_keys_len);

/**
 * Initializes the logger with a file descriptor where logs will be logged into.
 *
 * This is usually a pipe that was opened on the receiving side of the logs. The logger is
 * initialized on the invocation, subsequent calls won't have any effect.
 *
 * This function must be called right at the start, before any other call. Else the logger will
 * be initializes implicitely and log to stderr.
 */
cess_InitLogFdResponse *cess_init_log_fd(int log_fd);

/**
 * TODO: document
 *
 */
cess_MergeWindowPoStPartitionProofsResponse *cess_merge_window_post_partition_proofs(cess_RegisteredPoStProof registered_proof,
                                                                                     const cess_PartitionSnarkProof *partition_proofs_ptr,
                                                                                     size_t partition_proofs_len);

/**
 * Generate a new private key
 */
cess_PrivateKeyGenerateResponse *cess_private_key_generate(void);

/**
 * Generate a new private key with seed
 *
 * **Warning**: Use this function only for testing or with very secure seeds
 *
 * # Arguments
 *
 * * `raw_seed` - a seed byte array with 32 bytes
 *
 * Returns `NULL` when passed a NULL pointer.
 */
cess_PrivateKeyGenerateResponse *cess_private_key_generate_with_seed(cess_32ByteArray raw_seed);

/**
 * Generate the public key for a private key
 *
 * # Arguments
 *
 * * `raw_private_key_ptr` - pointer to a private key byte array
 *
 * Returns `NULL` when passed invalid arguments.
 */
cess_PrivateKeyPublicKeyResponse *cess_private_key_public_key(const uint8_t *raw_private_key_ptr);

/**
 * Sign a message with a private key and return the signature
 *
 * # Arguments
 *
 * * `raw_private_key_ptr` - pointer to a private key byte array
 * * `message_ptr` - pointer to a message byte array
 * * `message_len` - length of the byte array
 *
 * Returns `NULL` when passed invalid arguments.
 */
cess_PrivateKeySignResponse *cess_private_key_sign(const uint8_t *raw_private_key_ptr,
                                                   const uint8_t *message_ptr,
                                                   size_t message_len);

/**
 * TODO: document
 *
 */
cess_SealCommitPhase1Response *cess_seal_commit_phase1(cess_RegisteredSealProof registered_proof,
                                                       cess_32ByteArray comm_r,
                                                       cess_32ByteArray comm_d,
                                                       const char *cache_dir_path,
                                                       const char *replica_path,
                                                       uint64_t sector_id,
                                                       cess_32ByteArray prover_id,
                                                       cess_32ByteArray ticket,
                                                       cess_32ByteArray seed,
                                                       const cess_PublicPieceInfo *pieces_ptr,
                                                       size_t pieces_len);

cess_SealCommitPhase2Response *cess_seal_commit_phase2(const uint8_t *seal_commit_phase1_output_ptr,
                                                       size_t seal_commit_phase1_output_len,
                                                       uint64_t sector_id,
                                                       cess_32ByteArray prover_id);

/**
 * TODO: document
 *
 */
cess_SealPreCommitPhase1Response *cess_seal_pre_commit_phase1(cess_RegisteredSealProof registered_proof,
                                                              const char *cache_dir_path,
                                                              const char *staged_sector_path,
                                                              const char *sealed_sector_path,
                                                              uint64_t sector_id,
                                                              cess_32ByteArray prover_id,
                                                              cess_32ByteArray ticket,
                                                              const cess_PublicPieceInfo *pieces_ptr,
                                                              size_t pieces_len);

/**
 * TODO: document
 *
 */
cess_SealPreCommitPhase2Response *cess_seal_pre_commit_phase2(const uint8_t *seal_pre_commit_phase1_output_ptr,
                                                              size_t seal_pre_commit_phase1_output_len,
                                                              const char *cache_dir_path,
                                                              const char *sealed_sector_path);

/**
 * TODO: document
 */
cess_UnsealRangeResponse *cess_unseal_range(cess_RegisteredSealProof registered_proof,
                                            const char *cache_dir_path,
                                            int sealed_sector_fd_raw,
                                            int unseal_output_fd_raw,
                                            uint64_t sector_id,
                                            cess_32ByteArray prover_id,
                                            cess_32ByteArray ticket,
                                            cess_32ByteArray comm_d,
                                            uint64_t unpadded_byte_index,
                                            uint64_t unpadded_bytes_amount);

/**
 * Verify that a signature is the aggregated signature of hashes - pubkeys
 *
 * # Arguments
 *
 * * `signature_ptr`             - pointer to a signature byte array (SIGNATURE_BYTES long)
 * * `flattened_digests_ptr`     - pointer to a byte array containing digests
 * * `flattened_digests_len`     - length of the byte array (multiple of DIGEST_BYTES)
 * * `flattened_public_keys_ptr` - pointer to a byte array containing public keys
 * * `flattened_public_keys_len` - length of the array
 */
int cess_verify(const uint8_t *signature_ptr,
                const uint8_t *flattened_digests_ptr,
                size_t flattened_digests_len,
                const uint8_t *flattened_public_keys_ptr,
                size_t flattened_public_keys_len);

/**
 * Verifies the output of an aggregated seal.
 *
 */
cess_VerifyAggregateSealProofResponse *cess_verify_aggregate_seal_proof(cess_RegisteredSealProof registered_proof,
                                                                        cess_RegisteredAggregationProof registered_aggregation,
                                                                        cess_32ByteArray prover_id,
                                                                        const uint8_t *proof_ptr,
                                                                        size_t proof_len,
                                                                        cess_AggregationInputs *commit_inputs_ptr,
                                                                        size_t commit_inputs_len);

/**
 * Verifies the output of seal.
 *
 */
cess_VerifySealResponse *cess_verify_seal(cess_RegisteredSealProof registered_proof,
                                          cess_32ByteArray comm_r,
                                          cess_32ByteArray comm_d,
                                          cess_32ByteArray prover_id,
                                          cess_32ByteArray ticket,
                                          cess_32ByteArray seed,
                                          uint64_t sector_id,
                                          const uint8_t *proof_ptr,
                                          size_t proof_len);

/**
 * Verifies that a proof-of-spacetime is valid.
 */
cess_VerifyWindowPoStResponse *cess_verify_window_post(cess_32ByteArray randomness,
                                                       const cess_PublicReplicaInfo *replicas_ptr,
                                                       size_t replicas_len,
                                                       const cess_PoStProof *proofs_ptr,
                                                       size_t proofs_len,
                                                       cess_32ByteArray prover_id);

/**
 * Verifies that a proof-of-spacetime is valid.
 */
cess_VerifyWinningPoStResponse *cess_verify_winning_post(cess_32ByteArray randomness,
                                                         const cess_PublicReplicaInfo *replicas_ptr,
                                                         size_t replicas_len,
                                                         const cess_PoStProof *proofs_ptr,
                                                         size_t proofs_len,
                                                         cess_32ByteArray prover_id);

/**
 * TODO: document
 *
 */
cess_WriteWithAlignmentResponse *cess_write_with_alignment(cess_RegisteredSealProof registered_proof,
                                                           int src_fd,
                                                           uint64_t src_size,
                                                           int dst_fd,
                                                           const uint64_t *existing_piece_sizes_ptr,
                                                           size_t existing_piece_sizes_len);

/**
 * TODO: document
 *
 */
cess_WriteWithoutAlignmentResponse *cess_write_without_alignment(cess_RegisteredSealProof registered_proof,
                                                                 int src_fd,
                                                                 uint64_t src_size,
                                                                 int dst_fd);

#endif /* cesscrypto_H */

#ifdef __cplusplus
} /* extern "C" */
#endif
