# Logging Module
# 
# Example:
#   example:
#   1) with TeeLogger("out.log"):
#          print("hello")           # write to stdout & out.log
#   2) start_logging("app.log")
#      print("hello")               # write to stdout & out.log
#      stop_logging()
#

import sys
import threading
import contextlib
from typing import Optional, TextIO

# __all__ = ["TeeStream", "TeeLogger", "start_logging", "stop_logging", "is_logging"]

class TeeStream:
    """
    Streams Writer
    Supported write(), flush(), isatty(), encoding etc
    """
    def __init__(self, *streams: TextIO):
        if not streams:
            raise ValueError("Stream is not defined")
        self._streams = streams
        self._lock = threading.RLock()

    def write(self, data: str):
        if not data:
            return 0
        with self._lock:
            total = 0
            for s in self._streams:
                try:
                    written = s.write(data)
                except Exception:
                    written = 0
                try:
                    if written is not None:
                        total += written
                except Exception:
                    pass
            return total

    def flush(self):
        with self._lock:
            for s in self._streams:
                try:
                    s.flush()
                except Exception:
                    pass

    def isatty(self):
        try:
            return getattr(self._streams[0], "isatty", lambda: False)()
        except Exception:
            return False

    def __getattr__(self, name):
        return getattr(self._streams[0], name)

    def writelines(self, lines):
        with self._lock:
            for s in self._streams:
                try:
                    s.writelines(lines)
                except Exception:
                    pass

_original_stdout: Optional[TextIO] = sys.stdout
_original_stderr: Optional[TextIO] = sys.stderr
_log_file: Optional[TextIO] = None
_is_logging = False

def start_logging(path: str, mode: str = "a", encoding: str = "utf-8", *, also_stderr: bool = False):
    """
    Change sys.stdout (and sys.stderr) to TeeStream writer
    """
    global _original_stdout, _original_stderr, _log_file, _is_logging
    if _is_logging:
        raise RuntimeError("Logger has been already started")
    f = open(path, mode, encoding=encoding)
    tee_out = TeeStream(_original_stdout or sys.stdout, f)
    sys.stdout = tee_out
    if also_stderr:
        tee_err = TeeStream(_original_stderr or sys.stderr, f)
        sys.stderr = tee_err
    _log_file = f
    _is_logging = True

def stop_logging():
    """
    Stopping logging and restore sys.stdout/sys.stderr
    """
    global _original_stdout, _original_stderr, _log_file, _is_logging
    if not _is_logging:
        return
    try:
        sys.stdout = _original_stdout or sys.__stdout__
    except Exception:
        pass
    try:
        sys.stderr = _original_stderr or sys.__stderr__
    except Exception:
        pass
    try:
        if _log_file:
            _log_file.flush()
            _log_file.close()
    except Exception:
        pass
    _log_file = None
    _is_logging = False

def is_logging() -> bool:
    return _is_logging

@contextlib.contextmanager
def TeeLogger(path: str, mode: str = "a", encoding: str = "utf-8", *, also_stderr: bool = False):
    """
    Context manager временно дублирует stdout (и опционально stderr) в файл.
    """
    original_out = getattr(sys, "stdout")
    original_err = getattr(sys, "stderr")
    f = open(path, mode, encoding=encoding)
    try:
        sys.stdout = TeeStream(original_out, f)
        if also_stderr:
            sys.stderr = TeeStream(original_err, f)
        yield
    finally:
        try:
            sys.stdout = original_out
            sys.stderr = original_err
        finally:
            try:
                f.flush()
                f.close()
            except Exception:
                pass
