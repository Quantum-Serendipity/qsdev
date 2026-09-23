// SvelteKit form actions (+page.server.ts), the primary SvelteKit mutation
// entry point. Lines below a "ruleid:" comment must be reported by that rule;
// lines below an "ok:" comment must not be.
import { redirect } from "@sveltejs/kit";
import { exec } from "child_process";
import * as fs from "fs";
import { prisma } from "$lib/server/db";

export const actions = {
  run: async ({ request }) => {
    const data = await request.formData();
    // ruleid: qsdev.core.typescript.cmdi-sveltekit
    exec(data.get("cmd"));
  },
  go: async ({ request }) => {
    const data = await request.formData();
    // ruleid: qsdev.core.typescript.redirect-sveltekit-open-redirect
    throw redirect(303, data.get("next"));
  },
  fetchIt: async ({ request }) => {
    const data = await request.formData();
    // ruleid: qsdev.core.typescript.ssrf-sveltekit-form-action
    return fetch(data.get("target"));
  },
  header: async ({ request, setHeaders }) => {
    const data = await request.formData();
    // ruleid: qsdev.core.typescript.header-injection-sveltekit
    setHeaders({ "x-note": data.get("note") });
  },
  read: async ({ request }) => {
    const data = await request.formData();
    // ruleid: qsdev.core.typescript.path-traversal-sveltekit
    return fs.readFileSync(data.get("file"));
  },
  search: async ({ request }) => {
    const data = await request.formData();
    // ruleid: qsdev.core.typescript.sqli-sveltekit-form-action
    return prisma.$queryRawUnsafe("SELECT * FROM users WHERE name = '" + data.get("name") + "'");
  },
  mine: async ({ cookies }) => {
    const session = cookies.get("session");
    // ruleid: qsdev.core.typescript.sqli-sveltekit-cookie
    return prisma.$queryRawUnsafe("SELECT * FROM sessions WHERE id = '" + session + "'");
  },
};
