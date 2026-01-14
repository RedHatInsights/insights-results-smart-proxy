# Resource Based Access Control in Insights Results Smart Proxy

## Overview

The Insights Results Smart Proxy implements an authorization mechanism to control access to its endpoints. This functionality ensures that only authorized users can access certain resources based on their roles and permissions. The authorization is primarily handled through a middleware that checks the user's identity and permissions before allowing access to specific routes.

## Middlewares

### Authentication Middleware

The authentication middleware is responsible for verifying that the request has been made by a known party, and that said party's identity token has all the information necessary for identification of the concrete requester. It is implemented in the `auth` module and the `Authentication` function within the `auth_middleware.go` file. If enabled, authentication always happens before the authorization process.

### Authorization Middleware

The authorization middleware is responsible for enforcing role-based access control (RBAC) for incoming requests. It is implemented in the `auth` module and the `Authorization` function within the `auth_middleware.go` file. The authorization middleware is set up in the `setupAuthMiddleware` function, which configures the router to use the `Authorization` middleware for routes that require RBAC checks.

#### Key Features:
- **RBAC Enforcement**: The middleware checks if the user has the necessary permissions to access the requested resource. A configuration option, `enforce`, was added to the RBAC configuration in order to enforce or bypass RBAC.
- **Bypass for Specific URLs**: Certain URLs can be configured to bypass authorization checks, allowing public access.
- **User Agent Handling**: Requests from specific user agents (e.g. ACM) can bypass RBAC checks for now. This is a temporary solution that was agreed upon with them, as there is a lot more work related to service accounts configurations and RBAC roles definitions to do in order to ensure that this new functionality doesn't affect our internal API consumers.

### Configuration

To enable the authorization functionality, the following configuration options must be set in the server's configuration:

- **UseRBAC**: This must also be set to `true` to enable the use of role-based access control.

#### Example Configuration:

in TOML configuration files, `server__use_rbac` specifies if RBAC should be applied to incoming requests
```
[server]
use_rbac = false
```

If `server__use_rbac` is set to `true`, the `rbac` configuration section defines where is the RBAC server as well as if RBAC should be enforced

```
[server]
use_rbac = true

[rbac]
url = "https://console.redhat.com/api/rbac/v1"
enforce = false
```

## Authorization Logic

1. **Authentication vs Authorization**:
   - As mentioned earlier, depending on the server's configuration, all requests are first authenticated before authorization is even attempted.

2. **Token Decoding**:
   - The authorization middleware attempts to decode the token (`x-rh-identity`) from the request header, as it contains all the information required to indentify the requester.
   - If the token is missing or malformed, an error response is returned, which is the same as not authorizing the request.

3. **Bypass Conditions**:
   - If the request URI matches any of the `noAuthURLs` list, the middleware allows the request to proceed without authorization checks.
   - If the request method is `OPTIONS`, it is also allowed to bypass authorization.
   - Requests from the ACM user agent are allowed to bypass RBAC checks.

4. **RBAC Check**:
   - If the user is identified as a `ServiceAccount`, the middleware checks if the user is authorized to access the requested resource using the RBAC client.
   - The RBAC client queries the /access/ endpoint of the `Insights RBAC` server, and processes the response to look for access rights related to `ocp-advisor`. Said access rights are defined in the [Insights rbac-config repository][1].
   - If the user is not authorized and RBAC is enforced, a `403 Forbidden` response is returned.


### Enforcement

For the authorization middleware to take action when access is not authorized, the RBAC enforcement must be enabled. This is done by setting the `enforce` flag in the RBAC client configuration.
- If enforcement is not enabled, the middleware will not block access even if the user is not authorized.
- For now, any decision taken by the authorization middleware is logged, with the hope to get more information on customers' readiness and make future development easier.

## Testing

The authorization functionality is covered by unit tests that verify the following scenarios:
- Access to endpoints that should bypass authorization checks (e.g., `noAuthURLs`).
- Access attempts by users with valid and invalid permissions.
- Handling of requests from specific user agents.

End to end test cases using [IQE][2] have been planned, but they will not guarantee that customers are ready for this feature to be enabled. According to the [ADR][3] and the [knowledge base article][4], existing users are supposed to have migrated by end of CY2024. But there has been no way to confirm that, and even internal users of our APIs seem to have different timelimes when it comes to preparing their services for this change.

### Example Test Cases:
- **TestAuthorization_NoAuthURLs**: Verifies that requests to `noAuthURLs` are allowed without RBAC checks.
- **TestAuthorization_ACMUserAgent**: Verifies that requests from the ACM user agent are allowed without RBAC checks.

## Future work

### Proper Roles and Access definitions

As can be seen in the Insights rbac-config repository's [roles definitions][5], only the [OCP advisor Administrator role][6] exists as of today. Ideally, not all the users of our resources should be assigned this role. There is a need to defined more granular roles so that cluster administrators can assign those to each user.

> **Note:** [Guide for user access configuration for RBAC][7].

### Discussions with internal API consumers

Overall, although expected, discussions with our internal customers mainly concluded with them mentioning that they had no knowledge of these required changes. Therefore, some focus will be needed in order to ensure that said customers are not affected negatively by us supporting RBAC. The source code is prepared for that scenario (RBAC can be enforced or not when enabled, and only affects traffic coming from service accounts).

The potentially affected parties are described in our [internal documentation][8]. It needs to be confirmed, but the conclusion after discussing with them is that they are all applications that use the authenticated user's Openshift or SSO token to identify against console.redhat.com. Therefore, it makes sense to assume that if the user in question has properly configured service accounts, Insights RBAC will behave as expected and authorize only the allowed users.



[1]: https://github.com/RedHatInsights/rbac-config/blob/master/configs/prod/permissions/ocp-advisor.json
[2]: https://gitlab.cee.redhat.com/insights-qe/iqe-ccx-plugin
[3]: https://docs.google.com/document/d/1LNun1PuvL3EFyfbSWAa6ixUAm-pJoWzQiNJj7hVjkZY/edit?tab=t.0#heading=h.nq1osedn93p8
[4]: https://source.redhat.com/groups/public/consoledot/consoledot_blog/hcc__service_accounts_with_rbac_support_as_an_alternative_to_basic_auth
[5]: https://github.com/RedHatInsights/rbac-config/tree/bb2d840a33705e84e0819d5fe697aaa7c87c378d/configs/prod/roles
[6]: https://github.com/RedHatInsights/rbac-config/blob/bb2d840a33705e84e0819d5fe697aaa7c87c378d/configs/prod/roles/ocp-advisor.json
[7]: https://docs.redhat.com/en/documentation/red_hat_hybrid_cloud_console/1-latest/html/user_access_configuration_guide_for_role-based_access_control_rbac/index
[8]: https://ccx.pages.redhat.com/ccx-docs/docs/processing/customer/api-consumers-external-apps/
