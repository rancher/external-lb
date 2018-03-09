# Zevenet Loadbalancer Provider

## Labels (on backend services)

The following labels are supported on backend container, which host the services to be added to the loadbalancer.

| Label Name | Description | Example  |  Optional   |
|-----------|------|-------|-------|
| io.rancher.service.external_lb.endpoint | Hostname pattern to use for the service. | blog(\\.example\\.com)? | No |
| io.rancher.service.external_lb.farms | List of farms to add the service to. Separated by comma. | MainHTTP,MainHTTPS | No |
| io.rancher.service.external_lb.http_redirect_url | Redirect URL to use for HTTP requests (without HTTPS). | https://blog.example.com | Yes |
| io.rancher.service.external_lb.check | The Farm Guarian check command used to monitor the backend service.<br>If set to "true" the default "check_http -H HOST -p PORT" will be used. [1] | check_http -H HOST -p PORT | Yes |

 [1] see https://www.zevenet.com/knowledge-base/enterprise-edition/enterprise-edition-v5-0-administration-guide/lslb-farms-update-farm-guardian/

## Environment Variables

| Variable Name | Description | Default Value   | Optional   |
|-----------|------|-------|------|
| ZAPI_HOST | The hostname of the Zevenet Loadbalancer, like *mylbcluster:444*  |   | No  |
| ZAPI_KEY | The key of the *zapi* user.  |   | No    |
| LB_TARGET_RANCHER_SUFFIX | Service names in the farm will have this suffix. Choose something simple, like "rancher". | "rancher.internal" | Yes |
