module github.com/RedHatInsights/insights-results-smart-proxy

go 1.24.0

require (
	github.com/BurntSushi/toml v1.5.0
	github.com/RedHatInsights/insights-content-service v1.0.0
	github.com/RedHatInsights/insights-operator-utils v1.27.0
	github.com/RedHatInsights/insights-results-aggregator v1.4.3
	github.com/RedHatInsights/insights-results-aggregator-data v1.3.9
	github.com/RedHatInsights/insights-results-types v1.23.5
	github.com/go-redis/redismock/v9 v9.2.0
	github.com/golang-jwt/jwt/v5 v5.3.0
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/gorilla/mux v1.8.1
	github.com/openshift-online/ocm-sdk-go v0.1.480
	github.com/prometheus/client_golang v1.23.2
	github.com/prometheus/client_model v0.6.2
	github.com/redhatinsights/app-common-go v1.6.8
	github.com/redis/go-redis/v9 v9.15.1
	github.com/rs/zerolog v1.34.0
	github.com/spf13/viper v1.21.0
	github.com/stretchr/testify v1.11.1
	gopkg.in/h2non/gock.v1 v1.1.2
)

require (
	github.com/IBM/sarama v1.46.1 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/archdx/zerolog-sentry v1.8.5 // indirect
	github.com/aws/aws-sdk-go-v2 v1.39.2 // indirect
	github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream v1.7.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.31.12 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.18.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.9 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.3 // indirect
	github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs v1.58.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.29.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.1 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.38.6 // indirect
	github.com/aws/smithy-go v1.23.0 // indirect
	github.com/aymerick/douceur v0.2.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bitly/go-simplejson v0.5.1 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/cenkalti/backoff/v4 v4.1.3 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/eapache/go-resiliency v1.7.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20230731223053-c322873962e3 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/getkin/kin-openapi v0.133.0 // indirect
	github.com/getsentry/sentry-go v0.35.3 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-openapi/jsonpointer v0.22.1 // indirect
	github.com/go-openapi/swag/jsonname v0.25.1 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/go-yaml/yaml v2.1.0+incompatible // indirect
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/golang/glog v1.2.5 // indirect
	github.com/golang/mock v1.6.0 // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/pprof v0.0.0-20251007162407-5df77e3f7d1d // indirect
	github.com/gorilla/css v1.0.1 // indirect
	github.com/h2non/parth v0.0.0-20190131123155-b4df798d6542 // indirect
	github.com/hashicorp/go-uuid v1.0.3 // indirect
	github.com/itchyny/gojq v0.12.17 // indirect
	github.com/itchyny/timefmt-go v0.1.7 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgtype v1.14.4 // indirect
	github.com/jackc/pgx/v4 v4.18.3 // indirect
	github.com/jcmturner/aescts/v2 v2.0.0 // indirect
	github.com/jcmturner/dnsutils/v2 v2.0.0 // indirect
	github.com/jcmturner/gofork v1.7.6 // indirect
	github.com/jcmturner/gokrb5/v8 v8.4.4 // indirect
	github.com/jcmturner/rpc/v2 v2.0.3 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/lzap/cloudwatchwriter2 v1.6.0 // indirect
	github.com/mailru/easyjson v0.9.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/microcosm-cc/bluemonday v1.0.27 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/mozillazg/request v0.8.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/oasdiff/yaml v0.0.0-20250309154309-f31be36b4037 // indirect
	github.com/oasdiff/yaml3 v0.0.0-20250309153720-d2182401db90 // indirect
	github.com/onsi/ginkgo/v2 v2.26.0 // indirect
	github.com/onsi/gomega v1.38.2 // indirect
	github.com/openshift-online/ocm-api-model/clientapi v0.0.435 // indirect
	github.com/openshift-online/ocm-api-model/model v0.0.435 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/perimeterx/marshmallow v1.1.5 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/common v0.67.1 // indirect
	github.com/prometheus/procfs v0.17.0 // indirect
	github.com/qustavo/sqlhooks/v2 v2.1.0 // indirect
	github.com/rcrowley/go-metrics v0.0.0-20250401214520-65e299d6c5c9 // indirect
	github.com/sagikazarmark/locafero v0.12.0 // indirect
	github.com/skratchdot/open-golang v0.0.0-20200116055534-eef842397966 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/verdverm/frisby v0.0.0-20170604211311-b16556248a9a // indirect
	github.com/woodsbury/decimal128 v1.4.0 // indirect
	go.uber.org/automaxprocs v1.6.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.43.0 // indirect
	golang.org/x/mod v0.29.0 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/oauth2 v0.32.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	golang.org/x/tools v0.38.0 // indirect
	google.golang.org/protobuf v1.36.10 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
