# Gravitee Management API Provider

## Labels (on Rancher services)

The following labels are supported on backend container, which host the services to be added to the Gravitee API.

| Label Name | Description | Example | Optional |
|-----------|------|-------|-------|
| io.rancher.service.external_lb.endpoint | Gravitee API label to look for. | my-fancy-api | No 
| io.rancher.service.external_lb.provider | Has to be set to 'gravitee' to be handled by LB provider. Ignored if not set. | gravitee | Yes

Important: The `io.rancher.service.external_lb.endpoint` has to exist to trigger the load-balancer integration.

## Environment Variables

| Variable Name | Description | Default Value   | Optional   |
|-----------|------|-------|------|
| GRAVITEE_HOST | The hostname of the Gravitee Management API, like *api.mygravitee.net* or *http://my-gravitee:8080* |   | No  |
| GRAVITEE_USER | The name of the user to user for Management API access.  |   | No    |
| GRAVITEE_PWD | The password of the user to user for Management API access.  |   | No    |
| LB_TARGET_RANCHER_SUFFIX | Service names in the farm will have this suffix. Choose something simple, like "rancher". | "rancher.internal" | Yes |
