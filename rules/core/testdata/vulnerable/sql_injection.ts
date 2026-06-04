import express, { Request, Response } from "express";
import Knex from "knex";

const app = express();
const knex = Knex({ client: "pg", connection: "postgresql://localhost/mydb" });

app.get("/users/search", async (req: Request, res: Response) => {
  const name = req.query.name as string;

  // VULNERABLE: string interpolation in knex.raw()
  const users = await knex.raw(`SELECT * FROM users WHERE name = '${name}'`);
  res.json(users.rows);
});

app.get("/users/:id", async (req: Request, res: Response) => {
  const id = req.params.id;

  // VULNERABLE: concatenation in raw query
  const user = await knex.raw("SELECT * FROM users WHERE id = " + id);
  res.json(user.rows[0]);
});
