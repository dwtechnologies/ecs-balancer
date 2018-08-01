# ecs-balancer
When a new ECS instance register in a cluster it's empty. ECS doesn't have a way to reschedule tasks in order to fill the new capacity or spread the worload among AZs.

This Lambda function is triggered by "Container Instance State Change Events" CloudWatch event and:

- check nr of running and pending tasks for ContainerInstanceArn

- if both values are 0 and ECS agent is connected, trigger a cluster balance by upating all the services with 'DesiredCount > 1'



------

**Note**: according to AWS documentation[^ecs_cwe_events], this CW event covers a lot of scenarios.

To avoid too many invokes, CW event pattern is restricted to a specific ECS cluster:

```
Pattern:
  source: ["aws.ec2"]
  detail-type: ["EC2 Spot Instance Interruption Warning"]
```



[^ecs_cwe_events]: https://docs.aws.amazon.com/AmazonECS/latest/developerguide/ecs_cwe_events.html

