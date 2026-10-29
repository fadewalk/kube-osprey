# OAuth Setup Guide

This guide explains how to configure OAuth authentication for KubeOsprey, supporting multiple authentication providers.

## Creating OAuth Applications at Providers

1. Log in to your OAuth provider account (such as GitHub, Google, etc.).
2. Create a new OAuth application and fill in the necessary information.
3. In the redirect URI, fill in `https://${HOST}/api/auth/callback`, replacing `${HOST}` with your KubeOsprey deployment address.
   1. For example, if your KubeOsprey is deployed at `kube-osprey.example.com`, the redirect URI should be `https://kube-osprey.example.com/api/auth/callback`.
   2. KubeOsprey uses the backend Host and protocol from the request to generate the redirect Host by default.
   3. If KubeOsprey is deployed behind a proxy, it will read the `X-Forwarded-Host` and `X-Forwarded-Proto` headers by default.
   4. If the above information is not available, you can configure the `HOST` environment variable to specify explicitly.
4. Record the generated Client ID and Client Secret.

## Configuration

In the user interface with the **admin** role, the settings entry will be displayed in the upper right corner of the page.

Follow the instructions on the page to fill in the basic information to use OAuth login.
![OAuth](/screenshots/oauth.png)

### Advanced Settings

KubeOsprey supports overriding default OIDC claims and restricting access based on user groups. These options can be configured in the **Advanced Settings** section when adding or editing an OAuth Provider:

- **Username Claim**: Overrides the default claim used to extract the user's username (e.g., `preferred_username`, `upn`, or `nickname`). If left empty, KubeOsprey uses standard claims like `email` or `name`.
- **Groups Claim**: Overrides the default claim used to extract the user's groups (e.g., `groups`, `memberOf`, or `roles`). If left empty, KubeOsprey defaults to standard group representations.
- **Allowed Groups**: Restricts login to users who belong to specific groups. Enter a comma-separated list of group names (e.g., `admin, developers`). If configured, users must belong to at least one of these groups to log in. Users without a matching group are denied access.

## Common Issues

### User shows no permissions after login

By default, even after successful login, KubeOsprey will not grant any permissions to the user. You need to manually configure RBAC rules to grant access.

See the [RBAC Configuration Guide](./rbac-config) for details.

### How to map OAuth users to RBAC roles?

You can configure the mapping relationship between OAuth users and RBAC roles in the settings. For specific steps, please refer to the [RBAC Configuration Guide](./rbac-config).

### Login failure

Generally, these are configuration issues. You can check the following points:

1. Ensure the OAuth application's Client ID and Client Secret are configured correctly.
2. Check if the redirect URI matches what is configured in the OAuth application.
3. View KubeOsprey logs for more error information.
