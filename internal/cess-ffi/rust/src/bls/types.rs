use crate::bls::api::{cess_BLSDigest, cess_BLSPrivateKey, cess_BLSPublicKey, cess_BLSSignature};

/// HashResponse

#[repr(C)]
pub struct cess_HashResponse {
    pub digest: cess_BLSDigest,
}

#[no_mangle]
pub unsafe extern "C" fn cess_destroy_hash_response(ptr: *mut cess_HashResponse) {
    let _ = Box::from_raw(ptr);
}

/// AggregateResponse

#[repr(C)]
pub struct cess_AggregateResponse {
    pub signature: cess_BLSSignature,
}

#[no_mangle]
pub unsafe extern "C" fn cess_destroy_aggregate_response(ptr: *mut cess_AggregateResponse) {
    let _ = Box::from_raw(ptr);
}

/// PrivateKeyGenerateResponse

#[repr(C)]
pub struct cess_PrivateKeyGenerateResponse {
    pub private_key: cess_BLSPrivateKey,
}

#[no_mangle]
pub unsafe extern "C" fn cess_destroy_private_key_generate_response(
    ptr: *mut cess_PrivateKeyGenerateResponse,
) {
    let _ = Box::from_raw(ptr);
}

/// PrivateKeySignResponse

#[repr(C)]
pub struct cess_PrivateKeySignResponse {
    pub signature: cess_BLSSignature,
}

#[no_mangle]
pub unsafe extern "C" fn cess_destroy_private_key_sign_response(
    ptr: *mut cess_PrivateKeySignResponse,
) {
    let _ = Box::from_raw(ptr);
}

/// PrivateKeyPublicKeyResponse

#[repr(C)]
pub struct cess_PrivateKeyPublicKeyResponse {
    pub public_key: cess_BLSPublicKey,
}

#[no_mangle]
pub unsafe extern "C" fn cess_destroy_private_key_public_key_response(
    ptr: *mut cess_PrivateKeyPublicKeyResponse,
) {
    let _ = Box::from_raw(ptr);
}

/// AggregateResponse

#[repr(C)]
pub struct cess_ZeroSignatureResponse {
    pub signature: cess_BLSSignature,
}

#[no_mangle]
pub unsafe extern "C" fn cess_destroy_zero_signature_response(
    ptr: *mut cess_ZeroSignatureResponse,
) {
    let _ = Box::from_raw(ptr);
}
