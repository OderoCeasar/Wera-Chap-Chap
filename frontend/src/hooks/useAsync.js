import { useEffect, useState } from "react";

export function useAsync(fn, deps = []) {
  const [state, setState] = useState({ data: null, loading: true, error: null });
  useEffect(() => {
    let active = true;
    setState((current) => ({ ...current, loading: true }));
    fn().then((data) => active && setState({ data, loading: false, error: null })).catch((error) => active && setState({ data: null, loading: false, error }));
    return () => { active = false; };
  }, deps);
  return state;
}
