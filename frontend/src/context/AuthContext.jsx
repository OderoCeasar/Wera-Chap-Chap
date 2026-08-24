import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { endpoints, tokenStore } from "../api/client";

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(() => tokenStore.user());
  // Start "loading" only when a stored session needs revalidating, so guests
  // never wait on a network round trip.
  const [loading, setLoading] = useState(() => Boolean(tokenStore.access()));

  const applySession = useCallback((data) => {
    tokenStore.save(data);
    setUser(data.user);
    return data.user;
  }, []);

  const logout = useCallback(() => {
    endpoints.logout().catch(() => {});
    tokenStore.clear();
    setUser(null);
  }, []);

  // Revalidate a stored session on boot: the cached user may be stale or the
  // refresh token may have expired while the tab was closed.
  useEffect(() => {
    if (!tokenStore.access()) {
      setLoading(false);
      return;
    }
    let active = true;
    endpoints
      .me()
      .then(({ data }) => {
        if (!active) return;
        tokenStore.save({ user: data });
        setUser(data);
      })
      .catch(() => {
        if (!active) return;
        tokenStore.clear();
        setUser(null);
      })
      .finally(() => active && setLoading(false));
    return () => { active = false; };
  }, []);

  // The api client raises this when a refresh attempt fails for good.
  useEffect(() => {
    const onUnauthorized = () => setUser(null);
    window.addEventListener("wera:unauthorized", onUnauthorized);
    return () => window.removeEventListener("wera:unauthorized", onUnauthorized);
  }, []);

  const login = useCallback(
    async (payload) => applySession((await endpoints.login(payload)).data),
    [applySession]
  );

  const register = useCallback(
    async (payload) => applySession((await endpoints.register(payload)).data),
    [applySession]
  );

  const refreshUser = useCallback(async () => {
    const { data } = await endpoints.me();
    tokenStore.save({ user: data });
    setUser(data);
    return data;
  }, []);

  const value = useMemo(
    () => ({ user, loading, login, register, logout, refreshUser }),
    [user, loading, login, register, logout, refreshUser]
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (!context) throw new Error("useAuth must be used inside AuthProvider");
  return context;
}
