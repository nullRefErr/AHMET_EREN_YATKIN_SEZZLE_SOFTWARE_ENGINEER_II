/**
 * Every failure the interface can show, keyed by the backend's error code.
 *
 * Wording lives here, on the client, so it can be reworded or translated without the API
 * changing. The server sends a code; this is where a code becomes a sentence.
 */
const MESSAGES: Record<string, string> = {
  DIVISION_BY_ZERO: "You cannot divide by zero.",
  NEGATIVE_SQRT: "A negative number has no real square root.",
  NUMERIC_OVERFLOW: "That result is too large to represent.",
  UNDEFINED_RESULT: "That calculation has no defined result.",
  INVALID_OPERAND_COUNT: "That operation needs a different number of values.",
  INVALID_REQUEST: "That request could not be understood.",
  NETWORK_ERROR: "Could not reach the calculator.",
  INTERNAL_ERROR: "Something went wrong. Please try again.",
};

/** Only a failure that might succeed next time is worth a retry button. */
const RETRYABLE = new Set(["NETWORK_ERROR", "INTERNAL_ERROR"]);

type ErrorBannerProps = {
  code: string;
  onRetry: () => void;
};

export function ErrorBanner({ code, onRetry }: ErrorBannerProps) {
  return (
    <div className="error" role="alert">
      <span>{MESSAGES[code] ?? MESSAGES.INTERNAL_ERROR}</span>
      {RETRYABLE.has(code) && (
        <button type="button" className="error__retry" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  );
}
