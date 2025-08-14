import { CoreClient } from "../client";
import * as Schema from "../schema";

export class OrganizationsService {
  constructor(private core: CoreClient) {}

  /**
   * GET /v1/organizations
   * @returns List organizations
   */
  list(
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.OrganizationListDto_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/organizations`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for list
   * @returns ['v1/organizations']
   */
  list__queryKeys(
  ) {
    return ['v1/organizations'] as const;
  }

  /**
   * POST /v1/organizations
   * @summary Create organization
   */
  create(
    body: Schema.OrganizationCreateDto,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.OrganizationDto_Output> {
    return this.core.request({
      method: "POST",
      path: `/v1/organizations`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for create
   * @returns ['v1/organizations', body]
   */
  create__queryKeys(
    body: Schema.OrganizationCreateDto
  ) {
    return ['v1/organizations', body] as const;
  }

  /**
   * DELETE /v1/organizations/{id}
   * @returns Delete organization
   */
  delete(
    id: string,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<unknown> {
    return this.core.request({
      method: "DELETE",
      path: `/v1/organizations/${encodeURIComponent(id)}`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for delete
   * @returns ['v1/organizations', id]
   */
  delete__queryKeys(
    id: string
  ) {
    return ['v1/organizations', id] as const;
  }

  /**
   * GET /v1/organizations/{id}
   * @returns Get organization
   */
  get(
    id: string,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.OrganizationDto_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/organizations/${encodeURIComponent(id)}`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for get
   * @returns ['v1/organizations', id]
   */
  get__queryKeys(
    id: string
  ) {
    return ['v1/organizations', id] as const;
  }

  /**
   * PUT /v1/organizations/{id}
   * @returns Update organization
   */
  update(
    id: string,
    body: Schema.OrganizationUpdateDto,
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<Schema.OrganizationDto_Output> {
    return this.core.request({
      method: "PUT",
      path: `/v1/organizations/${encodeURIComponent(id)}`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for update
   * @returns ['v1/organizations', id, body]
   */
  update__queryKeys(
    id: string,
    body: Schema.OrganizationUpdateDto
  ) {
    return ['v1/organizations', id, body] as const;
  }
}
