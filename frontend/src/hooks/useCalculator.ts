import { useCallback, useEffect, useReducer, useState } from "react";

import { ApiError, calculate, listOperations, type OperationInfo } from "../api/client";
import { initialState, reduce, requestFrom } from "../calculator/reducer";

/**
 * Connects the calculator state machine to the API.
 *
 * The reducer holds every rule about what the calculator does; this hook only decides
 * when to talk to the server and what to do with the answer.
 */
export function useCalculator() {
  const [state, dispatch] = useReducer(reduce, initialState);
  const [operations, setOperations] = useState<OperationInfo[]>([]);
  const [cached, setCached] = useState(false);

  // The keypad is described by the backend, so a new operation appears without a
  // frontend release and the two can never disagree about what is supported.
  useEffect(() => {
    let active = true;

    listOperations()
      .then((available) => {
        if (active) {
          setOperations(available);
        }
      })
      .catch(() => {
        if (active) {
          setOperations([]);
        }
      });

    return () => {
      active = false;
    };
  }, []);

  /** Sends the calculation on screen, and reports whether it produced a result. */
  const submit = useCallback(async (): Promise<boolean> => {
    const request = requestFrom(state);
    if (request === null || state.pending) {
      return false;
    }

    dispatch({ type: "submit" });
    try {
      const calculation = await calculate(request);
      setCached(calculation.cached);
      dispatch({ type: "result", value: calculation.result });
      return true;
    } catch (error) {
      dispatch({
        type: "failure",
        code: error instanceof ApiError ? error.code : "INTERNAL_ERROR",
      });
      return false;
    }
  }, [state]);

  /**
   * Chooses the next operation, finishing whatever is already on screen first.
   *
   * Pressing an operator with a complete calculation showing means "give me that answer
   * and carry on from it" — that is what every physical calculator does, and without it
   * "75 + 52 - 30" would quietly discard the 75 + 52 and answer 22.
   */
  const chooseOperation = useCallback(
    async (operation: string, operandCount: number) => {
      if (state.entering && requestFrom(state) !== null) {
        const computed = await submit();
        if (!computed) {
          // Leave the failure on screen rather than letting the operator clear it.
          return;
        }
      }
      dispatch({ type: "operator", operation, operandCount });
    },
    [state, submit],
  );

  return { state, operations, cached, dispatch, submit, chooseOperation };
}
