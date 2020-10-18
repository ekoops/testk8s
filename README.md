# testk8s
In this repo there are some tests that evaluate performance (throughput, latency and CPU usage) of CNI installed in the cluster.

Two main scenarios are taken into consideration, shown below in a schematic way:
  - Pod to Pod 
  - Pod to Service to Pod.


In both sets of tests, throughput and CPU usage for TCP and UDP protocols are computed for CNI installed, with the aid of Iperf3, Iperf2 and Netperf.

In the former category pods are deployed in two different ways: in the first case two pods are created in different workers, in the other case two pods are deployed in the same worker (communication inter-node and intra-node). In addition, it is possible to run these tests with Network Policies installed in the cluster (the number of network policies installed is variable, now fixed to 1000 and 10000).

The latter category has similar scenarios (pod deployed in different and same nodes), but in this case PodClient contacts the service that redirect the traffic to PodServer. Several tests can be runned with a growing service number in the cluster (also the number of services is configurable, for now tests with 1,10,100,1000,10000 services are available) and a growing number of PodServer replica (for now 1,10,20,50,100).
Due to problem with netperf and iperf, tests with PodServer replica > 1 are executed with curl tool (with different outputs).

NB: PAY ATTENTION that an huge number of services implies a large number of pods created in the cluster (1 svc created <-> 1 pod created linked to service).


Tools used for testing:
  - Pod to Pod:
  
                A- iperf3 (tcp & udp mode) 
                B- netperf (tcp & udp mode)
                
  - Pod to Service to Pod:
  
                A- iperf3 tcp mode
                B- iperf2 udp mode
                C- netperf tcp mode
                D- curl (to measure latency & speed downloading) 

For now, all tests are executed sequentially. In the future filters to select specific test scenarios will be created.

To run these tests:
            
                1- clone this repo (git clone https://github.com/DavidLiffredo/testk8s.git)
                2- cd testk8s
                3- go run launchTests.go

