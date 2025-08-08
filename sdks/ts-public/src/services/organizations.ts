import { CoreClient } from "../client";
import * as Schema from "../schema";

export class OrganizationsService {
  constructor(private core: CoreClient) {}

  /**
   * GET /v1/organizations
   */
  list(init?: Omit<RequestInit, "method" | "body">): Promise<Schema.OrganizationListDto_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/organizations`,
      ...(init || {}),
    });
  }

  /**
   * POST /v1/organizations
   */
  create(body: Schema.OrganizationCreateDto, init?: Omit<RequestInit, "method" | "body">): Promise<Schema.OrganizationDto_Output> {
    return this.core.request({
      method: "POST",
      path: `/v1/organizations`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }

  /**
   * DELETE /v1/organizations/{id}
   */
  delete(id: string, init?: Omit<RequestInit, "method" | "body">): Promise<void> {
    return this.core.request({
      method: "DELETE",
      path: `/v1/organizations/${encodeURIComponent(id)}`,
      ...(init || {}),
    });
  }

  /**
   * GET /v1/organizations/{id}
   */
  retrieve(id: string, init?: Omit<RequestInit, "method" | "body">): Promise<Schema.OrganizationDto_Output> {
    return this.core.request({
      method: "GET",
      path: `/v1/organizations/${encodeURIComponent(id)}`,
      ...(init || {}),
    });
  }

  /**
   * PUT /v1/organizations/{id}
   */
  update(id: string, body: Schema.OrganizationUpdateDto, init?: Omit<RequestInit, "method" | "body">): Promise<Schema.OrganizationDto_Output> {
    return this.core.request({
      method: "PUT",
      path: `/v1/organizations/${encodeURIComponent(id)}`,
      headers: { ...(init?.headers || {}), "content-type": "application/json" },
      body: JSON.stringify(body),
      ...(init || {}),
    });
  }
}
