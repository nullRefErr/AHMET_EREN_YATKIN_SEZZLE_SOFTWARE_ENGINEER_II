import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { App } from "./App";

const operations = {
  operations: [
    { name: "add", operands: 2 },
    { name: "subtract", operands: 2 },
    { name: "multiply", operands: 2 },
    { name: "divide", operands: 2 },
    { name: "sqrt", operands: 1 },
  ],
};

type Reply = { status: number; body: unknown } | { throws: true };

/** Serves the operations list, then answers each calculation with the next reply. */
function stubBackend(...replies: Reply[]) {
  let next = 0;

  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string) => {
      if (url.endsWith("/operations")) {
        return new Response(JSON.stringify(operations), { status: 200 });
      }
      const reply = replies[next++] ?? { throws: true as const };
      if ("throws" in reply) {
        throw new TypeError("Failed to fetch");
      }
      return new Response(JSON.stringify(reply.body), { status: reply.status });
    }),
  );
}

/** Serves the operations list and actually performs the arithmetic it is asked for. */
function stubCalculator() {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (url: string, init?: RequestInit) => {
      if (url.endsWith("/operations")) {
        return new Response(JSON.stringify(operations), { status: 200 });
      }
      const body = JSON.parse(String(init?.body)) as { operation: string; operands: number[] };
      const a = body.operands[0] ?? 0;
      const b = body.operands[1] ?? 0;
      const results: Record<string, number> = {
        add: a + b,
        subtract: a - b,
        multiply: a * b,
        divide: a / b,
        sqrt: Math.sqrt(a),
      };
      return new Response(JSON.stringify({ ...body, result: results[body.operation] ?? 0, cached: false }), {
        status: 200,
      });
    }),
  );
}

function ok(result: number, cached = false) {
  return { status: 200, body: { operation: "add", operands: [1, 2], result, cached } };
}

function failure(code: string) {
  return { status: 422, body: { error: { code, message: "ignored on purpose" } } };
}

/** Renders the app and waits for the keypad the backend describes. */
async function renderApp() {
  render(<App />);
  await screen.findByRole("button", { name: "add" });
  return userEvent.setup();
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("the keypad", () => {
  it("offers the operations the backend reports, not a hardcoded list", async () => {
    stubBackend();
    await renderApp();

    expect(screen.getByRole("button", { name: "divide" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "sqrt" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "modulo" })).not.toBeInTheDocument();
  });

  it("shows the digits that were pressed", async () => {
    stubBackend();
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "4" }));
    await user.click(screen.getByRole("button", { name: "2" }));

    expect(screen.getByRole("status")).toHaveTextContent("42");
  });
});

describe("calculating", () => {
  it("shows the result", async () => {
    stubBackend(ok(5));
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "2" }));
    await user.click(screen.getByRole("button", { name: "add" }));
    await user.click(screen.getByRole("button", { name: "3" }));
    await user.click(screen.getByRole("button", { name: "equals" }));

    expect(await screen.findByRole("status")).toHaveTextContent("5");
  });

  it("says when a result was recalled rather than computed", async () => {
    stubBackend(ok(5, true));
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "2" }));
    await user.click(screen.getByRole("button", { name: "add" }));
    await user.click(screen.getByRole("button", { name: "3" }));
    await user.click(screen.getByRole("button", { name: "equals" }));

    expect(await screen.findByText(/cached/i)).toBeInTheDocument();
  });

  it("works from the keyboard as well as the keypad", async () => {
    stubBackend(ok(9));
    const user = await renderApp();

    await user.keyboard("6*3{Enter}");

    expect(await screen.findByRole("status")).toHaveTextContent("9");
  });
});

describe("failures", () => {
  // The backend message is deliberately unusable here: the app must read the code.
  it("explains a failure from its code, not from the message the server sent", async () => {
    stubBackend(failure("DIVISION_BY_ZERO"));
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "1" }));
    await user.click(screen.getByRole("button", { name: "divide" }));
    await user.click(screen.getByRole("button", { name: "0" }));
    await user.click(screen.getByRole("button", { name: "equals" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent(/divide by zero/i);
    expect(alert).not.toHaveTextContent("ignored on purpose");
  });

  it("offers a retry when the network fails, and recovers on the retry", async () => {
    stubBackend({ throws: true }, ok(5));
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "2" }));
    await user.click(screen.getByRole("button", { name: "add" }));
    await user.click(screen.getByRole("button", { name: "3" }));
    await user.click(screen.getByRole("button", { name: "equals" }));

    await user.click(await screen.findByRole("button", { name: /retry/i }));

    expect(await screen.findByRole("status")).toHaveTextContent("5");
  });

  it("clears the failure once the user types again", async () => {
    stubBackend(failure("DIVISION_BY_ZERO"));
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "1" }));
    await user.click(screen.getByRole("button", { name: "divide" }));
    await user.click(screen.getByRole("button", { name: "0" }));
    await user.click(screen.getByRole("button", { name: "equals" }));
    await screen.findByRole("alert");

    await user.click(screen.getByRole("button", { name: "7" }));

    await waitFor(() => expect(screen.queryByRole("alert")).not.toBeInTheDocument());
  });
});

describe("while a calculation is in flight", () => {
  it("disables the keypad so the same request is not sent twice", async () => {
    let release: () => void = () => {};
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });

    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string) => {
        if (url.endsWith("/operations")) {
          return new Response(JSON.stringify(operations), { status: 200 });
        }
        await held;
        return new Response(JSON.stringify(ok(5).body), { status: 200 });
      }),
    );

    const user = await renderApp();
    await user.click(screen.getByRole("button", { name: "2" }));
    await user.click(screen.getByRole("button", { name: "add" }));
    await user.click(screen.getByRole("button", { name: "3" }));
    await user.click(screen.getByRole("button", { name: "equals" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "equals" })).toBeDisabled());

    release();
    expect(await screen.findByRole("status")).toHaveTextContent("5");
  });
});

// Pressing an operator with a complete calculation on screen finishes it first, the way
// every physical calculator does. Without this, "75 + 52 - 30" silently drops the 75 + 52
// and answers 22.
describe("chaining operations", () => {
  it("computes the pending operation when the next operator is pressed", async () => {
    stubCalculator();
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "7" }));
    await user.click(screen.getByRole("button", { name: "5" }));
    await user.click(screen.getByRole("button", { name: "add" }));
    await user.click(screen.getByRole("button", { name: "5" }));
    await user.click(screen.getByRole("button", { name: "2" }));
    await user.click(screen.getByRole("button", { name: "subtract" }));

    expect(await screen.findByRole("status")).toHaveTextContent("127");

    await user.click(screen.getByRole("button", { name: "3" }));
    await user.click(screen.getByRole("button", { name: "0" }));
    await user.click(screen.getByRole("button", { name: "equals" }));

    expect(await screen.findByRole("status")).toHaveTextContent("97");
  });

  it("does not compute when an operator is only being changed", async () => {
    stubCalculator();
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "8" }));
    await user.click(screen.getByRole("button", { name: "add" }));
    await user.click(screen.getByRole("button", { name: "multiply" }));
    await user.click(screen.getByRole("button", { name: "3" }));
    await user.click(screen.getByRole("button", { name: "equals" }));

    expect(await screen.findByRole("status")).toHaveTextContent("24");
  });

  it("leaves a failure on screen instead of chaining past it", async () => {
    stubBackend(failure("DIVISION_BY_ZERO"));
    const user = await renderApp();

    await user.click(screen.getByRole("button", { name: "1" }));
    await user.click(screen.getByRole("button", { name: "divide" }));
    await user.click(screen.getByRole("button", { name: "0" }));
    await user.click(screen.getByRole("button", { name: "add" }));

    expect(await screen.findByRole("alert")).toBeInTheDocument();
  });
});
