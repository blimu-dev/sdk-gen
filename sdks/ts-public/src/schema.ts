// Generated types from OpenAPI components.schemas
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
  description?: string;
  key?: string;
  name: string;
  parentResourceTypeId?: string;
}
export const ResourceTypeList_Output_Items_Item_Status = {
  ACTIVE: "ACTIVE",
  ARCHIVED: "ARCHIVED",
} as const;

export type ResourceTypeList_Output_Items_Item_Status =
  (typeof ResourceTypeList_Output_Items_Item_Status)[keyof typeof ResourceTypeList_Output_Items_Item_Status];
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
export const ResourceType_Output_Status = {
  ACTIVE: "ACTIVE",
  ARCHIVED: "ARCHIVED",
} as const;

export type ResourceType_Output_Status =
  (typeof ResourceType_Output_Status)[keyof typeof ResourceType_Output_Status];
export interface ResourceType_Output {
  createdAt: string;
  description?: string;
  id: string;
  name: string;
  status: ResourceType_Output_Status;
  updatedAt: string;
}
