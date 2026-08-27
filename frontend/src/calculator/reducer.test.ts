import { describe, expect, it } from "vitest";

import { initialState, reduce, requestFrom, type State } from "./reducer";

/** Applies a sequence of actions, which is how the UI actually drives the reducer. */
function run(...actions: Parameters<typeof reduce>[1][]): State {
  return actions.reduce(reduce, initialState);
}

const digit = (digit: string) => ({ type: "digit", digit }) as const;
const binary = (operation: string) => ({ type: "operator", operation, operandCount: 2 }) as const;
const unary = (operation: string) => ({ type: "operator", operation, operandCount: 1 }) as const;

describe("entering numbers", () => {
  it("shows a digit that was pressed", () => {
    expect(run(digit("7")).display).toBe("7");
  });

  it("builds a longer number from successive digits", () => {
    expect(run(digit("1"), digit("2"), digit("3")).display).toBe("123");
  });

  it("replaces the leading zero rather than growing it", () => {
    expect(run(digit("0"), digit("5")).display).toBe("5");
  });

  it("adds a decimal point", () => {
    expect(run(digit("1"), { type: "decimal" }, digit("5")).display).toBe("1.5");
  });

  it("refuses a second decimal point", () => {
    expect(run(digit("1"), { type: "decimal" }, { type: "decimal" }, digit("5")).display).toBe("1.5");
  });

  it("starts a decimal with a leading zero", () => {
    expect(run({ type: "decimal" }, digit("5")).display).toBe("0.5");
  });
});

describe("choosing an operation", () => {
  it("keeps the first operand and starts the second", () => {
    const state = run(digit("1"), digit("2"), binary("add"), digit("3"));

    expect(state.pendingOperand).toBe(12);
    expect(state.operation).toBe("add");
    expect(state.display).toBe("3");
  });

  it("replaces an operation chosen twice in a row", () => {
    const state = run(digit("8"), binary("add"), binary("multiply"));

    expect(state.operation).toBe("multiply");
    expect(state.pendingOperand).toBe(8);
  });
});

describe("building the request", () => {
  it("has nothing to send before an operation is chosen", () => {
    expect(requestFrom(run(digit("5")))).toBeNull();
  });

  it("has nothing to send before the second operand is entered", () => {
    expect(requestFrom(run(digit("5"), binary("add")))).toBeNull();
  });

  it("sends both operands of a binary operation", () => {
    const state = run(digit("1"), digit("0"), binary("divide"), digit("4"));

    expect(requestFrom(state)).toEqual({ operation: "divide", operands: [10, 4] });
  });

  // Square root takes one operand, so it is ready as soon as it is chosen.
  it("sends the single operand of a unary operation", () => {
    expect(requestFrom(run(digit("9"), unary("sqrt")))).toEqual({ operation: "sqrt", operands: [9] });
  });

  it("sends zero as an operand", () => {
    const state = run(digit("0"), binary("add"), digit("0"));

    expect(requestFrom(state)).toEqual({ operation: "add", operands: [0, 0] });
  });

  it("sends a negative operand", () => {
    const state = run(digit("5"), { type: "negate" }, binary("add"), digit("3"));

    expect(requestFrom(state)).toEqual({ operation: "add", operands: [-5, 3] });
  });
});

describe("results and failures", () => {
  it("shows the result and clears the pending operation", () => {
    const state = run(digit("2"), binary("add"), digit("3"), { type: "result", value: 5 });

    expect(state.display).toBe("5");
    expect(state.operation).toBeNull();
    expect(state.pendingOperand).toBeNull();
    expect(state.pending).toBe(false);
  });

  it("continues from a result as the next first operand", () => {
    const state = run(
      digit("2"),
      binary("add"),
      digit("3"),
      { type: "result", value: 5 },
      binary("multiply"),
      digit("2"),
    );

    expect(requestFrom(state)).toEqual({ operation: "multiply", operands: [5, 2] });
  });

  it("records the failure code, not the message", () => {
    const state = run(digit("1"), binary("divide"), digit("0"), { type: "failure", code: "DIVISION_BY_ZERO" });

    expect(state.errorCode).toBe("DIVISION_BY_ZERO");
    expect(state.pending).toBe(false);
  });

  it("clears the failure as soon as the next digit is pressed", () => {
    const state = run({ type: "failure", code: "DIVISION_BY_ZERO" }, digit("7"));

    expect(state.errorCode).toBeNull();
    expect(state.display).toBe("7");
  });

  it("marks a submitted calculation as pending", () => {
    expect(run(digit("2"), binary("add"), digit("3"), { type: "submit" }).pending).toBe(true);
  });

  it("returns to the start when cleared", () => {
    const state = run(digit("2"), binary("add"), digit("3"), { type: "clear" });

    expect(state).toEqual(initialState);
  });
});
