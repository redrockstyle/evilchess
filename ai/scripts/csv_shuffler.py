#!/usr/bin/env python3
"""
CSV shuffler with chunked shuffle support

Features:
 - Two modes: 'chunk' (default) and 'full'
 - chunk shuffle reads the input CSV in chunks (default chunk_size=500_000), shuffles rows inside each chunk,
   writes temporary chunk files, then concatenates chunks in a random order to produce a shuffled CSV.
 - full mode loads the entire CSV into memory and performs a single global shuffle (useful for smaller files).
 - Requires an input CSV positional argument and an output path via -o/--out
 - Handles an 'id' column (via --idx): if input has 'id', it is preserved; if not, an 'id' column will be added to the
   **output** CSV and numbered sequentially starting from 0.
 - Deterministic behavior via --random-state

Usage examples:
  python shuffle_csv.py input.csv -o shuffled.csv
  python shuffle_csv.py input.csv -o shuffled.csv --mode full --random-state 123
  python shuffle_csv.py input.csv -o shuffled.csv --chunk-size 200000

"""
import argparse
import tempfile
import shutil
from pathlib import Path
import pandas as pd
import numpy as np
import csv
import sys


def chunk_shuffle(input_path: Path, output_path: Path, chunk_size: int, random_state: int, need_index: bool):
    rng = np.random.RandomState(random_state)
    tempdir = Path(tempfile.mkdtemp(prefix="csvshuffle_"))
    tmp_files = []
    print(f"Temporary directory for chunks: {tempdir}")

    # Quick read of header to know columns
    try:
        first_cols = pd.read_csv(input_path, nrows=0).columns.tolist()
    except Exception as e:
        shutil.rmtree(tempdir)
        raise

    id_present = ('id' in first_cols)
    print(f"Detected columns: {first_cols}")
    print(f"id column present: {id_present}")

    # Phase 1: read, shuffle inside chunk, write temp files
    print("Phase 1: reading input in chunks and creating shuffled chunk files...")
    for i, chunk in enumerate(pd.read_csv(input_path, chunksize=chunk_size)):
        # shuffle inside chunk with per-chunk seed
        seed = rng.randint(0, 2 ** 31 - 1)
        chunk = chunk.sample(frac=1, random_state=seed).reset_index(drop=True)

        tmp_path = tempdir / f"chunk_{i:06d}.csv"
        chunk.to_csv(tmp_path, index=False)
        tmp_files.append(tmp_path)
        if (i + 1) % 10 == 0:
            print(f"  created {i+1} chunk files...")

    if not tmp_files:
        shutil.rmtree(tempdir)
        raise SystemExit("No data read from input CSV (no chunks created).")

    # Phase 2: permute chunk order and write final output
    order = rng.permutation(len(tmp_files))
    print(f"Phase 2: concatenating {len(tmp_files)} chunk files in random order...")

    # Prepare output header
    sample_header = pd.read_csv(tmp_files[0], nrows=0).columns.tolist()
    if id_present:
        header = sample_header
    elif need_index:
        header = ['id'] + sample_header

    with output_path.open('w', newline='', encoding='utf-8') as out_f:
        writer = csv.writer(out_f)
        writer.writerow(header)

        next_id = 0
        # iterate chunks in randomized order
        for idx in order:
            tmp = tmp_files[idx]
            with tmp.open('r', encoding='utf-8', newline='') as tf:
                reader = csv.reader(tf)
                # skip header of chunk
                try:
                    chunk_header = next(reader)
                except StopIteration:
                    continue
                for row in reader:
                    if not id_present and need_index:
                        writer.writerow([next_id] + row)
                    else:
                        writer.writerow(row)
                    next_id += 1

    # cleanup
    shutil.rmtree(tempdir)
    print(f"Wrote shuffled CSV to {output_path}")
    print("Temporary files removed.")


def full_shuffle(input_path: Path, output_path: Path, random_state: int, idx: bool):
    print("Full shuffle: loading entire CSV into memory...")
    df = pd.read_csv(input_path)
    id_present = 'id' in df.columns.tolist()
    df = df.sample(frac=1, random_state=random_state).reset_index(drop=True)
    if not id_present and idx:
        df.insert(0, 'id', range(len(df)))
    df.to_csv(output_path, index=False)
    print(f"Wrote shuffled CSV to {output_path}")


def main():
    p = argparse.ArgumentParser(description="Shuffle CSV file (chunked or full shuffle)")
    p.add_argument('input', type=Path, help='Input CSV file')
    p.add_argument('-o', '--out', type=Path, required=True, help='Output CSV file (shuffled)')
    p.add_argument('--idx', action="store_false", default=False, help='Add "id" column (default false)')
    p.add_argument('--mode', choices=['chunk', 'full'], default='chunk', help='Shuffle mode: chunk (default) or full')
    p.add_argument('--chunk-size', type=int, default=500_000, help='Rows per chunk for chunk shuffle (default: 500000)')
    p.add_argument('--random-state', type=int, default=42, help='Random state / seed (default: 42)')

    args = p.parse_args()

    if not args.input.exists():
        raise SystemExit(f"Input file does not exist: {args.input}")

    if args.mode == 'chunk':
        chunk_shuffle(args.input, args.out, args.chunk_size, args.random_state, need_index=args.idx)
    else:
        full_shuffle(args.input, args.out, args.random_state, args.idx)


if __name__ == '__main__':
    main()
