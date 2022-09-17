# `gcr.io/paketo-buildpacks/maven`

The Paketo Buildpack for Maven is a Cloud Native Buildpack that builds Maven-based applications from source.

## Behavior

This buildpack will participate all the following conditions are met

* Another buildpack requires `maven`, `jvm-application-package` or both
* `<APPLICATION_ROOT>/pom.xml` exists or `BP_MAVEN_POM_FILE` is set to an existing POM file.

The buildpack will do the following:

* Requests that a JDK be installed
* Links the `~/.m2` to a layer for caching
* If `<APPLICATION_ROOT>/mvnw` does not exist and `mvn` is not on `$PATH`
  * Contributes Maven or Maven Daemon to a layer with all commands on `$PATH`
  * Runs `<MAVEN_ROOT>/bin/mvn -Dmaven.test.skip=true --no-transfer-progress package` to build the application
  * Caches `$BP_MAVEN_BUILT_ARTIFACT` to a layer
* If `<APPLICATION_ROOT>/mvnw` exists
  * Runs `<APPLICATION_ROOT>/mvnw -Dmaven.test.skip=true --no-transfer-progress package` to build the application
  * Caches `$BP_MAVEN_BUILT_ARTIFACT` to a layer
* If `mvn` is on `$PATH`
  * Runs `mvn -Dmaven.test.skip=true --no-transfer-progress package` to build the application
  * Caches `$BP_MAVEN_BUILT_ARTIFACT` to a layer
* Removes the source code in `<APPLICATION_ROOT>`, following include/exclude rules
* If `$BP_MAVEN_BUILT_ARTIFACT` matched a single file
  * Restores `$BP_MAVEN_BUILT_ARTIFACT` from the layer, expands the single file to `<APPLICATION_ROOT>`
* If `$BP_MAVEN_BUILT_ARTIFACT` matched a directory or multiple files
  * Restores the files matched by `$BP_MAVEN_BUILT_ARTIFACT` to `<APPLICATION_ROOT>`

## Configuration

| Environment Variable        | Description                                                                                                                                                                                                                                                          |
| --------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `$BP_MAVEN_BUILD_ARGUMENTS` | Configure the arguments to pass to Maven.  Defaults to `-Dmaven.test.skip=true --no-transfer-progress package`. `--batch-mode` will be prepended to the argument list in environments without a TTY.                                                                 |
| `$BP_MAVEN_BUILT_MODULE`    | Configure the module to find application artifact in.  Defaults to the root module (empty).                                                                                                                                                                          |
| `$BP_MAVEN_BUILT_ARTIFACT`  | Configure the built application artifact explicitly.  Supersedes `$BP_MAVEN_BUILT_MODULE`  Defaults to `target/*.[ejw]ar`. Can match a single file, multiple files or a directory. Can be one or more space separated patterns.                                      |
| `$BP_MAVEN_POM_FILE`        | Specifies a custom location to the project's `pom.xml` file. It should be a full path to the file under the `/workspace` directory or it should be relative to the root of the project (i.e. `/workspace'). Defaults to `pom.xml`.                                   |
| `$BP_MAVEN_DAEMON_ENABLED`  | Triggers apache maven-mvnd to be installed and configured for use instead of Maven. The default value is `false`. Set to `true` to use the Maven Daemon.                                                                                                             |
| `$BP_MAVEN_SETTINGS_PATH`   | Specifies a custom location to Maven's `settings.xml` file. If `$BP_MAVEN_SETTINGS_PATH` is set and a Maven binding is provided, the binding takes the higher precedence.                                                                                            |
| `$BP_INCLUDE_FILES`         | Colon separated list of glob patterns to match source files. Any matched file will be retained in the final image. Defaults to `static/*:templates/*:public/*:html/*`.                                                                                               |
| `$BP_EXCLUDE_FILES`         | Colon separated list of glob patterns to match source files. Any matched file will be specifically removed from the final image. If include patterns are also specified, then they are applied first and exclude patterns can be used to further reduce the fileset. |

## Bindings

The buildpack optionally accepts the following bindings:

### Type: `maven`

| Secret                  | Description                                                                                            |
| ----------------------- | ------------------------------------------------------------------------------------------------------ |
| `settings.xml`          | If present `--settings=<path/to/settings.xml>` is prepended to the `maven` arguments                   |
| `settings-security.xml` | If present `-Dsettings.security=<path/to/settings-security.xml>` is prepended to the `maven` arguments |

### Type: `dependency-mapping`

| Key                   | Value   | Description                                                                                       |
| --------------------- | ------- | ------------------------------------------------------------------------------------------------- |
| `<dependency-digest>` | `<uri>` | If needed, the buildpack will fetch the dependency with digest `<dependency-digest>` from `<uri>` |

## License

This buildpack is released under version 2.0 of the [Apache License][a].

[a]: http://www.apache.org/licenses/LICENSE-2.0
