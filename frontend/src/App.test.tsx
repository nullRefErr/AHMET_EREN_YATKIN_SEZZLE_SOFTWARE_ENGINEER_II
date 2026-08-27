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
