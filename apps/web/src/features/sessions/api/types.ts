// ============================================================
// Online session types — aligned with castor backend dto.SessionResp
// ============================================================

/** A live authorization session: neither revoked nor past its maximum lifetime. */
export interface Session {
  id: string;
  userId: number;
  username: string;
  ipAddr: string;
  userAgent: string;
  /** Signed in with "remember me": the session survives browser restarts. */
  remember: boolean;
  createdAt: string;
  /** Last token refresh, i.e. roughly when the session was last in use. */
  lastActiveAt: string;
  expiresAt: string;
  /** The caller's own session. */
  current: boolean;
}

export interface SessionFilters {
  page?: number;
  pageSize?: number;
  username?: string;
  ipAddr?: string;
  sort?: string;
}

export interface SessionsPageResult {
  list: Session[];
  total: number;
  page: number;
  pageSize: number;
  totalPages: number;
}
