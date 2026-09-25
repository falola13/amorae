// Thrown by apiFetch for every non-2xx response and network failure.
// `message` is always safe to show a user as-is — never a raw backend error.
export class ApiError extends Error {
  readonly status: number;
  readonly code: string;
  readonly fields?: Record<string, string>;
  readonly requestId?: string;
  /** Seconds to wait, from a 429's `Retry-After`. */
  readonly retryAfter?: number;

  constructor(
    status: number,
    code: string,
    message: string,
    fields?: Record<string, string>,
    requestId?: string,
    retryAfter?: number,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.fields = fields;
    this.requestId = requestId;
    this.retryAfter = retryAfter;
  }
}

export function isApiError(error: unknown): error is ApiError {
  return error instanceof ApiError;
}
