#!/bin/bash

set -e

make clean
cd rust
rm -f Cargo.lock
rustup target add x86_64-apple-darwin --toolchain $(cat rust-toolchain)
rustup target add aarch64-apple-darwin --toolchain $(cat rust-toolchain)
cargo update -p "cess-prooving-system-api"
cargo install cargo-lipo
cargo install cbindgen
cbindgen --clean --config cbindgen.toml --crate cesscrypto --output ../include/cesscrypto.h
cd ..
FFI_BUILD_FROM_SOURCE=1 make
make cgo-gen
go mod tidy
