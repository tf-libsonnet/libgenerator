# tf.libsonnet Library Generator

`tf-libsonnet/libgenerator` is a code generator for Jsonnet Terraform libraries.

This binary generates the library source code using the schema definitions for resources and data sources provided by
the provider binaries (from the `terraform providers schema` command).

The list of supported providers as well as the destination repositories can be found in the
[cfg/managed.json](./cfg/managed.json) config file.

## Usage

### Running the standalone binary

If you are managing a new custom provider, or would like to tweak the generated output, you can run the binary directly
to generate the libsonnet files on your local filesystem.

Compiled binaries for various platforms are available in [the releases
page](https://github.com/tf-libsonnet/libgenerator/releases). Make sure to put it somewhere on your `PATH`, such as
`/usr/local/bin`.

You can also compile from source using `go install` if you have [Go 1.19+](https://tip.golang.org/) installed. E.g., to
get the latest version:

```
go install github.com/tf-libsonnet/libgenerator/cmd/libgenerator@latest
```

Once you have installed the binary, you can generate the libsonnet for a provider by passing in the desired provider
parameters with the `--provider` flag. For example, if you want to generate the libsonnet for the [Doppler
provider](https://registry.terraform.io/providers/DopplerHQ/doppler/latest/docs):

```
libgenerator gen --provider 'src=DopplerHQ/doppler&version=~>1.0'
```

This will generate the libsonnet files in the `out` directory of the current working directory. You can then copy the
results into a repo you maintain, or as a vendored file in your main infrastructure repo where you are generating
Terraform code (e.g., `infrastructure-live`).

### Adding a new managed provider

Due to limited bandwidth, we do not default to generating and maintaining a library for all providers. However, we are
open to supporting any providers that will get active usage from the community.

You can request a new provider to be managed by the `tf.libsonnet` org by opening a pull request with the provider added
to the [cfg/managed.json config](./cfg/managed.json). During the PR process, we will allocate a new repository to house
the generated code and run the generator through the CI job to generate the new library. From there, you can test the
generated code to verify the changes.


## Status of provider support

While the generator is capable of generating libsonnet code for arbitrary providers, it is difficult to maintain and test
compatibility for all of them over time. This means that we can't guarantee that the generated code will work for all
production environments and use cases, nor that all functions on resources and data sources work as intended. However,
certain providers see active production usage from the community, and these providers are generally considered to be
stable.

We define the following stability levels for the generated provider libraries:

- **Stable**: Actively used in production workloads. The target repository contains regression tests to maintain
  compatibility across versions and releases. Issues relating to these providers have the highest priority.
- **Beta**: Actively used in production workloads, but there is no regression testing. Issues relating to these
  providers will be prioritized.
- **Alpha**: While automated releases are configured, there is no known active production usage. Issues relating to
  these providers are addressed on a best effort basis.

The following table clarifies the current status of the providers according to the aforementioned stability levels:

| Provider                                                               | Status |
|------------------------------------------------------------------------|--------|
| [azurerm](https://github.com/tf-libsonnet/hashicorp-azurerm)           | Beta   |
| [azuread](https://github.com/tf-libsonnet/hashicorp-azuread)           | Beta   |
| [hcp](https://github.com/tf-libsonnet/hashicorp-hcp)                   | Beta   |
| [null](https://github.com/tf-libsonnet/hashicorp-null)                 | Beta   |
| [DopplerHQ/doppler](https://github.com/tf-libsonnet/dopplerhq-doppler) | Beta   |
| [aws](https://github.com/tf-libsonnet/hashicorp-aws)                   | Alpha  |
| [google](https://github.com/tf-libsonnet/hashicorp-google)             | Alpha  |
| [google-beta](https://github.com/tf-libsonnet/hashicorp-google-beta)   | Alpha  |

Note that the stability levels within a provider library will vary across resources and data sources. That is, some
resources within a provider may be considered **Alpha** level, while others may be **Stable**. In general, a **Stable**
provider has a large enough community of users such that most resources and data sources are covered.
