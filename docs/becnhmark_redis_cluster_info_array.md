# Benchmark for storing ClusterInfo array in Redis with different data types

## Using `String` type with 30000 clusters in one organization

Key used: `organization:%d:clusters_info`

```
❯ go test -benchmem -run=^$ -bench ^Benchmark github.com/RedHatInsights/insights-results-smart-proxy/services -count 10
goos: linux
goarch: amd64
pkg: github.com/RedHatInsights/insights-results-smart-proxy/services
cpu: 11th Gen Intel(R) Core(TM) i7-1185G7 @ 3.00GHz
BenchmarkStoreClustersInfoForOrg-8   	126	   8816162 ns/op	 9768702 B/op	    2166 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	124	   8480939 ns/op	 9308795 B/op	    2200 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	139	   8447130 ns/op	 9659379 B/op	    1965 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	115	   9224393 ns/op	 9083709 B/op	    2370 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	120	   8928589 ns/op	 8902301 B/op	    2272 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	135	   8229917 ns/op	 9059775 B/op	    2022 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	116	   8957135 ns/op	 8612202 B/op	    2348 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	134	   8338725 ns/op	 8716508 B/op	    2036 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	114	  10978793 ns/op	 9710755 B/op	    2393 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	 93	  11711479 ns/op	 8851276 B/op	    2926 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 19	  63335464 ns/op	23237472 B/op	  109004 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 18	  56157604 ns/op	23388576 B/op	  110058 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 16	  73779307 ns/op	23747417 B/op	  112561 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 18	  63152410 ns/op	23388618 B/op	  110058 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 15	  76072282 ns/op	23962585 B/op	  114061 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 19	  59169575 ns/op	23237498 B/op	  109004 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 18	  57547196 ns/op	23388561 B/op	  110058 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 18	  61328365 ns/op	23388518 B/op	  110057 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 21	  54260694 ns/op	22978555 B/op	  107199 allocs/op
BenchmarkGetClustersInfoForOrg-8     	 21	  58311056 ns/op	22978575 B/op	  107199 allocs/op
PASS
ok  	github.com/RedHatInsights/insights-results-smart-proxy/services	39.266s
```

Average writing time: 9.2113262 ms
Average reading time: 62.3113953 ms

## Using `HSet` type with 30000 clusters in one organization

Key used: `organization:%d:cluster:%s:info`
Stored struct: simplification of `ClusterInfo`, removing the `ClusterName` as used in the key

```
❯ go test -benchmem -run=^$ -bench ^BenchmarkStore github.com/RedHatInsights/insights-results-smart-proxy/services -count 10
goos: linux
goarch: amd64
pkg: github.com/RedHatInsights/insights-results-smart-proxy/services
cpu: 11th Gen Intel(R) Core(TM) i7-1185G7 @ 3.00GHz
BenchmarkStoreClustersInfoForOrg-8   	9	 130180591 ns/op	33897600 B/op	  630078 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	7	 216593692 ns/op	35141674 B/op	  655802 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	5	 298012948 ns/op	37380931 B/op	  702104 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	5	 277542327 ns/op	37380110 B/op	  702097 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	4	 255210232 ns/op	39339910 B/op	  742614 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	6	 176066511 ns/op	36074120 B/op	  675087 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	6	 195984145 ns/op	36074932 B/op	  675098 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	5	 231129258 ns/op	37381185 B/op	  702103 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	4	 252233715 ns/op	39339188 B/op	  742608 allocs/op
BenchmarkStoreClustersInfoForOrg-8   	4	 279100256 ns/op	39339580 B/op	  742614 allocs/op
PASS
ok  	github.com/RedHatInsights/insights-results-smart-proxy/services	38.118s

Average writing time: 231.2053675 ms
