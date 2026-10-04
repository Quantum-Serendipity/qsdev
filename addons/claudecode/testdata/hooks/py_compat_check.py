"""Check a hook script loads on the oldest Python qsdev hooks support.

Usage: py_compat_check.py <script> <major.minor>

The target version is the Go constant types.MinHookPython (3.9, e.g. macOS's
/usr/bin/python3): the script must parse with that grammar, and any PEP 604
`X | Y` annotation, which 3.9 evaluates at import and rejects with TypeError,
needs `from __future__ import annotations`. Prints the problems, one per line;
prints nothing when the script is compatible."""
import ast
import sys

path, target = sys.argv[1], sys.argv[2]
version = tuple(int(p) for p in target.split("."))
src = open(path, encoding="utf-8").read()
try:
    tree = ast.parse(src, filename=path, feature_version=version)
except SyntaxError as e:
    print(f"{path}:{e.lineno}: not valid Python {target} syntax: {e.msg}")
    sys.exit(0)

postponed = any(
    isinstance(n, ast.ImportFrom) and n.module == "__future__"
    and any(a.name == "annotations" for a in n.names)
    for n in tree.body
)


def annotations(node):
    if isinstance(node, (ast.FunctionDef, ast.AsyncFunctionDef)):
        args = node.args
        for a in args.posonlyargs + args.args + args.kwonlyargs + [args.vararg, args.kwarg]:
            if a is not None and a.annotation is not None:
                yield a.annotation
        if node.returns is not None:
            yield node.returns
    elif isinstance(node, ast.AnnAssign):
        yield node.annotation


if not postponed:
    for node in ast.walk(tree):
        for ann in annotations(node):
            if any(isinstance(n, ast.BinOp) and isinstance(n.op, ast.BitOr) for n in ast.walk(ann)):
                print(f"{path}:{ann.lineno}: `|` annotation fails at import on Python {target}; "
                      "add `from __future__ import annotations`")
