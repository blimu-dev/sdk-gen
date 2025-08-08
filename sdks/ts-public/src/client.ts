export type ClientOption = {
  baseURL?: string;
  headers?: Record<string, string>;
  apikey?: string;
  bearer?: string;
  fetch?: typeof fetch;
};

export class CoreClient {
  public cfg: ClientOption;

  constructor(cfg: ClientOption = {}) {
    this.cfg = typeofcfg;
  }
  async request(
    init: RequestInit & {
      path: string;
      method: string;
      query?: Record<string, any>;
    }
  ) {
    const url = new URL((this.cfg.baseURL || "") + init.path);
    if (init.query) {
      Object.entries(init.query).forEach(([k, v]) => {
        if (v === undefined || v === null) return;
        if (Array.isArray(v))
          v.forEach((vv) => url.searchParams.append(k, String(vv)));
        else url.searchParams.set(k, String(v));
      });
    }
    const headers = new Headers({
      ...(this.cfg.headers || {}),
      ...(init.headers as any),
    });
    if (this.cfg?.apikey)
      headers.set("Authorization", String(this.cfg?.apikey));
    if (this.cfg.bearer)
      headers.set("Authorization", `Bearer ${this.cfg.bearer}`);
    const res = await (this.cfg.fetch || fetch)(url.toString(), {
      ...init,
      headers,
    });
    const ct = res.headers.get("content-type") || "";
    const body = ct.includes("application/json")
      ? await res.json()
      : await res.text();
    if (!res.ok)
      throw Object.assign(new Error(`HTTP ${res.status}`), {
        status: res.status,
        body,
      });
    return body as any;
  }
}
