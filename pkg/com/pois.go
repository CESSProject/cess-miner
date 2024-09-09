/*
	Copyright (C) CESS. All rights reserved.
	Copyright (C) Cumulus Encrypted Storage System. All rights reserved.

	SPDX-License-Identifier: Apache-2.0
*/

package com

import (
	"context"
	"time"

	"github.com/CESSProject/cess-miner/pkg/com/pb"
	"google.golang.org/grpc"
)

func NewPoisCertifierApiClient(addr string, opts ...grpc.DialOption) (pb.PoisCertifierApiClient, error) {
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return pb.NewPoisCertifierApiClient(conn), nil
}

func NewPoisVerifierApiClient(addr string, opts ...grpc.DialOption) (pb.PoisVerifierApiClient, error) {
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return pb.NewPoisVerifierApiClient(conn), nil
}

func RequestMinerGetNewKey(
	addr string,
	accountKey []byte,
	timeout time.Duration,
	dialOpts []grpc.DialOption,
	callOpts []grpc.CallOption,
) (*pb.ResponseMinerInitParam, error) {
	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := pb.NewPoisCertifierApiClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := c.RequestMinerGetNewKey(
		ctx,
		&pb.RequestMinerInitParam{
			MinerId: accountKey,
		},
		callOpts...)
	return result, err
}

func RequestMinerCommitGenChall(
	addr string,
	commitGenChall *pb.RequestMinerCommitGenChall,
	timeout time.Duration,
	dialOpts []grpc.DialOption,
	callOpts []grpc.CallOption,
) (*pb.Challenge, error) {
	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := pb.NewPoisCertifierApiClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := c.RequestMinerCommitGenChall(ctx, commitGenChall, callOpts...)
	return result, err
}

func RequestVerifyCommitProof(
	addr string,
	verifyCommitAndAccProof *pb.RequestVerifyCommitAndAccProof,
	timeout time.Duration,
	dialOpts []grpc.DialOption,
	callOpts []grpc.CallOption,
) (*pb.ResponseVerifyCommitOrDeletionProof, error) {
	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := pb.NewPoisCertifierApiClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := c.RequestVerifyCommitProof(ctx, verifyCommitAndAccProof, callOpts...)
	return result, err
}

func RequestVerifyDeletionProof(
	addr string,
	requestVerifyDeletionProof *pb.RequestVerifyDeletionProof,
	timeout time.Duration,
	dialOpts []grpc.DialOption,
	callOpts []grpc.CallOption,
) (*pb.ResponseVerifyCommitOrDeletionProof, error) {
	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := pb.NewPoisCertifierApiClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := c.RequestVerifyDeletionProof(ctx, requestVerifyDeletionProof, callOpts...)
	return result, err
}

func RequestSpaceProofVerifySingleBlock(
	addr string,
	requestSpaceProofVerify *pb.RequestSpaceProofVerify,
	timeout time.Duration,
	dialOpts []grpc.DialOption,
	callOpts []grpc.CallOption,
) (*pb.ResponseSpaceProofVerify, error) {
	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := pb.NewPoisVerifierApiClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := c.RequestSpaceProofVerifySingleBlock(ctx, requestSpaceProofVerify, callOpts...)
	return result, err
}

func RequestVerifySpaceTotal(
	addr string,
	requestSpaceProofVerifyTotal *pb.RequestSpaceProofVerifyTotal,
	timeout time.Duration,
	dialOpts []grpc.DialOption,
	callOpts []grpc.CallOption,
) (*pb.ResponseSpaceProofVerifyTotal, error) {
	conn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := pb.NewPoisVerifierApiClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	result, err := c.RequestVerifySpaceTotal(ctx, requestSpaceProofVerifyTotal, callOpts...)
	return result, err
}
