# CodeQL Tools

CodeQL Tools is a collection of tools that provide both an enablement and a compliance layer for CodeQL. These tools can
be used for automating the rollout of CodeQL to your organization, as well as for ensuring that CodeQL is used in a 
manner that is compliant with your organization's policies. The tools are designed to be used in a CI/CD pipeline, but 
can also be used locally.

## Available Enablement Tools

### GitHub Actions

#### Configure CodeQL

`Configure CodeQL` is a Go-based, Docker GitHub Action that automates the creation of pull requests to enable CodeQL 
on your repositories. It can be used to enable CodeQL on all repositories in an organization, or on a subset of 
repositories based on GitHub App Installations.

See the [Configure CodeQL](configure-codeql/README.md) documentation for additional information.

#### CodeQL Analysis

`CodeQL Analysis` is a GitHub Actions Composite Action that automates the analysis of your repositories using CodeQL 
leveraging the official [CodeQL Action](https://github.com/github/codeql-action) tools from GitHub, while extending them
to include additional use cases.

See the [CodeQL Analysis](codeql-analysis/README.md) documentation for additional information.

#### Parse Build Steps

`Parse Build Steps` is a Node.js-based GitHub Action that enables users to automate the process of providing custom
build steps to the CodeQL Analysis GitHub Action where the `codeql-action/autobuild` Action is not sufficient.

See the [Parse Build Steps](parse-build-steps/README.md) documentation for additional information.

#### Upload Database

`Upload Database` is a Node.js-based GitHub Action that automates the uploading of CodeQL databases to GitHub. This is 
useful for scenarios where the `codeql-action/analyze` Action is not used, but you still want to upload the CodeQL
database to GitHub for use with the [CodeQL VSCode Extension](https://codeql.github.com/docs/codeql-for-visual-studio-code/)
or with [Multi-Repository Variant Analysis](https://codeql.github.com/docs/codeql-for-visual-studio-code/running-codeql-queries-at-scale-with-mrva/).

See the [Upload Database](upload-database/README.md) documentation for additional information.

#### Validate Monorepo Access

`Validate Monorepo Access` is a Node.js-based GitHub Action that automates the validation of access to monorepo tools.
CodeQL exposes the ability to add `path-include` and `path-ignore` filters to your queries, which can be used to limit
the scope of the query to a specific directory or set of directories, but allows users to bypass requirements to scan
the entire repository. This Action can be used to validate that the user has access to the `config` property and is
not abusing the feature to bypass requirements.

See the [Validate Monorepo Access](validate-monorepo-access/README.md) documentation for additional information.

### Jenkins Shared Libraries

#### Linux

The `Linux Jenkins Shared Library` is a Shell-based collection of Groovy-based Shared Libraries that can be used to 
automate CodeQL scans in Jenkins pipelines. The libraries are designed to be used in a declarative pipeline, but can 
also be used in a scripted pipeline.

See the [Linux Jenkins Shared Libraries](jenkins/shared-libraries/linux/README.md) documentation for additional information.

#### Windows


The `Windows Jenkins Shared Library` is a PowerShell-based collection of Groovy-based Shared Libraries that can be used 
to automate CodeQL scans in Jenkins pipelines. The libraries are designed to be used in a declarative pipeline, but can 
also be used in a scripted pipeline.

See the [Windows Jenkins Shared Libraries](jenkins/shared-libraries/windows/README.md) documentation for additional information.

## Available Compliance Tools

### GitHub Actions

#### eMASS Promotion

`eMASS Promotion` is a Node.js-based GitHub Action that automates the promotion of CodeQL scans artifacts to eMASS 
repositories. This Action uploads both CodeQL databases and CodeQL SARIF files generated by the reusable workflow to 
eMASS repositories. This Action ensures that assets are placed in location that repository owners cannot modify, and
that the assets are placed in a location that is accessible to organization security teams.

See the [eMASS Promotion](emass-promotion/README.md) documentation for additional information.

#### Verify Scans

`Verify Scans` is a Node.js-based GitHub Action that automates the verification of CodeQL scans. This Action can be
used to ensure that CodeQL scans are being performed on a regular basis, and that the scans are being performed with
the correct configuration and that repositories are leveraging the reusable workflow for governance, and not using 
CodeQL directly.

See the [Verify Scans](verify-scans/README.md) documentation for additional information.
