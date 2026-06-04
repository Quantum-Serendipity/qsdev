import express, { Request, Response } from "express";
import { PrismaClient } from "@prisma/client";

const app = express();
const prisma = new PrismaClient();

app.get("/users/search", async (req: Request, res: Response) => {
  const name = req.query.name as string;

  // VULNERABLE: $queryRawUnsafe with string interpolation
  const users = await prisma.$queryRawUnsafe(
    `SELECT * FROM "User" WHERE name = '${name}'`
  );
  res.json(users);
});

app.post("/users/delete", async (req: Request, res: Response) => {
  const id = req.body.id;

  // VULNERABLE: $executeRawUnsafe with concatenation
  await prisma.$executeRawUnsafe("DELETE FROM \"User\" WHERE id = " + id);
  res.json({ deleted: true });
});
