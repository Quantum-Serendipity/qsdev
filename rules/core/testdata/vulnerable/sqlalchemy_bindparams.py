# bindparams() only parameterizes a literal SQL string. Lines below a
# "ruleid:" comment must be reported by that rule; lines below an "ok:"
# comment must not be.
from flask import Flask, request
from sqlalchemy import create_engine, text

app = Flask(__name__)
engine = create_engine("postgresql://localhost/mydb")


@app.route("/users/search")
def search_users():
    name = request.args.get("name")
    with engine.connect() as conn:
        # ok: qsdev.core.python.sqli-cursor-execute
        conn.execute(text("SELECT id FROM users WHERE name = :name").bindparams(name=name))
        # ruleid: qsdev.core.python.sqli-cursor-execute
        conn.execute(text("SELECT id FROM users WHERE name = '" + name + "'").bindparams())
