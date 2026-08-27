/**
 * The calculator state machine.
 *
 * It is a pure function with no React and no network in it, so the whole of the app's
 * behaviour can be tested by calling it. The component layer only dispatches and renders.
 */

export type State = {
  /** What the screen shows. Kept as text so "1." and "1.0" survive being typed. */
  display: string;
  /** True while the user is typing the operand now on screen. */
  entering: boolean;
  /** The first operand of a binary operation, once it has been fixed. */
  pendingOperand: number | null;
  operation: string | null;
  /** How many operands the chosen operation takes; the API is the source of that. */
  operandCount: number;
  /** The failure code from the backend. Never its message: codes are stable, text is not. */
  errorCode: string | null;
  /** True while a calculation is in flight. */
  pending: boolean;
};

export type Action =
  | { type: "digit"; digit: string }
  | { type: "decimal" }
  | { type: "negate" }
  | { type: "operator"; operation: string; operandCount: number }
  | { type: "submit" }
  | { type: "result"; value: number }
  | { type: "failure"; code: string }
  | { type: "clear" };

export const initialState: State = {
  display: "0",
  entering: false,
  pendingOperand: null,
  operation: null,
  operandCount: 0,
  errorCode: null,
  pending: false,
};

export function reduce(state: State, action: Action): State {
  switch (action.type) {
    case "digit": {
      // Typing is also how a user dismisses an error: they correct and carry on.
      const display = state.entering && state.display !== "0" ? state.display + action.digit : action.digit;
      return { ...state, display, entering: true, errorCode: null };
    }

    case "decimal": {
      if (!state.entering) {
        return { ...state, display: "0.", entering: true, errorCode: null };
      }
      if (state.display.includes(".")) {
        return state;
      }
      return { ...state, display: state.display + "." };
    }

    case "negate": {
      const display = state.display.startsWith("-") ? state.display.slice(1) : "-" + state.display;
      return { ...state, display };
    }

    case "operator": {
      // A unary operation needs no first operand: what is on screen is its only one.
      if (action.operandCount === 1) {
        return {
          ...state,
          operation: action.operation,
          operandCount: 1,
          pendingOperand: null,
          entering: true,
          errorCode: null,
        };
      }
      // Choosing an operator fixes whatever is on screen as the first operand. Pressing a
      // second operator only changes the operation, it does not fix a new operand.
      const pendingOperand = state.entering || state.pendingOperand === null ? Number(state.display) : state.pendingOperand;
      return {
        ...state,
        operation: action.operation,
        operandCount: action.operandCount,
        pendingOperand,
        entering: false,
        errorCode: null,
      };
    }

    case "submit":
      return { ...state, pending: true };

    case "result":
      // The answer becomes the starting point of whatever the user does next.
      return {
        ...initialState,
        display: String(action.value),
      };

    case "failure":
      return { ...state, errorCode: action.code, pending: false };

    case "clear":
      return initialState;
  }
}

/** What a request needs: the operation and its operands. */
export type CalculationRequest = {
  operation: string;
  operands: number[];
};

/**
 * Builds the request the current state describes, or null when it does not describe one
 * yet. Keeping this out of the reducer lets the component ask "can I send?" without
 * having to dispatch anything to find out.
 */
export function requestFrom(state: State): CalculationRequest | null {
  if (state.operation === null) {
    return null;
  }
  if (state.operandCount === 1) {
    return { operation: state.operation, operands: [Number(state.display)] };
  }
  if (state.pendingOperand === null || !state.entering) {
    return null;
  }
  return { operation: state.operation, operands: [state.pendingOperand, Number(state.display)] };
}
