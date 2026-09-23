// Call shapes the child_process sinks must cover beyond the single-argument
// form. Lines below a "ruleid:" comment must be reported by that rule; lines
// below an "ok:" comment must not be.
import express, { Request, Response } from "express";
import { exec, execSync, execFile } from "child_process";
import * as child_process from "child_process";

const app = express();

app.get("/ls", (req: Request, res: Response) => {
  // ruleid: qsdev.core.typescript.cmdi-child-process
  exec("ls " + req.query.dir, (err, stdout) => {
    res.send(stdout);
  });
});

app.get("/cat", (req: Request, res: Response) => {
  // ruleid: qsdev.core.typescript.cmdi-child-process
  const out = execSync(`cat ${req.query.file}`, { encoding: "utf8" });
  res.send(out);
});

app.get("/grep", (req: Request, res: Response) => {
  // ruleid: qsdev.core.typescript.cmdi-child-process
  child_process.exec("grep " + req.body.pattern, { timeout: 1000 }, (err, stdout) => {
    res.send(stdout);
  });
});

app.get("/safe", (req: Request, res: Response) => {
  // ok: qsdev.core.typescript.cmdi-child-process
  execFile("ls", [String(req.query.dir)], (err, stdout) => {
    res.send(stdout);
  });
});
