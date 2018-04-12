external-lb
==========
Rancher service facilitating integration of rancher with external load balancers. This service updates external LB with services created in Rancher that ask to be load balanced using an external LB. 
Initial version comes with f5 BIG-IP support; but a pluggable provider model makes it easy to implement other providers later.

Design
==========
* The external-lb gets deployed as a Rancher service containerized app. 

* It enables any other service to be registered to external LB if the service has exposed a public port and has the label 'io.rancher.service.external_lb_endpoint'

* Value of this label should be equal to the external LB endpoint that should be used for this service - example the VirtualServer Name for f5 BIG-IP

* The external-lb service will fetch info from rancher-metadata server at a periodic interval, then compare it with the data returned by the LB provider, and propagate the changes to the LB provider.

Environment Variables
==========

The following environment variables are used to configure global options.

| Variable | Description | Default value |
|----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------|
| POLL_INTERVAL | Value in milliseconds to check for rancher metadata updates | `1000` |
| FORCE_UPDATE_INTERVAL | Value in minutes to force a resource poll. Increasing this value may be required if you run into api limits enforced by your cloud providor | `1` |

Contact
========
For bugs, questions, comments, corrections, suggestions, etc., open an issue in
 [rancher/rancher](//github.com/rancher/rancher/issues).

Or just [click here](//github.com/rancher/rancher/issues/new?title=%5Brancher-dns%5D%20) to create a new issue.

License
=======
Copyright (c) 2015 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
