#!/usr/bin/env python3
"""
PGN -> CSV extractor for positions (streaming, configurable columns)

Features added per request:
  - input PGN file passed as argument (argparse)
  - parses game UID from [Site "https://lichess.org/XXXXXXXX"] -> "XXXXXXXX" saved as `gid`
  - `id` is a sequential integer (1,2,3,...) for each parsed game
  - flags to disable columns (by name) or provide a comma-separated --exclude list

Default output columns (order):
  fen,move,rating,halfmove,side,result,gid,id

Examples:
  python pgn_to_csv.py games.pgn -o positions.csv
  python pgn_to_csv.py games.pgn -o positions.csv --exclude move,side
  python pgn_to_csv.py games.pgn -o positions.csv --no-move --no-rating

"""
import argparse
import chess.pgn
import csv
import time
import re
from pathlib import Path
from typing import List, Optional

SITE_ID_RE = re.compile(r"([^/\s]+)$")

DEFAULT_COLUMNS = [
    "id",
    "gid",
    "rating",
    "side",
    "result",
    "halfmove",
    "move",
    "fen",
]

def parse_site_uid(site_header: Optional[str]) -> Optional[str]:
    if not site_header:
        return None
    m = SITE_ID_RE.search(site_header.strip())
    if not m:
        return None
    return m.group(1)


def parse_elo(elo_str: Optional[str]) -> Optional[int]:
    if not elo_str:
        return None
    try:
        # Some PGNs have '?' or empty; ignore non-int
        return int(elo_str)
    except Exception:
        return None


def column_flags_from_args(args) -> List[str]:
    cols = list(DEFAULT_COLUMNS)
    # handle exclude csv list
    if args.exclude:
        exc = [c.strip() for c in args.exclude.split(",") if c.strip()]
        cols = [c for c in cols if c not in exc]
    # handle individual --no- flags
    for col in DEFAULT_COLUMNS:
        flag = getattr(args, f"no_{col}", False) if hasattr(args, f"no_{col}") else False
        if flag and col in cols:
            cols.remove(col)
    return cols

def get_elapsed(offset):
    return time.time() - offset

def iter_games_positions(pgn_path: Path, progress_interval):
    """Generator yielding per-move data dict for each move in each game."""
    last_status_time = time.time()
    with pgn_path.open("r", encoding="utf-8", errors="replace") as f:
        game_index = 0
        fen_index = 0
        while True:
            game = chess.pgn.read_game(f)
            if game is None:
                break
            game_index += 1
            if game_index % progress_interval == 0:
                print(f"Processed {game_index} games and {fen_index} FEN's, elapsed={get_elapsed(last_status_time):.1f}s")
            headers = game.headers
            result_str = headers.get("Result", "*")
            result = {"1-0": 1.0, "0-1": 0.0, "1/2-1/2": 0.5}.get(result_str, None)
            white_elo = parse_elo(headers.get("WhiteElo"))
            black_elo = parse_elo(headers.get("BlackElo"))
            site = headers.get("Site")
            game_uid = parse_site_uid(site)

            board = game.board()
            halfmove = 0
            for move in game.mainline_moves():
                side = 'w' if board.turn else 'b'
                rating = white_elo if side == 'w' else black_elo
                fen = board.fen()
                row = {
                    "id": game_index,
                    "gid": game_uid if game_uid is not None else "",
                    "rating": rating if rating is not None else "",
                    "side": side,
                    "result": result if result is not None else "",
                    "halfmove": halfmove,
                    "move": move.uci(),
                    "fen": fen,
                }
                yield row
                board.push(move)
                halfmove += 1
            fen_index += halfmove
    print(f"Processed {game_index} games and {fen_index} FEN's, elapsed={get_elapsed(last_status_time):.1f}s")


def main():
    p = argparse.ArgumentParser(description="Extract FEN-positions from PGN into CSV with configurable columns")
    p.add_argument("pgn", type=Path, help="Input PGN file")
    p.add_argument("-o", "--out", type=Path, default=Path("positions.csv"), help="Output CSV file")
    p.add_argument('--status', type=int, default=1000, help='Status print interval (default 1000)')
    p.add_argument("--exclude", type=str, default="", help="Comma-separated column names to exclude (e.g. --exclude move,side)")

    # Add explicit --no- flags for each default column
    for col in DEFAULT_COLUMNS:
        p.add_argument(f"--no-{col}", dest=f"no_{col}", action="store_true", help=f"Exclude column '{col}'")

    p.add_argument("--encoding", type=str, default="utf-8", help="File encoding for PGN (default utf-8)")
    args = p.parse_args()

    columns = column_flags_from_args(args)
    if not columns:
        raise SystemExit("No columns selected for output. Use --exclude or avoid --no- flags.")

    out_path = args.out
    out_path.parent.mkdir(parents=True, exist_ok=True)

    # Open output CSV and stream rows
    with out_path.open("w", encoding="utf-8", newline="") as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=columns, extrasaction="ignore")
        writer.writeheader()

        # iterate over games and write only selected columns
        for row in iter_games_positions(args.pgn, args.status):
            filtered = {k: row[k] for k in columns}
            writer.writerow(filtered)

    print(f"Wrote CSV to {out_path} with columns: {', '.join(columns)}")


if __name__ == "__main__":
    main()
