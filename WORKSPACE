load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
http_archive(
    name = "io_bazel_rules_go",
    urls = ["https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/v0.20.1/rules_go-v0.20.1.tar.gz"],
    sha256 = "842ec0e6b4fbfdd3de6150b61af92901eeb73681fd4d185746644c338f51d4c0",
)

http_archive(
    name = "bazel_gazelle",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.0/bazel-gazelle-v0.19.0.tar.gz"],
    sha256 = "41bff2a0b32b02f20c227d234aa25ef3783998e5453f7eade929704dcff7cd4b",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")
go_rules_dependencies()
go_register_toolchains()
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
gazelle_dependencies()

go_repository(
     name = "com_github_stretchr_testify",
     importpath = "github.com/stretchr/testify",
     commit = "12b6f73e6084dad08a7c6e575284b177ecafbc71",  # v1.2.1[201~
     )

go_repository(
    name = "com_github_aws_aws_sdk_go",
    importpath = "github.com/aws/aws-sdk-go",
    commit = "c8a63b5774a90dab70f0dc6fddc7e7925416d90a",  # v1.12.61
    )

go_repository(
    name = "com_github_getsentry_raven_go",
    importpath = "github.com/getsentry/raven-go",
    commit = "a646e49f77dbb79f6a36a7043425b6c3d40397bd",
    )

go_repository(
    name = "com_github_stripe_veneur",
    importpath = "github.com/stripe/veneur",
    commit = "3caffd261880b1aaf304e4ed8970f74db8de2a9b",
    )

go_repository(
    name = "io_goji",
    importpath = "goji.io",
    commit = "8ec55ab31c920305eae42c9a5cb571f2534a672d",
    )

go_repository(
    name = "com_github_certifi_gocertifi",
    importpath = "github.com/certifi/gocertifi",
    commit = "03be5e6bb9874570ea7fb0961225d193cbc374c5",  # 2017.01.23
    )

go_repository(
    name = "com_github_pkg_errors",
    importpath = "github.com/pkg/errors",
    commit = "645ef00459ed84a119197bfb8d8205042c6df63d",  # v0.8.0
    )
