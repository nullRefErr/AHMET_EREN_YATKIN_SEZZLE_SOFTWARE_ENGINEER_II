import { useEffect } from "react";

import { Display } from "./components/Display";
import { ErrorBanner } from "./components/ErrorBanner";
import { Keypad } from "./components/Keypad";
import { useCalculator } from "./hooks/useCalculator";

/** Which operation each keyboard symbol stands for. */
const KEY_OPERATIONS: Record<string, string> = {
  "+": "add",
  "-": "subtract",
  "*": "multiply",
  "/": "divide",
  "^": "power",
  "%": "percentage",
};

export function App() {
  const { state, operations, cached, dispatch, submit, chooseOperation } = useCalculator();

  useEffect(() => {
    function handleKey(event: KeyboardEvent) {
      const { key } = event;

      if (key >= "0" && key <= "9") {
        dispatch({ type: "digit", digit: key });
        return;
      }
      if (key === ".") {
        dispatch({ type: "decimal" });
        return;
      }
      if (key === "Enter" || key === "=") {
        event.preventDefault();
        void submit();
        return;
      }
      if (key === "Escape") {
        dispatch({ type: "clear" });
        return;
      }

      const name = KEY_OPERATIONS[key];
      const operation = operations.find((candidate) => candidate.name === name);
      if (operation) {
        void chooseOperation(operation.name, operation.operands);
      }
    }

    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [chooseOperation, dispatch, operations, submit]);

  // The badge belongs to the value on screen, so it disappears the moment that value is
  // no longer a result.
  const showCached = cached && !state.entering && state.operation === null;

  return (
    <main className="app">
      <h1 className="app__title">Calculator</h1>

      <Display value={state.display} cached={showCached} />

      {state.errorCode && <ErrorBanner code={state.errorCode} onRetry={() => void submit()} />}

      <Keypad
        operations={operations}
        disabled={state.pending}
        dispatch={dispatch}
        onOperation={(operation, operandCount) => void chooseOperation(operation, operandCount)}
        onSubmit={() => void submit()}
      />
    </main>
  );
}
