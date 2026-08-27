type DisplayProps = {
  value: string;
  /** Whether the value on screen was recalled from the server's cache. */
  cached: boolean;
};

export function Display({ value, cached }: DisplayProps) {
  return (
    <div className="display">
      {/* role="status" with a polite live region: a screen reader announces the new
          value without interrupting whatever it is already saying. */}
      <output className="display__value" role="status" aria-live="polite">
        {value}
      </output>
      {cached && <span className="display__badge">cached</span>}
    </div>
  );
}
