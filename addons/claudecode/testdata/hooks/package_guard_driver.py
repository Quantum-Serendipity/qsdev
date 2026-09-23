"""Offline end-to-end driver for the package-guard hook.

Runs templates/hooks/package-guard.py as `__main__` (so its top-level error
handling is exercised too) with the hook envelope from PG_ENVELOPE on stdin and
urllib.request.urlopen replaced by a stub registry, so no network is touched:

  - OSV.dev reports a vulnerability for names in PG_VULN, none otherwise.
  - npm / PyPI / crates.io report a release published now for names in
    PG_FRESH, and one from 2015 otherwise.
  - PG_URL_ERROR=1 makes every request fail with URLError (network down);
    PG_INTERNAL_ERROR=1 makes it raise RuntimeError (a hook bug).

Prints one JSON object: the exit code, the parsed hookSpecificOutput (or
null), stderr, and every package the stub was asked about.
"""
import contextlib
import io
import json
import os
import runpy
import sys
import urllib.error
import urllib.parse
import urllib.request
from datetime import datetime, timezone

VULN = set(filter(None, os.environ.get("PG_VULN", "").split(",")))
FRESH = set(filter(None, os.environ.get("PG_FRESH", "").split(",")))
LOOKUPS = []


class _Resp(io.BytesIO):
    def __enter__(self):
        return self

    def __exit__(self, *exc):
        return False


def _published(name):
    if name in FRESH:
        return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    return "2015-01-01T00:00:00Z"


def fake_urlopen(req, timeout=None):
    url = req.full_url if hasattr(req, "full_url") else str(req)
    if os.environ.get("PG_INTERNAL_ERROR") == "1":
        raise RuntimeError("stub: internal hook error")
    if os.environ.get("PG_URL_ERROR") == "1":
        raise urllib.error.URLError("stub: network unreachable")
    if "osv.dev" in url:
        body = json.loads(req.data.decode())
        name = body["package"]["name"]
        LOOKUPS.append({"source": "osv", "name": name, "version": body.get("version")})
        vulns = [{"id": "GHSA-stub-0001"}] if name in VULN else []
        return _Resp(json.dumps({"vulns": vulns}).encode())
    if "registry.npmjs.org" in url:
        name = urllib.parse.unquote(url.split("registry.npmjs.org/", 1)[1])
        LOOKUPS.append({"source": "npm", "name": name})
        base = name.split("/")[0] if not name.startswith("@") else "/".join(name.split("/")[:2])
        return _Resp(json.dumps({
            "name": base,
            "version": "1.0.0",
            "dist-tags": {"latest": "1.0.0"},
            "time": {"1.0.0": _published(base)},
        }).encode())
    if "pypi.org" in url:
        name = urllib.parse.unquote(url.split("/pypi/", 1)[1].split("/", 1)[0])
        LOOKUPS.append({"source": "pypi", "name": name})
        return _Resp(json.dumps({
            "info": {"version": "1.0.0"},
            "urls": [{"upload_time_iso_8601": _published(name)}],
            "releases": {v: [{"upload_time_iso_8601": _published(name)}] for v in ("1.0.0", "2.31.0")},
        }).encode())
    if "crates.io" in url:
        name = urllib.parse.unquote(url.split("/crates/", 1)[1].split("/", 1)[0])
        LOOKUPS.append({"source": "crates", "name": name})
        return _Resp(json.dumps({
            "crate": {"newest_version": "1.0.0", "max_stable_version": "1.0.0"},
            "versions": [{"num": "1.0.0", "created_at": _published(name)}],
        }).encode())
    raise urllib.error.URLError("stub: unexpected URL " + url)


def main():
    urllib.request.urlopen = fake_urlopen
    sys.stdin = io.StringIO(os.environ["PG_ENVELOPE"])
    out, err = io.StringIO(), io.StringIO()
    code = 0
    with contextlib.redirect_stdout(out), contextlib.redirect_stderr(err):
        try:
            runpy.run_path(os.environ["PG_PATH"], run_name="__main__")
        except SystemExit as e:
            code = e.code if isinstance(e.code, int) else (0 if e.code is None else 1)
    text = out.getvalue().strip()
    hook_output = json.loads(text)["hookSpecificOutput"] if text else None
    print(json.dumps({
        "exit": code,
        "output": hook_output,
        "stderr": err.getvalue(),
        "lookups": LOOKUPS,
    }))


main()
