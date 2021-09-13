# `gcr.io/paketo-buildpacks/maven`
The Paketo Maven Buildpack is a Cloud Native Buildpack that builds Maven-based applications from source.

## Behavior
This buildpack will participate all the following conditions are met

* `<APPLICATION_ROOT>/pom.xml` exists or `BP_MAVEN_POM_FILE` is set to an existing POM file.

The buildpack will do the following:

* Requests that a JDK be installed
* Links the `~/.m2` to a layer for caching
* If `<APPLICATION_ROOT>/mvnw` exists
  * Runs `<APPLICATION_ROOT>/mvnw -Dmaven.test.skip=true package` to build the application
* If `<APPLICATION_ROOT>/mvnw` does not exist
  * Contributes Maven to a layer with all commands on `$PATH`
  * Runs `<MAVEN_ROOT>/bin/mvn -Dmaven.test.skip=true package` to build the application
* Removes the source code in `<APPLICATION_ROOT>`
* Expands `<APPLICATION_ROOT>/target/*.[ejw]ar` to `<APPLICATION_ROOT>`

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$BP_MAVEN_BUILD_ARGUMENTS` | Configure the arguments to pass to Maven.  Defaults to `-Dmaven.test.skip=true package`. `--batch-mode` will be prepended to the argument list in environments without a TTY.
| `$BP_MAVEN_BUILT_MODULE` | Configure the module to find application artifact in.  Defaults to the root module (empty).
| `$BP_MAVEN_BUILT_ARTIFACT` | Configure the built application artifact explicitly.  Supersedes `$BP_MAVEN_BUILT_MODULE`  Defaults to `target/*.[ejw]ar`.
| `$BP_MAVEN_POM_FILE` | Specifies a custom location to the project's `pom.xml` file. Must be relative to the root of the project. Defaults to `pom.xml`.

## Bindings
The buildpack optionally accepts the following bindings:

### Type: `maven`
|Secret                  | Description
|------------------------|--------------
|`settings.xml`          | If present `--settings=<path/to/settings.xml>` is prepended to the `maven` arguments
|`settings-security.xml` | If present `-Dsettings.security=<path/to/settings-security.xml>` is prepended to the `maven` arguments

### Type: `dependency-mapping`
|Key                   | Value   | Description
|----------------------|---------|------------
|`<dependency-digest>` | `<uri>` | If needed, the buildpack will fetch the dependency with digest `<dependency-digest>` from `<uri>`

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0

