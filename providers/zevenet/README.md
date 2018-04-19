# Zevenet Loadbalancer Provider

## Labels (on backend services)

The following labels are supported on backend container, which host the services to be added to the loadbalancer.

| Label Name | Description | Example | Optional | Farm-specific |
|-----------|------|-------|-------|---|
| io.rancher.service.external_lb.endpoint | Hostname pattern to use for the service. | blog(\\.example\\.com)? | No | Yes
| io.rancher.service.external_lb.provider | Has to be set to 'zevenet' to be handled by LB provider. Ignored if not set. | zevenet | Yes | No
| io.rancher.service.external_lb.farms | List of farms to add the service to. Separated by comma. | MainHTTP,MainHTTPS | No | No
| io.rancher.service.external_lb.http_redirect_url | Redirect URL to use for HTTP requests (without HTTPS). | https://blog.example.com | Yes | Yes
| io.rancher.service.external_lb.url_pattern | The URL pattern to use for limiting handled requests. | ^public/ | Yes | Yes
| io.rancher.service.external_lb.check | The Farm Guarian check command used to monitor the backend service.<br>If set to "true" the default "check_http [-S] -H HOST -p PORT" will be used. [1] | check_http -H HOST -p PORT | Yes | No
| io.rancher.service.external_lb.encrypt | The backend service port is an HTTPS endpoint and re-encryption is required. Default is false. | true | Yes | No

Some labels are farm-specific and are support both as `io.rancher.service.external_lb.setting` and `io.rancher.service.external_lb.farm.setting`. The name of the farm must be lowercase.

Important: The `io.rancher.service.external_lb.endpoint` has to exist to trigger the load-balancer integration.

 [1] see https://www.zevenet.com/knowledge-base/enterprise-edition/enterprise-edition-v5-0-administration-guide/lslb-farms-update-farm-guardian/

## Environment Variables

| Variable Name | Description | Default Value   | Optional   |
|-----------|------|-------|------|
| ZAPI_HOST | The hostname of the Zevenet Loadbalancer, like *mylbcluster:444*  |   | No  |
| ZAPI_KEY | The key of the *zapi* user.  |   | No    |
| LB_TARGET_RANCHER_SUFFIX | Service names in the farm will have this suffix. Choose something simple, like "rancher". | "rancher.internal" | Yes |
