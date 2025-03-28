// Copyright 2025 The Cockroach Authors.
//
// Use of this software is governed by the CockroachDB Software License
// included in the /LICENSE file.

package txncorrectnesstest

import (
	"testing"

	"github.com/cockroachdb/cockroach/pkg/util/leaktest"
	"github.com/cockroachdb/cockroach/pkg/util/log"
	"github.com/cockroachdb/errors"
)

// TestTxnDBLostDeleteRangeAnomaly verifies that SSI isolation is not
// subject to the lost delete range anomaly. See #6240.
//
// With lost delete range, the delete range for keys B-C leave no
// deletion tombstones (as there are an infinite number of keys in the
// range [B,C)). Without deletion tombstones, the anomaly manifests in
// snapshot mode when txn1 pushes txn2 to commit at a higher timestamp
// and then txn1 writes B and commits at an earlier timestamp. The
// delete range request therefore committed but failed to delete the
// value written to key B.
//
// Lost delete range would typically fail with a history such as:
//
//	D2(A) DR2(B-C) R1(A) C2 W1(B,A) C1
func TestTxnDBLostDeleteRangeAnomaly(t *testing.T) {
	defer leaktest.AfterTest(t)()
	defer log.Scope(t).Close(t)

	// B must not exceed A.
	txn1 := "R(A) W(B,A) C"
	txn2 := "D(A) DR(B-C) C"
	verify := &verifier{
		preHistory: "W(A,1)",
		history:    "R(A) R(B)",
		checkFn: func(env map[string]int64) error {
			if env["B"] != 0 && env["A"] == 0 {
				return errors.Errorf("expected B = %d <= %d = A", env["B"], env["A"])
			}
			return nil
		},
	}
	checkConcurrency("lost update (range delete)", onlySerializable, []string{txn1, txn2}, verify, t)
}
