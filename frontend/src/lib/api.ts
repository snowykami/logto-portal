export class ApiError extends Error {
  status: number;
  payload: unknown;

  constructor(status: number, payload: unknown) {
    super(`API request failed with ${status}`);
    this.status = status;
    this.payload = payload;
  }
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(path, {
    credentials: 'same-origin',
    headers: {
      Accept: 'application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...options.headers,
    },
    ...options,
  });

  const payload = await readPayload(response);
  if (response.status === 401) {
    const loginUrl = typeof payload === 'object' && payload && 'loginUrl' in payload
      ? String((payload as { loginUrl: unknown }).loginUrl)
      : '/auth/login';
    window.location.assign(loginUrl);
    throw new ApiError(response.status, payload);
  }
  if (!response.ok) {
    throw new ApiError(response.status, payload);
  }
  return payload as T;
}

async function readPayload(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) {
    return {};
  }
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}
