use std::io::{Error, SeekFrom};
use std::ptr;
use std::slice::from_raw_parts;

use anyhow::Result;
use cess_proving_system_api::{
    seal::SealCommitPhase2Output, PieceInfo, RegisteredAggregationProof, RegisteredPoStProof,
    RegisteredSealProof, UnpaddedBytesAmount,
};
use drop_struct_macro_derive::DropStructMacro;
use ffi_toolkit::{code_and_message_impl, free_c_str, CodeAndMessage, FCPResponseStatus};

#[repr(C)]
#[derive(Default, Debug, Clone, Copy)]
pub struct cess_32ByteArray {
    pub inner: [u8; 32],
}

/// FileDescriptorRef does not drop its file descriptor when it is dropped. Its
/// owner must manage the lifecycle of the file descriptor.
pub struct FileDescriptorRef(std::mem::ManuallyDrop<std::fs::File>);

impl FileDescriptorRef {
    #[cfg(not(target_os = "windows"))]
    pub unsafe fn new(raw: std::os::unix::io::RawFd) -> Self {
        use std::os::unix::io::FromRawFd;
        FileDescriptorRef(std::mem::ManuallyDrop::new(std::fs::File::from_raw_fd(raw)))
    }
}

impl std::io::Read for FileDescriptorRef {
    fn read(&mut self, buf: &mut [u8]) -> std::io::Result<usize> {
        self.0.read(buf)
    }
}

impl std::io::Write for FileDescriptorRef {
    fn write(&mut self, buf: &[u8]) -> Result<usize, Error> {
        self.0.write(buf)
    }

    fn flush(&mut self) -> Result<(), Error> {
        self.0.flush()
    }
}

impl std::io::Seek for FileDescriptorRef {
    fn seek(&mut self, pos: SeekFrom) -> Result<u64, Error> {
        self.0.seek(pos)
    }
}

#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub enum cess_RegisteredSealProof {
    StackedDrg2KiBV1,
    StackedDrg8MiBV1,
    StackedDrg32MiBV1,
    StackedDrg512MiBV1,
    StackedDrg32GiBV1,
    StackedDrg64GiBV1,
    StackedDrg2KiBV1_1,
    StackedDrg8MiBV1_1,
    StackedDrg32MiBV1_1,
    StackedDrg512MiBV1_1,
    StackedDrg32GiBV1_1,
    StackedDrg64GiBV1_1,
}

impl From<RegisteredSealProof> for cess_RegisteredSealProof {
    fn from(other: RegisteredSealProof) -> Self {
        match other {
            RegisteredSealProof::StackedDrg2KiBV1 => cess_RegisteredSealProof::StackedDrg2KiBV1,
            RegisteredSealProof::StackedDrg8MiBV1 => cess_RegisteredSealProof::StackedDrg8MiBV1,
            RegisteredSealProof::StackedDrg32MiBV1 => cess_RegisteredSealProof::StackedDrg32MiBV1,
            RegisteredSealProof::StackedDrg512MiBV1 => cess_RegisteredSealProof::StackedDrg512MiBV1,
            RegisteredSealProof::StackedDrg32GiBV1 => cess_RegisteredSealProof::StackedDrg32GiBV1,
            RegisteredSealProof::StackedDrg64GiBV1 => cess_RegisteredSealProof::StackedDrg64GiBV1,

            RegisteredSealProof::StackedDrg2KiBV1_1 => cess_RegisteredSealProof::StackedDrg2KiBV1_1,
            RegisteredSealProof::StackedDrg8MiBV1_1 => cess_RegisteredSealProof::StackedDrg8MiBV1_1,
            RegisteredSealProof::StackedDrg32MiBV1_1 => {
                cess_RegisteredSealProof::StackedDrg32MiBV1_1
            }
            RegisteredSealProof::StackedDrg512MiBV1_1 => {
                cess_RegisteredSealProof::StackedDrg512MiBV1_1
            }
            RegisteredSealProof::StackedDrg32GiBV1_1 => {
                cess_RegisteredSealProof::StackedDrg32GiBV1_1
            }
            RegisteredSealProof::StackedDrg64GiBV1_1 => {
                cess_RegisteredSealProof::StackedDrg64GiBV1_1
            }
        }
    }
}

impl From<cess_RegisteredSealProof> for RegisteredSealProof {
    fn from(other: cess_RegisteredSealProof) -> Self {
        match other {
            cess_RegisteredSealProof::StackedDrg2KiBV1 => RegisteredSealProof::StackedDrg2KiBV1,
            cess_RegisteredSealProof::StackedDrg8MiBV1 => RegisteredSealProof::StackedDrg8MiBV1,
            cess_RegisteredSealProof::StackedDrg32MiBV1 => RegisteredSealProof::StackedDrg32MiBV1,
            cess_RegisteredSealProof::StackedDrg512MiBV1 => RegisteredSealProof::StackedDrg512MiBV1,
            cess_RegisteredSealProof::StackedDrg32GiBV1 => RegisteredSealProof::StackedDrg32GiBV1,
            cess_RegisteredSealProof::StackedDrg64GiBV1 => RegisteredSealProof::StackedDrg64GiBV1,

            cess_RegisteredSealProof::StackedDrg2KiBV1_1 => RegisteredSealProof::StackedDrg2KiBV1_1,
            cess_RegisteredSealProof::StackedDrg8MiBV1_1 => RegisteredSealProof::StackedDrg8MiBV1_1,
            cess_RegisteredSealProof::StackedDrg32MiBV1_1 => {
                RegisteredSealProof::StackedDrg32MiBV1_1
            }
            cess_RegisteredSealProof::StackedDrg512MiBV1_1 => {
                RegisteredSealProof::StackedDrg512MiBV1_1
            }
            cess_RegisteredSealProof::StackedDrg32GiBV1_1 => {
                RegisteredSealProof::StackedDrg32GiBV1_1
            }
            cess_RegisteredSealProof::StackedDrg64GiBV1_1 => {
                RegisteredSealProof::StackedDrg64GiBV1_1
            }
        }
    }
}

#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub enum cess_RegisteredPoStProof {
    StackedDrgWinning2KiBV1,
    StackedDrgWinning8MiBV1,
    StackedDrgWinning32MiBV1,
    StackedDrgWinning512MiBV1,
    StackedDrgWinning32GiBV1,
    StackedDrgWinning64GiBV1,
    StackedDrgWindow2KiBV1,
    StackedDrgWindow8MiBV1,
    StackedDrgWindow32MiBV1,
    StackedDrgWindow512MiBV1,
    StackedDrgWindow32GiBV1,
    StackedDrgWindow64GiBV1,
}

impl From<RegisteredPoStProof> for cess_RegisteredPoStProof {
    fn from(other: RegisteredPoStProof) -> Self {
        use RegisteredPoStProof::*;

        match other {
            StackedDrgWinning2KiBV1 => cess_RegisteredPoStProof::StackedDrgWinning2KiBV1,
            StackedDrgWinning8MiBV1 => cess_RegisteredPoStProof::StackedDrgWinning8MiBV1,
            StackedDrgWinning32MiBV1 => cess_RegisteredPoStProof::StackedDrgWinning32MiBV1,
            StackedDrgWinning512MiBV1 => cess_RegisteredPoStProof::StackedDrgWinning512MiBV1,
            StackedDrgWinning32GiBV1 => cess_RegisteredPoStProof::StackedDrgWinning32GiBV1,
            StackedDrgWinning64GiBV1 => cess_RegisteredPoStProof::StackedDrgWinning64GiBV1,
            StackedDrgWindow2KiBV1 => cess_RegisteredPoStProof::StackedDrgWindow2KiBV1,
            StackedDrgWindow8MiBV1 => cess_RegisteredPoStProof::StackedDrgWindow8MiBV1,
            StackedDrgWindow32MiBV1 => cess_RegisteredPoStProof::StackedDrgWindow32MiBV1,
            StackedDrgWindow512MiBV1 => cess_RegisteredPoStProof::StackedDrgWindow512MiBV1,
            StackedDrgWindow32GiBV1 => cess_RegisteredPoStProof::StackedDrgWindow32GiBV1,
            StackedDrgWindow64GiBV1 => cess_RegisteredPoStProof::StackedDrgWindow64GiBV1,
        }
    }
}

impl From<cess_RegisteredPoStProof> for RegisteredPoStProof {
    fn from(other: cess_RegisteredPoStProof) -> Self {
        use RegisteredPoStProof::*;

        match other {
            cess_RegisteredPoStProof::StackedDrgWinning2KiBV1 => StackedDrgWinning2KiBV1,
            cess_RegisteredPoStProof::StackedDrgWinning8MiBV1 => StackedDrgWinning8MiBV1,
            cess_RegisteredPoStProof::StackedDrgWinning32MiBV1 => StackedDrgWinning32MiBV1,
            cess_RegisteredPoStProof::StackedDrgWinning512MiBV1 => StackedDrgWinning512MiBV1,
            cess_RegisteredPoStProof::StackedDrgWinning32GiBV1 => StackedDrgWinning32GiBV1,
            cess_RegisteredPoStProof::StackedDrgWinning64GiBV1 => StackedDrgWinning64GiBV1,
            cess_RegisteredPoStProof::StackedDrgWindow2KiBV1 => StackedDrgWindow2KiBV1,
            cess_RegisteredPoStProof::StackedDrgWindow8MiBV1 => StackedDrgWindow8MiBV1,
            cess_RegisteredPoStProof::StackedDrgWindow32MiBV1 => StackedDrgWindow32MiBV1,
            cess_RegisteredPoStProof::StackedDrgWindow512MiBV1 => StackedDrgWindow512MiBV1,
            cess_RegisteredPoStProof::StackedDrgWindow32GiBV1 => StackedDrgWindow32GiBV1,
            cess_RegisteredPoStProof::StackedDrgWindow64GiBV1 => StackedDrgWindow64GiBV1,
        }
    }
}

#[repr(C)]
#[derive(Debug, Clone, Copy)]
pub enum cess_RegisteredAggregationProof {
    SnarkPackV1,
}

impl From<RegisteredAggregationProof> for cess_RegisteredAggregationProof {
    fn from(other: RegisteredAggregationProof) -> Self {
        match other {
            RegisteredAggregationProof::SnarkPackV1 => cess_RegisteredAggregationProof::SnarkPackV1,
        }
    }
}

impl From<cess_RegisteredAggregationProof> for RegisteredAggregationProof {
    fn from(other: cess_RegisteredAggregationProof) -> Self {
        match other {
            cess_RegisteredAggregationProof::SnarkPackV1 => RegisteredAggregationProof::SnarkPackV1,
        }
    }
}

#[repr(C)]
#[derive(Clone)]
pub struct cess_PublicPieceInfo {
    pub num_bytes: u64,
    pub comm_p: [u8; 32],
}

impl From<cess_PublicPieceInfo> for PieceInfo {
    fn from(x: cess_PublicPieceInfo) -> Self {
        let cess_PublicPieceInfo { num_bytes, comm_p } = x;
        PieceInfo {
            commitment: comm_p,
            size: UnpaddedBytesAmount(num_bytes),
        }
    }
}

#[repr(C)]
pub struct cess_VanillaProof {
    pub proof_len: libc::size_t,
    pub proof_ptr: *const u8,
}

impl Clone for cess_VanillaProof {
    fn clone(&self) -> Self {
        let slice: &[u8] = unsafe { std::slice::from_raw_parts(self.proof_ptr, self.proof_len) };
        let cloned: Vec<u8> = slice.to_vec();
        debug_assert_eq!(self.proof_len, cloned.len());

        let proof_ptr = cloned.as_ptr();
        std::mem::forget(cloned);

        cess_VanillaProof {
            proof_len: self.proof_len,
            proof_ptr,
        }
    }
}

impl Drop for cess_VanillaProof {
    fn drop(&mut self) {
        // Note that this operation also does the equivalent of
        // libc::free(self.proof_ptr as *mut libc::c_void);
        let _ = unsafe {
            Vec::from_raw_parts(self.proof_ptr as *mut u8, self.proof_len, self.proof_len)
        };
    }
}

#[repr(C)]
pub struct cess_AggregateProof {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub proof_len: libc::size_t,
    pub proof_ptr: *const u8,
}

impl Default for cess_AggregateProof {
    fn default() -> cess_AggregateProof {
        cess_AggregateProof {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            proof_len: 0,
            proof_ptr: ptr::null(),
        }
    }
}

impl Drop for cess_AggregateProof {
    fn drop(&mut self) {
        unsafe {
            // Note that this operation also does the equivalent of
            // libc::free(self.proof_ptr as *mut libc::c_void);
            drop(Vec::from_raw_parts(
                self.proof_ptr as *mut u8,
                self.proof_len,
                self.proof_len,
            ));
            free_c_str(self.error_msg as *mut libc::c_char);
        }
    }
}

code_and_message_impl!(cess_AggregateProof);

#[derive(Clone, Debug)]
pub struct PoStProof {
    pub registered_proof: RegisteredPoStProof,
    pub proof: Vec<u8>,
}

#[repr(C)]
#[derive(Clone)]
pub struct cess_PoStProof {
    pub registered_proof: cess_RegisteredPoStProof,
    pub proof_len: libc::size_t,
    pub proof_ptr: *const u8,
}

impl Drop for cess_PoStProof {
    fn drop(&mut self) {
        let _ = unsafe {
            Vec::from_raw_parts(self.proof_ptr as *mut u8, self.proof_len, self.proof_len)
        };
    }
}

impl From<cess_PoStProof> for PoStProof {
    fn from(other: cess_PoStProof) -> Self {
        let proof = unsafe { from_raw_parts(other.proof_ptr, other.proof_len).to_vec() };

        PoStProof {
            registered_proof: other.registered_proof.into(),
            proof,
        }
    }
}

#[repr(C)]
#[derive(Clone)]
pub struct PartitionSnarkProof {
    pub registered_proof: RegisteredPoStProof,
    pub proof: Vec<u8>,
}

#[repr(C)]
pub struct cess_PartitionSnarkProof {
    pub registered_proof: cess_RegisteredPoStProof,
    pub proof_len: libc::size_t,
    pub proof_ptr: *const u8,
}

impl Clone for cess_PartitionSnarkProof {
    fn clone(&self) -> Self {
        let slice: &[u8] = unsafe { std::slice::from_raw_parts(self.proof_ptr, self.proof_len) };
        let cloned: Vec<u8> = slice.to_vec();
        debug_assert_eq!(self.proof_len, cloned.len());

        let proof_ptr = cloned.as_ptr();
        std::mem::forget(cloned);

        cess_PartitionSnarkProof {
            registered_proof: self.registered_proof,
            proof_len: self.proof_len,
            proof_ptr,
        }
    }
}

impl Drop for cess_PartitionSnarkProof {
    fn drop(&mut self) {
        let _ = unsafe {
            Vec::from_raw_parts(self.proof_ptr as *mut u8, self.proof_len, self.proof_len)
        };
    }
}

impl From<cess_PartitionSnarkProof> for PartitionSnarkProof {
    fn from(other: cess_PartitionSnarkProof) -> Self {
        let proof = unsafe { from_raw_parts(other.proof_ptr, other.proof_len).to_vec() };

        PartitionSnarkProof {
            registered_proof: other.registered_proof.into(),
            proof,
        }
    }
}

#[repr(C)]
#[derive(Clone)]
pub struct cess_PrivateReplicaInfo {
    pub registered_proof: cess_RegisteredPoStProof,
    pub cache_dir_path: *const libc::c_char,
    pub comm_r: [u8; 32],
    pub replica_path: *const libc::c_char,
    pub sector_id: u64,
}

#[repr(C)]
#[derive(Clone)]
pub struct cess_PublicReplicaInfo {
    pub registered_proof: cess_RegisteredPoStProof,
    pub comm_r: [u8; 32],
    pub sector_id: u64,
}

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GenerateWinningPoStSectorChallenge {
    pub error_msg: *const libc::c_char,
    pub status_code: FCPResponseStatus,
    pub ids_ptr: *const u64,
    pub ids_len: libc::size_t,
}

impl Default for cess_GenerateWinningPoStSectorChallenge {
    fn default() -> cess_GenerateWinningPoStSectorChallenge {
        cess_GenerateWinningPoStSectorChallenge {
            ids_len: 0,
            ids_ptr: ptr::null(),
            error_msg: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_GenerateWinningPoStSectorChallenge);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GenerateFallbackSectorChallengesResponse {
    pub error_msg: *const libc::c_char,
    pub status_code: FCPResponseStatus,
    pub ids_ptr: *const u64,
    pub ids_len: libc::size_t,
    pub challenges_ptr: *const u64,
    pub challenges_len: libc::size_t,
    pub challenges_stride: libc::size_t,
}

impl Default for cess_GenerateFallbackSectorChallengesResponse {
    fn default() -> cess_GenerateFallbackSectorChallengesResponse {
        cess_GenerateFallbackSectorChallengesResponse {
            challenges_len: 0,
            challenges_stride: 0,
            challenges_ptr: ptr::null(),
            ids_len: 0,
            ids_ptr: ptr::null(),
            error_msg: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_GenerateFallbackSectorChallengesResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GenerateSingleVanillaProofResponse {
    pub error_msg: *const libc::c_char,
    pub vanilla_proof: cess_VanillaProof,
    pub status_code: FCPResponseStatus,
}

impl Default for cess_GenerateSingleVanillaProofResponse {
    fn default() -> cess_GenerateSingleVanillaProofResponse {
        cess_GenerateSingleVanillaProofResponse {
            error_msg: ptr::null(),
            vanilla_proof: cess_VanillaProof {
                proof_len: 0,
                proof_ptr: ptr::null(),
            },
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_GenerateSingleVanillaProofResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GenerateWinningPoStResponse {
    pub error_msg: *const libc::c_char,
    pub proofs_len: libc::size_t,
    pub proofs_ptr: *const cess_PoStProof,
    pub status_code: FCPResponseStatus,
}

impl Default for cess_GenerateWinningPoStResponse {
    fn default() -> cess_GenerateWinningPoStResponse {
        cess_GenerateWinningPoStResponse {
            error_msg: ptr::null(),
            proofs_len: 0,
            proofs_ptr: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_GenerateWinningPoStResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GenerateWindowPoStResponse {
    pub error_msg: *const libc::c_char,
    pub proofs_len: libc::size_t,
    pub proofs_ptr: *const cess_PoStProof,
    pub faulty_sectors_len: libc::size_t,
    pub faulty_sectors_ptr: *const u64,
    pub status_code: FCPResponseStatus,
}

impl Default for cess_GenerateWindowPoStResponse {
    fn default() -> cess_GenerateWindowPoStResponse {
        cess_GenerateWindowPoStResponse {
            error_msg: ptr::null(),
            proofs_len: 0,
            proofs_ptr: ptr::null(),
            faulty_sectors_len: 0,
            faulty_sectors_ptr: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_GenerateWindowPoStResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GenerateSingleWindowPoStWithVanillaResponse {
    pub error_msg: *const libc::c_char,
    pub partition_proof: cess_PartitionSnarkProof,
    pub faulty_sectors_len: libc::size_t,
    pub faulty_sectors_ptr: *const u64,
    pub status_code: FCPResponseStatus,
}

impl Default for cess_GenerateSingleWindowPoStWithVanillaResponse {
    fn default() -> cess_GenerateSingleWindowPoStWithVanillaResponse {
        cess_GenerateSingleWindowPoStWithVanillaResponse {
            error_msg: ptr::null(),
            partition_proof: cess_PartitionSnarkProof {
                registered_proof: cess_RegisteredPoStProof::StackedDrgWinning2KiBV1,
                proof_len: 0,
                proof_ptr: ptr::null(),
            },
            faulty_sectors_len: 0,
            faulty_sectors_ptr: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_GenerateSingleWindowPoStWithVanillaResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GetNumPartitionForFallbackPoStResponse {
    pub error_msg: *const libc::c_char,
    pub status_code: FCPResponseStatus,
    pub num_partition: libc::size_t,
}

impl Default for cess_GetNumPartitionForFallbackPoStResponse {
    fn default() -> cess_GetNumPartitionForFallbackPoStResponse {
        cess_GetNumPartitionForFallbackPoStResponse {
            error_msg: ptr::null(),
            num_partition: 0,
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_GetNumPartitionForFallbackPoStResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_MergeWindowPoStPartitionProofsResponse {
    pub error_msg: *const libc::c_char,
    pub proof: cess_PoStProof,
    pub status_code: FCPResponseStatus,
}

impl Default for cess_MergeWindowPoStPartitionProofsResponse {
    fn default() -> cess_MergeWindowPoStPartitionProofsResponse {
        cess_MergeWindowPoStPartitionProofsResponse {
            error_msg: ptr::null(),
            proof: cess_PoStProof {
                registered_proof: cess_RegisteredPoStProof::StackedDrgWinning2KiBV1,
                proof_len: 0,
                proof_ptr: ptr::null(),
            },
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_MergeWindowPoStPartitionProofsResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_WriteWithAlignmentResponse {
    pub comm_p: [u8; 32],
    pub error_msg: *const libc::c_char,
    pub left_alignment_unpadded: u64,
    pub status_code: FCPResponseStatus,
    pub total_write_unpadded: u64,
}

impl Default for cess_WriteWithAlignmentResponse {
    fn default() -> cess_WriteWithAlignmentResponse {
        cess_WriteWithAlignmentResponse {
            comm_p: Default::default(),
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            left_alignment_unpadded: 0,
            total_write_unpadded: 0,
        }
    }
}

code_and_message_impl!(cess_WriteWithAlignmentResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_WriteWithoutAlignmentResponse {
    pub comm_p: [u8; 32],
    pub error_msg: *const libc::c_char,
    pub status_code: FCPResponseStatus,
    pub total_write_unpadded: u64,
}

impl Default for cess_WriteWithoutAlignmentResponse {
    fn default() -> cess_WriteWithoutAlignmentResponse {
        cess_WriteWithoutAlignmentResponse {
            comm_p: Default::default(),
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            total_write_unpadded: 0,
        }
    }
}

code_and_message_impl!(cess_WriteWithoutAlignmentResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_SealPreCommitPhase1Response {
    pub error_msg: *const libc::c_char,
    pub status_code: FCPResponseStatus,
    pub seal_pre_commit_phase1_output_ptr: *const u8,
    pub seal_pre_commit_phase1_output_len: libc::size_t,
}

impl Default for cess_SealPreCommitPhase1Response {
    fn default() -> cess_SealPreCommitPhase1Response {
        cess_SealPreCommitPhase1Response {
            error_msg: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
            seal_pre_commit_phase1_output_ptr: ptr::null(),
            seal_pre_commit_phase1_output_len: 0,
        }
    }
}

code_and_message_impl!(cess_SealPreCommitPhase1Response);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_FauxRepResponse {
    pub error_msg: *const libc::c_char,
    pub status_code: FCPResponseStatus,
    pub commitment: [u8; 32],
}

impl Default for cess_FauxRepResponse {
    fn default() -> cess_FauxRepResponse {
        cess_FauxRepResponse {
            error_msg: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
            commitment: Default::default(),
        }
    }
}

code_and_message_impl!(cess_FauxRepResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_SealPreCommitPhase2Response {
    pub error_msg: *const libc::c_char,
    pub status_code: FCPResponseStatus,
    pub registered_proof: cess_RegisteredSealProof,
    pub comm_d: [u8; 32],
    pub comm_r: [u8; 32],
}

impl Default for cess_SealPreCommitPhase2Response {
    fn default() -> cess_SealPreCommitPhase2Response {
        cess_SealPreCommitPhase2Response {
            error_msg: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
            registered_proof: cess_RegisteredSealProof::StackedDrg2KiBV1,
            comm_d: Default::default(),
            comm_r: Default::default(),
        }
    }
}

code_and_message_impl!(cess_SealPreCommitPhase2Response);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_SealCommitPhase1Response {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub seal_commit_phase1_output_ptr: *const u8,
    pub seal_commit_phase1_output_len: libc::size_t,
}

impl Default for cess_SealCommitPhase1Response {
    fn default() -> cess_SealCommitPhase1Response {
        cess_SealCommitPhase1Response {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            seal_commit_phase1_output_ptr: ptr::null(),
            seal_commit_phase1_output_len: 0,
        }
    }
}

code_and_message_impl!(cess_SealCommitPhase1Response);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_SealCommitPhase2Response {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub proof_ptr: *const u8,
    pub proof_len: libc::size_t,
    pub commit_inputs_ptr: *const cess_AggregationInputs,
    pub commit_inputs_len: libc::size_t,
}

impl From<&cess_SealCommitPhase2Response> for SealCommitPhase2Output {
    fn from(other: &cess_SealCommitPhase2Response) -> Self {
        let slice: &[u8] = unsafe { std::slice::from_raw_parts(other.proof_ptr, other.proof_len) };
        let proof: Vec<u8> = slice.to_vec();

        SealCommitPhase2Output { proof }
    }
}

impl Default for cess_SealCommitPhase2Response {
    fn default() -> cess_SealCommitPhase2Response {
        cess_SealCommitPhase2Response {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            proof_ptr: ptr::null(),
            proof_len: 0,
            commit_inputs_ptr: ptr::null(),
            commit_inputs_len: 0,
        }
    }
}

// General note on Vec::from_raw_parts vs std::slice::from_raw_parts:
//
// Vec::from_raw_parts takes ownership of the allocation and will free
// it when it's dropped.
//
// std::slice::from_raw_parts borrows the allocation, and does not
// affect ownership.
//
// In general, usages should borrow via the slice and Drop methods
// should take ownership using the Vec.
impl Clone for cess_SealCommitPhase2Response {
    fn clone(&self) -> Self {
        let slice: &[u8] = unsafe { std::slice::from_raw_parts(self.proof_ptr, self.proof_len) };
        let proof: Vec<u8> = slice.to_vec();
        debug_assert_eq!(self.proof_len, proof.len());

        let proof_len = proof.len();
        let proof_ptr = proof.as_ptr();

        let slice: &[cess_AggregationInputs] =
            unsafe { std::slice::from_raw_parts(self.commit_inputs_ptr, self.commit_inputs_len) };
        let commit_inputs: Vec<cess_AggregationInputs> = slice.to_vec();
        debug_assert_eq!(self.commit_inputs_len, commit_inputs.len());

        let commit_inputs_len = commit_inputs.len();
        let commit_inputs_ptr = commit_inputs.as_ptr();

        std::mem::forget(proof);
        std::mem::forget(commit_inputs);

        cess_SealCommitPhase2Response {
            status_code: self.status_code,
            error_msg: self.error_msg,
            proof_ptr,
            proof_len,
            commit_inputs_ptr,
            commit_inputs_len,
        }
    }
}

code_and_message_impl!(cess_SealCommitPhase2Response);

#[repr(C)]
#[derive(Clone, DropStructMacro)]
pub struct cess_AggregationInputs {
    pub comm_r: cess_32ByteArray,
    pub comm_d: cess_32ByteArray,
    pub sector_id: u64,
    pub ticket: cess_32ByteArray,
    pub seed: cess_32ByteArray,
}

impl Default for cess_AggregationInputs {
    fn default() -> cess_AggregationInputs {
        cess_AggregationInputs {
            comm_r: cess_32ByteArray::default(),
            comm_d: cess_32ByteArray::default(),
            sector_id: 0,
            ticket: cess_32ByteArray::default(),
            seed: cess_32ByteArray::default(),
        }
    }
}
#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_UnsealRangeResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
}

impl Default for cess_UnsealRangeResponse {
    fn default() -> cess_UnsealRangeResponse {
        cess_UnsealRangeResponse {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
        }
    }
}

code_and_message_impl!(cess_UnsealRangeResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_VerifySealResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub is_valid: bool,
}

impl Default for cess_VerifySealResponse {
    fn default() -> cess_VerifySealResponse {
        cess_VerifySealResponse {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            is_valid: false,
        }
    }
}

code_and_message_impl!(cess_VerifySealResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_VerifyAggregateSealProofResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub is_valid: bool,
}

impl Default for cess_VerifyAggregateSealProofResponse {
    fn default() -> cess_VerifyAggregateSealProofResponse {
        cess_VerifyAggregateSealProofResponse {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            is_valid: false,
        }
    }
}

code_and_message_impl!(cess_VerifyAggregateSealProofResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_VerifyWinningPoStResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub is_valid: bool,
}

impl Default for cess_VerifyWinningPoStResponse {
    fn default() -> cess_VerifyWinningPoStResponse {
        cess_VerifyWinningPoStResponse {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            is_valid: false,
        }
    }
}

code_and_message_impl!(cess_VerifyWinningPoStResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_VerifyWindowPoStResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub is_valid: bool,
}

impl Default for cess_VerifyWindowPoStResponse {
    fn default() -> cess_VerifyWindowPoStResponse {
        cess_VerifyWindowPoStResponse {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            is_valid: false,
        }
    }
}

code_and_message_impl!(cess_VerifyWindowPoStResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_FinalizeTicketResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub ticket: [u8; 32],
}

impl Default for cess_FinalizeTicketResponse {
    fn default() -> Self {
        cess_FinalizeTicketResponse {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            ticket: [0u8; 32],
        }
    }
}

code_and_message_impl!(cess_FinalizeTicketResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GeneratePieceCommitmentResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub comm_p: [u8; 32],
    /// The number of unpadded bytes in the original piece plus any (unpadded)
    /// alignment bytes added to create a whole merkle tree.
    pub num_bytes_aligned: u64,
}

impl Default for cess_GeneratePieceCommitmentResponse {
    fn default() -> cess_GeneratePieceCommitmentResponse {
        cess_GeneratePieceCommitmentResponse {
            status_code: FCPResponseStatus::FCPNoError,
            comm_p: Default::default(),
            error_msg: ptr::null(),
            num_bytes_aligned: 0,
        }
    }
}

code_and_message_impl!(cess_GeneratePieceCommitmentResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_GenerateDataCommitmentResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub comm_d: [u8; 32],
}

impl Default for cess_GenerateDataCommitmentResponse {
    fn default() -> cess_GenerateDataCommitmentResponse {
        cess_GenerateDataCommitmentResponse {
            status_code: FCPResponseStatus::FCPNoError,
            comm_d: Default::default(),
            error_msg: ptr::null(),
        }
    }
}

code_and_message_impl!(cess_GenerateDataCommitmentResponse);

///

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_StringResponse {
    pub status_code: FCPResponseStatus,
    pub error_msg: *const libc::c_char,
    pub string_val: *const libc::c_char,
}

impl Default for cess_StringResponse {
    fn default() -> cess_StringResponse {
        cess_StringResponse {
            status_code: FCPResponseStatus::FCPNoError,
            error_msg: ptr::null(),
            string_val: ptr::null(),
        }
    }
}

code_and_message_impl!(cess_StringResponse);

#[repr(C)]
#[derive(DropStructMacro)]
pub struct cess_ClearCacheResponse {
    pub error_msg: *const libc::c_char,
    pub status_code: FCPResponseStatus,
}

impl Default for cess_ClearCacheResponse {
    fn default() -> cess_ClearCacheResponse {
        cess_ClearCacheResponse {
            error_msg: ptr::null(),
            status_code: FCPResponseStatus::FCPNoError,
        }
    }
}

code_and_message_impl!(cess_ClearCacheResponse);
