import express, { Request, Response } from "express";
import { PrismaClient } from "@prisma/client";

const app = express();
const prisma = new PrismaClient();

app.get("/users/search", async (req: Request, res: Response) => {
  const name = req.query.name as string;

  // SAFE: tagged template literal auto-parameterizes
  const users = await prisma.$queryRaw`
    SELECT * FROM "User" WHERE name = ${name}
  `;
  res.json(users);
});

app.post("/users/delete", async (req: Request, res: Response) => {
  const id = parseInt(req.body.id, 10);
  if (isNaN(id)) {
    res.status(400).json({ error: "invalid id" });
    return;
  }

  // SAFE: tagged template with sanitized input
  await prisma.$executeRaw`DELETE FROM "User" WHERE id = ${id}`;
  res.json({ deleted: true });
});
