/** The calculator API client. Nothing else in the app knows the wire format. */

import type { CalculationRequest } from "../calculator/reducer";

const BASE_URL = "/api/v1";

export type OperationInfo = { name: string; operands: number };

export type Calculation = {
  operation: string;
  operands: number[];
  result: number;
  cached: boolean;
};

/**
 * A failure carrying the backend's stable error code.
 *
 * The code is what the interface reacts to. The server's message is for logs and for
 * developers; it is free to change wording, so branching on it would be brittle and it
 * could never be translated.
 */
export class ApiError extends Error {
  readonly code: string;

  constructor(code: string) {
    super(code);
    this.name = "ApiError";
    this.code = code;
  }
}

export async function listOperations(): Promise<OperationInfo[]> {
  const body = await request(`${BASE_URL}/operations`);
  if (!isOperationList(body)) {
    throw new ApiError("INTERNAL_ERROR");
  }
  return body.operations;
}

export async function calculate(calculation: CalculationRequest): Promise<Calculation> {
  const body = await request(`${BASE_URL}/calculations`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(calculation),
  });
  if (!isCalculation(body)) {
    throw new ApiError("INTERNAL_ERROR");
  }
  return body;
}

async function request(url: string, init?: RequestInit): Promise<unknown> {
  let response: Response;
  try {
    response = await fetch(url, init);
  } catch {
    // fetch only rejects when the request never completed: offline, DNS, CORS.
    throw new ApiError("NETWORK_ERROR");
  }

  const body: unknown = await response.json().catch(() => null);
  if (!response.ok) {
    throw new ApiError(errorCodeOf(body));
  }
  return body;
}

function errorCodeOf(body: unknown): string {
  if (typeof body === "object" && body !== null && "error" in body) {
    const { error } = body as { error: unknown };
    if (typeof error === "object" && error !== null && "code" in error) {
      const { code } = error as { code: unknown };
      if (typeof code === "string") {
        return code;
      }
    }
  }
  return "INTERNAL_ERROR";
}

// TypeScript types are erased at runtime, so anything that crosses the network is checked
// before it is trusted. A proxy returning an HTML error page must not become a NaN on
// screen.
function isCalculation(body: unknown): body is Calculation {
  if (typeof body !== "object" || body === null) {
    return false;
  }
  const candidate = body as Partial<Calculation>;
  return (
    typeof candidate.operation === "string" &&
    Array.isArray(candidate.operands) &&
    typeof candidate.result === "number" &&
    typeof candidate.cached === "boolean"
  );
}

function isOperationList(body: unknown): body is { operations: OperationInfo[] } {
  if (typeof body !== "object" || body === null || !("operations" in body)) {
    return false;
  }
  const { operations } = body as { operations: unknown };
  return (
    Array.isArray(operations) &&
    operations.every(
      (operation) =>
        typeof operation === "object" &&
        operation !== null &&
        typeof (operation as OperationInfo).name === "string" &&
        typeof (operation as OperationInfo).operands === "number",
    )
  );
}
