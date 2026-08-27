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

  const submit = useCallback(async () => {
    const request = requestFrom(state);
    if (request === null || state.pending) {
      return;
    }

    dispatch({ type: "submit" });
    try {
      const calculation = await calculate(request);
      setCached(calculation.cached);
      dispatch({ type: "result", value: calculation.result });
    } catch (error) {
      dispatch({
        type: "failure",
        code: error instanceof ApiError ? error.code : "INTERNAL_ERROR",
      });
    }
  }, [state]);

  return { state, operations, cached, dispatch, submit };
}
