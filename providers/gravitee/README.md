# Gravitee Management API Provider

## Labels (on Rancher services)

The following labels are supported on backend container, which host the services to be added to the Gravitee API.

| Label Name | Description | Example | Optional |
|-----------|------|-------|-------|
| io.rancher.service.external_lb.endpoint | Gravitee API label to look for. | my-fancy-api | No 
| io.rancher.service.external_lb.provider | Has to be set to 'gravitee' to be handled by LB provider. Ignored if not set. | gravitee | Yes
| io.rancher.service.external_lb.encrypt | The backend service port is an HTTPS endpoint and re-encryption is required. Default is false. | true | Yes 
| io.rancher.service.external_lb.keep_alive | Requests to the backend service will use HTTP keep alive. Default is true. | false | Yes
| io.rancher.service.external_lb.pipelining | Requests to the backend service will be written to connections without waiting for previous responses to return. Default is false. | true | Yes
| io.rancher.service.external_lb.compress | Enables gzip compression support, and will be able to handle compressed response bodies. Default is true. | false | Yes
| io.rancher.service.external_lb.follow_redirects | Follows redirects returned by the backend service. Default is false. | true | Yes
| io.rancher.service.external_lb.conn_timeout | Time in milliseconds to wait for the backend service to accept the connection. Default is 5000. | 10000 | Yes
| io.rancher.service.external_lb.read_timeout | Time in milliseconds to wait for the backend service to start sending the response. Default is 10000. | 60000 | Yes
| io.rancher.service.external_lb.idle_timeout | Time in milliseconds to wait for an idle connection to be closed. Default is 60000. | 60000 | Yes
| io.rancher.service.external_lb.max_conn | Maximum amount of concurrent connections to a single backend service. Default is 100. | 5000 | Yes

Important: The `io.rancher.service.external_lb.endpoint` has to exist to trigger the load-balancer integration.

## Environment Variables

| Variable Name | Description | Default Value   | Optional   |
|-----------|------|-------|------|
| GRAVITEE_HOST | The hostname of the Gravitee Management API, like *api.mygravitee.net* or *http://my-gravitee:8080* |   | No  |
| GRAVITEE_USER | The name of the user to user for Management API access.  |   | No    |
| GRAVITEE_PWD | The password of the user to user for Management API access.  |   | No    |
| LB_TARGET_RANCHER_SUFFIX | Service names in the farm will have this suffix. Choose something simple, like "rancher". | "rancher.internal" | Yes |
