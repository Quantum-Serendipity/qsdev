import express, { Request, Response } from "express";
import Knex from "knex";

const app = express();
const knex = Knex({ client: "pg", connection: "postgresql://localhost/mydb" });

app.get("/users/search", async (req: Request, res: Response) => {
  const name = req.query.name as string;

  // SAFE: knex.raw() with binding parameter
  const users = await knex.raw("SELECT * FROM users WHERE name = ?", [name]);
  res.json(users.rows);
});

app.get("/users/:id", async (req: Request, res: Response) => {
  const id = parseInt(req.params.id, 10);

  // SAFE: parseInt sanitizes input + parameterized query
  if (isNaN(id)) {
    res.status(400).json({ error: "invalid id" });
    return;
  }
  const user = await knex.raw("SELECT * FROM users WHERE id = ?", [id]);
  res.json(user.rows[0]);
});
