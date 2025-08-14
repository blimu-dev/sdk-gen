import { CoreClient } from "../client";
import * as Schema from "../schema";

export class AppService {
  constructor(private core: CoreClient) {}

  /**
   * GET /v1
   */
  gethello(
    init?: Omit<RequestInit, "method" | "body">
  ): Promise<unknown> {
    return this.core.request({
      method: "GET",
      path: `/v1`,
      ...(init || {}),
    });
  }

  
  /**
   * @summary Get query keys for gethello
   * @returns ['v1']
   */
  gethello__queryKeys(
  ) {
    return ['v1'] as const;
  }
}
