# Zevenet Loadbalancer Provider

## Labels (on backend services)

The following labels are supported on backend container, which host the services to be added to the loadbalancer.

| Label Name | Description | Example  |  Optional   |
|-----------|------|-------|-------|
| io.rancher.service.external_lb.endpoint | Hostname pattern to use for the service. | blog(\\.example\\.com)? | No |
| io.rancher.service.external_lb.http_redirect_url | Redirect URL to use for HTTP requests (without HTTPS). | https://blog.example.com | Yes |

## Environment Variables

| Variable Name | Description | Default Value   | Optional   |
|-----------|------|-------|------|
| ZAPI_HOST | The hostname of the Zevenet Loadbalancer, like *mylbcluster:444*  |   | No  |
| ZAPI_FARM | The name of the farm to register the services in. Use multiple containers to support multiple farms.  |   | No    |
| ZAPI_KEY | The key of the *zapi* user.  |   | No    |
| LB_TARGET_RANCHER_SUFFIX | Service names in the farm will have this suffix. Choose something simple, like "rancher". | "rancher.internal" | Yes |
