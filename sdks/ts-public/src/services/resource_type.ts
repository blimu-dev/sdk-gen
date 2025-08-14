import { CoreClient } from "../client";
import * as Schema from "../schema";

export class ResourceTypeService {
  constructor(private core: CoreClient) {}

  /**
   * GET /v1/resource-types
   * @returns List of resource types
   */
  list(
    query?: Schema.ResourceTypeListQuery,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.ResourceTypeList_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/resource-types`,
      query,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for list
   * @returns ['v1/resource-types', query]
   */
  list__queryKeys(
    query?: Schema.ResourceTypeListQuery
  ) {
    return ['v1/resource-types', query] as const;
  }

  /**
   * POST /v1/resource-types
   * @returns Resource type created
   */
  create(
    body: Schema.ResourceTypeCreateBody,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.ResourceType_Output> {
    return this.core.request({
      method: "POST",
      path: `/v1/resource-types`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for create
   * @returns ['v1/resource-types', body]
   */
  create__queryKeys(
    body: Schema.ResourceTypeCreateBody
  ) {
    return ['v1/resource-types', body] as const;
  }

  /**
   * DELETE /v1/resource-types/{resourceTypeId}
   */
  delete(
    resourceTypeId: string,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<unknown> {
    return this.core.request({
      method: "DELETE",
      path: `/v1/resource-types/${encodeURIComponent(resourceTypeId)}`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for delete
   * @returns ['v1/resource-types', resourceTypeId]
   */
  delete__queryKeys(
    resourceTypeId: string
  ) {
    return ['v1/resource-types', resourceTypeId] as const;
  }

  /**
   * GET /v1/resource-types/{resourceTypeId}
   * @returns Resource type
   */
  get(
    resourceTypeId: string,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.ResourceType_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/resource-types/${encodeURIComponent(resourceTypeId)}`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for get
   * @returns ['v1/resource-types', resourceTypeId]
   */
  get__queryKeys(
    resourceTypeId: string
  ) {
    return ['v1/resource-types', resourceTypeId] as const;
  }

  /**
   * PUT /v1/resource-types/{resourceTypeId}
   */
  update(
    resourceTypeId: string,
    body: Schema.ResourceTypeCreateBody,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<unknown> {
    return this.core.request({
      method: "PUT",
      path: `/v1/resource-types/${encodeURIComponent(resourceTypeId)}`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for update
   * @returns ['v1/resource-types', resourceTypeId, body]
   */
  update__queryKeys(
    resourceTypeId: string,
    body: Schema.ResourceTypeCreateBody
  ) {
    return ['v1/resource-types', resourceTypeId, body] as const;
  }
}
