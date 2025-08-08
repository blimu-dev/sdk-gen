

import { CoreClient, ClientOption } from "./client";
import { ApiKeysService } from "./services/api_keys";
import { AppService } from "./services/app";
import { OrganizationsService } from "./services/organizations";
import { ResourceTypeService } from "./services/resource_type";
import { MiscService } from "./services/misc";

export class Blimu {
  readonly apiKeys: ApiKeysService;
  readonly app: AppService;
  readonly organizations: OrganizationsService;
  readonly resourceType: ResourceTypeService;
  readonly misc: MiscService;

  constructor(options?: ClientOption) {
    const core = new CoreClient(options);
    this.apiKeys = new ApiKeysService(core);
    this.app = new AppService(core);
    this.organizations = new OrganizationsService(core);
    this.resourceType = new ResourceTypeService(core);
    this.misc = new MiscService(core);
  }
}

export type { ClientOption };
