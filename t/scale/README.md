Scale tests
===========

The goal of these tests is to collect and verify metrics obtained from status-go application
under load.

To run these tests you need:
- install docker-compose 
- build status-go container with metrics and prometheus tags
```bash
make docker-image BUILD_TAGS="metrics prometheus
```

Tests can be run with:

```
go test -v -timeout=20m ./t/scale/ -wnode-scale=20
```

Wnode-scale is optional and 12 will be used by default. Timeout is also
optional but you need to be aware of it if you are extending or changing parameters of tests.
Most of the tests should print summary table after they are finished, if you are not interested
in it - remove verbosity flag.

Example of whisper summary table:

|HEADERS	|ingress	|egress		|dups	|new	|dups/new|
|-		|-		|-		|-	|-	|-       |
|0		|5.740088 mb	|4.705319 mb	|3255	|800	|4.068750|
|1		|5.315731 mb	|8.843322 mb	|2795	|801	|3.489388|
|2		|7.029315 mb	|7.535808 mb	|4045	|806	|5.018610|
|3		|7.103868 mb	|5.796416 mb	|4212	|800	|5.265000|
|4		|4.842260 mb	|9.102069 mb	|2457	|800	|3.071250|
|5		|12.886810 mb	|6.930454 mb	|8457	|804	|10.518657|
|6		|5.703964 mb	|5.960101 mb	|3191	|805	|3.963975|
|7		|10.962758 mb	|6.197109 mb	|7015	|801	|8.757803|
|8		|5.423563 mb	|4.987119 mb	|3038	|800	|3.797500|
|9		|6.764746 mb	|7.007739 mb	|3793	|902	|4.205100|
|TOTAL		|170.663310 mb	|170.692327 mb	|96648	|21637	|4.500044|