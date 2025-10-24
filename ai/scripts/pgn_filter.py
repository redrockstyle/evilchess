#!/usr/bin/env python3
"""
PGN bulk filter

Description:
 A streaming PGN filter tuned for very large files. Reads input PGN and writes filtered games to output PGN.

Main features:
 - Streaming parsing (memory-efficient)
 - Many filtering flags (rating, moves, time control, time class, termination, bots, anonymous, rated, increment limit, rating-diff)
 - Interactive test/chunk mode (-t N) that shows N games and PASS/FAIL reasons, then waits for Enter to continue to next chunk
 - Intermediate status prints every --status games (default 100k)
 - Duplicate detection using an optional SQLite-based index (scales to large datasets)
 - Basic PGN sanity checks
"""

import argparse
import re
import sys
import time
import hashlib
import sqlite3
from pathlib import Path
from typing import Dict, Tuple, Optional
import gzip

# -------------------------- Utility functions --------------------------

def open_maybe_gzip(path, mode='rt'):
    if str(path).endswith('.gz'):
        return gzip.open(path, mode)
    return open(path, mode, encoding='utf-8', errors='replace')


def parse_tags_and_moves(game_text: str) -> Tuple[Dict[str, str], str]:
    """Parse tags line-by-line and return (tags_dict, moves_text)."""
    tags = {}
    lines = game_text.splitlines()
    tag_re = re.compile(r'^\s*\[([^\s]+)\s+"(.*)"\]\s*$')
    first_non_tag = 0
    for i, ln in enumerate(lines):
        m = tag_re.match(ln)
        if m:
            tags[m.group(1)] = m.group(2)
        else:
            first_non_tag = i
            break
    moves_text = '\n'.join(lines[first_non_tag:]).strip()
    return tags, moves_text


def extract_fullmove_count(moves_text: str) -> int:
    nums = re.findall(r'(\d+)\.', moves_text)
    if not nums:
        return 0
    try:
        return int(nums[-1])
    except Exception:
        return 0


def parse_timecontrol(tc: str) -> Tuple[Optional[int], Optional[int]]:
    if not tc:
        return None, None
    if tc.strip().lower() == 'unlimited':
        return None, None
    parts = tc.strip().split('+')
    try:
        base = int(parts[0]) if parts[0] != '?' else None
    except Exception:
        base = None
    inc = None
    if len(parts) > 1:
        try:
            inc = int(parts[1])
        except Exception:
            inc = None
    return base, inc


def is_bot(name: str) -> bool:
    if not name:
        return False
    s = name.lower()
    return 'bot' in s or (s.startswith('anon') and 'bot' in s)


def is_anonymous(name: str) -> bool:
    if not name:
        return True
    s = name.lower()
    return (s.startswith('anon') or 'anonymous' in s or 'guest' in s or s.strip() == '-')


def game_hash(game_text: str) -> str:
    h = hashlib.sha1()
    normalized = re.sub(r'\s+', ' ', game_text.strip())
    h.update(normalized.encode('utf-8', errors='ignore'))
    return h.hexdigest()


# -------------------------- Dedup store --------------------------

class DedupStore:
    def __init__(self, db_path: str):
        self.conn = sqlite3.connect(db_path)
        self.conn.execute('PRAGMA synchronous=OFF')
        self.conn.execute('PRAGMA journal_mode=WAL')
        self.conn.execute('CREATE TABLE IF NOT EXISTS seen(hash TEXT PRIMARY KEY)')
        self.conn.commit()

    def seen_before(self, h: str) -> bool:
        cur = self.conn.execute('SELECT 1 FROM seen WHERE hash=? LIMIT 1', (h,))
        return cur.fetchone() is not None

    def add(self, h: str):
        try:
            self.conn.execute('INSERT OR IGNORE INTO seen(hash) VALUES(?)', (h,))
        except Exception:
            pass

    def commit(self):
        self.conn.commit()

    def close(self):
        self.conn.commit()
        self.conn.close()


# -------------------------- Filtering logic --------------------------

class PGNFilter:
    def __init__(self, args):
        self.args = args

    def check_game(self, game_text: str) -> Tuple[bool, str]:
        tags, moves_text = parse_tags_and_moves(game_text)

        if not tags:
            return False, 'Missing tags'
        if not moves_text:
            return False, 'No moves'

        white = tags.get('White', '')
        black = tags.get('Black', '')
        white_title = tags.get('WhiteTitle', '')
        black_title = tags.get('BlackTitle', '')
        white_elo = safe_int(tags.get('WhiteElo'))
        black_elo = safe_int(tags.get('BlackElo'))
        wrd = safe_signed_int(tags.get('WhiteRatingDiff'))
        brd = safe_signed_int(tags.get('BlackRatingDiff'))
        result = tags.get('Result', '')
        event = tags.get('Event', '')
        termination = tags.get('Termination', '')
        timecontrol = tags.get('TimeControl', '')

        base, inc = parse_timecontrol(timecontrol)

        if self.args.skip_incomplete and (not result or result.strip() == '*' or result.strip() == ''):
            return False, f'Incomplete result: {result}'

        if self.args.skip_termination and termination and termination.lower() != 'normal':
            return False, f'Termination not normal: {termination}'

        if self.args.skip_bots and (is_bot(white) or is_bot(black) or is_bot(white_title) or is_bot(black_title)):
            return False, f'Bot involved: {white} / {black}'

        if self.args.skip_anonymous and (is_anonymous(white) or is_anonymous(black)):
            return False, f'Anonymous player: {white}/{black}'

        if self.args.skip_nonrated and event and 'rated' not in event.lower():
            return False, f'Non-rated event: {event}'

        if self.args.min_white and white_elo < self.args.min_white:
            return False, f'White rating too low: {white_elo} < {self.args.min_white}'
        if self.args.min_black and black_elo < self.args.min_black:
            return False, f'Black rating too low: {black_elo} < {self.args.min_black}'

        if self.args.max_rating_diff is not None:
            if wrd is not None and abs(wrd) > self.args.max_rating_diff:
                return False, f'WhiteRatingDiff too high: {wrd}'
            if brd is not None and abs(brd) > self.args.max_rating_diff:
                return False, f'BlackRatingDiff too high: {brd}'

        if self.args.max_increment is not None and inc is not None:
            if inc >= self.args.max_increment:
                return False, f'Increment {inc}s >= limit {self.args.max_increment}s'

        full_moves = extract_fullmove_count(moves_text)
        if self.args.min_moves and full_moves < self.args.min_moves:
            return False, f'Moves ({full_moves}) < min_moves ({self.args.min_moves})'

        if self.args.modes:
            evlow = event.lower() if event else ''
            allowed = False
            for m in self.args.modes:
                if m.lower() in evlow:
                    allowed = True
                    break
            if not allowed:
                return False, f'Event not in requested modes: {event}'

        if self.args.timecontrol_exact:
            if timecontrol != self.args.timecontrol_exact:
                return False, f'TimeControl {timecontrol} != required {self.args.timecontrol_exact}'

        if (self.args.time_min is not None) or (self.args.time_max is not None):
            if base is None:
                return False, f'TimeControl unknown: {timecontrol}'
            if self.args.time_min is not None and base < self.args.time_min:
                return False, f'Base time {base} < time_min {self.args.time_min}'
            if self.args.time_max is not None and base > self.args.time_max:
                return False, f'Base time {base} > time_max {self.args.time_max}'

        if '\x00' in game_text:
            return False, 'Binary (null) character present'

        if result and result.strip() and result.strip() != '*':
            if not moves_text.strip().endswith(result.strip()):
                if result.strip() not in moves_text[-40:]:
                    return False, f'Moves/result mismatch (moves end not with {result})'

        return True, 'OK'


def safe_int(s: Optional[str]) -> int:
    try:
        return int(s)
    except Exception:
        return 0


def safe_signed_int(s: Optional[str]) -> Optional[int]:
    if not s:
        return None
    try:
        return int(s.replace('+', ''))
    except Exception:
        return None


# -------------------------- Robust game stream --------------------------

RESULT_RE = re.compile(r'\b(1-0|0-1|1/2-1/2|\*)\s*$')

def game_stream(file_obj):
    """Accumulate lines and yield a game only when a result token is observed."""
    current = []
    seen_any = False
    for raw in file_obj:
        line = raw.rstrip('\n').rstrip('\r')
        if not seen_any and line.strip() == '':
            continue
        seen_any = True
        current.append(line)
        # look at tail of current for result
        joined = '\n'.join(current).strip()
        tail = joined[-200:]
        if RESULT_RE.search(tail):
            yield joined + '\n'
            current = []
            seen_any = False
    if current:
        yield '\n'.join(current).strip() + '\n'


# -------------------------- Processing --------------------------

def process_file(args):
    dedup = None
    if args.dedup_db:
        dedup = DedupStore(args.dedup_db)

    pf = PGNFilter(args)

    input_path = Path(args.input)
    output_path = Path(args.output)

    output_path.parent.mkdir(parents=True, exist_ok=True)

    infile = open_maybe_gzip(input_path, 'rt')
    outfile = open(output_path, 'a', encoding='utf-8')

    total = 0
    passed = 0
    failed = 0
    last_status_time = time.time()

    interactive_chunk = args.test_chunk

    try:
        gen = game_stream(infile)
        while True:
            if interactive_chunk and interactive_chunk > 0:
                for i in range(interactive_chunk):
                    try:
                        game = next(gen)
                    except StopIteration:
                        interactive_chunk = 0
                        break
                    total += 1
                    ok, reason = pf.check_game(game)
                    sys.stdout.write('\n' + '='*40 + f' GAME {total} ' + '='*40 + '\n')
                    sys.stdout.write(game + '\n')
                    sys.stdout.write(f'PREDICTION: {"PASS" if ok else "FAIL"} - {reason}\n')
                    if ok and not args.dry_run:
                        if dedup:
                            h = game_hash(game)
                            if dedup.seen_before(h):
                                failed += 1
                                sys.stdout.write('SKIP (duplicate)\n')
                            else:
                                outfile.write(game + '\n')
                                dedup.add(h)
                                passed += 1
                        else:
                            outfile.write(game + '\n')
                            passed += 1
                    else:
                        failed += 0 if ok else 1

                    if total % args.status == 0:
                        now = time.time()
                        elapsed = now - last_status_time
                        sys.stdout.write(f'--STATUS-- processed {total} games, passed {passed}, failed {failed}, elapsed={elapsed:.1f}s\n')

                if interactive_chunk > 0:
                    ans = input("Press Enter to continue to next chunk (or 'q' to quit): ")
                    if ans.strip().lower() == 'q':
                        print('Quitting as requested by user.')
                        break
                    continue
            else:
                for game in gen:
                    total += 1
                    ok, reason = pf.check_game(game)
                    if ok:
                        if dedup:
                            h = game_hash(game)
                            if dedup.seen_before(h):
                                failed += 1
                                continue
                            dedup.add(h)
                        if not args.dry_run:
                            outfile.write(game + '\n')
                        passed += 1
                    else:
                        failed += 1

                    if total % args.status == 0:
                        now = time.time()
                        elapsed = now - last_status_time
                        print(f'--STATUS-- processed {total} games, passed {passed}, failed {failed}, elapsed={elapsed:.1f}s')
                break
    finally:
        if dedup:
            dedup.commit()
            dedup.close()
        infile.close()
        outfile.close()

    print('\nFinished. Total games: %d, passed: %d, failed: %d' % (total, passed, failed))


# -------------------------- CLI --------------------------

def parse_args():
    p = argparse.ArgumentParser(description='Filter large Lichess PGN dumps (streaming)')
    p.add_argument('-i', '--input', required=True, help='Input PGN file (can be .gz)')
    p.add_argument('-o', '--output', required=True, help='Output PGN file (will be appended)')

    p.add_argument('-t', '--test-chunk', type=int, default=0,
                   help='Interactive test chunk size: show first N games, pause for Enter between chunks. 0 to disable.')
    p.add_argument('--dry-run', action='store_true', help='If set, do not write passing games to output (only preview)')

    p.add_argument('--min-white', type=int, default=0, help='Minimum white rating (inclusive)')
    p.add_argument('--min-black', type=int, default=0, help='Minimum black rating (inclusive)')
    p.add_argument('--min-moves', type=int, default=0, help='Minimum full-move count')
    p.add_argument('--modes', type=lambda s: [x.strip() for x in s.split(',')], default=None,
                   help='Comma-separated game types to allow (e.g. blitz,rapid,classic). Matches Event tag.')
    p.add_argument('--timecontrol-exact', default=None, help='Require exact TimeControl string (e.g. "600+0")')
    p.add_argument('--time-min', type=int, default=None, help='Minimum base time in seconds')
    p.add_argument('--time-max', type=int, default=None, help='Maximum base time in seconds')

    p.add_argument('--skip-incomplete', action='store_true', default=True, help='Skip incomplete games (default ON)')
    p.add_argument('--allow-incomplete', dest='skip_incomplete', action='store_false', help='Allow incomplete games')

    p.add_argument('--skip-termination', action='store_true', default=True, help='Skip non-normal terminations (default ON)')
    p.add_argument('--allow-termination', dest='skip_termination', action='store_false', help='Allow non-normal terminations')

    p.add_argument('--skip-bots', action='store_true', default=True, help='Skip games involving bots (default ON)')
    p.add_argument('--allow-bots', dest='skip_bots', action='store_false', help='Allow bots')

    p.add_argument('--max-rating-diff', type=int, default=150,
                   help='Maximum allowed absolute rating change per player (WhiteRatingDiff/BlackRatingDiff). Default 150')

    p.add_argument('--skip-anonymous', action='store_true', default=True, help='Skip anonymous/guest accounts (default ON)')
    p.add_argument('--allow-anonymous', dest='skip_anonymous', action='store_false', help='Allow anonymous accounts')

    p.add_argument('--skip-nonrated', action='store_true', default=True, help='Skip non-rated games (default ON)')
    p.add_argument('--allow-nonrated', dest='skip_nonrated', action='store_false', help='Allow non-rated games')

    p.add_argument('--max-increment', type=int, default=31,
                   help='Maximum increment seconds allowed per move (games with increment >= this value are skipped). Default 31 (so increments >=31 are skipped)')

    p.add_argument('--status', type=int, default=100000, help='Status print interval (games). Default 100k')

    p.add_argument('--dedup-db', default=None, help='Path to sqlite DB for deduplication (optional). Example: out.dedup.db')

    return p.parse_args()


if __name__ == '__main__':
    args = parse_args()

    print('Starting PGN filtering (fixed parser)')
    print(f'Input: {args.input} -> Output: {args.output}')

    process_file(args)
