import { CoreClient } from "../client";
import * as Schema from "../schema";

export class AppService {
  constructor(private core: CoreClient) {}

  /**
   * GET /v1
   */
  list(init?: Omit<RequestInit, "method" | "body">): Promise<void> {
    return this.core.request({
      method: "GET",
      path: `/v1`,
      ...(init || {}),
    });
  }
}
