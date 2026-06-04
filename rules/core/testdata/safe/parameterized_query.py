from flask import Flask, request, jsonify
from sqlalchemy import create_engine, text

app = Flask(__name__)
engine = create_engine("postgresql://localhost/mydb")


@app.route("/users/search")
def search_users():
    name = request.args.get("name")

    # SAFE: bound parameters via bindparams
    with engine.connect() as conn:
        result = conn.execute(
            text("SELECT id, email FROM users WHERE name = :name").bindparams(name=name)
        )
        rows = [dict(row._mapping) for row in result]
    return jsonify(rows)


@app.route("/users/<user_id>/notes")
def get_notes(user_id):
    search = request.args.get("q", "")

    # SAFE: parameterized query with named placeholders
    uid = int(user_id)
    with engine.connect() as conn:
        result = conn.execute(
            text("SELECT * FROM notes WHERE user_id = :uid AND body LIKE :search"),
            {"uid": uid, "search": f"%{search}%"},
        )
        rows = [dict(row._mapping) for row in result]
    return jsonify(rows)
