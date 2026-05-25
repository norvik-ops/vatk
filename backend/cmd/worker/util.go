// Copyright (c) 2026 NorvikOps. All rights reserved.
// SPDX-License-Identifier: Elastic-2.0
// See LICENSE file in the project root for terms.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/matharnica/vakt/internal/config"
	"github.com/matharnica/vakt/internal/shared/crypto"
	"github.com/matharnica/vakt/internal/shared/db"
)

func fromHexChar(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	default:
		return 255
	}
}

// hexDecodeKey decodes a hex-encoded 32-byte master key.
func hexDecodeKey(s string) ([]byte, error) {
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s)-1; i += 2 {
		hi := fromHexChar(s[i])
		lo := fromHexChar(s[i+1])
		if hi == 255 || lo == 255 {
			return nil, fmt.Errorf("invalid hex at position %d", i)
		}
		b[i/2] = hi<<4 | lo
	}
	return b, nil
}

// workerKey decodes cfg.SecretKey and derives a domain-separated sub-key for
// the given service purpose (e.g. "vakt-alert-v1", "vakt-vault-v1").
// Logs fatal and returns nil if the master key is invalid — callers must guard
// with cfg.SecretKey != "" before calling.
func workerKey(cfg *config.Config, purpose string) []byte {
	raw, err := hexDecodeKey(cfg.SecretKey)
	if err != nil {
		log.Fatal().Err(err).Str("purpose", purpose).Msg("worker: invalid VAKT_SECRET_KEY")
		return nil
	}
	k, err := crypto.DeriveServiceKey(raw, purpose)
	if err != nil {
		log.Fatal().Err(err).Str("purpose", purpose).Msg("worker: HKDF derivation failed")
		return nil
	}
	return k
}

func connectDB(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	dbURL := ""
	if cfg != nil {
		dbURL = cfg.DBUrl
	}
	if dbURL == "" {
		dbURL = os.Getenv("VAKT_DB_URL")
	}
	pool, err := db.Connect(ctx, dbURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return pool, nil
}
