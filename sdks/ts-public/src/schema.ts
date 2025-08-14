// Generated types from OpenAPI components.schemas

export type Enum<T> = T[keyof T];
export const ResourceTypeList_Output_Items_Item_Status = {
  "ACTIVE": "ACTIVE",
  "ARCHIVED": "ARCHIVED",
} as const;

export type ResourceTypeList_Output_Items_Item_Status = Enum<typeof ResourceTypeList_Output_Items_Item_Status>;

  
export const ResourceType_Output_Status = {
  "ACTIVE": "ACTIVE",
  "ARCHIVED": "ARCHIVED",
} as const;

export type ResourceType_Output_Status = Enum<typeof ResourceType_Output_Status>;

  
export interface ApiKeyCreateDto {
  name: string;
  organizationId: string;
}
export interface ApiKeyDto_Output {
  createdAt: string;
  id: string;
  isActive: boolean;
  key: string;
  name: string;
  organizationId: string;
  updatedAt: string;
}
export interface ApiKeyListDto_Output_Data_Item {
  createdAt: string;
  id: string;
  isActive: boolean;
  key: string;
  name: string;
  organizationId: string;
  updatedAt: string;
}
export interface ApiKeyListDto_Output {
  data: Array<ApiKeyListDto_Output_Data_Item>;
  total: number;
}
export interface OrganizationCreateDto {
  key?: string;
  name: string;
}
export interface OrganizationDto_Output {
  createdAt: string;
  id: string;
  name: string;
  updatedAt: string;
}
export interface OrganizationListDto_Output_Data_Item {
  createdAt: string;
  id: string;
  name: string;
  updatedAt: string;
}
export interface OrganizationListDto_Output {
  data: Array<OrganizationListDto_Output_Data_Item>;
  total: number;
}
export interface OrganizationUpdateDto {
  name: string;
}
export interface ResourceTypeCreateBody {
  /** The description of the resource type */
  description?: string;
  /** The short of the resource type, such as "org" or "team". If not provided, it will be generated from the name. Must be lowercase and contain only letters, numbers, and hyphens. */
  key?: string;
  /** The name of the resource type */
  name: string;
  /** The ID of the parent resource type */
  parentResourceTypeId?: string;
}
export interface ResourceTypeList_Output_Items_Item {
  createdAt: string;
  description?: string;
  id: string;
  name: string;
  status: ResourceTypeList_Output_Items_Item_Status;
  updatedAt: string;
}
export interface ResourceTypeList_Output {
  items: Array<ResourceTypeList_Output_Items_Item>;
  total: number;
}
export interface ResourceType_Output {
  createdAt: string;
  description?: string;
  id: string;
  name: string;
  status: ResourceType_Output_Status;
  updatedAt: string;
}



// Operation query parameter interfaces
/**
 * Query params for Resource Type.List
 */
export interface ResourceTypeListQuery {
  /** The parent resource type ID or slug */
  parentResourceTypeId?: string;
  /** The search query to filter resource types by name, description, or slug */
  search?: string;
  /** The status of the resource type */
  status?: "active" | "archived";
}
