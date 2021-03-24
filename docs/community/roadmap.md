# Volcano Roadmap

## v1.0 (Planned on June 30)

The major target of this release to make Volcano more stable for product.

### Stability and Resilience

Improve test coverage. In v1.0, more test cases will be added to improve Volcano stability for product.

### Preemption/Reclaim Enhancement

Preemption and Reclaim are two import features for resource sharing; there're two actions for now, but unstable. In v1.0, those two features are going to be enhanced for elastic workload, e.g. stream job, bigdata batch job.

### GPU Share ([#624](https://github.com/volcano-sh/volcano/issues/624))

A better performance has its cost, including GPU; and there are several scenarios that a Pod can not consume one GPU, e.g. inference workload, dev environment. One of solutions is to support GPU share, including related enhancement to both scheduler and kubelet.

### Integrate with Apache Flink

Flink is a widely used for Stateful Computations over Data Streams, but flink on kubernetes has some gaps now.

### Integrate with argo to support job dependencies

Investigate to cooperate with argo to support job dependencies.

### Support running MindSpore jobs

[MindSpore](https://www.mindspore.cn/) is a deep learning training and inference framework, support running MindSpore training with volcano job.

## v1.2(Planned on Oct 23,2020)
### Queue Resource Reservation
* Description: Support reserve specified resource for queue without restart Volcano.
* Priority: High
* Issue: https://github.com/volcano-sh/volcano/issues/1101
* Owner: @hudson741@Thor-wl

### Fair Scheduling For Jobs Of Same Priority And Different Queue
* Description: Schedule jobs of same priority but from different queue accord to create time.
* Priority: High
* Issue: https://github.com/volcano-sh/volcano/issues/1098
* Owner: @alcorj-mizar

### Differentiated Scheduling Strategies For Different Queue
* Description: Support configure actions and plugins for different queues.
* Priority: High
* Issue: https://github.com/volcano-sh/volcano/issues/1035
* Owner: @sresthas

### Support Hierarchy Queue
* Description: Support Hierarchy Queue algorithm.
* Priority: High
* Issue: https://github.com/volcano-sh/volcano/issues/1033
* Owner: @My-pleasure

### Job PriorityClassName Update
* Description: Support update vcjob priorityClassName update when job has not been scheduled.
* Priority: Middle
* Issue: https://github.com/volcano-sh/volcano/issues/1097
* Owner: @merryzhou

### Status Message Enhanced For CRD
* Description: Provide more status detail for CRD status when use CLI such job fail reason.
* Priority: Middle
* Issue: https://github.com/volcano-sh/volcano/issues/1094
* Owner:@mikechengwei

### Support MinAvailable For Task
* Description: Support MinAvailable for task
* Priority: Middle
* Issue: https://github.com/volcano-sh/volcano/issues/988
* Owner: @shinytang6

## v1.3(Planned on March 12, 2021)
### Task-Topology
* Description: Support task topology scheduling
* Priority: High
* Issue: https://github.com/volcano-sh/volcano/issues/1349

### Support multiple scheduler
* Description: Support multiple scheduler by admission controller.
* Priority: High
* Issue: https://github.com/volcano-sh/volcano/issues/1322

### Stability and Resilience
* Description: Improve the UT/E2E test coverage and add the stress test to improve stability.
* Priority: High
* Issue: https://github.com/volcano-sh/volcano/issues/1284

### Volcano Device Plugin enhancement
* Description: Support container using multiples GPU as well as part of GPU card.
* Priority: Hign
* Issue: https://github.com/volcano-sh/devices/issues/12

### Queue Resource Reservation
* Description: Support reserve specified resource for queue without restart Volcano.
* Priority: Medium
* Issue: https://github.com/volcano-sh/volcano/issues/1101

## v1.4 and later (To be updated)
### Support Hierarchy Queue
* Description: Support Hierarchy Queue algorithm.
* Issue: https://github.com/volcano-sh/volcano/issues/1033

### Support configuration hot update
* Description: Add hot update for Volcano components arguments.
* Issue: https://github.com/volcano-sh/volcano/issues/1326

### Improve resource calculation accuracy
* Description: Support high accurate resource calculation.
* Issue: https://github.com/volcano-sh/volcano/issues/1196

### Support job backfill
* Description: Add backfill functionality to improve the resource utilization.

### Improve the Autoscaling enficiency
* Description: Combine the Autoscaler and scheduler to improve the scaling efficiency.

### Add fine-grained monitor metrics
* Description: Enhance monitor to add more metrics for jobs, queues, etc.

### Add stress test
* Description: Add the stress test.