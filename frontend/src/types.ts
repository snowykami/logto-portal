export type User = {
  sub: string;
  email: string;
  emailVerified: boolean;
  name: string;
  preferredUsername: string;
  picture: string;
  roles: string[];
  organizations: string[];
  organizationRoles: string[];
};

export type AppCatalogItem = {
  id: string;
  name: string;
  description: string;
  url: string;
  icon: string;
  requiredRoles: string[];
  requiredOrganizations: string[];
  accessible: boolean;
  reasons: string[];
};

export type Announcement = {
  id: string;
  title: string;
  content: string;
  severity: 'info' | 'warning' | 'critical' | string;
  pinned: boolean;
  targetRoles: string[];
  targetOrganizations: string[];
  targetUsers: string[];
};

export type PortalData = {
  user: User;
  applications: AppCatalogItem[];
  announcements: Announcement[];
};

export type AppRequest = {
  id: string;
  requesterSub: string;
  requesterEmail: string;
  name: string;
  type: string;
  description: string;
  redirectUris: string[];
  postLogoutRedirectUris: string[];
  corsAllowedOrigins: string[];
  portalUrl: string;
  reason: string;
  status: string;
  logtoApplicationId?: string;
  reviewerSub?: string;
  reviewNote?: string;
  createdAt: string;
  reviewedAt?: string;
};

export type PermissionRequest = {
  id: string;
  requesterSub: string;
  requesterEmail: string;
  kind: string;
  roleId?: string;
  roleName?: string;
  applicationId?: string;
  reason: string;
  status: string;
  reviewerSub?: string;
  reviewNote?: string;
  createdAt: string;
  reviewedAt?: string;
};
