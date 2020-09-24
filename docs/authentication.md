---
layout: page
nav_order: 4
---
# Authentication

Authentication is working through `x-rh-identity` token which is provided by
 3scale. `x-rh-identity` is base64 encoded JSON, that includes data about user
 and organization, like:

```json
{
  "identity": {
    "account_number": "0369233",
    "type": "User",
    "user": {
      "username": "jdoe",
      "email": "jdoe@acme.com",
      "first_name": "John",
      "last_name": "Doe",
      "is_active": true,
      "is_org_admin": false,
      "is_internal": false,
      "locale": "en_US"
    },
    "internal": {
      "org_id": "3340851",
      "auth_type": "basic-auth",
      "auth_time": 6300
    }
  }
}
```

If Smart Proxy didn't get identity token or got invalid one, then it returns
error with status code `403` - Forbidden.
