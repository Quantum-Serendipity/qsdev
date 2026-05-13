# agent-postmortem-skill — Example Postmortem Output
# Source: https://github.com/plus8bit/agent-postmortem-skill/blob/develop/examples/POSTMORTEM.example.md
# Retrieved: 2026-05-12

# Agent Postmortem Report

## Task

Implement a Next.js (TypeScript) API route for checkout (`/api/checkout/route.ts`) with zero-dollar cart handling, and verify it with a Jest test suite that initially failed due to a missing edge-case guard.

---

## Intent Snapshot

- **Objective:** Create a POST `/api/checkout/route.ts` handler that validates cart items, rejects zero-dollar totals, charges via a payment provider stub, and returns a structured response.
- **Deliverables:**
  - `src/app/api/checkout/route.ts` — Next.js App Router POST handler
  - `src/app/api/checkout/checkout.test.ts` — Jest test file covering happy path, empty cart, zero-dollar cart, and missing fields
- **Required checks:**
  - `npm run build` → exit_code 0
  - `npm test -- --testPathPattern=checkout` → exit_code 0, all assertions green
- **Out of scope:** Frontend cart UI, real Stripe integration, database persistence, auth middleware.

---

## Evidence Collection — Attempt 1 (FAILED)

### git status after initial implementation

```
$ git status --short
M  src/app/api/checkout/route.ts
A  src/app/api/checkout/checkout.test.ts
```

### git diff — src/app/api/checkout/route.ts (initial)

```diff
+++ b/src/app/api/checkout/route.ts
@@ -0,0 +1,54 @@
+import { NextRequest, NextResponse } from 'next/server';
+
+interface CartItem {
+  id: string;
+  name: string;
+  quantity: number;
+  unitPrice: number;
+}
+
+interface CheckoutPayload {
+  items: CartItem[];
+  currency: string;
+  customerId: string;
+}
+
+function calculateTotal(items: CartItem[]): number {
+  return items.reduce((sum, item) => sum + item.quantity * item.unitPrice, 0);
+}
+
+export async function POST(req: NextRequest) {
+  let body: CheckoutPayload;
+
+  try {
+    body = await req.json();
+  } catch {
+    return NextResponse.json({ error: 'Invalid JSON payload' }, { status: 400 });
+  }
+
+  const { items, currency, customerId } = body;
+
+  if (!customerId || typeof customerId !== 'string') {
+    return NextResponse.json({ error: 'Missing or invalid customerId' }, { status: 400 });
+  }
+
+  if (!currency || typeof currency !== 'string') {
+    return NextResponse.json({ error: 'Missing or invalid currency' }, { status: 400 });
+  }
+
+  if (!Array.isArray(items) || items.length === 0) {
+    return NextResponse.json({ error: 'Cart is empty' }, { status: 422 });
+  }
+
+  const total = calculateTotal(items);
+
+  // BUG: zero-dollar guard is absent here — total === 0 case falls through to charge
+
+  const chargeResult = await stubCharge({ customerId, total, currency });
+
+  return NextResponse.json({ success: true, chargeId: chargeResult.id, total }, { status: 200 });
+}
+
+async function stubCharge(params: { customerId: string; total: number; currency: string }) {
+  return { id: `ch_stub_${Date.now()}`, ...params };
+}
```

### git diff — src/app/api/checkout/checkout.test.ts

```diff
+++ b/src/app/api/checkout/checkout.test.ts
@@ -0,0 +1,89 @@
+import { POST } from './route';
+import { NextRequest } from 'next/server';
+
+function makeRequest(body: unknown): NextRequest {
+  return new NextRequest('http://localhost/api/checkout', {
+    method: 'POST',
+    body: JSON.stringify(body),
+    headers: { 'Content-Type': 'application/json' },
+  });
+}
+
+describe('POST /api/checkout', () => {
+  it('returns 200 and a chargeId for a valid cart', async () => {
+    const req = makeRequest({
+      customerId: 'cust_123',
+      currency: 'USD',
+      items: [{ id: 'prod_1', name: 'Widget', quantity: 2, unitPrice: 19.99 }],
+    });
+    const res = await POST(req);
+    const json = await res.json();
+    expect(res.status).toBe(200);
+    expect(json.success).toBe(true);
+    expect(json.chargeId).toMatch(/^ch_stub_/);
+    expect(json.total).toBeCloseTo(39.98);
+  });
+
+  it('returns 422 for an empty cart', async () => {
+    const req = makeRequest({ customerId: 'cust_123', currency: 'USD', items: [] });
+    const res = await POST(req);
+    expect(res.status).toBe(422);
+    const json = await res.json();
+    expect(json.error).toBe('Cart is empty');
+  });
+
+  it('returns 422 for a zero-dollar cart', async () => {
+    const req = makeRequest({
+      customerId: 'cust_123',
+      currency: 'USD',
+      items: [{ id: 'prod_free', name: 'Free Sample', quantity: 5, unitPrice: 0 }],
+    });
+    const res = await POST(req);
+    expect(res.status).toBe(422);                  // <-- THIS WILL FAIL: route returns 200
+    const json = await res.json();
+    expect(json.error).toBe('Cart total must be greater than zero');
+  });
+
+  it('returns 400 when customerId is missing', async () => {
+    const req = makeRequest({ currency: 'USD', items: [{ id: 'p1', name: 'X', quantity: 1, unitPrice: 10 }] });
+    const res = await POST(req);
+    expect(res.status).toBe(400);
+  });
+
+  it('returns 400 for malformed JSON', async () => {
+    const req = new NextRequest('http://localhost/api/checkout', {
+      method: 'POST',
+      body: '{ broken json',
+      headers: { 'Content-Type': 'application/json' },
+    });
+    const res = await POST(req);
+    expect(res.status).toBe(400);
+  });
+});
```

---

### command: npm run build (attempt 1)

- **exit_code: 0**
- **output_summary:** Build passed. TypeScript clean. No type errors.

---

### command: npm test -- --testPathPattern=checkout (attempt 1)

- **exit_code: 1**
- **output_summary:** 1 test failed. Zero-dollar cart falls through to `stubCharge` because the guard condition was never written. Route returns 200 instead of 422.

---

## Fix Applied

Added zero-dollar total guard to `src/app/api/checkout/route.ts` immediately after `calculateTotal`:

```diff
   const total = calculateTotal(items);

-  // BUG: zero-dollar guard is absent here — total === 0 case falls through to charge
+  if (total <= 0) {
+    return NextResponse.json(
+      { error: 'Cart total must be greater than zero' },
+      { status: 422 }
+    );
+  }

   const chargeResult = await stubCharge({ customerId, total, currency });
```

---

## Evidence Collection — Attempt 2 (POST-FIX)

### command: npm run build (attempt 2)
- **exit_code: 0**

### command: npm test -- --testPathPattern=checkout (attempt 2)
- **exit_code: 0**
- **output_summary:** All 5 tests passed. Zero-dollar guard confirmed working.

---

## Verification Check

- **Claim:** `route.ts` created and modified as requested. **Result: PASS**
- **Claim:** `checkout.test.ts` created with full edge-case coverage. **Result: PASS**
- **Claim:** `npm run build` succeeded. **Result: PASS**
- **Claim:** All tests pass. **Result: PASS**
- **Claim:** No contradictions between summary and evidence. **Result: PASS**

---

## Verdict

```
VERIFIED DONE
```

## Residual Risks

- `stubCharge` is a local stub — real payment provider integration untested.
- No concurrent request or race-condition testing.
- `currency` not validated against ISO-4217.
- No integration tests against running Next.js server.

## Next Actions

- Replace `stubCharge` with real payment provider SDK.
- Add `currency` enum validation.
- Add e2e smoke test in CI preview environment.
- Consider rate-limiting middleware on `/api/checkout`.
