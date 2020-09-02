# testk8s
In this repo there are some tests that evaluate performance (throughput, latency and CPU usage) of CNI installed in the cluster.

Two main scenarios are taken into consideration, shown below in a schematic way:
  - Pod to Pod 
  - Pod to Service to Pod.


In both sets of tests, throughput and CPU usage for TCP and UDP protocols are computed for CNI installed, with the aid of Iperf3, Iperf2 and Netperf.
