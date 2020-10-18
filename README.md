# testk8s
A CNI benchmarking tool that returns information about performance of CNI installed in the cluster.    
It is composed of several tests that evaluate some parameters (throughput, latency and CPU usage) with both TCP and UDP protocol in different scenarios.
The scenarios taken into consideration can be divided in two main groups:
  - Pod to Pod 
  - Pod to Service to Pod.

Each test reproduces a specific scenario, measuring one or more parameters using famous network tools and saving the results obtained.

Tools used for testing:
  - Pod to Pod:
  
                A- iperf3 (tcp & udp mode) 
                B- netperf (tcp & udp mode)
                
  - Pod to Service to Pod:
  
                A- iperf3 tcp mode
                B- iperf2 udp mode
                C- netperf tcp mode
                D- curl (to measure latency & speed downloading) 

Some tests stress the installed plugin, creating a large number of resources (both services and network policies) to see the behavior of the network provider and how much their performance deteriorates.

NB: PAY ATTENTION that an huge number of services implies a large number of pods created in the cluster (1 svc created <-> 1 pod created linked to service).


For now, all tests are executed sequentially. In the future filters to select specific test scenarios will be created.

To run these tests:
            
                1- clone this repo (git clone https://github.com/DavidLiffredo/testk8s.git)
                2- cd testk8s
                3- go run launchTests.go

