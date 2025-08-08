import { CoreClient } from "../client";
import * as Schema from "../schema";

export class ApiKeysService {
  constructor(private core: CoreClient) {}

  /**
   * GET /v1/api-keys
   */
  list(init?: Omit<RequestInit, "method" | "body">): Promise<Schema.ApiKeyListDto_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/api-keys`,
      ...(init || {}),
    });
  }

  /**
   * POST /v1/api-keys
   */
  create(body: Schema.ApiKeyCreateDto, init?: Omit<RequestInit, "method" | "body">): Promise<Schema.ApiKeyDto_Output> {
    return this.core.request({
      method: "POST",
      path: `/v1/api-keys`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }

  /**
   * DELETE /v1/api-keys/{id}
   */
  delete(id: string, init?: Omit<RequestInit, "method" | "body">): Promise<void> {
    return this.core.request({
      method: "DELETE",
      path: `/v1/api-keys/${encodeURIComponent(id)}`,
      ...(init || {}),
    });
  }

  /**
   * GET /v1/api-keys/{id}
   */
  retrieve(id: string, init?: Omit<RequestInit, "method" | "body">): Promise<Schema.ApiKeyDto_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/api-keys/${encodeURIComponent(id)}`,
      ...(init || {}),
    });
  }
}
