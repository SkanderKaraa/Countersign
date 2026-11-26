#!/usr/bin/env bash
# Runs the whole suite on every engine with the race detector and reports
# the number of passing and failing tests.
cd "$(dirname "$0")"
for e in goref cedar opa; do
  out=$(COUNTERSIGN_ENGINE=$e go test -race -count=1 -v ./... 2>&1)
  pass=$(grep -c -- "--- PASS" <<<"$out")
  fail=$(grep -c -- "--- FAIL" <<<"$out")
  echo "$e: $pass passed, $fail failed"
  grep -E -- "--- FAIL|_test\.go:[0-9]+:|^(# |vet:)" <<<"$out" | head -8
done
