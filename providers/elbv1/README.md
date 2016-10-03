AWS ELBv1 (ELB Classic Load Balancer) Provider
==========

#### About ELB Classic Load Balancers
The [Classic Load Balancer](https://aws.amazon.com/elasticloadbalancing/classicloadbalancer/) option in AWS routes traffic based on application or network level information and is ideal for simple load balancing of traffic across multiple EC2 instances.

#### About this provider
This provider keeps pre-existing Classic Load Balancers updated with the EC2 instances Rancher services are running on, allowing one to use Elastic Load Balancing to load balancer Rancher services.

### Usage

1. Deploy the stack for this provider from the Rancher Catalog
2. Using the AWS Console create a Classic ELB load balancer with one or more listeners and configure it according to your applications requirements. Configure the listener(s) with an "instance protocol" matching that of your application as well as the "instance port" that your Rancher service will expose to the hosts.
3. Create or update your service to expose one or multiple host ports that match the configuration of your ELB listener(s). Then add the service label `io.rancher.service.external_lb.endpoint` using as value the name of the previously created ELB load balancer.

Environment Variables
==========

The following environment variables are used to configure global options for this provider.

| Variable | Description | Default value |
|----------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------|
| ELBV2_AWS_ACCESS_KEY | Your AWS Access Key. Make sure this key has sufficient permissions for the operations required to manage an ELB load balancer. | `-` |
| ELBV2_AWS_SECRET_KEY | Your AWS Secret Key. | `-` |
| ELBV2_AWS_REGION | By default the service will use the region of the instance it is running on to look up the IDs of EC instances. You can override the region by setting this variable. | `<Self-Region>` |
| ELBV2_AWS_VPCID | By default the service will use the VPC of the instance this service is running on to look up the IDs of EC instances. You can override the VPC by setting this variable. | `<Self-VPC>` |
| ELBV2_USE_PRIVATE_IP | If your EC2 instances are registered in Rancher with their private IP addresses, then set this variable to "true". | `false` |

Note: Instead of specifying AWS credentials when deploying the stack you can create an IAM policy and role and associate it with your EC2 instances.

Example IAM policy with the minimum required permissions
==========

TODO

License
=======
Copyright (c) 2016 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
