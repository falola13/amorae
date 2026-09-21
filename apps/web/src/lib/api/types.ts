// Mirrors the Go API's JSON contract exactly (snake_case field names) —
// see apps/api. Do not rename fields to camelCase here; that would require
// a translation layer for no benefit, since these types never leak past the
// server boundary (see lib/api/client.ts).
export interface User {
  id: string;
  email: string;
  display_name: string;
  created_at: string;
  updated_at: string;
}

export interface AuthResult {
  token: string;
  expires_at: string;
  user: User;
}
