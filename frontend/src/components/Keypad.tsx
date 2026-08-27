import type { OperationInfo } from "../api/client";
import type { Action } from "../calculator/reducer";

/** The symbol each operation is drawn with. Its accessible name stays the API's name. */
const SYMBOLS: Record<string, string> = {
  add: "+",
  subtract: "−",
  multiply: "×",
  divide: "÷",
  power: "xʸ",
  sqrt: "√",
  percentage: "%",
};

const DIGITS = ["7", "8", "9", "4", "5", "6", "1", "2", "3", "0"];

type KeypadProps = {
  operations: OperationInfo[];
  disabled: boolean;
  dispatch: (action: Action) => void;
  onSubmit: () => void;
};

export function Keypad({ operations, disabled, dispatch, onSubmit }: KeypadProps) {
  return (
    <div className="keypad">
      <div className="keypad__digits">
        {DIGITS.map((digit) => (
          <button
            key={digit}
            type="button"
            className="key"
            aria-label={digit}
            disabled={disabled}
            onClick={() => dispatch({ type: "digit", digit })}
          >
            {digit}
          </button>
        ))}
        <button
          type="button"
          className="key"
          aria-label="decimal point"
          disabled={disabled}
          onClick={() => dispatch({ type: "decimal" })}
        >
          .
        </button>
        <button
          type="button"
          className="key"
          aria-label="negate"
          disabled={disabled}
          onClick={() => dispatch({ type: "negate" })}
        >
          ±
        </button>
      </div>

      <div className="keypad__operations">
        {operations.map((operation) => (
          <button
            key={operation.name}
            type="button"
            className="key key--operation"
            aria-label={operation.name}
            disabled={disabled}
            onClick={() =>
              dispatch({
                type: "operator",
                operation: operation.name,
                operandCount: operation.operands,
              })
            }
          >
            {SYMBOLS[operation.name] ?? operation.name}
          </button>
        ))}
      </div>

      <div className="keypad__actions">
        <button
          type="button"
          className="key key--clear"
          aria-label="clear"
          disabled={disabled}
          onClick={() => dispatch({ type: "clear" })}
        >
          C
        </button>
        <button
          type="button"
          className="key key--equals"
          aria-label="equals"
          disabled={disabled}
          onClick={onSubmit}
        >
          =
        </button>
      </div>
    </div>
  );
}
