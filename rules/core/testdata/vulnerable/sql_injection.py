from flask import Flask, request, jsonify
from sqlalchemy import create_engine, text

app = Flask(__name__)
engine = create_engine("postgresql://localhost/mydb")


@app.route("/users/search")
def search_users():
    name = request.args.get("name")

    # VULNERABLE: string interpolation in SQL query
    query = f"SELECT id, email FROM users WHERE name = '{name}'"
    with engine.connect() as conn:
        result = conn.execute(text(query))
        rows = [dict(row._mapping) for row in result]
    return jsonify(rows)


@app.route("/users/<int:user_id>/notes")
def get_notes(user_id):
    search = request.args.get("q", "")

    # VULNERABLE: f-string in text()
    with engine.connect() as conn:
        result = conn.execute(
            text(f"SELECT * FROM notes WHERE user_id = {user_id} AND body LIKE '%{search}%'")
        )
        rows = [dict(row._mapping) for row in result]
    return jsonify(rows)
