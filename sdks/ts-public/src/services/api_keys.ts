import { CoreClient } from "../client";
import * as Schema from "../schema";

export class ApiKeysService {
  constructor(private core: CoreClient) {}

  /**
   * GET /v1/api-keys
   * @returns List API keys
   */
  list(
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.ApiKeyListDto_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/api-keys`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for list
   * @returns ['v1/api-keys']
   */
  list__queryKeys(
  ) {
    return ['v1/api-keys'] as const;
  }

  /**
   * POST /v1/api-keys
   * @returns Create API key
   */
  create(
    body: Schema.ApiKeyCreateDto,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.ApiKeyDto_Output> {
    return this.core.request({
      method: "POST",
      path: `/v1/api-keys`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for create
   * @returns ['v1/api-keys', body]
   */
  create__queryKeys(
    body: Schema.ApiKeyCreateDto
  ) {
    return ['v1/api-keys', body] as const;
  }

  /**
   * DELETE /v1/api-keys/{id}
   * @returns Delete API key
   */
  delete(
    id: string,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<unknown> {
    return this.core.request({
      method: "DELETE",
      path: `/v1/api-keys/${encodeURIComponent(id)}`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for delete
   * @returns ['v1/api-keys', id]
   */
  delete__queryKeys(
    id: string
  ) {
    return ['v1/api-keys', id] as const;
  }

  /**
   * GET /v1/api-keys/{id}
   * @returns Get API key
   */
  get(
    id: string,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.ApiKeyDto_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/api-keys/${encodeURIComponent(id)}`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for get
   * @returns ['v1/api-keys', id]
   */
  get__queryKeys(
    id: string
  ) {
    return ['v1/api-keys', id] as const;
  }
}
