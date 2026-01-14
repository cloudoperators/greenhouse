module github.com/cloudoperators/greenhouse

go 1.25.0

replace (
	// DEX import matches version v2.44.0.
	github.com/dexidp/dex => github.com/dexidp/dex v0.0.0-20250829100637-2dce75009acf
	github.com/dexidp/dex/api/v2 => github.com/dexidp/dex/api/v2 v2.4.0

	// Keep Flux dependencies in sync with v2.6.1.
	github.com/fluxcd/helm-controller/api => github.com/fluxcd/helm-controller/api v1.4.5
	github.com/fluxcd/kustomize-controller/api => github.com/fluxcd/kustomize-controller/api v1.7.3
	github.com/fluxcd/pkg/apis/kustomize => github.com/fluxcd/pkg/apis/kustomize v1.14.0
	github.com/fluxcd/pkg/apis/meta => github.com/fluxcd/pkg/apis/meta v1.23.0
	github.com/fluxcd/source-controller/api => github.com/fluxcd/source-controller/api v1.7.4
	github.com/fluxcd/source-watcher/api/v2 => github.com/fluxcd/source-watcher/api/v2 v2.0.3

	// Keep k8s dependencies in sync.
	k8s.io/api => k8s.io/api v0.34.3
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.34.3
	k8s.io/apimachinery => k8s.io/apimachinery v0.35.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.34.3
	k8s.io/client-go => k8s.io/client-go v0.34.3
	k8s.io/component-base => k8s.io/component-base v0.34.3
	k8s.io/kubectl => k8s.io/kubectl v0.34.3
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.22.4
)

require (
	github.com/cenkalti/backoff/v5 v5.0.3
	github.com/dexidp/dex v0.0.0-20250829100637-2dce75009acf
	github.com/fluxcd/helm-controller/api v1.4.5
	github.com/fluxcd/kustomize-controller/api v1.7.3
	github.com/fluxcd/pkg/apis/kustomize v1.21.0
	github.com/fluxcd/pkg/apis/meta v1.23.0
	github.com/fluxcd/source-controller/api v1.7.4
	github.com/fluxcd/source-watcher/api/v2 v2.0.3
	github.com/go-jose/go-jose/v4 v4.1.3
	github.com/google/cel-go v0.26.1
	github.com/google/go-cmp v0.7.0
	github.com/google/go-github/v80 v80.0.0
	github.com/hashicorp/go-retryablehttp v0.7.8
	github.com/jeremywohl/flatten/v2 v2.0.0-20211013061545-07e4a09fb8e4
	github.com/oklog/run v1.2.0
	github.com/onsi/ginkgo/v2 v2.27.3
	github.com/onsi/gomega v1.39.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.23.2
	github.com/spf13/cobra v1.10.2
	github.com/spf13/pflag v1.0.10
	github.com/stretchr/testify v1.11.1
	github.com/vladimirvivien/gexe v0.5.0
	github.com/wI2L/jsondiff v0.7.0
	go.uber.org/zap v1.27.1
	golang.org/x/text v0.32.0
	gopkg.in/yaml.v3 v3.0.1
	helm.sh/helm/v3 v3.19.4
	k8s.io/api v0.34.3
	k8s.io/apiextensions-apiserver v0.34.3
	k8s.io/apimachinery v0.35.0
	k8s.io/cli-runtime v0.34.3
	k8s.io/client-go v0.34.3
	k8s.io/utils v0.0.0-20251222233032-718f0e51e6d2
	sigs.k8s.io/controller-runtime v0.22.4
	sigs.k8s.io/kind v0.31.0
	sigs.k8s.io/yaml v1.6.0
)

require cloud.google.com/go/auth v0.16.5 // indirect

require (
	cel.dev/expr v0.24.0 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	dario.cat/mergo v1.0.1 // indirect
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/Masterminds/sprig/v3 v3.3.0 // indirect
	github.com/antlr4-go/antlr/v4 v4.13.0 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/containerd/errdefs v0.3.0 // indirect
	github.com/containerd/log v0.1.0 // indirect
	github.com/containerd/platforms v0.2.1 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/creack/pty v1.1.23 // indirect
	github.com/fluxcd/pkg/apis/acl v0.9.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fxamacker/cbor/v2 v2.9.0 // indirect
	github.com/ghodss/yaml v1.0.0 // indirect
	github.com/go-sql-driver/mysql v1.9.3 // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/google/gnostic-models v0.7.0 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/gorilla/websocket v1.5.4-0.20250319132907-e064f32e3674 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/mattn/go-sqlite3 v1.14.32 // indirect
	github.com/miekg/dns v1.1.58 // indirect
	github.com/mxk/go-flowrate v0.0.0-20140419014527-cca7078d478f // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.2 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/stoewer/go-strcase v1.3.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.61.0 // indirect
	go.yaml.in/yaml/v2 v2.4.3 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.31.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.13.0 // indirect
	k8s.io/kubectl v0.34.2 // indirect
	oras.land/oras-go/v2 v2.6.0 // indirect
	sigs.k8s.io/randfill v1.0.0 // indirect
	sigs.k8s.io/structured-merge-diff/v6 v6.3.0 // indirect
)

require (
	cloud.google.com/go/compute/metadata v0.8.0 // indirect
	github.com/AdaLogics/go-fuzz-headers v0.0.0-20240806141605-e8a1dd7889d6 // indirect
	github.com/AppsFlyer/go-sundheit v0.6.0 // indirect
	github.com/Azure/go-ansiterm v0.0.0-20250102033503-faa5f7b0171c // indirect
	github.com/Azure/go-ntlmssp v0.0.0-20221128193559-754e69321358 // indirect
	github.com/BurntSushi/toml v1.5.0 // indirect
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/semver/v3 v3.4.0 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2 // indirect
	github.com/beevik/etree v1.6.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chai2010/gettext-go v1.0.3 // indirect
	github.com/containerd/containerd v1.7.29 // indirect
	github.com/coreos/go-oidc/v3 v3.14.1 // indirect
	github.com/cyphar/filepath-securejoin v0.6.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dexidp/dex/api/v2 v2.3.0 // indirect
	github.com/emicklei/go-restful/v3 v3.12.2 // indirect
	github.com/evanphx/json-patch v5.9.11+incompatible // indirect
	github.com/evanphx/json-patch/v5 v5.9.11 // indirect
	github.com/exponent-io/jsonpath v0.0.0-20210407135951-1de76d718b3f // indirect
	github.com/fatih/color v1.17.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/go-asn1-ber/asn1-ber v1.5.8-0.20250403174932-29230038a667 // indirect
	github.com/go-errors/errors v1.5.1 // indirect
	github.com/go-gorp/gorp/v3 v3.1.0 // indirect
	github.com/go-ldap/ldap/v3 v3.4.11 // indirect
	github.com/go-logr/logr v1.4.3
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-logr/zapr v1.3.0 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/google/btree v1.1.3 // indirect
	github.com/google/pprof v0.0.0-20250403155104-27863c87afa6 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.6 // indirect
	github.com/googleapis/gax-go/v2 v2.15.0 // indirect
	github.com/gorilla/handlers v1.5.2 // indirect
	github.com/gorilla/mux v1.8.1 // indirect
	github.com/gosuri/uitable v0.0.4 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
	github.com/jonboulle/clockwork v0.5.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/liggitt/tabwriter v0.0.0-20181228230101-89fcab3d43de // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattermost/xml-roundtrip-validator v0.1.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/spdystream v0.5.0 // indirect
	github.com/moby/term v0.5.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.3-0.20250322232337-35a7c28c31ee // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/image-spec v1.1.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.66.1 // indirect
	github.com/prometheus/procfs v0.16.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rubenv/sql-migrate v1.8.0 // indirect
	github.com/russellhaering/goxmldsig v1.5.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shopspring/decimal v1.4.0 // indirect
	github.com/spf13/cast v1.7.0 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/exp v0.0.0-20251219203646-944ab1f22d93
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/oauth2 v0.34.0
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	golang.org/x/term v0.38.0 // indirect
	golang.org/x/time v0.14.0
	golang.org/x/tools v0.40.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/api v0.248.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
	google.golang.org/grpc v1.75.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/apiserver v0.34.3 // indirect
	k8s.io/component-base v0.34.3 // indirect
	k8s.io/klog/v2 v2.130.1
	k8s.io/kube-openapi v0.0.0-20250910181357-589584f1c912 // indirect
	sigs.k8s.io/json v0.0.0-20250730193827-2d320260d730 // indirect
	sigs.k8s.io/kustomize/api v0.20.1 // indirect
	sigs.k8s.io/kustomize/kyaml v0.20.1 // indirect
)
