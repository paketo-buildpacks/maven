# `gcr.io/paketo-buildpacks/maven`
The Paketo Maven Buildpack is a Cloud Native Buildpack that builds Maven-based applications from source.

## Behavior
This buildpack will participate all the following conditions are met

* `<APPLICATION_ROOT>/pom.xml` exists

The buildpack will do the following:

* Requests that a JDK be installed
* Links the `~/.m2` to a layer for caching
* If `<APPLICATION_ROOT>/mvnw` exists
  * Runs `<APPLICATION_ROOT>/mvnw -Dmaven.test.skip=true package` to build the application
* If `<APPLICATION_ROOT>/mvnw` does not exist
  * Contributes Maven to a layer with all commands on `$PATH`
  * Runs `<MAVEN_ROOT>/bin/mvn -Dmaven.test.skip=true package` to build the application
* Removes the source code in `<APPLICATION_ROOT>`
* Expands `<APPLICATION_ROOT>/target/*.[jw]ar` to `<APPLICATION_ROOT>`

## Configuration
| Environment Variable | Description
| -------------------- | -----------
| `$BP_MAVEN_BUILD_ARGUMENTS` | Configure the arguments to pass to Maven.  Defaults to `-Dmaven.test.skip=true package`.
| `$BP_MAVEN_BUILT_MODULE` | Configure the module to find application artifact in.  Defaults to the root module (empty).
| `$BP_MAVEN_BUILT_ARTIFACT` | Configure the built application artifact explicitly.  Supersedes `$BP_MAVEN_BUILT_MODULE`  Defaults to `target/*.[jw]ar`.

## Bindings
The buildpack optionally accepts the following bindings:

### Type: `maven`
|Secret | Description
|-----|--------------
|`settings.xml` | If present, `--settings=<path/to/settings.xml>` is prepended to the `maven` arguments

### Type: `dependency-mapping`
|Key                   | Value   | Description
|----------------------|---------|------------
|`<dependency-digest>` | `<uri>` | If needed, the buildpack will fetch the dependency with digest `<dependency-digest>` from `<uri>`

## License
This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
