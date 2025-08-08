import { CoreClient } from "../client";
import * as Schema from "../schema";

export class ResourceTypeService {
  constructor(private core: CoreClient) {}

  /**
   * GET /v1/resource-types
   */
  list(query?: {parentResourceTypeId?: string, search?: string, status?: "active" | "archived"}, init?: Omit<RequestInit, "method" | "body">): Promise<Schema.ResourceTypeList_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/resource-types`,
      query,
      ...(init || {}),
    });
  }

  /**
   * POST /v1/resource-types
   */
  create(body: Schema.ResourceTypeCreateBody, init?: Omit<RequestInit, "method" | "body">): Promise<Schema.ResourceType_Output> {
    return this.core.request({
      method: "POST",
      path: `/v1/resource-types`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }

  /**
   * DELETE /v1/resource-types/{resourceTypeId}
   */
  delete(resourceTypeId: string, init?: Omit<RequestInit, "method" | "body">): Promise<void> {
    return this.core.request({
      method: "DELETE",
      path: `/v1/resource-types/${encodeURIComponent(resourceTypeId)}`,
      ...(init || {}),
    });
  }

  /**
   * GET /v1/resource-types/{resourceTypeId}
   */
  retrieve(resourceTypeId: string, init?: Omit<RequestInit, "method" | "body">): Promise<Schema.ResourceType_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/resource-types/${encodeURIComponent(resourceTypeId)}`,
      ...(init || {}),
    });
  }

  /**
   * PUT /v1/resource-types/{resourceTypeId}
   */
  update(resourceTypeId: string, body: Schema.ResourceTypeCreateBody, init?: Omit<RequestInit, "method" | "body">): Promise<void> {
    return this.core.request({
      method: "PUT",
      path: `/v1/resource-types/${encodeURIComponent(resourceTypeId)}`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }
}
